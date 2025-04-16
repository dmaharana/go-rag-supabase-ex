package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"document-rag/internal/config"
	"document-rag/internal/db"
	"document-rag/internal/embedding"
	"document-rag/internal/parser"
	"document-rag/internal/rag"
)

const (
	configFilePath = "./configs/config.yaml"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Caller().Logger()

	filePath := flag.String("file", "", "Path to the document file")
	query := flag.String("query", "", "Query to be answered")
	flag.Parse()

	if *filePath != "" && *query != "" {
		log.Fatal().Msg("Please provide either a document file using the -file flag or a query using the -query flag, but not both")
	}

	if *filePath != "" {
		storeFileEmbedding(context.Background(), *filePath)
		return
	}

	if *query != "" {
		performRAG(context.Background(), *query)
		return
	}

	log.Fatal().Msg("Please provide either a document file using the -file flag or a query using the -query flag")
}

func storeFileEmbedding(ctx context.Context, filePath string) {
	cfg, err := config.LoadConfig(configFilePath)
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
	}

	log.Debug().Interface("config", cfg).Msg("Loaded config")

	dbClient, err := db.ConnectDB(&cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Error connecting to database")
	}
	dbInstance := db.NewDB(dbClient)
	defer dbInstance.Close()

	if err := db.DropDocuments(ctx, dbInstance); err != nil {
		log.Fatal().Err(err).Msg("Error clearing documents")
	}

	if err := db.InitDB(ctx, dbInstance); err != nil {
		log.Fatal().Err(err).Msg("Error initializing database")
	}

	embedder, err := embedding.NewOllamaEmbedder(&cfg.EmbedLLM)
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing embedder")
	}

	markdown, err := parser.ParseToMarkdown(filePath)
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing document")
	}

	embedding, err := embedding.GenerateEmbedding(ctx, embedder, markdown)
	if err != nil {
		log.Fatal().Err(err).Msg("Error generating embedding")
	}

	if err := db.StoreDocument(ctx, dbInstance, markdown, embedding); err != nil {
		log.Fatal().Err(err).Msg("Error storing document")
	}
}

func performRAG(ctx context.Context, query string) {
	cfg, err := config.LoadConfig(configFilePath)
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
	}

	log.Debug().Interface("config", cfg).Msg("Loaded config")

	dbClient, err := db.ConnectDB(&cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Error connecting to database")
	}
	dbInstance := db.NewDB(dbClient)
	defer dbInstance.Close()

	embedder, err := embedding.NewOllamaEmbedder(&cfg.EmbedLLM)
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing embedder")
	}

	rag := rag.NewRAG(dbInstance, embedder, cfg)
	response, err := rag.Query(ctx, query)
	if err != nil {
		log.Fatal().Err(err).Msg("Error querying")
	}
	fmt.Println(response)

}
