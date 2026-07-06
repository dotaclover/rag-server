package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
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

func main() {
	input := "data/source.jsonl"
	output := "data/index.bin"
	if len(os.Args) > 1 {
		input = os.Args[1]
	}
	if len(os.Args) > 2 {
		output = os.Args[2]
	}
	store, err := buildStore(input)
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		panic(err)
	}
	f, err := os.Create(output)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := gob.NewEncoder(f).Encode(store); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %d docs to %s\n", len(store.Docs), output)
}

func buildStore(input string) (*Store, error) {
	f, err := os.Open(input)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	store := &Store{Version: 1, Model: "keyword+dify-docs-zh", Dimension: 16}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 8*1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var in JSONLDocument
		if err := json.Unmarshal([]byte(line), &in); err != nil {
			return nil, fmt.Errorf("parse line %d: %w", lineNo, err)
		}
		if strings.TrimSpace(in.Text) == "" {
			continue
		}
		if in.ID == "" {
			in.ID = fmt.Sprintf("doc_%04d", lineNo)
		}
		store.Docs = append(store.Docs, Document{
			ID:        in.ID,
			Title:     in.Title,
			Source:    in.Source,
			Section:   in.Section,
			Text:      in.Text,
			Metadata:  in.Metadata,
			Embedding: pseudoEmbedding(in.Title + "\n" + in.Section + "\n" + in.Text),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return store, nil
}

func pseudoEmbedding(text string) []float64 {
	sum := sha256.Sum256([]byte(text))
	vec := make([]float64, 16)
	var length float64
	for i := range vec {
		vec[i] = (float64(sum[i]) - 128) / 128
		length += vec[i] * vec[i]
	}
	if length == 0 {
		return vec
	}
	length = math.Sqrt(length)
	for i := range vec {
		vec[i] = vec[i] / length
	}
	return vec
}
