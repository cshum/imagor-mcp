// Package imagormcp exists to carry the skill into the binary.
//
// The skill is the part of using this server that does not fit in a tool
// description: the order a path is written in, how overlays and text are placed,
// and a working method. A client that only reads tools/list never sees it, so it
// is embedded here and served as a resource, and the server's instructions point
// at it.
package imagormcp

import _ "embed"

//go:embed SKILL.md
var Skill string
