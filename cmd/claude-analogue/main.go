package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vorogurcov/ai-agent/internal/agent"
	"github.com/vorogurcov/ai-agent/internal/config"
	"github.com/vorogurcov/ai-agent/internal/llm"
	"github.com/vorogurcov/ai-agent/internal/tools"
	logger2 "github.com/vorogurcov/ai-agent/internal/utils/logger"
)

func main() {
	var prompt string
	var modelFlag string

	flag.StringVar(&prompt, "p", "", "Prompt to send to LLM")
	flag.StringVar(&modelFlag, "m", "", "Model name")
	flag.Parse()

	logger, _ := (&logger2.FSLogger{}).GetLogger("log/log.txt", "PROD")
	_ = func() error {
		if logger == nil {
			return nil
		}
		return logger.Write("Logs from your program will appear here!")
	}()

	cfg, err := config.Load(config.LoadParams{
		ModelFlag: modelFlag,
		Prompt:    prompt,
	})
	if err != nil {
		if logger != nil {
			_ = logger.Write("ERROR [config.Load]: " + err.Error())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	l, lerr := (&logger2.FSLogger{}).GetLogger(filepath.Join(cfg.WorkspaceRoot, "log", "log.txt"), "PROD")
	if lerr == nil && l != nil {
		logger = l
		_ = logger.Write("Logger initialized at workspace: " + cfg.WorkspaceRoot)
	}

	client := llm.NewAPIClient(cfg.AIApiKey, cfg.APIBaseURL)
	reg := tools.NewRegistry(cfg.WorkspaceRoot, logger)

	runner := agent.Runner{
		Client: client,
		Tools:  reg.Tools(),
		Caller: reg,
		Logger: logger,
	}
	if ans, err := runner.Run(cfg.ModelName, cfg.SystemPrompt, cfg.Prompt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	} else {
		fmt.Println(ans)
	}
}
