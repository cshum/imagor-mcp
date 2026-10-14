# Development

```sh
docker run -d -p 8197:8000 -e IMAGOR_UNSAFE=1 ghcr.io/cshum/imagor:1.9.10
IMAGOR_BASE_URL=http://localhost:8197 IMAGOR_UNSAFE=1 go test ./...
python3 cookbook/render.py /tmp/gallery   # render every recipe, build the sheet
python3 cookbook/index.py                 # regenerate the embedded recipe index
```

### The CLI

`imagor-template` is the second binary, for running a template with no agent and no MCP
client in the path — a stored layout, a build step, a script checking the cookbook:

```sh
imagor-template compile cookbook/event.json   # a layout in, one imagor URL out
imagor-template decompile '<path>'            # a path back to a layout
```

Compiling is total and deterministic, so the same layout gives the same path every time.
**Decompiling is lossy** — see [One direction is the source](format.md#one-direction-is-the-source)
for what a path cannot carry. Both call the same engine the server does, and
`cmd/imagor-template/main_test.go` compiles every recipe through both front ends and
fails if they ever disagree.

Use the published image rather than one built for your machine: it is the only place
the declared font families exist, and a host binary substitutes a system face
silently while the tests pass. See [AGENTS.md](../AGENTS.md) for the conventions and
[build-log.md](build-log.md) for the faults that shaped them.

CI runs `gofmt -l`, `go vet`, `go test -race`, a drift guard on the generated filter
catalogue, and a job that re-renders the gallery.

The CLI and the server are the same engine: `cmd/imagor-template/main_test.go` compiles every
recipe through both front ends and fails if they diverge.
