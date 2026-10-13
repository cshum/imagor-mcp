package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// No text may run into a picture.
//
// The comparison is against the picture's own pixels, not its slot: a fitted image
// centres in its box and fills only part of it, so a caption may overlap the slot and
// still clear the picture, and a portrait source fills the full height and leaves the
// caption no margin at all. The source's aspect gives the ink, and the aspect is known
// offline - the fixtures are in the manifest.
//
// Found by hand once: a caption written for one row's rhythm while the row's images moved
// down independently. Nothing else would have caught it.
func TestNoTextRunsIntoAPicture(t *testing.T) {
	files, err := filepath.Glob("../../cookbook/*.json")
	if err != nil {
		t.Fatalf("glob the cookbook: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no recipes found; the cookbook is the point of this test")
	}

	checked, faults := 0, 0
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var recipe struct {
			Canvas json.RawMessage   `json:"canvas"`
			Width  int               `json:"canvas_width"`
			Height int               `json:"canvas_height"`
			Slots  []json.RawMessage `json:"slots"`
		}
		if err := json.Unmarshal(raw, &recipe); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		w, h := recipe.Width, recipe.Height
		if len(recipe.Canvas) > 0 {
			var pair []int
			var obj struct{ Width, Height int }
			if json.Unmarshal(recipe.Canvas, &pair) == nil && len(pair) == 2 {
				w, h = pair[0], pair[1]
			} else if json.Unmarshal(recipe.Canvas, &obj) == nil {
				w, h = obj.Width, obj.Height
			}
		}
		if w == 0 {
			w, h = 1200, 630
		}

		name := strings.TrimSuffix(filepath.Base(path), ".json")
		var texts, subjects []namedBox
		collectSlots(recipe.Slots, 0, 0, w, h, &texts, &subjects)
		checked += len(texts)

		for _, tb := range texts {
			if tb.empty() {
				continue // no explicit box: the ink goes where it goes
			}
			for _, sb := range subjects {
				sb = inkBox(sb, aspectOf(sb.image))
				if tb.runsInto(sb) {
					t.Errorf("%s: text %q runs into %q (overlap %.0fx%.0fpx)",
						name, tb.name, sb.name,
						min(tb.x2, sb.x2)-max(tb.x1, sb.x1),
						min(tb.y2, sb.y2)-max(tb.y1, sb.y1))
					faults++
				}
			}
		}
	}
	t.Logf("%d text slots checked, %d faults", checked, faults)
}

type namedBox struct {
	name           string
	image          string
	x1, y1, x2, y2 float64
}

func (b namedBox) empty() bool { return b.x2 == b.x1 || b.y2 == b.y1 }

func (b namedBox) runsInto(o namedBox) bool {
	return !(b.x2 <= o.x1 || o.x2 <= b.x1 || b.y2 <= o.y1 || o.y2 <= b.y1)
}

// ink narrows a picture's box to where its pixels actually land. fit-in centres the
// picture, so a source whose aspect differs from the box leaves a margin on two sides.
func inkBox(b namedBox, aspect float64) namedBox {
	bw, bh := b.x2-b.x1, b.y2-b.y1
	if aspect <= 0 || bw <= 0 || bh <= 0 {
		return b
	}
	if bw/bh > aspect {
		iw := bh * aspect
		return namedBox{b.name, b.image, b.x1 + (bw-iw)/2, b.y1, b.x1 + (bw+iw)/2, b.y2}
	}
	ih := bw / aspect
	return namedBox{b.name, b.image, b.x1, b.y1 + (bh-ih)/2, b.x2, b.y1 + (bh+ih)/2}
}

// collectSlots flattens a recipe, translating nested canvases. cw and ch are the canvas
// these slots lay out on - the recipe's at the top level, and the carrying slot's own
// box for anything nested. Resolving an inner slot against the outer canvas makes every
// percentage inside it wrong by the scale factor.
func collectSlots(slots []json.RawMessage, ox, oy float64, cw, ch int,
	texts, subjects *[]namedBox) {
	for _, raw := range slots {
		var s struct {
			Name   string            `json:"name"`
			Kind   string            `json:"kind"`
			Text   string            `json:"text"`
			Image  string            `json:"image"`
			X      string            `json:"x"`
			Y      string            `json:"y"`
			Width  string            `json:"width"`
			Height string            `json:"height"`
			Slots  []json.RawMessage `json:"slots"`
		}
		if json.Unmarshal(raw, &s) != nil {
			continue
		}
		x := resolveDim(s.X, cw)
		y := resolveDim(s.Y, ch)
		x2 := x + resolveDim(s.Width, cw)
		y2 := y + resolveDim(s.Height, ch)
		b := namedBox{s.Name, s.Image, x + ox, y + oy, x2 + ox, y2 + oy}

		if s.Text != "" {
			*texts = append(*texts, b)
		}
		// A full-canvas image is a background and a color: value is a panel; neither is a
		// picture. What is left is a source a fit will letterbox, which is the only thing
		// a caption can collide with.
		if s.Image != "" && (s.Kind == "" || s.Kind == "image") &&
			!strings.HasPrefix(s.Image, "color:") {
			full := resolveDim(s.Width, cw) == float64(cw) && resolveDim(s.Height, ch) == float64(ch)
			if !full {
				*subjects = append(*subjects, namedBox{s.Name, s.Image, b.x1, b.y1, b.x2, b.y2})
			}
		}
		if len(s.Slots) > 0 {
			iw := int(resolveDim(s.Width, cw))
			if iw == 0 {
				iw = cw
			}
			ih := int(resolveDim(s.Height, ch))
			if ih == 0 {
				ih = ch
			}
			collectSlots(s.Slots, x+ox, y+oy, iw, ih, texts, subjects)
		}
	}
}

// resolveDim reads the template's geometry forms for comparison. It is deliberately not
// the renderer: this test compares boxes, so it only needs them to be the right size.
func resolveDim(v string, total int) float64 {
	if v == "" {
		return 0
	}
	switch v {
	case "full", "f":
		return float64(total)
	case "half", "h":
		return float64(total) / 2
	case "third", "t":
		return float64(total) / 3
	case "quarter", "q":
		return float64(total) / 4
	}
	if strings.HasPrefix(v, "full-") {
		if n, err := strconv.Atoi(v[5:]); err == nil {
			return float64(total - n)
		}
	}
	if i := strings.Index(v, "p+"); i > 0 {
		p, err1 := strconv.Atoi(v[:i])
		off, err2 := strconv.Atoi(v[i+2:])
		if err1 == nil && err2 == nil {
			return float64(p)/100*float64(total) + float64(off)
		}
	}
	if strings.HasSuffix(v, "p") {
		if p, err := strconv.Atoi(strings.TrimSuffix(v, "p")); err == nil {
			return float64(p) / 100 * float64(total)
		}
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}
	return 0
}

// aspectOf reads a fixture's aspect from the committed manifest. A source that is not a
// fixture has no offline aspect, and the check falls back to the whole box - the
// conservative reading, which reports more rather than fewer collisions.
func aspectOf(url string) float64 {
	raw, err := os.ReadFile("../../sources/manifest.json")
	if err != nil {
		return 0
	}
	var entries []struct {
		URL    string  `json:"url"`
		Aspect float64 `json:"aspect"`
	}
	if json.Unmarshal(raw, &entries) != nil {
		return 0
	}
	for _, e := range entries {
		if e.URL == url {
			return e.Aspect
		}
	}
	return 0
}
