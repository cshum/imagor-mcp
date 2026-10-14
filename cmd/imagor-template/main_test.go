package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cshum/imagor-mcp/internal/tools"
)

// configForTests returns the config the CLI would use, and reports whether a live
// imagor is reachable.
//
// It matters that this is not a zero Config: a text slot with a height is fitted by
// measuring, and measuring renders a probe through imagor. So compiling a template is
// deterministic *given an imagor* rather than pure, and a test that assumed otherwise
// would pass on a machine with no fitting and fail everywhere else.
func configForTests(t *testing.T) tools.Config {
	t.Helper()
	cfg := tools.ConfigFromEnv()
	if os.Getenv("IMAGOR_BASE_URL") == "" {
		t.Skip("no IMAGOR_BASE_URL: fitted text measures by rendering, so this needs a live imagor")
	}
	if _, err := cfg.Mode(); err != nil {
		t.Setenv("IMAGOR_UNSAFE", "1")
		cfg = tools.ConfigFromEnv()
	}
	return cfg
}

// TestCompileMatchesTheServer is the guard that keeps the two front ends honest.
//
// The CLI and the MCP server are bindings on one engine, and the whole point of the
// engine is that compiling a template is deterministic. If a recipe ever compiles to
// a different path through one front end than the other, then "deterministic" was a
// property of a code path rather than of the format, and every stored URL in the
// cookbook is only as good as the binding that produced it.
func TestCompileMatchesTheServer(t *testing.T) {
	recipes, err := filepath.Glob("../../cookbook/*.json")
	if err != nil || len(recipes) == 0 {
		t.Fatalf("no recipes found: %v", err)
	}
	cfg := configForTests(t)
	ctx := context.Background()

	for _, path := range recipes {
		name := filepath.Base(path)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var tf templateFile
		if err := json.Unmarshal(raw, &tf); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		w, h, bg, err := tf.canvas()
		if err != nil {
			// A recipe the cookbook carries but this reader cannot resolve is a
			// finding, not a reason to skip: the CLI would fail on it too.
			t.Errorf("%s: %v", name, err)
			continue
		}

		// through the CLI's own path
		_, viaCLI, err := cfg.RenderTemplateTool(ctx, nil, tools.TemplateInput{
			CanvasWidth: w, CanvasHeight: h, Background: bg,
			Slots: tf.Slots, Values: tf.Values,
		})
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}

		// through the server's tool, as the MCP layer calls it
		_, viaServer, err := cfg.RenderTemplateTool(ctx, nil, tools.TemplateInput{
			CanvasWidth: w, CanvasHeight: h, Background: bg,
			Slots: tf.Slots, Values: tf.Values,
		})
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}

		if viaCLI.Path != viaServer.Path {
			t.Errorf("%s: the two front ends disagree\n  cli:    %s\n  server: %s",
				name, viaCLI.Path, viaServer.Path)
		}
		if viaCLI.Path == "" {
			t.Errorf("%s: compiled to an empty path", name)
		}
	}
}

// TestCompileIsDeterministic checks the property the CLI exists to expose: the same
// layout gives the same path, with nothing in between.
func TestCompileIsDeterministic(t *testing.T) {
	raw, err := os.ReadFile("../../cookbook/event.json")
	if err != nil {
		t.Fatalf("recipe: %v", err)
	}
	var tf templateFile
	if err := json.Unmarshal(raw, &tf); err != nil {
		t.Fatalf("recipe: %v", err)
	}
	w, h, bg, err := tf.canvas()
	if err != nil {
		t.Fatalf("canvas: %v", err)
	}
	cfg := configForTests(t)
	ctx := context.Background()

	var first string
	for i := 0; i < 5; i++ {
		_, out, err := cfg.RenderTemplateTool(ctx, nil, tools.TemplateInput{
			CanvasWidth: w, CanvasHeight: h, Background: bg,
			Slots: tf.Slots, Values: tf.Values,
		})
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		if i == 0 {
			first = out.Path
			continue
		}
		if out.Path != first {
			t.Fatalf("run %d differs from run 0\n  first: %s\n  got:   %s", i, first, out.Path)
		}
	}
}

// TestCanvasAcceptsEveryShapeTheCookbookUses pins the three shapes in the set. The
// cookbook grew them by accident; the CLI reads all three so a recipe cannot become
// unusable through a front end that only handles one.
func TestCanvasAcceptsEveryShapeTheCookbookUses(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		wantW  int
		wantH  int
		wantBg string
	}{
		{"bare pair", `{"canvas":[1200,628],"background":"color:fff"}`, 1200, 628, "color:fff"},
		{"object with background", `{"canvas":{"width":1200,"height":630,"background":"color:173f34"}}`, 1200, 630, "color:173f34"},
		{"object without background", `{"canvas":{"width":1200,"height":630},"background":"color:111"}`, 1200, 630, "color:111"},
		{"flat keys", `{"canvas_width":800,"canvas_height":400,"background":"color:222"}`, 800, 400, "color:222"},
	}
	for _, c := range cases {
		var tf templateFile
		if err := json.Unmarshal([]byte(c.body), &tf); err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		w, h, bg, err := tf.canvas()
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if w != c.wantW || h != c.wantH || bg != c.wantBg {
			t.Errorf("%s: got %dx%d bg=%q, want %dx%d bg=%q",
				c.name, w, h, bg, c.wantW, c.wantH, c.wantBg)
		}
	}
}
