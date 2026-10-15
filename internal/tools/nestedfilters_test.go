package tools

import (
	"context"
	"strings"
	"testing"
)

// A slot's own filters belong in a filters: namespace inside the nested path. Emitted without the
// prefix they were read as the source, so filters: "upscale()" asked DNS for a host called
// upscale() and the rendered URL 404'd - while render_template still returned it as success.
func TestSlotFiltersCarryTheFiltersPrefix(t *testing.T) {
	c := Config{Unsafe: true}
	_, out, err := c.RenderTemplateTool(context.Background(), nil, TemplateInput{
		CanvasWidth:  1200,
		CanvasHeight: 630,
		Background:   "color:ffffff",
		Slots: []TemplateSlot{{
			Name: "photo", Kind: "image",
			Image: "https://example.com/a.png",
			X:     "0", Y: "0", Width: "400", Height: "full",
			Filters: "upscale()",
		}},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out.Path, "filters:upscale()") {
		t.Errorf("the nested path does not carry filters:upscale(): %s", out.Path)
	}
	if strings.Contains(out.Path, "400xfull/upscale()") {
		t.Errorf("the nested path still puts the filter where the source goes: %s", out.Path)
	}
}

// A filter's Source may itself be an imagor path, the same way the top-level image can. Encoded as
// a URL it became https://200x200/... and imagor was asked to resolve "200x200" as a host.
func TestFilterSourceThatIsAPathIsNotTreatedAsAURL(t *testing.T) {
	c := Config{Unsafe: true}
	_, out, err := c.BuildURLTool(context.Background(), nil, TransformInput{
		Image:  "https://example.com/a.png",
		Width:  400,
		Height: 300,
		Filters: []FilterInput{{
			Name: "image", Source: "200x200/https://example.com/b.png", Args: "0,0",
		}},
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !strings.Contains(out.Path, "path:200x200/") {
		t.Errorf("a path-typed Source was not wrapped as a nested path: %s", out.Path)
	}
	if strings.Contains(out.Path, "https://200x200/") {
		t.Errorf("a path-typed Source was still encoded as a URL: %s", out.Path)
	}
}
