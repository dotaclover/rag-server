package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"acflow-rag/rag"
)

func main() {
	input := flag.String("input", "data/source.jsonl", "Source JSONL file")
	output := flag.String("output", "data/index.bin", "Output index file")
	baseURL := flag.String("embedding-url", "http://127.0.0.1:9092", "Embedding server URL")
	model := flag.String("model", "bge-small-zh-v1.5", "Model name")
	apiPath := flag.String("api-path", "/v1/embeddings", "Embedding API path")
	flag.Parse()

	embedder := rag.NewOpenAIEmbedderWithPath("", *baseURL, *model, 0, *apiPath)
	log.Printf("Building index from %s using %s at %s%s...", *input, *model, *baseURL, *apiPath)

	ctx := context.Background()
	store, err := rag.BuildFromJSONL(ctx, *input, embedder)
	if err != nil {
		log.Fatalf("Build failed: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(*output), 0755); err != nil {
		log.Fatalf("Mkdir failed: %v", err)
	}
	if err := store.Save(*output); err != nil {
		log.Fatalf("Save failed: %v", err)
	}

	fmt.Printf("Done: %d docs, %d-dim vectors, model=%s\n", len(store.Docs), store.Dimension, store.Model)
}