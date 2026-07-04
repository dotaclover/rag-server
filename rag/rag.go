package rag

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

type Chunk struct {
	ID     string             `json:"id"`
	Title  string             `json:"title"`
	Text   string             `json:"text"`
	Vector map[string]float64 `json:"-"`
}

type Hit struct {
	Chunk Chunk   `json:"chunk"`
	Score float64 `json:"score"`
}

type Engine struct {
	chunks []Chunk
}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) AddDocument(title, content string) {
	parts := splitMarkdown(content)
	for i, part := range parts {
		text := strings.TrimSpace(part)
		if text == "" {
			continue
		}
		chunk := Chunk{
			ID:     title + "-" + string(rune('A'+i)),
			Title:  title,
			Text:   text,
			Vector: vectorize(text),
		}
		e.chunks = append(e.chunks, chunk)
	}
}

func (e *Engine) Search(query string, limit int) []Hit {
	qv := vectorize(query)
	hits := make([]Hit, 0, len(e.chunks))
	for _, chunk := range e.chunks {
		score := cosine(qv, chunk.Vector)
		if score > 0 {
			hits = append(hits, Hit{Chunk: chunk, Score: score})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})
	if limit > 0 && len(hits) > limit {
		return hits[:limit]
	}
	return hits
}

func (e *Engine) Answer(query string) map[string]any {
	hits := e.Search(query, 3)
	var context strings.Builder
	for i, hit := range hits {
		context.WriteString("[")
		context.WriteString(hit.Chunk.ID)
		context.WriteString("] ")
		context.WriteString(hit.Chunk.Text)
		if i < len(hits)-1 {
			context.WriteString("\n")
		}
	}
	answer := "未检索到足够相关的资料。"
	if len(hits) > 0 {
		answer = "根据知识库资料，" + summarize(query, hits)
	}
	return map[string]any{
		"question":  query,
		"answer":    answer,
		"context":   context.String(),
		"citations": hits,
	}
}

func splitMarkdown(content string) []string {
	lines := strings.Split(content, "\n")
	var chunks []string
	var current strings.Builder
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") && current.Len() > 0 {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}

func summarize(query string, hits []Hit) string {
	best := strings.TrimSpace(hits[0].Chunk.Text)
	best = regexp.MustCompile(`(?m)^#+\s*`).ReplaceAllString(best, "")
	if utf8.RuneCountInString(best) > 180 {
		runes := []rune(best)
		best = string(runes[:180]) + "..."
	}
	return best + "\n\n这个回答由检索片段生成，建议结合原文和实际法律场景进一步确认。问题：" + query
}

func vectorize(text string) map[string]float64 {
	text = strings.ToLower(text)
	re := regexp.MustCompile(`[a-z0-9]+|[\p{Han}]`)
	tokens := re.FindAllString(text, -1)
	vector := map[string]float64{}
	for _, token := range tokens {
		vector[token]++
	}
	return vector
}

func cosine(a, b map[string]float64) float64 {
	var dot, na, nb float64
	for k, av := range a {
		dot += av * b[k]
		na += av * av
	}
	for _, bv := range b {
		nb += bv * bv
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
