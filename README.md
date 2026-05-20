# claude_analogue

Консольный агент на Go с tool calling через OpenAI-совместимый API. Читает запросы из stdin, ходит в LLM, при необходимости вызывает инструменты (поиск, скрейпинг, файлы) и печатает ответ в stdout.

Модуль в `go.mod`: `github.com/vorogurcov/ai-agent`.

## Требования

* Go 1.26+
* Ключ API и модель с поддержкой function calling
* Yandex Cloud Search API: IAM-токен и folder id
* Для `Scrape`: установленный Chromium/Chrome (используется `chromedp`)
* Опционально: bash и `agent.sh` для интерактивного меню (coproc, Linux/macOS/Git Bash)

## Быстрый старт

Создайте `.env` в корне репозитория (файл в `.gitignore`):

```env
AI_API_KEY=sk-...
THINKING_MODEL_NAME=provider/model-name

YANDEX_IAM_TOKEN=...
YANDEX_FOLDER_ID=...

# опционально
AI_API_BASE_URL=https://openrouter.ai/api/v1
ANALYZING_MODEL_NAME=provider/cheaper-model
```

Сборка:

```bash
go build -o agent.exe ./cmd/claude-analogue/main.go
```

Запуск одной строкой:

```bash
echo "что такое go modules" | ./agent.exe
```

Модель можно переопределить флагом:

```bash
./agent.exe -m provider/model-name
```

## agent.sh

Скрипт держит процесс агента в coproc и даёт меню: start / stop / prompt / quit.

```bash
chmod +x agent.sh
./agent.sh
```

Переменные:

* `AGENT_EXE_FILENAME` (по умолчанию `agent.exe`)
* `AGENT_MAIN_FOLDER` (по умолчанию `./cmd/claude-analogue/main.go`)

Если бинарника нет, скрипт сам вызывает `go build`.