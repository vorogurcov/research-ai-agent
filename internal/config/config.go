package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	Prompt string

	ModelName string

	AIApiKey   string
	APIBaseURL string

	WorkspaceRoot string
	SystemPrompt  string
}

type LoadParams struct {
	Prompt       string
	ModelFlag    string
	SystemPrompt string
}

func Load(p LoadParams) (Config, error) {
	// Best-effort load local env; ok if missing.
	_ = godotenv.Load(".env")

	if p.Prompt == "" {
		return Config{}, errors.New("Prompt must not be empty")
	}
	if p.SystemPrompt == "" {
		p.SystemPrompt = `Ты — основной ассистент агента. Отвечай по делу, структурированно и без лишнего текста.

Инструменты (вызовы функций):
- Search({ "query": string, "page"?: int }): используй для поиска по текстовому запросу. Возвращает HTML поисковой выдачи (строкой).
- Scrape({ "url": string, "needs": string }): используй, чтобы открыть конкретный URL в браузере (JS/SPA поддерживаются) и извлечь ТОЛЬКО релевантную информацию по needs. Возвращает только валидный JSON (без пояснений).
- Write({ "task_name": string, "file_path": string, "content": string }): используй, чтобы сохранять артефакты. Пишет строго в writes/{task_name}_{datetime}_{index}/. Возвращает относительный путь записанного файла.
- Read({ "file_path": string }): используй, чтобы прочитать ранее записанный артефакт. Читает строго из writes/. Передавай относительный путь, который вернул Write.

Правила использования инструментов:
- Сначала Search, чтобы найти релевантные источники и ссылки. Затем Scrape по 1–3 самым релевантным URL.
- Scrape вызывай только для http/https URL. Если сайт блокирует (403/429/anti-bot) — попробуй другой источник.
- Не проси пользователя “вставить текст страницы”: для этого есть Scrape.
- Файлы эксперимента: всегда запоминай/копируй пути, которые вернул Write, и используй их для Read в рамках ЭТОГО запуска. Не пытайся угадывать datetime — ориентируйся на возвращённые пути.
- Не зацикливайся на повторении одного и того же вызова инструмента. Если данных недостаточно — явно скажи, чего не хватает и почему.`
	}
	model := p.ModelFlag
	if model == "" {
		model = os.Getenv("THINKING_MODEL_NAME")
	}
	if model == "" {
		return Config{}, errors.New("Model must not be empty")
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return Config{}, errors.New("Env variable OPENROUTER_API_KEY not found")
	}

	baseURL := os.Getenv("OPENROUTER_BASE_URL")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}

	cwd, err := os.Getwd()
	if err != nil {
		return Config{}, fmt.Errorf("failed to get working directory: %w", err)
	}
	rootLog := filepath.Clean(cwd) + "/log"

	return Config{
		Prompt:        p.Prompt,
		ModelName:     model,
		AIApiKey:      apiKey,
		APIBaseURL:    baseURL,
		WorkspaceRoot: rootLog,
		SystemPrompt:  p.SystemPrompt,
	}, nil
}
