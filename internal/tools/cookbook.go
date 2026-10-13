package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	imagormcp "github.com/cshum/imagor-template"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The cookbook is the agent's channel, not a reader's. A recipe is what gets handed to
// render_template - it is not the product and it is not something a person copies by
// hand. So the page lists the name and what the card is for, and the template arrives
// by name through this tool. The recipes travel in the binary, which means a client
// cannot end up holding a template from a version the server is not running.

// CookbookInput names a recipe to fetch.
type CookbookInput struct {
	Name string `json:"name" jsonschema:"a recipe name from the cookbook, for example event or og-standard. Read the imagor-mcp://cookbook resource for the list, what each one is for, and which values it takes."`
}

// CookbookOutput is one recipe, ready to hand to render_template.
type CookbookOutput struct {
	Name string `json:"name"`
	Note string `json:"note,omitempty"`
	// Template is the recipe as written: canvas, slots and values, with the slot names
	// and {{placeholders}} intact. Pass it to render_template unchanged to see what it
	// looks like, then change the values.
	//
	// A map rather than json.RawMessage: the tool's output schema is derived from this
	// struct, and RawMessage is a []byte to the schema generator, which then rejects the
	// object it was handed. A map has no such problem and gives the caller real JSON.
	Template map[string]any `json:"template"`
	// Render is the call that uses it, so a caller does not have to reconstruct the
	// shape of a render_template invocation from the template alone.
	Render string `json:"render,omitempty"`
}

// CookbookNames lists the recipes the binary carries.
func CookbookNames() []string {
	entries, err := imagormcp.Cookbook.ReadDir("cookbook")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			names = append(names, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	sort.Strings(names)
	return names
}

// CookbookTool returns a recipe by name.
//
// A name that is not in the set is an error rather than an empty result, and the error
// lists what is available: a caller that guessed a name has one obvious next move, and
// an empty template would be handed to render_template and fail there instead.
func (c Config) CookbookTool(_ context.Context, _ *mcp.CallToolRequest, in CookbookInput) (*mcp.CallToolResult, CookbookOutput, error) {
	name := strings.TrimSpace(strings.ToLower(in.Name))
	name = strings.TrimSuffix(name, ".json")
	if name == "" {
		return nil, CookbookOutput{}, fmt.Errorf("no recipe name given; available: %s",
			strings.Join(CookbookNames(), ", "))
	}
	// path.Clean and the explicit check keep a name like ../../etc/passwd from reaching
	// the embedded filesystem, which would otherwise be a read primitive.
	if name != path.Base(name) || strings.Contains(name, "..") {
		return nil, CookbookOutput{}, fmt.Errorf("%q is not a recipe name", in.Name)
	}
	raw, err := imagormcp.Cookbook.ReadFile("cookbook/" + name + ".json")
	if err != nil {
		return nil, CookbookOutput{}, fmt.Errorf("no recipe %q; available: %s",
			name, strings.Join(CookbookNames(), ", "))
	}
	var tmpl map[string]any
	if err := json.Unmarshal(raw, &tmpl); err != nil {
		return nil, CookbookOutput{}, fmt.Errorf("recipe %q is not valid JSON in this build: %w", name, err)
	}
	// The note is lifted out for the same reason it exists in the format: it says what
	// the card is for, so a caller can judge a change against something.
	var meta struct {
		Note string `json:"note"`
	}
	_ = json.Unmarshal(raw, &meta)
	return nil, CookbookOutput{
		Name:     name,
		Note:     meta.Note,
		Template: tmpl,
		Render: "render_template with this template's canvas, slots and values, " +
			"at the size you need",
	}, nil
}
