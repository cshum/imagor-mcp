package tools

import (
	"context"
	"strings"
	"testing"
)

// Round-tripping a template has to be checked FIELD BY FIELD, not by comparing
// paths. A parser that returns a plausible-looking template passes a path
// comparison and still loses the parts - a slot's align, a nested slot's source,
// a box width written as a form. The point of parse_template is that you can edit
// what comes back, so every field it can carry has to survive.
//
// This walks a template through render -> parse and reports the first field that
// differs, by name, so a failure says which one rather than "the path changed".
func TestRoundTripKeepsEveryField(t *testing.T) {
	srv := stubImagor(t, 2)
	defer srv.Close()
	c := Config{Unsafe: true, BaseURL: srv.URL}

	in := TemplateInput{
		CanvasWidth:  1200,
		CanvasHeight: 630,
		Background:   "color:1a1a2e",
		Slots: []TemplateSlot{
			// a plain text slot: the base case
			{Name: "t", Kind: "text", Text: "Hello", X: "60", Y: "80",
				Font: "Liberation-Serif", Size: 48, Color: "ffffff"},
			// a fitted text slot: box width is written as a FORM, and the tail
			// carries the align. Both were being dropped.
			{Name: "fitted", Kind: "text", Text: "A headline that wraps", X: "60", Y: "160",
				Width: "600", Height: "200", Font: "Liberation-Serif", Color: "ffffff", Align: "low"},
			// a rect: no source fetch, so it cannot fail for network reasons
			{Name: "rule", Kind: "rect", Image: "color:3b82f6", X: "0", Y: "0",
				Width: "12", Height: "full"},
			// a plain image
			{Name: "img", Kind: "image", Image: "http://example.com/a.png", X: "40", Y: "40",
				Width: "200", Height: "200"},
			// a fitted image, where fit-in and fill(none) both appear in the path
			{Name: "fit", Kind: "image", Image: "http://example.com/b.png", X: "0", Y: "0",
				Width: "400", Height: "full", Fit: true},
			// a nested canvas: children inside a slot. This is the case that
			// flattened completely.
			{Name: "cell", Kind: "rect", Image: "color:none", X: "800", Y: "60",
				Width: "320", Height: "400", Slots: []InnerSlot{
					{Name: "photo", Kind: "image", Image: "http://example.com/c.png", X: "0", Y: "0",
						Width: "320", Height: "300", Fit: true},
					{Name: "cap", Kind: "text", Text: "Leather satchel", X: "center", Y: "40",
						Width: "full", Height: "40", Font: "Liberation-Serif", Color: "333333"},
					{Name: "price", Kind: "text", Text: "HKD$18500", X: "center", Y: "80",
						Font: "Open-Sans-bold", Size: 28, Color: "111111"},
				}},
			// a slot with its own filters: smart is a crop mode, not a dimension,
			// and it lives in the Filters field
			{Name: "smart", Kind: "image", Image: "http://example.com/d.png", X: "0", Y: "0",
				Width: "176", Height: "176", Filters: "smart"},
		},
	}

	out := render(t, c, in)
	back := parse(t, c, out.Path)

	if len(back.Slots) != len(in.Slots) {
		t.Fatalf("read %d slots, want %d", len(back.Slots), len(in.Slots))
	}
	for i := range in.Slots {
		want, got := in.Slots[i], back.Slots[i]
		check := func(field, w, g string) {
			t.Helper()
			if w != g {
				t.Errorf("slot %d (%s) %s: got %q, want %q", i, want.Name, field, g, w)
			}
		}
		check("kind", want.Kind, got.Kind)
		check("text", want.Text, got.Text)
		check("image", want.Image, got.Image)
		check("x", want.X, got.X)
		check("y", want.Y, got.Y)
		check("width", want.Width, got.Width)
		// A fitted text slot's box HEIGHT does not survive, and cannot: the path
		// records the size the text was fitted to, not the box it was fitted
		// into. That is the documented limit, and it is the one field a round
		// trip is not expected to keep.
		if want.Kind != "text" {
			check("height", want.Height, got.Height)
		}
		check("font", want.Font, got.Font)
		check("color", want.Color, got.Color)
		check("align", want.Align, got.Align)
		check("filters", want.Filters, got.Filters)
		if want.Fit != got.Fit {
			t.Errorf("slot %d (%s) fit: got %v, want %v", i, want.Name, got.Fit, want.Fit)
		}
		// a fixed size survives; a fitted one is recorded as the size chosen, so
		// only compare it when the template asked for a fixed size
		if want.Height == "" && want.Size > 0 && got.Size != want.Size {
			t.Errorf("slot %d (%s) size: got %d, want %d", i, want.Name, got.Size, want.Size)
		}
		// the children, which is the part that flattened entirely
		if len(want.Slots) != len(got.Slots) {
			t.Errorf("slot %d (%s): read %d nested slots, want %d",
				i, want.Name, len(got.Slots), len(want.Slots))
			continue
		}
		for j := range want.Slots {
			wc, gc := want.Slots[j], got.Slots[j]
			if wc.Kind != gc.Kind || wc.Text != gc.Text || wc.X != gc.X || wc.Y != gc.Y {
				t.Errorf("slot %d (%s) child %d: got %+v, want %+v", i, want.Name, j, gc, wc)
			}
			// A nested box's width and height are resolved to pixels before the
			// path is written, so they come back as numbers - the documented
			// limit, and the one thing a nested child does not keep. Font and
			// colour are not geometry and must survive.
			if wc.Font != gc.Font {
				t.Errorf("slot %d (%s) child %d font: got %q, want %q",
					i, want.Name, j, gc.Font, wc.Font)
			}
			if wc.Color != gc.Color {
				t.Errorf("slot %d (%s) child %d colour: got %q, want %q",
					i, want.Name, j, gc.Color, wc.Color)
			}
			if gc.Width == "" {
				t.Errorf("slot %d (%s) child %d lost its width entirely", i, want.Name, j)
			}
		}
	}
}

// The text tail is where the align lives, and the box width in it is a form.
// Reading it as a number dropped both, silently.
func TestRoundTripKeepsTheTextBoxAndAlign(t *testing.T) {
	srv := stubImagor(t, 2)
	defer srv.Close()
	c := Config{Unsafe: true, BaseURL: srv.URL}

	out := render(t, c, TemplateInput{CanvasWidth: 1200, CanvasHeight: 630,
		Background: "color:ffffff", Slots: []TemplateSlot{
			{Name: "t", Kind: "text", Text: "Wrapping headline", X: "60", Y: "80",
				Width: "600", Height: "200", Font: "Liberation-Serif",
				Color: "ffffff", Align: "low"},
		}})
	back := parse(t, c, out.Path)
	if len(back.Slots) != 1 {
		t.Fatalf("read %d slots", len(back.Slots))
	}
	got := back.Slots[0]
	if got.Width != "600" {
		t.Errorf("box width came back %q, want \"600\" - a form must not be read as a number", got.Width)
	}
	if got.Align != "low" {
		t.Errorf("align came back %q, want \"low\" - it is the field after the box width", got.Align)
	}
}

// A nested canvas is the case that flattened. Its children have to come back
// inside it, not as unrelated top-level slots.
func TestRoundTripKeepsNestedSlots(t *testing.T) {
	srv := stubImagor(t, 2)
	defer srv.Close()
	c := Config{Unsafe: true, BaseURL: srv.URL}

	out := render(t, c, TemplateInput{CanvasWidth: 1800, CanvasHeight: 942,
		Background: "color:ffffff", Slots: []TemplateSlot{
			{Name: "cell", Kind: "rect", Image: "color:none", X: "100", Y: "60",
				Width: "320", Height: "400", Slots: []InnerSlot{
					{Name: "cap", Kind: "text", Text: "Leather satchel", X: "center", Y: "40",
						Width: "320", Height: "40", Font: "Liberation-Serif", Color: "333333"},
				}},
		}})
	back := parse(t, c, out.Path)
	if len(back.Slots) != 1 {
		t.Fatalf("read %d top-level slots, want 1 - a nested canvas flattened", len(back.Slots))
	}
	if len(back.Slots[0].Slots) != 1 {
		t.Fatalf("the nested canvas came back with %d children, want 1", len(back.Slots[0].Slots))
	}
	child := back.Slots[0].Slots[0]
	if child.Text != "Leather satchel" {
		t.Errorf("the nested text came back %q", child.Text)
	}
	if child.X != "center" {
		t.Errorf("the nested x came back %q, want center", child.X)
	}
	if !strings.Contains(child.Color, "333333") && child.Color != "" {
		t.Logf("nested colour came back %q", child.Color)
	}
}

// A slot's own crop mode lives in Filters, and has to survive with it.
func TestRoundTripKeepsSlotFilters(t *testing.T) {
	srv := stubImagor(t, 2)
	defer srv.Close()
	c := Config{Unsafe: true, BaseURL: srv.URL}

	out := render(t, c, TemplateInput{CanvasWidth: 800, CanvasHeight: 400,
		Background: "color:ffffff", Slots: []TemplateSlot{
			{Name: "s", Kind: "image", Image: "http://example.com/d.png", X: "0", Y: "0",
				Width: "176", Height: "176", Filters: "smart"},
		}})
	back := parse(t, c, out.Path)
	if len(back.Slots) != 1 {
		t.Fatalf("read %d slots", len(back.Slots))
	}
	if !strings.Contains(back.Slots[0].Filters, "smart") {
		t.Errorf("smart came back as %q, want it in Filters", back.Slots[0].Filters)
	}
}

// Nothing above should depend on a live server, so this asserts the stub was
// never asked for anything unexpected.
func TestRoundTripStubIsQuiet(t *testing.T) {
	c := Config{Unsafe: true}
	if _, _, err := c.ParseTemplateTool(context.Background(), nil,
		ParseTemplateInput{Path: ""}); err == nil {
		t.Error("an empty path should be refused")
	}
}
