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
	"strconv"
	"strings"
)

//go:embed static/*
var content embed.FS

var (
	host      = flag.String("host", "127.0.0.1", "HTTP host")
	port      = flag.String("port", "9093", "HTTP port")
	indexPath = flag.String("index", "data/index.bin", "Path to RAG index")
	minScore  = flag.Float64("min-score", defaultMinScore(), "Minimum result score; accepts 0-1 or 0-100")
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
	Query    string   `json:"query"`
	TopK     int      `json:"top_k"`
	MinScore *float64 `json:"min_score,omitempty"`
}

type SearchResponse struct {
	Query    string   `json:"query"`
	Results  []Result `json:"results"`
	Total    int      `json:"total"`
	MinScore float64  `json:"min_score"`
	Message  string   `json:"message,omitempty"`
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
	fmt.Print("\n" + strings.Repeat("=", 50) + "\n")
	fmt.Printf("  ACFlow RAG 检索服务\n")
	fmt.Print(strings.Repeat("=", 50) + "\n\n")
	fmt.Printf("  🌐 访问地址: http://%s\n", addr)
	fmt.Printf("  📚 文档数量: %d\n", len(store.Docs))
	fmt.Printf("  📊 向量维度: %d\n", store.Dimension)
	fmt.Printf("  🤖 模型: %s\n\n", store.Model)
	fmt.Printf("  🎚️ 最低相关度: %d%%\n\n", scorePercent(configuredMinScore()))

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

	threshold := configuredMinScore()
	if req.MinScore != nil {
		threshold = normalizeScoreThreshold(*req.MinScore)
	}
	results := search(req.Query, req.TopK, threshold)

	resp := SearchResponse{
		Query:    req.Query,
		Results:  results,
		Total:    len(results),
		MinScore: threshold,
	}
	if len(results) == 0 {
		resp.Message = noResultsMessage(threshold)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"loaded":    store != nil && len(store.Docs) > 0,
		"documents": 0,
		"indexPath": *indexPath,
		"minScore":  configuredMinScore(),
	}
	if store != nil {
		status["documents"] = len(store.Docs)
		status["dimension"] = store.Dimension
		status["model"] = store.Model
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func search(query string, topK int, minScore float64) []Result {
	queryLower := expandProductDocsQuery(strings.ToLower(query))
	minScore = normalizeScoreThreshold(minScore)

	type scored struct {
		doc   Document
		score float64
	}

	var scores []scored
	for _, doc := range store.Docs {
		textLower := strings.ToLower(doc.Title + " " + doc.Section + " " + doc.Text)
		score := keywordScore(queryLower, textLower)

		if score >= minScore {
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
		if len(runes) > 520 {
			text = string(runes[:520]) + "..."
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

func defaultMinScore() float64 {
	value := strings.TrimSpace(os.Getenv("RAG_MIN_SCORE"))
	if value == "" {
		return 50
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 50
	}
	return parsed
}

func configuredMinScore() float64 {
	return normalizeScoreThreshold(*minScore)
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

func scorePercent(score float64) int {
	return int(math.Round(normalizeScoreThreshold(score) * 100))
}

func noResultsMessage(minScore float64) string {
	return fmt.Sprintf("未找到相关度不低于 %d%% 的参考资料，请换个更具体的问题或降低阈值后重试。", scorePercent(minScore))
}

func keywordScore(query, text string) float64 {
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

var keywordTerms = []string{
	"default model", "http request", "knowledge", "marketplace", "provider", "workflow", "chatflow",
	"embedding", "webapp", "agent", "dify", "dify cloud", "api", "llm", "mcp", "rag",
	"docker", "docker compose", "compose", "cloud", "sandbox", "community edition",
	"模型供应商", "外部知识库", "默认模型", "工作空间", "聊天助手", "文本生成", "问题分类器",
	"知识库", "工作流", "对话流", "数据源", "提示词", "发布", "应用", "节点", "模型",
	"插件", "集成", "工具", "团队", "成员", "权限", "变量", "会话", "记忆", "日志",
	"监控", "文档", "分段", "索引", "检索", "召回", "重排序", "嵌入", "接口", "密钥",
	"创建", "测试", "配置", "导入", "上传", "部署", "安装", "调用", "发布", "调试",
	"运行", "输出", "输入", "文件", "套餐", "用量", "主要功能", "功能", "区别",
	"本地", "线上", "版本", "自部署", "本地部署", "托管", "基础设施", "开源",
	"开箱即用", "沙箱", "社区版",
}

func expandProductDocsQuery(query string) string {
	var expansions []string
	add := func(terms ...string) {
		for _, term := range terms {
			if !strings.Contains(query, strings.ToLower(term)) {
				expansions = append(expansions, term)
			}
		}
	}
	if strings.Contains(query, "线上") || strings.Contains(query, "cloud") {
		add("dify", "dify cloud", "托管", "无需安装", "sandbox")
	}
	if strings.Contains(query, "本地") || strings.Contains(query, "自部署") {
		add("dify", "自部署", "community edition", "docker compose", "基础设施")
	}
	if strings.Contains(query, "安装") || strings.Contains(query, "部署") {
		add("dify", "自部署", "docker compose", "community edition")
	}
	if strings.Contains(query, "主要功能") || strings.Contains(query, "功能") {
		add("dify", "应用", "工作流", "对话流", "知识库", "agent", "发布")
	}
	if strings.Contains(query, "版本") {
		add("dify", "dify cloud", "自部署", "community edition")
	}
	if len(expansions) == 0 {
		return query
	}
	return strings.TrimSpace(query + " " + strings.Join(expansions, " "))
}

func tokenize(text string) []string {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return nil
	}

	seen := map[string]bool{}
	var tokens []string
	add := func(token string) {
		token = strings.TrimSpace(strings.ToLower(token))
		if token == "" || seen[token] {
			return
		}
		seen[token] = true
		tokens = append(tokens, token)
	}

	for _, term := range keywordTerms {
		if strings.Contains(text, term) {
			add(term)
		}
	}

	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			start := i
			for i < len(runes) && ((runes[i] >= 'a' && runes[i] <= 'z') || (runes[i] >= '0' && runes[i] <= '9')) {
				i++
			}
			word := string(runes[start:i])
			if len(word) > 1 {
				add(word)
			}
			i--
		}
	}

	return tokens
}
