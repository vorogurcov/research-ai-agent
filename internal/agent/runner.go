package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type Logger interface {
	Write(text string) error
}

type Runner struct {
	Client *openai.Client
	Tools  []openai.Tool
	Caller ToolCaller
	Logger Logger
}

type ToolCaller interface {
	Call(name string, rawArgs string) (string, error)
}

func (r Runner) Run(model, systemPrompt, userPrompt string) (*string, error) {
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: userPrompt},
	}
	var answer string

	// Human-readable trace (stderr) without huge JSON dumps.
	trace := !strings.EqualFold(os.Getenv("TRACE_LLM"), "0") &&
		!strings.EqualFold(os.Getenv("TRACE_LLM"), "false") &&
		!strings.EqualFold(os.Getenv("TRACE_LLM"), "no")
	if trace {
		fmt.Fprintf(os.Stderr, "Q: %s\n", compactText(userPrompt, 500))
	}

	inCycle := true
	const maxCycles = 30
	cycle := 0
	toolCache := map[string]string{}
	var lastToolKey string
	repeatSameTool := 0

	for inCycle {
		cycle++
		if cycle > maxCycles {
			return nil, fmt.Errorf("tool loop detected: exceeded %d cycles", maxCycles)
		}

		req := openai.ChatCompletionRequest{
			Model:    model,
			Messages: messages,
			Tools:    r.Tools,
		}

		resp, err := r.Client.CreateChatCompletion(context.Background(), req)
		if err != nil {
			if r.Logger != nil {
				_ = r.Logger.Write("ERROR [llm.CreateChatCompletion]: " + err.Error())
			}
			return nil, fmt.Errorf("error: %w", err)
		}
		if len(resp.Choices) == 0 {
			if r.Logger != nil {
				_ = r.Logger.Write("ERROR [llm.CreateChatCompletion]: no choices in response")
			}
			return nil, fmt.Errorf("no choices in response")
		}

		debug := strings.EqualFold(os.Getenv("DEBUG_LLM"), "1") ||
			strings.EqualFold(os.Getenv("DEBUG_LLM"), "true") ||
			strings.EqualFold(os.Getenv("DEBUG_LLM"), "yes")
		if debug {
			if data, mErr := json.MarshalIndent(resp, "", "  "); mErr == nil {
				fmt.Fprintln(os.Stderr, string(data))
			} else if r.Logger != nil {
				_ = r.Logger.Write("ERROR [json.MarshalIndent(resp)]: " + mErr.Error())
			}
		} else {
			// Keep it readable: show either tools being called or the final answer snippet.
			msg := resp.Choices[0].Message
			if trace {
				if len(msg.ToolCalls) > 0 {
					fmt.Fprintf(os.Stderr, "A(tool): %s\n", summarizeToolCalls(msg.ToolCalls))
				} else {
					fmt.Fprintf(os.Stderr, "A: %s\n", compactText(msg.Content, 800))
				}
			} else {
				// Minimal single-line trace.
				finish := resp.Choices[0].FinishReason
				contentLen := len(msg.Content)
				toolCalls := len(msg.ToolCalls)
				fmt.Fprintf(os.Stderr, "llm: model=%s finish=%s content_chars=%d tool_calls=%d tokens=%d (prompt=%d completion=%d)\n",
					resp.Model,
					finish,
					contentLen,
					toolCalls,
					resp.Usage.TotalTokens,
					resp.Usage.PromptTokens,
					resp.Usage.CompletionTokens,
				)
			}
		}

		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			answer = msg.Content
			inCycle = false
			continue
		}

		// TODO: Рассмотреть возможность сделать более эффективно
		// Append assistant's message with tool calls to history
		messages = append(messages, msg)

		for _, tc := range msg.ToolCalls {
			toolKey := tc.Function.Name + ":" + tc.Function.Arguments
			if toolKey == lastToolKey {
				repeatSameTool++
			} else {
				repeatSameTool = 0
				lastToolKey = toolKey
			}

			// If the model is repeating the exact same tool call, return cached output (if available)
			// and nudge it to continue without calling tools again.
			if cached, ok := toolCache[toolKey]; ok && repeatSameTool >= 1 {
				if trace {
					fmt.Fprintf(os.Stderr, "Tool(%s) -> <cached>\n", tc.Function.Name)
				}
				toolMsg := openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    cached + "\n\n(Замечание: этот вызов инструмента уже выполнялся с теми же аргументами. Не повторяй его; продолжай рассуждение и дай итоговый ответ на основе имеющихся данных.)",
					ToolCallID: tc.ID,
				}
				messages = append(messages, toolMsg)

				// After a few identical repeats, force a turn of summarization.
				if repeatSameTool >= 3 {
					messages = append(messages, openai.ChatCompletionMessage{
						Role: openai.ChatMessageRoleUser,
						Content: "Ты зациклился на повторяющемся вызове инструмента. НЕ вызывай инструменты снова. " +
							"Сформируй итоговый ответ/файл на основе уже полученных данных (или явно укажи, что данных недостаточно из-за блокировок сайтов).",
					})
				}
				// If it keeps going, stop.
				if repeatSameTool >= 6 {
					return nil, fmt.Errorf("tool loop detected: repeated identical tool call %d times: %s", repeatSameTool+1, tc.Function.Name)
				}
				continue
			}

			toolMsg := r.dispatchToolCall(tc)
			if trace {
				fmt.Fprintf(os.Stderr, "Tool(%s) -> %s\n", tc.Function.Name, compactText(toolMsg.Content, 500))
			}
			toolCache[toolKey] = toolMsg.Content
			messages = append(messages, toolMsg)
		}
	}

	return &answer, nil
}

func compactText(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "<empty>"
	}
	// collapse whitespace to single spaces
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func summarizeToolCalls(calls []openai.ToolCall) string {
	names := make([]string, 0, len(calls))
	for _, tc := range calls {
		n := strings.TrimSpace(tc.Function.Name)
		if n == "" {
			n = "<unknown>"
		}
		names = append(names, n)
	}
	return strings.Join(names, ", ")
}
