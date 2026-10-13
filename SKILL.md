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

- `fit-in/WxH` fits **inside** the box and **does not upscale**, so it letterboxes.
  `filters:upscale()` lifts the no-enlargement rule but does **not** make it cover: a
  694×941 photo in `fit-in/580x600/filters:upscale()` is 443×600, still short.
  `full-fit-in` is its opposite — it **covers**, overrunning one side (a 300×200 frame
  returns 393×200) — which is what you want only when the box cannot be reshaped and
  whatever holds it clips. Add `upscale()` when the source is smaller than the box.
- `WxH` fills the box exactly by **cropping** from the centre, so it can never aim at
  the one edge that is wrong, and a source of another shape loses its edges.
- `stretch/WxH` ignores the aspect ratio, so it **distorts** — a 290×175 source in
  `280x154` comes back at aspect 1.818 against 1.657, nearly 10% wrong. Reshape the
  box instead, or `full-fit-in` if it cannot be reshaped.
- `crop()` is the only mode that **aims**, which is what removes an unwanted sliver
  on a known side. `focal` and `smart` also crop: `focal` lands where a centre crop
  does unless the subject is off-centre, and `smart` frames by content.
- `trim()` only removes a uniform border, so on a product on a panel it is a no-op
  with byte-identical output.

**When you control the frame, shape it to the source and use plain `WxH`** — no fit at
all, because a fit mode adds nothing when the shapes already agree. **Fit is for a cell
you cannot shape**, a grid cell whose size the layout fixes: there the image sits inside
without distortion and the cells align. A fitted image is centred in its box, because
the template adds `fill(none)`, which is what keeps a row of tiles sharing a baseline.

## Overlays and text

`image(path, x, y[, alpha[, blend_mode]])` composites a processed image and its
path is a full imagor path, so it nests. `text(text, x, y, font, color)` draws a
label. Positions are relative to the **current canvas**, so inside a nested path
that is the overlay, not the parent.

```
fit-in/200x260/filters:fill(white)
  :image({crop}/fit-in/190x170/filters:upscale()/{IMAGE},center,10)
  :text({NAME},center,196,sans%2016,888888)
  /color:white
```

Accepted positions, on both axes:

- A number: from the left or top. **Negative: from the right or bottom**, an inset
  of that many pixels.
- `left`/`l`, `right`/`r`, `top`/`t`, `bottom`/`b`, `center`, optionally offset.
  `right-20` moves **past** the right edge — the opposite of `-20` — so it can clip.
- `Np`, or a float `0.0`–`1.0`, for a percentage of the canvas, placing the **left**
  or **top** edge: `0.5` is `50p`, and neither centres. `center` takes no offset —
  `center-100` does not parse and silently becomes `0`.
- `repeat` tiles.

A position is the **top-left of the box**, so `full-70` for a caption puts the
*box* 70px from the bottom and the ink reaches below it. Leave more room than the
margin you want.

**A marker beside text is sized against the x-height, not the line.** A square dot
tall enough to span ascender to baseline is centred on the text's ink and still reads
as sitting low, because the eye measures it against the lowercase letters beside it.
For 15px type the x-height is about 8px — match that, and check the dot's own rows
against the text's, not its centre against the text's centre.

`full` or `f` in a **nested dimension** inherits the parent: `fullxfull`,
`fit-in/full-20xfull-20`. A **position** has no such form — `full-N` written as a
position is read as `0`, lands the element at the origin and returns `200`. The
build tools warn when they see one. This is the one form that survives a round trip
and still means something; see *Editing a path you already have*.

## Nested path order

Inside `image()` the path is a normal path, so crop and dimensions come before
`filters:`. `watermark()` is the older, narrower form of `image()`, and takes the
source first, then `x`, then `y`.

Sources inside `image()` need a full URL; a bare filename is a `404`, even when
the same name works as the top-level image.

## Colors and awkward URLs

`color:white` is a canvas with no source: `image(311x35/color:f1f2ee,1096,600)` is
a rectangle. Stacking one behind an inset photo draws a border, which keeps the
canvas size under your control — `padding(color, N)` grows the output instead
(300px becomes 340px at `padding(x,20)`).

`round_corner` takes pixels, not a ratio: `round_corner(0.5)` is a half-pixel radius
and renders square. A circle is `round_corner(r,r,color)` with `r` half the width, and
the colour is required — leave it out and the corners come back black. Do not combine
it with `fit`: a fitted image letterboxes, so the corner is cut from the letterboxed box
rather than the photograph and the circle clips the picture. Use a square crop instead.

## When a value has to be encoded

**Text takes a whitelist.** Only `[A-Za-z0-9_-]` is left alone; anything else is
encoded, and markup characters escaped, because the renderer reads Pango markup in
a text argument.

**A source takes a character set, plus reserved words.** A source is a path or URL
and legitimately contains `/`, `:`, `.` and `=`, so a whitelist would encode every
real source. Encoded are **space `?` `#` `&` `(` `)` `,`** — each changes how the
path is read — and a source **beginning with `trim/`, `meta/`, `fit-in/`,
`full-fit-in/`, `adaptive-fit-in/`, `stretch/`, `top/`, `left/`, `right/`,
`bottom/`, `center/` or `smart/`**, which imagor reads as a command.

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

`render_template` renders a layout at a canvas size, so a layout is written once and
served at any size. A text slot given a height is **fitted against the box its geometry
resolves to** — see *Placing text* for what that costs and how to cap it.

**A generic font is resolved to a narrower face and the note says so** — `sans` to
`Open-Sans`, `serif` to `Liberation-Serif`. See *Placing text*.

```json
{
  "canvas_width": 1200, "canvas_height": 630, "background": "color:1a1a2e",
  "slots": [
    {"name":"bar","kind":"rect","image":"color:3b82f6","x":"0","y":"0",
     "width":"12","height":"full"},
    {"name":"eyebrow","kind":"text","text":"{{eyebrow}}","x":"40","y":"40",
     "font":"Open-Sans-bold-22","color":"3b82f6"},
    {"name":"title","kind":"text","text":"{{title}}","x":"40","y":"140",
     "width":"full-80","height":"220","font":"Liberation-Serif","color":"ffffff",
     "align":"low","max_size":84,"min_size":24}
  ],
  "values": {"eyebrow":"ENGINEERING","title":"…"}
}
```

**A slot's `filters` field applies filters to that slot's own source**, because the
image field is data rather than syntax — a path written there is encoded as a source.
Write the filters **without** the `filters:` prefix: `smart/round_corner(230,230,fff)`.
Writing the prefix too is accepted and ignored, so a doubled one is not an error you
will see. The signature is minted at render time, so the layout stays portable.

Slots are applied in order, so later ones draw over earlier ones. Values fill
`{{placeholders}}` in text, image **and background** fields. A placeholder with no value
**refuses the render** rather than drawing its own braces — an unfilled one renders as
literal `{{name}}`, which reads as a layout fault instead of a missing value.

**A slot carrying its own `slots` is a canvas.** Its width and height become the canvas
the inner slots lay out on, and the result draws at the outer slot's position — the way
to centre text on something narrower than the frame. Inner slots may be text, an image
or a rect, so a canvas holds a panel or a photograph as well as a label. It is
transparent: it adds only what it draws.

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

imagor has no fraction or percentage form for a dimension, so those are the template
layer's own and resolve to pixels. `full` and `full-N` imagor does have, so they reach
the path. Positions resolve to pixels too; everything else imagor accepts for one
(`center`, `top`, `bottom`, `left`, `right` and their offsets) passes through.

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

There are fourteen ready-made layouts. **Call `get_cookbook_recipe` with no name** and it
returns the list: what each layout is for, its size, and which values it takes. Then call
it again with the name you want.

```
get_cookbook_recipe                      ->  the list, to choose from
get_cookbook_recipe  name: "event"       ->  the template, ready for render_template
```

**Start with the no-name call rather than guessing a name**, and rather than reading the
`imagor-mcp://cookbook` resource: the resource carries the same list, but a client that
surfaces only tools never shows it, and a run that tried reported exactly that. Match on
what a layout *is for* — the note — not on its name, which will not resemble the request.

Take the closest and change the values. They carry spacing and type sizes that took
iteration to get right, and they are rendered by the test suite at three canvas sizes,
so a recipe that no longer works cannot reach the list.

The recipes reference their images from **`s.imagor.net`**, so they render as they are.
For the full set with dimensions and what each is for, read the `imagor-mcp://sources`
resource.

## Placing text is a search, not a calculation

Text lands wherever the font's own metrics put it, and nothing reports those.

- **A generic is resolved to a narrower face and the note says so.** `sans`, `serif`
  and `monospace` become `Open-Sans`, `Liberation-Serif` and `Liberation-Mono`. The
  generics are DejaVu, the widest face imagor carries, so type sized for one overflows
  a box drawn for anything narrower. Name `DejaVu-Sans` to keep it.
- **An uninstalled name falls back to DejaVu, silently and with a `200`**, so which
  faces exist belongs to the imagor you point at — check before relying on one.
- **The font is a Pango description with hyphens for spaces.** `sans-bold-40` is
  `sans bold 40`: **family first, size last**. A number anywhere else is ignored —
  `sans-40-bold` renders at the default 20 — and a word before the family becomes it,
  so `bold-sans-40` renders as `sans-40`. A literal space is a **malformed URL**: use
  a hyphen or `b64:`.
- **Only the weights a face carries exist, and an absent one falls back silently.** On
  the official image Open Sans has Light, Regular, Semibold, Bold and ExtraBold;
  Liberation has Regular and Bold. `Open-Sans-Medium` renders as Regular and
  `Open-Sans-Black` as ExtraBold, both with a `200` — the same silent substitution as
  an unknown family. Check by rendering the weight and comparing against Regular.
- **A fitted text slot takes the largest size that fits its box.** That is what a
  headline wants and what a supporting line does not — a short line balloons to fill
  the box it was given. Cap it with `max_size`, or give the slot no height and a
  `size` instead.
- **To centre text on something narrower than the frame, give it a canvas of its
  own: a slot carrying its own `slots`.** `x=center` resolves against the canvas it
  is on, and a numeric `x` leaves text hanging left whatever `align` says. The outer
  slot's width and height become that canvas; the result draws where it is placed.
  A label measured this way landed at centre 921 against a product centre of 922.
- **A wordmark is one `text()` call, even though that loses the tracking.** `text()`
  has no letter-spacing, and its `spacing` argument is *line* spacing. One glyph per
  call reaches the reference's spacing but hard-codes the string, so a different
  value breaks the layout. Prefer one call; the rest of the layout stays exact.
- **Measure the region you changed, not the whole image.** A whole-image number
  hides a label that is 40px out, because most of the canvas did not move.

## Grids of images

Sizing each panel to its own source gives panels of different shapes, which reads as
unrelated pictures rather than a grid — and no amount of position tuning fixes it.

- **Give every cell one identical size, on even rows and columns, and fit the image
  inside its cell** (`fit`). The cells align; the images inside them do not fill
  them, so nothing is distorted and the space around each is even.
- **Percentage forms make the sameness checkable**: five cells of `16p+12 x 26p+5`
  stay one size at every canvas; five of 300x250 do not.

## Responsive compositions

`fit-in/full-40xfull-40/PHOTO` insets 40px on every side, `12xfull/color:3b82f6` is an
accent bar at full height, and `full-80` as a `text()` wrap width wraps to the canvas it
lands on.

**Sizes adapt; type does not.** A font size is in pixels, so the same composition
renders relatively smaller type on a larger canvas, and there is no percentage font
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
3. `build_imagor_url` refuses invalid compositions, so a refusal is a mistake worth
   fixing rather than working around.
4. `verify_imagor_url` the result, passing the same typed parameters you built from —
   it takes parameters, not a URL. `applied` means it ran; `declined` means it was
   recognised and did nothing; `not_reported` means this processor does not handle it.
   Absence is not failure.
5. **Look at what came back.** A composition can report `applied` and still be wrong,
   and placement, cropping and proportions are visible at preview size. Where the
   preview is too small to read a label, call again with `preview_region` as `x,y,w,h`
   in **result** pixels — full resolution for a fraction of the cost. Iterate until it
   matches.
6. To composite, build small pieces first: one tile, then the grid, then the banner. A
   wrong number in a nested path is much harder to see in the finished image.
7. Measure only what looking cannot settle, and measure the **rows**, not the centres.
   Compare the thing's own ink rows against the rows it is meant to match — a dot's
   rows against its text's, a panel's against its neighbour's. Two centres can agree
   to the pixel while the shapes read as misaligned, and a whole-image average hides
   a 40px error. Crop to the region you changed and check the pairing before you
   believe the number.

**These rules are procedures and facts on purpose.** A procedure says what to type and
is followed; a preference asks the reader to want the right thing and is not, so where
one would be needed the server enforces it instead — a generic font resolves to a
narrower face, `fit` centres in its box.
