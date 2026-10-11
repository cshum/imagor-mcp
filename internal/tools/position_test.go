package tools

import (
	"strings"
	"testing"

	"github.com/cshum/imagor/imagorpath"
)

// full and full-N are dimension forms. The position grammar has no such form and
// reads one as 0, so the overlay lands at the origin and the request still
// returns 200. Measured: y=full-220 on a 630px canvas put a title at the top.
func TestPositionWarningsCatchesDimensionForms(t *testing.T) {
	for _, tc := range []struct {
		args []string
		warn bool
	}{
		{[]string{"hi", "60", "410", "serif-40,ffffff"}, false},
		{[]string{"hi", "60", "full-220", "serif-40,ffffff"}, true},
		{[]string{"hi", "full", "40", "serif-40,ffffff"}, true},
		{[]string{"hi", "f-40", "40", "serif-40,ffffff"}, true},
		{[]string{"hi", "center", "bottom-40", "serif-40,ffffff"}, false}, // imagor's own forms
		{[]string{"hi", "20p", "0.5", "serif-40,ffffff"}, false},          // percentages
	} {
		p := imagorpath.Params{Image: "color:000000", Filters: imagorpath.Filters{
			{Name: "text", Args: strings.Join(tc.args, ",")},
		}}
		w := positionWarnings(p.Filters)
		if tc.warn && len(w) == 0 {
			t.Errorf("text(%s) should have warned", strings.Join(tc.args, ","))
		}
		if !tc.warn && len(w) > 0 {
			t.Errorf("text(%s) should not have warned, got %q", strings.Join(tc.args, ","), w[0])
		}
	}
}

// A nested path carries its own filters, and the same trap applies inside one.
func TestPositionWarningsWalksANestedPath(t *testing.T) {
	p := imagorpath.Params{Image: "color:000000", Filters: imagorpath.Filters{
		{Name: "image", Args: "600x600/filters:text(hi,10,full-20,serif-30,ffffff),60,60"},
	}}
	w := positionWarnings(p.Filters)
	if len(w) == 0 {
		t.Error("a position inside a nested path should have warned")
	}
}

// The legacy spellings are still filters, so they get the same warning.
func TestPositionWarningsCoversLegacyForms(t *testing.T) {
	for _, name := range []string{"label", "watermark", "image", "text"} {
		p := imagorpath.Params{Filters: imagorpath.Filters{
			{Name: name, Args: "60,full-20,serif-30,ffffff"},
		}}
		if len(positionWarnings(p.Filters)) == 0 {
			t.Errorf("%s() with a full-N position should have warned", name)
		}
	}
}
