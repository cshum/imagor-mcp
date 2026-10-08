package pathgen

import (
	"crypto/sha1"
	"crypto/sha256"
	"hash"
	"strings"
	"testing"

	"github.com/cshum/imagor/imagorpath"
)

// docsPath and the expected signatures below are taken verbatim from
// imagor's published security documentation, so they verify our signer
// against an external ground truth rather than against itself.
const docsPath = "500x500/top/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png"

func TestSignerMatchesDocsVectors(t *testing.T) {
	for _, tc := range []struct {
		name     string
		alg      func() hash.Hash
		truncate int
		want     string
	}{
		{"sha1 default", sha1.New, 0, "cST4Ko5_FqwT3BDn-Wf4gO3RFSk="},
		{"sha256 truncate 40", sha256.New, 40, "IGEn3TxngivD0jy4uuiZim2bdUCvhcnVi1Nm0xGy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := imagorpath.NewHMACSigner(tc.alg, tc.truncate, "mysecret").Sign(docsPath)
			if got != tc.want {
				t.Fatalf("signature mismatch\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

func TestBuildAndValidate(t *testing.T) {
	p := imagorpath.Params{
		Image:  "raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png",
		Width:  500,
		Height: 500,
		VAlign: imagorpath.VAlignTop,
		Filters: imagorpath.Filters{
			{Name: "fill", Args: "white"},
			{Name: "format", Args: "jpeg"},
			{Name: "quality", Args: "80"},
		},
	}
	if err := Validate(p); err != nil {
		t.Fatalf("valid params failed validation: %v", err)
	}
	want := "500x500/top/filters:fill(white):format(jpeg):quality(80)/raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png"
	if got := Build(p); got != want {
		t.Fatalf("path mismatch\n got: %s\nwant: %s", got, want)
	}
}

// Regression: a caller that specifies no filters must not be reported as a
// mismatch merely because the parsed path yields nil where the caller built an
// empty slice. Both describe the same transformation.
func TestValidateAcceptsNoFilters(t *testing.T) {
	p := imagorpath.Params{
		Image:   "example.com/a.jpg",
		Width:   320,
		Height:  240,
		Smart:   true,
		Filters: imagorpath.Filters{},
	}
	if err := Validate(p); err != nil {
		t.Fatalf("params without filters must validate: %v", err)
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	p := imagorpath.Params{
		Image:  "example.com/a.jpg",
		Width:  320,
		Height: 240,
		Smart:  true,
		Filters: imagorpath.Filters{
			{Name: "quality", Args: "75"},
		},
	}
	first := Build(p)
	for i := 0; i < 100; i++ {
		if got := Build(p); got != first {
			t.Fatalf("non-deterministic at iteration %d: %s != %s", i, got, first)
		}
	}
}

func TestBuildURLModes(t *testing.T) {
	p := imagorpath.Params{
		Image:  "example.com/a.jpg",
		Width:  320,
		Height: 240,
	}
	signed := BuildURL(p, imagorpath.NewHMACSigner(sha256.New, 40, "s3cret"))
	parts := strings.SplitN(signed, "/", 2)
	if len(parts) != 2 || len(parts[0]) != 40 {
		t.Fatalf("expected 40-char signature prefix, got %q", signed)
	}
	if parts[1] != Build(p) {
		t.Fatalf("signed path body does not match Build output: %q vs %q", parts[1], Build(p))
	}

	unsafe := BuildURL(p, nil)
	if !strings.HasPrefix(unsafe, "unsafe/") {
		t.Fatalf("nil signer should produce unsafe prefix, got %q", unsafe)
	}
}

// The validation guard must actually reject a composition imagor cannot
// reproduce, otherwise it is decoration.
func TestValidateRejectsNonRoundTripping(t *testing.T) {
	// Negative dimensions are normalised into flip flags by GeneratePath,
	// so the parameter that went in is not the parameter that comes back.
	p := imagorpath.Params{
		Image:  "example.com/a.jpg",
		Width:  -320,
		Height: 240,
	}
	if err := Validate(p); err == nil {
		t.Fatal("expected validation to reject negative width, got nil")
	} else if !strings.Contains(err.Error(), "size") && !strings.Contains(err.Error(), "flip") {
		t.Fatalf("unexpected error: %v", err)
	}
}
