#!/usr/bin/env python3
"""Rebuild product-grid with percentage+offset geometry.

The reference is 1800x942. Absolute pixels match it exactly but do not adapt, and
the cookbook test renders every recipe at three canvas sizes - so absolute geometry
fails there, correctly. A bare percentage adapts but must be a whole number, which
loses up to 9px at this width. Percentage plus offset gives both: 790 on 1800 is
43p+16, which is 774+16 at the reference size and stays proportional at any other.

Layout technique comes from the authoring run, which measured label centres to
within 0.5px of their product centres.
"""
import json, os

SRC = os.path.expanduser("~/Developer/imagor-mcp/cookbook")
IMG = "http://localhost:8198"
W, H = 1800, 942


def form(v, axis):
    total = W if axis == "x" else H
    pct = int(v / total * 100)
    rem = int(v) - round(pct / 100 * total)
    return f"{pct}p+{rem}" if rem else f"{pct}p"


def fx(v):
    return form(v, "x")


def fy(v):
    return form(v, "y")


# From the reference: image x, y, w, h. box is the label box width - wider than
# the product and centred on it, which is what makes a label read as centred.
PRODUCTS = {
    "c1r1": dict(x=790, y=60, w=265, h=240, box=300),
    "c2r1": dict(x=1455, y=100, w=265, h=185, box=300),
    "c1r2": dict(x=720, y=655, w=290, h=175, box=320),
    "c2r2": dict(x=1090, y=590, w=310, h=200, box=340),
    "c3r2": dict(x=1460, y=560, w=280, h=240, box=310),
}

slots = [
    {"name": "photo", "kind": "image", "image": f"{IMG}/src_life.png",
     "x": fx(0), "y": fy(0), "width": fx(694), "height": "full", "fit": True},
    {"name": "mark", "kind": "text", "text": "{{mark}}",
     "x": fx(1050), "y": fy(228), "width": fx(350), "height": fy(120),
     "font": "serif", "color": "111111", "max_size": 92, "min_size": 60},
]

for key, p in PRODUCTS.items():
    bx = p["x"] + p["w"] // 2 - p["box"] // 2
    slots += [
        {"name": f"{key}_img", "kind": "image", "image": f"{IMG}/src_p{key[1]}.png",
         "x": fx(p["x"]), "y": fy(p["y"]), "width": fx(p["w"]), "height": fy(p["h"]),
         "fit": True},
        {"name": f"{key}_name", "kind": "text", "text": "{{" + key + "_name}}",
         "x": fx(bx), "y": fy(p["y"] + p["h"] + 18), "width": fx(p["box"]), "height": fy(52),
         "font": "sans", "color": "9a9a9a", "max_size": 17, "min_size": 12},
        {"name": f"{key}_price", "kind": "text", "text": "{{" + key + "_price}}",
         "x": fx(bx), "y": fy(p["y"] + p["h"] + 78), "width": fx(p["box"]), "height": fy(34),
         "font": "sans-bold", "color": "111111", "max_size": 22, "min_size": 16},
    ]

recipe = {
    "name": "Product grid banner",
    "note": ("Five products beside a full-height photo, wordmark between the rows. "
             "Geometry is percentage+offset, so it matches the reference at 1800x942 "
             "and stays proportional at another size."),
    "canvas": [W, H],
    "background": "color:ffffff",
    "slots": slots,
    "values": {
        "mark": "MAISON",
        "c1r1_name": "Padded leather shoulder bag",     "c1r1_price": "HKD$18500",
        "c2r1_name": "Coated-canvas messenger bag",     "c2r1_price": "HKD$11540",
        "c1r2_name": "Leather messenger bag",           "c1r2_price": "HKD$12875",
        "c2r2_name": "Debossed leather messenger bag",  "c2r2_price": "HKD$13625",
        "c3r2_name": "Logo-embossed leather tote bag",  "c3r2_price": "HKD$15170",
    },
}

p = os.path.join(SRC, "product-grid.json")
json.dump(recipe, open(p, "w"), indent=2, ensure_ascii=False)
open(p, "a").write("\n")
print(f"  {len(slots)} slots")
for key, pr in PRODUCTS.items():
    print(f"    {key}  img x={fx(pr['x']):10} y={fy(pr['y']):9}  (790 -> {fx(790)})")
