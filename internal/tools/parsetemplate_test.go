package tools

import (
	"context"
	"strings"
	"testing"
)

func parse(t *testing.T, c Config, path string) ParseTemplateOutput {
	t.Helper()
	_, out, err := c.ParseTemplateTool(context.Background(), nil, ParseTemplateInput{Path: path})
	if err != nil {
		t.Fatalf("parse %q: %v", path, err)
	}
	return out
}

func render(t *testing.T, c Config, in TemplateInput) TemplateOutput {
	t.Helper()
	_, out, err := c.RenderTemplateTool(context.Background(), nil, in)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return out
}

// The two directions are meant to be used together, so the property that matters
// is that a path survives the round trip unchanged. If it does not, a template
// read back from a path renders differently from the one that produced it, which
// is the worst possible failure for an editing loop.
func TestParseTemplateRoundTripsAPath(t *testing.T) {
	srv := stubImagor(t, 2)
	defer srv.Close()
	c := Config{Unsafe: true, BaseURL: srv.URL}

	cases := []struct {
		name string
		in   TemplateInput
	}{
		{"text only", TemplateInput{CanvasWidth: 1200, CanvasHeight: 630,
			Background: "color:1a1a2e", Slots: []TemplateSlot{
				{Name: "t", Kind: "text", Text: "Hello", X: "60", Y: "80", Font: "serif", Size: 48, Color: "ffffff"}}}},
		{"fitted text", TemplateInput{CanvasWidth: 1200, CanvasHeight: 630,
			Background: "color:1a1a2e", Slots: []TemplateSlot{
				{Name: "t", Kind: "text", Text: "A longer headline that wraps", X: "60", Y: "80",
					Width: "600", Height: "200", Font: "serif", Color: "ffffff", Align: "low"}}}},
		{"image and rect", TemplateInput{CanvasWidth: 800, CanvasHeight: 400,
			Background: "color:ffffff", Slots: []TemplateSlot{
				{Name: "r", Kind: "rect", Image: "color:3b82f6", X: "0", Y: "0", Width: "12", Height: "full"},
				{Name: "i", Kind: "image", Image: "http://example.com/a.png", X: "40", Y: "40", Width: "200", Height: "200"}}}},
		{"fitted image", TemplateInput{CanvasWidth: 800, CanvasHeight: 400,
			Background: "color:ffffff", Slots: []TemplateSlot{
				{Name: "i", Kind: "image", Image: "http://example.com/a.png", X: "0", Y: "0",
					Width: "400", Height: "full", Fit: true}}}},
		{"text with punctuation", TemplateInput{CanvasWidth: 1200, CanvasHeight: 630,
			Background: "color:1a1a2e", Slots: []TemplateSlot{
				{Name: "t", Kind: "text", Text: "Pipelines, bytes & the same result?", X: "60", Y: "80",
					Font: "serif", Size: 48, Color: "ffffff"}}}},
		{"a source that needs encoding", TemplateInput{CanvasWidth: 800, CanvasHeight: 400,
			Background: "color:ffffff", Slots: []TemplateSlot{
				{Name: "i", Kind: "image", Image: "http://example.com/a,b.png?v=2", X: "10", Y: "10",
					Width: "100", Height: "100"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			first := render(t, c, tc.in)
			back := parse(t, c, first.Path)
			again := render(t, c, TemplateInput{
				CanvasWidth: back.CanvasWidth, CanvasHeight: back.CanvasHeight,
				Background: back.Background, Slots: back.Slots, Values: back.Values,
			})
			if first.Path != again.Path {
				t.Errorf("the path did not survive the round trip\n  first: %s\n  again: %s", first.Path, again.Path)
			}
		})
	}
}

// A path is what a person is most likely to paste, but a URL is what they are
// most likely to have. Both have to work.
func TestParseTemplateAcceptsAURL(t *testing.T) {
	c := Config{Unsafe: true}
	path := "1200x630/filters:text(b64:SGVsbG8,60,80,serif-48,ffffff)/color:1a1a2e"
	for _, in := range []string{
		path,
		"/" + path,
		"http://localhost:8123/" + path,
		"http://localhost:8123/AbCdEf0123456789=/" + path,
		"https://cdn.example.com/AbCdEf0123456789=/" + path,
	} {
		got := parse(t, c, in)
		if got.CanvasWidth != 1200 || got.CanvasHeight != 630 {
			t.Errorf("%q gave canvas %dx%d, want 1200x630", in, got.CanvasWidth, got.CanvasHeight)
		}
		if len(got.Slots) != 1 || got.Slots[0].Text != "Hello" {
			t.Errorf("%q did not read the slot back: %+v", in, got.Slots)
		}
	}
}

// The text argument is base64 so that punctuation survives. Reading it back has
// to undo both that and the markup escaping.
func TestParseTemplateDecodesText(t *testing.T) {
	c := Config{Unsafe: true}
	for _, text := range []string{
		"Hello",
		"Pipelines, bytes & the same result?",
		"a < b > c",
		"日本語のタイトル",
		"emoji 🎉 and spaces",
	} {
		out := render(t, c, TemplateInput{CanvasWidth: 800, CanvasHeight: 400,
			Background: "color:ffffff", Slots: []TemplateSlot{
				{Name: "t", Kind: "text", Text: text, X: "10", Y: "10", Font: "sans", Size: 24, Color: "000000"}}})
		back := parse(t, c, out.Path)
		if len(back.Slots) != 1 {
			t.Fatalf("%q: read %d slots", text, len(back.Slots))
		}
		if back.Slots[0].Text != text {
			t.Errorf("text did not survive: %q -> %q", text, back.Slots[0].Text)
		}
	}
}

// A slot's parts have to come back individually, because the point is to edit
// them rather than re-emit the same path.
func TestParseTemplateRecoversEachPart(t *testing.T) {
	srv := stubImagor(t, 2)
	defer srv.Close()
	c := Config{Unsafe: true, BaseURL: srv.URL}
	out := render(t, c, TemplateInput{CanvasWidth: 1200, CanvasHeight: 630, Background: "color:1a1a2e",
		Slots: []TemplateSlot{
			{Name: "hero", Kind: "image", Image: "http://example.com/a.png", X: "0", Y: "0",
				Width: "400", Height: "full", Fit: true},
			{Name: "title", Kind: "text", Text: "A headline", X: "440", Y: "80",
				Width: "600", Height: "200", Font: "serif", Color: "ffffff", Align: "low"},
		}})
	back := parse(t, c, out.Path)
	if len(back.Slots) != 2 {
		t.Fatalf("read %d slots, want 2", len(back.Slots))
	}
	img, txt := back.Slots[0], back.Slots[1]
	// the canvas is 1200x630, so full has already become 630 by the time a path
	// exists - that is what "the geometry comes back absolute" means
	if img.Kind != "image" || !img.Fit || img.Image != "http://example.com/a.png" ||
		img.X != "0" || img.Y != "0" || img.Width != "400" || img.Height != "630" {
		t.Errorf("the image slot came back wrong: %+v", img)
	}
	// and the fitted size is recorded in the path, so it comes back as a number
	// rather than as a request to fit
	if txt.Kind != "text" || txt.Text != "A headline" || txt.X != "440" || txt.Y != "80" ||
		txt.Font != "serif" || txt.Size != 96 || txt.Width != "600" || txt.Color != "ffffff" {
		t.Errorf("the text slot came back wrong: %+v", txt)
	}
}

// The canvas and the background are part of the template, not decoration.
func TestParseTemplateRecoversTheCanvas(t *testing.T) {
	c := Config{Unsafe: true}
	out := render(t, c, TemplateInput{CanvasWidth: 1080, CanvasHeight: 1080, Background: "color:0f172a",
		Slots: []TemplateSlot{{Name: "t", Kind: "text", Text: "x", X: "10", Y: "10", Font: "sans", Size: 20, Color: "ffffff"}}})
	back := parse(t, c, out.Path)
	if back.CanvasWidth != 1080 || back.CanvasHeight != 1080 {
		t.Errorf("canvas came back %dx%d", back.CanvasWidth, back.CanvasHeight)
	}
	if back.Background != "color:0f172a" {
		t.Errorf("background came back %q", back.Background)
	}
}

// A path that has nothing to read should say so, rather than return an empty
// template that silently renders nothing.
func TestParseTemplateRefusesWhatItCannotRead(t *testing.T) {
	c := Config{Unsafe: true}
	for _, in := range []string{
		"",
		"   ",
		"1200x630/color:ffffff",      // no filters at all
		"1200x630/filters:fill(fff)", // a filter, but not one that makes a slot
		"1200x630/filters:quality(80)",
	} {
		if _, _, err := c.ParseTemplateTool(context.Background(), nil, ParseTemplateInput{Path: in}); err == nil {
			t.Errorf("%q should have been refused", in)
		}
	}
}

// The note has to be honest about the one thing that does not survive: a path
// holds the numbers a template resolved to, not the forms it was written in.
func TestParseTemplateSaysTheGeometryIsAbsolute(t *testing.T) {
	c := Config{Unsafe: true}
	out := render(t, c, TemplateInput{CanvasWidth: 1800, CanvasHeight: 942, Background: "color:ffffff",
		Slots: []TemplateSlot{
			{Name: "hero", Kind: "image", Image: "http://example.com/a.png", X: "0", Y: "0",
				Width: "38p", Height: "full"},
		}})
	back := parse(t, c, out.Path)
	if !strings.Contains(back.Note, "absolute") {
		t.Errorf("the note does not mention that the geometry is absolute: %q", back.Note)
	}
	// and the percentage really has become a number, which is the thing the note
	// is warning about
	if back.Slots[0].Width == "38p" {
		t.Error("the percentage should not survive a path; it is resolved before it gets there")
	}
	if back.Slots[0].Width != "684" {
		t.Errorf("38p of 1800 came back as %q, want 684", back.Slots[0].Width)
	}
}

// The legacy spellings are still filters and still have to be readable.
func TestParseTemplateReadsLegacyLabel(t *testing.T) {
	c := Config{Unsafe: true}
	back := parse(t, c, "800x400/filters:label(b64:TG9nbw,40,10,44,ffffff)/color:1a1a2e")
	if len(back.Slots) != 1 {
		t.Fatalf("read %d slots", len(back.Slots))
	}
	s := back.Slots[0]
	if s.Kind != "text" || s.Text != "Logo" || s.X != "40" || s.Y != "10" || s.Size != 44 || s.Color != "ffffff" {
		t.Errorf("label came back wrong: %+v", s)
	}
}

// The parts, on their own.
func TestDecodeTextAndSplitHelpers(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"plain", "plain"},
		{"b64:SGVsbG8", "Hello"},
		{"b64:YSAmYW1wOyBi", "a & b"},                    // Pango escaping undone
		{"b64:Jmx0O3gmZ3Q7", "<x>"},                      // the escaping is undone, not kept
		{"b64:not!valid!base64", "b64:not!valid!base64"}, // left alone rather than mangled
	} {
		if got := decodeText(tc.in); got != tc.want {
			t.Errorf("decodeText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	for _, tc := range []struct {
		in   string
		font string
		size int
	}{
		{"serif-48", "serif", 48},
		{"sans-bold-22", "sans-bold", 22},
		{"serif", "serif", 0},
		{"Palatino-63", "Palatino", 63},
	} {
		f, s := splitFont(tc.in)
		if f != tc.font || s != tc.size {
			t.Errorf("splitFont(%q) = %q,%d want %q,%d", tc.in, f, s, tc.font, tc.size)
		}
	}
	for _, tc := range []struct{ in, w, h string }{
		{"400x300", "400", "300"},
		{"fullxfull", "full", "full"},
		{"38pxfull", "38p", "full"},
	} {
		w, h, ok := splitDims(tc.in)
		if !ok || w != tc.w || h != tc.h {
			t.Errorf("splitDims(%q) = %q,%q,%v want %q,%q,true", tc.in, w, h, ok, tc.w, tc.h)
		}
	}
}
