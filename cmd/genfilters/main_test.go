package main

import (
	"reflect"
	"testing"
)

// A filter whose argument points at a section carries that section's forms, so
// the reference can write a grammar several filters share exactly once.
func TestArgumentsFollowASectionLink(t *testing.T) {
	doc := []string{
		"### `image(imagorpath, x)`",
		"",
		"- `imagorpath` — the image to load",
		"- `x` — horizontal position: see [Horizontal position](#horizontal-position)",
		"",
		"### Horizontal position",
		"",
		"`x` takes one of these forms:",
		"",
		"- Positive number indicates position from the left, negative from the right",
		"- `left` or `l`, `right` or `r`, `center` for alignment, optionally offset past that edge e.g. `left-20`, `r-10`",
	}
	body := []string{
		"",
		"- `imagorpath` — the image to load",
		"- `x` — horizontal position: see [Horizontal position](#horizontal-position)",
	}

	args := arguments(body, []string{"imagorpath", "x"}, sections(doc))
	if len(args) != 2 {
		t.Fatalf("expected 2 arguments, got %d: %+v", len(args), args)
	}
	if args[1].Lead != "horizontal position:" {
		t.Errorf("lead = %q, want the link stripped so the forms carry it", args[1].Lead)
	}
	want := []string{
		"Positive number indicates position from the left, negative from the right",
		"`left` or `l`, `right` or `r`, `center` for alignment, optionally offset past that edge e.g. `left-20`, `r-10`",
	}
	if !reflect.DeepEqual(args[1].Forms, want) {
		t.Errorf("forms = %#v, want the linked section's bullets %#v", args[1].Forms, want)
	}
}

// The accepted forms stay separate. Joined into one sentence they read as a
// single requirement, which is how a caller mistakes an inset for an offset past
// the edge.
func TestArgumentsKeepFormsSeparate(t *testing.T) {
	body := []string{
		"- `x` — horizontal position:",
		"  - Positive number indicates position from the left, negative from the right",
		"  - `repeat` to tile horizontally",
	}
	args := arguments(body, []string{"x"}, nil)
	if len(args) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(args))
	}
	if len(args[0].Forms) != 2 {
		t.Fatalf("expected 2 forms, got %#v", args[0].Forms)
	}
	if args[0].Lead != "horizontal position:" {
		t.Errorf("lead = %q, want the argument's own name removed", args[0].Lead)
	}
}

// A markdown rule must not be read as an accepted form. Left in, it trims to
// "--" and reaches the caller as one.
func TestArgumentsSkipMarkdownRules(t *testing.T) {
	body := []string{
		"- `x` — horizontal position:",
		"  - `repeat` to tile horizontally",
		"---",
		"",
		"Something else entirely.",
	}
	args := arguments(body, []string{"x"}, nil)
	if len(args) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(args))
	}
	for _, f := range args[0].Forms {
		if f == "--" || f == "-" {
			t.Errorf("a markdown rule reached the caller as a form: %#v", args[0].Forms)
		}
	}
}

func TestAnchorOf(t *testing.T) {
	for heading, want := range map[string]string{
		"### Horizontal position":              "horizontal-position",
		"## Position arguments":                "position-arguments",
		"### `image(imagorpath, x, y)`":        "imageimagorpath-x-y",
		"#### `format(svg)`":                   "formatsvg",
		"### `focal(AxB:CxD)` or `focal(X,Y)`": "focalaxbcxd-or-focalxy",
	} {
		if got := anchorOf(heading); got != want {
			t.Errorf("anchorOf(%q) = %q, want %q", heading, got, want)
		}
	}
}
