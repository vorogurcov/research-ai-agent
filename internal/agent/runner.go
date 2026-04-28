package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

type Runner struct {
	Client *openai.Client
	Tools  []openai.Tool
	Caller ToolCaller
}

type ToolCaller interface {
	Call(name string, rawArgs string) (string, error)
}

func (r Runner) Run(model, systemPrompt, userPrompt string) error {
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: userPrompt},
	}

	inCycle := true
	for inCycle {
		req := openai.ChatCompletionRequest{
			Model:    model,
			Messages: messages,
			Tools:    r.Tools,
		}

		resp, err := r.Client.CreateChatCompletion(context.Background(), req)
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}
		if len(resp.Choices) == 0 {
			return fmt.Errorf("no choices in response")
		}

		// Preserve previous behavior: dump raw response to stderr.
		if data, err := json.MarshalIndent(resp, "", "  "); err == nil {
			fmt.Fprintln(os.Stderr, string(data))
		}

		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			fmt.Fprint(os.Stdout, msg.Content)
			inCycle = false
			continue
		}

		// Append assistant's message with tool calls to history
		messages = append(messages, msg)

		for _, tc := range msg.ToolCalls {
			toolMsg := r.dispatchToolCall(tc)
			messages = append(messages, toolMsg)
		}
	}

	return nil
}

