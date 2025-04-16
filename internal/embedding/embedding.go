package embedding

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

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
func NewOllamaEmbedder(baseURL, embeddingModel string) (*embeddings.EmbedderImpl, error) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Caller().Logger()

	log.Debug().Interface("config", map[string]string{
		"base_url":        baseURL,
		"embedding_model": embeddingModel,
	}).Msg("Loaded config")

	llm, err := ollama.New(
		ollama.WithServerURL(baseURL),
		ollama.WithModel(embeddingModel),
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

func GenerateEmbedding(ctx context.Context, embedder *embeddings.EmbedderImpl, content string) ([]float32, error) {
	chunks := chunkContent(content, 4000)
	if len(chunks) == 0 {
		log.Info().Msg("No chunks generated from content")
		return nil, nil
	}

	embedding, err := embedder.EmbedQuery(ctx, chunks[0])
	if err != nil {
		log.Fatal().Err(err).Msg("Error generating embedding")
		return nil, err
	}
	return embedding, nil
}

func chunkContent(content string, maxChars int) []string {
	var chunks []string
	words := strings.Split(content, " ")
	var chunk strings.Builder
	for _, word := range words {
		if chunk.Len()+len(word)+1 > maxChars {
			chunks = append(chunks, chunk.String())
			chunk.Reset()
		}
		chunk.WriteString(word + " ")
	}
	if chunk.Len() > 0 {
		chunks = append(chunks, chunk.String())
	}
	return chunks
}
