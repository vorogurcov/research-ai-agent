package llm

import "github.com/sashabaranov/go-openai"

func NewAPIClient(apiKey, baseURL string) *openai.Client {
	// Should be OpenAI API compatible
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	return openai.NewClientWithConfig(cfg)
}
