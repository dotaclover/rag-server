package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"acflow-rag/rag"
	"acflow-rag/rag/profiles"
)

//go:embed static/*
var content embed.FS

var (
	host             = flag.String("host", "127.0.0.1", "HTTP host")
	port             = flag.String("port", "9093", "HTTP port")
	indexPath        = flag.String("index", "", "Path to single RAG index (legacy mode)")
	domainsDir       = flag.String("domains-dir", "data/domains", "Directory containing domain subdirectories")
	defaultDomain    = flag.String("default-domain", "labor_law", "Default search domain")
	rebuild          = flag.Bool("rebuild", false, "Rebuild index from source JSONL")
	sourcePath       = flag.String("source", "data/source.jsonl", "Source JSONL for rebuild")
	minScore         = flag.Float64("min-score", 0.5, "Minimum result score (0-1)")
	embeddingBaseURL = flag.String("embedding-url", "http://127.0.0.1:9092", "Embedding server base URL")
	embeddingModel   = flag.String("embedding-model", "bge-small-zh-v1.5", "Embedding model name")
	embeddingPath    = flag.String("embedding-path", "/v1/embeddings", "Embedding API path")
)

type SearchRequest struct {
	Query    string   `json:"query"`
	Domain   string   `json:"domain,omitempty"`
	TopK     int      `json:"top_k"`
	MinScore *float64 `json:"min_score,omitempty"`
}

type Result struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Source  string  `json:"source"`
	Section string  `json:"section"`
	Text    string  `json:"text"`
	Score   float64 `json:"score"`
}

type SearchResponse struct {
	Domain   string   `json:"domain"`
	Query    string   `json:"query"`
	Results  []Result `json:"results"`
	Total    int      `json:"total"`
	MinScore float64  `json:"min_score"`
	Message  string   `json:"message,omitempty"`
}

type DomainRuntime struct {
	Name      string
	IndexPath string
	Store     *rag.Store
	Searcher  *rag.Searcher
	Profile   *rag.DomainProfile
}

var domains map[string]*DomainRuntime

func main() {
	flag.Parse()

	embedder := rag.NewOpenAIEmbedderWithPath("",
		strings.TrimRight(*embeddingBaseURL, "/"),
		*embeddingModel, 0, *embeddingPath)

	if *rebuild {
		log.Printf("Rebuilding index from: %s", *sourcePath)
		ctx := context.Background()
		store, err := rag.BuildFromJSONL(ctx, *sourcePath, embedder)
		if err != nil {
			log.Fatalf("Rebuild failed: %v", err)
		}
		if err := store.Save(*indexPath); err != nil {
			log.Fatalf("Save index failed: %v", err)
		}
		log.Printf("Rebuild done: %d docs, %d dims, model=%s", len(store.Docs), store.Dimension, store.Model)
		return
	}

	loadedDomains, err := loadDomains(embedder)
	if err != nil {
		log.Fatalf("Failed to load domains: %v", err)
	}
	domains = loadedDomains

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/search", handleSearch)
	http.HandleFunc("/api/status", handleStatus)

	addr := *host + ":" + *port
	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("  ACFlow RAG 检索服务\n")
	fmt.Print(strings.Repeat("=", 50) + "\n\n")
	fmt.Printf("  🌐 访问地址: http://%s\n", addr)
	fmt.Printf("  🧭 默认知识库: %s\n", *defaultDomain)
	for _, name := range sortedDomainNames(domains) {
		domain := domains[name]
		profile := ""
		if domain.Profile != nil {
			profile = domain.Profile.Name
		}
		fmt.Printf("  📚 %s: %d docs, %d dims, model=%s, profile=%s\n",
			domain.Name, len(domain.Store.Docs), domain.Store.Dimension, domain.Store.Model, profile)
	}
	fmt.Printf("  🔗 Embedding 服务器: %s\n\n", *embeddingBaseURL)
	fmt.Printf("  🎚️ 最低相关度: %d%%\n\n", int(*minScore*100))

	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := content.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	if req.TopK <= 0 {
		req.TopK = 5
	}
	if req.TopK > 10 {
		req.TopK = 10
	}

	domainName := normalizeDomain(req.Domain)
	if domainName == "" {
		domainName = normalizeDomain(r.URL.Query().Get("domain"))
	}
	if domainName == "" {
		domainName = normalizeDomain(r.Header.Get("X-RAG-Domain"))
	}
	if domainName == "" {
		domainName = normalizeDomain(*defaultDomain)
	}
	domain, ok := domains[domainName]
	if !ok || domain == nil || domain.Searcher == nil || !domain.Searcher.Loaded() {
		http.Error(w, fmt.Sprintf("Unknown or unloaded domain: %s", domainName), 404)
		return
	}

	threshold := *minScore
	if req.MinScore != nil {
		threshold = normalizeScoreThreshold(*req.MinScore)
	}

	ragResults, err := domain.Searcher.SearchWithOptions(r.Context(), req.Query, rag.SearchOptions{
		TopK:     req.TopK,
		MinScore: threshold,
		Profile:  domain.Profile,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("Search error: %v", err), 500)
		return
	}

	results := make([]Result, 0, len(ragResults))
	for _, rr := range ragResults {
		text := rr.Text
		runes := []rune(text)
		if len(runes) > 520 {
			text = string(runes[:520]) + "..."
		}
		results = append(results, Result{
			ID:      rr.ID,
			Title:   rr.Title,
			Source:  rr.Source,
			Section: rr.Section,
			Text:    text,
			Score:   math.Round(rr.Score*1000000) / 1000000,
		})
	}

	resp := SearchResponse{
		Domain:   domain.Name,
		Query:    req.Query,
		Results:  results,
		Total:    len(results),
		MinScore: threshold,
	}
	if len(results) == 0 {
		resp.Message = fmt.Sprintf("未找到相关度不低于 %d%% 的参考资料，请换个更具体的问题或降低阈值后重试。",
			int(threshold*100))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	domainName := normalizeDomain(r.URL.Query().Get("domain"))
	if domainName != "" {
		domain, ok := domains[domainName]
		if !ok {
			http.Error(w, fmt.Sprintf("Unknown domain: %s", domainName), 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(domainStatus(domain))
		return
	}

	status := map[string]interface{}{
		"loaded":        len(domains) > 0,
		"defaultDomain": *defaultDomain,
		"domainsDir":    *domainsDir,
		"minScore":      *minScore,
		"domains":       map[string]interface{}{},
	}
	domainStatuses := status["domains"].(map[string]interface{})
	for _, name := range sortedDomainNames(domains) {
		domainStatuses[name] = domainStatus(domains[name])
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func loadDomains(embedder rag.Embedder) (map[string]*DomainRuntime, error) {
	if strings.TrimSpace(*indexPath) != "" {
		store, err := rag.Load(*indexPath)
		if err != nil {
			return nil, fmt.Errorf("load index %s: %w", *indexPath, err)
		}
		name := normalizeDomain(*defaultDomain)
		if name == "" {
			name = detectDomainName(*indexPath, store)
		}
		return map[string]*DomainRuntime{
			name: newDomainRuntime(name, *indexPath, store, embedder),
		}, nil
	}

	entries, err := os.ReadDir(*domainsDir)
	if err != nil {
		return nil, fmt.Errorf("read domains dir %s: %w", *domainsDir, err)
	}

	out := map[string]*DomainRuntime{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := normalizeDomain(entry.Name())
		if name == "" {
			continue
		}
		index := filepath.Join(*domainsDir, entry.Name(), "index.bin")
		store, err := rag.Load(index)
		if err != nil {
			return nil, fmt.Errorf("load domain %s index %s: %w", name, index, err)
		}
		out[name] = newDomainRuntime(name, index, store, embedder)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no domains loaded from %s", *domainsDir)
	}
	if _, ok := out[normalizeDomain(*defaultDomain)]; !ok {
		return nil, fmt.Errorf("default domain %q not loaded", *defaultDomain)
	}
	return out, nil
}

func newDomainRuntime(name, indexPath string, store *rag.Store, embedder rag.Embedder) *DomainRuntime {
	return &DomainRuntime{
		Name:      name,
		IndexPath: indexPath,
		Store:     store,
		Searcher:  rag.NewSearcher(store, embedder),
		Profile:   detectProfile(name, indexPath, store),
	}
}

func detectProfile(domainName, indexPath string, store *rag.Store) *rag.DomainProfile {
	switch normalizeDomain(domainName) {
	case "labor_law", "labor", "law":
		return profiles.LaborLawProfile()
	case "dify_docs", "dify":
		return profiles.DifyDocsProfile()
	}
	detected := detectDomainName(indexPath, store)
	if detected == "labor_law" {
		return profiles.LaborLawProfile()
	}
	if detected == "dify_docs" {
		return profiles.DifyDocsProfile()
	}
	return nil
}

func detectDomainName(indexPath string, store *rag.Store) string {
	lowerPath := strings.ToLower(filepath.ToSlash(indexPath))
	if strings.Contains(lowerPath, "labor") || storeContainsSource(store, "劳动法") || storeContainsSource(store, "劳动合同法") {
		return "labor_law"
	}
	if strings.Contains(lowerPath, "dify") || storeContainsSource(store, "Dify 中文文档") {
		return "dify_docs"
	}
	return "default"
}

func storeContainsSource(store *rag.Store, source string) bool {
	if store == nil {
		return false
	}
	for _, doc := range store.Docs {
		if doc.Source == source {
			return true
		}
	}
	return false
}

func domainStatus(domain *DomainRuntime) map[string]interface{} {
	status := map[string]interface{}{
		"name":      domain.Name,
		"loaded":    false,
		"documents": 0,
		"indexPath": domain.IndexPath,
	}
	if domain == nil {
		return status
	}
	if domain.Profile != nil {
		status["profile"] = domain.Profile.Name
	}
	if domain.Searcher != nil {
		stats := domain.Searcher.Stats()
		for k, v := range stats {
			status[k] = v
		}
		status["loaded"] = domain.Searcher.Loaded()
	}
	return status
}

func sortedDomainNames(domains map[string]*DomainRuntime) []string {
	names := make([]string, 0, len(domains))
	for name := range domains {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func normalizeDomain(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeScoreThreshold(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0.5
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		value = value / 100
	}
	if value > 1 {
		return 1
	}
	return value
}
