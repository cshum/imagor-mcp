package main

import (
	"context"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

// The count of tools was written into three files by hand and drifted the moment a
// tool was added or removed - nothing checked it. Rather than check a number, check
// the invariant the number was standing in for: every tool the server serves is
// documented, and every tool the docs name is served.
//
// That is exact, needs no count to be maintained, and cannot go stale.
var docToolRow = regexp.MustCompile("(?m)^\\| `([a-z_]+)` \\|")

func TestDocsListEveryToolAndNothingElse(t *testing.T) {
	s := connect(t, unsafeEnv)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := s.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	served := map[string]bool{}
	for _, tool := range res.Tools {
		served[tool.Name] = true
	}
	if len(served) == 0 {
		t.Fatal("no tools served")
	}

	raw, err := os.ReadFile("../../docs/tools.md")
	if err != nil {
		t.Fatalf("docs/tools.md: %v", err)
	}
	documented := map[string]bool{}
	for _, m := range docToolRow.FindAllStringSubmatch(string(raw), -1) {
		documented[m[1]] = true
	}

	var missing, extra []string
	for name := range served {
		if !documented[name] {
			missing = append(missing, name)
		}
	}
	for name := range documented {
		if !served[name] {
			extra = append(extra, name)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)

	if len(missing) > 0 {
		t.Errorf("served but not documented in docs/tools.md: %s", strings.Join(missing, ", "))
	}
	if len(extra) > 0 {
		t.Errorf("documented in docs/tools.md but not served: %s", strings.Join(extra, ", "))
	}
}
