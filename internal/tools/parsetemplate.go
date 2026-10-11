package tools

import (
	"context"
	"encoding/base64"
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
		"read from a path, so the geometry is absolute: %dx%d. A path records the numbers a template "+
			"resolved to and not the forms it was written in, so a relative value such as 38p or full-80 "+
			"has already become a number here. It renders identically and edits cleanly, but it will not "+
			"adapt to another canvas size.", p.Width, p.Height)
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
		// the inner path is [fit-in/]WxH/source
		rest := inner
		if strings.HasPrefix(rest, "fit-in/") {
			slot.Fit = true
			rest = strings.TrimPrefix(rest, "fit-in/")
		}
		if i := strings.Index(rest, "/"); i > 0 {
			dims := rest[:i]
			// a source carrying ? or , was encoded on the way in, and has to be
			// decoded on the way back or the round trip is not stable
			slot.Image = decodeText(rest[i+1:])
			if w, h, ok := splitDims(dims); ok {
				slot.Width, slot.Height = w, h
			}
		} else {
			slot.Image = decodeText(rest)
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
			Text: decodeText(args[0]),
			X:    args[1], Y: args[2],
		}
		slot.Font, slot.Size = splitFont(args[3])
		if len(args) > 4 {
			slot.Color = args[4]
		}
		// colour, alpha, blend, wrap width, align, justify, wrap mode, spacing
		if len(args) > 7 {
			if w, err := strconv.Atoi(args[7]); err == nil {
				slot.Width = strconv.Itoa(w)
			}
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
			Text: decodeText(args[0]),
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

// decodeText undoes what textArgs did: a b64: prefix and Pango escaping.
func decodeText(v string) string {
	if !strings.HasPrefix(v, "b64:") {
		return v
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(v, "b64:"))
	if err != nil {
		return v
	}
	return strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">").Replace(string(raw))
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
