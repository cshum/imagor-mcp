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

// InnerSlot is a slot laid out inside another slot's canvas.
//
// It carries the fields a label needs and no Slots of its own: the schema
// generator rejects a type that contains itself, and a canvas inside a canvas is
// not something a layout has needed. The canvas it sits on is what makes `center`
// mean something, since a position inside a nested canvas resolves against that
// canvas rather than the banner.
type InnerSlot struct {
	Name    string `json:"name" jsonschema:"a label for this slot, used in the resolved output"`
	Kind    string `json:"kind,omitempty" jsonschema:"text or rect; defaults to text"`
	Text    string `json:"text,omitempty" jsonschema:"the string, which may contain {{values}}"`
	X       string `json:"x,omitempty" jsonschema:"horizontal position within this canvas; defaults to center, which is the only value that centres text"`
	Y       string `json:"y,omitempty" jsonschema:"vertical position within this canvas"`
	Width   string `json:"width,omitempty" jsonschema:"the box width; defaults to the canvas width"`
	Height  string `json:"height,omitempty" jsonschema:"the box height; defaults to the canvas height"`
	Font    string `json:"font,omitempty" jsonschema:"font family without a size, for example sans-bold; the size is fitted to the box unless size is given"`
	Size    int    `json:"size,omitempty" jsonschema:"a fixed size, used instead of fitting"`
	Color   string `json:"color,omitempty" jsonschema:"text colour as RRGGBB"`
	Align   string `json:"align,omitempty" jsonschema:"low, center or high; defaults to center"`
	MaxSize int    `json:"max_size,omitempty" jsonschema:"for fitted text: the largest size to try"`
	MinSize int    `json:"min_size,omitempty" jsonschema:"for fitted text: the smallest size to accept"`
}

// TemplateSlot is one element of a template.
//
// A slot carrying its own Slots is a canvas, which is how a label is centred
// under the image it describes: positions inside a nested canvas are relative to
// it, so `center` resolves there rather than against the whole banner.
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
	Fit     bool   `json:"fit,omitempty" jsonschema:"for image: fit the whole image inside the box rather than filling it, which keeps the aspect ratio instead of stretching"`
	MaxSize int    `json:"max_size,omitempty" jsonschema:"for fitted text: the largest size to try"`
	MinSize int    `json:"min_size,omitempty" jsonschema:"for fitted text: the smallest size to accept"`

	// A slot that carries its own slots is a canvas: its width and height become
	// the canvas the inner slots are laid out on, and the result is drawn at this
	// slot's x and y. Positions inside are relative to that canvas, so `center`
	// centres a label under the image it describes rather than on the banner.
	Slots []InnerSlot `json:"slots,omitempty" jsonschema:"for a nested canvas: the slots to lay out inside this slot's width x height, positioned relative to it"`
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
	// The background is a field like any other, so it takes values too - a palette
	// that comes from a value should not work everywhere except the canvas.
	background := substitute(in.Background, in.Values)
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
		// A nested canvas draws text too, so its placeholders are checked here as
		// well - a {{value}} inside a canvas that was never filled renders as the
		// literal braces, which looks like a layout fault rather than a missing value.
		for _, inner := range slot.Slots {
			for _, n := range unfilled(inner.Text, in.Values) {
				missing = append(missing, fmt.Sprintf("%s.%s.%s", slot.Name, inner.Name, n))
			}
		}
		if len(slot.Slots) > 0 {
			// A slot carrying its own slots is a canvas: the inner slots are laid
			// out on a composition of this slot's size, and the result is drawn
			// here. Positions inside are relative to that canvas, so `center`
			// centres a label under the image it describes rather than on the
			// banner - the one way to centre text, since a numeric x always
			// leaves it hanging left.
			inner, innerResolved, err := c.nestedCanvas(ctx, slot, in)
			if err != nil {
				return nil, TemplateOutput{}, err
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
				Detail: fmt.Sprintf("canvas %sx%s at %s,%s: %s", slot.Width, orDefault(slot.Height, slot.Width), sx, sy, innerResolved),
			})
			continue
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
			boxWs, err := c.dimensionValue(slot.Width, in.CanvasWidth)
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
			//
			// The order of preference is size, then the font's own, then the
			// default. Taking the font's own matters: a slot that writes
			// "sans-bold-11" and nothing else means 11, and reading that as the
			// default 48 renders it four times too large.
			fontFamily, fontSize := splitFontName(font)
			font = fontFamily
			size, lines, detail := slot.Size, 0, ""
			if size <= 0 {
				size = fontSize
			}
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
				args += fmt.Sprintf(",0,normal,%s,%s,normal,word", boxWs, align)
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
			// a source is data, not syntax: a ? ends the path a signature covers
			// and a , splits the argument list. Same treatment as the filter
			// layer, for the same reason.
			source = encodeSourceValue(source)
			inner := source
			if w := strings.TrimSpace(slot.Width); w != "" {
				h := orDefault(slot.Height, w)
				// The nested path is imagor's own dimension segment, which has
				// no fractions. A slot written with half or third has to be
				// turned into pixels here, or the render 404s.
				iw, err := c.dimensionValue(w, in.CanvasWidth)
				if err != nil {
					return nil, TemplateOutput{}, fmt.Errorf("slot %q width: %w", slot.Name, err)
				}
				ih, err := c.dimensionValue(h, in.CanvasHeight)
				if err != nil {
					return nil, TemplateOutput{}, fmt.Errorf("slot %q height: %w", slot.Name, err)
				}
				inner = fmt.Sprintf("%sx%s/%s", iw, ih, source)
				if slot.Fit {
					// fit-in comes before the dimensions, and a nested path has
					// to end with its source
					inner = fmt.Sprintf("fit-in/%sx%s/%s", iw, ih, source)
				}
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
	for _, n := range unfilled(in.Background, in.Values) {
		missing = append(missing, "background."+n)
	}
	if len(missing) > 0 {
		// Refused rather than noted: an unfilled placeholder does not fail on its
		// own, it draws its own braces, so the render succeeds and the image is
		// wrong. That is the one outcome this tool exists to prevent.
		return nil, TemplateOutput{}, fmt.Errorf(
			"these placeholders had no value: %s", strings.Join(missing, ", "))
	}
	notes := compositionWarnings(p)
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
	// The base decides the value and any offset adjusts it, in one place, so
	// every form takes an offset the same way.
	base, rest := splitBase(v)
	switch {
	case base == "":
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf(
				"%q is not pixels, full, full-N, a fraction such as half, or a percentage such as 38p", v)
		}
		return n, nil
	case strings.HasSuffix(base, "p"):
		pct, err := strconv.Atoi(strings.TrimSuffix(base, "p"))
		if err != nil {
			return 0, fmt.Errorf("%q ends in p but is not a percentage", v)
		}
		return applyOffset(dimension*pct/100, dimension, rest, v)
	}
	value := dimension
	switch base {
	case "full", "f":
	case "half", "h":
		value = dimension / 2
	case "third", "t":
		value = dimension / 3
	case "quarter", "q":
		value = dimension / 4
	default:
		return 0, fmt.Errorf("%q is not a form this knows; use full, half, third, quarter or a percentage such as 38p", v)
	}
	return applyOffset(value, dimension, rest, v)
}

// dimensionValue resolves a slot's size, but leaves a form imagor can express.
//
// imagor's dimension segment has full and full-N, so there is no reason to throw
// that away and write a pixel: keeping it means a path still says what it meant,
// and parse_template can hand it back unchanged. A fraction or a percentage has
// no imagor equivalent, so those are resolved here.
func (c Config) dimensionValue(v string, dimension int) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", nil
	}
	if base, _ := splitBase(v); base == "full" || base == "f" {
		return v, nil
	}
	n, err := c.resolve(v, dimension, 0)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(n), nil
}

// splitBase separates a base form from a trailing -N or +N offset.
//
// imagor's dimension segment has full and full-N but no fraction and no
// percentage, so those are this layer's own and have to be turned into pixels
// before a path is built. A value this does not recognise comes back as no base,
// which leaves a plain number a plain number.
func splitBase(v string) (base, rest string) {
	for _, name := range []string{"full", "f", "half", "h", "third", "t", "quarter", "q"} {
		if v == name {
			return name, ""
		}
		if strings.HasPrefix(v, name+"-") || strings.HasPrefix(v, name+"+") {
			return name, v[len(name):]
		}
	}
	if i := strings.IndexAny(v, "+-"); i > 0 && strings.HasSuffix(v[:i], "p") {
		return v[:i], v[i:]
	}
	if strings.HasSuffix(v, "p") {
		return v, ""
	}
	return "", ""
}

// applyOffset applies a -N or +N suffix, refusing one that leaves nothing.
func applyOffset(value, dimension int, rest, original string) (int, error) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return value, nil
	}
	if !strings.HasPrefix(rest, "-") && !strings.HasPrefix(rest, "+") {
		return 0, fmt.Errorf("%q is not a form this understands; an offset is written -N or +N", original)
	}
	n, err := strconv.Atoi(rest[1:])
	if err != nil {
		// a percentage offset, e.g. 6p+34p - a fraction of the same dimension,
		// which is what a layout means when it says "a third down, and then on"
		if !strings.HasSuffix(rest[1:], "p") {
			return 0, fmt.Errorf("%q has an offset that is neither pixels nor a percentage", original)
		}
		pct, perr := strconv.Atoi(rest[1 : len(rest)-1])
		if perr != nil {
			return 0, fmt.Errorf("%q has an offset that is neither pixels nor a percentage", original)
		}
		n = dimension * pct / 100
	}
	if rest[0] == '-' {
		if n > value {
			return 0, fmt.Errorf("%q takes %d off %d, which leaves nothing", original, n, value)
		}
		return value - n, nil
	}
	return value + n, nil
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
	if base, _ := splitBase(v); base == "" {
		// pixels, center, top, bottom, right, left and their offsets are imagor's
		// own position forms and pass through untouched
		return v, nil
	}
	n, err := c.resolve(v, dimension, 0)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(n), nil
}

// splitFontName separates a font argument into its family and the size it names,
// if any. "sans-bold-22" is sans-bold at 22; "serif" is serif at no size.
func splitFontName(font string) (string, int) {
	if i := strings.LastIndex(font, "-"); i > 0 {
		if n, err := strconv.Atoi(font[i+1:]); err == nil {
			return font[:i], n
		}
	}
	return font, 0
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

// nestedCanvas lays a slot's inner slots out on a canvas of their own, and
// returns the path for that canvas plus a one-line report of what went in.
//
// The canvas is color:none - fully transparent - so it composites over whatever
// is beneath it and the slot only adds what it draws. The point of the canvas is
// position resolution: a numeric x leaves text hanging left of its box whatever
// align says, and only `center` centres it, so text that must read as centred
// needs a canvas narrow enough for `center` to mean something.
//
// Text is fitted against the canvas it sits on, so the size that comes out is the
// size that fits this canvas, not the banner.
func (c Config) nestedCanvas(ctx context.Context, slot TemplateSlot, in TemplateInput) (string, string, error) {
	w, err := c.resolve(slot.Width, in.CanvasWidth, 0)
	if err != nil {
		return "", "", fmt.Errorf("slot %q width: %w", slot.Name, err)
	}
	h, err := c.resolve(orDefault(slot.Height, slot.Width), in.CanvasHeight, 0)
	if err != nil {
		return "", "", fmt.Errorf("slot %q height: %w", slot.Name, err)
	}
	if w <= 0 || h <= 0 {
		return "", "", fmt.Errorf("slot %q: a nested canvas needs a width and a height, got %sx%s", slot.Name, slot.Width, slot.Height)
	}

	filters := make(imagorpath.Filters, 0, len(slot.Slots))
	var parts []string
	for _, inner := range slot.Slots {
		text := substitute(inner.Text, in.Values)
		if strings.TrimSpace(text) == "" {
			parts = append(parts, fmt.Sprintf("%s skipped: no text", inner.Name))
			continue
		}
		font := orDefault(inner.Font, "sans")
		fontFamily, fontSize := splitFontName(font)
		size := inner.Size
		if size <= 0 {
			size = fontSize
		}
		// The box is the whole canvas unless the inner slot narrows it: a nested
		// canvas is usually exactly the label, so the text fits the canvas.
		boxW, err := c.resolve(orDefault(inner.Width, strconv.Itoa(w)), w, 0)
		if err != nil {
			return "", "", fmt.Errorf("slot %q.%s width: %w", slot.Name, inner.Name, err)
		}
		boxH, err := c.resolve(orDefault(inner.Height, strconv.Itoa(h)), h, 0)
		if err != nil {
			return "", "", fmt.Errorf("slot %q.%s height: %w", slot.Name, inner.Name, err)
		}
		if size <= 0 && boxH > 0 {
			fitted, err := c.fitForTemplate(ctx, text, boxW, boxH, fontFamily, TemplateSlot{
				MaxSize: inner.MaxSize, MinSize: inner.MinSize, Font: inner.Font,
			})
			if err != nil {
				return "", "", fmt.Errorf("slot %q.%s: %w", slot.Name, inner.Name, err)
			}
			size = fitted.Size
		}
		if size <= 0 {
			size = 48
		}
		sx, err := c.position(orDefault(inner.X, "center"), w)
		if err != nil {
			return "", "", fmt.Errorf("slot %q.%s x: %w", slot.Name, inner.Name, err)
		}
		sy, err := c.position(inner.Y, h)
		if err != nil {
			return "", "", fmt.Errorf("slot %q.%s y: %w", slot.Name, inner.Name, err)
		}
		args := fmt.Sprintf("%s,%s,%s-%d,%s", sx, sy, fontFamily, size, orDefault(inner.Color, "000000"))
		if boxW > 0 {
			args += fmt.Sprintf(",0,normal,%d,%s,normal,word", boxW, orDefault(inner.Align, "center"))
		}
		filters = append(filters, imagorpath.Filter{Name: "text", Args: textArgs(text, args)})
		parts = append(parts, fmt.Sprintf("%s %s-%d at %s,%s", inner.Name, fontFamily, size, sx, sy))
	}

	path := pathgen.Build(imagorpath.Params{
		Width:   w,
		Height:  h,
		Filters: filters,
		Image:   "color:none",
	})
	return path, strings.Join(parts, "; "), nil
}
