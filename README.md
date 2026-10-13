# imagor-mcp

**Give an AI agent an image pipeline it can actually see.** An MCP server that
exposes [imagor](https://github.com/cshum/imagor)'s URL language as typed tools —
and hands the rendered result back, so the agent can look at what it built and
correct it.

![Fourteen cards, each one a single imagor URL](docs/gallery/contact-sheet.png)

*Every card above is one imagor URL. No design file, no template editor, no image
library — composed by an agent through these tools.*

## What you get

Ask for a card and the agent builds it:

> *Build a 1200×630 launch card: a deep green ground, a serif headline, a product
> photo on the right, and the price below it. Show me what it looks like.*

It calls `render_template`, gets a signed URL back, calls `verify_imagor_url` to
fetch the render, **looks at it**, and revises. That loop — write, render, look,
correct — is the whole point. An agent that can see its own output is an agent you
can hand a blank brief to.

That is not how image tooling usually works. The products this resembles are
**editor-first**: someone designs a template in a drag-and-drop canvas and an API
fills the slots. The canvas is the product. This is **URL-first** — no canvas, no
template document, no library. A layout *is* a URL, and a URL is something an agent
can write, check and rewrite.

**Given a brief with no layout, no positions and no sizes, an agent designed three
distinct banners across three runs** — hierarchy, colour, type pairing, invented
copy — and zoomed into regions to read the small text it had drawn.

## Quick start

**1. You need an imagor to render with.** The official image is the one the
cookbook targets:

```sh
docker run -d -p 8000:8000 \
  -e IMAGOR_SECRET=your-secret-here \
  cshum/imagor-base
```

Already running imagor? Skip this and point the config below at it.

**2. Install the server.** A prebuilt binary, or build from source:

```sh
# pick your platform from the releases page
curl -L -o imagor-mcp https://github.com/cshum/imagor-mcp/releases/latest/download/imagor-mcp_v0.3.0_darwin_arm64
chmod +x imagor-mcp

# or, with Go installed
go install github.com/cshum/imagor-mcp@latest
```

Pure Go, with no libvips and no cgo — unlike imagor itself. Every platform is on the
[releases page](https://github.com/cshum/imagor-mcp/releases).

**3. Add it to your MCP client.**

```json
{
  "mcpServers": {
    "imagor": {
      "command": "imagor-mcp",
      "env": {
        "IMAGOR_SECRET": "your-secret-here",
        "IMAGOR_BASE_URL": "http://localhost:8000"
      }
    }
  }
}
```

That is the whole setup. **`IMAGOR_SECRET` must match the imagor you point at**, or
its signature check rejects what this mints. The server speaks MCP over stdio, so
nothing listens on a port.

**4. Ask it for something.** Start from one of the fourteen recipes in `cookbook/`,
or describe what you want and let it compose.

### Prefer not to hand over the secret?

`build_imagor_path` returns an **unsigned** path and leaves signing to you — for a
service that constructs compositions but should never hold the key.

## Tools

| Tool | Purpose |
|---|---|
| `render_template` | Render a layout at a canvas size, with values filled in. **Start here.** |
| `verify_imagor_url` | Fetch the rendered result and report which filters applied. **Returns the image.** |
| `parse_template` | Read a path back into a layout, so one you have can be edited. |
| `fit_text` | Find the font size at which a string fits a box, by measuring. |
| `build_imagor_url` | Build a request path ready to send to imagor, signed as configured. |
| `build_imagor_path` | Build an **unsigned** path for you to sign. |
| `validate_imagor_path` | Parse a path, report whether it round-trips, and summarize the transformation. |
| `list_imagor_filters` | Accepted filters with signature, description and argument forms. |
| `describe_imagor` | The supported path grammar and the current signing mode. |
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
label under a product, a caption in a cell.

**A slot's `filters` field applies filters to that slot's own source**, written the
way imagor writes them: `smart/filters:round_corner(230,230,f4f1e8)`.

`parse_template` reads a path back into a layout, so one you already have can be
edited rather than retyped. The one thing a path cannot carry is a *form* — it holds
the pixels a box resolved to, not whether it was written `38p` or `half`.

## Cookbook and skill

`cookbook/` holds **fourteen ready-made layouts** for the formats that come up most
— OG cards, a product grid, a profile card, quotes, testimonials, an event, a
comparison table, a price ladder. Take the closest, change the values, render it.

Every recipe is rendered by the test suite **at three canvas sizes on every
commit**, and the contact sheet above is generated from them by the same code path,
so it cannot drift from what the recipes produce.

The server also carries a **skill**, served as `imagor-mcp://skill`. It covers what
a tool description cannot: the order a path is written in, where text lands, how a
label is centred, and a method that builds small pieces before assembling them.

## Modes

Exactly one applies, selected by the environment. There is no implicit fallback.

| Mode | Selected by | Output |
|---|---|---|
| **signed** | `IMAGOR_SECRET` set | `<signature>/<path>` — production |
| **unsafe** | `IMAGOR_UNSAFE=1` | `unsafe/<path>` — development only |

**Setting both is refused at startup, and so is setting neither** — they state
opposite intentions, and silently picking one is how an unsigned deployment reaches
production.

## Configuration

| Variable | Meaning |
|---|---|
| `IMAGOR_SECRET` | Signing secret. Selects **signed** mode. |
| `IMAGOR_UNSAFE` | `1`, `true`, `yes` or `on`. Selects **unsafe** mode. Must be explicit. |
| `IMAGOR_BASE_URL` | Your imagor origin. Needed for `verify_imagor_url`, and it adds `absolute_url` to output. |
| `IMAGOR_SIGNER_TYPE` | `sha1` (default), `sha256` or `sha512`. |
| `IMAGOR_SIGNER_TRUNCATE` | Signature length in characters. `0` means none. |
| `IMAGOR_MCP_EXTRA_FILTERS` | Extra filter names for forks that register their own. |

The signer settings must match your imagor exactly, or it rejects the URLs.
**`IMAGOR_BASE_URL` has to be reachable from inside the imagor container** — the
preview fetches the render back through it to scale it down.

### Making previews faster

`verify_imagor_url` fetches through imagor's `preview()` filter, which skips result
storage and opts into the in-memory cache. **Turn the cache on** and repeated looks
at the same composition get much cheaper — measured on a 1600×600 composition,
**386 ms** for the first preview and **25 ms** for a repeat:

```bash
VIPS_CACHE_SIZE=52428800      # byte budget; 0 disables the cache (default)
VIPS_CACHE_MAX_WIDTH=2400     # images larger than this are not cached
VIPS_CACHE_MAX_HEIGHT=2000
```

## Security

**A trusted client of imagor, not a new surface on it.** It holds the signing secret
for the same reason any backend service does: to mint valid URLs. Trust the
component; do not trust the agent using it.

- **A mode is required.** It signs or emits `unsafe/`; it never runs unsigned.
- **Unsafe mode is opt-in and loud** — it warns on stderr, and `describe_imagor`
  reports it at runtime.
- **stdio only.** Nothing listens on the network.
- **No generic signing tool.** It can only emit paths its own typed tools built.
- **Blast radius stays imagor's to set** — `HTTP_LOADER_ALLOWED_SOURCES`,
  `VIPS_MAX_RESOLUTION`, `IMAGOR_REQUEST_TIMEOUT` and the rest. A signed URL from an
  errant agent is still subject to every one of those limits.

Do not host this multi-tenant on behalf of users whose keys you would then hold.
Ship it self-hosted alongside imagor.

## Development

```sh
docker run -d -p 8197:8000 -e IMAGOR_UNSAFE=1 ghcr.io/cshum/imagor:1.9.9
IMAGOR_BASE_URL=http://localhost:8197 IMAGOR_UNSAFE=1 go test ./...
python3 cookbook/render.py /tmp/gallery   # render every recipe, build the sheet
```

Use the published image rather than one built for your machine: it is the only place
the declared font families exist, and a host binary substitutes a system face
silently while the tests pass. See [AGENTS.md](AGENTS.md) for the conventions and
[docs/build-log.md](docs/build-log.md) for the faults that shaped them.

CI runs `gofmt -l`, `go vet`, `go test -race`, a drift guard on the generated filter
catalogue, and a job that re-renders the gallery.

## Licence

MIT. See [LICENSE](LICENSE).
