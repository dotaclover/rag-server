package main

import (
	"embed"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
)

//go:embed static/*
var content embed.FS

var (
	host      = flag.String("host", "127.0.0.1", "HTTP host")
	port      = flag.String("port", "9093", "HTTP port")
	indexPath = flag.String("index", "data/index.bin", "Path to RAG index")
)

type Document struct {
	ID        string
	Title     string
	Source    string
	Section   string
	Text      string
	Metadata  map[string]string
	Embedding []float64
}

type Store struct {
	Version   int
	Model     string
	Dimension int
	Docs      []Document
}

type SearchRequest struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k"`
}

type SearchResponse struct {
	Query   string   `json:"query"`
	Results []Result `json:"results"`
	Total   int      `json:"total"`
}

type Result struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Source  string  `json:"source"`
	Section string  `json:"section"`
	Text    string  `json:"text"`
	Score   float64 `json:"score"`
}

var store *Store

func main() {
	flag.Parse()
	
	log.Printf("Loading index from: %s", *indexPath)
	if err := loadIndex(*indexPath); err != nil {
		log.Fatalf("Failed to load index: %v", err)
	}
	
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/search", handleSearch)
	http.HandleFunc("/api/status", handleStatus)
	
	addr := *host + ":" + *port
	fmt.Printf("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("  ACFlow RAG 检索服务\n")
	fmt.Printf(strings.Repeat("=", 50) + "\n\n")
	fmt.Printf("  🌐 访问地址: http://%s\n", addr)
	fmt.Printf("  📚 文档数量: %d\n", len(store.Docs))
	fmt.Printf("  📊 向量维度: %d\n", store.Dimension)
	fmt.Printf("  🤖 模型: %s\n\n", store.Model)
	
	log.Fatal(http.ListenAndServe(addr, nil))
}

func loadIndex(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open index: %w", err)
	}
	defer f.Close()
	
	store = &Store{}
	if err := gob.NewDecoder(f).Decode(store); err != nil {
		return fmt.Errorf("decode index: %w", err)
	}
	
	if len(store.Docs) == 0 {
		return fmt.Errorf("index is empty")
	}
	
	return nil
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
	
	if store == nil || len(store.Docs) == 0 {
		http.Error(w, "Index not loaded", 503)
		return
	}
	
	results := search(req.Query, req.TopK)
	
	resp := SearchResponse{
		Query:   req.Query,
		Results: results,
		Total:   len(results),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"loaded":    store != nil && len(store.Docs) > 0,
		"documents": 0,
		"indexPath": *indexPath,
	}
	if store != nil {
		status["documents"] = len(store.Docs)
		status["dimension"] = store.Dimension
		status["model"] = store.Model
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func search(query string, topK int) []Result {
	queryLower := strings.ToLower(query)
	
	type scored struct {
		doc   Document
		score float64
	}
	
	var scores []scored
	for _, doc := range store.Docs {
		textLower := strings.ToLower(doc.Title + " " + doc.Section + " " + doc.Text)
		score := keywordScore(queryLower, textLower)
		
		if score > 0 {
			scores = append(scores, scored{doc, score})
		}
	}
	
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})
	
	if len(scores) > topK {
		scores = scores[:topK]
	}
	
	results := make([]Result, len(scores))
	for i, s := range scores {
		text := s.doc.Text
		// 按字符数截断，不是字节数，避免中文乱码
		runes := []rune(text)
		if len(runes) > 200 {
			text = string(runes[:200]) + "..."
		}
		results[i] = Result{
			ID:      s.doc.ID,
			Title:   s.doc.Title,
			Source:  s.doc.Source,
			Section: s.doc.Section,
			Text:    text,
			Score:   math.Round(s.score*1000000) / 1000000,
		}
	}
	
	return results
}

func keywordScore(query, text string) float64 {
	// 中文按字符分词，英文按空格分词
	tokens := tokenize(query)
	if len(tokens) == 0 {
		return 0
	}
	
	matched := 0
	for _, token := range tokens {
		if token != "" && strings.Contains(text, token) {
			matched++
		}
	}
	
	score := float64(matched) / float64(len(tokens))
	
	// 完全匹配加分
	if strings.Contains(text, query) {
		score += 0.3
	}
	
	if score > 1 {
		score = 1
	}
	
	return score
}

func tokenize(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	
	var tokens []string
	runes := []rune(text)
	
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		// 中文字符
		if r >= 0x4e00 && r <= 0x9fff {
			tokens = append(tokens, string(r))
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			// 英文单词
			start := i
			for i < len(runes) && ((runes[i] >= 'a' && runes[i] <= 'z') || 
				(runes[i] >= 'A' && runes[i] <= 'Z') || 
				(runes[i] >= '0' && runes[i] <= '9')) {
				i++
			}
			tokens = append(tokens, strings.ToLower(string(runes[start:i])))
			i--
		}
	}
	
	return tokens
}
