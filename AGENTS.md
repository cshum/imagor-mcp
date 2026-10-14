# Working on imagor-template

Conventions for changing this repo. Only what cannot be worked out from the code
belongs here — if it is inferable, it is not written down.

## Running the tests

The suite needs a live imagor, and it must be **the published image**, not a binary
built for this machine:

```
docker run -d -p 8197:8000 -e IMAGOR_UNSAFE=1 ghcr.io/cshum/imagor:1.9.10
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
python3 cookbook/index.py     # regenerate references/cookbook.md
```

**`cookbook/index.py` generates `references/cookbook.md`, which is `go:embed`ed into the
binary.** The server serves it at `imagor-mcp://cookbook` and returns it from
`get_cookbook_recipe` with no name. Regenerate it whenever a recipe is added, renamed, or
given a new value, or the list the agent chooses from goes stale — CI does this beside the
gallery so a change cannot land without it.

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

## A filter that should not be offered

`superseded` in `internal/tools/tools.go` hides a filter from `list_imagor_filters`
while leaving it in the catalogue. **Hiding is not deleting.** `filtercat.Check` is
what `build_imagor_url` refuses on and what `validate_imagor_path` reports, so a
filter removed from the catalogue would stop a path that already carries it from
building or validating. Regenerating the catalogue is safe — `superseded` is not
generated.

Two things justify hiding one:

- **Its signature is not readable**, so a caller that sees it listed can only guess at
  the arguments. `label()` is undocumented and `text()` covers everything it does.
- **It exists for imagor's own machinery rather than for composing.** `focal` carries
  a detection box between processors for the face-detection plugin, and
  `draw_detections` draws it back. `focal` is also left out of the report in `/meta`,
  so a path using it could not be verified even if it were offered.

`TestSupersededFiltersAreAbsentFromTheListing` asserts on the listing itself rather
than on the predicate, because a filter hidden in one and not the other is the failure.

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

## The cookbook is a channel, not a document

**A recipe is what gets handed to `render_template`.** It travels in the binary and
arrives by name through `get_cookbook_recipe`, and the cookbook page deliberately does not
carry it — the page has the card, what it is for, and the URL it resolves to, because that
is what a reader wants.

Two things will look like improvements and are not:

- **Putting the template back on the page.** It was there. It made a channel into a
  document, and 71% of the page was JSON nobody was meant to read.
- **Serving the index only as a resource.** It is served as `imagor-mcp://cookbook` *and*
  returned by the tool with no name, because a client that surfaces only tools never shows
  a resource. A delegated run reported exactly that — it chose correctly anyway, but
  without the list it was meant to choose from.

The page gives each card a sentence to paste instead of a tool call to understand. A
reader knows how to ask their agent for something; they do not know what an MCP tool is.

## Two front ends, one engine — keep them in step

`cmd/imagor-template` and `cmd/imagor-mcp` are bindings on the same code in
`internal/tools`.

**The CLI is a runner, not an authoring path.** A template has to be written, and writing
one is what the agent does — so the CLI cannot stand in for the server. It compiles a
template you already have, in a place with no agent in it: a stored layout, a build step,
a script checking the cookbook. Describing it as an alternative to the agent was wrong and
is worth not repeating.

**`cmd/imagor-template/main_test.go` pins this**: `TestCompileMatchesTheServer` compiles
every recipe through both front ends and fails if they disagree. If a change to the
engine makes the two diverge, that test names the recipe. Add a recipe and it is covered
without touching the test.

**Compiling is deterministic given an imagor, not pure.** A text slot with a height is
fitted by measuring, and measuring renders a probe through imagor, so `IMAGOR_BASE_URL`
has to resolve or every recipe with a fitted slot fails. **The CLI tests skip rather
than fail without it** — a skip is the honest outcome, and CI provides a base URL.

**The cookbook carries three canvas shapes** (`[w, h]`, `{width, height, background}`
and `{width, height}` with the background alongside). `templateFile.canvas()` accepts
all three so a recipe cannot become unusable through a front end that handles one. They
should be normalised to one shape, but until then this is the compatibility layer.

## The `.mcpb` bundle: resolve paths before changing directory

`mcpb/build.sh` runs `cd "$work"` to pack, and `dest` used to be resolved after that — so
the relative `dist` CI passes became `$work/dist`, and the `EXIT` trap deleted the bundle
on the way out. **Every release before v0.4.0 listed six binaries and no bundle**, while
the build log printed `dist/imagor-mcp-v0.3.0.mcpb 7.1M` and a checksum for it, so the
step looked like it had worked.

Anything added to that script must resolve its paths **before** the `cd`. Reproduce with a
*relative* output directory — an absolute one hides the bug, which is why it survived.

## `sources/` deploys from this repo, and takes its time about it

`s.imagor.net` is the **`imagor-mcp-sources`** Cloudflare Pages project, connected to this
repository with `sources/` as its root and `main` as its production branch. **A push
deploys it — there is no `wrangler` command to run for a normal change.**

Two things make that hard to confirm if you try:

- **It is slow, and the deployments list is misleading.** Most entries read `Skipped`,
  and a `Skipped` entry sits alongside a `Production` one for the same commit. A check
  made a few minutes after a push will 404 and look like a broken connection. **Wait,
  and check again before concluding anything** — a probe file that 404s at +8 minutes
  may be live at +20.
- **`wrangler whoami` says the OAuth token expired and cannot be refreshed**, which
  reads as "not authenticated" and sends you looking for an API token. The refresh
  token has a lifetime of its own, so a long-idle machine genuinely is logged out —
  `npx wrangler login` fixes it, and it needs a browser.

Useful commands once authenticated:

```
npx wrangler pages project list
npx wrangler pages deployment list --project-name imagor-mcp-sources
```

**The edge caches hard.** A `.svg` is served `max-age=31536000, immutable`, so adding or
changing a file is immediate while removing one is not — a deleted fixture keeps
answering for a year. Treat a rename as an add plus a permanent tombstone.

## `IMAGOR_BASE_URL` has to be reachable from inside imagor

The preview nests the absolute URL as a source, so imagor fetches the composed
image back **from inside its own container** to scale it. A base URL that only the
MCP process can reach therefore verifies fine and previews nothing:

```
IMAGOR_BASE_URL=http://localhost:8197   ->  preview 500, dial tcp [::1]:8197 refused
IMAGOR_BASE_URL=http://127.0.0.1:8197   ->  preview 500, the container's own loopback
```

Both are the container talking to itself. Use a host that resolves on both sides —
a public origin in production, `host.docker.internal` or the Docker host's LAN
address locally. The failure arrives as a note (*"no preview: the image request
returned 500"*) rather than an error, so a report can be perfect while the picture
never comes back.

## Filters on a slot's own source

An `image` field is data, not syntax, so a value that looks like a path is encoded and
cannot carry filters. A filter on the slot's own source lives in `filters`, written
without the `filters:` prefix — `smart`, or `smart/filters:round_corner(230,230,fff)`,
which is one segment with an internal slash.
