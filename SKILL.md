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
- **`center` as an `x` centres the text on its parent canvas**, and it is the only
  form that does: measured at 1px off centre on an 800px canvas, where `0.5` and
  `400` both put the *left edge* at that point.
- **`center` takes no offset.** `center-100` and `c-100` do not parse as positions,
  fall through to a number, and silently become `0` — the text lands at the far
  left with a `200` and no warning. There is no way to shift a centre.
- **A negative `x` sets the box's right edge**, so `-300` on an 800px canvas ends
  the text at 500. It is not a leftward offset.
- **`center` resolves against the parent, so a group can centre its contents — but
  only if the canvas can hold them.** A group's canvas is the resized source, so
  sizing it to anything other than the source's own size resizes the image. A
  nested canvas can be placed anywhere, and a label inside one needs no x.
- **Weigh that against what it costs the image.** Measured on one product tile:
  placing the product and label directly kept the product region at 2.1 mean
  difference and put the label 17px off; wrapping them in a group sized to hold
  the label centred the label exactly but pushed the product to 39.7, because the
  source was resized to fit. A group sized exactly to the source preserved the
  product at 2.1 and could not hold a label below it. **Centring and image
  fidelity pull against each other unless the source already includes the space
  the label needs** — so either capture sources with their label band, or place
  text absolutely and measure it.
- **`y` is the top of the text box**, not a baseline. Text is placed to the pixel
  and the request still returns `200` when it is a few pixels out.
- **Font names take hyphens for spaces**: `Palatino-63` means Palatino at 63.
- **A font that is not installed falls back silently**, usually to sans, and the
  request still returns `200`. Measured on one host, these serif names all fell
  back: `Times`, `Times-New-Roman`, `Hoefler-Text`, `New-York`, `Big-Caslon`,
  `Rockwell`. These resolved: `serif`, `Palatino`, `Baskerville`, `Bodoni-72`,
  `Didot`, `Georgia`, `Cochin`, `Charter`, `PT-Serif`. `Palatino` matched a
  reference wordmark best. Expect only one usable sans face as well: `Helvetica`,
  `Arial` and `Roboto` all resolved to the same fallback, which caps how close
  sans text can get.
- **A wordmark is usually letter-spaced, so draw it one glyph per call.** A logo
  with tracking cannot be reproduced by one `text()`: measured on a reference
  wordmark, the gaps ran about 8px wider than the font's own spacing, and
  single-call attempts came out worse than leaving the space blank. Measuring
  each glyph's centre and placing thirteen separate calls reached 90% ink overlap.
- **When the sources are exact crops, the whole residual is text.** Image regions
  matched bit-exactly, so every point of difference was glyph shape. Measure the
  text bands separately from the image regions, or you will tune placement
  against a number that placement cannot move.
- **Text a crop cut off has to be redrawn**, and per glyph when the font is
  wrong for it. Prices are the usual case.
- **Inside a nested path, dimensions come before the source**: `134x12/color:efefeb`
  works, `color:efefeb/134x12` is a `400`.
- **Sources inside `image()` need a full URL.** A bare filename is a `404`, even
  when the same name works as the top-level image.
- **A strip no source covers can be painted flat**: `image(311x35/color:f1f2ee,1096,600)`
  is a rectangle, and it is often the cheapest way to repair a background. Stacking a
  colour rectangle behind an inset photo is also how a border is drawn, and it keeps
  the canvas size under your control.
- **Text that comes from data goes in the `text` field, not `args`.** A title
  containing a comma truncates: imagor renders nothing and still returns `200`.
  One containing `?` changes the path a signature covers, so the request comes
  back as a signature mismatch. Both look like something other than a text
  problem. The `text` field is encoded as base64url with markup characters
  escaped, which handles commas, `?`, `&` and angle brackets together. If you
  build the argument by hand instead, send it through `b64:` yourself, and escape
  `&`, `<` and `>` as `&amp;`, `&lt;` and `&gt;` or the render fails with
  *invalid markup*.
- **`padding(color, N)` is a border in one filter** — but it grows the output, a
  300px image becoming 340px at `padding(x,20)`. The same padding is also a path
  parameter, written as negative dimensions, which keeps the canvas size fixed.
- **`round_corner` takes pixels, not a ratio.** `round_corner(0.5)` is a half-pixel
  radius and renders square. A circle is `round_corner(r,r,background)` where `r`
  is half the width, and the colour matters: leave it out and the corners come
  back black.
- **`smart` is a crop mode**, like `fit-in`, for content-aware framing of a photo
  with a subject. It composes with `round_corner` for a circular avatar.

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

## Responsive compositions

A nested path can inherit its parent's dimensions, so one composition can serve
several canvas sizes instead of being pinned to one. In any dimension segment,
`full` or `f` means the parent's dimension, and `full-NNN` means that minus NNN
pixels.

- `fullxfull` (or `fxf`) — the overlay fills the parent canvas
- `fit-in/full-40xfull-40/PHOTO` — inset 40px on every side, fitted within
- `12xfull/color:3b82f6` — a 12px accent bar at full height, at any canvas size
- `full-80` as a `text()` wrap width — the text wraps to the canvas it lands on

Measured: the same composition rendered at 1200x630, 1080x1080 and 1600x900 put
the accent bar at columns 0-11 and inset the photo by exactly 40px in all three.

**Sizes adapt; type does not.** A font size is in pixels and a text position is
absolute, so the same composition renders relatively smaller type on a larger
canvas - at 1080x1080 the fitted title used 960 of 1080px where at 1600x900 it
used 1530. There is no percentage font size. Size the text per canvas with
`fit_text`, or accept that it will not scale with the frame.

## Prefer the current filter to its legacy form

`text()` supersedes `label()`, and `image()` supersedes `watermark()`. Both pairs
take the same shape — the source or text first, then `x`, then `y` — so a `text`
or `source` field works for either. Reach for the current one.

## A position is not a dimension

`full` and `full-N` belong to the dimension segment. The position grammar has no
such form: **written as a position it is read as `0`** and the overlay lands at
the origin, with a `200`. Measured: `y=full-220` on a 630px canvas put a title at
the top.

The build tools now warn when they see one. For a position use pixels, a
percentage such as `20p`, a float `0`–`1`, or an alignment such as `center` or
`bottom-40` — and remember a position is the **top-left of the box**, so an
alignment offset moves *past* the edge while a negative number insets from it.

## Nested sources

`image()` and `watermark()` take a source that is itself a path or URL, and a URL
carries `?` and `,` without meaning anything by them. Written raw into `args`,
a `?` ends the path the signature covers (**403**) and a `,` splits the argument
list (**404**) — both measured. Put the source in the `source` field instead of
`args`, and it is encoded when it needs to be:

```json
{"name":"image","source":"https://cdn.example.com/a,b.png?v=2","args":"600x600,60,60"}
```

The same reasoning as `text`: the argument is data, not syntax, so it does not
belong in a string that is parsed for syntax.

## Templates

`render_template` renders a layout at a canvas size, so a layout is written once
and served at any size. Geometry is written the way imagor writes it, so `full`
and `full-N` work in any slot, and a text slot given a height is **fitted against
the box its geometry resolves to** — which is what makes the type adapt, since a
font size on its own is absolute.

```json
{
  "canvas_width": 1200, "canvas_height": 630, "background": "color:1a1a2e",
  "slots": [
    {"name":"bar","kind":"rect","image":"color:3b82f6","x":"0","y":"0",
     "width":"12","height":"full"},
    {"name":"eyebrow","kind":"text","text":"{{eyebrow}}","x":"40","y":"40",
     "font":"sans-bold-22","color":"3b82f6"},
    {"name":"title","kind":"text","text":"{{title}}","x":"40","y":"140",
     "width":"full-80","height":"220","font":"serif","color":"ffffff",
     "align":"low","max_size":84,"min_size":24},
    {"name":"author","kind":"text","text":"{{author}}","x":"40","y":"full-70",
     "font":"sans-24","color":"888888"}
  ],
  "values": {"eyebrow":"ENGINEERING","title":"…","author":"Adrian Shum"}
}
```

### Geometry forms

A slot's geometry is written the way imagor writes it, plus fractions:

| form | means |
|---|---|
| `40` | 40px |
| `full`, `f` | the canvas dimension |
| `full-80`, `f-80` | the canvas dimension minus 80px |
| `half`, `h` | half the canvas dimension |
| `third`, `t` / `quarter`, `q` | a third / a quarter |
| `half-40` | half, minus 40px |

**imagor has no fraction form, and no `full-N` for a position.** `full-N` is a
*dimension*: written as a position it is read as `0` and the element lands at the
origin, silently, returning `200`. The template resolves positions to pixels for
you, and passes everything else imagor accepts for a position (`center`, `top`,
`bottom`, `right`, `left`, and their offset forms) straight through.

**A position is the top-left of the element's box**, so `full-70` for a caption
puts the *box* 70px from the bottom and the ink reaches below that. Across nine
real layouts, four ended up with text inside a 40px margin for exactly this
reason — leave more room than the margin you want.

Measured on that template at three canvas sizes: **serif-84, serif-76, serif-84**,
with the box following the canvas each time and every element inside the frame.
The slot list is applied in order, so later slots draw over earlier ones, and
values fill `{{placeholders}}` in both text and image fields.

## When imagor refuses to serve

The server and the imagor it points at must agree on how a request is authorised.
A disagreement does not look like a configuration error: it comes back as a `403`
with `url signature mismatch` in the body. Three settings have to line up.

- **Unsigned or signed.** `IMAGOR_UNSAFE=1` sends no signature; `IMAGOR_SECRET`
  signs. A client sending no signature to a signed imagor is refused. A client
  that signs, against an imagor running unsafe, is accepted.
- **`IMAGOR_SIGNER_TYPE`** — `sha1` (the default), `sha256` or `sha512`.
- **`IMAGOR_SIGNER_TRUNCATE`** — signature length in characters; `0` means none.

A signature covers the whole path, so **any** difference in those three produces
the same refusal, and neither side can read the other's secret back. The only way
to know they agree is to make a request. `describe_imagor` reports what this
server is using — compare that against how imagor was started before debugging
anything else.

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
