package tools

import (
	"testing"

	filtercat "github.com/cshum/imagor-mcp/internal/filters"
)

// A filter whose source travels in its own documented field must validate. The path builder folds
// Source and Text in, so validating f.Args alone counted one argument too few and refused the form
// the schema tells callers to prefer - "image takes 3 to 5" for a path that was correct.
func TestFilterArgsCountsTheFieldsTheBuilderEmits(t *testing.T) {
	const src = "https://example.com/a.png"

	source := FilterInput{Name: "image", Source: src, Args: "0,0"}
	if got := filterArgs(source); got != src+",0,0" {
		t.Errorf("filterArgs with Source = %q, want %q", got, src+",0,0")
	}
	if outcome, msg := filtercat.Check("image", filterArgs(source)); outcome == filtercat.Rejected {
		t.Errorf("a filter carrying its source in Source was refused: %s", msg)
	}

	// the control: the same arguments written positionally must reach the same verdict
	positional := FilterInput{Name: "image", Args: src + ",0,0"}
	if filterArgs(source) != filterArgs(positional) {
		t.Error("Source and the same value in Args must produce the same argument string")
	}

	// and it still refuses a genuine shortfall, so the fix did not blunt the check
	short := FilterInput{Name: "image", Args: "0,0"}
	if outcome, _ := filtercat.Check("image", filterArgs(short)); outcome != filtercat.Rejected {
		t.Error("image with only 2 arguments must still be refused")
	}
}
