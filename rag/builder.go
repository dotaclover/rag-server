package rag

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type JSONLDocument struct {
	ID       string            `json:"id"`
	Title    string            `json:"title"`
	Source   string            `json:"source"`
	Section  string            `json:"section"`
	Text     string            `json:"text"`
	Metadata map[string]string `json:"metadata"`
}

func BuildFromJSONL(ctx context.Context, inputPath string, embedder Embedder) (*Store, error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("open jsonl: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 8*1024*1024)
	store := &Store{Version: 1, Model: embedder.Name(), Dimension: embedder.Dimension()}
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var in JSONLDocument
		if err := json.Unmarshal([]byte(line), &in); err != nil {
			return nil, fmt.Errorf("parse jsonl line %d: %w", lineNo, err)
		}
		if strings.TrimSpace(in.Text) == "" {
			continue
		}
		if in.ID == "" {
			in.ID = fmt.Sprintf("doc_%04d", lineNo)
		}
		vec, err := embedder.Embed(ctx, in.Title+"\n"+in.Section+"\n"+in.Text)
		if err != nil {
			return nil, fmt.Errorf("embed line %d: %w", lineNo, err)
		}
		store.Docs = append(store.Docs, Document{
			ID:        in.ID,
			Title:     in.Title,
			Source:    in.Source,
			Section:   in.Section,
			Text:      in.Text,
			Metadata:  in.Metadata,
			Embedding: vec,
		})
		if len(vec) > 0 {
			store.Dimension = len(vec)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan jsonl: %w", err)
	}
	return store, nil
}
