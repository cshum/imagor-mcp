package tools

import (
	"encoding/base64"
	"strings"
	"testing"
)

// A title travels in the path, where a comma separates arguments and a question
// mark starts a query string. Before this, a comma truncated the string and
// imagor rendered nothing while still returning 200, and a question mark changed
// the path a signature covered, so the request came back as a signature
// mismatch. Both are silent in the sense that matters: nothing says the title
// was the problem.
func TestTextArgsEncodesWhatWouldBreakThePath(t *testing.T) {
	for _, title := range []string{
		"Pipelines, bytes and the same result everywhere",
		"Images & bytes",
		"Why does the render differ?",
		"Why <canvas> beats a screenshot",
	} {
		got := textArgs(title, "60,240,serif-68,ffffff")
		if !strings.HasPrefix(got, "b64:") {
			t.Errorf("%q was not encoded: %s", title, got)
			continue
		}
		if strings.ContainsAny(got, "?&<>") {
			t.Errorf("%q left path syntax in the argument: %s", title, got)
		}
		// The encoded part must decode back to the title, with markup escaped,
		// since Pango reads the decoded string as markup.
		first := strings.Split(got, ",")[0]
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(first, "b64:"))
		if err != nil {
			t.Errorf("%q did not encode as base64url: %v", title, err)
			continue
		}
		want := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(title)
		if string(raw) != want {
			t.Errorf("%q decoded to %q, want %q", title, raw, want)
		}
		// The rest of the arguments have to survive untouched.
		if !strings.HasSuffix(got, ",60,240,serif-68,ffffff") {
			t.Errorf("%q lost its tail arguments: %s", title, got)
		}
	}
}

// A filter that is not text() must not be rewritten, and a text() with no text
// given separately keeps the arguments it was handed.
func TestTextArgsLeavesEverythingElseAlone(t *testing.T) {
	if got := textArgs("", "white"); got != "white" {
		t.Errorf("an empty text changed the arguments: %q", got)
	}
	if got := textArgs("", ""); got != "" {
		t.Errorf("an empty text produced %q, want nothing", got)
	}
	if got := textArgs("just the text", ""); !strings.HasPrefix(got, "b64:") {
		t.Errorf("text with no tail was not encoded: %q", got)
	}
}

// A text already encoded by the caller must not be encoded twice.
func TestTextArgsLeavesAPreEncodedArgument(t *testing.T) {
	// the caller passing b64: through args is the documented escape hatch, and
	// textArgs is only reached when the typed field is used, so this checks the
	// assembly path rather than the helper.
	got := textArgs("plain", "b64:YWJj,60,240")
	if !strings.HasPrefix(got, "b64:") || !strings.Contains(got, "b64:YWJj") {
		t.Errorf("the tail was not preserved: %s", got)
	}
}

// A nested source is the one argument that is a URL, and a URL can carry a ? or
// a , without meaning to. Measured on a real server: written raw those gave 403
// and 404; encoded they render.
func TestSourceArgsEncodesWhatWouldBreakThePath(t *testing.T) {
	for _, tc := range []struct {
		source, tail, want string
	}{
		{"", "600x600,60,60", "600x600,60,60"},
		{"http://host/plain.png", "60,60", "http://host/plain.png,60,60"},
		{"http://host/a.png?v=2", "60,60", "b64:aHR0cDovL2hvc3QvYS5wbmc_dj0y,60,60"},
		{"http://host/a,b.png", "", "b64:aHR0cDovL2hvc3QvYSxiLnBuZw"},
		{"http://host/a,b.png?v=2", "60,60", "b64:aHR0cDovL2hvc3QvYSxiLnBuZz92PTI,60,60"},
	} {
		got := sourceArgs(tc.source, tc.tail)
		if got != tc.want {
			t.Errorf("sourceArgs(%q, %q) = %q, want %q", tc.source, tc.tail, got, tc.want)
		}
	}
}
