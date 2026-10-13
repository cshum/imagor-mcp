# Cookbook

Fourteen layouts, each one a single imagor URL. **The template is the whole
input** — it is complete on its own:

- **The `note`** says what the card is for, so a change has something to be
  judged against.
- **The `slots`** are the layout. Names, geometry relative to the canvas, and
  the type.
- **The `values`** are the content, filled into the `{{placeholders}}`.

Hand that to `render_template` and it resolves to one imagor URL, shown under
each card. A path can be read back with `parse_template`, but what comes back
renders once rather than re-filling: the names and the `values` are the part a
path cannot hold.

[Back to the README](../README.md) · [All fourteen, one image](contact-sheet.png)

## announcement

![announcement](announcement.png)

*1200×628* — Headline, one supporting line, one visual. The tech/startup announcement card.

**Template** — the source. Copy it and iterate:

```json
{
  "name": "announcement",
  "note": "Headline, one supporting line, one visual. The tech/startup announcement card.",
  "canvas": [1200, 628],
  "background": "color:142e2b",
  "slots": [
    {"name": "eyebrow", "kind": "text", "text": "{{eyebrow}}", "x": "7p+4", "y": "14p", "font": "Open-Sans-bold-18", "color": "d2f17f"},
    {"name": "headline-accent", "kind": "rect", "image": "color:d2f17f", "x": "7p+4", "y": "22p", "width": "84", "height": "5"},
    {"name": "headline", "kind": "text", "text": "{{headline}}", "x": "7p+4", "y": "28p+4", "width": "59p", "height": "34p", "font": "Liberation-Serif", "color": "f4f1e8", "max_size": 68, "min_size": 52},
    {"name": "supporting", "kind": "text", "text": "{{supporting}}", "x": "7p+4", "y": "62p", "width": "58p", "height": "12p", "font": "Open-Sans", "color": "b3bec0", "max_size": 25, "min_size": 20},
    {"name": "status-panel", "kind": "rect", "image": "color:1d3b34", "x": "68p+4", "y": "22p", "width": "300", "height": "352"},
    {"name": "status-mark-background", "kind": "rect", "image": "color:d2f17f", "x": "76p+4", "y": "30p", "width": "108", "height": "107"},
    {"name": "status-mark", "kind": "text", "text": "\u2713", "x": "78p+4", "y": "30p", "font": "Open-Sans-bold-72", "color": "1d3b34"},
    {"name": "status-track-one", "kind": "rect", "image": "color:2f4a42", "x": "71p+4", "y": "58p", "width": "204", "height": "5"},
    {"name": "status-progress-one", "kind": "rect", "image": "color:d2f17f", "x": "71p+4", "y": "58p", "width": "120", "height": "5"},
    {"name": "status-indicator-one", "kind": "rect", "image": "color:d2f17f", "x": "90p", "y": "58p", "width": "12", "height": "12"},
    {"name": "status-track-two", "kind": "rect", "image": "color:2f4a42", "x": "71p+4", "y": "64p", "width": "204", "height": "5"},
    {"name": "status-progress-two", "kind": "rect", "image": "color:d2f17f", "x": "71p+4", "y": "64p", "width": "168", "height": "5"},
    {"name": "status-indicator-two", "kind": "rect", "image": "color:d2f17f", "x": "90p", "y": "64p", "width": "12", "height": "12"},
    {"name": "status-track-three", "kind": "rect", "image": "color:2f4a42", "x": "71p+4", "y": "70p", "width": "204", "height": "5"},
    {"name": "status-progress-three", "kind": "rect", "image": "color:d2f17f", "x": "71p+4", "y": "70p", "width": "96", "height": "5"},
    {"name": "status-indicator-three", "kind": "rect", "image": "color:d2f17f", "x": "90p", "y": "70p", "width": "12", "height": "12"},
    {"name": "mark", "kind": "text", "text": "{{mark}}", "x": "7p+4", "y": "80p", "font": "Open-Sans-bold-20", "color": "d2f17f"}
  ],
  "values": {
    "eyebrow": "IMAGOR 1.9.8",
    "headline": "Every filter now reports whether it ran",
    "supporting": "A filter that did nothing used to look like one that worked.",
    "mark": "imagor"
  }
}
```

**Resolves to** — the one imagor URL it produces. Copy it to render:

```
1200x628/filters:text(b64:SU1BR09SIDEuOS44,88,87,Open-Sans-bold-18,d2f17f):image(84x5/color:d2f17f,88,138):text(b64:RXZlcnkgZmlsdGVyIG5vdyByZXBvcnRzIHdoZXRoZXIgaXQgcmFu,88,179,Liberation-Serif-68,f4f1e8,0,normal,708,low,normal,word):text(b64:QSBmaWx0ZXIgdGhhdCBkaWQgbm90aGluZyB1c2VkIHRvIGxvb2sgbGlrZSBvbmUgdGhhdCB3b3JrZWQu,88,389,Open-Sans-25,b3bec0,0,normal,696,low,normal,word):image(300x352/color:1d3b34,820,138):image(108x107/color:d2f17f,916,188):text(b64:4pyT,940,188,Open-Sans-bold-72,1d3b34):image(204x5/color:2f4a42,856,364):image(120x5/color:d2f17f,856,364):image(12x12/color:d2f17f,1080,364):image(204x5/color:2f4a42,856,401):image(168x5/color:d2f17f,856,401):image(12x12/color:d2f17f,1080,401):image(204x5/color:2f4a42,856,439):image(96x5/color:d2f17f,856,439):image(12x12/color:d2f17f,1080,439):text(imagor,88,502,Open-Sans-bold-20,d2f17f)/color:142e2b
```

## comparison

![comparison](comparison.png)

*1200×630* — Two matched columns of points, for a before and after.

**Template** — the source. Copy it and iterate:

```json
{
  "name": "Comparison card",
  "note": "Two matched columns of points, for a before and after.",
  "canvas": [1200, 630],
  "background": "color:f7f5ef",
  "slots": [
    {"name": "brand", "kind": "text", "text": "{{brand}}", "x": "5p+4", "y": "6p+6", "font": "Open-Sans-bold-18", "color": "213d36"},
    {"name": "top_rule", "kind": "rect", "image": "color:c58e43", "x": "5p+4", "y": "13p", "width": "3p+8", "height": "0p+3"},
    {"name": "headline", "kind": "text", "text": "{{headline}}", "x": "5p+4", "y": "16p+4", "width": "full-128", "height": "11p+1", "font": "Open-Sans-bold", "max_size": 45, "min_size": 32, "color": "182a27", "align": "low"},
    {"name": "left_panel", "kind": "rect", "image": "color:efede6", "x": "5p+4", "y": "34p+4", "width": "43p+8", "height": "53p"},
    {"name": "right_panel", "kind": "rect", "image": "color:1d443c", "x": "51p", "y": "34p+4", "width": "43p+8", "height": "53p"},
    {"name": "left_accent", "kind": "rect", "image": "color:c58e43", "x": "5p+4", "y": "34p+4", "width": "0p+6", "height": "53p"},
    {"name": "right_accent", "kind": "rect", "image": "color:79c7a6", "x": "51p", "y": "34p+4", "width": "0p+6", "height": "53p"},
    {"name": "left_label", "kind": "text", "text": "{{left_label}}", "x": "7p+8", "y": "38p+6", "font": "Open-Sans-bold-24", "color": "756f62"},
    {"name": "right_label", "kind": "text", "text": "{{right_label}}", "x": "53p+4", "y": "38p+6", "font": "Open-Sans-bold-24", "color": "a5e4c6"},
    {"name": "left_label_rule", "kind": "rect", "image": "color:d1cdc2", "x": "7p+8", "y": "44p+6", "width": "39p", "height": "0p+1"},
    {"name": "right_label_rule", "kind": "rect", "image": "color:45685f", "x": "53p+4", "y": "44p+6", "width": "39p", "height": "0p+1"},
    {"name": "left_bullet_1", "kind": "rect", "image": "color:c58e43", "x": "7p+10", "y": "50p+2", "width": "0p+8", "height": "1p+2"},
    {"name": "left_point_1", "kind": "text", "text": "{{left_point_1}}", "x": "9p+10", "y": "49p+3", "width": "36p+8", "font": "Open-Sans-18", "color": "3d403b"},
    {"name": "right_bullet_1", "kind": "rect", "image": "color:79c7a6", "x": "53p+6", "y": "50p+2", "width": "0p+8", "height": "1p+2"},
    {"name": "right_point_1", "kind": "text", "text": "{{right_point_1}}", "x": "55p+6", "y": "49p+3", "width": "36p+8", "font": "Open-Sans-18", "color": "eef4ef"},
    {"name": "left_bullet_2", "kind": "rect", "image": "color:c58e43", "x": "7p+10", "y": "62p+3", "width": "0p+8", "height": "1p+2"},
    {"name": "left_point_2", "kind": "text", "text": "{{left_point_2}}", "x": "9p+10", "y": "61p+5", "width": "36p+8", "font": "Open-Sans-18", "color": "3d403b"},
    {"name": "right_bullet_2", "kind": "rect", "image": "color:79c7a6", "x": "53p+6", "y": "62p+3", "width": "0p+8", "height": "1p+2"},
    {"name": "right_point_2", "kind": "text", "text": "{{right_point_2}}", "x": "55p+6", "y": "61p+5", "width": "36p+8", "font": "Open-Sans-18", "color": "eef4ef"},
    {"name": "left_bullet_3", "kind": "rect", "image": "color:c58e43", "x": "7p+10", "y": "74p+5", "width": "0p+8", "height": "1p+2"},
    {"name": "left_point_3", "kind": "text", "text": "{{left_point_3}}", "x": "9p+10", "y": "73p+6", "width": "36p+8", "font": "Open-Sans-18", "color": "3d403b"},
    {"name": "right_bullet_3", "kind": "rect", "image": "color:79c7a6", "x": "53p+6", "y": "74p+5", "width": "0p+8", "height": "1p+2"},
    {"name": "right_point_3", "kind": "text", "text": "{{right_point_3}}", "x": "55p+6", "y": "73p+6", "width": "36p+8", "font": "Open-Sans-18", "color": "eef4ef"},
    {"name": "footer", "kind": "text", "text": "{{footer}}", "x": "5p+4", "y": "91p+4", "font": "Open-Sans-bold-14", "color": "7b7e76"}
  ],
  "values": {
    "brand": "SABLON",
    "headline": "What changed in the last release",
    "left_label": "Before",
    "left_point_1": "A filter that did nothing looked like one that worked",
    "left_point_2": "Errors were reported after the response",
    "left_point_3": "No way to tell a working URL from an inert one",
    "right_label": "After",
    "right_point_1": "Every filter reports whether it ran",
    "right_point_2": "A refusal arrives before the render",
    "right_point_3": "The catalogue is generated from imagor itself",
    "footer": "imagor"
  }
}
```

**Resolves to** — the one imagor URL it produces. Copy it to render:

```
1200x630/filters:text(SABLON,64,43,Open-Sans-bold-18,213d36):image(44x3/color:c58e43,64,81):text(b64:V2hhdCBjaGFuZ2VkIGluIHRoZSBsYXN0IHJlbGVhc2U,64,104,Open-Sans-bold-45,182a27,0,normal,full-128,low,normal,word):image(524x333/color:efede6,64,218):image(524x333/color:1d443c,612,218):image(6x333/color:c58e43,64,218):image(6x333/color:79c7a6,612,218):text(Before,92,245,Open-Sans-bold-24,756f62):text(After,640,245,Open-Sans-bold-24,a5e4c6):image(468x1/color:d1cdc2,92,283):image(468x1/color:45685f,640,283):image(8x8/color:c58e43,94,317):text(b64:QSBmaWx0ZXIgdGhhdCBkaWQgbm90aGluZyBsb29rZWQgbGlrZSBvbmUgdGhhdCB3b3JrZWQ,118,311,Open-Sans-18,3d403b,0,normal,440,low,normal,word):image(8x8/color:79c7a6,642,317):text(b64:RXZlcnkgZmlsdGVyIHJlcG9ydHMgd2hldGhlciBpdCByYW4,666,311,Open-Sans-18,eef4ef,0,normal,440,low,normal,word):image(8x8/color:c58e43,94,393):text(b64:RXJyb3JzIHdlcmUgcmVwb3J0ZWQgYWZ0ZXIgdGhlIHJlc3BvbnNl,118,389,Open-Sans-18,3d403b,0,normal,440,low,normal,word):image(8x8/color:79c7a6,642,393):text(b64:QSByZWZ1c2FsIGFycml2ZXMgYmVmb3JlIHRoZSByZW5kZXI,666,389,Open-Sans-18,eef4ef,0,normal,440,low,normal,word):image(8x8/color:c58e43,94,471):text(b64:Tm8gd2F5IHRvIHRlbGwgYSB3b3JraW5nIFVSTCBmcm9tIGFuIGluZXJ0IG9uZQ,118,465,Open-Sans-18,3d403b,0,normal,440,low,normal,word):image(8x8/color:79c7a6,642,471):text(b64:VGhlIGNhdGFsb2d1ZSBpcyBnZW5lcmF0ZWQgZnJvbSBpbWFnb3IgaXRzZWxm,666,465,Open-Sans-18,eef4ef,0,normal,440,low,normal,word):text(imagor,64,577,Open-Sans-bold-14,7b7e76)/color:f7f5ef
```

## event

![event](event.png)

*1200×628* — Date block, title, venue. The open-house / meetup format.

**Template** — the source. Copy it and iterate:

```json
{
  "name": "event",
  "note": "Date block, title, venue. The open-house / meetup format.",
  "canvas": [1200, 628],
  "background": "color:f3f0e8",
  "slots": [
    {"name": "date-card", "kind": "rect", "image": "color:174c43", "x": "5p+4", "y": "14p", "width": "21p", "height": "71p+6"},
    {"name": "date-accent", "kind": "rect", "image": "color:e6a26b", "x": "8p", "y": "29p+2", "width": "4p+4", "height": "0p+4"},
    {"name": "day", "kind": "text", "text": "{{day}}", "x": "8p", "y": "35p", "font": "Open-Sans-bold", "size": 118, "color": "fffaf1"},
    {"name": "month", "kind": "text", "text": "{{month}}", "x": "8p+4", "y": "55p+3", "font": "Open-Sans-bold", "size": 34, "color": "d9e5d9"},
    {"name": "title", "kind": "text", "text": "{{title}}", "x": "31p", "y": "28p+6", "width": "63p+8", "height": "27p+4", "font": "Liberation-Serif", "color": "173c36", "max_size": 76, "min_size": 44},
    {"name": "title-rule", "kind": "rect", "image": "color:d47b54", "x": "31p", "y": "58p+2", "width": "6p", "height": "0p+4"},
    {"name": "venue", "kind": "text", "text": "{{venue}}", "x": "31p", "y": "63p+5", "font": "Open-Sans", "size": 30, "color": "49645d"}
  ],
  "values": {
    "day": "14",
    "month": "NOV",
    "title": "Deterministic image pipelines",
    "venue": "Harbour office, 6pm"
  }
}
```

**Resolves to** — the one imagor URL it produces. Copy it to render:

```
1200x628/filters:image(252x451/color:174c43,64,87):image(52x4/color:e6a26b,96,184):text(14,96,219,Open-Sans-bold-118,fffaf1):text(NOV,100,348,Open-Sans-bold-34,d9e5d9):text(b64:RGV0ZXJtaW5pc3RpYyBpbWFnZSBwaXBlbGluZXM,372,181,Liberation-Serif-76,173c36,0,normal,764,low,normal,word):image(72x4/color:d47b54,372,366):text(b64:SGFyYm91ciBvZmZpY2UsIDZwbQ,372,400,Open-Sans-30,49645d)/color:f3f0e8
```

## launch-card

![launch-card](launch-card.png)

*1200×630* — Editorial split layout for a seasonal accessories collection announcement.

**Template** — the source. Copy it and iterate:

```json
{
  "name": "launch-card",
  "note": "Editorial split layout for a seasonal accessories collection announcement.",
  "canvas": [1200, 630],
  "background": "color:f5f1e9",
  "slots": [
    {"name": "hero-photo", "kind": "image", "image": "{{photo}}", "x": "61p+6", "y": "8p+5", "width": "32p", "height": "82p+3", "fit": true},
    {"name": "brand", "kind": "text", "text": "{{brand}}", "x": "7p+11", "y": "14p+3", "font": "Open-Sans", "size": 17, "color": "302a25"},
    {"name": "brand-rule", "kind": "rect", "image": "color:a7654b", "x": "7p+11", "y": "22p", "width": "4p+4", "height": "0p+2"},
    {"name": "headline", "kind": "text", "text": "{{headline}}", "x": "7p+8", "y": "32p+5", "width": "49p+8", "height": "19p", "font": "Liberation-Serif", "color": "302a25", "align": "low", "max_size": 76, "min_size": 54},
    {"name": "supporting", "kind": "text", "text": "{{supporting}}", "x": "7p+11", "y": "55p+2", "width": "46p+3", "font": "Open-Sans", "size": 19, "color": "746b61"},
    {"name": "cta-button", "kind": "rect", "image": "color:302a25", "x": "7p+11", "y": "68p+5", "width": "20p", "height": "8p+4"},
    {"name": "cta-label", "kind": "text", "text": "{{cta}}", "x": "9p+8", "y": "71p+4", "width": "17p", "font": "Open-Sans", "size": 15, "color": "f5f1e9"},
    {"name": "availability", "kind": "text", "text": "{{availability}}", "x": "7p+11", "y": "86p+2", "width": "46p+3", "font": "Open-Sans", "size": 15, "color": "746b61"}
  ],
  "values": {
    "photo": "https://s.imagor.net/scene-tall.svg",
    "brand": "SABLON",
    "headline": "The Autumn Edit",
    "supporting": "Twelve pieces in vegetable-tanned leather",
    "cta": "Explore the collection",
    "availability": "Available from 14 November"
  }
}
```

**Resolves to** — the one imagor URL it produces. Copy it to render:

```
1200x630/filters:image(fit-in/384x519/filters:fill(none)/https://s.imagor.net/scene-tall.svg,738,55):text(SABLON,95,91,Open-Sans-17,302a25):image(52x2/color:a7654b,95,138):text(b64:VGhlIEF1dHVtbiBFZGl0,92,206,Liberation-Serif-76,302a25,0,normal,596,low,normal,word):text(b64:VHdlbHZlIHBpZWNlcyBpbiB2ZWdldGFibGUtdGFubmVkIGxlYXRoZXI,95,348,Open-Sans-19,746b61,0,normal,555,low,normal,word):image(240x54/color:302a25,95,433):text(b64:RXhwbG9yZSB0aGUgY29sbGVjdGlvbg,116,451,Open-Sans-15,f5f1e9,0,normal,204,low,normal,word):text(b64:QXZhaWxhYmxlIGZyb20gMTQgTm92ZW1iZXI,95,543,Open-Sans-15,746b61,0,normal,555,low,normal,word)/color:f5f1e9
```

## news-announcement

![news-announcement](news-announcement.png)

*1200×630* — A restrained, editorial announcement for a product return, pairing a prominent headline with a supporting quote and dateline.

**Template** — the source. Copy it and iterate:

```json
{
  "name": "Maison editorial news announcement",
  "note": "A restrained, editorial announcement for a product return, pairing a prominent headline with a supporting quote and dateline.",
  "canvas": {"width": 1200, "height": 630, "background": "color:f6f4ed"},
  "slots": [
    {"name": "kicker", "kind": "text", "text": "{{kicker}}", "x": "7p", "y": "12p", "font": "Open-Sans-bold", "size": 18, "color": "526653"},
    {"name": "kicker-rule", "kind": "rect", "image": "color:526653", "x": "7p", "y": "18p", "width": "4p", "height": "2"},
    {"name": "headline", "kind": "text", "text": "{{headline}}", "x": "7p", "y": "22p", "width": "86p", "height": "120", "font": "Open-Sans-bold", "color": "20251f", "max_size": 66, "min_size": 56},
    {"name": "quote-rule", "kind": "rect", "image": "color:9a7958", "x": "7p", "y": "46p", "width": "3", "height": "29"},
    {"name": "quote", "kind": "text", "text": "{{quote}}", "x": "9p", "y": "46p", "width": "84p", "height": "102", "font": "Liberation-Serif", "color": "343b34", "max_size": 32, "min_size": 27},
    {"name": "source", "kind": "text", "text": "{{source}}", "x": "9p", "y": "70p", "font": "Open-Sans", "size": 16, "color": "697064"},
    {"name": "footer-rule", "kind": "rect", "image": "color:d7d6cc", "x": "7p", "y": "87p", "width": "86p", "height": "1"}
  ],
  "values": {
    "kicker": "SABLON",
    "headline": "Hand-woven leather, back in stock",
    "quote": "The same bag, made the same way, for nineteen years.",
    "source": "Hong Kong, October 2026"
  }
}
```

**Resolves to** — the one imagor URL it produces. Copy it to render:

```
1200x630/filters:text(SABLON,84,75,Open-Sans-bold-18,526653):image(48x2/color:526653,84,113):text(b64:SGFuZC13b3ZlbiBsZWF0aGVyLCBiYWNrIGluIHN0b2Nr,84,138,Open-Sans-bold-60,20251f,0,normal,1032,low,normal,word):image(3x29/color:9a7958,84,289):text(b64:VGhlIHNhbWUgYmFnLCBtYWRlIHRoZSBzYW1lIHdheSwgZm9yIG5pbmV0ZWVuIHllYXJzLg,108,289,Liberation-Serif-32,343b34,0,normal,1008,low,normal,word):text(b64:SG9uZyBLb25nLCBPY3RvYmVyIDIwMjY,108,441,Open-Sans-16,697064):image(1032x1/color:d7d6cc,84,548)/color:f6f4ed
```

## og-standard

![og-standard](og-standard.png)

*1200×630* — A text-first 1200x630 Open Graph layout with a small brand mark, prominent title, supporting description, and quiet domain footer.

**Template** — the source. Copy it and iterate:

```json
{
  "name": "og-standard",
  "note": "A text-first 1200x630 Open Graph layout with a small brand mark, prominent title, supporting description, and quiet domain footer.",
  "canvas": [1200, 630],
  "background": "color:f4f0e7",
  "slots": [
    {"name": "brand", "kind": "text", "text": "{{brand}}", "x": "7p+8", "y": "11p+3", "font": "Open-Sans-bold-24", "color": "263a32"},
    {"name": "brand_rule", "kind": "rect", "image": "color:c58b4b", "x": "7p+8", "y": "18p+3", "width": "4p+6", "height": "0p+3"},
    {"name": "title", "kind": "text", "text": "{{title}}", "x": "7p+8", "y": "30p+3", "width": "80p+10", "height": "26p+2", "font": "Liberation-Serif", "color": "263a32", "max_size": 82, "min_size": 60},
    {"name": "description", "kind": "text", "text": "{{description}}", "x": "7p+8", "y": "59p", "width": "81p+8", "height": "13p", "font": "Open-Sans", "color": "4e5b52", "max_size": 32, "min_size": 30},
    {"name": "footer_rule", "kind": "rect", "image": "color:d8d0c1", "x": "7p+8", "y": "84p+2", "width": "84p+8", "height": "0p+1"},
    {"name": "domain", "kind": "text", "text": "{{domain}}", "x": "7p+8", "y": "87p+4", "font": "Open-Sans-20", "color": "58645b"}
  ],
  "values": {
    "brand": "SABLON",
    "title": "The Autumn Edit is here",
    "description": "Twelve pieces in vegetable-tanned leather, made in small batches",
    "domain": "sablon.example"
  }
}
```

**Resolves to** — the one imagor URL it produces. Copy it to render:

```
1200x630/filters:text(SABLON,92,72,Open-Sans-bold-24,263a32):image(54x3/color:c58b4b,92,116):text(b64:VGhlIEF1dHVtbiBFZGl0IGlzIGhlcmU,92,192,Liberation-Serif-82,263a32,0,normal,970,low,normal,word):text(b64:VHdlbHZlIHBpZWNlcyBpbiB2ZWdldGFibGUtdGFubmVkIGxlYXRoZXIsIG1hZGUgaW4gc21hbGwgYmF0Y2hlcw,92,371,Open-Sans-32,4e5b52,0,normal,980,low,normal,word):image(1016x1/color:d8d0c1,92,531):text(b64:c2FibG9uLmV4YW1wbGU,92,552,Open-Sans-20,58645b)/color:f4f0e7
```

## quote-card

![quote-card](quote-card.png)

*1200×630* — A pull quote on deep green: the quotation in serif, attributed below a rule.

**Template** — the source. Copy it and iterate:

```json
{
  "name": "Quote card",
  "note": "A pull quote on deep green: the quotation in serif, attributed below a rule.",
  "source": "A daily quote card with a flat, typographic treatment (quote, attribution, role)",
  "canvas": {"width": 1200, "height": 630, "background": "color:142E2B"},
  "slots": [
    {"name": "panel", "kind": "rect", "image": "color:F4F0E6", "x": "4p", "y": "6p+6", "width": "full-96", "height": "full-88"},
    {"name": "accent", "kind": "rect", "image": "color:C57B4A", "x": "4p", "y": "6p+6", "width": "1p", "height": "full-88"},
    {"name": "mark", "kind": "text", "text": "\u201c", "x": "11p+4", "y": "13p", "font": "Liberation-Serif", "color": "C57B4A", "size": 96},
    {"name": "quote", "kind": "text", "text": "{{quote}}", "x": "11p+8", "y": "26p+2", "width": "75p", "height": "34p+6", "font": "Liberation-Serif", "color": "173C37", "max_size": 72, "min_size": 52},
    {"name": "rule", "kind": "rect", "image": "color:C57B4A", "x": "11p+8", "y": "63p+5", "width": "5p+4", "height": "0p+4"},
    {"name": "attribution", "kind": "text", "text": "{{attribution}}", "x": "11p+8", "y": "67p+4", "font": "Open-Sans", "color": "244F47", "size": 24},
    {"name": "role", "kind": "text", "text": "{{role}}", "x": "11p+8", "y": "73p+4", "font": "Open-Sans", "color": "687A72", "size": 18}
  ],
  "values": {
    "quote": "The same bytes in,\nthe same bytes out.",
    "attribution": "Mei Lam, Harbour",
    "role": "Platform engineering, Harbour"
  }
}
```

**Resolves to** — the one imagor URL it produces. Copy it to render:

```
1200x630/filters:image(full-96xfull-88/color:F4F0E6,48,43):image(12xfull-88/color:C57B4A,48,43):text(b64:4oCc,136,81,Liberation-Serif-96,C57B4A):text(b64:VGhlIHNhbWUgYnl0ZXMgaW4sCnRoZSBzYW1lIGJ5dGVzIG91dC4,140,165,Liberation-Serif-72,173C37,0,normal,900,low,normal,word):image(64x4/color:C57B4A,140,401):text(b64:TWVpIExhbSwgSGFyYm91cg,140,426,Open-Sans-24,244F47):text(b64:UGxhdGZvcm0gZW5naW5lZXJpbmcsIEhhcmJvdXI,140,463,Open-Sans-18,687A72)/color:142E2B
```

## testimonial

![testimonial](testimonial.png)

*1200×628* — B2B customer quote with an inset forest panel, warm paper margins and restrained brass accents.

**Template** — the source. Copy it and iterate:

```json
{
  "name": "testimonial",
  "note": "B2B customer quote with an inset forest panel, warm paper margins and restrained brass accents.",
  "canvas": [1200, 628],
  "background": "color:f2efe6",
  "slots": [
    {"name": "panel", "kind": "rect", "image": "color:142c29", "x": "5p", "y": "8p-2", "width": "90p", "height": "84p+4"},
    {"name": "eyebrow-rule", "kind": "rect", "image": "color:c6a76a", "x": "10p+6", "y": "16p+3", "width": "34", "height": "2"},
    {"name": "eyebrow", "kind": "text", "text": "CUSTOMER STORY", "x": "15p", "y": "14p+7", "font": "Open-Sans-bold-14", "color": "c6a76a"},
    {"name": "quote-mark", "kind": "text", "text": "\u201c", "x": "10p+6", "y": "21p+2", "font": "Liberation-Serif-72", "color": "c6a76a"},
    {"name": "quote", "kind": "text", "text": "{{quote}}", "x": "10p+6", "y": "27p", "width": "80p", "height": "35p+3", "font": "Liberation-Serif", "color": "f7f4ec", "max_size": 62, "min_size": 42},
    {"name": "rule", "kind": "rect", "image": "color:c6a76a", "x": "10p+6", "y": "58p+4", "width": "48", "height": "2"},
    {"name": "name", "kind": "text", "text": "{{name}}", "x": "10p+6", "y": "62p+1", "font": "Open-Sans-bold-25", "color": "ffffff"},
    {"name": "company", "kind": "text", "text": "{{company}}", "x": "10p+6", "y": "69p", "font": "Open-Sans-18", "color": "b8c4bd"}
  ],
  "values": {
    "quote": "We replaced a queue of image workers with one URL.",
    "name": "Mei Lam",
    "company": "Platform engineering, Harbour"
  }
}
```

**Resolves to** — the one imagor URL it produces. Copy it to render:

```
1200x628/filters:image(1080x531/color:142c29,60,48):image(34x2/color:c6a76a,126,103):text(b64:Q1VTVE9NRVIgU1RPUlk,180,94,Open-Sans-bold-14,c6a76a):text(b64:4oCc,126,133,Liberation-Serif-72,c6a76a):text(b64:V2UgcmVwbGFjZWQgYSBxdWV1ZSBvZiBpbWFnZSB3b3JrZXJzIHdpdGggb25lIFVSTC4,126,169,Liberation-Serif-62,f7f4ec,0,normal,960,low,normal,word):image(48x2/color:c6a76a,126,368):text(b64:TWVpIExhbQ,126,390,Open-Sans-bold-25,ffffff):text(b64:UGxhdGZvcm0gZW5naW5lZXJpbmcsIEhhcmJvdXI,126,433,Open-Sans-18,b8c4bd)/color:f2efe6
```

## tip

![tip](tip.png)

*1200×628* — One actionable tip per card. The educational format.

**Template** — the source. Copy it and iterate:

```json
{
  "name": "tip",
  "note": "One actionable tip per card. The educational format.",
  "canvas": [1200, 628],
  "background": "color:102c2a",
  "slots": [
    {"name": "number", "kind": "text", "text": "{{number}}", "x": "8p+4", "y": "14p", "font": "Open-Sans-bold-26", "color": "e5a15b"},
    {"name": "top-rule", "kind": "rect", "image": "color:456a60", "x": "12p+10", "y": "18p+3", "width": "78p+10", "height": "0p+2"},
    {"name": "quote-accent", "kind": "rect", "image": "color:e5a15b", "x": "8p+4", "y": "30p+4", "width": "0p+5", "height": "22p+6"},
    {"name": "tip", "kind": "text", "text": "{{tip}}", "x": "11p+2", "y": "31p+5", "width": "80p+6", "height": "35p+4", "font": "Liberation-Serif", "color": "fff7e8", "max_size": 70, "min_size": 48},
    {"name": "mark", "kind": "text", "text": "{{mark}}", "x": "8p+4", "y": "81p+1", "font": "Open-Sans-bold-22", "color": "e5a15b"}
  ],
  "values": {
    "number": "01",
    "tip": "A signature covers the whole path, not the source.",
    "mark": "imagor"
  }
}
```

**Resolves to** — the one imagor URL it produces. Copy it to render:

```
1200x628/filters:text(01,100,87,Open-Sans-bold-26,e5a15b):image(946x2/color:456a60,154,116):image(5x144/color:e5a15b,100,192):text(b64:QSBzaWduYXR1cmUgY292ZXJzIHRoZSB3aG9sZSBwYXRoLCBub3QgdGhlIHNvdXJjZS4,134,199,Liberation-Serif-70,fff7e8,0,normal,966,low,normal,word):text(imagor,100,509,Open-Sans-bold-22,e5a15b)/color:102c2a
```
