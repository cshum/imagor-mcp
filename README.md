# imagor-mcp

**Give an AI agent an image pipeline it can actually see.** An MCP server that
exposes [imagor](https://github.com/cshum/imagor)'s URL language as typed tools —
and hands the rendered result back, so the agent can look at what it built and
correct it.

![Fourteen cards, each one a single imagor URL](docs/gallery/contact-sheet.png)

*Every card above is one imagor URL. No design file, no template editor, no image
library — composed through these tools, and rendered on every commit by the same
code path the tests exercise.*

## What that means

The products this resembles are **editor-first**: a human designs a template in a
drag-and-drop canvas and an API fills the slots. The canvas is the product.

This is **URL-first**. There is no canvas, no template document, no library. A
layout *is* a URL, and a URL is something an agent can write, check and rewrite.
Given a brief with no layout, no positions and no sizes, an agent designed three
distinct banners across three runs — hierarchy, colour, type pairing, invented
copy — and zoomed into regions to read the small text it had drawn.

**An agent can author a layout, not just fill one in.**

## Why a strict layer is needed

imagor is permissive, deliberately: an unknown filter name or a wrong argument
count returns **`200` with the image unchanged**. Reasonable for a server, and
dangerous for an agent, which cannot tell a working URL from an inert one. Measured
against imagor v1.9.7, all three of these were byte-identical to a filter-less
request:

```
filters:vintage(70)      unknown name
filters:blur(1,2,3)      too many arguments
filters:blur()           missing a required argument
```

This server is the strict layer. Names and argument counts are checked **before a
path is signed**, against a catalogue generated from the exact imagor release in
`go.mod`. And when a filter the server recognises turns out to have left the image
unchanged, `verify_imagor_url` **fails rather than returning success** — because a
`200` that did nothing is the failure this whole tool exists to catch.

## Install

```sh
go install github.com/cshum/imagor-mcp@latest
```

Pure Go. Builds with `CGO_ENABLED=0` and needs neither libvips nor cgo, unlike
imagor itself, so `go install` works with no native toolchain.

The path grammar, generation, parsing and signing are **not reimplemented here** —
they come from imagor's own `imagorpath` package, so behaviour cannot drift from
the server.

## Quick start

**1. Run an imagor.** The fonts are the reason to use the official image:

```sh
docker run -d -p 8000:8000 \
  -e IMAGOR_SECRET=your-secret-here \
  -e VIPS_CACHE_SIZE=52428800 \
  cshum/imagor-base
```

**2. Point the server at it.**

```json
{
  "mcpServers": {
    "imagor": {
      "command": "imagor-mcp",
      "env": {
        "IMAGOR_SECRET": "your-secret-here",
        "IMAGOR_SIGNER_TYPE": "sha256",
        "IMAGOR_SIGNER_TRUNCATE": "40",
        "IMAGOR_BASE_URL": "http://localhost:8000"
      }
    }
  }
}
```

**3. Ask for something.** The server speaks MCP over stdio, so nothing listens on
a port.

> *Build a 1200×630 card: a dark ground, a serif headline, and a product photo on
> the right. Show me what it looks like.*

The agent calls `render_template`, gets a signed URL back, calls
`verify_imagor_url` to fetch the result, **looks at it**, and revises. That loop is
the product.

### Turn the cache on

`verify_imagor_url` fetches through imagor's `preview()` filter, which skips result
storage and opts into the in-memory cache. Measured on a 1600×600 composition,
**386 ms** for the first whole preview and **25 ms** for a repeat:

```bash
VIPS_CACHE_SIZE=52428800      # byte budget; 0 disables the cache (default)
VIPS_CACHE_MAX_WIDTH=2400     # images larger than this are not cached
VIPS_CACHE_MAX_HEIGHT=2000
```

This matters most when a caller looks at the same composition repeatedly, which is
exactly what iterating on a layout does. `preview()` never reaches the URL a caller
is given, so none of this changes what a composition means.

### Fonts

Which faces exist is a property of **the imagor you point at**, not of this server.
A name that is not installed **falls back silently and still returns `200`**, so the
composition succeeds and the type is not what was asked for.

`cshum/imagor-base` carries **DejaVu** (reached by `sans`, `serif`, `monospace`),
**Liberation2** (metric-compatible with Arial and Times) and **Open Sans** — a
common choice for social and OG cards. Add `-bold` for weight. For anything else —
CJK, commercial faces — mount your own fonts into the container.

## Tools

| Tool | Purpose |
|---|---|
| `build_imagor_url` | Build a request path ready to send to imagor, signed or `unsafe/` as configured. |
| `build_imagor_path` | Build an **unsigned** path for the caller to sign. Use when the server should not hold the secret. |
| `validate_imagor_path` | Parse a path, report whether it round-trips, and summarize the transformation. |
| `verify_imagor_url` | Fetch the rendered result and report which filters applied. **Returns the image.** |
| `list_imagor_filters` | Accepted filters with signature, description and argument forms. |
| `describe_imagor` | The supported path grammar and the current signing mode. |
| `fit_text` | Find the font size at which a string fits a box, by measuring. |
| `render_template` | Render a layout at a canvas size, with values filled in. |
| `parse_template` | Read a path back into a layout, so one you have can be edited. |
| `get_imagor_skill` | The skill, for a client that reads it as a tool rather than a resource. |

Every parameter maps onto a concept in the imagor path grammar. There are no
invented parameters and no generic "sign this string" tool.

## Templates

`render_template` takes a layout and a canvas size, so **one layout serves 1200×630
and 1080×1080**: geometry is relative to the canvas, and text with a height is
**fitted** to the box its geometry resolves to, which keeps type legible as the
frame grows.

**A slot carrying its own `slots` is a canvas.** Its width and height become the
canvas the inner slots are laid out on, and the result draws at the outer slot's
position. That is how text is centred on something narrower than the frame — a
label under a product, a caption in a cell — because a numeric `x` leaves text
hanging left and `x=center` only resolves against the canvas it is on.

**A slot's `filters` field applies filters to that slot's own source**, written the
way imagor writes them: `smart/filters:round_corner(230,230,f4f1e8)`. The `image`
field is data rather than syntax, so a path written there is encoded as a source —
this is where it belongs. The signature is minted at render time, so a layout stays
portable.

`parse_template` reads a path back into a layout, so one you already have can be
edited rather than retyped. It is the inverse of `render_template` and stays so: a
nested layout comes back as itself. The one thing a path cannot carry is a *form* —
it holds the pixels a box resolved to, not whether it was written `38p` or `half`.

## Cookbook

`cookbook/` holds **fourteen ready-made layouts** for the formats that come up most
— OG cards, a product grid, a profile card, quotes, testimonials, an event, a
comparison table, a price ladder. Take the closest, change the values, render it.

Every recipe is rendered by the test suite **at three canvas sizes on every
commit**, so a layout that no longer works fails CI rather than teaching a shape
that fails. **The contact sheet at the top of this page is generated from them by
the same code path**, so it cannot drift from what the recipes actually produce.

## The skill

The server carries a **skill** and serves it as `imagor-mcp://skill`, and its
instructions point at it. It covers what a tool description cannot: the order a path
is written in, where text lands, how a label is centred, and a working method that
builds small pieces before assembling them. The same file is `SKILL.md` here.

## Modes

Exactly one applies, selected by the environment. There is no implicit fallback.

| Mode | Selected by | Output |
|---|---|---|
| **signed** | `IMAGOR_SECRET` set | `<signature>/<path>` — production |
| **unsafe** | `IMAGOR_UNSAFE=1` | `unsafe/<path>` — development only |

**Setting both is refused at startup, and so is setting neither.** They state
opposite intentions, and silently picking one — or running unsigned — is how an
unsigned deployment reaches production.

## Configuration

| Variable | Meaning |
|---|---|
| `IMAGOR_SECRET` | Signing secret. Selects **signed** mode. |
| `IMAGOR_UNSAFE` | `1`, `true`, `yes` or `on`. Selects **unsafe** mode. Must be explicit. |
| `IMAGOR_SIGNER_TYPE` | `sha1` (default), `sha256` or `sha512`. |
| `IMAGOR_SIGNER_TRUNCATE` | Signature length in characters. `0` means none. |
| `IMAGOR_BASE_URL` | Optional imagor origin. When set, output includes an `absolute_url`. |
| `IMAGOR_MCP_EXTRA_FILTERS` | Extra filter names for forks that register their own. Bypass name checking and are not arity-validated. |

These must match the target imagor exactly, or its signature verification rejects
the URLs.

## Security model

**A trusted client of imagor, not a new surface on it.** It holds the signing secret
for the same reason any backend service does: to mint valid URLs. Trust the
component; do not trust the agent using it.

- **A mode is required.** It signs or emits `unsafe/`; it does not run as a
  pass-through holding no credential.
- **Unsafe mode is opt-in and loud** — it warns on stderr, and `describe_imagor`
  reports it at runtime.
- **stdio only.** Nothing listens on the network.
- **No generic signing tool.** It can only emit paths its own typed tools built.
- **Blast radius is set by imagor's own configuration** —
  `HTTP_LOADER_ALLOWED_SOURCES`, `HTTP_LOADER_BASE_URL`,
  `VIPS_MAX_RESOLUTION`/`_WIDTH`/`_HEIGHT`, `IMAGOR_REQUEST_TIMEOUT`,
  `IMAGOR_PROCESS_CONCURRENCY`. A signed URL from an errant agent is still subject
  to every one of those limits.

Do not host this multi-tenant on behalf of users whose keys you would then hold.
Ship it self-hosted alongside imagor.

## What is not validated

Being precise about this is the point of the tool, so it is worth being precise
about its own edges.

- **Argument values are not validated, only names and counts.** `quality(abc)` has
  the right arity and passes; imagor then ignores it.
- **Some filters accept fewer arguments than documented and still work.** On v1.9.7
  `sharpen()`, `pixelate()`, `fill()` and `text(a,b)` take effect with fewer
  arguments than the reference states, so a refusal here is not proof imagor would
  have failed.
- **Too many arguments warns rather than refuses**, because imagor's tolerance
  genuinely varies — `grayscale(1)` still applies while `blur(1,2,3)` is ignored.
- **The catalogue is tied to one imagor release.** After upgrading, run
  `go generate ./internal/filters` and commit; CI fails if the file no longer
  matches.
- **A declined filter fails `verify_imagor_url`, and that is deliberate.** If a
  filter is genuinely conditional, pass `allow_declined` to get the report instead
  of an error.

## Development

```sh
docker run -d -p 8197:8000 -e IMAGOR_UNSAFE=1 ghcr.io/cshum/imagor:1.9.9
IMAGOR_BASE_URL=http://localhost:8197 IMAGOR_UNSAFE=1 go test ./...
go vet ./...
```

The suite needs that container: **it is the only place the declared font families
exist**, and a host binary substitutes a system face silently while the tests pass.
See [AGENTS.md](AGENTS.md) for the conventions and
[docs/build-log.md](docs/build-log.md) for the faults that shaped them.

CI runs `gofmt -l`, `go vet`, `go test -race`, a check that the generated catalogue
still matches the pinned imagor release, and a job that re-renders the gallery.

## Licence

MIT. See [LICENSE](LICENSE).
