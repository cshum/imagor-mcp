// Package imagormcp exists to carry the skill into the binary.
//
// The skill is the part of using this server that does not fit in a tool description:
// the order a path is written in, how overlays and text are placed, and a working
// method. A client that only reads tools/list never sees it, so it is embedded here
// and served as a resource, and the server's instructions point at it.
package imagormcp

import "embed"

// The skill travels with the server: a client that reads only the tool list never sees
// a file, and these carry what a tool description cannot.
//
//go:embed SKILL.md
var Skill string

// Sources is the fixture table. It is separate from the skill because every agent that
// touches this server reads the skill, while only an agent choosing what to put in an
// image field wants sixteen rows of names and dimensions.
//
//go:embed references/sources.md
var Sources string

// Cookbook is the recipe set. A recipe is the agent's intermediate channel - it is what
// gets handed to render_template, not what a reader of the cookbook page wants - so the
// page carries the name and what the card is for, and the template arrives by name.
//
//go:embed cookbook/*.json
var Cookbook embed.FS

//go:embed references/cookbook.md
var CookbookIndex string
