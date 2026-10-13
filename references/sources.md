## Sample sources

Ready-made images, served from **`https://s.imagor.net`**. A file at `sources/<name>.svg` in this repo
serves at `<base>/<name>.svg`. They are pure-vector SVG, so one file rasterises crisply at
any size, and the sizes below are the files' own.

This table is generated from `sources/manifest.json` by `sources/generate.py` - edit that,
not this.

| name | size | for |
|---|---|---|
| | | ***Product** — a grid cell or a product card* |
| `product-bag` | 400×400 | Soft and rounded. The default for a grid cell. |
| `product-bottle` | 300×480 | Tall and narrow. A different aspect to the others. |
| `product-box` | 400×400 | Rigid with a lid. Sits well beside a bag. |
| | | ***Scene** — a subject in a frame, for copy beside an image* |
| `scene-tall` | 700×900 | A subject in a tall frame, for a launch card. |
| `scene-wide` | 1600×600 | A subject in a wide frame, for a banner. |
| | | ***Edge cases** — these exist to break things, not to look good* |
| `huge` | 4000×4000 | Much larger than any box, so a fit has to downscale. |
| `square` | 600×600 | A shape no other fixture has, for testing crop. |
| `tall` | 200×1600 | The other extreme. |
| `tiny` | 64×64 | A fit has to upscale it, and fit-in will refuse. Smaller than any box. |
| `transparent` | 400×400 | No ground at all. Tests what a card does over nothing. |
| `wide` | 1600×200 | An extreme aspect. Exposes letterboxing. |

Put one in a slot's `image`, or in `values`:

```json
"values": { "photo": "https://s.imagor.net/product-bag.svg" }
```

`python3 sources/generate.py` rewrites the files, the manifest and this table — edit its
`FIXTURES` table rather than the output.

`cookbook/` holds ready-made `render_template` payloads for the formats that come up most.
Take the closest one, change the values, render it. They are rendered by the test suite at
three canvas sizes — a recipe that no longer works teaches a shape that fails.
