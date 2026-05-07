package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
)

type Registry struct {
	workspaceRoot string
	handlers      map[string]func(rawArgs string) (string, error)
	tools         []openai.Tool
	Logger        Logger

	mu            sync.Mutex
	writeSessions map[string]*writeSession
}

type writeSession struct {
	startedAt string
	nextIndex int
}

type Logger interface {
	Write(text string) error
}

func NewRegistry(workspaceRoot string, logger Logger) *Registry {
	r := &Registry{
		workspaceRoot: workspaceRoot,
		handlers:      map[string]func(rawArgs string) (string, error){},
		Logger:        logger,
		writeSessions: map[string]*writeSession{},
	}

	r.registerRead()
	r.registerLs()
	r.registerWrite()
	//r.registerBash()
	r.registerSearch()
	r.registerScrape()
	return r
}

func (r *Registry) nextWriteDir(task string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.writeSessions[task]
	if !ok || s == nil {
		s = &writeSession{
			startedAt: timeNowCompact(),
			nextIndex: 1,
		}
		r.writeSessions[task] = s
	}

	idx := s.nextIndex
	s.nextIndex++

	return fmt.Sprintf("%s_%s_%03d", task, s.startedAt, idx)
}

func timeNowCompact() string {
	// YYYYMMDD_HHMMSS in local time is enough for folder grouping.
	return time.Now().Format("20060102_150405")
}

func (r *Registry) Tools() []openai.Tool {
	return append([]openai.Tool(nil), r.tools...)
}

func (r *Registry) Call(name string, rawArgs string) (string, error) {
	h, ok := r.handlers[name]
	if !ok {
		err := fmt.Errorf("unknown tool: %s", name)
		r.writeError("tool registry", err)
		return "", err
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

func (r *Registry) writeError(context string, err error) {
	if r.Logger == nil || err == nil {
		return
	}
	_ = r.Logger.Write("ERROR [" + context + "]: " + err.Error())
}
