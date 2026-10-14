---
name: imagor-composition
description: Use when composing or debugging an imagor URL with imagor-mcp — building a composite image, placing an overlay or text, cropping and scaling, or working out why a filter did nothing.
---

# Composing imagor URLs with imagor-mcp

`verify_imagor_url` is the only way to know a composition worked: imagor answers `200`
with the image unchanged when a filter does not apply. It returns the rendered result as
an image, so check it against your reference rather than reading the report alone — the
report says a filter ran, not that the result is right.

## Order in a path

```
{trim}/{crop}/{fit-in|full-fit-in}/{WxH}/{halign}{valign}/{filters}/{image}
```

**Everything after `filters:` is read as the image.** A parameter written there becomes
part of the image name and the request fails to load. True inside nested paths too.

## Two crops, opposite orders

| written as | order | output |
|---|---|---|
| `0x0:120x90/240x210/IMAGE` | crops the source, then scales | 240×210 |
| `240x210/filters:crop(0,0,120,90)/IMAGE` | resizes, then crops | 120×90 |

The filter discards the resize. To take a region and scale it — the usual intent — use
the crop parameters. `verify_imagor_url` warns when `crop()` is combined with
dimensions. Coordinates are pixels or floats `0.0`–`1.0`. The crop parameters are four
flat fields, not a nested object:

```json
{"crop_left": 201, "crop_top": 65, "crop_right": 289, "crop_bottom": 99}
```

## Resize modes

- `fit-in/WxH` fits **inside** the box and **does not upscale**, so it letterboxes.
  `filters:upscale()` lifts the no-enlargement rule but does **not** make it cover: a
  694×941 photo in `fit-in/580x600/filters:upscale()` is 443×600, still short.
  `full-fit-in` is its opposite — it **covers**, overrunning one side (a 300×200 frame
  returns 393×200) — wanted only when the box cannot be reshaped and its container
  clips. Add `upscale()` when the source is smaller than the box.
- `WxH` fills the box exactly by **cropping** from the centre, so it can never aim at the
  one edge that is wrong, and a source of another shape loses its edges.
- `stretch/WxH` ignores the aspect ratio, so it **distorts** — a 290×175 source in
  `280x154` comes back at aspect 1.818 against 1.657, nearly 10% wrong. Reshape the box
  instead, or `full-fit-in` if it cannot be reshaped.
- `crop()` is the only mode that **aims**, which is what removes an unwanted sliver on a
  known side. `smart` also crops, framing by content.
- `trim()` only removes a uniform border, so on a product on a panel it is a no-op with
  byte-identical output.

**When you control the frame, shape it to the source and use plain `WxH`** — a fit mode
adds nothing when the shapes agree. **Fit is for a cell you cannot shape**, a grid cell
whose size the layout fixes: there the image sits inside it undistorted and the cells
align. A fitted image is centred in its box — the template adds `fill(none)` — which keeps a row of tiles on one baseline.

## Overlays and text

`image(path, x, y[, alpha[, blend_mode]])` composites a processed image; its path is a
full imagor path, so it nests. `text(text, x, y, font, color)` draws a label. Positions
are relative to the **current canvas**, so inside a nested path that is the overlay, not
the parent.

```
fit-in/200x260/filters:fill(white)
  :image({crop}/fit-in/190x170/filters:upscale()/{IMAGE},center,10)
  :text({NAME},center,196,sans%2016,888888)
  /color:white
```

Accepted positions are imagor's own, and `list_imagor_filters` carries the full set — the
three filters share them. What it does not say is where they mislead: **`center` takes no
offset**, and `center-100` becomes a silent `0`; a position is the **top-left of the
box**, so `full-70` for a caption puts the *box* 70px from the bottom and the ink reaches
below it, wanting more room than the margin you asked for; and `right-20` moves **past**
the right edge while `-20` insets from it.

**A marker beside text is sized against the x-height, not the line.** A square dot tall
enough to span ascender to baseline is centred on the text's ink and still reads as low,
because the eye measures it against the lowercase beside it. For 15px type
the x-height is about 8px — match that, and check the dot's own rows against the text's,
not its centre against the text's centre.

`full` or `f` in a **nested dimension** inherits the parent: `fullxfull`,
`fit-in/full-20xfull-20`. A **position** has no such form — `full-N` written as one is
read as `0`, lands the element at the origin and returns `200`. The build tools warn when
they see one.

## Nested path order

Inside `image()` the path is a normal path, so crop and dimensions come before `filters:`.
`watermark()` is the older, narrower form of `image()`: the source, then `x`, `y` and
`alpha`, with optional width and height ratios in place of a nested path.

A source inside `image()` needs a full URL; a bare filename is a `404`, even when the same
name works as the top-level image. **`path:<imagor path>` is the other kind of source** —
an imagor path rather than a URL, resolved by the server already serving the request, so a
nested image nothing outside it can reach still loads. It needs imagor 1.9.10 — see *Which
imagor the server needs* for what an older server does with one.

Nested under nothing, `path:` only re-encodes: the server resolves the inner path to pixels and
encodes the result again, so the inner `quality()` and `format()` are not applied. Put the
transformation outside the wrapper, or leave the path bare. An image argument that is itself a
path is nested for you, and a wrapper with nothing outside it is collapsed for you.

## Colors

`color:white` is a canvas with no source: `image(311x35/color:f1f2ee,1096,600)` is a
rectangle. Stacking one behind an inset photo draws a border, which keeps the canvas size
under your control — `padding(color, N)` grows the output instead (300px becomes 340px at
`padding(x,20)`).

`round_corner` takes pixels, not a ratio: `round_corner(0.5)` is a half-pixel radius and
renders square. A circle is `round_corner(r,r,color)` with `r` half the width, and the
colour is required — leave it out and the corners come back black. Do not combine it with
`fit`: a fitted image letterboxes, so the corner is cut from the letterboxed box rather
than the photograph and the circle clips the picture. Use a square crop instead.

## When a value has to be encoded

**Text takes a whitelist.** Only `[A-Za-z0-9_-]` is left alone; anything else is encoded,
and markup characters escaped, because the renderer reads Pango markup in a text argument.

**A source takes a character set, plus reserved words.** A source is a path or URL and
legitimately contains `/`, `:`, `.` and `=`, so a whitelist would encode every real
source. Encoded are **space `?` `#` `&` `(` `)` `,`** — each changes how the path is read
— and a source **beginning with `trim/`, `meta/`, `fit-in/`, `full-fit-in/`,
`adaptive-fit-in/`, `stretch/`, `top/`, `left/`, `right/`, `bottom/`, `center/` or
`smart/`**, which imagor reads as a command.

Put text in `text` and a source in `source` and both are handled: the argument is data,
not syntax, and does not belong in a string parsed for syntax.

```json
{"name":"image","source":"https://cdn.example.com/a,b.png?v=2","args":"600x600,60,60"}
```

**What cannot be left alone is written `b64:`** — base64url, unpadded — which carries
arbitrary Unicode and multi-word strings past the rules above. It is imagor's syntax as much
as this server's, so a path read back may carry one, and the tools emit it where a value
would otherwise be mangled.

These rules match the converter in imagor-studio (`server/internal/imagortemplate`), so a path from either
project means the same thing.

## Templates

`render_template` renders a layout at a canvas size, so a layout is written once and
served at any size. A text slot's font resolves the way `text()` resolves it, and a text
slot given a height is **fitted against the box its geometry resolves to** — see *Placing
text* for both, and for what fitting costs and how to cap it.

A canvas, its slots, and the values that fill them:

```json
{"canvas_width": 1200, "canvas_height": 630, "background": "color:1a1a2e",
 "slots": [{"name":"bar","kind":"rect","image":"color:3b82f6","x":"0","y":"0",
            "width":"12","height":"full"}],
 "values": {"eyebrow":"ENGINEERING"}}
```

The cookbook holds fourteen of these, so reach for one before writing the JSON out.

**A slot's `filters` field applies filters to that slot's own source**, because the image
field is data rather than syntax — a path written there is encoded as a source. Write the
filters **without** the `filters:` prefix: `smart/round_corner(230,230,fff)`. Writing the
prefix too is accepted and ignored, so a doubled one is not an error you will see. The
signature is minted at render time, so the layout stays portable.

Slots apply in order, so later ones draw over earlier ones. Values fill
`{{placeholders}}` in text, image **and background** fields. A placeholder with no value
**refuses the render** rather than drawing its own braces — an unfilled one renders as
literal `{{name}}`, reading as a layout fault instead of a missing value.

**A slot carrying its own `slots` is a canvas.** Its width and height become the canvas the
inner slots lay out on, and the result draws at the outer slot's position — the way to
centre text on something narrower than the frame. Inner slots may be text, an image or a
rect, so a canvas holds a panel or a photograph as well as a label. It is transparent: it
adds only what it draws.

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

Only `full` and `full-N` exist in imagor as dimensions; the fraction and percentage forms
are the template layer's own, resolved to pixels before a path exists.

## Editing a path you already have

`parse_template` reads a path back into a template, so a composition you already have can
be edited rather than retyped. It is the inverse of `render_template`:

```
template JSON ──render_template──> path ──render──> image
     ▲                                                │
     └──────────── parse_template ◄──── path ◄────────┘
```

A form imagor can express comes back as a form: `full` and `full-80` survive. A form it
lacks has become a number, and a path cannot carry it:

| written | comes back as | why |
|---|---|---|
| `half`, `third` | `471`, `314` | imagor has no fraction |
| `38p` | `684` | imagor has no percentage |
| a text slot with a height | the size fitting chose | chosen before a path exists |

Rewrite those as forms if the template has to adapt to another canvas size. A path and a
URL both work, signed or not.

## Cookbook

Fourteen ready-made layouts, fetched by name with `get_cookbook_recipe`.

**Match on what a layout is for, not on its name**, which will not resemble the request.
Take the closest and change the values rather than composing from nothing: the recipes
carry spacing and type sizes that took iteration to get right, and the test suite renders
them at three canvas sizes, so one that no longer works cannot reach the index.

They reference their images from **`s.imagor.net`**, so they render as they are.

## Placing text is a search, not a calculation

Text lands where the font's metrics put it, and imagor does not report those.

- **A font name is a lookup that cannot fail loudly.** A generic resolves to a narrower
  face: `sans`, `serif` and `monospace` become `Open-Sans`, `Liberation-Serif` and
  `Liberation-Mono`, and the generics are DejaVu, the widest face imagor carries, so type
  sized for one overflows a box drawn for anything narrower. Name `DejaVu-Sans` to keep
  it. An uninstalled family and an absent weight fall back exactly as silently, both with
  a `200`: the official image's Open Sans carries Light, Regular, Semibold, Bold and
  ExtraBold, its Liberation only Regular and Bold, so `Open-Sans-Medium` renders as
  Regular and `Open-Sans-Black` as ExtraBold. Check by rendering a weight and comparing
  against Regular.
- **The font is a Pango description with hyphens for spaces.** `sans-bold-40` is `sans
  bold 40`: **family first, size last**. A number anywhere else is ignored — `sans-40-bold`
  renders at the default 20 — and a word before the family becomes it, so `bold-sans-40`
  renders as `sans-40`. A literal space is a **malformed URL**: use a hyphen or `b64:`.
- **A fitted text slot takes the largest size that fits its box.** That is what a headline
  wants and what a supporting line does not — a short line balloons to fill the box it was
  given. Cap it with `max_size`, or give the slot no height and a `size` instead.
- **To centre text on something narrower than the frame, give it a canvas of its own: a
  slot carrying its own `slots`.** `x=center` resolves against the canvas it is on, and a
  numeric `x` leaves text hanging left whatever `align` says. A label measured this way
  landed at centre 921 against a product centre of 922.
- **A wordmark is one `text()` call, even though that loses the tracking.** `text()` has no
  letter-spacing, and its `spacing` argument is *line* spacing. One glyph per call reaches
  the reference's spacing but hard-codes the string, so a different value breaks the
  layout. Prefer one call; the rest of the layout stays exact.

## Grids of images

Sizing each panel to its own source gives panels of different shapes, which reads as
unrelated pictures rather than a grid, and tuning positions does not fix it.

- **Give every cell one identical size, on even rows and columns, and fit the image inside
  its cell** (`fit`). The cells align; the images inside them do not fill them, so nothing
  is distorted and the space around each is even.
- **Percentage forms make the sameness checkable**: five cells of `16p+12 x 26p+5` stay one
  size at every canvas; five of 300x250 do not.

## Responsive compositions

`fit-in/full-40xfull-40/PHOTO` insets 40px on every side, `12xfull/color:3b82f6` is an
accent bar at full height, and `full-80` as a `text()` wrap width wraps to the canvas it
lands on.

**Sizes adapt; type does not.** A font size is in pixels, so the same composition renders
relatively smaller type on a larger canvas, and there is no percentage font size. Size the
text per canvas by calling **`fit_text`** — it finds the largest size at which a string
still fits a box and returns the font argument for `text()` — or accept that it will not
scale.

## When imagor refuses to serve

A `403` with `url signature mismatch` is a configuration disagreement, not a broken
request. Three settings have to line up:

- **Unsigned or signed.** `IMAGOR_UNSAFE=1` sends no signature; `IMAGOR_SECRET` signs. A
  client sending no signature to a signed imagor is refused; a client that signs against
  an unsafe imagor is accepted.
- **`IMAGOR_SIGNER_TYPE`** — `sha1` (the default), `sha256` or `sha512`.
- **`IMAGOR_SIGNER_TRUNCATE`** — signature length in characters; `0` means none.

A signature covers the whole path, so any difference in those three produces the same
refusal and neither side can read the other's secret back. `describe_imagor` reports what
this server is using — compare before debugging anything else.

## Which imagor the server needs

**imagor 1.9.10 or later.** Two features this tooling leans on arrived in specific
releases:

- **1.9.8** — `/meta` reports what each filter did. Before it there is nothing to verify
  against, and every outcome comes back `unknown`.
- **1.9.10** — a nested path source, `path:<imagor path>`, resolves in-process.

On an older server `path:` is not syntax, so the token is read as part of the image name
and the **whole request is refused with `400 invalid`**, naming no part of the path. That
is indistinguishable from a genuinely bad composition, so `verify_imagor_url` probes for
the syntax first and refuses with the release named rather than letting you debug the
composition.

The probe answers on behaviour, not a version string: imagor reports its version in a
startup log and on no endpoint, so there is nothing to ask.

## Working method

1. `describe_imagor` once, for the grammar this server targets, and `list_imagor_filters`
   before using a filter you have not used here — imagor ignores an unrecognised one
   silently, and the request comes back `200` with the image unchanged.
2. Build with `build_imagor_url`, which refuses an invalid composition rather than emitting
   it, so a refusal is a mistake worth fixing rather than working around. Its `path` field
   is the bare path with no prefix and no signature, for a caller that signs itself; `url`
   and `absolute_url` carry the prefix and signature the server is configured for.
   `validate_imagor_path` reads a path back to report whether it round-trips.
3. `verify_imagor_url` the result, passing the same typed parameters you built from — it
   takes parameters, not a URL. An outcome is `applied`, `declined` (the server recognised
   the filter and left the image unchanged) or one of the two that mean nothing can be
   concluded — `not_reported` and `unknown`. Neither of those last two is a failure; a
   `declined` is, and says so itself.
4. **Look at what came back** — `applied` does not mean correct. Where the preview is too
   small to read a label, call again with `preview_region` as `x,y,w,h` in **result**
   pixels: full resolution for a fraction of the cost. `preview: false` for the report
   alone. Iterate until it matches.
5. To composite, build small pieces first — one tile, then the grid, then the banner. A
   wrong number in a nested path is much harder to see in the finished image.
6. Measure only what looking cannot settle, and measure the **rows**, not the centres: a
   dot's rows against its text's, a panel's against its neighbour's. Two centres can agree
   to the pixel while the shapes read as misaligned. Crop to the region you changed before
   you believe the number.
7. **Give the url to the user, in full, alongside the picture.** The url is the output;
   the image you checked is a scaled preview of it. Showing the result and dropping the
   url leaves them with nothing they can serve.

**These rules are procedures and facts on purpose.** A procedure says what to type and is
followed; a preference asks the reader to want the right thing and is not.
