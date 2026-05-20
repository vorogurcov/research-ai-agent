package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vorogurcov/ai-agent/internal/agent"
	"github.com/vorogurcov/ai-agent/internal/config"
	"github.com/vorogurcov/ai-agent/internal/llm"
	"github.com/vorogurcov/ai-agent/internal/tools"
	logger2 "github.com/vorogurcov/ai-agent/internal/utils/logger"
)

func main() {
	var modelFlag string

	flag.StringVar(&modelFlag, "m", "", "Model name")
	flag.Parse()

	cfg, err := config.Load(config.LoadParams{
		ModelFlag: modelFlag,
		Prompt:    "asd",
	})

	logger, _ := logger2.GetLogger(filepath.Join(cfg.WorkspaceRoot, "log", "log.txt"), "PROD")
	_ = func() error {
		if logger == nil {
			return nil
		}
		return logger.WriteNonPrettified("Logs from your program will appear here!")
	}()

	if err != nil {
		if logger != nil {
			_ = logger.WriteNonPrettified("ERROR [config.Load]: " + err.Error())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	searchLogger, _ := logger2.GetLogger(filepath.Join(cfg.WorkspaceRoot, "log", ".search_log.txt"), "PROD")
	_ = func() error {
		if searchLogger == nil {
			return nil
		}
		return searchLogger.WriteNonPrettified("Search logs from your program will appear here!")
	}()

	client := llm.NewAPIClient(cfg.AIApiKey, cfg.APIBaseURL)
	reg := tools.NewRegistry(cfg.WorkspaceRoot, logger, searchLogger)

	runner := agent.Runner{
		Client: client,
		Tools:  reg.Tools(),
		Caller: reg,
		Logger: logger,
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("[Agent] Started! Listening for prompts!")

	for scanner.Scan() {
		prompt := strings.TrimSpace(scanner.Text())
		if ans, err := runner.Run(cfg.ModelName, cfg.SystemPrompt, prompt); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		} else {
			fmt.Println(ans)
		}

	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Ошибка чтения stdin:", err)
	}
	fmt.Println("[Agent] Поток stdin закрыт. Завершение работы агента.")
}
