#!/usr/bin/env python3
"""Generate the cookbook's source fixtures.

Every fixture is a pure-vector SVG: rectangles, circles, paths and gradients, and no
<text> anywhere. Text in an SVG is rendered by whatever font the imagor you point at
happens to carry, and an absent face falls back silently, so a fixture containing type
would look right here and wrong somewhere else. Shapes have no such dependency.

Author in pixels, never in physical units. An SVG with width="2.667in" has no intrinsic
pixel size: imagor reads it at 72dpi and returns 192x144, not the 267x200 anyone meant.
With width="800" height="600" the intrinsic size is the number in the file, imagor
rasterises crisply at any size asked for, and manifest.py can record the dimensions by
reading them rather than measuring a bitmap.

Sizes are deliberately small. A few hundred bytes each, so the whole set is a few KB:
small enough to embed in the binary, diff in a pull request, and serve from anywhere.

    python3 generate.py            # write the fixtures and manifest.json
"""

import argparse
import json
import os
import textwrap

HERE = os.path.dirname(os.path.abspath(__file__))

# Where the fixtures are served. The repo layout maps to the URL one for one: a file at
# sources/product-bag.svg is served at <base>/product-bag.svg, which is what makes the
# deployed set and the committed set checkable against each other.
DEFAULT_BASE = "https://s.imagor.net"


def svg(width, height, body):
    """A fixture: pixel dimensions, and a viewBox that matches them."""
    return textwrap.dedent(f"""\
        <svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" \
viewBox="0 0 {width} {height}">
        {textwrap.indent(textwrap.dedent(body).strip(), '  ')}
        </svg>
        """)


# A palette shared by the fixtures, so a cookbook card and its source look related.
CREAM, NAVY, GREEN, TAN, SLATE = "#f6f4ee", "#0f172a", "#173f34", "#c9b79c", "#2b3441"


def product(w, h, ground, body, accent):
    """A product shape on a plain ground: what a grid cell or a product card wants."""
    return svg(w, h, f"""
        <rect width="{w}" height="{h}" fill="{ground}"/>
        <path d="{body}" fill="{accent}"/>
        <rect x="{w*0.18:.0f}" y="{h*0.78:.0f}" width="{w*0.64:.0f}" height="{h*0.02:.0f}" \
fill="{accent}" opacity="0.18"/>
        """)


def portrait(size, ground, accent):
    """An abstract portrait: a head and shoulders as geometry, no face and no letters."""
    r = size / 2
    return svg(size, size, f"""
        <rect width="{size}" height="{size}" fill="{ground}"/>
        <circle cx="{r:.0f}" cy="{size*0.40:.0f}" r="{size*0.19:.0f}" fill="{accent}"/>
        <path d="M {size*0.20:.0f} {size} Q {r:.0f} {size*0.60:.0f} {size*0.80:.0f} {size} Z" \
fill="{accent}"/>
        """)


def texture(w, h, stops):
    """A gradient: for cards that want a ground with depth rather than a flat colour."""
    grad = "".join(
        f'<stop offset="{o}" stop-color="{c}"/>' for o, c in stops
    )
    return svg(w, h, f"""
        <defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1">{grad}</linearGradient></defs>
        <rect width="{w}" height="{h}" fill="url(#g)"/>
        """)


def scene(w, h, ground, accent):
    """A tall panel with a subject: what a hero or a launch card puts beside its copy."""
    return svg(w, h, f"""
        <rect width="{w}" height="{h}" fill="{ground}"/>
        <circle cx="{w*0.5:.0f}" cy="{h*0.34:.0f}" r="{min(w,h)*0.20:.0f}" fill="{accent}"/>
        <rect x="{w*0.30:.0f}" y="{h*0.55:.0f}" width="{w*0.40:.0f}" \
height="{h*0.30:.0f}" rx="{w*0.06:.0f}" fill="{accent}" opacity="0.85"/>
        """)


# name, kind, what it is for, the file
FIXTURES = {
    # Products: three shapes, so a grid has something to arrange.
    "product-bag": ("product", "Soft and rounded. The default for a grid cell.",
                    product(400, 400, CREAM, "M 90 150 Q 200 110 310 150 L 320 330 L 80 330 Z",
                            "#2f2a26")),
    "product-box": ("product", "Rigid with a lid. Sits well beside a bag.",
                    product(400, 400, CREAM, "M 100 140 L 300 140 L 300 330 L 100 330 Z",
                            "#5a4a3a")),
    "product-bottle": ("product", "Tall and narrow. A different aspect to the others.",
                       product(300, 480, CREAM, "M 120 120 L 180 120 L 185 200 L 195 460 "
                                                "L 105 460 L 115 200 Z", "#2b4a3f")),

    # Portraits: geometry, so no face is being faked and no font is needed.
    "portrait-a": ("portrait", "A dark ground and a light subject.",
                   portrait(400, GREEN, "#c9b79c")),
    "portrait-b": ("portrait", "A light ground and a dark subject.",
                   portrait(400, TAN, "#173f34")),
    "portrait-c": ("portrait", "A grey ground and a warm subject.",
                   portrait(400, SLATE, "#c36c4a")),

    # Textures: for cards that want depth behind their copy.
    "texture-navy": ("texture", "A dark diagonal gradient, for a hero ground.",
                     texture(1200, 800, [("0", NAVY), ("1", "#1e3a5f")])),
    "texture-forest": ("texture", "A deep green diagonal gradient, the house ground.",
                       texture(1200, 800, [("0", "#142e2b"), ("1", "#20443c")])),
    "texture-warm": ("texture", "A warm gradient, light to dark.",
                     texture(1200, 800, [("0", CREAM), ("1", TAN)])),

    # Scenes: a subject in a frame, for the cards that place copy beside an image.
    "scene-tall": ("scene", "A subject in a tall frame, for a launch card.",
                   scene(700, 900, CREAM, "#173f34")),
    "scene-wide": ("scene", "A subject in a wide frame, for a banner.",
                   scene(1600, 600, NAVY, "#c9b79c")),

    # Deliberately awkward: these exist to break things, not to look good.
    "tiny": ("edge", "A fit has to upscale it, and fit-in will refuse. Smaller than any box.",
             texture(64, 64, [("0", GREEN), ("1", TAN)])),
    "huge": ("edge", "Much larger than any box, so a fit has to downscale.",
             texture(4000, 4000, [("0", NAVY), ("1", "#c36c4a")])),
    "wide": ("edge", "An extreme aspect. Exposes letterboxing.",
             texture(1600, 200, [("0", SLATE), ("1", CREAM)])),
    "tall": ("edge", "The other extreme.",
             texture(200, 1600, [("0", TAN), ("1", GREEN)])),
    "square": ("edge", "A shape no other fixture has, for testing crop.",
               texture(600, 600, [("0", "#c36c4a"), ("1", NAVY)])),
    "transparent": ("edge", "No ground at all. Tests what a card does over nothing.",
                    svg(400, 400, f'<circle cx="200" cy="200" r="140" fill="{GREEN}" '
                                  f'opacity="0.75"/>')),
}


def main():
    ap = argparse.ArgumentParser(description="Write the fixtures and manifest.json")
    ap.add_argument("--base", default=DEFAULT_BASE,
                    help="public base URL for the fixtures")
    args = ap.parse_args()
    base = args.base.rstrip("/")

    manifest = []
    for name, (kind, note, body) in sorted(FIXTURES.items()):
        path = os.path.join(HERE, name + ".svg")
        with open(path, "w") as f:
            f.write(body)
        # The dimensions are the ones written in the file, so read them back rather
        # than trusting the call site.
        head = body.split(">", 1)[0]
        w = int(head.split('width="')[1].split('"')[0])
        h = int(head.split('height="')[1].split('"')[0])
        manifest.append({
            "name": name, "file": name + ".svg", "kind": kind,
            "url": f"{base}/{name}.svg",
            "width": w, "height": h,
            "aspect": round(w / h, 3), "note": note,
        })
        print(f"  {name:14} {w:>5}x{h:<5} {os.path.getsize(path):>5} bytes  {kind}")

    with open(os.path.join(HERE, "manifest.json"), "w") as f:
        json.dump(manifest, f, indent=2)
        f.write("\n")
    total = sum(os.path.getsize(os.path.join(HERE, m["file"])) for m in manifest)
    print(f"\n  {len(manifest)} fixtures, {total} bytes total")


if __name__ == "__main__":
    main()
