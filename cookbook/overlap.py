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


# Not *.json: two tests glob cookbook/*.json and read every match as a recipe.
CACHE_PATH = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".aspect-cache")


def load_cache():
    try:
        with open(CACHE_PATH) as f:
            return json.load(f)
    except Exception:
        return {}


def save_cache(cache):
    try:
        with open(CACHE_PATH, "w") as f:
            json.dump(cache, f, indent=2, sort_keys=True)
            f.write("\n")
    except Exception:
        pass


def source_aspect(url, cache, manifest):
    """The source's width/height, or None when it cannot be known offline.

    A fixture is in the manifest. A URL with a width and no height returns the source's
    own aspect, so fetching it once tells us the shape - cached, because the check should
    not depend on the network to run.
    """
    if url in cache:
        return cache[url]
    for entry in manifest:
        if entry.get("url") == url:
            cache[url] = entry.get("aspect")
            return cache[url]
    if "w=" in url and "h=" not in url:
        try:
            import urllib.request, io
            req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
            with urllib.request.urlopen(req, timeout=20) as r:
                data = r.read()
            from PIL import Image
            im = Image.open(io.BytesIO(data))
            cache[url] = round(im.width / im.height, 4)
            return cache[url]
        except Exception:
            return None
    return None


def ink_box(box, aspect):
    """Where a fitted image's pixels actually land inside its box.

    fit-in centres the picture and fills only part of the box, so the ink is the box
    narrowed to the source's aspect. Without an aspect there is nothing to narrow by, and
    the box is returned - which is the conservative reading.
    """
    x1, y1, x2, y2 = box
    bw, bh = x2 - x1, y2 - y1
    if not aspect or bw <= 0 or bh <= 0:
        return box
    if bw / bh > aspect:          # box is wider than the source: letterbox left/right
        iw = bh * aspect
        return (x1 + (bw - iw) / 2, y1, x1 + (bw + iw) / 2, y2)
    ih = bw / aspect              # box is taller: letterbox top/bottom
    return (x1, y1 + (bh - ih) / 2, x2, y1 + (bh + ih) / 2)


def hits_ink(a, b):
    """Does text box a run into the picture in box b?

    b is the ink box - the picture's own pixels inside its slot, computed from the
    source's aspect - so any overlap is a real collision. Comparing the slot box instead
    was wrong twice over: it reported a caption sitting in a fitted image's letterbox as
    a fault, and it missed a caption running into a portrait source, whose ink fills the
    full height and leaves no margin to sit in.
    """
    if not (a[2] <= b[0] or b[2] <= a[0] or a[3] <= b[1] or b[3] <= a[1]):
        return "runs into"
    return None


def substitute(value, values):
    """Resolve {{name}} against the recipe's values, as the renderer does."""
    if not isinstance(value, str):
        return value
    out = value
    for k, v in (values or {}).items():
        out = out.replace("{{" + k + "}}", str(v))
    return out


def collect(slots, ox, oy, W, H, values=None):
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
        src = substitute(s.get("image"), values)
        if src and kind == "image" and not str(src).startswith("color:"):
            # A full-canvas image is a background and a color: value is a panel; neither
            # is a picture. What is left is a source a fit will letterbox, which is the
            # only thing a caption can collide with.
            if not (px(s.get("width"), W) == W and px(s.get("height"), H) == H):
                subjects.append((s.get("name", "?"), b, src))
        if s.get("slots"):
            # The inner canvas is this slot's own box.
            iw = px(s.get("width"), W) or W
            ih = px(s.get("height"), H) or H
            it, isub = collect(s["slots"], x + ox, y + oy, iw, ih, values)
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
    cache = load_cache()
    manifest = []
    mpath = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "sources", "manifest.json")
    try:
        with open(mpath) as f:
            manifest = json.load(f)
    except Exception:
        pass
    for path in sorted(glob.glob(os.path.join(os.path.expanduser(root), "*.json"))):
        name = os.path.splitext(os.path.basename(path))[0]
        d = json.load(open(path))
        W, H = canvas_of(d)
        texts, subjects = collect(d.get("slots", []), 0, 0, W, H, d.get("values"))
        checked += len(texts)
        for tn, tb in texts:
            if tb[2] == tb[0] or tb[3] == tb[1]:
                continue  # no explicit box: nothing to compare against
            for sn, sb, surl in subjects:
                aspect = source_aspect(surl, cache, manifest)
                sb = ink_box(sb, aspect)
                why = hits_ink(tb, sb)
                if why:
                    ow = min(tb[2], sb[2]) - max(tb[0], sb[0])
                    oh = min(tb[3], sb[3]) - max(tb[1], sb[1])
                    print(f"  {name:18} text {tn:18} {why} {sn:18} "
                          f"(boxes overlap {ow}x{oh}px)")
                    bad += 1
    save_cache(cache)
    print(f"\n  {checked} text slots checked, {bad} overlaps")


if __name__ == "__main__":
    main()
