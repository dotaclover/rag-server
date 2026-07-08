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
	terms := opts.Profile.keywordTerms()
	results := make([]Result, 0, len(s.store.Docs))
	for _, doc := range s.store.Docs {
		if !opts.Profile.allowsSource(doc.Source) {
			continue
		}
		vectorScore := cosineFloat64(qv, doc.Embedding)
		textLower := strings.ToLower(doc.Title + " " + doc.Section + " " + doc.Text)
		keywordScore := keywordScore(queryLower, textLower, terms)
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

func cosineFloat64(a, b []float64) float64 {
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

func keywordScore(query, text string, terms []string) float64 {
	tokens := tokenize(query, terms)
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

func tokenize(text string, terms []string) []string {
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
	for _, term := range terms {
		if strings.Contains(text, term) {
			add(term)
		}
	}
	for len(text) > 0 {
		r, size := utf8.DecodeRuneInString(text)
		text = text[size:]
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			var current []rune
			current = append(current, r)
			for len(text) > 0 {
				next, nextSize := utf8.DecodeRuneInString(text)
				if !((next >= 'a' && next <= 'z') || (next >= '0' && next <= '9')) {
					break
				}
				current = append(current, next)
				text = text[nextSize:]
			}
			word := string(current)
			if len(word) > 1 {
				add(word)
			}
		}
	}
	return tokens
}

func isTokenRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= 0x4e00 && r <= 0x9fff)
}
