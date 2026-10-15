# Reference

## Tools

Every tool the server answers `tools/list` with. Every parameter maps onto a concept in the
[layout format](#the-layout-format) — there are no invented parameters, and nothing is signed
except by a tool that builds a path.

| Tool | Purpose |
|---|---|
| `render_template` | Render a layout at a canvas size, with values filled in. Start here: half the loop. |
| `verify_imagor_url` | Fetch the rendered result and report which filters applied. Returns the image, so the agent can look at what it built — the other half. |
| `parse_template` | Read a path back into a layout, so one you have can be edited. |
| `fit_text` | Find the font size at which a string fits a box, by measuring. |
| `build_imagor_url` | Build a request path ready to send to imagor, signed as configured. Its `path` field is the bare path, for a caller that signs itself. |
| `validate_imagor_path` | Parse a path, report whether it round-trips, and summarize the transformation. |
| `list_imagor_filters` | Accepted filters with signature, description and argument forms. |
| `describe_imagor` | The supported path grammar and the current signing mode. |
| `get_cookbook_recipe` | Fetch a ready-made layout by name. Call it with no name first to get the list — [every recipe, with the url each resolves to](cookbook.md). |
| `get_imagor_skill` | The skill, for a client that reads it as a tool rather than a resource. Called with no section it returns the outline; name one to read just that part. |

The server also serves one prompt, `compose-imagor-image`, which loads the skill with a brief
attached. Prompts are chosen by the user from the client's menu rather than called by the model,
so it costs a session nothing until someone picks it.

## The layout format

A template carries no canvas size. Geometry is relative, and text with a `height` is fitted to
its box, so the same layout serves 1200×630 and 1080×1080 — write it once, render it anywhere.

The rest of the format — nested canvases, filters on a slot's own source, what a path can and
cannot carry — is in [the skill](#the-skill), where the reader is an agent that has to write one.

### One direction is the source

| | |
|---|---|
| template → path | Total and deterministic. The compiler. This is the source of truth. |
| path → template | Lossy. A path holds what a layout resolved to, not the decisions behind it. |

Reading a path back gives the geometry, colours and text, but the slot names and the `{{values}}`
are gone, and a relative form like `38p` or `half` is already the number it resolved to. So hand
an agent the template, not the path: a template re-fills for the next person, a path renders once.

One caveat: a text slot with a height is fitted by measuring, and measuring renders a probe
through imagor — so a compile needs a running imagor. Everything else is pure.

## The skill

The server carries a skill, served at `imagor-mcp://skill`. It carries what a tool description
cannot, and an agent that skips it writes worse layouts:

- The path grammar — the order a path is written in, and the two crops whose orders are opposite.
- Where text lands — placement is a search, not a calculation, and the skill says so.
- Awkward values — colours, and what has to be encoded.
- A working method — build small pieces, check each, then assemble.

It travels with the server, so a client that reads only the tool list never sees it.
`get_imagor_skill` returns it for a client that reads tools rather than resources.

### Served in sections

The document is 18 KB and most of it is irrelevant to any one composition, so the tool serves it
in two steps. Called with no section it returns the frontmatter and the outline: the sixteen
section names, each with the first line of its prose. Called with a section name it returns that
section. Every response carries the full name list, so a caller that guessed wrong does not need
an extra call to find the right one, and an unknown name is answered with the outline rather than
an error.

It is a tool parameter rather than one resource per section because a resource is read when the
application decides to read it, and a client that surfaces only tools would not see the skill at
all.

The prompt `compose-imagor-image` still carries the whole document — a user who picks it from a
menu has asked for it.
