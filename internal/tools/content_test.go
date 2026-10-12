package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The cookbook's brands, people and copy come from three invented worlds, recorded in
// content/worlds.json. That mapping drifted once already: three worlds named recipes that
// were never written, and five recipes belonged to no world at all. Nothing noticed,
// because nothing checked.
//
// This is the check. A recipe with no world is a card whose voice nobody decided, and a
// world naming a recipe that does not exist is documentation that has fallen behind.
func TestEveryCookbookRecipeBelongsToAWorld(t *testing.T) {
	recipes, err := filepath.Glob("../../cookbook/*.json")
	if err != nil {
		t.Fatalf("glob the cookbook: %v", err)
	}
	if len(recipes) == 0 {
		t.Fatal("no recipes found; the cookbook is the point of this test")
	}

	raw, err := os.ReadFile("../../content/worlds.json")
	if err != nil {
		t.Fatalf("read the content system: %v", err)
	}
	var content struct {
		Worlds map[string]struct {
			Recipes []string `json:"recipes"`
		} `json:"worlds"`
	}
	if err := json.Unmarshal(raw, &content); err != nil {
		t.Fatalf("parse the content system: %v", err)
	}

	assigned := map[string]string{}
	for world, w := range content.Worlds {
		for _, r := range w.Recipes {
			if prev, dup := assigned[r]; dup {
				t.Errorf("%q is in two worlds: %s and %s", r, prev, world)
			}
			assigned[r] = world
		}
	}

	for _, path := range recipes {
		name := trimExt(filepath.Base(path))
		if _, ok := assigned[name]; !ok {
			t.Errorf("cookbook/%s.json belongs to no world; add it to content/worlds.json", name)
		}
	}
	// And the other direction, so the mapping cannot describe recipes that are gone.
	present := map[string]bool{}
	for _, path := range recipes {
		present[trimExt(filepath.Base(path))] = true
	}
	for name, world := range assigned {
		if !present[name] {
			t.Errorf("content/worlds.json lists %q under %s, but there is no cookbook/%s.json",
				name, world, name)
		}
	}
}

func trimExt(name string) string {
	return name[:len(name)-len(filepath.Ext(name))]
}
