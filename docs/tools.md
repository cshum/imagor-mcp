# Tools

The same eleven the server answers `tools/list` with. Every parameter maps onto a
concept in the [imagor path grammar](format.md) — there are no invented parameters, and
signing happens only as the by-product of a typed tool.

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

A client discovers these at runtime; this page is the same list written down.
