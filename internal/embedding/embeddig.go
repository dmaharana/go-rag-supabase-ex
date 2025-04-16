package embedding

import (
	"context"
	"strings"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

func NewEmbedder(openRouterKey, baseURL, embeddingModel string) (*embeddings.EmbedderImpl, error) {
	llm, err := openai.New(
		openai.WithBaseURL(baseURL),
		openai.WithToken(strings.TrimPrefix(openRouterKey, "Bearer ")),
		openai.WithModel(embeddingModel),
	)
	if err != nil {
		return nil, err
	}
	embedder, err := embeddings.NewEmbedder(llm) // Handle both return values
	if err != nil {
		return nil, err
	}
	return embedder, nil
}

func GenerateEmbedding(ctx context.Context, embedder *embeddings.EmbedderImpl, content string) ([]float32, error) {
	chunks := chunkContent(content, 4000)
	if len(chunks) == 0 {
		return nil, nil
	}

	embedding, err := embedder.EmbedQuery(ctx, chunks[0])
	if err != nil {
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
