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
	Prompt    string
	ModelFlag string
}

func Load(p LoadParams) (Config, error) {
	// Best-effort load local env; ok if missing.
	_ = godotenv.Load(".env")

	if p.Prompt == "" {
		return Config{}, errors.New("Prompt must not be empty")
	}

	model := p.ModelFlag
	if model == "" {
		model = os.Getenv("MODEL_NAME")
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
	root := filepath.Clean(cwd)

	return Config{
		Prompt:        p.Prompt,
		ModelName:     model,
		AIApiKey:      apiKey,
		APIBaseURL:    baseURL,
		WorkspaceRoot: root,
		SystemPrompt: `Ты - автономный агент-разработчик. 
Сначала попытайся собрать информацию по проекту, а не галлюционируй. Это ОБЯЗАТЕЛЬНО.
Для выполнения ЛЮБЫХ действий с файлами (чтение, запись, выполнение команд) ты ОБЯЗАН использовать предоставленные инструменты.
`,
	}, nil
}
