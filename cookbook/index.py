"""Generate the cookbook index the server embeds, from the recipes themselves.

The index is what an agent reads to choose a layout: the name, what it is for, its
size, and which values it takes. It is generated here rather than written by hand so
it cannot drift from the recipes - a recipe added, renamed or given a new value shows
up in the index the next time this runs.
"""
import glob
import io
import json
import os

rows = []
for f in sorted(glob.glob("cookbook/*.json")):
    d = json.load(io.open(f, encoding="utf-8"))
    c = d["canvas"]
    w, h = (c["width"], c["height"]) if isinstance(c, dict) else (c[0], c[1])
    rows.append((
        os.path.basename(f)[:-5],
        d.get("note", "").strip(),
        w, h,
        sorted((d.get("values") or {}).keys()),
    ))

out = [
    "# Cookbook",
    "",
    "Fourteen ready-made layouts. **Read this to choose one, then fetch it by name.**",
    "A recipe is the channel between you and `render_template` - it is not the product,",
    "and it is not something to reproduce by hand.",
    "",
    "```",
    'get_cookbook_recipe  name: "event"      ->  the template, ready for render_template',
    "```",
    "",
    "Every recipe renders at three canvas sizes in CI, so one that no longer works cannot",
    "reach this list. Their images come from `s.imagor.net` and render as they are.",
    "",
    "| name | what it is | size | values to fill |",
    "|---|---|---|---|",
]
for name, note, w, h, vals in rows:
    vs = ", ".join("`%s`" % v for v in vals) if vals else "—"
    out.append("| `%s` | %s | %dx%d | %s |" % (name, note, w, h, vs))
out += [
    "",
    "## Choosing one",
    "",
    "Start from the closest recipe and change the values rather than composing from",
    "nothing: they carry spacing and type sizes that took iteration to get right. If none",
    "fits, build from scratch - the skill covers the path grammar and the method.",
    "",
]

io.open("references/cookbook.md", "w", encoding="utf-8").write("\n".join(out))
print("  references/cookbook.md: %d recipes" % len(rows))
