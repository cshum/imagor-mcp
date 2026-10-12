# imagor-mcp

An MCP server that gives AI agents [imagor](https://github.com/cshum/imagor)'s
image pipeline as typed, validated tools.

**The value is validation, not string assembly.** An agent can already write an
imagor URL by hand and get it subtly wrong — argument order, an invented filter
name, a combination imagor rejects. imagor answers `200` with the image unchanged
when a filter does not apply, so a broken composition looks exactly like a working
one. This server is the strict layer: the agent calls tools with typed arguments,
and only paths that round-trip through imagor's own parser come out.

It does **not** upload, store, or render images. It constructs and signs
imagorpaths.

**Status: v0.3.0.** Ten tools, three signing modes, a filter catalogue generated
from imagor's own definitions, and a skill the server serves to the agent using it.

## Install

```sh
go install github.com/cshum/imagor-mcp@latest
```

Pure Go. Builds with `CGO_ENABLED=0` and needs neither libvips nor cgo, unlike
imagor itself, so `go install` works with no native toolchain.

The path grammar, generation, parsing and signing are **not reimplemented here** —
they come from imagor's own `imagorpath` package, so behaviour cannot drift from
the server.

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

### Preview performance

`verify_imagor_url` returns the rendered result, requested with imagor's
`preview()` filter, which skips result storage and opts into the in-memory cache.

**Enabling the cache on the imagor you point at makes repeated previews much
faster** — measured on a 1600×600 composition, **386 ms** for the first whole
preview and **25 ms** for a repeat:

```bash
VIPS_CACHE_SIZE=52428800      # byte budget; 0 disables the cache (default)
VIPS_CACHE_MAX_WIDTH=2400     # images larger than this are not cached
VIPS_CACHE_MAX_HEIGHT=2000
```

This matters most when a caller looks at the same composition repeatedly, which is
what iterating on a layout does. `preview()` never reaches the URL a caller is
given, so none of this changes what a composition means.

### Fonts

Which faces exist is a property of **the imagor you point at**, not of this
server. A name that is not installed **falls back silently and still returns
`200`**, so the composition succeeds and the type is not what was asked for.

The official `cshum/imagor-base` image carries DejaVu (reached by `sans`, `serif`,
`monospace`), Liberation2 (metric-compatible with Arial and Times) and Open Sans
(a common choice for social and OG cards). Add `-bold` for weight. For anything
else — CJK, commercial faces — mount your own fonts into the container.

## MCP client configuration

```json
{
  "mcpServers": {
    "imagor": {
      "command": "imagor-mcp",
      "env": {
        "IMAGOR_SECRET": "your-secret-here",
        "IMAGOR_SIGNER_TYPE": "sha256",
        "IMAGOR_SIGNER_TRUNCATE": "40"
      }
    }
  }
}
```

The server speaks MCP over stdio, so nothing listens on a port.

## Tools

| Tool | Purpose |
|---|---|
| `build_imagor_url` | Build a request path ready to send to imagor, signed or `unsafe/` as configured. |
| `build_imagor_path` | Build an **unsigned** path for the caller to sign. Use when the server should not hold the secret. |
| `validate_imagor_path` | Parse a path, report whether it round-trips, and summarize the transformation. |
| `verify_imagor_url` | Fetch the rendered result and report which filters applied. Returns the image. |
| `list_imagor_filters` | Accepted filters with signature, description and argument forms. |
| `describe_imagor` | The supported path grammar and the current signing mode. |
| `fit_text` | Find the font size at which a string fits a box, by measuring. |
| `render_template` | Render a layout at a canvas size, with values filled in. |
| `parse_template` | Read a path back into a layout, so one you have can be edited. |
| `get_imagor_skill` | The skill, for a client that reads it as a tool rather than a resource. |

Every parameter maps onto a concept in the imagor path grammar. There are no
invented parameters and no generic "sign this string" tool.

## Skill and cookbook

The server carries a **skill** and serves it as `imagor-mcp://skill`, and its
instructions point at it. It covers what a tool description cannot: the order a
path is written in, where text lands, how a label is centred, and a working method
that builds small pieces before assembling them. The same file is `SKILL.md` here.

`cookbook/` holds ready-made `render_template` payloads for common formats — OG
cards, product grids, quotes, announcements. Take the closest, change the values,
render it. They are rendered by the test suite at three canvas sizes, so a recipe
that no longer works fails CI rather than teaching a shape that fails.

## Templates, nesting and the cookbook

`render_template` takes a layout and a canvas size, so one layout serves 1200×630 and
1080×1080: geometry is relative to the canvas, and text with a height is **fitted** to
the box its geometry resolves to, which is what keeps type legible as the frame grows.
`parse_template` reads a path back into a layout, so one you already have can be edited
rather than retyped.

**A slot carrying its own `slots` is a canvas.** Its width and height become the canvas
the inner slots are laid out on, and the result draws at the outer slot's position. That
is how text is centred on something narrower than the frame — a label under a product, a
caption in a cell — because a numeric `x` leaves text hanging left and `x=center` only
resolves against the canvas it is on.

**A slot's `filters` field applies filters to that slot's own source**, written the way
imagor writes them: `smart/filters:round_corner(230,230,f4f1e8)`. The `image` field is
data rather than syntax, so a path written there is encoded as a source — this is where
it belongs. The signature is minted at render time, so a layout stays portable.

`cookbook/` holds twelve ready-made layouts for the formats that come up most — OG
cards, a product grid, a profile card, quotes, announcements, an event, a tip. Take the
closest, change the values, render it. They are rendered by the test suite at three
canvas sizes, so a layout that no longer works fails CI rather than teaching a shape
that fails.

## Security model

**A trusted client of imagor, not a new surface on it.** It holds the signing
secret for the same reason any backend service does: to mint valid URLs. Trust the
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

## Filter validation

imagor accepts any `name(args)` in a `filters:` segment. **Unknown names and wrong
argument counts produce no error** — the request returns `200` with the image
unchanged. Measured against imagor v1.9.7, all of these were byte-identical to a
filter-less request:

```
filters:vintage(70)      unknown name
filters:blur(1,2,3)      too many arguments
filters:blur()           missing a required argument
```

That is reasonable for a permissive server and dangerous for an agent, which
cannot tell a working URL from an inert one. Names and argument counts are checked
here before a path is signed, against a catalogue generated from the exact imagor
release in `go.mod`.

Refusal is reserved for what is certain. Too many arguments only warns, because
imagor's tolerance genuinely varies — `grayscale(1)` still applies while
`blur(1,2,3)` is ignored.

## Limitations

- **Argument values are not validated, only names and counts.** `quality(abc)` has
  the right arity and passes; imagor then ignores it.
- **Some filters accept fewer arguments than documented and still work.** On
  v1.9.7 `sharpen()`, `pixelate()`, `fill()` and `text(a,b)` take effect with
  fewer arguments than the reference states, so a refusal here is not proof imagor
  would have failed.
- **The catalogue is tied to one imagor release.** After upgrading, run
  `go generate ./internal/filters` and commit; CI fails if the file no longer
  matches.
- **No upload, no POST, no rendering.** External upload needs its own signed-grant
  mechanism and is deliberately out of scope.

## Development

```sh
go test ./...          # unit + protocol-level tests
go vet ./...
```

The suite spawns the real binary over stdio and includes signature values verified
against a running imagor instance and against imagor's published documentation, so
assertions are anchored to external ground truth rather than to this
implementation.

CI runs `gofmt -l`, `go vet`, `go test -race`, and a check that the generated
catalogue still matches the pinned imagor release.

## Licence

MIT. See [LICENSE](LICENSE).
