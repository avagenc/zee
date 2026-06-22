package zee

import (
	_ "embed"
	"fmt"

	adktuya "go.naturallyfunny.dev/adk/tuya"
	tuya "go.naturallyfunny.dev/tuya"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
)

//go:embed internal/system-instruction.txt
var systemInstruction string

const (
	name        = "zee"
	description = "Avagenc Tuya Smart Agent"
)

// Config holds the dependencies a consumer must supply. Zee's identity and
// system instruction are owned by the module, not configured here. Per-channel
// or per-run instruction is the consumer's concern — append it from a
// llmagent.BeforeModelCallback / plugin on your own runner.
type Config struct {
	Model      model.LLM
	TuyaClient *tuya.Client
}

// New builds the Zee agent — a Tuya smart-home LLM agent. Running it (runner,
// session, and any per-run instruction) is the consumer's responsibility.
func New(cfg Config) (agent.Agent, error) {
	tools, err := adktuya.Tools(cfg.TuyaClient)
	if err != nil {
		return nil, fmt.Errorf("zee: tuya tools: %w", err)
	}

	instruction := "[SYSTEM_INSTRUCTION]" + systemInstruction + "\n[/SYSTEM_INSTRUCTION]"

	a, err := llmagent.New(llmagent.Config{
		Name:        name,
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
