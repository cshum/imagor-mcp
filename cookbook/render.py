#!/usr/bin/env python3
"""Render every cookbook recipe and build a contact sheet.

Handles both canvas spellings, which the recipes legitimately use:
    "canvas": [1200, 628], "background": "color:0f172a"
    "canvas": {"width": 1200, "height": 630, "background": "color:ffffff"}

Getting this wrong is quiet: a recipe falls back to a default size and background
and still renders, so it looks like a layout problem rather than a parsing one.
"""
import io, json, os, subprocess, sys, urllib.request
from PIL import Image, ImageDraw

# Render against the imagor image, not a host binary.
#
# The recipes name Liberation Serif and Open Sans, and those families live in the
# published image at /usr/share/fonts/truetype. A binary built for macOS does not carry
# them and substitutes an Apple face silently - so six different family names came back
# byte-identical, and every card looked like one font. The substitution is invisible in
# the output, which is why this has to be the image rather than whatever is installed.
#
# Point this at the image, run with IMAGOR_UNSAFE=1 (the published image is configured
# that way), and use the tag the recipes target:
#
#   docker run -d -p 8197:8000 -e IMAGOR_UNSAFE=1 ghcr.io/cshum/imagor:1.9.9
MCP = os.environ.get("IMAGOR_MCP_BIN", "imagor-mcp")
BASE = os.environ.get("IMAGOR_BASE_URL", "http://localhost:8197")
# The recipes live beside this script. An absolute path here works on one machine
# and nowhere else - CI checked out to /home/runner/work, not ~/Developer, and the
# job failed on a directory that only exists on a laptop.
SRC = os.environ.get("IMAGOR_COOKBOOK_DIR", os.path.dirname(os.path.abspath(__file__)))
OUT = sys.argv[1] if len(sys.argv) > 1 else "/tmp/gallery3"


def canvas_of(recipe):
    c = recipe.get("canvas")
    if isinstance(c, list) and len(c) == 2:
        return int(c[0]), int(c[1]), recipe.get("background", "color:ffffff")
    if isinstance(c, dict):
        return (int(c.get("width", 1200)), int(c.get("height", 630)),
                c.get("background") or recipe.get("background") or "color:ffffff")
    return 1200, 630, recipe.get("background", "color:ffffff")


def main():
    # The published image runs unsafe, and the binary refuses a secret alongside
    # IMAGOR_UNSAFE - both set makes it exit without answering, which surfaces as a
    # bare "no result" per recipe rather than as a configuration error.
    env = {k: v for k, v in os.environ.items() if not k.startswith("IMAGOR_")}
    env["IMAGOR_UNSAFE"] = "1"
    env["IMAGOR_BASE_URL"] = BASE
    p = subprocess.Popen([MCP], stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                         stderr=subprocess.DEVNULL, text=True, bufsize=1, env=env)

    def rpc(o):
        """Send a request and return its reply.

        The reader took the next line whatever it was, so a notification
        arriving before the reply was parsed as the reply and reported as a
        failed call.
        """
        p.stdin.write(json.dumps(o) + "\n"); p.stdin.flush()
        while True:
            line = p.stdout.readline()
            if not line.strip():
                return {}
            m = json.loads(line)
            if "id" in m:
                return m

    rpc({"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {
        "protocolVersion": "2025-06-18", "capabilities": {},
        "clientInfo": {"name": "cookbook", "version": "1"}}})
    p.stdin.write(json.dumps({"jsonrpc": "2.0",
                              "method": "notifications/initialized"}) + "\n")
    p.stdin.flush()

    os.makedirs(OUT, exist_ok=True)
    ok, failed = 0, []
    for f in sorted(os.listdir(SRC)):
        if not f.endswith(".json") or f.startswith("_"):
            continue
        recipe = json.load(open(os.path.join(SRC, f)))
        w, h, bg = canvas_of(recipe)
        r = rpc({"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {
            "name": "render_template", "arguments": {
                "canvas_width": w, "canvas_height": h, "background": bg,
                "slots": recipe.get("slots", []), "values": recipe.get("values", {})}}})
        if "result" not in r:
            failed.append(f[:-5]); print(f"  {f[:-5]:20} ERR"); continue
        raw = r["result"]["content"][0]["text"]
        try:
            t = json.loads(raw)
        except json.JSONDecodeError:
            failed.append(f[:-5]); print(f"  {f[:-5]:20} {raw[:80]}"); continue
        url = t.get("absolute_url") or (BASE + "/" + t.get("url", ""))
        try:
            with urllib.request.urlopen(url, timeout=60) as rr:
                im = Image.open(io.BytesIO(rr.read())).convert("RGB")
            im.save(os.path.join(OUT, f[:-5] + ".png"))
            ok += 1
            print(f"  {f[:-5]:20} {w}x{h}  {bg}")
        except Exception as e:
            failed.append(f[:-5]); print(f"  {f[:-5]:20} fetch {str(e)[:40]}")
    p.kill()

    ims = [(f[:-4], Image.open(os.path.join(OUT, f)))
           for f in sorted(os.listdir(OUT)) if f.endswith(".png") and not f.startswith("_")]
    COLS, CELL, PAD, LABEL = 4, 300, 12, 22
    rows = (len(ims) + COLS - 1) // COLS
    sheet = Image.new("RGB", (COLS * CELL + (COLS + 1) * PAD,
                              rows * (CELL + LABEL) + (rows + 1) * PAD), (245, 245, 247))
    d = ImageDraw.Draw(sheet)
    for i, (name, im) in enumerate(ims):
        r, c = divmod(i, COLS)
        x = PAD + c * (CELL + PAD); y = PAD + r * (CELL + LABEL + PAD)
        t = im.copy(); t.thumbnail((CELL, CELL), Image.LANCZOS)
        sheet.paste(t, (x + (CELL - t.size[0]) // 2, y))
        d.text((x, y + CELL + 4), name, fill=(60, 60, 70))
    sheet.save(os.path.join(OUT, "_sheet.png"))
    print(f"\n  {ok} of {ok + len(failed)} rendered"
          + (f"   failed: {failed}" if failed else "")
          + f"\n  sheet: {OUT}/_sheet.png")


if __name__ == "__main__":
    main()
