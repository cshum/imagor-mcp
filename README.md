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

## Configuration

| Variable | Meaning |
|---|---|
| `IMAGOR_SECRET` | Signing secret. **Unset means path-out mode**: the server returns unsigned paths for the caller to sign. |
| `IMAGOR_SIGNER_TYPE` | `sha1` (default), `sha256`, or `sha512`. Use `sha256` in production. |
| `IMAGOR_SIGNER_TRUNCATE` | Signature length in characters, e.g. `40`. |
| `IMAGOR_BASE_URL` | Optional imagor origin, reported back in tool output. |

These must match the target imagor instance exactly, or its signature
verification will reject the URLs.

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
| `build_imagor_url` | Build a **signed** URL from typed transformation parameters. Refused when no secret is configured. |
| `build_imagor_path` | Build an **unsigned** path for the caller to sign. Use when the MCP server should not hold the secret. |
| `validate_imagor_path` | Parse a path, report whether it round-trips unchanged, and summarize the transformation. |
| `describe_imagor` | Report the supported path grammar and the current signing mode. |

Every transformation parameter maps onto a concept in the imagor path grammar.
There are no invented parameters and no generic "sign this string" tool.

## Security model

**The server is a trusted client of imagor, not a new surface on it.** It holds
the signing secret for the same reason any backend service does: to mint valid
URLs. Trust the component; do not trust the agent using it.

- **No secret = path-out mode.** The safest deployment gives the server no
  secret at all and signs in the calling application.
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

## Limitations

- **The filter catalogue is not enumerated.** imagor does not currently expose
  its filter registry in a machine-readable form, and a hand-maintained copy
  would silently rot as filters change. Pass filter names and arguments
  directly; `describe_imagor` reports this limitation at runtime.
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

Apache-2.0, matching imagor.
