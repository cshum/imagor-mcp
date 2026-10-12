package tools

import (
	"strings"
	"testing"
)

// The two cases use different rules on purpose, and the difference is the point:
// text is arbitrary input from someone's database, so anything not trivially safe
// is encoded; a source is nearly always a path, which legitimately contains '/'
// and ':' and '=', so a whitelist there would make every path opaque.
//
// These align with imagor-studio's converter, so a path either project builds
// means the same thing.

// Text: only characters that are unambiguously inert pass through.
func TestEncodeTextValueIsAWhitelist(t *testing.T) {
	for _, tc := range []struct {
		in      string
		encoded bool
	}{
		{"Hello", false},
		{"hello_world-2026", false},
		{"", false},
		// everything below has to be encoded
		{"hello world", true}, // a space ends the path
		{"a,b", true},         // a comma splits the arguments
		{"why?", true},        // a question mark starts the query string
		{"a & b", true},       // an ampersand needs markup escaping too
		{"a < b > c", true},   // so does markup
		{"café", true},        // anything outside ascii
		{"日本語", true},
		{"emoji 🎉", true},
		{"a/b", true}, // a slash would look like a nested path
		{"a:b", true},
		{"a#b", true},
		{"a(b)", true},
		{"a'b\"c", true},
	} {
		got := encodeTextValue(tc.in)
		if tc.encoded && !strings.HasPrefix(got, "b64:") {
			t.Errorf("encodeTextValue(%q) = %q, should have been encoded", tc.in, got)
		}
		if !tc.encoded && got != tc.in {
			t.Errorf("encodeTextValue(%q) = %q, should have been left alone", tc.in, got)
		}
		if tc.encoded {
			if back := decodeTextValue(got); back != tc.in {
				t.Errorf("%q did not survive: %q -> %q", tc.in, got, back)
			}
		}
	}
}

// A source: only the characters that change how the path is read, plus the words
// imagor would read as a command.
func TestEncodeSourceValueCoversWhatBreaksAPath(t *testing.T) {
	for _, tc := range []struct {
		in      string
		encoded bool
	}{
		// left alone, because a real source looks like this
		{"http://example.com/a.png", false},
		{"color:ff0000", false},
		{"https://cdn.example.com/path/to/photo.jpg", false},
		{"photos/a-b_c.png", false},
		// encoded, because these change how the path is read
		{"a b.png", true},  // a space
		{"a?b.png", true},  // a query string
		{"a#b.png", true},  // a fragment
		{"a&b.png", true},  // query parameters
		{"a,b.png", true},  // the argument separator
		{"a(b).png", true}, // filter syntax
		// encoded even though they hold no special character, because imagor
		// would read the first word as a command
		{"trim/a.png", true},
		{"meta/a.png", true},
		{"fit-in/a.png", true},
		{"stretch/a.png", true},
		{"top/a.png", true},
		{"left/a.png", true},
		{"right/a.png", true},
		{"bottom/a.png", true},
		{"center/a.png", true},
		{"smart/a.png", true},
		// and these are not reserved, because the prefix does not match whole
		{"trimming/a.png", false},
		{"metadata/a.png", false},
		{"smartphone/a.png", false},
	} {
		got := encodeSourceValue(tc.in)
		if tc.encoded && !strings.HasPrefix(got, "b64:") {
			t.Errorf("encodeSourceValue(%q) = %q, should have been encoded", tc.in, got)
		}
		if !tc.encoded && got != tc.in {
			t.Errorf("encodeSourceValue(%q) = %q, should have been left alone", tc.in, got)
		}
		if tc.encoded {
			if back := decodeTextValue(got); back != tc.in {
				t.Errorf("%q did not survive: %q -> %q", tc.in, got, back)
			}
		}
	}
}

// Every one of those characters has to survive the whole way through a filter
// argument, not just the helper.
func TestAboutArgumentsSurviveTheAssembly(t *testing.T) {
	for _, text := range []string{
		"Hello", "hello world", "a,b", "why?", "a & b", "a < b > c",
		"café", "日本語", "emoji 🎉", "a/b", "a:b",
	} {
		got := textArgs(text, "60,240,serif-68,ffffff")
		if !strings.HasSuffix(got, ",60,240,serif-68,ffffff") {
			t.Errorf("%q lost its tail: %s", text, got)
		}
		head := strings.TrimSuffix(got, ",60,240,serif-68,ffffff")
		if !strings.ContainsAny(head, ",") && decodeTextValue(head) != text {
			t.Errorf("%q did not round trip through the assembly: %s", text, got)
		}
		if strings.Count(head, ",") > 0 {
			t.Errorf("%q put an unencoded comma in front of the tail: %s", text, got)
		}
	}
	// a source carrying any of the breaking characters must not split the args:
	// the tail has to arrive intact, whatever the source contained
	for _, src := range []string{"a b.png", "a,b.png", "a?b.png", "a&b.png", "a(b).png", "a#b.png"} {
		got := sourceArgs(src, "100,100,0,0")
		if !strings.HasSuffix(got, ",100,100,0,0") {
			t.Errorf("sourceArgs(%q) = %q, the tail was split", src, got)
		}
		if strings.HasPrefix(got, "b64:") && strings.ContainsAny(strings.SplitN(strings.TrimPrefix(got, "b64:"), ",", 2)[0], ",?&#() ") {
			t.Errorf("sourceArgs(%q) = %q, an encoded source still holds a breaking character", src, got)
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
	// simple text is deliberately left readable rather than encoded
	if got := textArgs("plain", ""); got != "plain" {
		t.Errorf("simple text should pass through: %q", got)
	}
	// and anything else is encoded
	if got := textArgs("just the text", ""); !strings.HasPrefix(got, "b64:") {
		t.Errorf("text with a space was not encoded: %q", got)
	}
}

// A text already encoded by the caller must not be encoded twice.
func TestTextArgsLeavesAPreEncodedArgument(t *testing.T) {
	got := textArgs("plain", "b64:YWJj,60,240")
	// "plain" is simple text and is left alone; the caller's own b64 argument is
	// not touched either, which is the documented escape hatch
	if got != "plain,b64:YWJj,60,240" {
		t.Errorf("textArgs(\"plain\", \"b64:YWJj,60,240\") = %q, want it unchanged", got)
	}
}

// The characters that break a path, measured on a real server rather than assumed.
func TestASourceArgumentKeepsItsCharacters(t *testing.T) {
	for _, tc := range []struct{ source, tail, want string }{
		{"", "600x600,60,60", "600x600,60,60"},
		{"http://host/plain.png", "60,60", "http://host/plain.png,60,60"},
		{"http://host/a.png?v=2", "60,60", "b64:aHR0cDovL2hvc3QvYS5wbmc_dj0y,60,60"},
		{"http://host/a,b.png", "", "b64:aHR0cDovL2hvc3QvYSxiLnBuZw"},
		{"http://host/a b.png", "60,60", "b64:aHR0cDovL2hvc3QvYSBiLnBuZw,60,60"},
	} {
		if got := sourceArgs(tc.source, tc.tail); got != tc.want {
			t.Errorf("sourceArgs(%q, %q) = %q, want %q", tc.source, tc.tail, got, tc.want)
		}
	}
}

// Every word imagor reads as a command has to be in the reserved list. A source
// that begins with one is otherwise left unencoded and read as that command, and
// the failure is silent: the path resolves to a different image rather than
// erroring. The list is derived from imagor's own parser, so it grows whenever
// imagor grows a prefix.
func TestReservedPrefixesCoverImagorsCommands(t *testing.T) {
	// The prefixes imagorpath.Parse recognises before the source, taken from
	// imagorpath/parse.go.
	commands := []string{
		"trim", "meta", "fit-in", "full-fit-in", "adaptive-fit-in",
		"stretch", "top", "left", "right", "bottom", "center", "smart",
	}
	have := map[string]bool{}
	for _, p := range reservedSourcePrefixes {
		have[strings.TrimSuffix(p, "/")] = true
	}
	for _, c := range commands {
		if !have[c] {
			t.Errorf("%q is a command but is not a reserved prefix, so a source "+
				"beginning with it would be read as one", c+"/")
		}
	}
}

// A filter that is listed but has no readable signature invites a guess. label()
// is undocumented and text() does everything it does, so it is not offered - but
// it still parses, so a path that already uses it keeps working.
func TestSupersededFiltersAreNotOffered(t *testing.T) {
	if !superseded("label") {
		t.Error("label should not be offered: text covers it and it has no signature")
	}
	for _, keep := range []string{"text", "modulate", "trim", "padding", "image"} {
		if superseded(keep) {
			t.Errorf("%s should stay offered", keep)
		}
	}
}
