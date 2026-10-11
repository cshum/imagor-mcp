---
name: imagor-composition
description: Use when composing or debugging an imagor URL with imagor-mcp — building a composite image, placing an overlay or text, cropping and scaling, or working out why a filter did nothing.
---

# Composing imagor URLs with imagor-mcp

The tools construct and validate paths. `verify_imagor_url` runs one and reports
what the server did with each filter, which is the only way to know a
composition worked: imagor answers `200` with the image unchanged when a filter
does not apply.

## Order in a path

Parameters first, then filters, then the image:

```
{trim}/{crop}/{fit-in}/{WxH}/{halign}{valign}/{filters}/{image}
```

**Everything after `filters:` is read as the image.** A parameter written after
the filters becomes part of the image name, and the request fails to load. This
holds inside nested paths too.

## Two crops, opposite orders

| written as | order | output |
|---|---|---|
| `0x0:120x90/240x210/IMAGE` | crops the source, then scales | 240×210 |
| `240x210/filters:crop(0,0,120,90)/IMAGE` | resizes, then crops | 120×90 |

The filter discards the resize. To take a region and scale it — the usual
intent — use the crop parameters. `verify_imagor_url` warns when a `crop()`
filter is combined with dimensions.

Coordinates are pixels, or floats `0.0`–`1.0` for a fraction of the dimension.

The crop parameters are four flat fields, not a nested object:

```json
{"crop_left": 201, "crop_top": 65, "crop_right": 289, "crop_bottom": 99}
```

## Scaling

- `fit-in/WxH` fits inside the box and **does not upscale**. To enlarge, add
  `filters:upscale()`.
- `WxH` without `fit-in` fills the box, cropping the overflow.
- `stretch/WxH` ignores the aspect ratio.

## Overlays and text

`image(path, x, y[, alpha[, blend_mode]])` composites a processed image, and its
path is a full imagor path, so it nests. `text(text, x, y, font, color)` draws a
label.

Positions are relative to the **current canvas**. Inside a nested path that is
the overlay, not the parent — so a label centred under a product belongs in the
nested path, where `center` means the overlay:

```
fit-in/200x260/filters:fill(white)
  :image({crop}/fit-in/190x170/filters:upscale()/{IMAGE},center,10)
  :text({NAME},center,196,sans%2016,888888)
  /color:white
```

Accepted positions, on both axes:

- A number: from the left or top. **Negative: from the right or bottom**, i.e.
  an inset of that many pixels.
- `left`/`l`, `right`/`r`, `top`/`t`, `bottom`/`b`, `center`, optionally with an
  offset. `right-20` moves **past** the right edge, the opposite direction to
  `-20`, so it can clip off the canvas.
- `Np` or a float `0.0`–`1.0` for a percentage of the canvas.
- `repeat` tiles.

`full` or `f` in a nested dimension inherits the parent: `fullxfull`,
`fit-in/full-20xfull-20`.

## Nested path order

Inside `image()` and `watermark()` the path is a normal path, so the same rule
applies: crop and dimensions before `filters:`. `image()` is the one to reach for
— `watermark()` is the older, narrower form.

## Colors and awkward URLs

`color:white` is a canvas with no source. `b64:` prefixes a base64url-encoded
image URL when it contains characters that are awkward in a path.

`image()` separates its arguments with commas, so a path inside it must not
contain one: a `fill(rgb(1,2,3))` would end the argument early. Encode such a
path with `b64:`.

## A whole composition

A banner built from a `color:` canvas, a cropped panel, four product tiles and
their labels. Overlays are placed at absolute canvas coordinates; each tile's
height follows its crop's aspect ratio, so `fit-in` to a common box gives a
consistent grid:

```json
{
  "image": "color:white", "width": 1200, "height": 640,
  "filters": [
    {"name": "image", "args": "0x0:194x262/440x640/PHOTO,0,0"},
    {"name": "text",  "args": "PRADA,540,20,sans-bold-48,000000"},
    {"name": "image", "args": "201x65:289x99/fit-in/190x170/filters:upscale()/PHOTO,540,100"},
    {"name": "image", "args": "414x20:488x98/fit-in/190x170/filters:upscale()/PHOTO,860,100"},
    {"name": "text",  "args": "leather%20shoulder%20bag,540,290,sans-16,888888"},
    {"name": "text",  "args": "HKD$29900,540,316,sans-bold-24,000000"}
  ]
}
```

Text is drawn at absolute canvas coordinates, so a label under a tile is placed
at the tile's x. The nested-path form in the section above is for when the x is
not known in advance or the label should follow the overlay's own width.

## What the tools will not tell you

Learned by building composites and measuring them against a reference. Each of
these is invisible in a `200` response and obvious only in the finished image.

- **Set the output format first.** JPEG artefacts across a flat background
  dominate every measurement of a composite; adding `format(png)` was the single
  largest improvement in one measured run, larger than any placement fix. Do it
  before tuning anything else.
- **`x` is the left edge.** A 0–1 float is a fraction of canvas width used as the
  left edge, so `0.5` starts the text at the middle of the canvas rather than
  centring a box on it. Alignment is the `text()` argument that centres, not `x`.
- **`y` is the top of the text box**, not a baseline. Text is placed to the pixel
  and the request still returns `200` when it is a few pixels out.
- **Font names take hyphens for spaces**: `Palatino-63` means Palatino at 63.
- **A font that is not installed falls back silently**, usually to sans, and the
  request still returns `200`. If the face you asked for looks wrong, ask for the
  generic family (`serif`, `sans`, `monospace`) instead of guessing a name.
- **A wordmark no font matches is better drawn one letter per call.** A whole
  string rendered in the nearest font can be worse than leaving the space blank;
  placing each letter at its own x is what makes a logo land.
- **Text a crop cut off has to be redrawn**, and per glyph when the font is
  wrong for it. Prices are the usual case.
- **Inside a nested path, dimensions come before the source**: `134x12/color:efefeb`
  works, `color:efefeb/134x12` is a `400`.
- **Sources inside `image()` need a full URL.** A bare filename is a `404`, even
  when the same name works as the top-level image.
- **A strip no source covers can be painted flat**: `image(311x35/color:f1f2ee,1096,600)`
  is a rectangle, and it is often the cheapest way to repair a background.

### Resize and crop are not interchangeable

- **A plain `WxH` is already a cover-crop, and always from the centre.** Use it
  for sizing. It takes a little off both sides and can never aim at the one edge
  that is wrong.
- **`crop()` is the only mode that aims.** It is what removes an unwanted sliver
  on a known side — measured at 0.2% residual against 4.5% for every resize mode.
- **`fit-in` does not crop at all**; it letterboxes, so it keeps the sliver and
  adds bars.
- **`trim()` only removes a uniform border.** On a product sitting on a panel it
  is a no-op, returning byte-identical output. Do not reach for it to clean an edge.
- **`focal` lands where a centre crop lands** unless the subject is off-centre.

## Placing text is a search, not a calculation

Text lands wherever the font's own metrics put it, and nothing reports those. Do
not tune a whole composition and guess which part moved.

- Render **one glyph** on its own, fetch it, and find the ink. That is where the
  pen starts for that font at that size.
- Two or three probes give you the offset for a whole label. One measured run
  placed a wordmark letter by letter this way after a single string came out
  worse than blank.
- Measure the region you changed, not just the whole image: a whole-image number
  hides a label that is 40px out, because most of the canvas did not move.

## Working method

1. `describe_imagor` once, for the grammar this server targets.
2. `list_imagor_filters` before using a filter you have not used here: it carries
   imagor's own signature, description and accepted forms.
3. `build_imagor_url` refuses invalid compositions, so a refusal is a mistake
   worth fixing rather than working around.
4. `verify_imagor_url` the result, passing the same typed parameters you built
   from — it takes parameters, not a URL. It reports each filter's outcome and
   the dimensions the server produced, which is how you catch a composition that
   ran but came out the wrong size. `applied` means it ran; `declined` means it
   was recognised and did nothing, which is what `blur()` with no sigma does;
   `not_reported` means this processor does not handle it or the deployment
   disabled it. Absence is not failure.
5. To composite, build small pieces first and check each: one product tile, then
   the grid, then the whole banner. A wrong number in a nested path is much
   harder to see in the finished image than in the piece.
6. Measure what you rendered against what you intended, numerically. A `200` and
   a plausible-looking image are not evidence; a mean difference over the region
   you changed is.
