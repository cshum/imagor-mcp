# Development

Building from source, with Go 1.27 or later:

```sh
go install github.com/cshum/imagor-mcp/cmd/imagor-mcp@latest
```

The `./cmd/` path is required: the module root is not a command, and nothing here is importable —
the engine is all `internal/`.

Tests need an imagor to measure against:

```sh
docker run -d -p 8197:8000 ghcr.io/cshum/imagor:1.9.10 -imagor-unsafe
IMAGOR_BASE_URL=http://localhost:8197 IMAGOR_UNSAFE=1 go test ./...
python3 cookbook/render.py /tmp/gallery   # render every recipe, build the sheet
python3 cookbook/index.py                 # regenerate the embedded recipe index
```

## The CLI

`imagor-template` is the second binary, for running a template with no agent and no MCP client in
the path — a stored layout, a build step, a script checking the cookbook:

```sh
imagor-template compile cookbook/event.json   # a layout in, one imagor URL out
imagor-template decompile '<path>'            # a path back to a layout
```

Compiling is total and deterministic, so the same layout gives the same path every time.
Decompiling is lossy — see
[One direction is the source](docs/reference.md#one-direction-is-the-source) for what a path
cannot carry. Both call the same engine the server does, and `cmd/imagor-template/main_test.go`
compiles every recipe through both front ends and fails if they diverge.

Use the published image rather than one built for your machine: it is the only place the declared
font families exist, and a host binary substitutes a system face silently while the tests pass.
See [AGENTS.md](AGENTS.md) for the conventions and [docs/build-log.md](docs/build-log.md) for the
bugs behind them.

CI runs `gofmt -l`, `go vet`, `go test -race`, a drift guard on the generated filter catalogue,
and a job that re-renders the gallery.
