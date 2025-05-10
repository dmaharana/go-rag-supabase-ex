package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/philippgille/chromem-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"document-rag/internal/chromemdb"
	"document-rag/internal/config"
	"document-rag/internal/db"
	"document-rag/internal/embedding"
	"document-rag/internal/helper"
	"document-rag/internal/parser"
	"document-rag/internal/rag"
)

const (
	configFilePath = "./configs/config.yaml"
	vectorSize     = 768
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Caller().Logger()

	filePath := flag.String("file", "", "Path to the document file")
	query := flag.String("query", "", "Query to be answered")
	dryRun := flag.Bool("dry-run", false, "Dry run, do not save to database")
	flag.Parse()

	// TODO: parse bg file and print the result
	if *filePath != "" {
		parseBGText(context.Background(), *filePath, *dryRun)
		return
	}

	if *filePath != "" && *query != "" {
		log.Fatal().Msg("Please provide either a document file using the -file flag or a query using the -query flag, but not both")
	}

	if *query != "" {
		searchBGContent(context.Background(), *query)
		return
	}

	// if *filePath != "" {
	// 	storeFileEmbedding(context.Background(), *filePath)
	// 	return
	// }

	// if *query != "" {
	// 	performRAG(context.Background(), *query)
	// 	return
	// }

	// log.Fatal().Msg("Please provide either a document file using the -file flag or a query using the -query flag")
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
	dbInstance := db.NewDB(dbClient, cfg.Database.Debug)
	defer dbInstance.Close()

	if err := db.DropDocuments(ctx, dbInstance); err != nil {
		log.Fatal().Err(err).Msg("Error clearing documents")
	}

	if err := db.InitDB(ctx, dbInstance, vectorSize); err != nil {
		log.Fatal().Err(err).Msg("Error initializing database")
	}

	embedder, err := embedding.NewOllamaEmbedder(&cfg.EmbedLLM)
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing embedder")
	}

	chunks, err := parser.ParseToMarkdown(filePath, cfg)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing document")
		return
	}

	chunkEmbeddings, err := embedding.GenerateEmbedding(ctx, embedder, filePath, chunks)
	if err != nil {
		log.Fatal().Err(err).Msg("Error generating embedding")
	}

	// Convert chunk embeddings to Document records for batch storage
	docs := make([]db.Document, len(chunkEmbeddings))
	for i, ce := range chunkEmbeddings {
		docs[i] = db.Document{
			Content:        ce.Content,
			Embedding:      ce.Embedding,
			SourceFilename: ce.SourceFilename,
			PageNumber:     ce.PageNumber,
			ChunkID:        ce.ChunkID,
		}
	}

	if err := db.StoreDocuments(ctx, dbInstance, docs); err != nil {
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
	dbInstance := db.NewDB(dbClient, cfg.Database.Debug)
	defer dbInstance.Close()

	embedder, err := embedding.NewOllamaEmbedder(&cfg.EmbedLLM)
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing embedder")
	}

	rag := rag.NewRAG(dbInstance, nil, embedder, cfg)
	response, err := rag.Query(ctx, query)
	if err != nil {
		log.Fatal().Err(err).Msg("Error querying")
	}

	log.Info().Msg("Query: ~~~~~~~~~~~~~~~~~~~~~~~~~>>>>>")
	fmt.Printf("%s\n\n", query)

	log.Info().Msg("Source: ~~~~~~~~~~~~~~~~~~~~~~~~~>>>>>")
	fmt.Printf("%s\n\n", response.Source)

	log.Info().Msg("Assistant: ~~~~~~~~~~~~~~~~~~~~~~~~~>>>>>")

	fmt.Printf("%s\n\n", response.Content)

}

const (
	dbPath         = "./chromemdb"
	collectionName = "bg_collection"
	inMemory       = false
)

func parseBGText(ctx context.Context, filePath string, dryRun bool) {
	// load config
	cfg, err := config.LoadConfig(configFilePath)
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
	}

	// create folder
	err = helper.CreateFolder(dbPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating folder")
	}

	log.Debug().Interface("config", cfg).Msg("Loaded config")

	// parse content
	content := parser.ParseBGText(filePath, cfg)
	log.Info().Msg("Parsed content")
	helper.PrettyPrint(content)

	// add context
	// content = parser.AddContextByChapter(content, cfg)
	// log.Info().Msg("Content with context")
	// helper.PrettyPrint(content)

	if dryRun {
		return
	}
	// embed content
	embedder, err := embedding.NewOllamaEmbedder(&cfg.EmbedLLM)
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing embedder")
	}

	var docs []chromem.Document
	for _, section := range content {
		// if content is empty, skip
		if section.Content == "" {
			continue
		}
		embedding, err := embedder.EmbedQuery(ctx, section.Content)
		if err != nil {
			log.Fatal().Err(err).Msg("Error generating embedding")
		}
		docs = append(docs, chromem.Document{
			ID:        fmt.Sprintf("%s-%s-%d", section.Chapter, section.Speaker, section.ChunkID),
			Content:   section.Content,
			Metadata:  parser.CreateMetadata(section),
			Embedding: embedding,
		})
	}

	// store content in database
	// create chromemdb
	db, err := chromemdb.NewVectorDBManager(dbPath, collectionName, inMemory, cfg.RAG.EncryptionKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating vector database manager")
	}

	// create collection
	_, err = db.GetOrCreateCollection(collectionName)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating collection")
	}

	// delete collection first
	// err = db.DeleteCollection()
	// if err != nil {
	// 	log.Fatal().Err(err).Msg("Error deleting collection")
	// }

	log.Info().Msgf("Adding %d documents to vector database", len(docs))

	// add content to chromemdb
	err = db.CreateDocs(docs)
	if err != nil {
		log.Fatal().Err(err).Msg("Error adding content to vector database")
	}

	if inMemory {
		// export collection
		err = db.Export(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("Error exporting collection")
		}
	}
}

// search bg content
func searchBGContent(ctx context.Context, query string) error {
	cfg, err := config.LoadConfig(configFilePath)
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
	}
	// create chromemdb
	db, err := chromemdb.NewVectorDBManager(dbPath, collectionName, inMemory, cfg.RAG.EncryptionKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating vector database manager")
	}

	// create collection
	_, err = db.GetOrCreateCollection(collectionName)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating collection")
	}

	embedder, err := embedding.NewOllamaEmbedder(&cfg.EmbedLLM)
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing embedder")
	}

	rag := rag.NewRAG(nil, db, embedder, cfg)
	response, err := rag.Query(ctx, query)
	if err != nil {
		log.Fatal().Err(err).Msg("Error querying")
	}

	log.Info().Msg("Query: ~~~~~~~~~~~~~~~~~~~~~~~~~>>>>>")
	fmt.Printf("%s\n\n", query)

	log.Info().Msg("Source: ~~~~~~~~~~~~~~~~~~~~~~~~~>>>>>")
	fmt.Printf("%s\n\n", response.Source)

	log.Info().Msg("Assistant: ~~~~~~~~~~~~~~~~~~~~~~~~~>>>>>")
	fmt.Printf("%s\n\n", response.Content)

	return nil
}
