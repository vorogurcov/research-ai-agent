package tools

import (
	"context"
	"fmt"
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
			r.SearchLogger.WriteNonPrettified(fmt.Sprintf("ERROR mustUnmarshal: %v | rawArgs=%q", err, rawArgs))
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		r.SearchLogger.WriteNonPrettified(fmt.Sprintf("CALL query=%q page=%d", p.Query, intOrDefault(&p.Page, 0)))

		if p.Query == "" {
			err := fmt.Errorf("query must not be empty")
			r.writeError("tools.Search.validate", err)
			r.SearchLogger.WriteNonPrettified("ERROR validate: query is empty")
			return "", err
		}

		yandexClient, err := yandex.NewYandexSearchClientFromEnv()
		if err != nil {
			r.writeError("tools.Search.tavily.env", err)
			r.SearchLogger.WriteNonPrettified(fmt.Sprintf("ERROR tavily env: %v", err))
			return "", err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()

		r.SearchLogger.WriteNonPrettified("YANDEX search start")
		resp, err := yandexClient.Search(ctx, p.Query, intOrDefault(&p.Page, 0))

		if err != nil {
			r.writeError("tools.Search.yandex.search", err)
			r.SearchLogger.WriteNonPrettified(fmt.Sprintf("ERROR yandex search: %v", err))
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

func (r *Registry) appendSearchLogPayload(toolName string, payload string) {
	if r == nil {
		return
	}
	_ = r.SearchLogger.WriteNonPrettified(fmt.Sprintf("PAYLOAD %s", toolName))
	_ = r.SearchLogger.WritePrettified(payload)
	_ = r.SearchLogger.WriteNonPrettified(fmt.Sprintf("PAYLOAD %s END\n", toolName))
}
