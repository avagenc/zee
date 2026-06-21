package zee

import (
	"context"
	_ "embed"
	"fmt"

	adktuya "go.naturallyfunny.dev/adk/tuya"
	tuya "go.naturallyfunny.dev/tuya"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
)

//go:embed internal/system-instruction.txt
var systemInstruction string

type Config struct {
	Name               string
	AppName            string
	Description        string
	ChannelInstruction string
	Model              model.LLM
	Session            session.Service
	TuyaClient         *tuya.Client
}

// NewAgent builds the LLM agent without wiring a runner or session service.
// Useful when the caller manages the session lifecycle (e.g. ADK web launcher).
func NewAgent(cfg Config) (agent.Agent, error) {
	tools, err := adktuya.Tools(cfg.TuyaClient)
	if err != nil {
		return nil, fmt.Errorf("zee: tuya tools: %w", err)
	}

	instruction := "[SYSTEM_INSTRUCTION]" + systemInstruction + "\n" + cfg.ChannelInstruction + "\n[/SYSTEM_INSTRUCTION]"

	a, err := llmagent.New(llmagent.Config{
		Name:        cfg.Name,
		Model:       cfg.Model,
		Tools:       tools,
		Description: cfg.Description,
		Instruction: instruction,
	})
	if err != nil {
		return nil, fmt.Errorf("zee: agent: %w", err)
	}
	return a, nil
}

func New(ctx context.Context, cfg Config) (*runner.Runner, error) {
	a, err := NewAgent(cfg)
	if err != nil {
		return nil, err
	}

	return runner.New(runner.Config{
		AppName:           cfg.AppName,
		Agent:             a,
		SessionService:    cfg.Session,
		AutoCreateSession: true,
	})
}
