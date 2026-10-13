# Cookbook sources

Ready-made images for the recipes in `../`. Every one is a pure-vector SVG, a few
hundred bytes each, so the whole set is under 5 KB.

They exist because the recipes need sources and a URL is the only input imagor takes
with no configuration at all: a filesystem path or a bucket means every reader sets up
storage before the first card renders.

Served from **`https://s.imagor.net/<name>.svg`** — Cloudflare Pages builds the site from
this directory, so the repo layout and the URLs map one for one and the deployed set can
be checked against the committed one.

## Generic material, and subjects

**Eleven of the sixteen are generic**: they test what a fit does, or provide a ground, and
any recipe can use them. **The other five are subjects** — a bag, a box, a bottle, a tall
scene, a wide scene — and a card that needs a bag needs *that* bag, so they are used once
and never shared.

| kind | count | what it is | reuse |
|---|---|---|---|
| `edge` | 6 | a size or shape that breaks a naive fit | any recipe |
| `texture` | 2 | a gradient ground | any recipe |
| `product` | 3 | a bag, a box, a bottle | one card each |
| `scene` | 2 | a subject in a frame | one card each |
| `portrait` | 3 | abstract head-and-shoulders geometry | **none yet** |

## The rules these follow

**No `<text>`, ever.** Type inside an SVG is rendered by whatever font the imagor you
point at happens to carry, and a face that is not installed falls back silently — a
fixture with letters would look right on one image and wrong on another. Shapes,
gradients and geometry only.

**No faces.** The portraits are geometry, not photographs of people. A stock photograph of
a person carries a likeness right that the photographer's model release covers, not the
reader's use of the recipe — and an abstract placeholder makes the point that the image is
standing in for something.

**Pixels, never physical units.** An SVG at `width="2.667in"` has no intrinsic pixel
size; imagor reads it at 72dpi and returns 192x144 rather than the 267x200 anyone meant.
With `width="400" height="400"` the intrinsic size is the number in the file.

**Small, and diffable.** These are source files, not assets. A pull request that changes
a fixture shows the change as text.

## Using them

`manifest.json` carries each fixture's name, kind, dimensions, aspect and what it is for.
It is embedded in the binary, so a tool can answer from it without a network call.

A recipe references one by URL:

```json
"values": { "photo": "https://s.imagor.net/product-bag.svg" }
```

## Regenerating

```sh
python3 generate.py
```

It rewrites every fixture and `manifest.json`. Edit the `FIXTURES` table rather than the
output — the generated files are overwritten.

## The `edge` fixtures

Six of these exist to break things rather than to look good, and they are the reason the
set is worth having:

| fixture | dimensions | what it exposes |
|---|---|---|
| `tiny` | 64x64 | a fit must not upscale it |
| `huge` | 4000x4000 | a fit must downscale it |
| `wide` | 1600x200 | extreme letterboxing |
| `tall` | 200x1600 | the other extreme |
| `square` | 600x600 | a shape nothing else has, for crop |
| `transparent` | 400x400, no ground | what a card does over nothing |

Measured through imagor: `fit-in/400x400` gives `64x64` for `tiny`, `400x50` for `wide`
and `50x400` for `tall`.

## Photographs, in one recipe

`product-grid` is the exception: it draws on six **Unsplash** photographs rather than
fixtures, to show the recipes working with real product photography. Unsplash permits
commercial use without attribution and hotlinking their CDN is their documented approach,
**but it makes that one recipe depend on a third party staying up where the rest of the
cookbook is self-contained.** The other thirteen use these fixtures.

## Brands and copy

The recipes' brands, people and copy are invented. `content/worlds.json` records the
three worlds they come from, what each world's voice is, and which recipes belong to it.
