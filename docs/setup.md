# Setup

Set exactly one of `IMAGOR_SECRET` or `IMAGOR_UNSAFE`. Setting both, or neither, is refused at
startup.

| Variable | Meaning |
|---|---|
| `IMAGOR_SECRET` | Signing secret. Selects signed mode: `<signature>/<path>`. |
| `IMAGOR_UNSAFE` | `1`, `true`, `yes` or `on`. Selects unsafe mode: `unsafe/<path>`, development only. Must be explicit. |
| `IMAGOR_BASE_URL` | Your imagor origin, as you reach it. Needed for `verify_imagor_url`, and it adds `absolute_url` to output. |
| `IMAGOR_SIGNER_TYPE` | `sha1` (default), `sha256` or `sha512`. |
| `IMAGOR_SIGNER_TRUNCATE` | Signature length in characters. `0` means none. |
| `IMAGOR_MCP_EXTRA_FILTERS` | Extra filter names for forks that register their own. |

The signer settings must match your imagor exactly, or it rejects the URLs.

Compose in `IMAGOR_UNSAFE`; serve in `IMAGOR_SECRET`. A signature covers the whole path, so
editing any part of a signed URL invalidates it. Switch when the URL is finished and something
else is going to serve it.

`IMAGOR_BASE_URL` does two jobs. It is where the server fetches a render back to verify it, and
it is what the URLs it returns are prefixed with. Both have to be an address you can open, which
is why it names `localhost` and not a container-internal host.

That is the one real cost of running the server in a container. There, the host is
`host.docker.internal` and `localhost` is the container, so the server needs
`-e IMAGOR_BASE_URL=http://host.docker.internal:8000` to work at all — and every URL it hands you
then carries a name your browser cannot resolve. On the host, one address does both jobs.
Anything that puts a proxy or a different hostname in front of imagor does the same.

## Clients

The server speaks MCP over stdio, so it is a `command` a client runs. Nearly every client
describes that with the same three fields, and then disagrees about where the file lives and what
the servers are keyed under.

This is the shape most clients read — Claude Desktop, Claude Code, Cursor, Windsurf:

```json
{
  "mcpServers": {
    "imagor": {
      "command": "imagor-mcp",
      "env": { "IMAGOR_UNSAFE": "1", "IMAGOR_BASE_URL": "http://localhost:8000" }
    }
  }
}
```

Two clients read something else, and both fail silently if you paste the block above:

| client | config | servers key | format |
|---|---|---|---|
| Claude Desktop | `~/Library/Application Support/Claude/claude_desktop_config.json` | `mcpServers` | JSON |
| Claude Code | `.mcp.json` (project) or `~/.claude.json` (local, user) | `mcpServers` | JSON |
| Cursor | `~/.cursor/mcp.json` | `mcpServers` | JSON |
| Windsurf (Devin Desktop) | `~/.codeium/windsurf/mcp_config.json` | `mcpServers` | JSON |
| VS Code | `~/Library/Application Support/Code/User/mcp.json`, or `.vscode/mcp.json` per project | `servers` | JSONC |
| Codex | `~/.codex/config.toml` | `mcp_servers` | TOML |

Windows is `%APPDATA%` where macOS is `~/Library/Application Support`, and `%USERPROFILE%` where
macOS is `~`.

### VS Code

Keyed under `servers`, not `mcpServers`. A block copied from anywhere else parses fine and does
nothing, which is the most common way this goes wrong. This is the same file the install link
writes to.

If the link does not open, paste this into your browser's address bar:

```
vscode:mcp/install?%7B%22name%22%3A%22imagor%22%2C%22command%22%3A%22imagor-mcp%22%2C%22env%22%3A%7B%22IMAGOR_UNSAFE%22%3A%221%22%2C%22IMAGOR_BASE_URL%22%3A%22http%3A%2F%2Flocalhost%3A8000%22%7D%7D
```

The file itself:

```jsonc
{
  "servers": {
    "imagor": {
      "type": "stdio",
      "command": "imagor-mcp",
      "env": { "IMAGOR_UNSAFE": "1", "IMAGOR_BASE_URL": "http://localhost:8000" }
    }
  }
}
```

### Codex

TOML, so the JSON above is not a near miss — it is a syntax error:

```toml
[mcp_servers.imagor]
command = "imagor-mcp"
env = { IMAGOR_UNSAFE = "1", IMAGOR_BASE_URL = "http://localhost:8000" }
```

### Claude Desktop

Usually the simplest path is the bundle rather than this file:
[`imagor-mcp.mcpb`](https://github.com/cshum/imagor-mcp/releases/latest/download/imagor-mcp.mcpb)
installs as an extension, one click.

### Anything else

If a client speaks MCP over stdio it takes the block above, and the only question is which file
it reads and what it calls the servers. `command: imagor-mcp` assumes the
[brew install](../README.md#quick-start) put it on `PATH`.

## Previews

`verify_imagor_url` fetches through imagor's `preview()` filter, which opts the request into
imagor's in-memory cache. That cache is imagor's setting, not this server's, and it is keyed by
source image rather than by composition — see
[imagor's in-memory cache](https://docs.imagor.net/in-memory-cache) for the variables and what
they do.

## Security

This server is a trusted client of imagor, not a new surface on it. It holds the signing secret
to mint valid URLs, like any backend. Trust the component; do not trust the agent using it.

- **The secret is environment-only.** It is read from `IMAGOR_SECRET` and never crosses the tool
  surface: no tool takes a secret, and none returns one. An agent has no access to the key, so it
  cannot leak it or be prompted into using it.
- **Unsafe mode is opt-in and loud.** A warning on stderr, reported by `describe_imagor`.
- **Blast radius stays imagor's to set.** A URL from an errant agent is still subject to every
  limit on the instance it lands on: `HTTP_LOADER_ALLOWED_SOURCES`, `VIPS_MAX_RESOLUTION`,
  `IMAGOR_REQUEST_TIMEOUT`, and the `expire()` filter for a hard expiry. All of them are imagor's
  settings, documented at [docs.imagor.net/security](https://docs.imagor.net/security) — this
  server has no opinion about them and no way to override them.

Do not host this multi-tenant for users whose keys you would then hold.
