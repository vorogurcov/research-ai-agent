package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	agentv1 "github.com/vorogurcov/ai-agent/api/gen/agent/v1"
	"github.com/vorogurcov/ai-agent/internal/agent"
	"github.com/vorogurcov/ai-agent/internal/config"
	"github.com/vorogurcov/ai-agent/internal/llm"
	"github.com/vorogurcov/ai-agent/internal/tools"
	agentGrpc "github.com/vorogurcov/ai-agent/internal/transport/grpc"
	logger2 "github.com/vorogurcov/ai-agent/internal/utils/logger"
	"google.golang.org/grpc"
)

func main() {
	var modelFlag string

	flag.StringVar(&modelFlag, "m", "", "Model name")
	flag.Parse()

	cfg, err := config.Load(config.LoadParams{
		ModelFlag: modelFlag,
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
	fmt.Println("[Agent] Started! Listening for prompts!")

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 3333))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	agentv1.RegisterResearchAgentServer(grpcServer, agentGrpc.NewAgentGRPCServer(&cfg, &runner))
	grpcServer.Serve(lis)

	fmt.Println("[Agent] Поток stdin закрыт. Завершение работы агента.")
}
