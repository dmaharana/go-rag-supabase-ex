package llmservice

import (
	"context"
	"strings"

	"document-rag/internal/config"

	"github.com/rs/zerolog/log"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// call llm
func GenerateContent(ctx context.Context, llmConfig *config.LLMConfig, tools []llms.Tool, messages []llms.MessageContent) (*llms.ContentResponse, error) {
	log.Debug().Interface("llmConfig", llmConfig).Msg("Generating content")
	llm, err := openai.New(
		openai.WithBaseURL(llmConfig.BaseURL),
		openai.WithToken(strings.TrimPrefix(llmConfig.Key, "Bearer ")),
		openai.WithModel(llmConfig.Model),
	)
	if err != nil {
		return nil, err
	}

	if tools != nil && len(tools) > 0 {
		return llm.GenerateContent(ctx, messages, llms.WithTools(tools))
	}

	return llm.GenerateContent(ctx, messages)
}
