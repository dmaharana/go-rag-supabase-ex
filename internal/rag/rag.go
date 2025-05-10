package rag

import (
	"context"
	"fmt"
	"strings"

	"document-rag/internal/config"
	"document-rag/internal/db"
	"document-rag/internal/models"

	"document-rag/internal/chromemdb"
	"document-rag/internal/llmservice"

	"github.com/philippgille/chromem-go"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/uptrace/bun"
)

type RAG struct {
	db         *bun.DB
	chromemdb  *chromemdb.VectorDBManager
	embedder   *embeddings.EmbedderImpl
	cfg        *config.Config
	maxResults int
}

const defaultMaxResults = 5

func NewRAG(db *bun.DB, chromemdb *chromemdb.VectorDBManager, embedder *embeddings.EmbedderImpl, cfg *config.Config) *RAG {
	return &RAG{
		db:        db,
		chromemdb: chromemdb,
		embedder:  embedder,
		cfg:       cfg,
		maxResults: func() int {
			if cfg.RAG.MaxResults > 0 {
				return cfg.RAG.MaxResults
			}
			return defaultMaxResults
		}(),
	}
}

func (r *RAG) Query(ctx context.Context, query string) (models.PromptResponse, error) {
	rsp := models.PromptResponse{
		Query:   query,
		Source:  "",
		Content: "",
	}
	queryEmbedding, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return rsp, err
	}

	// if maxResults is 0, use default value
	if r.maxResults == 0 {
		r.maxResults = defaultMaxResults
	}

	var qContext strings.Builder
	var references []string
	if r.db != nil {
		docs, err := db.SearchDocuments(ctx, r.db, queryEmbedding, r.maxResults)
		if err != nil {
			return rsp, err
		}

		for i, doc := range docs {
			qContext.WriteString(doc.Content + "\n\n")

			// Build reference string
			ref := fmt.Sprintf("Source: %s, Page: %d, Chunk: %d", doc.SourceFilename, doc.PageNumber, doc.ChunkID)
			references = append(references, fmt.Sprintf("[%d] %s", i+1, ref))
		}
	} else if r.chromemdb != nil {
		queryOptions := chromem.QueryOptions{
			QueryText:      query,
			QueryEmbedding: queryEmbedding,
			NResults:       r.maxResults,
		}
		docs, err := r.chromemdb.SearchWithQueryOptions(ctx, queryOptions)
		if err != nil {
			return rsp, err
		}

		for _, doc := range docs {
			qContext.WriteString(doc.Content + "\n\n")

			// Build reference string
			// ref := fmt.Sprintf("Source: %s, Page: %d, Chunk: %d", doc.Metadata["source_filename"], doc.Metadata["page_number"], doc.Metadata["chunk_id"])
			references = append(references, fmt.Sprintf("%v", doc.Metadata))
		}
	}

	if qContext.String() == "" {
		return rsp, fmt.Errorf("no documents found")
	}

	rsp.Source = qContext.String()

	// llm, err := openai.New(
	// 	openai.WithBaseURL(r.cfg.QueryLLM.BaseURL),
	// 	openai.WithToken(strings.TrimPrefix(r.cfg.QueryLLM.Key, "Bearer ")),
	// 	openai.WithModel(r.cfg.QueryLLM.Model),
	// )
	// if err != nil {
	// 	return rsp, err
	// }

	var response strings.Builder
	prompt := fmt.Sprintf("Based on the following context, answer the query: %s\n\nContext:\n%s", query, qContext.String())
	msgContent := []llms.MessageContent{
		llms.MessageContent{
			Role:  llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{llms.TextContent{Text: "You are a helpful assistant. Answer the query based only on the provided context. If the context does not contain the answer, respond with 'I don't know.'"}},
		},
		llms.MessageContent{
			Role:  llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{llms.TextContent{Text: prompt}},
		},
	}

	// Stream the response
	// _, err = llm.GenerateContent(ctx, msgContent, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
	// 	chunkStr := string(chunk)
	// 	if strings.Contains(chunkStr, ": OPENROUTER PROCESSING") {
	// 		return nil
	// 	}
	// 	response.WriteString(chunkStr)
	// 	return nil
	// }))

	// res, err := llm.GenerateContent(ctx, msgContent)
	res, err := llmservice.GenerateContent(ctx, &r.cfg.QueryLLM, nil, msgContent)
	if err != nil {
		return rsp, err
	}

	if len(res.Choices) == 0 {
		return rsp, fmt.Errorf("no response from LLM")
	}
	response.WriteString(res.Choices[0].Content)

	// Append references to the response
	response.WriteString("\n\nReferences:\n")
	for _, ref := range references {
		response.WriteString(ref + "\n")
	}

	rsp.Content = response.String()

	return rsp, nil
}
