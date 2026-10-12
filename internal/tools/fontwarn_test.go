package tools

import (
	"strings"
	"testing"

	"github.com/cshum/imagor/imagorpath"
)

// Pango reads a font description family-first and size-last, so a size written
// anywhere else is discarded. imagor answers 200 either way - the only evidence is
// the type rendering at the wrong size - so the server says it instead.
func TestFontSizeNotLastIsWarned(t *testing.T) {
	text := func(font string) imagorpath.Filter {
		return imagorpath.Filter{Name: "text", Args: "Hello,20,30," + font + ",000000"}
	}
	cases := []struct {
		font string
		warn bool
	}{
		{"sans", false},
		{"sans-40", false},
		{"sans-bold", false},
		{"sans-bold-40", false}, // size last, read
		{"sans-40-bold", true},  // size in the middle, dropped
		{"Open-Sans-60-bold", true},
		{"Liberation-Sans-40", false}, // a hyphenated family, no size
		{"bold-sans-40", false},       // a word before the family: Pango's problem, not a size
	}
	for _, c := range cases {
		got := fontWarnings([]imagorpath.Filter{text(c.font)})
		if c.warn && len(got) == 0 {
			t.Errorf("%q: expected a warning, got none", c.font)
		}
		if !c.warn && len(got) > 0 {
			t.Errorf("%q: unexpected warning %q", c.font, got[0])
		}
	}
	// The warning has to name the font and say what to do about it.
	got := fontWarnings([]imagorpath.Filter{text("sans-40-bold")})
	if len(got) != 1 || !strings.Contains(got[0], "sans-40-bold") || !strings.Contains(got[0], "last") {
		t.Fatalf("the warning does not name the font and the fix: %q", got)
	}
	// And it must suggest moving the size, not adding a second one.
	if !strings.Contains(got[0], "Write sans-bold-40 instead") {
		t.Fatalf("the suggestion should move the size to the end: %q", got)
	}
}
