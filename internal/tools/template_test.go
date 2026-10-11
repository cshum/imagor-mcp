package tools

import (
	"context"
	"strings"
	"testing"
)

// Geometry has to resolve against the canvas, because that is what makes one
// template serve several sizes. Anything without a full prefix is already pixels.
func TestResolveGeometryAgainstTheCanvas(t *testing.T) {
	c := Config{Unsafe: true}
	for _, tc := range []struct {
		in   string
		dim  int
		want int
		ok   bool
	}{
		{"40", 1200, 40, true},
		{"full", 1200, 1200, true},
		{"f", 1200, 1200, true},
		{"full-80", 1200, 1120, true},
		{"f-80", 1200, 1120, true},
		{"full-80", 630, 550, true},
		{"", 1200, 0, true},
		{"full-2000", 1200, 0, false}, // leaves nothing
		{"half", 1200, 0, false},      // not a form imagor has either
	} {
		got, err := c.resolve(tc.in, tc.dim, 0)
		if tc.ok && err != nil {
			t.Errorf("resolve(%q, %d) failed: %v", tc.in, tc.dim, err)
			continue
		}
		if !tc.ok {
			if err == nil {
				t.Errorf("resolve(%q, %d) should have been refused, got %d", tc.in, tc.dim, got)
			}
			continue
		}
		if got != tc.want {
			t.Errorf("resolve(%q, %d) = %d, want %d", tc.in, tc.dim, got, tc.want)
		}
	}
}

// A slot's text comes from data, so the placeholders have to be filled.
func TestSubstituteFillsPlaceholders(t *testing.T) {
	values := map[string]string{"title": "Pipelines, bytes & the same result", "author": "Adrian"}
	got := substitute("{{title}} by {{author}}", values)
	want := "Pipelines, bytes & the same result by Adrian"
	if got != want {
		t.Errorf("substitute = %q, want %q", got, want)
	}
	if got := substitute("no placeholders", values); got != "no placeholders" {
		t.Errorf("substitute changed a string with no placeholders: %q", got)
	}
	if got := substitute("{{title}}", nil); got != "{{title}}" {
		t.Errorf("substitute with no values changed the string: %q", got)
	}
}

// The whole reason a text slot carries a height is that the type is fitted
// against the box the geometry resolves to. Without that, the same template
// gives relatively smaller type as the canvas grows.
func TestTemplateFitsTypePerCanvas(t *testing.T) {
	srv := stubImagor(t, 2) // a size S draws 2S pixels of ink
	defer srv.Close()
	c := Config{Unsafe: true, BaseURL: srv.URL}

	slot := TemplateSlot{Name: "title", Kind: "text", Text: "a title",
		X: "40", Y: "140", Width: "full-80", Height: "220",
		Font: "serif", MaxSize: 84, MinSize: 24}

	_, small, err := c.RenderTemplateTool(context.Background(), nil, TemplateInput{
		CanvasWidth: 600, CanvasHeight: 400, Slots: []TemplateSlot{slot}})
	if err != nil {
		t.Fatalf("small canvas: %v", err)
	}
	_, large, err := c.RenderTemplateTool(context.Background(), nil, TemplateInput{
		CanvasWidth: 1600, CanvasHeight: 900, Slots: []TemplateSlot{slot}})
	if err != nil {
		t.Fatalf("large canvas: %v", err)
	}
	// The box width follows the canvas, but the box height is fixed, so the fit
	// should land on the same size for both. What must not happen is the size
	// ignoring the box entirely.
	if small.Path == "" || large.Path == "" {
		t.Fatal("a template rendered no path")
	}
	detail := func(o TemplateOutput) string {
		for _, r := range o.Resolved {
			if r.Name == "title" {
				return r.Detail
			}
		}
		return ""
	}
	for _, d := range []string{detail(small), detail(large)} {
		if !strings.Contains(d, "fitted") {
			t.Errorf("the text slot was not fitted: %q", d)
		}
		if !strings.Contains(d, "box") {
			t.Errorf("the resolved detail does not report the box: %q", d)
		}
	}
	// The wrap width is what follows the canvas.
	if !strings.Contains(detail(small), "520x220") {
		t.Errorf("small canvas resolved the wrong box: %q", detail(small))
	}
	if !strings.Contains(detail(large), "1520x220") {
		t.Errorf("large canvas resolved the wrong box: %q", detail(large))
	}
}

// A template has to refuse what it cannot render, rather than emit something
// that returns 200 with the wrong picture.
func TestTemplateRefusesWhatItCannotRender(t *testing.T) {
	c := Config{Unsafe: true, BaseURL: "http://example.invalid"}
	if _, _, err := c.RenderTemplateTool(context.Background(), nil, TemplateInput{Slots: []TemplateSlot{{Name: "x", Kind: "text"}}}); err == nil {
		t.Error("a template with no canvas size should be refused")
	}
	if _, _, err := c.RenderTemplateTool(context.Background(), nil, TemplateInput{CanvasWidth: 10, CanvasHeight: 10}); err == nil {
		t.Error("a template with no slots should be refused")
	}
	if _, _, err := c.RenderTemplateTool(context.Background(), nil, TemplateInput{
		CanvasWidth: 10, CanvasHeight: 10, Slots: []TemplateSlot{{Name: "x", Kind: "blur"}}}); err == nil {
		t.Error("an unknown slot kind should be refused")
	}
}

// Values fill the slots, and the encoded text argument keeps a title with
// punctuation intact - the same trap the build tools have.
func TestTemplateValuesSurvivePunctuation(t *testing.T) {
	c := Config{Unsafe: true}
	_, out, err := c.RenderTemplateTool(context.Background(), nil, TemplateInput{
		CanvasWidth: 1200, CanvasHeight: 630,
		Slots: []TemplateSlot{{Name: "title", Kind: "text", Text: "{{title}}",
			X: "40", Y: "140", Size: 48, Font: "serif"}},
		Values: map[string]string{"title": "Pipelines, bytes & the same result?"},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.ContainsAny(out.Path, "?") {
		t.Errorf("a question mark reached the path: %s", out.Path)
	}
	if !strings.Contains(out.Path, "b64:") {
		t.Errorf("the text was not encoded: %s", out.Path)
	}
}
