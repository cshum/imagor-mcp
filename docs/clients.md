# Installing into a client

The server speaks MCP over **stdio**, so it is a `command` a client runs. Nearly every
client describes that with the same three fields, and then disagrees about where the file
lives and what the servers are keyed under.

**This is the shape most clients read** — Claude Desktop, Claude Code, Cursor, Windsurf:

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

**Two clients read something else, and both fail silently if you paste the block above:**

| client | config | servers key | format |
|---|---|---|---|
| Claude Desktop | `~/Library/Application Support/Claude/claude_desktop_config.json` | `mcpServers` | JSON |
| Claude Code | `.mcp.json` (project) or `~/.claude.json` (local, user) | `mcpServers` | JSON |
| Cursor | `~/.cursor/mcp.json` | `mcpServers` | JSON |
| Windsurf (Devin Desktop) | `~/.codeium/windsurf/mcp_config.json` | `mcpServers` | JSON |
| **VS Code** | `~/Library/Application Support/Code/User/mcp.json`, or `.vscode/mcp.json` per project | **`servers`** | JSONC |
| **Codex** | `~/.codex/config.toml` | **`mcp_servers`** | **TOML** |

Windows is `%APPDATA%` where macOS is `~/Library/Application Support`, and
`%USERPROFILE%` where macOS is `~`.

### VS Code

Keyed under `servers`, not `mcpServers` — a block copied from anywhere else parses fine and
does nothing, which is the single most common way this goes wrong. This is the same file
the install link writes to.

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

If a client speaks MCP over stdio it takes the block at the top of this page, and the only
question is which file it reads and what it calls the servers. `command: imagor-mcp` assumes
the [brew install](../README.md#quick-start) put it on `PATH`.
