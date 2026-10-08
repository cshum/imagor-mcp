// Package filters exposes the imagor filter catalogue so that imagor-mcp can
// reject filter names and argument counts imagor would silently ignore.
//
// imagor accepts any "name(args)" in a filters: segment. Unknown names and
// malformed argument counts produce no error: the request returns 200 with the
// image unchanged. That is a reasonable choice for a permissive server and a
// dangerous one for an agent, which cannot tell a working URL from an
// inert one. This package is the strict layer, kept outside imagor.
//
// The catalogue is generated from the imagor version pinned in go.mod. See
// cmd/genfilters.
package filters

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/cshum/imagor/imagorpath"
)

//go:generate go run ../../cmd/genfilters

// Spec describes one imagor filter.
type Spec struct {
	// MinArgs is the smallest accepted argument count.
	MinArgs int
	// MaxArgs is the largest accepted argument count, or -1 when the filter's
	// signature is undocumented and arity is therefore not validated.
	MaxArgs int
	// Documented reports whether the signature came from imagor's filter
	// reference. Undocumented filters are name-checked only.
	Documented bool
}

// Lookup returns the spec for a filter name, including any filter names
// supplied by the operator for local extensions.
func Lookup(name string) (Spec, bool) {
	if s, ok := catalog[name]; ok {
		return s, true
	}
	if extraFilter(name) {
		return Spec{MaxArgs: -1, Documented: false}, true
	}
	return Spec{}, false
}

// Names returns every known filter name, sorted.
func Names() []string {
	out := make([]string, 0, len(catalog))
	for name := range catalog {
		out = append(out, name)
	}
	out = append(out, extraNames()...)
	sort.Strings(out)
	return out
}

// Catalog returns the generated catalogue. Operator-supplied extra filters are
// not included, so callers can tell the two apart.
func Catalog() map[string]Spec {
	out := make(map[string]Spec, len(catalog))
	for k, v := range catalog {
		out[k] = v
	}
	return out
}

// Validate checks a filter name and its argument string against the catalogue.
//
// Arguments are split with imagor's own SplitArgs, so the count this function
// validates is exactly the count imagor would receive.
func Validate(name, args string) error {
	spec, ok := Lookup(name)
	if !ok {
		return fmt.Errorf("unknown filter %q%s; imagor ignores unrecognised filters silently, "+
			"so this would have produced an unchanged image with a 200 response. "+
			"Call list_imagor_filters to see the %d supported filters",
			name, suggest(name), len(Names()))
	}
	if !spec.Documented || spec.MaxArgs < 0 {
		return nil
	}
	got := len(imagorpath.SplitArgs(args))
	switch {
	case got < spec.MinArgs:
		return fmt.Errorf("filter %q expects at least %d argument(s) but got %d%s",
			name, spec.MinArgs, got, arityHint(name, spec))
	case got > spec.MaxArgs:
		return fmt.Errorf("filter %q accepts at most %d argument(s) but got %d%s",
			name, spec.MaxArgs, got, arityHint(name, spec))
	}
	return nil
}

func arityHint(name string, spec Spec) string {
	if spec.MinArgs == spec.MaxArgs {
		return fmt.Sprintf(" (%s takes %d)", name, spec.MinArgs)
	}
	return fmt.Sprintf(" (%s takes %d to %d)", name, spec.MinArgs, spec.MaxArgs)
}

// extraEnv is the environment variable naming operator-supplied filters, for
// extensions and forks that register filters imagor does not document.
const extraEnv = "IMAGOR_MCP_EXTRA_FILTERS"

func extraNames() []string {
	raw := strings.TrimSpace(os.Getenv(extraEnv))
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func extraFilter(name string) bool {
	for _, n := range extraNames() {
		if n == name {
			return true
		}
	}
	return false
}

// suggest offers up to three near-miss filter names, to catch typos such as
// "grayscal" for "grayscale".
func suggest(name string) string {
	type scored struct {
		name string
		dist int
	}
	var near []scored
	for _, candidate := range Names() {
		if d := editDistance(strings.ToLower(name), candidate); d <= 3 {
			near = append(near, scored{candidate, d})
		}
	}
	if len(near) == 0 {
		return ""
	}
	sort.Slice(near, func(i, j int) bool {
		if near[i].dist != near[j].dist {
			return near[i].dist < near[j].dist
		}
		return near[i].name < near[j].name
	})
	if len(near) > 3 {
		near = near[:3]
	}
	names := make([]string, 0, len(near))
	for _, n := range near {
		names = append(names, n.name)
	}
	return ". Did you mean " + strings.Join(names, ", ") + "?"
}

// editDistance is the Levenshtein distance between two strings.
func editDistance(a, b string) int {
	if a == b {
		return 0
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, min(curr[j-1]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}
