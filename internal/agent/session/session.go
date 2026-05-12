package session

import (
	"errors"

	"github.com/sashabaranov/go-openai"
)

var (
	InvalidContextWindowSize = errors.New("context size can not be negative")
)

type AgentSession struct {
	history           []openai.ChatCompletionMessage
	fixed             [2]openai.ChatCompletionMessage
	contextWindowSize int
}

func NewAgentSession(n int, systemMessage openai.ChatCompletionMessage, userMessage openai.ChatCompletionMessage) (*AgentSession, error) {
	if n < 0 {
		return nil, InvalidContextWindowSize
	}

	fixed := [2]openai.ChatCompletionMessage{systemMessage, userMessage}
	history := []openai.ChatCompletionMessage{fixed[0], fixed[1]}
	contextWindowSize := n

	return &AgentSession{
		history,
		fixed,
		contextWindowSize,
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
