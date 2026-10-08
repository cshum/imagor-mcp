// Package pathgen builds and validates imagor paths from typed inputs.
//
// The point of this package is that it validates rather than assembles:
// every generated path is parsed back and compared against the parameters
// that produced it, so a malformed composition is refused before it ever
// reaches imagor.
package pathgen

import (
	"fmt"
	"strings"

	"github.com/cshum/imagor/imagorpath"
)

// Build renders an unsigned imagorpath.
func Build(p imagorpath.Params) string {
	return imagorpath.GeneratePath(p)
}

// BuildURL renders a path prefixed with its signature, or "unsafe/" when
// signer is nil.
func BuildURL(p imagorpath.Params, signer imagorpath.Signer) string {
	return imagorpath.Generate(p, signer)
}

// Validate generates the path for p, parses it back, and reports any field
// that did not survive the round trip.
//
// A returned error means the parameters describe a composition imagor would
// not reproduce, so the path must not be handed to a client.
func Validate(p imagorpath.Params) error {
	path := imagorpath.GeneratePath(p)
	back := imagorpath.Parse(path)

	var problems []string
	if p.Image != back.Image {
		problems = append(problems, fmt.Sprintf("image: in=%q out=%q", p.Image, back.Image))
	}
	if p.Width != back.Width || p.Height != back.Height {
		problems = append(problems, fmt.Sprintf("size: in=%dx%d out=%dx%d", p.Width, p.Height, back.Width, back.Height))
	}
	if p.CropLeft != back.CropLeft || p.CropTop != back.CropTop ||
		p.CropRight != back.CropRight || p.CropBottom != back.CropBottom {
		problems = append(problems, fmt.Sprintf(
			"crop: in=%v,%v,%v,%v out=%v,%v,%v,%v",
			p.CropLeft, p.CropTop, p.CropRight, p.CropBottom,
			back.CropLeft, back.CropTop, back.CropRight, back.CropBottom))
	}
	if p.HAlign != back.HAlign {
		problems = append(problems, fmt.Sprintf("h_align: in=%q out=%q", p.HAlign, back.HAlign))
	}
	if p.VAlign != back.VAlign {
		problems = append(problems, fmt.Sprintf("v_align: in=%q out=%q", p.VAlign, back.VAlign))
	}
	if p.Trim != back.Trim || p.TrimBy != back.TrimBy {
		problems = append(problems, fmt.Sprintf("trim: in=%v/%q out=%v/%q", p.Trim, p.TrimBy, back.Trim, back.TrimBy))
	}
	for _, f := range []struct {
		name    string
		in, out bool
	}{
		{"fit_in", p.FitIn, back.FitIn},
		{"stretch", p.Stretch, back.Stretch},
		{"smart", p.Smart, back.Smart},
		{"h_flip", p.HFlip, back.HFlip},
		{"v_flip", p.VFlip, back.VFlip},
		{"meta", p.Meta, back.Meta},
	} {
		if f.in != f.out {
			problems = append(problems, fmt.Sprintf("%s: in=%v out=%v", f.name, f.in, f.out))
		}
	}
	if !filtersEqual(p.Filters, back.Filters) {
		problems = append(problems, fmt.Sprintf("filters: in=%v out=%v", p.Filters, back.Filters))
	}

	if len(problems) > 0 {
		return fmt.Errorf("path does not round-trip (%s): %s", path, strings.Join(problems, "; "))
	}
	return nil
}

// filtersEqual compares two filter slices, treating nil and empty as equal.
// A caller that builds no filters produces an empty slice while a parsed path
// produces nil; both describe the same transformation.
func filtersEqual(a, b imagorpath.Filters) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// MustValidate panics on a path that does not round-trip. For tests.
func MustValidate(p imagorpath.Params) {
	if err := Validate(p); err != nil {
		panic(err)
	}
}
