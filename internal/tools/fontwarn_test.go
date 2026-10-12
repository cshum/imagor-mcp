package tools

import (
	"strings"
	"testing"
)

// Pango reads a font description family-first and size-last, so a size written
// anywhere else is discarded. imagor answers 200 either way - the only evidence is
// the type rendering at the wrong size - so the server says it.
func TestFontSizeNotLastIsWarned(t *testing.T) {
	cases := []struct {
		font string
		warn string // the corrected form, or "" for no warning
	}{
		{"sans", ""},
		{"sans-40", ""},
		{"sans-bold", ""},
		{"sans-bold-40", ""},             // size last, read
		{"sans-40-bold", "sans-bold-40"}, // size in the middle, dropped
		{"Open-Sans-60-bold", "Open-Sans-bold-60"},
		{"Liberation-Sans-40", ""}, // a hyphenated family, no size
		{"bold-sans-40", ""},       // a word before the family is not a size
	}
	for _, c := range cases {
		got := fontWarning(c.font)
		if c.warn == "" {
			if got != "" {
				t.Errorf("%q: unexpected warning %q", c.font, got)
			}
			continue
		}
		if got == "" {
			t.Errorf("%q: expected a warning, got none", c.font)
			continue
		}
		if !strings.Contains(got, c.font) || !strings.Contains(got, "Write "+c.warn+" instead") {
			t.Errorf("%q: warning should name the font and the corrected form: %q", c.font, got)
		}
	}
}

// sans, serif and monospace resolve to DejaVu, the widest face imagor carries, so a
// slot asking for one gets the narrower face it meant and a note saying so. DejaVu
// stays reachable by name.
func TestGenericFontsResolveToTheNarrowerFace(t *testing.T) {
	cases := []struct{ in, want string }{
		{"sans", "Open-Sans"},
		{"sans-bold", "Open-Sans-bold"},
		{"sans-bold-24", "Open-Sans-bold-24"},
		{"serif", "Liberation-Serif"},
		{"serif-72", "Liberation-Serif-72"},
		{"monospace", "Liberation-Mono"},
		{"Open-Sans-bold-24", "Open-Sans-bold-24"}, // already a face
		{"DejaVu-Sans-40", "DejaVu-Sans-40"},       // asked for by name, kept
		{"Liberation-Serif-24", "Liberation-Serif-24"},
	}
	for _, c := range cases {
		got, note := resolveFont(c.in)
		if got != c.want {
			t.Errorf("resolveFont(%q) = %q, want %q", c.in, got, c.want)
		}
		swapped := got != c.in
		if swapped && note == "" {
			t.Errorf("resolveFont(%q) swapped without saying so", c.in)
		}
		if !swapped && note != "" {
			t.Errorf("resolveFont(%q) left it alone but returned a note: %q", c.in, note)
		}
	}
}
