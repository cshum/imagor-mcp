#!/usr/bin/env python3
"""Generate the Imagor MCP page for docs.imagor.net from this repository's README.

The page is the README, section for section and paragraph for paragraph. Two things
change on the way: links into this repository become absolute GitHub URLs, because the
page is read on a docs site rather than in the repository, and a link back to
docs.imagor.net becomes relative, because the page is already there. The docs-site
frontmatter, the badges and the info box are added.

It was written by hand until now, which is how it drifted three times in one day.
Anything that should appear on that page and is not in the README is written in the
README first.

    python3 scripts/sync-docs-page.py                     print the page
    python3 scripts/sync-docs-page.py -w <path>           write it
    python3 scripts/sync-docs-page.py --check <path>      exit 1 if it differs
"""

import argparse
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
REPO = "https://github.com/cshum/imagor-mcp"
BLOB = f"{REPO}/blob/main"
SHORT = REPO.removeprefix("https://")

# The sections the page mirrors, in the order the README has them. Adding a section to
# the README means adding it here; that is the one thing this script cannot infer.
SECTIONS = [
    ("Quick start", "What an agent does with it"),
    ("What an agent does with it", "What this adds"),
    ("What this adds", "Tools"),
    ("Tools", "The cookbook"),
    ("The cookbook", "Going deeper"),
    ("Going deeper", "Licence"),
]

INTRO_END = "[![Cards"

# Pages the README links to on this docs site, and the file each slug is. Relative, because
# the page is served from there.
SITE_PAGES = {
    "security": "./security.mdx",
}


def section(readme, title, nxt):
    a = readme.index(f"## {title}") + len(f"## {title}")
    b = readme.index(f"## {nxt}") if nxt else len(readme)
    return readme[a:b].strip()


def to_docs_site(text):
    """Repository-relative links become absolute; a link back to the site is relative."""
    text = re.sub(r"\]\(docs/([^)]+)\)", lambda m: f"]({BLOB}/docs/{m.group(1)})", text)
    text = re.sub(r"\]\(cookbook/([^)]+)\)", lambda m: f"]({BLOB}/cookbook/{m.group(1)})", text)
    text = re.sub(r"\]\((CONTRIBUTING\.md)\)", lambda m: f"]({BLOB}/{m.group(1)})", text)
    # The page is served by docs.imagor.net, so a README link to that site sends the
    # reader out and back for a page one hop away. The slug is not always the filename,
    # so this is a table; an unmapped one is an error rather than a silent absolute link.
    for slug, target in SITE_PAGES.items():
        text = text.replace(f"https://docs.imagor.net/{slug}", target)
    left = re.search(r"https?://docs\.imagor\.net/\S*", text)
    if left:
        raise SystemExit(f"add {left.group(0)} to SITE_PAGES: the page is on that site already")
    return text


def build():
    readme = (ROOT / "README.md").read_text()
    # Everything between the title and the contact sheet: the subtitle and the line under it.
    # Found by position, not by wording, so rewording or unbolding them cannot break this.
    intro = readme[readme.index("\n", readme.index("# Imagor MCP")):readme.index(INTRO_END)].strip()
    parts = [
        "---",
        "slug: /mcp",
        "description: Extends imagor with template capability for agents, compiling a layout to one imagor URL at any canvas size.",
        "keywords:",
        "  - imagor-mcp",
        "  - Imagor MCP",
        "  - imagor templates",
        "  - imagor MCP server",
        "  - imagor agent skill",
        "  - imagor image composition",
        "---",
        "",
        "# Imagor MCP",
        "",
        f"[![Test Status]({REPO}/workflows/ci/badge.svg)]({REPO}/actions/workflows/ci.yml)",
        "[![MCP Registry](https://img.shields.io/badge/registry-io.github.cshum%2Fimagor--mcp-white)]"
        "(https://registry.modelcontextprotocol.io/)",
        "",
        intro,
        "",
        ":::info",
        f"**GitHub:** [{SHORT}]({REPO})  ",
        "**Registry:** [registry.modelcontextprotocol.io](https://registry.modelcontextprotocol.io/) "
        "— install as `io.github.cshum/imagor-mcp`  ",
        f"**Docker:** [ghcr.io/cshum/imagor-mcp]({REPO}/pkgs/container/imagor-mcp)  ",
        f"**Releases:** [macOS, Linux and Windows binaries]({REPO}/releases)  ",
        "**Requires:** imagor **1.9.10**",
        ":::",
        "",
        "[![Cards, each one a single imagor URL]"
        "(https://raw.githubusercontent.com/cshum/imagor-mcp/main/docs/gallery/contact-sheet.png)]"
        f"({BLOB}/docs/cookbook.md)",
        "",
        f"*[See the recipes »]({BLOB}/docs/cookbook.md)*",
        "",
    ]
    for title, nxt in SECTIONS:
        parts += [f"## {title}", "", to_docs_site(section(readme, title, nxt)), ""]
    parts += [f"Everything above is the [imagor-mcp README]({REPO}#readme), kept in step with it.", ""]
    return "\n".join(parts)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("-w", "--write", metavar="PATH")
    ap.add_argument("--check", metavar="PATH", help="fail if the file differs from the README")
    args = ap.parse_args()
    page = build()
    if args.check:
        target = pathlib.Path(args.check)
        if not target.exists():
            print(f"{target} does not exist")
            return 1
        current = target.read_text()
        if current != page:
            print(f"{target} differs from the README. Run: scripts/sync-docs-page.py -w {target}")
            for line in __import__("difflib").unified_diff(
                    current.splitlines(), page.splitlines(), "on disk", "generated", lineterm="", n=1):
                print(line)
            return 1
        print(f"{target} matches the README")
        return 0
    if args.write:
        pathlib.Path(args.write).write_text(page)
        print(f"wrote {args.write}")
        return 0
    sys.stdout.write(page)
    return 0


if __name__ == "__main__":
    sys.exit(main())
