package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

type Registry struct {
	workspaceRoot string
	handlers      map[string]func(rawArgs string) (string, error)
	tools         []openai.Tool
}

func NewRegistry(workspaceRoot string) *Registry {
	r := &Registry{
		workspaceRoot: workspaceRoot,
		handlers:      map[string]func(rawArgs string) (string, error){},
	}

	r.registerRead()
	r.registerWrite()
	r.registerBash()

	return r
}

func (r *Registry) Tools() []openai.Tool {
	return append([]openai.Tool(nil), r.tools...)
}

func (r *Registry) Call(name string, rawArgs string) (string, error) {
	h, ok := r.handlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return h(rawArgs)
}

func (r *Registry) registerTool(def openai.FunctionDefinition, handler func(rawArgs string) (string, error)) {
	if def.Name == "" {
		panic("tool name is empty")
	}
	if _, exists := r.handlers[def.Name]; exists {
		panic("duplicate tool: " + def.Name)
	}
	r.handlers[def.Name] = handler
	r.tools = append(r.tools, openai.Tool{
		Type:     openai.ToolTypeFunction,
		Function: &def,
	})
}

func mustUnmarshal[T any](raw string, out *T) error {
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}
