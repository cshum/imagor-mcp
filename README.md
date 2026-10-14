# imagor-mcp

**An MCP server and a skill that add template capability to
[imagor](https://github.com/cshum/imagor).**

An agent writes a layout; the layout compiles to one imagor URL — pasted in a browser,
stored, served from a CDN, put in a page.

[![Fourteen cards, each one a single imagor URL](docs/gallery/contact-sheet.png)](docs/gallery/cookbook.md)

*Every card above is one imagor URL, composed by an agent from a brief. **[All fourteen,
with the URL each resolves to, and a prompt to ask for it »](docs/gallery/cookbook.md)***

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

## What this adds

**imagor-mcp adds a template layer to imagor.** Rendering stays in the imagor instance you
run, at the throughput it already has.

**A layout compiles to a URL; a hosted template API returns an image.** Those bill and
transfer the bytes on every render. imagor serves a URL instead, which any browser, proxy or
CDN can cache — so the bytes are transferred once.

**Measured.** A 1200×630 composite from the cookbook renders on a `shumc/imagor` container in
**119 ms p50** (p95 122 ms, 25 fetches) and returns **36 KB** of JPEG; the URL that produced
it is **865 characters**.

*The 119 ms and the 36 KB are mine, measured against a `shumc/imagor` container.*

## Quick start

**1. You need an imagor to render with.** Any imagor will do; the published image is
[`shumc/imagor`](https://hub.docker.com/r/shumc/imagor):

```sh
docker run -d -p 8000:8000 -e IMAGOR_SECRET=your-secret-here shumc/imagor
```

Already running one? Point the config below at it instead. (`imagor-base` is the image
imagor is *built* from, not one to run.)

**2. Install.** **In a host that takes bundles, this is the whole install** — download
[`imagor-mcp-v0.5.0.mcpb`](https://github.com/cshum/imagor-mcp/releases/download/v0.5.0/imagor-mcp-v0.5.0.mcpb)
and open it. It asks for the imagor URL and secret and runs the server for you, on macOS,
Linux and Windows. The same bundle is on
[Smithery](https://smithery.ai/server/adrian-pzhc/imagor-mcp) and in the
[MCP Registry](https://registry.modelcontextprotocol.io/) as `io.github.cshum/imagor-mcp`.

**Or run the server as a container**, which carries the skill with it:

```sh
docker run -i --rm -e IMAGOR_SECRET=your-secret-here ghcr.io/cshum/imagor-mcp
```

Point a client at the container instead of a binary:

```json
{
  "mcpServers": {
    "imagor": {
      "command": "docker",
      "args": ["run", "-i", "--rm",
        "-e", "IMAGOR_SECRET=your-secret-here",
        "-e", "IMAGOR_BASE_URL=http://host.docker.internal:8000",
        "ghcr.io/cshum/imagor-mcp"]
    }
  }
}
```

Inside a container `localhost` is the container, so a local imagor is reached at
`host.docker.internal` (`--add-host=host.docker.internal:host-gateway` on Linux).

Anywhere else, the release carries both binaries for six platforms:

```sh
curl -L -o imagor-mcp \
  https://github.com/cshum/imagor-mcp/releases/download/v0.5.0/imagor-mcp_v0.5.0_darwin_arm64
chmod +x imagor-mcp
```

Every platform is on the [releases page](https://github.com/cshum/imagor-mcp/releases).
**The server is the one you need**; `imagor-template` runs a layout you already have,
covered in [docs/development.md](docs/development.md).

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
| `get_cookbook_recipe` | Fetch a ready-made layout by name. **Call it with no name first** to get the list. |
| `get_imagor_skill` | The skill, for a client that reads it as a tool rather than a resource. |

Every parameter maps onto a concept in the imagor path grammar, and signing happens only
as the by-product of a typed tool.

## The cookbook

`cookbook/` holds **fourteen ready-made layouts** — OG cards, a product grid, a profile
card, quotes, testimonials, an event, a comparison table, a price ladder. Take the
closest, change the values, render it.

**[Every card, and the path it resolves to »](docs/gallery/cookbook.md)** The template
is not on that page: a recipe is the channel between an agent and the renderer, so it
travels in the server and arrives by name.

Every recipe renders **at three canvas sizes on every commit**, and the contact sheet
above comes from that same code path — so it cannot drift.

## Configuration

**A mode is required, and exactly one applies** — selected by the environment, with no
implicit fallback. Setting both is refused at startup, and so is setting neither: they
state opposite intentions, and silently picking one is how an unsigned deployment reaches
production.

| Variable | Meaning |
|---|---|
| `IMAGOR_SECRET` | Signing secret. Selects **signed** mode: `<signature>/<path>`. |
| `IMAGOR_UNSAFE` | `1`, `true`, `yes` or `on`. Selects **unsafe** mode: `unsafe/<path>`, development only. Must be explicit. |
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

## Going deeper

The format, and the conventions an agent needs to compose well:

- **[docs/format.md](docs/format.md)** — a template carries no canvas size, and why a path
  cannot be read back into a template.
- **[docs/skill.md](docs/skill.md)** — the skill the server carries, and what it covers.
- **[docs/security.md](docs/security.md)** — what it is trusted with, and what it refuses.
- **[docs/development.md](docs/development.md)** — building, testing, and the CLI.

## Licence

MIT.
