// Command imagor-template turns a layout into an imagor URL, and back.
//
// The MCP server is one way to reach the template engine; this is the other. It
// exists because compiling a template is total and deterministic - the same input
// gives the same path every time, with no model, no network and no MCP client - so
// the common case should not need any of them.
//
//	imagor-template compile cookbook/event.json
//	imagor-template decompile '1200x628/filters:...'
//
// Decompiling is the lossy direction: a path holds what a layout resolved to, not
// the names, the {{values}} or the relative forms that produced it.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/cshum/imagor-mcp/internal/tools"
)

const usage = `imagor-template - a layout format for imagor

Usage:
  imagor-template compile <template.json>     a layout, to the imagor path it renders to
  imagor-template decompile <path>            an imagor path, back to a layout
  imagor-template version

Compile is total and deterministic. Decompile is lossy: a path carries the geometry a
layout resolved to, not the slot names, the {{values}} or the relative forms
(38p, half) that produced it.

Environment:
  IMAGOR_SECRET            signing secret, required unless IMAGOR_UNSAFE is set
  IMAGOR_UNSAFE            set to 1 to emit unsafe/ URLs; development only
  IMAGOR_SIGNER_TYPE       sha1 (default), sha256 or sha512
  IMAGOR_SIGNER_TRUNCATE   signature length in characters
`

// templateFile is a recipe on disk. The canvas is accepted in either shape the
// cookbook uses - an object carrying its own background, or a bare [w, h] with the
// background alongside it - because both appear in the set and a reader should not
// have to normalize one by hand to reuse it.
type templateFile struct {
	Name         string               `json:"name"`
	Note         string               `json:"note"`
	Canvas       json.RawMessage      `json:"canvas"`
	Background   string               `json:"background"`
	Slots        []tools.TemplateSlot `json:"slots"`
	Values       map[string]string    `json:"values"`
	CanvasWidth  int                  `json:"canvas_width"`
	CanvasHeight int                  `json:"canvas_height"`
}

func (t *templateFile) canvas() (w, h int, bg string, err error) {
	bg = t.Background
	if len(t.Canvas) > 0 {
		// [1200, 628]
		var pair []int
		if json.Unmarshal(t.Canvas, &pair) == nil && len(pair) == 2 {
			return pair[0], pair[1], bg, nil
		}
		// {"width": 1200, "height": 628, "background": "color:..."}
		var obj struct {
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			Background string `json:"background"`
		}
		if json.Unmarshal(t.Canvas, &obj) == nil && obj.Width > 0 {
			if obj.Background != "" {
				bg = obj.Background
			}
			return obj.Width, obj.Height, bg, nil
		}
	}
	if t.CanvasWidth > 0 {
		return t.CanvasWidth, t.CanvasHeight, bg, nil
	}
	return 0, 0, "", fmt.Errorf("no canvas: expected {\"canvas\": [w, h]}, " +
		"{\"canvas\": {\"width\": w, \"height\": h}} or canvas_width/canvas_height")
}

// version is the release tag, injected by -ldflags "-X main.version=...".
var version = ""

// resolveVersion reports which build this is, the same way the server does: a release
// carries the tag in version, a `go install ...@vX.Y.Z` build gets no ldflags and falls
// back to what the toolchain records, and a working tree build says so.
func resolveVersion() string {
	if version != "" {
		return version
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "dev"
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cfg := tools.ConfigFromEnv()
	ctx := context.Background()

	switch os.Args[1] {
	case "compile", "render":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: imagor-template compile <template.json>")
			os.Exit(2)
		}
		if err := compile(ctx, cfg, os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "imagor-template: %v\n", err)
			os.Exit(1)
		}
	case "decompile", "parse":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "usage: imagor-template decompile <path>")
			os.Exit(2)
		}
		if err := decompile(ctx, cfg, os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "imagor-template: %v\n", err)
			os.Exit(1)
		}
	case "version", "-version", "--version":
		fmt.Println("imagor-template", resolveVersion())
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "imagor-template: unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

func compile(ctx context.Context, cfg tools.Config, path string) error {
	if _, err := cfg.Mode(); err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var tf templateFile
	if err := json.Unmarshal(raw, &tf); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	w, h, bg, err := tf.canvas()
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	_, out, err := cfg.RenderTemplateTool(ctx, nil, tools.TemplateInput{
		CanvasWidth:  w,
		CanvasHeight: h,
		Background:   bg,
		Slots:        tf.Slots,
		Values:       tf.Values,
	})
	if err != nil {
		return err
	}
	// The path alone on stdout, so it pipes: `imagor-template compile x.json | pbcopy`.
	fmt.Println(out.Path)
	fmt.Fprintf(os.Stderr, "%s: %dx%d, %d slots, mode %s", tf.Name, w, h, len(tf.Slots), out.Mode)
	if out.Note != "" {
		fmt.Fprintf(os.Stderr, "\n  note: %s", out.Note)
	}
	fmt.Fprintln(os.Stderr)
	if os.Getenv("IMAGOR_BASE_URL") != "" && out.AbsoluteURL != "" {
		fmt.Fprintln(os.Stderr, out.AbsoluteURL)
	}
	return nil
}

func decompile(ctx context.Context, cfg tools.Config, path string) error {
	_, out, err := cfg.ParseTemplateTool(ctx, nil, tools.ParseTemplateInput{Path: strings.TrimSpace(path)})
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return err
	}
	// The lossiness is reported on stderr, where it cannot corrupt a piped template.
	if out.Note != "" {
		fmt.Fprintf(os.Stderr, "note: %s\n", out.Note)
	}
	return nil
}
