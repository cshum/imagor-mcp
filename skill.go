package imagorcomposemcp

import _ "embed"

// The skill travels with the server: a client that reads only the tool list never sees
// a file, and these carry what a tool description cannot.
//
//go:embed SKILL.md
var Skill string

// Sources is the fixture table. It is separate from the skill because every agent that
// touches this server reads the skill, while only an agent choosing what to put in an
// image field wants sixteen rows of names and dimensions.
//
//go:embed docs/sources.md
var Sources string
