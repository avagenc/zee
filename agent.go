package zee

import (
	_ "embed"
	"fmt"

	adktuya "go.naturallyfunny.dev/agentkit/tuya/adk"
	tuya "go.naturallyfunny.dev/tuya"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
)

//go:embed internal/description.txt
var description string

//go:embed internal/instruction.txt
var systemInstruction string

// Config holds the dependencies a consumer must supply. Zee's identity and
// system instruction are owned by the module, not configured here. Per-channel
// or per-run instruction is the consumer's concern — append it from a
// llmagent.BeforeModelCallback / plugin on your own runner.
type Config struct {
	Model      model.LLM
	TuyaClient *tuya.Client
	// AdditionalInstruction is appended to Zee's base system instruction.
	// Use it to supply channel-specific or deployment-specific context that
	// the module itself cannot know.
	AdditionalInstruction string
}

// New builds the Zee agent — a Tuya smart-home LLM agent. Running it (runner,
// session, and any per-run instruction) is the consumer's responsibility.
func New(cfg Config) (agent.Agent, error) {
	tools, err := adktuya.Tools(cfg.TuyaClient)
	if err != nil {
		return nil, fmt.Errorf("zee: tuya tools: %w", err)
	}

	instruction := "[SYSTEM_INSTRUCTION]" + systemInstruction + "\n[/SYSTEM_INSTRUCTION]"
	if cfg.AdditionalInstruction != "" {
		instruction = "[SYSTEM_INSTRUCTION]" + systemInstruction + "\n\n" + cfg.AdditionalInstruction + "\n[/SYSTEM_INSTRUCTION]"
	}

	a, err := llmagent.New(llmagent.Config{
		Name:        "zee",
		Model:       cfg.Model,
		Tools:       tools,
		Description: description,
		Instruction: instruction,
	})
	if err != nil {
		return nil, fmt.Errorf("zee: agent: %w", err)
	}
	return a, nil
}
