// Command spike proves the imagor-mcp foundation:
// typed construction -> path generation -> signing -> round-trip validation.
package main

import (
	"crypto/sha256"
	"fmt"
	"reflect"

	"github.com/cshum/imagor/imagorpath"
)

func main() {
	// 1. Typed construction — what an MCP tool call would produce.
	p := imagorpath.Params{
		Image:  "raw.githubusercontent.com/cshum/imagor/master/testdata/gopher.png",
		Width:  500,
		Height: 500,
		VAlign: imagorpath.VAlignTop,
		Filters: imagorpath.Filters{
			{Name: "fill", Args: "white"},
			{Name: "format", Args: "jpeg"},
			{Name: "quality", Args: "80"},
		},
	}

	path := imagorpath.GeneratePath(p)
	fmt.Println("path-out (unsigned) :", path)

	// 2. URL-out — signed, ready to fetch.
	signer := imagorpath.NewHMACSigner(sha256.New, 40, "mysecret")
	fmt.Println("url-out  (signed)   :", imagorpath.Generate(p, signer))

	// 3. Unsafe mode (dev only).
	fmt.Println("unsafe              :", imagorpath.GenerateUnsafe(p))

	// 4. Round-trip validation — the core safety property.
	back := imagorpath.Parse(path)
	fmt.Println()
	fmt.Println("parsed image        :", back.Image)
	fmt.Printf("parsed w x h        : %d x %d (valign=%q)\n", back.Width, back.Height, back.VAlign)
	fmt.Println("parsed filters      :", back.Filters)
	fmt.Println("round-trip equal    :",
		p.Image == back.Image &&
			p.Width == back.Width &&
			p.Height == back.Height &&
			p.VAlign == back.VAlign &&
			reflect.DeepEqual(p.Filters, back.Filters))

	// 5. Determinism — same input must regenerate identical bytes.
	regen := imagorpath.GeneratePath(back)
	fmt.Println("regenerated         :", regen)
	fmt.Println("deterministic       :", regen == path)
}
