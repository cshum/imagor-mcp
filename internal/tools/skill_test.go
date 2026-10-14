package tools

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The skill is split on `##`, so the head is everything before the first one and
// the sections are the rest. If the embedded document ever loses that shape the
// outline would silently become the whole thing, which is the cost this exists to
// avoid.
func TestSkillPartsSplitsTheDocument(t *testing.T) {
	head, secs := skillParts()
	if head == "" {
		t.Fatal("head must not be empty: it carries the frontmatter and the opening statement")
	}
	if len(secs) < 5 {
		t.Fatalf("expected the skill's sections, got %d", len(secs))
	}
	if strings.HasPrefix(head, "## ") {
		t.Errorf("head should stop before the first `##`, got %q", head[:40])
	}
	seen := map[string]bool{}
	for _, s := range secs {
		if s.Slug == "" || s.Title == "" {
			t.Errorf("section with no slug or title: %+v", s)
		}
		if !strings.HasPrefix(s.Body, "## ") {
			t.Errorf("section %q should include its heading", s.Slug)
		}
		if seen[s.Slug] {
			t.Errorf("duplicate slug %q: a caller asking for it would get the wrong section", s.Slug)
		}
		seen[s.Slug] = true
	}
	// The whole point: the outline is a fraction of the document, not all of it.
	if total := len(head) + len(secs[0].Body); total >= len(head)+1 && len(head) > 4000 {
		t.Errorf("head is %d chars; it should be the frontmatter and intro only", len(head))
	}
}

func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"Resize modes": "resize-modes",
		"Placing text is a search, not a calculation": "placing-text-is-a-search-not-a-calculation",
		"Colors and awkward URLs":                     "colors-and-awkward-urls",
		"When a value has to be encoded":              "when-a-value-has-to-be-encoded",
	} {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGistSkipsStructure(t *testing.T) {
	// A heading, a blank line, then a table would otherwise gist as markdown
	// furniture rather than as what the section is about.
	got := gist([]string{"", "# sub", "| a | b |", "- bullet", "Real prose here."})
	if got != "Real prose here." {
		t.Errorf("gist = %q, want the first prose line", got)
	}
	if gist([]string{"", "```", "code"}) != "" {
		t.Error("a section with no prose should gist as empty rather than as code")
	}
	long := gist([]string{strings.Repeat("x", 200)})
	if len(long) > 100 || !strings.HasSuffix(long, "...") {
		t.Errorf("a long gist should be trimmed, got %d chars", len(long))
	}
}

func TestSkillToolServesSections(t *testing.T) {
	c := Config{}
	head, secs := skillParts()

	res, out, err := c.SkillTool(t.Context(), nil, SkillInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Sections) != len(secs) {
		t.Errorf("outline should list every section: %d vs %d", len(out.Sections), len(secs))
	}
	if !strings.Contains(out.Skill, head) {
		t.Error("the outline should carry the head, so a caller knows what the skill is for")
	}
	full := len(head)
	for _, s := range secs {
		full += len(s.Body)
	}
	if len(out.Skill) >= full/2 {
		t.Errorf("the outline is %d chars against a %d char document; it should be a fraction",
			len(out.Skill), full)
	}
	txt, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("want a text block, got %T", res.Content[0])
	}
	if txt.Text != out.Skill {
		t.Error("the text block should carry the readable outline")
	}

	// A section by slug.
	_, byslug, err := c.SkillTool(t.Context(), nil, SkillInput{Section: secs[2].Slug})
	if err != nil {
		t.Fatal(err)
	}
	if byslug.Section != secs[2].Slug {
		t.Errorf("want section %q, got %q", secs[2].Slug, byslug.Section)
	}
	if byslug.Skill != secs[2].Body {
		t.Error("a named section should be returned verbatim")
	}
	if len(byslug.Sections) != len(secs) {
		t.Error("every name should travel with a section, so a wrong guess costs no extra call")
	}

	// A section by the heading it was written under.
	_, bytitle, err := c.SkillTool(t.Context(), nil, SkillInput{Section: secs[2].Title})
	if err != nil {
		t.Fatal(err)
	}
	if bytitle.Section != secs[2].Slug {
		t.Errorf("a heading should resolve to its slug, got %q", bytitle.Section)
	}

	// An unknown name is answered with the outline, not an error.
	_, unknown, err := c.SkillTool(t.Context(), nil, SkillInput{Section: "no-such-section"})
	if err != nil {
		t.Fatalf("an unknown section should not be an error: %v", err)
	}
	if unknown.Section != "" {
		t.Error("nothing was matched, so no section should be reported")
	}
	if !strings.Contains(unknown.Note, "no-such-section") {
		t.Errorf("the note should name what was asked for, got %q", unknown.Note)
	}
	if len(unknown.Sections) != len(secs) {
		t.Error("the outline should come back so the caller can pick a real name")
	}
}
