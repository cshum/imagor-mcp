# Cookbook sources

Ready-made images for the recipes in `../`, served from **`https://s.imagor.net`**. A file
at `sources/<name>.svg` serves at `<base>/<name>.svg`, so the repo layout and the URLs map
one for one.

Cloudflare serves these with `max-age=31536000, immutable`, so a fixture that is removed
keeps answering from the edge cache long after the origin has stopped serving it. Adding or
changing one is immediate; removing one is not.

Every one is a pure-vector SVG, a few hundred bytes each, and the whole set is under 4 KB.

## Why URLs

A URL is the only image source imagor takes with no configuration. A filesystem path or a
bucket means every reader sets up storage before their first card renders; a URL works
immediately, from anywhere.

## What is here

| kind | count | for |
|---|---|---|
| `product` | 3 | a grid cell or a product card |
| `scene` | 2 | a subject in a frame, for copy beside an image |
| `edge` | 6 | a size or shape that breaks a naive fit |

The `edge` six are worth knowing about, because they are how you find out what a fit does
before it does it to your own images:

| fixture | dimensions | exposes |
|---|---|---|
| `tiny` | 64x64 | a fit must not upscale it |
| `huge` | 4000x4000 | a fit must downscale it |
| `wide` | 1600x200 | extreme letterboxing |
| `tall` | 200x1600 | the other extreme |
| `square` | 600x600 | a shape nothing else has, for crop |
| `transparent` | 400x400, no ground | what a card does over nothing |

Measured through imagor: `fit-in/400x400` gives `64x64` for `tiny`, `400x50` for `wide`
and `50x400` for `tall`.

## Using them

Put one in a slot's `image`, or in a recipe's `values`:

```json
"values": { "photo": "https://s.imagor.net/product-bag.svg" }
```

`manifest.json` carries each fixture's name, kind, dimensions, aspect and note. It is
embedded in the binary, so the server answers from it without a network call.

## Rules these follow

**No `<text>`.** Type inside an SVG is rendered by whatever font the imagor you point at
carries, and a face that is not installed falls back silently — a fixture with letters
would look right on one deployment and wrong on another. Shapes, gradients and geometry
only.

**Pixels, never physical units.** An SVG at `width="2.667in"` has no intrinsic pixel size:
imagor reads it at 72dpi and returns 192x144 rather than the 267x200 anyone meant. With
`width="400" height="400"` the intrinsic size is the number in the file.

**Small, and diffable.** These are source files, not assets. A pull request that changes a
fixture shows the change as text.

## Regenerating

```sh
python3 generate.py
```

It rewrites the fixtures, `manifest.json` and `references/sources.md`, and removes any
`.svg` the table no longer lists. Edit the `FIXTURES` table in `generate.py` rather than
the output.

## Brands and copy

The recipes' brands, people and copy are invented. `content/worlds.json` records the three
worlds they come from, what each world's voice is, and which recipes belong to it.
