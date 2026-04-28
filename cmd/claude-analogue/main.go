package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/codecrafters-io/claude-code-starter-go/internal/agent"
	"github.com/codecrafters-io/claude-code-starter-go/internal/config"
	"github.com/codecrafters-io/claude-code-starter-go/internal/llm"
	"github.com/codecrafters-io/claude-code-starter-go/internal/tools"
)

func main() {
	var prompt string
	var modelFlag string

	flag.StringVar(&prompt, "p", "", "Prompt to send to LLM")
	flag.StringVar(&modelFlag, "m", "", "Model name")
	flag.Parse()

	cfg, err := config.Load(config.LoadParams{
		ModelFlag: modelFlag,
		Prompt:    prompt,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")

	client := llm.NewAPIClient(cfg.AIApiKey, cfg.APIBaseURL)
	reg := tools.NewRegistry(cfg.WorkspaceRoot)

	runner := agent.Runner{
		Client: client,
		Tools:  reg.Tools(),
		Caller: reg,
	}

	if err := runner.Run(cfg.ModelName, cfg.SystemPrompt, cfg.Prompt); err != nil {
		log.Fatal(err)
	}
}
