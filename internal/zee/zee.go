package zee

import _ "embed"

const Name = "Zee"

//go:embed user-interaction-instruction.xml
var UserInstruction string

//go:embed ava-interaction-instruction.xml
var AvaInstruction string
