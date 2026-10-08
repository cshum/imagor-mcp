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

// parseDocs reads filter signatures from the filter reference headings.
//
// A heading may carry more than one form, as in
// "`focal(AxB:CxD)` or `focal(X,Y)`", so every alternative is parsed and the
// accepted range is widened to cover all of them.
func parseDocs(path string) (map[string]spec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]spec{}
	for _, line := range strings.Split(string(b), "\n") {
		m := headingRe.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		heading := strings.ReplaceAll(m[1], "`", "")
		for _, form := range strings.Split(heading, " or ") {
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
				out[name] = prev
				continue
			}
			out[name] = spec{Name: name, MinArgs: min, MaxArgs: max, Documented: true}
		}
	}
	return out, nil
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
		fmt.Fprintf(&b, "\t%q: {MinArgs: %d, MaxArgs: %d, Documented: %t},\n", name, s.MinArgs, s.MaxArgs, s.Documented)
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
