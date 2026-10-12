---
name: imagor-composition
description: Use when composing or debugging an imagor URL with imagor-mcp — building a composite image, placing an overlay or text, cropping and scaling, or working out why a filter did nothing.
---

# Composing imagor URLs with imagor-mcp

`verify_imagor_url` is the only way to know a composition worked: imagor answers
`200` with the image unchanged when a filter does not apply. It returns the
rendered result as an image, so check it against your reference rather than only
reading what the server reported — the report says a filter ran, not that the
result is right.

## Order in a path

```
{trim}/{crop}/{fit-in|full-fit-in}/{WxH}/{halign}{valign}/{filters}/{image}
```

**Everything after `filters:` is read as the image.** A parameter written after
the filters becomes part of the image name and the request fails to load. This
holds inside nested paths too.

## Two crops, opposite orders

| written as | order | output |
|---|---|---|
| `0x0:120x90/240x210/IMAGE` | crops the source, then scales | 240×210 |
| `240x210/filters:crop(0,0,120,90)/IMAGE` | resizes, then crops | 120×90 |

The filter discards the resize. To take a region and scale it — the usual intent
— use the crop parameters. `verify_imagor_url` warns when `crop()` is combined
with dimensions. Coordinates are pixels or floats `0.0`–`1.0`. The crop
parameters are four flat fields, not a nested object:

```json
{"crop_left": 201, "crop_top": 65, "crop_right": 289, "crop_bottom": 99}
```

## Resize modes

- `fit-in/WxH` fits **inside** the box and **does not upscale**; the result can be
  smaller than the box on one side, so it letterboxes. `filters:upscale()` lifts
  the no-enlargement rule but does **not** make it cover: a 694×941 photo in
  `fit-in/580x600/filters:upscale()` is 443×600, still short of the box.
- `full-fit-in/WxH` is the one that **covers**: one side matches the box and the
  other **exceeds** it, so the aspect is kept and nothing is cropped. The same
  photo in `full-fit-in/580x600` is 580×786. Use it whenever an image has to fill
  a panel without cutting into it. It only enlarges when the source is smaller
  than the box, so add `filters:upscale()` when it is.
- `WxH` fills the box exactly by **cropping** the overflow from the centre, so it
  can never aim at the one edge that is wrong, and a source whose shape differs
  from the box loses its edges.
- `stretch/WxH` ignores the aspect ratio.
- `crop()` is the only mode that **aims**. It is what removes an unwanted sliver
  on a known side.
- `fit-in` letterboxes, so it keeps the sliver and adds bars.
- `trim()` only removes a uniform border; on a product sitting on a panel it is a
  no-op with byte-identical output.
- `focal` lands where a centre crop lands unless the subject is off-centre.
- `smart` is a crop mode too, for content-aware framing; it composes with
  `round_corner` for a circular avatar.

## Overlays and text

`image(path, x, y[, alpha[, blend_mode]])` composites a processed image and its
path is a full imagor path, so it nests. `text(text, x, y, font, color)` draws a
label. Positions are relative to the **current canvas** — inside a nested path
that is the overlay, not the parent, so a label centred under a product belongs
in the nested path.

```
fit-in/200x260/filters:fill(white)
  :image({crop}/fit-in/190x170/filters:upscale()/{IMAGE},center,10)
  :text({NAME},center,196,sans%2016,888888)
  /color:white
```

Accepted positions, on both axes:

- A number: from the left or top. **Negative: from the right or bottom**, an
  inset of that many pixels.
- `left`/`l`, `right`/`r`, `top`/`t`, `bottom`/`b`, `center`, optionally with an
  offset. `right-20` moves **past** the right edge, the opposite direction to
  `-20`, so it can clip off the canvas.
- `Np` or a float `0.0`–`1.0` for a percentage of the canvas, placing the **left**
  or **top** edge — `0.5` is the same as `50p`, and neither centres. `center`
  centres, and takes no offset: `center-100` does not parse and silently becomes
  `0`.
- `repeat` tiles.

A position is the **top-left of the box**, so `full-70` for a caption puts the
*box* 70px from the bottom and the ink reaches below it. Leave more room than the
margin you want.

`full` or `f` in a nested dimension inherits the parent: `fullxfull`,
`fit-in/full-20xfull-20`. A **position** has no such form — `full-N` written as a
position is read as `0` and the element lands at the origin, with a `200`. The
build tools warn when they see one.

## Nested path order

Inside `image()` the path is a normal path, so crop and dimensions come before
`filters:`. `image()` is the one to reach for; `watermark()` is the older,
narrower form, as `label()` is to `text()`. Both pairs take the source or text
first, then `x`, then `y`, so a `text` or `source` field works for either.

Sources inside `image()` need a full URL; a bare filename is a `404`, even when
the same name works as the top-level image.

## Colors and awkward URLs

`color:white` is a canvas with no source. A strip no source covers can be painted
flat: `image(311x35/color:f1f2ee,1096,600)` is a rectangle, and stacking a colour
rectangle behind an inset photo is also how a border is drawn — it keeps the
canvas size under your control, where `padding(color, N)` grows the output (300px
becomes 340px at `padding(x,20)`).

`round_corner` takes pixels, not a ratio: `round_corner(0.5)` is a half-pixel
radius and renders square. A circle is `round_corner(r,r,background)` where `r` is
half the width, and the colour matters — leave it out and the corners come back
black.

## When a value has to be encoded

**Text takes a whitelist.** Only `[A-Za-z0-9_-]` is left alone; anything else is
encoded, and markup characters escaped, because the renderer reads Pango markup in
a text argument.

**A source takes a character set, plus reserved words.** A source is nearly always
a path or URL, which legitimately contains `/`, `:`, `.` and `=`, so a whitelist
there would encode every real source and make every path opaque. Encoded are any
of **space `?` `#` `&` `(` `)` `,`** — each changes how the path is read — and a
source **beginning with `trim/`, `meta/`, `fit-in/`, `full-fit-in/`,
`adaptive-fit-in/`, `stretch/`, `top/`, `left/`, `right/`, `bottom/`, `center/`
or `smart/`**, which imagor reads as a command.

Put text in the `text` field and a source in the `source` field and both are
handled — the argument is data, not syntax, and does not belong in a string
parsed for syntax:

```json
{"name":"image","source":"https://cdn.example.com/a,b.png?v=2","args":"600x600,60,60"}
```

These rules match the converter in imagor-studio
(`server/internal/imagortemplate`), so a path either project builds means the
same thing.

## Templates

`render_template` renders a layout at a canvas size, so a layout is written once
and served at any size. A text slot given a height is **fitted against the box its
geometry resolves to**, which is what makes the type adapt — a font size on its
own is absolute.

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
     "align":"low","max_size":84,"min_size":24}
  ],
  "values": {"eyebrow":"ENGINEERING","title":"…"}
}
```

Slots are applied in order, so later ones draw over earlier ones. Values fill
`{{placeholders}}` in both text and image fields.

### Geometry forms

| form | means |
|---|---|
| `40` | 40px |
| `full`, `f` | the canvas dimension |
| `full-80`, `f-80` | the canvas dimension minus 80px |
| `half`, `h` | half the canvas dimension |
| `third`, `t` / `quarter`, `q` | a third / a quarter |
| `half-40` | half, minus 40px |
| `38p` | 38% of the canvas dimension |
| `6p+34p` | a percentage, plus a percentage of the same dimension |

imagor has no fraction and no percentage form for a dimension, so those are the
template layer's own and are resolved to pixels. `full` and `full-N` imagor does
have, so they survive to the path. The template resolves **positions** to pixels
too, and passes everything else imagor accepts for one (`center`, `top`,
`bottom`, `right`, `left` and their offset forms) straight through.

## Editing a path you already have

`parse_template` reads a path back into a template, so a composition you already
have can be edited rather than retyped. It is the inverse of `render_template`:

```
template JSON ──render_template──> path ──render──> image
     ▲                                                │
     └──────────── parse_template ◄──── path ◄────────┘
```

A form imagor can express comes back as a form: `full` and `full-80` survive. A
form it lacks has become a number and a path cannot carry it:

| written | comes back as | why |
|---|---|---|
| `half`, `third` | `471`, `314` | imagor has no fraction |
| `38p` | `684` | imagor has no percentage |
| a text slot with a height | the size fitting chose | chosen before a path exists |

Rewrite those as forms if the template has to adapt to another canvas size. A
path and a URL both work, signed or not.

## Cookbook

`cookbook/` holds ready-made `render_template` payloads for the formats that come
up most. Take the closest one, change the values, render it. They are rendered by
the test suite at three canvas sizes — a recipe that no longer works teaches a
shape that fails.

## Placing text is a search, not a calculation

Text lands wherever the font's own metrics put it, and nothing reports those.

- Render **one glyph** on its own, fetch it, and find the ink: that is where the
  pen starts for that font at that size. Two or three probes give the offset for
  a whole label.
- Font names take hyphens for spaces: `Palatino-63` is Palatino at 63.
- **A font that is not installed falls back silently**, usually to sans, with a
  `200`. Expect only one usable sans face.
- **A wordmark is usually letter-spaced, so draw it one glyph per call.** One
  `text()` cannot reproduce tracking; measuring each glyph and placing separate
  calls is what reaches the reference.
- Measure the region you changed, not the whole image: a whole-image number hides
  a label that is 40px out, because most of the canvas did not move. When the
  sources are exact crops, the whole residual is text, so measure the text bands
  separately or you will tune placement against a number placement cannot move.

## Responsive compositions

In any dimension segment `full` or `f` means the parent's dimension and
`full-NNN` means that minus NNN pixels, so one composition can serve several
canvas sizes: `fullxfull` fills the parent, `fit-in/full-40xfull-40/PHOTO` insets
40px on every side, `12xfull/color:3b82f6` is an accent bar at full height, and
`full-80` as a `text()` wrap width wraps to the canvas it lands on.

**Sizes adapt; type does not.** A font size is in pixels, so the same composition
renders relatively smaller type on a larger canvas and there is no percentage font
size. Size the text per canvas with `fit_text`, or accept that it will not scale.

## When imagor refuses to serve

A `403` with `url signature mismatch` is a configuration disagreement, not a
broken request. Three settings have to line up:

- **Unsigned or signed.** `IMAGOR_UNSAFE=1` sends no signature; `IMAGOR_SECRET`
  signs. A client sending no signature to a signed imagor is refused; a client
  that signs against an unsafe imagor is accepted.
- **`IMAGOR_SIGNER_TYPE`** — `sha1` (the default), `sha256` or `sha512`.
- **`IMAGOR_SIGNER_TRUNCATE`** — signature length in characters; `0` means none.

A signature covers the whole path, so any difference in those three produces the
same refusal and neither side can read the other's secret back. `describe_imagor`
reports what this server is using — compare before debugging anything else.

## Working method

1. `describe_imagor` once, for the grammar this server targets.
2. `list_imagor_filters` before using a filter you have not used here: it carries
   imagor's own signature, description and accepted forms.
3. `build_imagor_url` refuses invalid compositions, so a refusal is a mistake
   worth fixing rather than working around.
4. `verify_imagor_url` the result, passing the same typed parameters you built
   from — it takes parameters, not a URL. `applied` means it ran; `declined` means
   it was recognised and did nothing; `not_reported` means this processor does not
   handle it or the deployment disabled it. Absence is not failure.
5. **Look at what came back.** The rendered result is attached, scaled, and the
   note says by how much. Compare it against your reference: placement, cropping
   and proportions are visible at preview size, and a composition can report
   `applied` and still be wrong. Where the preview is too small to read a label,
   call again with `preview_region` as `x,y,w,h` in **result** pixels — that part
   comes back at full resolution for a fraction of the cost of the whole render.
   Iterate: build, look, adjust, until it matches.
6. To composite, build small pieces first and check each: one product tile, then
   the grid, then the whole banner. A wrong number in a nested path is much harder
   to see in the finished image than in the piece.
7. Where you cannot see the answer — a colour, a dimension, whether a filter
   changed anything — measure it. A `200` and a plausible-looking image are not
   evidence; a mean difference over the region you changed is. But measure only
   what looking cannot settle, because a number is not a substitute for the
   picture when the question is whether it looks right.
