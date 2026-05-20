package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/vorogurcov/ai-agent/internal/config"
	"github.com/vorogurcov/ai-agent/internal/llm"
)

var (
	InvalidContextWindowSize = errors.New("context size can not be negative")
	summarizationThreshold   = 18000
)

type AgentSession struct {
	history                       []openai.ChatCompletionMessage
	fixed                         [2]openai.ChatCompletionMessage
	contextWindowSize             int
	summarizationTriggerThreshold int
}

func NewAgentSession(n int, systemMessage openai.ChatCompletionMessage, userMessage openai.ChatCompletionMessage) (*AgentSession, error) {
	if n < 0 {
		return nil, InvalidContextWindowSize
	}

	fixed := [2]openai.ChatCompletionMessage{systemMessage, userMessage}
	history := []openai.ChatCompletionMessage{fixed[0], fixed[1]}
	contextWindowSize := n
	summarizationTriggerThreshold := summarizationThreshold
	return &AgentSession{
		history,
		fixed,
		contextWindowSize,
		summarizationTriggerThreshold,
	}, nil
}

func (as *AgentSession) GetCurrentContextWindow() []openai.ChatCompletionMessage {
	n := as.contextWindowSize
	if n == 0 {
		return []openai.ChatCompletionMessage{as.fixed[0], as.fixed[1]}
	}

	count := 0
	i := len(as.history) - 1
	for ; i >= 0 && count < n; i-- {
		if len(as.history[i].ToolCalls) > 0 {
			count++
		}
	}
	start := i + 1
	if start < 2 {
		start = 2
	}
	out := make([]openai.ChatCompletionMessage, 0, 2+len(as.history[start:]))
	out = append(out, as.fixed[0], as.fixed[1])
	out = append(out, as.history[start:]...)
	return out
}

func (as *AgentSession) AppendToHistory(newMessage openai.ChatCompletionMessage) {
	as.history = append(as.history, newMessage)
}

func (as *AgentSession) IsNormalTokenUsage(promptTokens int) bool {
	return promptTokens < as.summarizationTriggerThreshold
}
func (as *AgentSession) summarizeUsingLLM(messagesToSummarize []openai.ChatCompletionMessage) (openai.ChatCompletionMessage, error) {
	var sb strings.Builder

	sb.WriteString("Твоя задача — суммаризовать историю сообщений родительского ИИ агента в рамках слишком большого количества накопленных токенов. " +
		"Дальше я структурированно передам сообщения.")
	for _, message := range messagesToSummarize {
		byteMessage, _ := json.Marshal(message)
		sb.Write(byteMessage)
	}
	prompt := sb.String()

	cfg, err := config.Load(config.LoadParams{
		ModelFlag: os.Getenv("ANALYZING_MODEL_NAME"),
		Prompt:    prompt,
		SystemPrompt: "Ты — узкоспециализированный анализатор текстового содержимого .\n" +
			"Твоя задача: извлекать из предоставленного текста только полезную информацию, отсекая шум (пустые ответы, теги ненужные, и тд).\n" +
			"\n" +
			"КРИТИЧЕСКОЕ ТРЕБОВАНИЕ К ФОРМАТУ ВЫВОДА:\n" +
			"Ты ОБЯЗАН вернуть ТОЛЬКО валидный MARKDOWN c ТЕЗИСАМИ (никаких пояснений, никаких префиксов):\n",
	})
	if err != nil {
		return openai.ChatCompletionMessage{}, err
	}

	client := llm.NewAPIClient(cfg.AIApiKey, cfg.APIBaseURL)
	systemMessage := openai.ChatCompletionMessage{Role: openai.ChatMessageRoleSystem, Content: cfg.SystemPrompt}
	userMessage := openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: cfg.Prompt}

	req := openai.ChatCompletionRequest{
		Model:    cfg.ModelName,
		Messages: []openai.ChatCompletionMessage{systemMessage, userMessage},
		Tools:    nil,
	}

	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return openai.ChatCompletionMessage{}, err
	}

	if len(resp.Choices) == 0 {
		return openai.ChatCompletionMessage{}, fmt.Errorf("no choices in response")
	}
	answerMessage := resp.Choices[0].Message

	return answerMessage, nil
}
func (as *AgentSession) SummarizeHistory() error {
	messagesToSummarize := as.history[2:]
	ans, err := as.summarizeUsingLLM(messagesToSummarize)
	if err != nil {
		return err
	}
	out := make([]openai.ChatCompletionMessage, 0, 3)
	out = append(out, as.fixed[0], as.fixed[1])
	out = append(out, ans)
	as.history = out
	return nil
}
