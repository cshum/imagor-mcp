# The skill

The server carries a skill, served at `imagor-mcp://skill`. It carries what a tool description cannot, and an agent that skips it writes worse layouts:

- **The path grammar** — the order a path is written in, and the two crops whose orders
  are opposite.
- **Where text lands** — placement is a *search*, not a calculation, and the skill says so.
- **Awkward values** — colours, and what has to be encoded.
- **A working method** — build small pieces, check each, then assemble.

It travels with the server, so a client that reads only the tool list never sees it.
`get_imagor_skill` returns it for a client that reads tools rather than resources.

### Served in sections

The document is 18 KB and most of it is irrelevant to any one composition, so the tool
serves it in two steps. Called with no section it returns the frontmatter and the
outline: the sixteen section names, each with the first line of its prose. Called with
a section name it returns that section. Every response carries the full name list, so a
caller that guessed wrong does not need an extra call to find the right one, and an
unknown name is answered with the outline rather than an error.

It is a tool parameter rather than one resource per section because a resource is read when
the application decides to read it, and a client that surfaces only tools would not see the
skill at all.

The prompt `compose-imagor-image` still carries the whole document — a user who picks it
from a menu has asked for it.

