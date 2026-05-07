package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/vorogurcov/ai-agent/internal/search/yandex"
)

func (r *Registry) registerSearch() {
	r.registerTool(openai.FunctionDefinition{
		Name:        "Search",
		Description: "Возвращает поисковые результаты по запросу, содержащие возможные источники для получения информации(сайти и краткое описание).",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"query": {
					Type:        jsonschema.String,
					Description: "Поисковый запрос, может содержать дорки",
				},
				"page": {
					Type:        jsonschema.Integer,
					Description: "Номер поисковой страницы, по умолчанию 0, т.е. первая страница",
				},
			},
			Required: []string{"query"},
		},
	}, func(rawArgs string) (string, error) {
		type props struct {
			Query string `json:"query"`
			Page  string `json:"page"`
		}
		var p props
		if err := mustUnmarshal(rawArgs, &p); err != nil {
			r.writeError("tools.Search.mustUnmarshal", err)
			r.appendSearchLog(fmt.Sprintf("ERROR mustUnmarshal: %v | rawArgs=%q", err, rawArgs))
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		r.appendSearchLog(fmt.Sprintf("CALL query=%q page=%d", p.Query, intOrDefault(&p.Page, 0)))

		if p.Query == "" {
			err := fmt.Errorf("query must not be empty")
			r.writeError("tools.Search.validate", err)
			r.appendSearchLog("ERROR validate: query is empty")
			return "", err
		}

		yandexClient, err := yandex.NewYandexSearchClientFromEnv()
		if err != nil {
			r.writeError("tools.Search.tavily.env", err)
			r.appendSearchLog(fmt.Sprintf("ERROR tavily env: %v", err))
			return "", err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()

		r.appendSearchLog("YANDEX search start")
		resp, err := yandexClient.Search(ctx, p.Query, intOrDefault(&p.Page, 0))

		if err != nil {
			r.writeError("tools.Search.yandex.search", err)
			r.appendSearchLog(fmt.Sprintf("ERROR yandex search: %v", err))
			return "", err
		}

		r.appendSearchLogPayload("Search", resp)
		return resp, nil
	})
}

func intOrDefault(p *string, def int) int {
	if p == nil {
		return def
	}

	val, _ := strconv.Atoi(*p)
	return val
}

func (r *Registry) appendSearchLog(line string) {
	if r == nil {
		return
	}
	path := r.searchLogPath()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	ts := time.Now().Format(time.RFC3339)
	_, _ = f.WriteString(ts + " " + line + "\n")
}

func (r *Registry) appendSearchLogPayload(toolName string, payload string) {
	if r == nil {
		return
	}
	prettyPayload := prettifyPayloadForLog(payload)
	path := r.searchLogPath()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	ts := time.Now().Format(time.RFC3339)
	_, _ = f.WriteString(fmt.Sprintf("%s PAYLOAD %s BEGIN chars=%d pretty_chars=%d\n", ts, toolName, len(payload), len(prettyPayload)))
	_, _ = f.WriteString(prettyPayload)
	if !strings.HasSuffix(prettyPayload, "\n") {
		_, _ = f.WriteString("\n")
	}
	_, _ = f.WriteString(fmt.Sprintf("%s PAYLOAD %s END\n", ts, toolName))
}

func (r *Registry) searchLogPath() string {
	logDir := filepath.Join(r.workspaceRoot, "log")
	if err := os.MkdirAll(logDir, 0o755); err == nil {
		return filepath.Join(logDir, ".search_log.txt")
	}
	// Fallback to workspace root if log dir is unavailable.
	return filepath.Join(r.workspaceRoot, ".search_log.txt")
}

func prettifyPayloadForLog(payload string) string {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return payload
	}

	var jsonBuf bytes.Buffer
	if json.Valid([]byte(trimmed)) && json.Indent(&jsonBuf, []byte(trimmed), "", "  ") == nil {
		return jsonBuf.String()
	}

	if strings.HasPrefix(trimmed, "<") {
		if pretty, err := prettifyXML(trimmed); err == nil {
			return pretty
		}
	}

	return payload
}

func prettifyXML(raw string) (string, error) {
	dec := xml.NewDecoder(strings.NewReader(raw))
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if err := enc.EncodeToken(tok); err != nil {
			return "", err
		}
	}

	if err := enc.Flush(); err != nil {
		return "", err
	}
	return buf.String(), nil
}
