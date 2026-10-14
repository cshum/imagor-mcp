package tools

// The skill is served in sections rather than whole.
//
// It is 18 KB of prose and most of it is irrelevant to any one composition, so a
// caller that needs the part about placement should not pay for the parts about
// cookies or the cookbook. This is the same shape as a filesystem skill: the
// caller is told what exists and reads what it needs.
//
// It is a tool parameter rather than a set of resources on purpose. Resources are
// read when the application decides to read them, and a host that surfaces only
// tools never sees them at all - the host that most needs this. A model can always
// call a tool.
//
// The prompt is the deliberate path and still carries the whole document: a user
// who picks "compose an imagor image" from a menu has asked for exactly that.

import (
	"fmt"
	"strings"
	"sync"

	imagormcp "github.com/cshum/imagor-mcp"
)

// SkillInput selects part of the skill.
type SkillInput struct {
	Section string `json:"section,omitempty" jsonschema:"a section name from the list this tool returns when it is omitted; omit it to get the list"`
}

// skillSection is one `##` heading and everything under it.
type skillSection struct {
	Slug  string
	Title string
	Body  string
	Gist  string
}

var (
	skillOnce sync.Once
	skillHead string
	skillSecs []skillSection
)

// skillParts splits the embedded skill once. The text before the first `##` is
// the head: the frontmatter and the opening statement, which stay in every
// response because they say what the skill is for.
func skillParts() (string, []skillSection) {
	skillOnce.Do(func() {
		lines := strings.Split(imagormcp.Skill, "\n")
		start := -1
		for i, l := range lines {
			if strings.HasPrefix(l, "## ") {
				start = i
				break
			}
		}
		if start < 0 {
			skillHead = imagormcp.Skill
			return
		}
		skillHead = strings.TrimRight(strings.Join(lines[:start], "\n"), "\n")
		for i := start; i < len(lines); {
			title := strings.TrimSpace(strings.TrimPrefix(lines[i], "## "))
			end := len(lines)
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(lines[j], "## ") {
					end = j
					break
				}
			}
			body := strings.TrimRight(strings.Join(lines[i:end], "\n"), "\n")
			skillSecs = append(skillSecs, skillSection{
				Slug:  slugify(title),
				Title: title,
				Body:  body,
				Gist:  gist(lines[i+1 : end]),
			})
			i = end
		}
	})
	return skillHead, skillSecs
}

// gist is the first line of prose in a section: a heading alone does not tell a
// caller whether this is the section it wants.
func gist(lines []string) string {
	for _, l := range lines {
		t := strings.TrimSpace(l)
		// A fence ends the search: prose usually follows a code block, but a line
		// of code inside one is not a summary of anything.
		if strings.HasPrefix(t, "```") {
			return ""
		}
		if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "|") ||
			strings.HasPrefix(t, "-") {
			continue
		}
		if len(t) > 96 {
			t = strings.TrimSpace(t[:96]) + "..."
		}
		return t
	}
	return ""
}

func slugify(s string) string {
	var b strings.Builder
	last := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			last = false
		default:
			if !last {
				b.WriteByte('-')
				last = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func skillNames(secs []skillSection) []string {
	out := make([]string, 0, len(secs))
	for _, s := range secs {
		out = append(out, s.Slug)
	}
	return out
}

// skillOutline is what a caller gets for asking nothing: what the skill is, and
// what is in it.
func skillOutline(head string, secs []skillSection) string {
	var b strings.Builder
	b.WriteString(head)
	b.WriteString("\n\n## Sections\n\n")
	b.WriteString("This skill is served in sections. Call get_imagor_skill with a section name to " +
		"read one, or with none to get this list again.\n\n")
	for _, s := range secs {
		fmt.Fprintf(&b, "- %s", s.Slug)
		if s.Gist != "" {
			fmt.Fprintf(&b, " - %s", s.Gist)
		}
		b.WriteString("\n")
	}
	return b.String()
}
