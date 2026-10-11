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
