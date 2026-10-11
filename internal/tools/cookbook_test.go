package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// cookbookFile is one recipe: a layout, its default values, and a note saying
// what it is for.
type cookbookFile struct {
	Name       string            `json:"name"`
	Note       string            `json:"note"`
	Canvas     json.RawMessage   `json:"canvas"`
	Background string            `json:"background"`
	Slots      []TemplateSlot    `json:"slots"`
	Values     map[string]string `json:"values"`
}

// The cookbook is only worth keeping if it is rendered. A recipe that no longer
// works is worse than no recipe: it teaches a shape that fails. So every recipe
// is rendered at several canvas sizes, and the result is checked the way a caller
// would check it - does it build, did the type get fitted, did anything land
// outside the frame.
func TestCookbookRendersAtEverySize(t *testing.T) {
	files, err := filepath.Glob("../../cookbook/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no cookbook recipes found: %v", err)
	}
	srv := stubImagor(t, 2)
	defer srv.Close()
	c := Config{Unsafe: true, BaseURL: srv.URL}

	sizes := [][2]int{{1200, 628}, {1080, 1080}, {1600, 900}}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		var recipe cookbookFile
		if err := json.Unmarshal(raw, &recipe); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if recipe.Name == "" || len(recipe.Slots) == 0 {
			t.Errorf("%s: needs a name and at least one slot", path)
			continue
		}
		cw, ch, ok := parseCanvas(recipe.Canvas)
		if !ok {
			t.Errorf("%s: needs a canvas, either [w,h] or {width,height}", path)
			continue
		}
		_ = cw
		_ = ch
		for _, size := range sizes {
			_, out, err := c.RenderTemplateTool(context.Background(), nil, TemplateInput{
				CanvasWidth: size[0], CanvasHeight: size[1],
				Background: recipe.Background, Slots: recipe.Slots, Values: recipe.Values,
			})
			if err != nil {
				t.Errorf("%s at %dx%d: %v", recipe.Name, size[0], size[1], err)
				continue
			}
			if out.Path == "" {
				t.Errorf("%s at %dx%d: no path", recipe.Name, size[0], size[1])
				continue
			}
			// Everything that resolved has to be inside the frame. A slot that
			// lands outside is the failure this whole loop keeps finding, and it
			// returns 200.
			for _, r := range out.Resolved {
				at, ok := positionOf(r.Detail)
				if !ok {
					continue
				}
				if at[0] < 0 || at[0] > size[0] || at[1] < 0 || at[1] > size[1] {
					t.Errorf("%s at %dx%d: %s resolves to %d,%d, outside the frame",
						recipe.Name, size[0], size[1], r.Name, at[0], at[1])
				}
			}
			// A text slot given a height must have been fitted, or the type is
			// the same size on every canvas and the recipe is not adaptive.
			for i, slot := range recipe.Slots {
				if slot.Kind != "text" || slot.Height == "" || strings.TrimSpace(substitute(slot.Text, recipe.Values)) == "" {
					continue
				}
				if !strings.Contains(out.Resolved[i].Detail, "fitted") {
					t.Errorf("%s at %dx%d: slot %q has a height but was not fitted",
						recipe.Name, size[0], size[1], slot.Name)
				}
			}
		}
	}
}

// parseCanvas accepts both the [w,h] and the {width,height} spelling, because
// a recipe written by hand and one written against a design tool differ.
func parseCanvas(raw json.RawMessage) (int, int, bool) {
	var pair []int
	if err := json.Unmarshal(raw, &pair); err == nil && len(pair) == 2 {
		return pair[0], pair[1], true
	}
	var obj struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Width > 0 && obj.Height > 0 {
		return obj.Width, obj.Height, true
	}
	return 0, 0, false
}

// positionOf reads the "at X,Y" out of a resolved detail.
func positionOf(detail string) ([2]int, bool) {
	i := strings.Index(detail, " at ")
	if i < 0 {
		return [2]int{}, false
	}
	rest := detail[i+4:]
	j := strings.IndexAny(rest, ", ")
	if j < 0 {
		return [2]int{}, false
	}
	x, err := strconv.Atoi(rest[:j])
	if err != nil {
		return [2]int{}, false
	}
	rest = strings.TrimPrefix(rest[j:], ",")
	j = strings.IndexAny(rest, ", ")
	if j < 0 {
		j = len(rest)
	}
	y, err := strconv.Atoi(strings.Fields(rest[:j])[0])
	if err != nil {
		return [2]int{}, false
	}
	return [2]int{x, y}, true
}
