package tools

import (
	"fmt"
	"os"

	"github.com/vorogurcov/ai-agent/internal/agent"
	"github.com/vorogurcov/ai-agent/internal/config"
	"github.com/vorogurcov/ai-agent/internal/llm"
)

func runAnalyzeAgent(html, needs string) (string, error) {
	cfg, err := config.Load(config.LoadParams{
		ModelFlag: os.Getenv("ANALYZING_MODEL_NAME"),
		SystemPrompt: "Ты — узкоспециализированный анализатор текстового содержимого веб-страниц.\n" +
			"Твоя задача: извлекать из предоставленного HTML/текста только полезную информацию, строго по запросу needs, отсекая шум (навигацию, футеры, рекламу).\n" +
			"\n" +
			"КРИТИЧЕСКОЕ ТРЕБОВАНИЕ К ФОРМАТУ ВЫВОДА:\n" +
			"Ты ОБЯЗАН вернуть ТОЛЬКО валидный JSON строго следующей структуры (никакого markdown, никаких пояснений, никаких префиксов):\n" +
			"{\n" +
			"  \"answer\": string,\n" +
			"  \"key_points\": string[],\n" +
			"  \"facts\": string[],\n" +
			"  \"numbers\": string[],\n" +
			"  \"entities\": string[],\n" +
			"  \"links\": string[],\n" +
			"  \"warnings\": string[]\n" +
			"}\n" +
			"\n" +
			"Правила заполнения:\n" +
			"- \"answer\": краткий ответ по needs (если данных нет — пустая строка).\n" +
			"- Остальные поля: всегда массив строк; если данных нет — пустой массив.\n" +
			"- \"links\": только URL (если есть в тексте); не выдумывай ссылки.\n" +
			"- Ничего не выдумывай и не дополняй знаниями извне страницы.\n" +
			"- Не меняй факты, только извлекай и структурируй.\n" +
			"- Если есть сомнения/ограничения качества (шум, paywall, редиректы) — пиши в \"warnings\".",
	})

	prompt := fmt.Sprintf("Твоя задача — извлечь из текста веб-страницы только релевантную информацию по запросу: %q.\n\nТекст страницы:\n%s", needs, html)

	if err != nil {
		return "", err
	}

	client := llm.NewAPIClient(cfg.AIApiKey, cfg.APIBaseURL)

	runner := agent.Runner{
		Client: client,
		Tools:  nil,
		Caller: nil,
		Logger: nil,
	}

	ans, err := runner.Run(cfg.ModelName, cfg.SystemPrompt, prompt)
	if err != nil {
		return "", err
	}
	return *ans, nil
}
