"""Find text that overlaps an image, across every recipe.

A gap is fine: a fitted image centres in its box and a caption sits below it. An overlap
is not, because where text meets an image the words lose - the reader sees a caption
running into a photograph.

What is legitimate, and excluded:

  - a rect behind text. A coloured panel is meant to sit under a heading.
  - a full-canvas image behind everything. That is a background.
  - an image behind text, where the image is a ground rather than a subject - the
    distinction cannot be made from geometry, so it is left to the caller to name one.

Text is found wherever it is: a slot of kind text, or a text slot inside a nested canvas,
which is how a caption is written when it needs centring. Inner slots resolve against
their canvas's box, so they are translated before being compared.

    python3 overlap.py <cookbook dir>
"""

import glob
import json
import os
import sys


def px(v, total):
    """Resolve a template geometry form to pixels. Reading only, not rendering."""
    if v is None:
        return None
    s = str(v)
    if s in ("full", "f"):
        return total
    if s.startswith("full-"):
        return total - int(s[5:])
    if s in ("half", "h"):
        return total // 2
    if s in ("third", "t"):
        return total // 3
    if s in ("quarter", "q"):
        return total // 4
    if "p+" in s:
        p, off = s.split("p+")
        return round(int(p) / 100 * total) + int(off)
    if s.endswith("p"):
        return round(int(s[:-1]) / 100 * total)
    if s.startswith("0."):
        return round(float(s) * total)
    try:
        return int(float(s))
    except Exception:
        return None


def box(s, W, H):
    x = px(s.get("x"), W) or 0
    y = px(s.get("y"), H) or 0
    w = px(s.get("width"), W)
    h = px(s.get("height"), H)
    return (x, y, x + (w or 0), y + (h or 0))


def overlaps(a, b):
    return not (a[2] <= b[0] or b[2] <= a[0] or a[3] <= b[1] or b[3] <= a[1])


def hits_centre(a, b):
    """Does box a cover the centre of box b?

    A fitted image centres in its box and fills only part of it, so a text box can
    overlap the image's box by a little and still clear the picture. Comparing boxes
    reports those as faults; comparing ink needs each source's aspect ratio, which a
    photograph's URL does not carry.

    The centre is the one point that is always inside the ink, whatever the source's
    shape and whichever way fit-in letterboxes it. A text box covering an image's centre
    is a caption sitting on a picture, which is the fault worth failing on.
    """
    cx, cy = (b[0] + b[2]) / 2, (b[1] + b[3]) / 2
    return a[0] <= cx <= a[2] and a[1] <= cy <= a[3]


def collect(slots, ox, oy, W, H):
    """Flatten a recipe to (texts, subjects), translating nested canvases.

    W and H are the canvas these slots lay out on - the recipe's for the top level, and
    the carrying slot's own box for anything nested. Resolving an inner slot against the
    outer canvas makes every percentage inside a canvas wrong by its scale factor.
    """
    texts, subjects = [], []
    for s in slots:
        x, y = px(s.get("x"), W) or 0, px(s.get("y"), H) or 0
        kind = s.get("kind") or "text"
        b = box(s, W, H)
        b = (b[0] + ox, b[1] + oy, b[2] + ox, b[3] + oy)
        if s.get("text"):
            texts.append((s.get("name", "?"), b))
        if s.get("image") and kind == "image":
            # A full-canvas image is a background, not a subject.
            if not (px(s.get("width"), W) == W and px(s.get("height"), H) == H):
                subjects.append((s.get("name", "?"), b))
        if s.get("slots"):
            # The inner canvas is this slot's own box.
            iw = px(s.get("width"), W) or W
            ih = px(s.get("height"), H) or H
            it, isub = collect(s["slots"], x + ox, y + oy, iw, ih)
            texts += it
            subjects += isub
    return texts, subjects


def canvas_of(d):
    c = d.get("canvas")
    if isinstance(c, list):
        return c[0], c[1]
    if isinstance(c, dict):
        return c.get("width", 1200), c.get("height", 630)
    return d.get("canvas_width", 1200), d.get("canvas_height", 630)


def main():
    root = sys.argv[1] if len(sys.argv) > 1 else "cookbook"
    bad = 0
    checked = 0
    for path in sorted(glob.glob(os.path.join(os.path.expanduser(root), "*.json"))):
        name = os.path.splitext(os.path.basename(path))[0]
        d = json.load(open(path))
        W, H = canvas_of(d)
        texts, subjects = collect(d.get("slots", []), 0, 0, W, H)
        checked += len(texts)
        for tn, tb in texts:
            if tb[2] == tb[0] or tb[3] == tb[1]:
                continue  # no explicit box: nothing to compare against
            for sn, sb in subjects:
                if hits_centre(tb, sb):
                    ow = min(tb[2], sb[2]) - max(tb[0], sb[0])
                    oh = min(tb[3], sb[3]) - max(tb[1], sb[1])
                    print(f"  {name:18} text {tn:18} covers the centre of {sn:18} "
                          f"(boxes overlap {ow}x{oh}px)")
                    bad += 1
    print(f"\n  {checked} text slots checked, {bad} overlaps")


if __name__ == "__main__":
    main()
