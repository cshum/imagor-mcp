# The template format


A template carries **no canvas size**. Geometry is relative and text with a `height` is
**fitted** to its box, so the same layout serves 1200×630 and 1080×1080 — write it once,
render it anywhere. That is what makes a layout worth keeping.

The rest of the format — nested canvases, filters on a slot's own source, what a path can
and cannot carry — is in [the skill](skill.md), where the reader is an agent that has to write one.

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

