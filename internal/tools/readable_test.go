package tools

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSkillResultCarriesReadableText(t *testing.T) {
	out := SkillOutput{Skill: "# Heading\n\nA paragraph.\n"}
	res := skillResult(out)
	if len(res.Content) != 1 {
		t.Fatalf("want one content block, got %d", len(res.Content))
	}
	txt, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("want a text block, got %T", res.Content[0])
	}
	// The point of the change: a document travels with its own line breaks rather
	// than escaped inside a JSON envelope.
	if txt.Text != out.Skill {
		t.Errorf("skill text should be returned verbatim, got %q", txt.Text)
	}
	if strings.Contains(txt.Text, `\\n`) {
		t.Errorf("text still contains an escaped newline: %q", txt.Text)
	}
}

func TestFiltersText(t *testing.T) {
	out := ListFiltersOutput{
		ImagorVersion: "v1.9.10",
		Count:         2,
		Filters: []FilterDoc{
			{Name: "round_corner", Signature: "round_corner(rx [, ry [, color]])",
				Description: "Adds rounded corners.", Documented: true},
			{Name: "undocumented_thing", Signature: "undocumented_thing()", Documented: false},
		},
	}
	txt := filtersText(out)
	for _, want := range []string{
		"2 filters, as of imagor v1.9.10.",
		"round_corner(rx [, ry [, color]])",
		"Adds rounded corners.",
		// The signature leads, so a caller can copy the form a path needs.
		"undocumented_thing()",
		// A filter with no upstream description has to say so rather than look empty.
		"not documented upstream",
	} {
		if !strings.Contains(txt, want) {
			t.Errorf("missing %q in:\n%s", want, txt)
		}
	}

	// A search that matched nothing reports why rather than returning silence.
	empty := filtersText(ListFiltersOutput{ImagorVersion: "v1.9.10", Note: `no filter name contains "zzz"`})
	if !strings.Contains(empty, "no filter name contains") {
		t.Errorf("want the note carried through, got %q", empty)
	}
	if strings.Contains(empty, "\n\n") {
		t.Errorf("no filters should not emit entry spacing, got %q", empty)
	}
}

func TestDescribeText(t *testing.T) {
	out := DescribeOutput{
		Mode: "unsafe", ImagorVersion: "v1.9.10", FilterCount: 46,
		SignerType: "sha1", Truncate: 0, Signed: false,
		BaseURL:     "http://localhost:8000",
		SetupNote:   "Three settings must agree.",
		Limitations: []string{"does not upload"},
	}
	txt := describeText(out)
	for _, want := range []string{
		"mode unsafe, imagor v1.9.10, 46 filters.",
		"urls are built against http://localhost:8000",
		"signing: sha1, truncate 0, secret false",
		"Three settings must agree.",
		"does not upload",
	} {
		if !strings.Contains(txt, want) {
			t.Errorf("missing %q in:\n%s", want, txt)
		}
	}

	// Unsigned and unconfigured still has to render, since that is the state a
	// fresh install is in.
	bare := describeText(DescribeOutput{Mode: "signed", SignerType: "sha256", Signed: true})
	if !strings.Contains(bare, "mode signed") || strings.Contains(bare, "urls are built against") {
		t.Errorf("an empty base URL should be omitted, got:\n%s", bare)
	}
	if strings.Contains(bare, "limitations:") {
		t.Errorf("no limitations should not print the heading, got:\n%s", bare)
	}
}
