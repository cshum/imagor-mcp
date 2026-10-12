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

// A nested canvas holds more than text. A coloured panel or a photograph inside one
// is the other half of what a canvas is for, and the inner type had no image field,
// so a rect inside a canvas was refused as an unknown property rather than built.
func TestNestedCanvasHoldsARectAndAnImage(t *testing.T) {
	c := Config{Unsafe: true}
	in := TemplateInput{
		CanvasWidth: 400, CanvasHeight: 200, Background: "color:ffffff",
		Slots: []TemplateSlot{{
			Name: "panel", Kind: "image", X: "50", Y: "50", Width: "300", Height: "100",
			Slots: []InnerSlot{
				{Name: "bg", Kind: "rect", Image: "color:e11d48", X: "0", Y: "0", Width: "300", Height: "100"},
				{Name: "t", Kind: "text", Text: "HELLO", Y: "40", Font: "sans-bold", Size: 28},
			},
		}},
	}
	_, out, err := c.RenderTemplateTool(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// The rect is an image() inside the nested path, sized to the canvas, with its
	// colour as the source.
	if !strings.Contains(out.Path, "image(300x100/color:e11d48,0,0)") {
		t.Fatalf("the inner rect is not in the nested path:\n%s", out.Path)
	}
	if !strings.Contains(out.Path, ":text(HELLO,") {
		t.Fatalf("the inner text is not in the nested path:\n%s", out.Path)
	}
	// A rect with no source is skipped rather than emitted empty.
	in.Slots[0].Slots[0].Image = ""
	if _, out2, err := c.RenderTemplateTool(context.Background(), nil, in); err != nil {
		t.Fatalf("render with an empty rect: %v", err)
	} else if strings.Contains(out2.Path, "color:,0,0") {
		t.Fatalf("an empty rect was emitted:\n%s", out2.Path)
	}
}

// A nested canvas may hold an image that fits itself and a caption below it. The two
// fields that made this impossible were Fit, absent from InnerSlot, and an x that the
// image branch resolved as a dimension rather than a position - so a nested picture
// could not be centred and could not keep its aspect ratio.
func TestNestedCanvasHoldsAFittedImageAndACaption(t *testing.T) {
	cfg := Config{Unsafe: true}
	in := TemplateInput{
		CanvasWidth:  600,
		CanvasHeight: 400,
		Slots: []TemplateSlot{{
			Name: "cell", Kind: "image", X: "6p", Y: "8p", Width: "30p", Height: "80p",
			Image: "https://s.imagor.net/product-bag.svg",
			Slots: []InnerSlot{
				{Name: "photo", Kind: "image", X: "center", Y: "0p",
					Width: "full", Height: "70p",
					Image: "https://s.imagor.net/product-bag.svg", Fit: true},
				{Name: "caption", Kind: "text", X: "center", Y: "76p",
					Width: "full", Height: "18p",
					Text: "Leather satchel bag", Font: "Open-Sans-16", Color: "000000"},
			},
		}},
	}
	_, out, err := cfg.RenderTemplateTool(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out.Path, "fit-in/") {
		t.Errorf("the nested image should fit itself, path was %s", out.Path)
	}
	if !strings.Contains(out.Path, "fill(none)") {
		t.Errorf("a fitted nested image should carry fill(none), path was %s", out.Path)
	}
	if !strings.Contains(out.Path, ",center,0)") {
		t.Errorf("the nested image should be centred, path was %s", out.Path)
	}
}
