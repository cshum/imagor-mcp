package filters

import (
	"strings"
	"testing"
)

// The catalogue exists to catch filters imagor would ignore in silence, so the
// rejections below are the point of the package, not incidental behaviour.
func TestUnknownFilterIsRejected(t *testing.T) {
	for _, name := range []string{"vintage", "sepia", "smartcrop", "autoorient"} {
		err := Validate(name, "70")
		if err == nil {
			t.Fatalf("filter %q must be rejected: imagor would ignore it and return the image unchanged", name)
		}
		if !strings.Contains(err.Error(), "unknown filter") {
			t.Errorf("filter %q: error should name the problem, got %q", name, err)
		}
	}
}

func TestTypoSuggestionIsOffered(t *testing.T) {
	err := Validate("grayscal", "")
	if err == nil {
		t.Fatal("expected a rejection for the misspelling")
	}
	if !strings.Contains(err.Error(), "grayscale") {
		t.Errorf("expected a suggestion of grayscale, got %q", err)
	}
}

func TestArityIsEnforced(t *testing.T) {
	for _, tc := range []struct {
		name, args string
		wantErr    bool
		why        string
	}{
		{"blur", "5", false, "blur takes one argument"},
		{"blur", "", true, "missing required argument"},
		{"blur", "1,2,3", true, "too many arguments, imagor would ignore the filter"},
		{"grayscale", "", false, "no arguments is correct"},
		{"grayscale", "1", true, "grayscale takes none"},
		{"text", "hello,10,10", false, "three required arguments"},
		{"text", "hello,10", true, "one required argument short"},
		{"image", "path,x,y", false, "three required arguments"},
		{"redact", "", false, "all arguments optional"},
		{"redact", "a,b,c", true, "more than the two optional arguments"},
		{"expire", "1700000000000", false, "exactly one argument"},
	} {
		t.Run(tc.name+"/"+tc.args, func(t *testing.T) {
			err := Validate(tc.name, tc.args)
			if tc.wantErr && err == nil {
				t.Fatalf("%s(%s) should be rejected: %s", tc.name, tc.args, tc.why)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("%s(%s) should be accepted (%s), got: %v", tc.name, tc.args, tc.why, err)
			}
		})
	}
}

// focal is documented in two forms, "focal(AxB:CxD)" and "focal(X,Y)", so the
// generator must widen the accepted range to cover both rather than reject a
// legitimate one-argument call.
func TestMultipleDocumentedFormsAreUnioned(t *testing.T) {
	if err := Validate("focal", "AxB:CxD"); err != nil {
		t.Errorf("single-argument focal should be accepted: %v", err)
	}
	if err := Validate("focal", "100,200"); err != nil {
		t.Errorf("two-argument focal should be accepted: %v", err)
	}
	if err := Validate("focal", "100,200,300"); err == nil {
		t.Error("three-argument focal should be rejected")
	}
}

// Filters registered in imagor but absent from its reference cannot have their
// signatures derived, so they are name-checked only. Rejecting them would
// block legitimate use.
func TestUndocumentedFiltersAcceptAnyArity(t *testing.T) {
	for _, name := range []string{"label", "modulate"} {
		spec, ok := Lookup(name)
		if !ok {
			t.Fatalf("%q is registered in imagor and must be present in the catalogue", name)
		}
		if spec.Documented {
			t.Errorf("%q has no documented signature, Documented should be false", name)
		}
		if err := Validate(name, "a,b,c,d,e"); err != nil {
			t.Errorf("%q should accept any arity, got: %v", name, err)
		}
	}
}

func TestCatalogueShape(t *testing.T) {
	names := Names()
	if len(names) < 40 {
		t.Fatalf("catalogue looks truncated: %d filters", len(names))
	}
	for _, name := range names {
		if name != strings.ToLower(name) {
			t.Errorf("filter %q is not lowercase", name)
		}
		spec, ok := Lookup(name)
		if !ok {
			t.Errorf("Lookup failed for a name returned by Names: %q", name)
			continue
		}
		if spec.Documented && spec.MinArgs < 0 {
			t.Errorf("filter %q: documented filters must have a non-negative minimum", name)
		}
		if spec.Documented && spec.MaxArgs < spec.MinArgs {
			t.Errorf("filter %q: max %d is below min %d", name, spec.MaxArgs, spec.MinArgs)
		}
	}
	// The version is baked in at generation time and reported to clients.
	if ImagorVersion == "" {
		t.Error("ImagorVersion must be set")
	}
}

// Filter names in the catalogue must be real, so a sample of well-known ones is
// checked for presence rather than trusting the generator blindly.
func TestWellKnownFiltersArePresent(t *testing.T) {
	for _, name := range []string{
		"blur", "brightness", "contrast", "crop", "fill", "format", "focal",
		"grayscale", "image", "invert", "quality", "rotate", "sharpen",
		"strip_exif", "strip_icc", "text", "watermark", "expire", "max_age",
		"max_bytes", "max_frames", "dpi", "page", "pixelate", "background_color",
	} {
		if _, ok := Lookup(name); !ok {
			t.Errorf("well-known filter %q is missing from the catalogue", name)
		}
	}
}

// Extensions and forks can register filters imagor does not document, so the
// strictness must be escapable without patching the binary.
func TestExtraFiltersEscapeHatch(t *testing.T) {
	t.Setenv(extraEnv, "custom_filter, another_one")

	if err := Validate("custom_filter", "anything"); err != nil {
		t.Errorf("extra filter should be accepted: %v", err)
	}
	if err := Validate("another_one", ""); err != nil {
		t.Errorf("extra filter should be accepted: %v", err)
	}
	if err := Validate("not_registered", ""); err == nil {
		t.Error("a filter not in the catalogue or the extra list must still be rejected")
	}
	if got := len(Names()); got <= len(Catalog()) {
		t.Errorf("Names should include extra filters: %d names vs %d catalogue entries", got, len(Catalog()))
	}
}
