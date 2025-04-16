package main

import (
	"context"
	"flag"
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
	flag.Parse()

	if *filePath == "" {
		log.Fatal().Msg("Please provide a document file using the -file flag")
	}

	ctx := context.Background()

	cfg, err := config.LoadConfig(configFilePath)
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
	}

	log.Debug().Interface("config", cfg).Msg("Loaded config")

	// dbinstance, err := db.NewDB(cfg.SupabaseURL, cfg.SupabaseKey)
	// if err != nil {
	// 	log.Fatal().Err(err).Msg("Error initializing database")
	// }
	// defer dbinstance.Close()

	// connect to database
	dbClient, err := db.ConnectDB(cfg.SupabaseURL, cfg.SupabaseKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Error connecting to database")
	}
	dbinstance := db.NewDB(dbClient)

	if err := db.DropDocuments(ctx, dbinstance); err != nil {
		log.Fatal().Err(err).Msg("Error clearing documents")
	}

	if err := db.InitDB(ctx, dbinstance); err != nil {
		log.Fatal().Err(err).Msg("Error initializing database")
	}

	// embedder, err := embedding.NewEmbedder(cfg.OpenRouterKey, cfg.OpenRouterBase, cfg.EmbeddingModel)
	embedder, err := embedding.NewOllamaEmbedder(cfg.OpenRouterBase, cfg.EmbeddingModel)
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing embedder")
	}

	markdown, err := parser.ParseToMarkdown(*filePath)
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing document")
	}

	embedding, err := embedding.GenerateEmbedding(ctx, embedder, markdown)
	if err != nil {
		log.Fatal().Err(err).Msg("Error generating embedding")
	}

	if err := db.StoreDocument(ctx, dbinstance, markdown, embedding); err != nil {
		log.Fatal().Err(err).Msg("Error storing document")
	}

	rag := rag.NewRAG(dbinstance, embedder, cfg)
	query := "What is the main topic of the document?"
	response, err := rag.Query(ctx, cfg.InferenceModel, query)
	if err != nil {
		log.Fatal().Err(err).Msg("Error querying")
	}
	log.Info().Msgf("Response: %s", response)
}
