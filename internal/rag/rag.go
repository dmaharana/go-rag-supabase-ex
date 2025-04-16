package rag

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"document-rag/internal/config"
	"document-rag/internal/db"
	"document-rag/internal/embedding"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/uptrace/bun"
)

type RAG struct {
	db       *bun.DB
	embedder *embeddings.EmbedderImpl
	cfg      *config.Config
}

func NewRAG(db *bun.DB, embedder *embeddings.EmbedderImpl, cfg *config.Config) *RAG {
	return &RAG{db: db, embedder: embedder, cfg: cfg}
}

func (r *RAG) Query(ctx context.Context, inferenceModel, query string) (string, error) {
	queryEmbedding, err := embedding.GenerateEmbedding(ctx, r.embedder, query)
	if err != nil {
		return "", err
	}

	docs, err := db.SearchDocuments(ctx, r.db, queryEmbedding, 5)
	if err != nil {
		return "", err
	}

	var context strings.Builder
	for _, doc := range docs {
		context.WriteString(doc.Content + "\n\n")
	}

	// Stream response from OpenRouter
	payload := struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream bool `json:"stream"`
	}{
		Model: inferenceModel,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "system", Content: "You are a helpful assistant. Use the provided context to answer the query."},
			{Role: "user", Content: fmt.Sprintf("Context:\n%s\nQuery: %s", context.String(), query)},
		},
		Stream: true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", r.cfg.OpenRouterBase+"/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", r.cfg.OpenRouterKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("request failed: %d, %s", resp.StatusCode, string(body))
	}

	var response strings.Builder
	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if line == "data: [DONE]" {
			break
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				response.WriteString(chunk.Choices[0].Delta.Content)
			}
		}
	}

	return response.String(), nil
}
