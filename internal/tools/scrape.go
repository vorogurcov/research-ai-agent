package tools

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func (r *Registry) registerScrape() {
	r.registerTool(openai.FunctionDefinition{
		Name:        "Scrape",
		Description: "Открывает URL в браузере и извлекает только релевантную информацию по needs. Возвращает строго валидный JSON структуры: {answer:string, key_points:string[], facts:string[], numbers:string[], entities:string[], links:string[], warnings:string[]}.",
		Parameters: jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"url": {
					Type:        jsonschema.String,
					Description: "URL страницы",
				},
				"needs": {
					Type:        jsonschema.String,
					Description: "Что именно нужно извлечь со страницы (используется для фильтрации контента).",
				},
			},
			Required: []string{"url", "needs"},
		},
	}, func(rawArgs string) (string, error) {
		type props struct {
			Url   string `json:"url"`
			Needs string `json:"needs"`
		}
		var p props
		if err := mustUnmarshal(rawArgs, &p); err != nil {
			r.writeError("tools.Scrape.mustUnmarshal", err)
			r.appendSearchLog(fmt.Sprintf("ERROR mustUnmarshal: %v | rawArgs=%q", err, rawArgs))
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		r.appendSearchLog(fmt.Sprintf("CALL scrape url=%q", p.Url))

		if p.Url == "" {
			err := fmt.Errorf("url must not be empty")
			r.writeError("tools.Scrape.validate", err)
			r.appendSearchLog("ERROR validate: url is empty")
			return "", err
		}
		if p.Needs == "" {
			err := fmt.Errorf("needs must not be empty")
			r.writeError("tools.Scrape.validate", err)
			r.appendSearchLog("ERROR validate: needs is empty")
			return "", err
		}

		u, parseErr := url.Parse(strings.TrimSpace(p.Url))
		if parseErr != nil || u.Scheme == "" || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			err := fmt.Errorf("invalid url: %q", p.Url)
			r.writeError("tools.Scrape.validate", err)
			r.appendSearchLog("ERROR validate: invalid url")
			return "", err
		}

		ctx, cancel := chromedp.NewContext(context.Background())
		defer cancel()

		ctx, cancel = context.WithTimeout(ctx, time.Minute)
		defer cancel()

		var htmlContent string
		err := chromedp.Run(ctx,
			chromedp.Navigate(p.Url),
			chromedp.WaitVisible(`body`, chromedp.ByQuery),
			chromedp.OuterHTML(`html`, &htmlContent, chromedp.ByQuery),
		)
		if err != nil {
			r.writeError("tools.Scrape.chromedp", err)
			r.appendSearchLog(fmt.Sprintf("ERROR chromedp: %v", err))
			return "", err
		}
		content, err := runAnalyzeAgent(htmlContent, p.Needs)
		if err != nil {
			r.writeError("tools.Scrape.runAnalyzeAgent", err)
			r.appendSearchLog(fmt.Sprintf("ERROR runAnalyzeAgent: %v", err))
			return "", err
		}
		r.appendSearchLog(fmt.Sprintf("DONE ok: output_chars=%d", len(content)))
		return content, nil
	})
}
