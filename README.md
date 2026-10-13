# imagor-template

**Templates for [imagor](https://github.com/cshum/imagor) — the fast, self-hosted,
open-source image server — with agent capabilities on top.**

**An agent writes the layout; you get back one imagor URL.** Paste it in a browser, store
it, serve it from a CDN, put it in a page — no image library, no template editor, no
design file.

![Fourteen cards, each one a single imagor URL](docs/gallery/contact-sheet.png)

*Every card above is one imagor URL, composed by an agent from a brief. **[All fourteen,
with the template behind each »](docs/gallery/cookbook.md)***

Reusable here means **programmable**: the output is a URL, so your code can hold one,
template it, or hand it to an `<img>` tag, and imagor renders it at 4–8× the speed of the
quickest ImageMagick settings. The layout an agent writes is how you get one — it is not
something you have to read.

## What an agent does with it

Ask for a card and it builds one:

> *Build a 1200×630 launch card: a deep green ground, a serif headline, a product photo
> on the right, and the price below it. Show me what it looks like.*

It writes the template, renders it, **looks at the image**, and revises. **Given a brief
with no layout, no positions and no sizes, an agent designed three distinct banners
across three runs** — hierarchy, colour, type pairing, invented copy — and zoomed into
regions to read the small text it had drawn.

That is the difference from editor-first image tooling: there is no canvas to design in.
A layout *is* a URL, and a URL is something an agent can write, check and rewrite.

## Where this fits

Three ways to get a dynamic image:

| | how | what it costs you | what limits you |
|---|---|---|---|
| **Chromium** (Puppeteer, Satori) | screenshot an HTML page per image | nothing, but a browser to run | **slow** — [they say so themselves](https://www.bannerbear.com/blog/how-to-make-a-custom-open-graph-image-using-puppeteer/) |
| **Hosted template APIs** (Bannerbear, Placid, APITemplate) | send values, **get a blob back** | $0.019–0.049 per image, from $49/mo | the per-image fee, and the transfer every time you serve it |
| **imagor-template** | a layout compiles to an imagor URL | nothing beyond the imagor you already run | nothing per image — **you serve a URL** |

**The difference is not that this is cheaper. It is that the output is a URL.**

A hosted API hands you bytes, so every render is a call you pay for and a transfer you
carry. **This hands you a URL**, and imagor serves it as one: **`Cache-Control:
max-age=168h` by default**, so a CDN, a proxy or a browser holds it without asking you
again. Turn on result storage and the origin renders it once for the life of the entry.

**imagor does the image work at libvips speed** — its own benchmark puts it ahead of
thumbor overall and level with imgproxy across common formats, and both are libvips-based,
so the floor is the same. **The saving is architectural, not a faster loop:** a
deterministic URL is a cacheable one, and the byte transfer is the part that actually
scales with traffic.

Rates are Bannerbear, Placid and APITemplate's published pricing as of October 2026
($49/mo for 1,000 images at the top of that range). The benchmark figures are imagor's,
not measurements I ran.

## Quick start

**1. You need an imagor to render with.** The official image is the one the
cookbook targets:

```sh
docker run -d -p 8000:8000 \
  -e IMAGOR_SECRET=your-secret-here \
  cshum/imagor-base
```

Already running imagor? Skip this and point the config below at it.

**2. Install.** Two front ends on one engine, and both ship together. **If you are
driving it from an agent you only need the server**; the compiler is there for the cases
with no agent in them — a backend resolving a stored layout, a script checking the
cookbook:

```sh
# the MCP server: an agent composes, renders, looks at the result and revises
imagor-mcp

# the compiler on its own: a layout in, one imagor URL out
imagor-template compile cookbook/event.json
```

Both come prebuilt for six platforms, or build from source:

```sh
# pick your platform from the releases page
curl -L -o imagor-mcp https://github.com/cshum/imagor-template/releases/latest/download/imagor-mcp_v0.3.0_darwin_arm64
chmod +x imagor-mcp

# or, with Go installed
go install github.com/cshum/imagor-template@latest
```

Pure Go, with no libvips and no cgo — unlike imagor itself. Every platform is on the
[releases page](https://github.com/cshum/imagor-template/releases).

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
| `render_template` | Render a layout at a canvas size, with values filled in. **Start here — half the loop.** |
| `verify_imagor_url` | Fetch the rendered result and report which filters applied. **Returns the image, so the agent can look at what it built — the other half.** |
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

A template has **no canvas size in it**. Geometry is relative, so one layout serves
1200×630 and 1080×1080, and text with a `height` is **fitted** to its box — it composes
once and renders at every format.

Three things worth knowing:

- **A slot with its own `slots` is a canvas.** Its width and height become the canvas the
  inner slots lay out on, and the result draws at the outer slot's position. This is how
  a label sits under a product, or a caption inside a cell.
- **A slot's `filters` apply to that slot's own source**, written as imagor writes them:
  `smart/filters:round_corner(230,230,f4f1e8)`.
- **`parse_template` reads a path back** — useful for a layout you already have. It cannot
  recover a *form*: `38p` comes back as the number it resolved to.

## The skill

The server carries a skill, served at `imagor-mcp://skill`. It is the part of using
this that does not fit in a tool description, and an agent that skips it composes badly:

- **The path grammar** — the order a path is written in, and the two crops whose orders
  are opposite.
- **Where text lands** — placement is a *search*, not a calculation, and the skill says so.
- **Awkward values** — colours, and what has to be encoded.
- **A working method** — build small pieces, check each, then assemble.

It travels with the server, so a client that reads only the tool list never sees it.
`get_imagor_skill` returns it for a client that reads tools rather than resources.

## The cookbook

`cookbook/` holds **fourteen ready-made layouts** — OG cards, a product grid, a profile
card, quotes, testimonials, an event, a comparison table, a price ladder. Take the
closest, change the values, render it.

**[Every card, its template, and the path it resolves to »](docs/gallery/cookbook.md)**
Hand an agent the **template**; the path is there to show what it compiles to.

Every recipe renders **at three canvas sizes on every commit**, and the contact sheet
above comes from that same code path — so it cannot drift.

## One direction is the source

| | |
|---|---|
| **template → path** | total and deterministic. The compiler. **The source of truth.** |
| **path → template** | **lossy.** A path holds what a layout *resolved to*, not the decisions behind it. |

Reading a path back gives the geometry, colours and text — but **the slot names and the
`{{values}}` are gone, and a relative form like `38p` or `half` is already the number it
resolved to.** So hand an agent the **template**, not the path: a template re-fills for
the next person, a path renders once.

**A caveat:** a text slot with a height is *fitted* by measuring, and measuring renders a
probe through imagor — so a compile needs a running imagor. Everything else is pure.

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

**A trusted client of imagor, not a new surface on it.** It holds the signing secret to
mint valid URLs, like any backend. **Trust the component; do not trust the agent using
it.**

- **A mode is required** — it signs or emits `unsafe/`, never unsigned.
- **Unsafe mode is opt-in and loud** — a warning on stderr, reported by `describe_imagor`.
- **stdio only.** Nothing listens on a port.
- **No generic signing tool** — it can only emit paths its own typed tools built.
- **Blast radius stays imagor's to set** — `HTTP_LOADER_ALLOWED_SOURCES`,
  `VIPS_MAX_RESOLUTION`, `IMAGOR_REQUEST_TIMEOUT`. A URL from an errant agent is still
  subject to every one.

Do not host this multi-tenant for users whose keys you would then hold.

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
