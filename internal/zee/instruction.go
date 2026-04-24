package zee

import _ "embed"

const Name = "Zee"

//go:embed base-instruction.xml
var baseInstruction string

//go:embed user-interaction-instruction.xml
var userInteraction string

//go:embed ava-interaction-instruction.xml
var avaInteraction string

type options struct {
	interaction string
}

type Option func(*options)

func ForAva() Option {
	return func(o *options) {
		o.interaction = avaInteraction
	}
}

func SystemInstruction(opts ...Option) string {
	cfg := &options{
		interaction: userInteraction,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return "<system_instruction>\n" + baseInstruction + "\n" + cfg.interaction + "\n</system_instruction>"
}
