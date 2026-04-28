package agent

import (
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func (r Runner) dispatchToolCall(tc openai.ToolCall) openai.ChatCompletionMessage {
	name := tc.Function.Name
	args := tc.Function.Arguments

	tool := findTool(r.Tools, name)
	if tool == nil || tool.Function == nil {
		return openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    fmt.Sprintf("unknown tool: %s", name),
			ToolCallID: tc.ID,
		}
	}

	if r.Caller == nil {
		return openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    "tool dispatcher not configured",
			ToolCallID: tc.ID,
		}
	}

	out, err := r.Caller.Call(name, args)
	if err != nil {
		return openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    err.Error(),
			ToolCallID: tc.ID,
		}
	}

	return openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    out,
		ToolCallID: tc.ID,
	}
}

func findTool(tools []openai.Tool, name string) *openai.Tool {
	for i := range tools {
		if tools[i].Function != nil && strings.EqualFold(tools[i].Function.Name, name) {
			return &tools[i]
		}
	}
	return nil
}

