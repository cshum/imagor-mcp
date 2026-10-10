// Command genfilters generates the imagor filter catalogue used by
// imagor-mcp to reject filter names and argument counts that imagor would
// silently ignore.
//
// It reads the imagor module pinned in go.mod, so the catalogue is tied to an
// exact released version rather than to whatever happens to be checked out.
// Sources, in order of authority:
//
//  1. docs/docs/filters.md — titles of the form "name(arg[, opt])", which give
//     the filter name and its accepted argument count.
//  2. processor/vipsprocessor/processor.go — the FilterMap literal, which
//     catches filters registered but not documented as a title.
//
// Run with: go generate ./internal/filters
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	imagorModule = "github.com/cshum/imagor"
	docsFile     = "docs/docs/filters.md"
	processFile  = "processor/vipsprocessor/processor.go"
	outFile      = "internal/filters/catalog_gen.go"
)

type spec struct {
	Name       string
	MinArgs    int
	MaxArgs    int
	Documented bool
	// Signature is the heading as written in the filter reference, and
	// Description the paragraph under it, so the catalogue describes filters in
	// imagor's own words rather than a paraphrase that can drift.
	Signature   string
	Description string
	// Arguments are the per-argument descriptions from the reference, which is
	// where the accepted forms are written down.
	Arguments []string
}

func main() {
	dir, err := goList("{{.Dir}}")
	must(err)
	version, err := goList("{{.Version}}")
	must(err)

	// Resolve the output against the module root so the generator behaves the
	// same whether it is run from the repository root or via go generate, which
	// executes with the package directory as the working directory.
	root, err := currentModuleDir()
	must(err)
	out := filepath.Join(root, outFile)

	docs, err := parseDocs(filepath.Join(dir, docsFile))
	must(err)
	registered, err := parseFilterMap(filepath.Join(dir, processFile))
	must(err)

	catalog := merge(docs, registered)
	must(write(out, version, catalog))

	undocumented := 0
	for _, s := range catalog {
		if !s.Documented {
			undocumented++
		}
	}
	fmt.Printf("genfilters: imagor %s -> %s\n", version, out)
	fmt.Printf("genfilters: %d filters (%d from docs, %d name-only from the FilterMap)\n",
		len(catalog), len(catalog)-undocumented, undocumented)
}

// currentModuleDir returns the directory of the module being built.
func currentModuleDir() (string, error) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		return "", fmt.Errorf("go list -m: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func goList(tmpl string) (string, error) {
	out, err := exec.Command("go", "list", "-m", "-f", tmpl, imagorModule).Output()
	if err != nil {
		return "", fmt.Errorf("go list -m %s: %w", imagorModule, err)
	}
	return strings.TrimSpace(string(out)), nil
}

var headingRe = regexp.MustCompile(`^###\s+(.+)$`)

// parseDocs reads filter signatures and summaries from the filter reference.
//
// A heading may carry more than one form, as in
// "`focal(AxB:CxD)` or `focal(X,Y)`", so every alternative is parsed and the
// accepted range is widened to cover all of them. The heading is kept as the
// signature and the paragraph under it as the description, so the catalogue
// carries imagor's own wording.
func parseDocs(path string) (map[string]spec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]spec{}
	for _, sec := range splitSections(string(b)) {
		desc := firstParagraph(sec.body)
		args := arguments(sec.body, signatureArgs(sec.heading))
		for _, form := range strings.Split(sec.heading, " or ") {
			name, min, max, ok := parseForm(strings.TrimSpace(form))
			if !ok {
				continue
			}
			if prev, exists := out[name]; exists {
				if min < prev.MinArgs {
					prev.MinArgs = min
				}
				if max > prev.MaxArgs {
					prev.MaxArgs = max
				}
				if prev.Description == "" {
					prev.Description = desc
				}
				out[name] = prev
				continue
			}
			out[name] = spec{
				Name: name, MinArgs: min, MaxArgs: max, Documented: true,
				Signature: sec.heading, Description: desc, Arguments: args,
			}
		}
	}
	return out, nil
}

// docsSection is one filter heading with the lines that follow it.
type docsSection struct {
	heading string
	body    []string
}

// splitSections groups the filter reference by heading, so a heading can be read
// together with the prose it introduces.
func splitSections(text string) []docsSection {
	var out []docsSection
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		if m := headingRe.FindStringSubmatch(line); m != nil {
			out = append(out, docsSection{heading: strings.ReplaceAll(m[1], "`", "")})
			continue
		}
		if len(out) > 0 {
			out[len(out)-1].body = append(out[len(out)-1].body, line)
		}
	}
	return out
}

// firstParagraph returns the summary directly under a heading. Bullet lists,
// tables and anything past the first blank line are left out: the catalogue
// carries a description, not the documentation. A section that opens with a
// list or a table has no summary.
func firstParagraph(body []string) string {
	var b strings.Builder
	for _, line := range body {
		t := strings.TrimSpace(line)
		if t == "" {
			if b.Len() > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(t, "-") || strings.HasPrefix(t, "|") ||
			strings.HasPrefix(t, "<") || strings.HasPrefix(t, "#") {
			if b.Len() == 0 {
				continue
			}
			break
		}
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(t)
	}
	return b.String()
}

// arguments reads the argument descriptions under a heading. Only bullets whose
// label matches an argument name in the signature are kept, so a section's other
// lists (supported formats, examples) are not mistaken for grammar.
func arguments(body []string, names []string) []string {
	if len(names) == 0 {
		return nil
	}
	want := map[string]bool{}
	for _, n := range names {
		want[n] = true
	}
	var out []string
	var cur strings.Builder
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			out = append(out, s)
		}
		cur.Reset()
	}
	for _, line := range body {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(line, "- ") {
			flush()
			label := bulletLabel(strings.TrimPrefix(trimmed, "- "))
			if !want[label] {
				continue
			}
			cur.WriteString(strings.TrimPrefix(trimmed, "- "))
			continue
		}
		// Sub-bullets carry the accepted forms, so they continue their argument.
		if cur.Len() == 0 || strings.HasPrefix(trimmed, "<") ||
			strings.HasPrefix(trimmed, "|") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasSuffix(cur.String(), ":") {
			cur.WriteString(" ")
		} else {
			cur.WriteString("; ")
		}
		cur.WriteString(strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
	}
	flush()
	for i, s := range out {
		out[i] = truncate(s, maxArgLen)
	}
	return out
}

// maxArgLen bounds one argument's description. The reference is more detailed
// than a catalogue entry needs to be.
const maxArgLen = 400

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n]) + "…"
}

// bulletLabel returns the argument name a bullet introduces, which the reference
// writes in backticks before a dash: "`x` — horizontal position".
func bulletLabel(bullet string) string {
	b := strings.TrimSpace(bullet)
	if !strings.HasPrefix(b, "`") {
		return ""
	}
	if end := strings.Index(b[1:], "`"); end >= 0 {
		return b[1 : 1+end]
	}
	return ""
}

// signatureArgs lists the argument names in a signature such as
// "image(imagorpath, x, y[, alpha[, blend_mode]])".
func signatureArgs(signature string) []string {
	open := strings.Index(signature, "(")
	closeIdx := strings.LastIndex(signature, ")")
	if open < 0 || closeIdx < open {
		return nil
	}
	inner := strings.NewReplacer("[", "", "]", "").Replace(signature[open+1 : closeIdx])
	var names []string
	for _, part := range strings.Split(inner, ",") {
		name := strings.TrimSpace(part)
		if i := strings.IndexAny(name, "= "); i >= 0 {
			name = name[:i]
		}
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// parseForm extracts the filter name and accepted argument count from a
// signature such as "image(imagorpath, x, y[, alpha[, blend_mode]])".
//
// Arguments outside any square bracket are required; arguments inside a
// bracket are optional, and each nested bracket adds another optional
// argument. Square brackets, not parentheses, carry the optionality.
func parseForm(form string) (name string, min, max int, ok bool) {
	open := strings.Index(form, "(")
	if open < 0 {
		return "", 0, 0, false
	}
	name = strings.TrimSpace(form[:open])
	if name == "" || strings.ContainsAny(name, " ()[]") {
		return "", 0, 0, false
	}
	closeIdx := strings.LastIndex(form, ")")
	if closeIdx < open {
		return "", 0, 0, false
	}
	inner := form[open+1 : closeIdx]
	if strings.TrimSpace(inner) == "" {
		return name, 0, 0, true
	}

	depth, segDepth := 0, -1
	required, optional := 0, 0
	flush := func() {
		switch {
		case segDepth == 0:
			required++
		case segDepth > 0:
			optional++
		}
		segDepth = -1
	}
	for _, ch := range inner {
		switch ch {
		case ',':
			flush()
		case '[':
			depth++
		case ']':
			depth--
		default:
			if segDepth < 0 && ch != ' ' && ch != '\t' {
				segDepth = depth
			}
		}
	}
	flush()
	return name, required, required + optional, true
}

var keyRe = regexp.MustCompile(`^"([a-z_0-9]+)":`)

// parseFilterMap lists the names registered in the vips processor's filter map.
func parseFilterMap(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var names []string
	inBlock := false
	for _, line := range strings.Split(string(b), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "v.Filters = FilterMap{"):
			inBlock = true
			continue
		case inBlock && trimmed == "}":
			inBlock = false
		}
		if !inBlock {
			continue
		}
		if m := keyRe.FindStringSubmatch(trimmed); m != nil {
			names = append(names, m[1])
		}
	}
	return names, nil
}

// merge joins both sources. Documented signatures win; names that only appear
// in the filter map get an unconstrained arity, because their signatures are
// not machine-readable.
func merge(docs map[string]spec, registered []string) map[string]spec {
	out := map[string]spec{}
	for name, s := range docs {
		out[name] = s
	}
	for _, name := range registered {
		if _, exists := out[name]; exists {
			continue
		}
		out[name] = spec{Name: name, MinArgs: 0, MaxArgs: -1, Documented: false}
	}
	return out
}

func write(path, version string, catalog map[string]spec) error {
	names := make([]string, 0, len(catalog))
	for name := range catalog {
		names = append(names, name)
	}
	sort.Strings(names)

	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by cmd/genfilters from imagor %s. DO NOT EDIT.\n\n", version)
	fmt.Fprintf(&b, "package filters\n\n")
	fmt.Fprintf(&b, "// ImagorVersion is the imagor release this catalogue was generated from.\n")
	fmt.Fprintf(&b, "const %s\n\n", fmt.Sprintf("ImagorVersion = %q", version))
	fmt.Fprintf(&b, "var catalog = map[string]Spec{\n")
	for _, name := range names {
		s := catalog[name]
		fmt.Fprintf(&b, "	%q: {MinArgs: %d, MaxArgs: %d, Documented: %t, Signature: %q, Description: %q, Arguments: %#v},\n",
			name, s.MinArgs, s.MaxArgs, s.Documented, s.Signature, s.Description, s.Arguments)
	}
	fmt.Fprintf(&b, "}\n")

	src, err := format.Source(b.Bytes())
	if err != nil {
		return fmt.Errorf("formatting generated source: %w", err)
	}
	return os.WriteFile(path, src, 0o644)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "genfilters:", err)
		os.Exit(1)
	}
}
