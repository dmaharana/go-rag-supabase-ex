package embedding

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"document-rag/internal/config"
	"document-rag/internal/llmservice"
	"document-rag/internal/models"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

// NewEmbedder creates a new embedder
func NewEmbedder(openRouterKey, baseURL, embeddingModel string) (*embeddings.EmbedderImpl, error) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Caller().Logger()

	log.Debug().Interface("config", map[string]string{
		"base_url":        baseURL,
		"openrouter_key":  openRouterKey,
		"embedding_model": embeddingModel,
	}).Msg("Loaded config")

	llm, err := openai.New(
		openai.WithBaseURL(baseURL),
		openai.WithToken(strings.TrimPrefix(openRouterKey, "Bearer ")),
		openai.WithModel(embeddingModel),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing LLM")
		return nil, err
	}
	embedder, err := embeddings.NewEmbedder(llm) // Handle both return values
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating embedder")
		return nil, err
	}
	return embedder, nil
}

// new ollama embedder
func NewOllamaEmbedder(LLMconfig *config.LLMConfig) (*embeddings.EmbedderImpl, error) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Caller().Logger()

	log.Debug().Interface("config", map[string]string{
		"base_url":        LLMconfig.BaseURL,
		"embedding_model": LLMconfig.Model,
	}).Msg("Loaded config")

	llm, err := ollama.New(
		ollama.WithServerURL(LLMconfig.BaseURL),
		ollama.WithModel(LLMconfig.Model),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing LLM")
		return nil, err
	}
	embedder, err := embeddings.NewEmbedder(llm) // Handle both return values
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating embedder")
		return nil, err
	}
	return embedder, nil
}

// GenerateEmbedding generates embeddings for a given file
func GenerateEmbedding(ctx context.Context, embedder *embeddings.EmbedderImpl, filename string, chunks []models.Chunk) ([]models.ChunkEmbedding, error) {
	if len(chunks) == 0 {
		log.Info().Msg("No chunks generated from content")
		return nil, nil
	}

	var chunkEmbeddings []models.ChunkEmbedding
	for _, chunk := range chunks {
		embedding, err := embedder.EmbedQuery(ctx, chunk.Content)
		if err != nil {
			return nil, err
		}
		chunkEmbeddings = append(chunkEmbeddings, models.ChunkEmbedding{
			Content:        chunk.Content,
			Embedding:      embedding,
			SourceFilename: filename,
			PageNumber:     chunk.PageNumber,
			ChunkID:        chunk.ChunkID,
		})
	}

	return chunkEmbeddings, nil
}

// generate context for each chunk and return new chunks
func GenerateContext(ctx context.Context, llmConfig *config.LLMConfig, document, chunk string) (string, error) {
	log.Debug().Msgf("Generating context for chunk: %s", chunk)
	prompt := fmt.Sprintf(models.ContextPromptTemplate, document, chunk)

	msgContent := []llms.MessageContent{
		llms.MessageContent{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextContent{Text: prompt}},
		},
	}

	res, err := llmservice.GenerateContent(ctx, llmConfig, nil, msgContent)
	if err != nil {
		return "", err
	}
	return res.Choices[0].Content, nil
}
