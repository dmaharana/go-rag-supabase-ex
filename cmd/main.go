package main

import (
	"context"
	"fmt"

	"document-rag/internal/config"
	"document-rag/internal/db" // Ensure db package is imported
	"document-rag/internal/embedding"
	"document-rag/internal/parser"
	"document-rag/internal/rag"
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	dbinstance, err := db.NewDB(cfg.SupabaseURL, cfg.SupabaseKey)
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		return
	}
	defer dbinstance.Close()

	embedder, err := embedding.NewEmbedder(cfg.OpenRouterKey, cfg.OpenRouterBase)
	if err != nil {
		fmt.Printf("Error initializing embedder: %v\n", err)
		return
	}

	filePath := "sample.pdf" // Replace with your document
	markdown, err := parser.ParseToMarkdown(filePath)
	if err != nil {
		fmt.Printf("Error parsing document: %v\n", err)
		return
	}

	embedding, err := embedding.GenerateEmbedding(ctx, embedder, markdown)
	if err != nil {
		fmt.Printf("Error generating embedding: %v\n", err)
		return
	}

	if err := db.StoreDocument(ctx, dbinstance, markdown, embedding); err != nil {
		fmt.Printf("Error storing document: %v\n", err)
		return
	}

	rag := rag.NewRAG(dbinstance, embedder, cfg)
	query := "What is the main topic of the document?"
	response, err := rag.Query(ctx, query)
	if err != nil {
		fmt.Printf("Error querying: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n", response)
}
