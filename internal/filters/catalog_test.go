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

// Refusal is reserved for outcomes that are certain; excess arguments warn
// because imagor's tolerance genuinely differs per filter.
func TestArityOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name, args string
		want       Outcome
		why        string
	}{
		{"blur", "5", Accepted, "blur takes exactly one argument"},
		{"blur", "", Rejected, "measured: blur() is silently ignored"},
		{"blur", "1,2,3", Warned, "measured: blur(1,2,3) is silently ignored, but other filters tolerate extras"},
		{"grayscale", "", Accepted, "takes no arguments"},
		{"grayscale", "1", Warned, "measured: grayscale(1) still applies, so it must not be refused"},
		{"text", "hello,10,10", Accepted, "three required arguments"},
		{"text", "hello,10", Rejected, "one required argument short"},
		{"image", "path,x,y", Accepted, "three required arguments"},
		{"redact", "", Accepted, "all arguments optional"},
		{"redact", "a,b,c", Warned, "more than the two documented optional arguments"},
		{"expire", "1700000000000", Accepted, "exactly one argument"},
	} {
		t.Run(tc.name+"/"+tc.args, func(t *testing.T) {
			got, msg := Check(tc.name, tc.args)
			if got != tc.want {
				t.Fatalf("Check(%q, %q) = %v, want %v (%s); message: %q",
					tc.name, tc.args, got, tc.want, tc.why, msg)
			}
			if got != Accepted && msg == "" {
				t.Error("non-accepted outcomes must explain themselves")
			}
		})
	}
}

// Validate must refuse only what Check rejects.
func TestValidateRefusesOnlyRejections(t *testing.T) {
	for _, tc := range []struct {
		name, args string
		wantErr    bool
	}{
		{"blur", "5", false},
		{"blur", "", true},
		{"blur", "1,2,3", false},  // warned, not refused
		{"grayscale", "1", false}, // warned, not refused
		{"vintage", "70", true},   // unknown name
	} {
		err := Validate(tc.name, tc.args)
		if tc.wantErr && err == nil {
			t.Errorf("Validate(%q, %q) should have refused", tc.name, tc.args)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("Validate(%q, %q) should not have refused: %v", tc.name, tc.args, err)
		}
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
	if got, msg := Check("focal", "100,200,300"); got != Warned {
		t.Errorf("three-argument focal should warn, got %v (%q)", got, msg)
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
