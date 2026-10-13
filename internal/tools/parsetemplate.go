package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cshum/imagor/imagorpath"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ParseTemplateInput is a path to read back into a template.
type ParseTemplateInput struct {
	Path string `json:"path" jsonschema:"an imagor path, or a full URL, to read back into a template. The path is what render_template produces, so a path you already have can be turned into something you can edit."`
}

// ParseTemplateOutput is the template a path decomposes into.
type ParseTemplateOutput struct {
	CanvasWidth  int               `json:"canvas_width"`
	CanvasHeight int               `json:"canvas_height"`
	Background   string            `json:"background"`
	Slots        []TemplateSlot    `json:"slots"`
	Values       map[string]string `json:"values,omitempty"`
	Note         string            `json:"note,omitempty"`
}

// ParseTemplateTool reads a path back into a template.
//
// The path is the output of the template layer's own resolution, so this is the
// inverse of render_template and the two are meant to be used together: take a
// path you already have - one a design produced, one a colleague sent, one this
// tool built last week - and get something you can edit rather than retype.
//
// One thing does not survive the round trip, and it is worth knowing before you
// rely on it. A path records the numbers a template resolved to, not the forms
// it was written in: 38p is already 684 by the time it reaches a path, and there
// is nothing in the path to say so. A template recovered from a path is therefore
// absolute. It renders identically and edits cleanly, but it will not adapt to
// another canvas size the way a hand-written one does.
func (c Config) ParseTemplateTool(_ context.Context, _ *mcp.CallToolRequest, in ParseTemplateInput) (*mcp.CallToolResult, ParseTemplateOutput, error) {
	path := strings.TrimSpace(in.Path)
	if path == "" {
		return nil, ParseTemplateOutput{}, fmt.Errorf("give me a path to read; this is the inverse of render_template")
	}
	// accept a full URL as readily as a path. Only strip a scheme if the string
	// starts with one - a nested source inside the path contains "://" too, and
	// looking for it anywhere threw away everything before it.
	if i := strings.Index(path, "://"); i >= 0 && i < 8 {
		rest := path[i+3:]
		if j := strings.Index(rest, "/"); j >= 0 {
			path = rest[j+1:]
		}
	}
	// a signed URL carries its signature as a leading segment
	if i := strings.Index(path, "/"); i > 0 && i < 64 {
		if head := path[:i]; strings.HasSuffix(head, "=") {
			path = path[i+1:]
		}
	}
	// Parse wants a leading slash, and a path is what it wants, not a URL
	p := imagorpath.Parse("/" + strings.TrimPrefix(path, "/"))
	out := ParseTemplateOutput{
		CanvasWidth:  p.Width,
		CanvasHeight: p.Height,
		Background:   p.Image,
		Slots:        []TemplateSlot{},
	}
	if out.Background == "" {
		out.Background = "color:white"
	}

	for _, f := range p.Filters {
		slot, ok := c.slotFromFilter(f)
		if !ok {
			continue
		}
		out.Slots = append(out.Slots, slot)
	}
	if len(out.Slots) == 0 {
		return nil, ParseTemplateOutput{}, fmt.Errorf("no slots found in that path; it has no image(), text() or rect filters to read")
	}
	out.Note = fmt.Sprintf(
		"read from a path, %dx%d. A form imagor can express is still a form here: full and full-80 "+
			"survive, because the dimension segment has them. A form it lacks has become a number - a "+
			"fraction such as half, a percentage such as 38p, and the size fitting chose for a text slot, "+
			"none of which a path can carry. Rewrite those as forms if the template has to adapt to "+
			"another canvas size.", p.Width, p.Height)
	return nil, out, nil
}

// slotFromFilter turns one filter into a slot, or reports that it is not one.
func (c Config) slotFromFilter(f imagorpath.Filter) (TemplateSlot, bool) {
	args := imagorpath.SplitArgs(f.Args)
	switch f.Name {
	case "image":
		if len(args) < 3 {
			return TemplateSlot{}, false
		}
		inner, x, y := args[0], args[1], args[2]
		slot := TemplateSlot{Name: "image", Kind: "image", X: x, Y: y}
		dims, middle, src := splitInnerPath(inner)
		slot.Fit = strings.HasPrefix(inner, "fit-in/")
		if w, h, ok := splitDims(dims); ok {
			slot.Width, slot.Height = w, h
		}
		slot.Filters = middle
		slot.Image = decodeTextValue(src)
		_ = middle
		// The inner filters that are themselves slots are the slot's children,
		// which is what stops a nested canvas flattening into unrelated top-level
		// slots. They are read from the middle, where they sit.
		if strings.Contains(middle, "filters:") || strings.Contains(middle, "text(") ||
			strings.Contains(middle, "image(") {
			parsed := imagorpath.Parse("/filters:" + strings.TrimPrefix(middle, "filters:"))
			for _, child := range parsed.Filters {
				if child.Name != "image" && child.Name != "text" && child.Name != "label" {
					continue
				}
				cs, ok := c.slotFromFilter(child)
				if !ok {
					continue
				}
				cs.Name = child.Name
				slot.Slots = append(slot.Slots, InnerSlot{
					Name: cs.Name, Kind: cs.Kind, Image: cs.Image, Text: cs.Text,
					X: cs.X, Y: cs.Y, Width: cs.Width, Height: cs.Height,
					Font: cs.Font, Size: cs.Size, Color: cs.Color,
					Align: cs.Align, Fit: cs.Fit, Filters: cs.Filters,
				})
				// The filter that became a child is the child now. Leaving it in
				// Filters as well would carry it twice: once nested, once as a
				// filter on the slot's own source.
				if k := strings.Index(slot.Filters, child.Name+"("); k >= 0 {
					if e := matchingParen(slot.Filters, k+len(child.Name)); e > k {
						slot.Filters = strings.Trim(slot.Filters[:k]+slot.Filters[e+1:], ":")
					}
				}
			}
		}
		if strings.HasPrefix(slot.Image, "color:") {
			slot.Kind = "rect"
		}
		return slot, true

	case "text":
		if len(args) < 4 {
			return TemplateSlot{}, false
		}
		slot := TemplateSlot{
			Name: "text", Kind: "text",
			Text: decodeTextValue(args[0]),
			X:    args[1], Y: args[2],
		}
		slot.Font, slot.Size = splitFont(args[3])
		if len(args) > 4 {
			slot.Color = args[4]
		}
		// colour, alpha, blend, wrap width, align, justify, wrap mode, spacing
		// The wrap width is a form, not a number: the emitter writes "full-144",
		// and reading it with Atoi dropped it and the align that follows it.
		if len(args) > 7 {
			slot.Width = args[7]
		}
		if len(args) > 8 {
			slot.Align = args[8]
		}
		return slot, true

	case "label":
		// the legacy form of text(), same shape
		if len(args) < 4 {
			return TemplateSlot{}, false
		}
		slot := TemplateSlot{
			Name: "label", Kind: "text",
			Text: decodeTextValue(args[0]),
			X:    args[1], Y: args[2],
		}
		if size, err := strconv.Atoi(args[3]); err == nil {
			slot.Size = size
		}
		if len(args) > 4 {
			slot.Color = args[4]
		}
		return slot, true
	}
	return TemplateSlot{}, false
}

// splitInnerPath takes an image slot's inner path apart into its three parts:
// the dimensions, whatever sits between them and the source, and the source.
//
// Neither imagor's parser nor a plain slash split can do this. imagor cannot read
// the dimension segment, because here it is the template layer's vocabulary:
// "full", "full-144" and "38p" are forms this layer resolves before a path is
// written, and imagor has never heard of them. A plain split on the last slash is
// worse, because a nested image() carries its own source and its own slashes.
//
// So the path is walked once, tracking parenthesis depth. The source is the last
// slash at depth zero: everything after it is the slot's own source, everything
// between the dimensions and it is the slot's own filters, and the filter groups
// in between are left intact for the caller to read.
func splitInnerPath(inner string) (dims, middle, src string) {
	rest := strings.TrimPrefix(inner, "fit-in/")
	i := strings.Index(rest, "/")
	if i < 0 {
		return rest, "", ""
	}
	dims, rest = rest[:i], rest[i+1:]

	// Walk once, tracking depth. A source is found either at a scheme at depth
	// zero - a URL carries slashes of its own, so the last slash is inside it -
	// or, failing that, at the last slash at depth zero.
	depth, last, scheme := 0, -1, -1
	for k := 0; k < len(rest); k++ {
		switch rest[k] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '/':
			if depth == 0 {
				last = k
			}
		}
		if depth != 0 || scheme >= 0 {
			continue
		}
		// a boundary: the start, or just after a slash
		if k != 0 && rest[k-1] != '/' {
			continue
		}
		for _, marker := range []string{"http://", "https://", "b64:", "color:", "file://"} {
			if strings.HasPrefix(rest[k:], marker) {
				scheme = k
				break
			}
		}
	}
	cut := last
	if scheme >= 0 {
		cut = scheme
	}
	if cut < 0 {
		return dims, "", rest
	}
	middle, src = rest[:cut], rest[cut:]
	middle = strings.TrimSuffix(middle, "/")
	middle = strings.TrimPrefix(middle, "filters:")
	// fill(none) is the template layer's own centring, not the slot's own filter
	middle = strings.TrimPrefix(middle, "fill(none):")
	middle = strings.TrimPrefix(middle, "fill(none)/")
	if middle == "none" || middle == "fill(none)" {
		middle = ""
	}
	return dims, middle, src
}

// matchingParen returns the index of the ")" that closes the "(" at open, or -1.
func matchingParen(s string, open int) int {
	if open < 0 || open >= len(s) || s[open] != '(' {
		return -1
	}
	depth := 0
	for k := open; k < len(s); k++ {
		switch s[k] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return k
			}
		}
	}
	return -1
}

// splitFont separates a font argument into a family and a size, if it carries one.
func splitFont(v string) (string, int) {
	if i := strings.LastIndex(v, "-"); i > 0 {
		if n, err := strconv.Atoi(v[i+1:]); err == nil {
			return v[:i], n
		}
	}
	return v, 0
}

// splitDims reads a WxH pair, either of which may be zero.
func splitDims(v string) (string, string, bool) {
	i := strings.Index(v, "x")
	if i <= 0 {
		return "", "", false
	}
	return v[:i], v[i+1:], true
}
