package tools

import (
	"context"
	"strings"
	"testing"
)

// A slot carrying its own slots is a canvas, and that is the only way text gets
// centred on something narrower than the banner: a numeric x leaves it hanging
// left whatever align says, and x=center resolves against whatever canvas it is
// on. If nesting breaks, every label in the cookbook silently goes back to
// hanging left, which is not an error anyone would see.
func TestNestedCanvasCentresItsText(t *testing.T) {
	c := Config{Unsafe: true}
	in := TemplateInput{
		CanvasWidth:  1800,
		CanvasHeight: 300,
		Background:   "color:ffffff",
		Slots: []TemplateSlot{{
			Name: "label", Kind: "image",
			X: "772", Y: "100", Width: "300", Height: "52",
			Slots: []InnerSlot{{
				Name: "t", Kind: "text", Text: "Padded leather shoulder bag",
				Y: "14", Font: "sans", Color: "9a9a9a", Size: 17,
			}},
		}},
	}
	_, out, err := c.RenderTemplateTool(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// The canvas is the label box and the text is centred in it, so the inner
	// path has to carry both: a nested size and a center x.
	if !strings.Contains(out.Path, "image(300x52/filters:text(") {
		t.Fatalf("the nested canvas is not 300x52 in the path:\n%s", out.Path)
	}
	if !strings.Contains(out.Path, ",center,14,") {
		t.Fatalf("the inner text is not centred at x=center:\n%s", out.Path)
	}
	// The outer position is the box, not the canvas centre: a label under a
	// product sits where the product is.
	if !strings.Contains(out.Path, "/color:none,772,100)") {
		t.Fatalf("the nested canvas is not placed at 772,100:\n%s", out.Path)
	}
	if len(out.Resolved) != 1 || !strings.Contains(out.Resolved[0].Detail, "canvas 300x52 at 772,100") {
		t.Fatalf("the resolved report does not describe the canvas: %+v", out.Resolved)
	}
}

// A nested canvas is transparent, so it adds only what it draws and does not
// paint over the layout beneath it.
func TestNestedCanvasIsTransparent(t *testing.T) {
	c := Config{Unsafe: true}
	in := TemplateInput{
		CanvasWidth: 600, CanvasHeight: 200, Background: "color:ffffff",
		Slots: []TemplateSlot{{
			Name: "label", Kind: "image", X: "10", Y: "10", Width: "300", Height: "52",
			Slots: []InnerSlot{{Name: "t", Kind: "text", Text: "hi", Y: "10", Font: "sans", Size: 20}},
		}},
	}
	_, out, err := c.RenderTemplateTool(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out.Path, "/color:none,") {
		t.Fatalf("the nested canvas is not transparent:\n%s", out.Path)
	}
}

// A nested canvas with no size cannot be built, and saying so beats rendering a
// canvas of zero that silently drops the label.
func TestNestedCanvasNeedsASize(t *testing.T) {
	c := Config{Unsafe: true}
	in := TemplateInput{
		CanvasWidth: 600, CanvasHeight: 200, Background: "color:ffffff",
		Slots: []TemplateSlot{{
			Name: "label", Kind: "image", X: "10", Y: "10",
			Slots: []InnerSlot{{Name: "t", Kind: "text", Text: "hi", Font: "sans", Size: 20}},
		}},
	}
	if _, _, err := c.RenderTemplateTool(context.Background(), nil, in); err == nil {
		t.Fatal("a nested canvas without a width and height should be refused")
	}
}
