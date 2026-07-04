package rag

import (
	"context"
	"math"
	"sort"
	"strings"
	"unicode/utf8"
)

type Result struct {
	ID            string            `json:"id"`
	Title         string            `json:"title"`
	Source        string            `json:"source"`
	Section       string            `json:"section"`
	ArticleNumber string            `json:"article_number,omitempty"`
	Text          string            `json:"text"`
	Score         float64           `json:"score"`
	Meta          map[string]string `json:"meta,omitempty"`
}

type Searcher struct {
	store    *Store
	embedder Embedder
}

type SearchOptions struct {
	TopK     int
	MinScore float64
	// Profile carries domain-specific tuning (synonyms, score weights,
	// source filter). Nil means generic, domain-agnostic retrieval.
	Profile *DomainProfile
}

func NewSearcher(store *Store, embedder Embedder) *Searcher {
	return &Searcher{store: store, embedder: embedder}
}

func (s *Searcher) Loaded() bool {
	return s != nil && s.store != nil && len(s.store.Docs) > 0
}

func (s *Searcher) Stats() map[string]interface{} {
	if s == nil || s.store == nil {
		return map[string]interface{}{"loaded": false}
	}
	return map[string]interface{}{
		"loaded":    true,
		"documents": len(s.store.Docs),
		"model":     s.store.Model,
		"dimension": s.store.Dimension,
	}
}

func (s *Searcher) Search(ctx context.Context, query string, topK int) ([]Result, error) {
	return s.SearchWithOptions(ctx, query, SearchOptions{TopK: topK})
}

func (s *Searcher) SearchWithOptions(ctx context.Context, query string, opts SearchOptions) ([]Result, error) {
	topK := opts.TopK
	if topK <= 0 {
		topK = 5
	}
	if s == nil || s.store == nil {
		return nil, nil
	}
	qv, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}
	queryLower := opts.Profile.expand(strings.ToLower(query))
	weights := opts.Profile.weights()
	results := make([]Result, 0, len(s.store.Docs))
	for _, doc := range s.store.Docs {
		if !opts.Profile.allowsSource(doc.Source) {
			continue
		}
		vectorScore := cosine(qv, doc.Embedding)
		textLower := strings.ToLower(doc.Title + " " + doc.Section + " " + doc.Text)
		keywordScore := keywordScore(queryLower, textLower)
		score := vectorScore*weights.Vector + keywordScore*weights.Keyword
		roundedScore := round(score)
		if opts.MinScore > 0 && roundedScore < opts.MinScore {
			continue
		}
		results = append(results, Result{
			ID:            doc.ID,
			Title:         doc.Title,
			Source:        doc.Source,
			Section:       doc.Section,
			ArticleNumber: doc.Metadata["number"],
			Text:          excerpt(doc.Text, 520),
			Score:         roundedScore,
			Meta:          doc.Metadata,
		})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func cosine(a, b []float64) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if n == 0 {
		return 0
	}
	var dot, la, lb float64
	for i := 0; i < n; i++ {
		dot += a[i] * b[i]
		la += a[i] * a[i]
		lb += b[i] * b[i]
	}
	if la == 0 || lb == 0 {
		return 0
	}
	return dot / (math.Sqrt(la) * math.Sqrt(lb))
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
	if strings.Contains(text, query) {
		score += 0.2
	}
	if score > 1 {
		score = 1
	}
	return score
}

func excerpt(text string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}

func round(v float64) float64 {
	return math.Round(v*1000000) / 1000000
}

func tokenize(text string) []string {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return nil
	}
	var tokens []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			tokens = append(tokens, string(current))
			current = current[:0]
		}
	}
	for len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		text = text[size:]
		if isTokenRune(r) {
			current = append(current, r)
			if r >= 0x4e00 && r <= 0x9fff {
				flush()
			}
			continue
		}
		flush()
	}
	flush()
	return tokens
}

func isTokenRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= 0x4e00 && r <= 0x9fff)
}
