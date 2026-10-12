#!/usr/bin/env python3
"""Rewrite a recipe's geometry as percentage+offset.

The authoring run writes absolute pixels, which match the reference exactly but do
not adapt - and the cookbook test renders every recipe at three canvas sizes, so
absolute geometry fails there. A bare percentage must be a whole number, losing up
to 9px at 1800 wide; percentage plus offset is exact at the reference size and
proportional anywhere else: 790 on 1800 is 43p+16, which is 774+16.

Inner slots are relative to their own canvas, not the banner, so they convert
against the canvas the outer slot defines.
"""
import json, os, sys

REF_W, REF_H = 1800, 942


def form(v, total):
    pct = int(v / total * 100)
    rem = int(v) - round(pct / 100 * total)
    return f"{pct}p+{rem}" if rem else f"{pct}p"


def convert(slot, cw, ch):
    # A nested canvas is its own coordinate space, so its inner slots are converted
    # against the canvas's pixel size - read before the canvas itself becomes
    # relative, or the inner geometry is measured against the wrong thing.
    if slot.get("slots"):
        nw = int(slot["width"]) if str(slot.get("width", "")).isdigit() else cw
        nh = int(slot["height"]) if str(slot.get("height", "")).isdigit() else ch
        for inner in slot["slots"]:
            for key, total in (("x", nw), ("width", nw), ("y", nh), ("height", nh)):
                v = inner.get(key)
                if isinstance(v, str) and v.isdigit():
                    inner[key] = form(int(v), total)
    for key, total in (("x", cw), ("width", cw), ("y", ch), ("height", ch)):
        v = slot.get(key)
        if isinstance(v, str) and v.isdigit():
            slot[key] = form(int(v), total)
    return slot


def main():
    path = sys.argv[1]
    d = json.load(open(path))
    c = d["canvas"]
    cw, ch = (c[0], c[1]) if isinstance(c, list) else (c["width"], c["height"])
    for slot in d["slots"]:
        convert(slot, cw, ch)
    json.dump(d, open(path, "w"), indent=2, ensure_ascii=False)
    open(path, "a").write("\n")
    print(f"  {len(d['slots'])} slots converted against {cw}x{ch}")
    for s in d["slots"]:
        print(f"    {s['name']:14} x={str(s.get('x')):10} y={str(s.get('y')):10} "
              f"w={str(s.get('width')):10} h={str(s.get('height')):8}"
              + (f"  ({len(s['slots'])} inner)" if s.get("slots") else ""))


main()
