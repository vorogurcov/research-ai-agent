package grpc

import (
	"context"
	"strings"

	agentv1 "github.com/vorogurcov/ai-agent/api/gen/agent/v1"
	"github.com/vorogurcov/ai-agent/internal/agent"
	"github.com/vorogurcov/ai-agent/internal/config"
)

type AgentGRPCServer struct {
	cfg    *config.Config
	runner *agent.Runner
	agentv1.UnimplementedResearchAgentServer
}

func NewAgentGRPCServer(cfg *config.Config, runner *agent.Runner) *AgentGRPCServer {
	return &AgentGRPCServer{
		cfg:    cfg,
		runner: runner,
	}
}

func (a AgentGRPCServer) SendPrompt(ctx context.Context, prompt *agentv1.Prompt) (*agentv1.PromptAnswer, error) {
	userPrompt := strings.TrimSpace(prompt.Prompt)
	if ans, err := a.runner.Run(a.cfg.ModelName, a.cfg.SystemPrompt, userPrompt); err != nil {
		return nil, err
	} else {
		answer := agentv1.PromptAnswer{Answer: *ans}
		return &answer, nil
	}
}
