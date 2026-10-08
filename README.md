# imagor-mcp

An MCP server that exposes [imagor](https://github.com/cshum/imagor)'s image
pipeline to AI agents as typed, validated tools.

It does **not** upload, store, or render images. It constructs imagor paths
and, when configured with a signing secret, signs them. Every path is
validated by round-tripping through imagor's own parser before it is returned,
so an invalid composition is refused rather than emitted as a broken URL.

**Status: pre-release.** The tool surface is implemented and tested; the
filter catalogue is not yet machine-readable (see Limitations).

## Why this exists

Agents can already write image URLs by hand, and they get them subtly wrong —
wrong argument order, invented filter names, combinations imagor rejects. This
server inverts that: the agent calls tools with typed arguments, and only valid
paths can come out.

The path grammar, path generation, parsing and signing are **not reimplemented
here**. They come from imagor's own `imagorpath` package, so behaviour cannot
drift from the server.

## Install

```sh
go install github.com/cshum/imagor-mcp@latest
```

Or build from source:

```sh
go build -o imagor-mcp ./cmd/imagor-mcp
```

The server is pure Go. It builds with `CGO_ENABLED=0` and needs neither libvips
nor cgo, unlike imagor itself, so `go install` works with no native toolchain.

## Modes

Exactly one mode applies, selected by the environment. There is no implicit
fallback between them.

| Mode | Selected by | Output |
|---|---|---|
| **signed** | `IMAGOR_SECRET` set | `<signature>/<path>` — production |
| **unsafe** | `IMAGOR_UNSAFE=1` | `unsafe/<path>` — development only |
| **path_out** | neither set | the bare `<path>`, for the caller to sign |

**Setting both `IMAGOR_SECRET` and `IMAGOR_UNSAFE` is refused at startup.** They
state opposite intentions, and silently picking one is how an unsigned
deployment reaches production. The server never falls back to unsigned URLs on
its own.

## Configuration

| Variable | Meaning |
|---|---|
| `IMAGOR_SECRET` | Signing secret. Selects **signed** mode. |
| `IMAGOR_UNSAFE` | `1`, `true`, `yes` or `on`. Selects **unsafe** mode. Must be explicit. |
| `IMAGOR_SIGNER_TYPE` | `sha1` (default), `sha256` or `sha512`. Use `sha256` in production. |
| `IMAGOR_SIGNER_TRUNCATE` | Signature length in characters, e.g. `40`. `0` means no truncation. |
| `IMAGOR_BASE_URL` | Optional imagor origin. When set, output includes a full `absolute_url`. |
| `IMAGOR_MCP_EXTRA_FILTERS` | Comma-separated extra filter names, for extensions and forks that register filters imagor does not document. Bypass name checking and are not arity-validated. |

The signing settings must match the target imagor instance exactly, or its
signature verification will reject the URLs.

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
| `build_imagor_url` | Build a request path ready to send to imagor, signed or `unsafe/` as configured. Refused in path-out mode. |
| `build_imagor_path` | Build an **unsigned** path for the caller to sign. Use when the MCP server should not hold the secret. |
| `validate_imagor_path` | Parse a path, report whether it round-trips unchanged, and summarize the transformation. |
| `list_imagor_filters` | List the accepted filters with their argument counts, optionally filtered by name. |
| `describe_imagor` | Report the supported path grammar and the current signing mode. |

Every transformation parameter maps onto a concept in the imagor path grammar.
There are no invented parameters and no generic "sign this string" tool.

## Security model

**The server is a trusted client of imagor, not a new surface on it.** It holds
the signing secret for the same reason any backend service does: to mint valid
URLs. Trust the component; do not trust the agent using it.

- **No secret = path-out mode.** The safest deployment gives the server no
  secret at all and signs in the calling application.
- **Unsafe mode is opt-in and loud.** It warns on stderr at startup, and
  `describe_imagor` reports it as a limitation at runtime. The target imagor
  must itself be running with `IMAGOR_UNSAFE=1`.
- **stdio only.** With stdio transport nothing listens on the network.
- **No generic signing tool.** The server can only emit paths its own typed
  tools constructed.
- **Blast radius is bounded by imagor's own configuration** —
  `HTTP_LOADER_ALLOWED_SOURCES`, `HTTP_LOADER_BASE_URL`,
  `HTTP_LOADER_HTTPS_ONLY`, `VIPS_MAX_RESOLUTION`/`_WIDTH`/`_HEIGHT`,
  `IMAGOR_REQUEST_TIMEOUT`, `IMAGOR_PROCESS_CONCURRENCY`. A signed URL from an
  errant agent is still subject to every one of those limits.

Do not host this server multi-tenant on behalf of users whose signing keys you
would then hold. Ship it self-hosted alongside imagor.

## Filter validation

imagor accepts any `name(args)` in a `filters:` segment. **Unknown filter names
and wrong argument counts produce no error**: the request returns `200` with the
image unchanged. Measured against imagor v1.9.7, all of the following returned
byte-identical output to a filter-less request:

```
filters:vintage(70)      unknown name
filters:blur(1,2,3)      too many arguments
filters:blur()           missing a required argument
filters:rotate()         missing a required argument
filters:brightness()     missing a required argument
filters:format()         missing a required argument
```

That is a reasonable choice for a permissive server and a dangerous one for an
agent, which cannot tell a working URL from an inert one. This server is the
strict layer, kept outside imagor: filter names and argument counts are checked
before a path is signed, and the catalogue is generated from the exact imagor
release in `go.mod` (see `cmd/genfilters`, run via `go generate ./internal/filters`).

One combination is valid but inert, and is reported as a warning rather than an
error: `quality()` only affects lossy output. With no `format()` filter the
result may be lossless PNG, where `quality(30)` changes nothing.

## Limitations

- **Argument values are not validated, only names and counts.** `quality(abc)`
  has the correct arity and passes validation; imagor then ignores it. Value
  types are not reliably derivable from the filter reference, so they are not
  enforced.
- **Arity checking is stricter than imagor.** Several filters tolerate fewer
  arguments than their documentation states and fall back to defaults. Measured
  against v1.9.7: `sharpen()`, `pixelate()`, `fill()`, `text(a,b)` and
  `grayscale(1)` all still take effect. This server refuses them anyway. The
  asymmetry is deliberate — a refusal is loud and the agent corrects it in one
  turn, whereas imagor's silent no-op is invisible — but it is a real
  difference, and `IMAGOR_MCP_EXTRA_FILTERS` is not a workaround for it.
- **The catalogue is tied to one imagor release.** It is generated from the
  version in `go.mod`. After upgrading imagor, re-run `go generate
  ./internal/filters`; the diff shows exactly which filters changed.
- **No upload or POST support.** External upload needs its own signed-grant
  mechanism and is deliberately out of scope.
- **No rendering.** The server never fetches or processes an image.

## Development

```sh
go test ./...          # unit + protocol-level tests
go vet ./...
```

The test suite spawns the real binary over stdio and includes signature values
verified against a running imagor instance and against imagor's published
documentation, so the assertions are anchored to external ground truth rather
than to this implementation.

## Licence

MIT. See [LICENSE](LICENSE).
