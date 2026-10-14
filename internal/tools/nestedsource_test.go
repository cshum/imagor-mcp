package tools

import (
	"strings"
	"testing"

	"github.com/cshum/imagor/imagorpath"
)

const nestedSrc = "https://s.imagor.net/photos/a.jpg"

// An image argument that is itself an imagor path cannot be written at the top
// level: parameters come before filters, so imagor reads the whole token as one
// source and fails to load it. Handing the path in has to produce the path.
func TestAPathHandedInAsTheImageIsTheRequest(t *testing.T) {
	in := TransformInput{Image: "fit-in/491x414/filters:fill(none)/" + nestedSrc}
	p, err := in.toParams()
	if err != nil {
		t.Fatalf("toParams: %v", err)
	}
	if p.Image != nestedSrc {
		t.Errorf("the image should be the source, got %q", p.Image)
	}
	if !p.FitIn || p.Width != 491 || p.Height != 414 {
		t.Errorf("the path's own parameters should apply, got %dx%d fit-in=%v", p.Width, p.Height, p.FitIn)
	}
	got := imagorpath.GeneratePath(p)
	if strings.Contains(got, "%2F") {
		t.Errorf("the path was escaped as one image token, which imagor loads as a url: %s", got)
	}
}

// With a transformation of its own the image has to be nested, and nested means
// path:, the only source that resolves a path in-process.
func TestAPathUnderATransformationIsNested(t *testing.T) {
	in := TransformInput{Image: "fit-in/491x414/filters:fill(none)/" + nestedSrc, Width: 515, Height: 438}
	p, err := in.toParams()
	if err != nil {
		t.Fatalf("toParams: %v", err)
	}
	if !strings.HasPrefix(p.Image, "path:") {
		t.Fatalf("expected a nested source, got %q", p.Image)
	}
	got := imagorpath.GeneratePath(p)
	if !strings.Contains(got, "path:") {
		t.Errorf("the nesting was lost: %s", got)
	}
}

// path:X with nothing outside the wrapper asks imagor to resolve X to pixels
// and encode the result again, so X's own quality() and format() do not apply.
// X is the request instead.
func TestABareWrapperIsCollapsed(t *testing.T) {
	inner := "1200x630/filters:quality(30)/color:102542"
	p, err := TransformInput{Image: "path:" + inner}.toParams()
	if err != nil {
		t.Fatalf("toParams: %v", err)
	}
	if p.Image != "color:102542" {
		t.Errorf("expected the inner path to become the request, got image %q", p.Image)
	}
	if got := imagorpath.GeneratePath(p); got != inner {
		t.Errorf("collapsed path = %q, want %q", got, inner)
	}
}

// The wrapper is kept when it is doing work, and when the inner path does not
// survive a parse-then-regenerate it is left alone rather than guessed at.
func TestTheWrapperIsKeptWhenItDoesWork(t *testing.T) {
	inner := "300x200/filters:round_corner(20,20)/color:102542"
	p, err := TransformInput{Image: "path:" + inner, Width: 100}.toParams()
	if err != nil {
		t.Fatalf("toParams: %v", err)
	}
	if !strings.HasPrefix(p.Image, "path:") {
		t.Errorf("a transformation outside the wrapper needs the wrapper: %q", p.Image)
	}
	if p.Width != 100 {
		t.Errorf("width = %d, want 100", p.Width)
	}
}

// A source that is not a path is left exactly as it was found.
func TestPlainSourcesAreLeftAlone(t *testing.T) {
	for _, image := range []string{
		nestedSrc,
		"photos/a.jpg",
		"color:red",
		"example.com/a.jpg",
		"path:" + nestedSrc,
	} {
		if got, ok := nestedSource(image); ok {
			t.Errorf("nestedSource(%q) = %q, want not a path", image, got)
		}
	}
	if got, ok := nestedSource("300x200/a.jpg"); !ok || got != "300x200/a.jpg" {
		t.Errorf("nestedSource on a path = %q, %v; want the path and true", got, ok)
	}
}
