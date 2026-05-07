package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

		// Это запрос: используем Tavily Search.
		r.appendSearchLog("YANDEX search start")
		resp, err := yandexClient.Search(ctx, p.Query, intOrDefault(&p.Page, 0))

		if err != nil {
			r.writeError("tools.Search.yandex.search", err)
			r.appendSearchLog(fmt.Sprintf("ERROR yandex search: %v", err))
			return "", err
		}
		var sr struct {
			RawData string `json:"rawData"`
		}
		if err := json.Unmarshal([]byte(resp), &sr); err != nil {
			err = fmt.Errorf("failed to decode yandex response json: %w", err)
			r.writeError("tools.Search.yandex.unmarshal", err)
			r.appendSearchLog(fmt.Sprintf("ERROR yandex unmarshal: %v", err))
			return "", err
		}
		xmlBytes, err := base64.StdEncoding.DecodeString(sr.RawData)
		if err != nil {
			err = fmt.Errorf("failed to decode yandex rawData (base64): %w", err)
			r.writeError("tools.Search.yandex.base64", err)
			r.appendSearchLog(fmt.Sprintf("ERROR yandex base64: %v", err))
			return "", err
		}

		return string(xmlBytes), nil
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
	// Best-effort logging; never break tool execution because of log IO.
	if r == nil {
		return
	}
	path := filepath.Join(r.workspaceRoot, ".search_log.txt")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	ts := time.Now().Format(time.RFC3339)
	_, _ = f.WriteString(ts + " " + line + "\n")
}
