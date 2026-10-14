# The skill

The server carries a skill, served at `imagor-mcp://skill`. It is the part of using
this that does not fit in a tool description, and an agent that skips it composes badly:

- **The path grammar** — the order a path is written in, and the two crops whose orders
  are opposite.
- **Where text lands** — placement is a *search*, not a calculation, and the skill says so.
- **Awkward values** — colours, and what has to be encoded.
- **A working method** — build small pieces, check each, then assemble.

It travels with the server, so a client that reads only the tool list never sees it.
`get_imagor_skill` returns it for a client that reads tools rather than resources.

