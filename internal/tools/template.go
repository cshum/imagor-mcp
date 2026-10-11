package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cshum/imagor-mcp/internal/pathgen"
	"github.com/cshum/imagor/imagorpath"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TemplateSlot is one element of a template.
//
// Geometry is written the way imagor writes it, so `full`, `f`, `full-N` and
// `f-N` all work: a slot can be positioned and sized relative to the canvas and
// the same template then serves several canvas sizes. A text slot with both a
// width and a height is fitted, which is what keeps a long title from
// overflowing the frame it was written for.
type TemplateSlot struct {
	Name    string `json:"name" jsonschema:"a label for this slot, used in the resolved output"`
	Kind    string `json:"kind" jsonschema:"text, image or rect"`
	Text    string `json:"text,omitempty" jsonschema:"for text: the string, which may contain {{values}}"`
	Image   string `json:"image,omitempty" jsonschema:"for image or rect: a source URL, or color:RRGGBB for a rect"`
	X       string `json:"x" jsonschema:"horizontal position: pixels, or full-N / f-N relative to the canvas"`
	Y       string `json:"y" jsonschema:"vertical position: pixels, or full-N / f-N relative to the canvas"`
	Width   string `json:"width,omitempty" jsonschema:"for text: the box width, which sets the wrap and the fitting; pixels or full-N. For image: the overlay width"`
	Height  string `json:"height,omitempty" jsonschema:"for text: the box height, which is what the text is fitted into. For image: the overlay height"`
	Font    string `json:"font,omitempty" jsonschema:"for text: font family without a size, for example serif; the size is chosen by fitting when a height is given"`
	Size    int    `json:"size,omitempty" jsonschema:"for text: a fixed size, used only when no height is given so the slot is not fitted"`
	Color   string `json:"color,omitempty" jsonschema:"text colour, or overlay colour for an image slot"`
	Align   string `json:"align,omitempty" jsonschema:"text alignment within the box: low, center or high"`
	MaxSize int    `json:"max_size,omitempty" jsonschema:"for fitted text: the largest size to try"`
	MinSize int    `json:"min_size,omitempty" jsonschema:"for fitted text: the smallest size to accept"`
}

// TemplateInput is a layout plus the values to fill it with.
type TemplateInput struct {
	CanvasWidth  int               `json:"canvas_width" jsonschema:"canvas width in pixels"`
	CanvasHeight int               `json:"canvas_height" jsonschema:"canvas height in pixels"`
	Background   string            `json:"background,omitempty" jsonschema:"canvas background, for example color:1a1a2e; defaults to color:white"`
	Slots        []TemplateSlot    `json:"slots" jsonschema:"the elements, applied in order, so later slots draw over earlier ones"`
	Values       map[string]string `json:"values,omitempty" jsonschema:"values for {{placeholders}} in slot text and image fields"`
}

// ResolvedSlot reports what a slot became, so a caller can see the choices the
// template made rather than only the URL.
type ResolvedSlot struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// TemplateOutput is the rendered path, and what each slot resolved to.
type TemplateOutput struct {
	Path        string         `json:"path"`
	URL         string         `json:"url,omitempty"`
	AbsoluteURL string         `json:"absolute_url,omitempty"`
	Signed      bool           `json:"signed"`
	Mode        string         `json:"mode"`
	Resolved    []ResolvedSlot `json:"resolved"`
	Note        string         `json:"note,omitempty"`
}

// RenderTemplateTool renders a template at a given canvas size.
//
// The point is that a layout is written once and rendered at any size: geometry
// is relative to the canvas, so the same template serves 1200x630 and 1080x1080.
// Type is the exception - a font size is absolute - so a text slot with a height
// is fitted against the box that geometry resolves to, per canvas.
func (c Config) RenderTemplateTool(ctx context.Context, _ *mcp.CallToolRequest, in TemplateInput) (*mcp.CallToolResult, TemplateOutput, error) {
	if in.CanvasWidth <= 0 || in.CanvasHeight <= 0 {
		return nil, TemplateOutput{}, fmt.Errorf("a template needs canvas_width and canvas_height in pixels")
	}
	if len(in.Slots) == 0 {
		return nil, TemplateOutput{}, fmt.Errorf("a template needs at least one slot")
	}
	background := in.Background
	if background == "" {
		background = "color:white"
	}

	filters := make(imagorpath.Filters, 0, len(in.Slots))
	resolved := make([]ResolvedSlot, 0, len(in.Slots))
	var missing []string

	for _, slot := range in.Slots {
		text := substitute(slot.Text, in.Values)
		source := substitute(slot.Image, in.Values)
		for _, n := range append(unfilled(slot.Text, in.Values), unfilled(slot.Image, in.Values)...) {
			missing = append(missing, fmt.Sprintf("%s.%s", slot.Name, n))
		}
		switch slot.Kind {
		case "text":
			if strings.TrimSpace(text) == "" {
				resolved = append(resolved, ResolvedSlot{Name: slot.Name, Detail: "skipped: no text"})
				continue
			}
			boxW, err := c.resolve(slot.Width, in.CanvasWidth, 0)
			if err != nil {
				return nil, TemplateOutput{}, fmt.Errorf("slot %q width: %w", slot.Name, err)
			}
			boxH, err := c.resolve(slot.Height, in.CanvasHeight, 0)
			if err != nil {
				return nil, TemplateOutput{}, fmt.Errorf("slot %q height: %w", slot.Name, err)
			}
			font := slot.Font
			if font == "" {
				font = "sans"
			}
			// A font argument carries one size. A slot may name one in its font
			// ("sans-bold-22") or ask for one with size, but the size that is
			// chosen here is the one that goes in - otherwise the argument ends
			// up with two, and "sans-28-48" is not a font.
			font = stripFontSize(font)
			size, lines, detail := slot.Size, 0, ""
			if boxH > 0 {
				// Fitting needs a real box, so the box is resolved per canvas.
				// This is the step that keeps type legible as the frame grows.
				fitted, err := c.fitForTemplate(ctx, text, boxW, boxH, font, slot)
				if err != nil {
					return nil, TemplateOutput{}, fmt.Errorf("slot %q: %w", slot.Name, err)
				}
				size, lines = fitted.Size, fitted.Lines
				detail = fmt.Sprintf(" (fitted, %d line(s))", lines)
			}
			if size <= 0 {
				size = 48
			}
			fontArg := fmt.Sprintf("%s-%d", font, size)
			// The tail of the argument, in imagor's order: colour, alpha,
			// blend mode, wrap width, align, justify, wrap mode, spacing.
			sx, err := c.position(slot.X, in.CanvasWidth)
			if err != nil {
				return nil, TemplateOutput{}, fmt.Errorf("slot %q x: %w", slot.Name, err)
			}
			sy, err := c.position(slot.Y, in.CanvasHeight)
			if err != nil {
				return nil, TemplateOutput{}, fmt.Errorf("slot %q y: %w", slot.Name, err)
			}
			args := fmt.Sprintf("%s,%s,%s,%s", sx, sy, fontArg, orDefault(slot.Color, "000000"))
			if boxW > 0 {
				align := orDefault(slot.Align, "low")
				args += fmt.Sprintf(",0,normal,%d,%s,normal,word", boxW, align)
			}
			filters = append(filters, imagorpath.Filter{Name: "text", Args: textArgs(text, args)})
			resolved = append(resolved, ResolvedSlot{
				Name:   slot.Name,
				Detail: fmt.Sprintf("text at %s,%s %s, box %dx%d%s", sx, sy, fontArg, boxW, boxH, detail),
			})
		case "image", "rect":
			if source == "" {
				resolved = append(resolved, ResolvedSlot{Name: slot.Name, Detail: "skipped: no source"})
				continue
			}
			inner := source
			if w := strings.TrimSpace(slot.Width); w != "" {
				h := orDefault(slot.Height, w)
				inner = w + "x" + h + "/" + source
			}
			sx, err := c.position(slot.X, in.CanvasWidth)
			if err != nil {
				return nil, TemplateOutput{}, fmt.Errorf("slot %q x: %w", slot.Name, err)
			}
			sy, err := c.position(slot.Y, in.CanvasHeight)
			if err != nil {
				return nil, TemplateOutput{}, fmt.Errorf("slot %q y: %w", slot.Name, err)
			}
			filters = append(filters, imagorpath.Filter{
				Name: "image",
				Args: fmt.Sprintf("%s,%s,%s", inner, sx, sy),
			})
			resolved = append(resolved, ResolvedSlot{
				Name:   slot.Name,
				Detail: fmt.Sprintf("%s at %s,%s sized %sx%s", slot.Kind, sx, sy, orDefault(slot.Width, "auto"), orDefault(slot.Height, "auto")),
			})
		default:
			return nil, TemplateOutput{}, fmt.Errorf("slot %q has kind %q; use text, image or rect", slot.Name, slot.Kind)
		}
	}

	p := imagorpath.Params{
		Image:   background,
		Width:   in.CanvasWidth,
		Height:  in.CanvasHeight,
		Filters: filters,
	}
	mode, err := c.Mode()
	if err != nil {
		return nil, TemplateOutput{}, err
	}
	out := TemplateOutput{Mode: string(mode), Path: pathgen.Build(p), Resolved: resolved}
	switch mode {
	case ModeSigned:
		signer, err := c.Signer()
		if err != nil {
			return nil, TemplateOutput{}, err
		}
		out.URL = pathgen.BuildURL(p, signer)
		out.Signed = true
	case ModeUnsafe:
		out.URL = pathgen.BuildURL(p, nil)
	}
	if c.BaseURL != "" {
		out.AbsoluteURL = c.BaseURL + "/" + out.URL
	}
	notes := compositionWarnings(p)
	if len(missing) > 0 {
		notes = append(notes, fmt.Sprintf(
			"these placeholders had no value: %s. An image slot left unfilled is a nested source "+
				"that cannot be fetched, and the render will fail.", strings.Join(missing, ", ")))
	}
	if len(notes) > 0 {
		out.Note = strings.Join(notes, "; ")
	}
	return nil, out, nil
}

// fitForTemplate runs the same search fit_text does, for a slot's box.
func (c Config) fitForTemplate(ctx context.Context, text string, boxW, boxH int, font string, slot TemplateSlot) (FitTextOutput, error) {
	in := FitTextInput{
		Text: text, BoxWidth: boxW, BoxHeight: boxH, Font: font,
		MaxSize: orDefaultInt(slot.MaxSize, 96), MinSize: orDefaultInt(slot.MinSize, 16),
		Color: slot.Color,
	}
	_, out, err := c.FitTextTool(ctx, nil, in)
	return out, err
}

// resolve turns a geometry value into pixels. Anything without a `full` prefix
// is already pixels; `full`, `f`, `full-N` and `f-N` are measured against the
// canvas dimension, which is what makes a template portable between sizes.
func (c Config) resolve(v string, dimension, fallback int) (int, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback, nil
	}
	if _, _, isFraction := splitFraction(v); !isFraction {
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("%q is not pixels and not full, full-N, half, third or quarter", v)
		}
		return n, nil
	}
	// Fractions come first, because a layout that wants half the canvas should
	// not have to know how wide the canvas is. imagor has no fraction form, so
	// this is the layer that can offer one.
	base, rest, ok := splitFraction(v)
	if !ok {
		return 0, fmt.Errorf("%q is not a form this understands; use pixels, full, full-N, half, third or quarter, each optionally minus pixels", v)
	}
	value := dimension
	switch base {
	case "", "full", "f":
	case "half", "h":
		value = dimension / 2
	case "third", "t":
		value = dimension / 3
	case "quarter", "q":
		value = dimension / 4
	default:
		return 0, fmt.Errorf("%q is not a fraction this knows; use half, third or quarter", v)
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return value, nil
	}
	if !strings.HasPrefix(rest, "-") && !strings.HasPrefix(rest, "+") {
		return 0, fmt.Errorf("%q is not a form this understands; an offset is written -N or +N", v)
	}
	n, err := strconv.Atoi(rest[1:])
	if err != nil {
		return 0, fmt.Errorf("%q has an offset that is not a number", v)
	}
	if rest[0] == '-' {
		if n > value {
			return 0, fmt.Errorf("%q takes %d off %d, which leaves nothing", v, n, value)
		}
		return value - n, nil
	}
	return value + n, nil
}

// splitFraction separates a fraction or full prefix from a trailing -N offset.
func splitFraction(v string) (base, rest string, ok bool) {
	for _, name := range []string{"full", "f", "half", "h", "third", "t", "quarter", "q"} {
		if v == name {
			return name, "", true
		}
		if strings.HasPrefix(v, name+"-") || strings.HasPrefix(v, name+"+") {
			return name, v[len(name):], true
		}
	}
	return "", "", false
}

// position resolves a slot's x or y into something imagor will place correctly.
//
// imagor's position grammar has no full or full-N: a relative position written
// that way is read as zero and the element lands at the origin, silently. So a
// full-based position is turned into pixels here. Everything else - pixels,
// center, top, bottom, right, left and their offset forms - is imagor's own and
// is passed through untouched.
func (c Config) position(v string, dimension int) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "0", nil
	}
	if _, _, isFraction := splitFraction(v); !isFraction {
		return v, nil
	}
	n, err := c.resolve(v, dimension, 0)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(n), nil
}

// stripFontSize removes a trailing -NN from a font name, so a fitted slot does
// not end up with two sizes in one argument.
func stripFontSize(font string) string {
	if i := strings.LastIndex(font, "-"); i > 0 {
		if _, err := strconv.Atoi(font[i+1:]); err == nil {
			return font[:i]
		}
	}
	return font
}

// unfilled reports placeholders that no value supplied. An image slot left as
// {{cover}} is a nested source that cannot be fetched, so the render 404s - worth
// saying before it does.
func unfilled(text string, values map[string]string) []string {
	var out []string
	for {
		i := strings.Index(text, "{{")
		if i < 0 {
			break
		}
		j := strings.Index(text[i:], "}}")
		if j < 0 {
			break
		}
		name := text[i+2 : i+j]
		if _, ok := values[name]; !ok {
			out = append(out, name)
		}
		text = text[i+j+2:]
	}
	return out
}

// substitute fills {{name}} placeholders from values.
func substitute(s string, values map[string]string) string {
	if s == "" || len(values) == 0 {
		return s
	}
	for k, v := range values {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func orDefaultInt(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}
