# Working on imagor-mcp

Conventions for changing this repo. Only what cannot be worked out from the code
belongs here — if it is inferable, it is not written down.

## Running the tests

The suite needs a live imagor, and it must be **the published image**, not a binary
built for this machine:

```
docker run -d -p 8197:8000 -e IMAGOR_UNSAFE=1 ghcr.io/cshum/imagor:1.9.9
IMAGOR_BASE_URL=http://localhost:8197 IMAGOR_UNSAFE=1 go test ./...
```

The image is the one the recipes target and the only place the declared font families
exist. A macOS binary carries none of them and substitutes a system face silently, so
a test that renders text passes while proving nothing about the type.

`cookbook/render.py` renders every recipe and builds a contact sheet. Point it at the
same host:

```
IMAGOR_BASE_URL=http://localhost:8197 IMAGOR_MCP_BIN=./imagor-mcp \
  python3 cookbook/render.py /tmp/gallery
```

## What CI checks, and what it does not

CI runs `gofmt`, `go vet`, `go test -race`, and a drift guard on the generated filter
catalogue. It does **not** look at a rendered image.

```
go generate ./internal/filters   # after a dependency bump; commit the result
```

The catalogue is generated from the imagor release pinned in `go.mod`. A stale one
accepts filters imagor ignores, or rejects ones that exist.

**A green suite does not mean the output is right.** Every fault recorded in
`docs/build-log.md` passed every check that existed at the time. Rendering the sheet
and looking at it is the only test that has ever caught a layout fault, a substituted
font, or a caption sitting on a picture.

## Changing the template schema

A field belongs on **both** `TemplateSlot` and `InnerSlot`. They are separate types
because the schema generator rejects a type that contains itself, so an omission on
`InnerSlot` is not a compile error — it is `unexpected additional properties` at
runtime, which a delegated run will silently work around and mention in its closing
summary.

**`parse_template` is the inverse of `render_template` and has to stay so.** A change
to what `render_template` writes is a change to what `parse_template` must read, in the
same commit. The round trip is not byte-identical by design — a path holds the pixels a
box resolved to, not the form it was written in — but every field that *can* survive
must, and `roundtrip_test.go` is where that is asserted.

## Writing a test for a round trip, or any transformation

**Assert the fields, not the artifact.** A test that compares a path with a path passes
while the parts are lost. A test that names each field fails with the name of the one
that broke, which is the difference between two attempts that "made progress" and one
that lands.

This is not a style preference. Two reverted attempts at the nested round trip failed
the same three tests because the only instrument was a path comparison, and a path that
looks plausible is exactly what a lossy parser produces.

## The nested canvas

A slot carrying its own `slots` is a canvas, and its outer source is a **transparent
canvas** (`color:none`) that composites over what is beneath it. The panel ground a
recipe wants is a child, not the slot's own `image`. Getting this wrong renders a
plausible card with the picture missing.

Its children are laid out relative to that canvas, which is what makes `x: center`
mean something. A numeric x leaves text hanging left of its box whatever `align` says.

## Sources, photographs and trademarks

The cookbook is committed and its contact sheet is the README's hero image, so
everything in a recipe is **published**, not just rendered locally.

**Photographs are from Unsplash**, whose licence permits commercial use,
modification and distribution with no attribution. That covers committing them and
showing them. It does **not** permit compiling them to replicate a stock-image
service, which this is not.

**Names, brands and products are invented.** `SABLON`, `Mei Lam`, `Jon Ho`,
`Harbour`, every price. Keep it that way. A real brand term is not a neutral
placeholder — it is a trademark use, and in a project whose whole subject is
plausible-looking labels that are not what they claim to be, it is also the wrong
joke. `Intrecciato` was written into a recipe once and had to be removed.

When adding a recipe: invent the brand, invent the people, and if a caption needs a
material, describe it generically — *hand-woven*, *vegetable-tanned*, *grained*.

## Filters on a slot's own source

An `image` field is data, not syntax, so a value that looks like a path is encoded and
cannot carry filters. A filter on the slot's own source lives in `filters`, written
without the `filters:` prefix — `smart`, or `smart/filters:round_corner(230,230,fff)`,
which is one segment with an internal slash.
