package main

import _ "embed"

//go:embed base-instruction.xml
var baseInstruction string

//go:embed for-human-instruction.xml
var humanInteraction string

//go:embed for-ava-instruction.xml
var avaInteraction string

func systemInstruction(forAva bool) string {
	interaction := humanInteraction
	if forAva {
		interaction = avaInteraction
	}
	return "[SYSTEM_INSTRUCTION]" + baseInstruction + "\n" + interaction + "\n[/SYSTEM_INSTRUCTION]"
}
