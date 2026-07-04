package rag

import (
	"encoding/gob"
	"fmt"
	"os"
	"strconv"
)

type Document struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Source    string            `json:"source"`
	Section   string            `json:"section"`
	Text      string            `json:"text"`
	Metadata  map[string]string `json:"metadata"`
	Embedding []float64         `json:"embedding"`
}

type Store struct {
	Version   int        `json:"version"`
	Model     string     `json:"model"`
	Dimension int        `json:"dimension"`
	Docs      []Document `json:"docs"`
}

func (s *Store) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create rag index: %w", err)
	}
	defer f.Close()
	if err := gob.NewEncoder(f).Encode(s); err != nil {
		return fmt.Errorf("encode rag index: %w", err)
	}
	return nil
}

func Load(path string) (*Store, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open rag index: %w", err)
	}
	defer f.Close()
	var store Store
	if err := gob.NewDecoder(f).Decode(&store); err != nil {
		return nil, fmt.Errorf("decode rag index: %w", err)
	}
	if len(store.Docs) == 0 {
		if legacy, err := loadLegacy(path); err == nil && len(legacy.Articles) > 0 {
			return legacy.ToStore(), nil
		}
	}
	return &store, nil
}

type legacyArticle struct {
	Number    int
	Chapter   string
	Source    string
	Text      string
	Embedding []float64
}

type legacyRAGStore struct {
	Version   int
	Model     string
	Dimension int
	Articles  []legacyArticle
}

func loadLegacy(path string) (*legacyRAGStore, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open legacy rag index: %w", err)
	}
	defer f.Close()
	var store legacyRAGStore
	if err := gob.NewDecoder(f).Decode(&store); err != nil {
		return nil, fmt.Errorf("decode legacy rag index: %w", err)
	}
	return &store, nil
}

func (s *legacyRAGStore) ToStore() *Store {
	out := &Store{
		Version:   s.Version,
		Model:     s.Model,
		Dimension: s.Dimension,
		Docs:      make([]Document, 0, len(s.Articles)),
	}
	for _, article := range s.Articles {
		id := article.Source + "_" + strconv.Itoa(article.Number)
		out.Docs = append(out.Docs, Document{
			ID:      id,
			Title:   article.Source,
			Source:  article.Source,
			Section: article.Chapter,
			Text:    article.Text,
			Metadata: map[string]string{
				"number":  strconv.Itoa(article.Number),
				"source":  article.Source,
				"chapter": article.Chapter,
				"legacy":  "ai-rag-demo",
			},
			Embedding: article.Embedding,
		})
		if out.Dimension == 0 && len(article.Embedding) > 0 {
			out.Dimension = len(article.Embedding)
		}
	}
	return out
}
