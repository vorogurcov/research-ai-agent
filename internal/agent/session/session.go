package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sashabaranov/go-openai"
	"github.com/vorogurcov/ai-agent/internal/config"
	"github.com/vorogurcov/ai-agent/internal/llm"
)

var (
	InvalidContextWindowSize = errors.New("context size can not be negative")
	summarizationThreshold   = 18000
)

type AgentSession struct {
	fixed                         [2]openai.ChatCompletionMessage
	contextWindowSize             int
	summarizationTriggerThreshold int
	redisClient                   *redis.Client
	historyKey                    string
}

func NewAgentSession(n int, systemMessage openai.ChatCompletionMessage, userMessage openai.ChatCompletionMessage) (*AgentSession, error) {
	if n < 0 {
		return nil, InvalidContextWindowSize
	}

	fixed := [2]openai.ChatCompletionMessage{systemMessage, userMessage}
	history := []openai.ChatCompletionMessage{fixed[0], fixed[1]}
	contextWindowSize := n
	summarizationTriggerThreshold := summarizationThreshold
	sessionId := uuid.NewString()
	historyKey := fmt.Sprintf("history:%v", sessionId)
	var redisAddr string
	if os.Getenv("DOCKER_ENV") == "TRUE" {
		redisAddr = os.Getenv("DOCKER_REDIS_URL")
	} else {
		redisAddr = os.Getenv("LOCAL_REDIS_URL")
	}
	if redisAddr == "" {
		log.Fatal("redis addr is not set")
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password
		DB:       0,  // use default DB
		Protocol: 2,
	})
	b, _ := json.Marshal(history)

	err := redisClient.Set(context.Background(), historyKey, b, 0).Err()
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}
	return &AgentSession{
		fixed,
		contextWindowSize,
		summarizationTriggerThreshold,
		redisClient,
		historyKey,
	}, nil
}

func (as *AgentSession) GetCurrentContextWindow() []openai.ChatCompletionMessage {
	n := as.contextWindowSize
	if n == 0 {
		return []openai.ChatCompletionMessage{as.fixed[0], as.fixed[1]}
	}

	count := 0
	historyJson, err := as.redisClient.Get(context.Background(), as.historyKey).Result()
	var history []openai.ChatCompletionMessage
	err = json.Unmarshal([]byte(historyJson), &history)
	if err != nil {
		log.Fatal(err)
	}

	i := len(history) - 1
	for ; i >= 0 && count < n; i-- {
		if len(history[i].ToolCalls) > 0 {
			count++
		}
	}
	start := i + 1
	if start < 2 {
		start = 2
	}
	out := make([]openai.ChatCompletionMessage, 0, 2+len(history[start:]))
	out = append(out, as.fixed[0], as.fixed[1])
	out = append(out, history[start:]...)
	return out
}

func (as *AgentSession) AppendToHistory(newMessage openai.ChatCompletionMessage) {
	historyJson, err := as.redisClient.Get(context.Background(), as.historyKey).Result()
	var history []openai.ChatCompletionMessage
	err = json.Unmarshal([]byte(historyJson), &history)
	if err != nil {
		log.Fatal(err)
	}

	history = append(history, newMessage)
	b, _ := json.Marshal(history)

	err = as.redisClient.Set(context.Background(), as.historyKey, b, 0).Err()
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}
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
	userMessage := openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: prompt}

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
	historyJson, err := as.redisClient.Get(context.Background(), as.historyKey).Result()
	var history []openai.ChatCompletionMessage
	err = json.Unmarshal([]byte(historyJson), &history)
	if err != nil {
		log.Fatal(err)
	}

	messagesToSummarize := history[2:]
	ans, err := as.summarizeUsingLLM(messagesToSummarize)
	if err != nil {
		return err
	}
	out := make([]openai.ChatCompletionMessage, 0, 3)
	out = append(out, as.fixed[0], as.fixed[1])
	out = append(out, ans)
	history = out

	b, _ := json.Marshal(history)

	err = as.redisClient.Set(context.Background(), as.historyKey, b, 0).Err()
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
		return err
	}

	return nil
}
