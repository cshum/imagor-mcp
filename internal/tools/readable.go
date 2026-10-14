package tools

// A tool result carries two views: content, which a model reads, and
// structuredContent, which a program reads. Left to itself the SDK fills content
// with the JSON of the structured output, and anything document-shaped then
// arrives as a single line with escaped newlines - legible to a parser and to
// nothing else. A caller seeing that has to reconstruct the line breaks before it
// can read the document at all.
//
// verify_imagor_url already sets content explicitly for the same reason. So do
// the tools below, which return prose the caller reads rather than data it passes
// on.
//
// get_cookbook_recipe is deliberately not among them: a recipe's template is
// handed straight back to render_template, so the caller wants the structure, not
// a rendering of it.

import (
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// textResult is a tool result whose content is prose, with the structured output
// still filled in by the SDK for callers that read that instead.
func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// skillResult returns the skill as markdown. It is the one document this server
// serves in full, so it travels with its own line breaks rather than as an
// escaped string inside a JSON envelope.
func skillResult(out SkillOutput) *mcp.CallToolResult {
	return textResult(out.Skill)
}

// filtersText renders the catalogue as a list, one entry per filter. The
// signature is the line that matters - it is what a path has to contain - so it
// leads, with the description indented beneath it.
func filtersText(out ListFiltersOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d filters, as of imagor %s.\n", out.Count, out.ImagorVersion)
	if out.Note != "" {
		fmt.Fprintf(&b, "%s\n", out.Note)
	}
	for _, f := range out.Filters {
		fmt.Fprintf(&b, "\n%s\n", f.Signature)
		if f.Description != "" {
			fmt.Fprintf(&b, "  %s\n", f.Description)
		}
		if !f.Documented {
			b.WriteString("  (not documented upstream; the signature is inferred from imagor's reference)\n")
		}
	}
	return b.String()
}

// describeText renders the server's own report. The setup note goes first
// because a mismatch between the two sides is silent until a request fails, and
// the limitations last so they are not buried above the settings they qualify.
func describeText(out DescribeOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "mode %s, imagor %s, %d filters.\n", out.Mode, out.ImagorVersion, out.FilterCount)
	if out.BaseURL != "" {
		fmt.Fprintf(&b, "urls are built against %s\n", out.BaseURL)
	}
	fmt.Fprintf(&b, "signing: %s, truncate %v, secret %t\n",
		out.SignerType, out.Truncate, out.Signed)
	if out.SetupNote != "" {
		fmt.Fprintf(&b, "\n%s\n", out.SetupNote)
	}
	if len(out.Limitations) > 0 {
		b.WriteString("\nlimitations:\n")
		for _, l := range out.Limitations {
			fmt.Fprintf(&b, "  - %s\n", l)
		}
	}
	return b.String()
}
