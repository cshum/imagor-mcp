# Cookbook

Fourteen layouts, each one a single imagor URL. Every card below comes with the
three things it takes to work on it:

- **What it is** — so a change can be judged against something.
- **The template** — what `render_template` takes. **Copy this to iterate.** It
  keeps the slot names and the `values` placeholders, so a card can be re-filled
  for someone else.

Render one and it comes back as a single imagor URL. Every card here is that and
nothing else - no design file, no template editor. **The template is the source;
the URL is what it resolves to:** pass a template to `render_template` and the
path is returned. A path can be read back with `parse_template`, but what comes
back renders once rather than re-filling - the names and the `values`
placeholders are the part a path cannot hold.

[Back to the README](../README.md) · [All fourteen, one image](contact-sheet.png)

## announcement

![announcement](announcement.png)

*1200×628* — Headline, one supporting line, one visual. The tech/startup announcement card.

**Template** — copy this to iterate:

```json
{
"name": "announcement",
"note": "Headline, one supporting line, one visual. The tech/startup announcement card.",
"canvas": [
1200,
628
],
"background": "color:142e2b",
"slots": [
{
"name": "eyebrow",
"kind": "text",
"text": "{{eyebrow}}",
"x": "7p+4",
"y": "14p",
"font": "Open-Sans-bold-18",
"color": "d2f17f"
},
{
"name": "headline-accent",
"kind": "rect",
"image": "color:d2f17f",
"x": "7p+4",
"y": "22p",
"width": "84",
"height": "5"
},
{
"name": "headline",
"kind": "text",
"text": "{{headline}}",
"x": "7p+4",
"y": "28p+4",
"width": "59p",
"height": "34p",
"font": "Liberation-Serif",
"color": "f4f1e8",
"max_size": 68,
"min_size": 52
},
{
"name": "supporting",
"kind": "text",
"text": "{{supporting}}",
"x": "7p+4",
"y": "62p",
"width": "58p",
"height": "12p",
"font": "Open-Sans",
"color": "b3bec0",
"max_size": 25,
"min_size": 20
},
{
"name": "status-panel",
"kind": "rect",
"image": "color:1d3b34",
"x": "68p+4",
"y": "22p",
"width": "300",
"height": "352"
},
{
"name": "status-mark-background",
"kind": "rect",
"image": "color:d2f17f",
"x": "76p+4",
"y": "30p",
"width": "108",
"height": "107"
},
{
"name": "status-mark",
"kind": "text",
"text": "\u2713",
"x": "78p+4",
"y": "30p",
"font": "Open-Sans-bold-72",
"color": "1d3b34"
},
{
"name": "status-track-one",
"kind": "rect",
"image": "color:2f4a42",
"x": "71p+4",
"y": "58p",
"width": "204",
"height": "5"
},
{
"name": "status-progress-one",
"kind": "rect",
"image": "color:d2f17f",
"x": "71p+4",
"y": "58p",
"width": "120",
"height": "5"
},
{
"name": "status-indicator-one",
"kind": "rect",
"image": "color:d2f17f",
"x": "90p",
"y": "58p",
"width": "12",
"height": "12"
},
{
"name": "status-track-two",
"kind": "rect",
"image": "color:2f4a42",
"x": "71p+4",
"y": "64p",
"width": "204",
"height": "5"
},
{
"name": "status-progress-two",
"kind": "rect",
"image": "color:d2f17f",
"x": "71p+4",
"y": "64p",
"width": "168",
"height": "5"
},
{
"name": "status-indicator-two",
"kind": "rect",
"image": "color:d2f17f",
"x": "90p",
"y": "64p",
"width": "12",
"height": "12"
},
{
"name": "status-track-three",
"kind": "rect",
"image": "color:2f4a42",
"x": "71p+4",
"y": "70p",
"width": "204",
"height": "5"
},
{
"name": "status-progress-three",
"kind": "rect",
"image": "color:d2f17f",
"x": "71p+4",
"y": "70p",
"width": "96",
"height": "5"
},
{
"name": "status-indicator-three",
"kind": "rect",
"image": "color:d2f17f",
"x": "90p",
"y": "70p",
"width": "12",
"height": "12"
},
{
"name": "mark",
"kind": "text",
"text": "{{mark}}",
"x": "7p+4",
"y": "80p",
"font": "Open-Sans-bold-20",
"color": "d2f17f"
}
],
"values": {
"eyebrow": "IMAGOR 1.9.8",
"headline": "Every filter now reports whether it ran",
"supporting": "A filter that did nothing used to look like one that worked.",
"mark": "imagor"
}
}
```

## avatar-card

![avatar-card](avatar-card.png)

*1200×630* — A landscape profile card with a circular portrait and a left-aligned identity block.

**Template** — copy this to iterate:

```json
{
"name": "Mei Lam landscape profile card",
"note": "A landscape profile card with a circular portrait and a left-aligned identity block.",
"canvas": {
"width": 1200,
"height": 630,
"background": "color:173f34"
},
"slots": [
{
"name": "cream inset border",
"kind": "rect",
"image": "color:f4f1e8",
"x": "34",
"y": "34",
"width": "full-68",
"height": "full-68"
},
{
"name": "green card panel",
"kind": "rect",
"image": "color:173f34",
"x": "40",
"y": "40",
"width": "full-80",
"height": "full-80"
},
{
"name": "circular portrait",
"kind": "image",
"image": "https://images.unsplash.com/photo-1586297135537-94bc9ba060aa?auto=format&fit=crop&crop=faces&facepad=30&w=400&h=400&q=80",
"filters": "smart/filters:round_corner(160,160,173f34)",
"x": "135",
"y": "155",
"width": "320",
"height": "320"
},
{
"name": "gold divider",
"kind": "rect",
"image": "color:d8c79b",
"x": "500",
"y": "160",
"width": "2",
"height": "310"
},
{
"name": "name",
"kind": "text",
"text": "{{name}}",
"x": "555",
"y": "207",
"font": "Open-Sans-bold",
"size": 64,
"color": "f4f1e8"
},
{
"name": "role",
"kind": "text",
"text": "{{role}}",
"x": "558",
"y": "302",
"font": "Open-Sans",
"size": 34,
"color": "cbd4ce"
},
{
"name": "handle",
"kind": "text",
"text": "{{handle}}",
"x": "558",
"y": "365",
"font": "Open-Sans",
"size": 30,
"color": "d8c79b"
},
{
"name": "identity underline",
"kind": "rect",
"image": "color:d8c79b",
"x": "558",
"y": "430",
"width": "500",
"height": "2"
}
],
"values": {
"name": "Mei Lam",
"role": "Platform engineering",
"handle": "@meilam"
}
}
```

## circles

![circles](circles.png)

*1200×630* — A row of contributors as circular portraits with names and roles beneath. The portraits should be images on a plain or transparent ground - a source with its own background shows that background as a square block around the circle, which is the source's shape and not a fault in the corner.

**Template** — copy this to iterate:

```json
{
"name": "Contributor circles",
"note": "A row of contributors as circular portraits with names and roles beneath. The portraits should be images on a plain or transparent ground - a source with its own background shows that background as a square block around the circle, which is the source's shape and not a fault in the corner.",
"canvas": {
"width": 1200,
"height": 630,
"background": "color:f6f4ee"
},
"slots": [
{
"name": "brand-accent",
"kind": "rect",
"image": "color:c36c4a",
"x": "6p",
"y": "10p+1",
"width": "2p+2",
"height": "0p+3"
},
{
"name": "brand",
"kind": "text",
"text": "{{brand}}",
"x": "9p+4",
"y": "8p+3",
"font": "Open-Sans-bold-18",
"color": "242a27"
},
{
"name": "headline",
"kind": "text",
"text": "{{headline}}",
"x": "6p",
"y": "16p+2",
"width": "full-144",
"height": "13p+6",
"font": "Liberation-Serif",
"color": "242a27",
"max_size": 64,
"min_size": 54
},
{
"name": "portrait-1",
"kind": "image",
"image": "{{person1_image}}",
"x": "7p+8",
"y": "37p+5",
"width": "14p+8",
"height": "27p+6",
"filters": "smart/filters:round_corner(88,88,f6f4ee)"
},
{
"name": "portrait-2",
"kind": "image",
"image": "{{person2_image}}",
"x": "31p",
"y": "37p+5",
"width": "14p+8",
"height": "27p+6",
"filters": "smart/filters:round_corner(88,88,f6f4ee)"
},
{
"name": "portrait-3",
"kind": "image",
"image": "{{person3_image}}",
"x": "54p+4",
"y": "37p+5",
"width": "14p+8",
"height": "27p+6",
"filters": "smart/filters:round_corner(88,88,f6f4ee)"
},
{
"name": "portrait-4",
"kind": "image",
"image": "{{person4_image}}",
"x": "77p+8",
"y": "37p+5",
"width": "14p+8",
"height": "27p+6",
"filters": "smart/filters:round_corner(88,88,f6f4ee)"
},
{
"name": "person-1-labels",
"kind": "image",
"x": "5p+10",
"y": "68p+3",
"width": "18p+4",
"height": "13p+2",
"slots": [
{
"name": "name",
"kind": "text",
"text": "{{person1_name}}",
"x": "center",
"y": "0p",
"width": "full",
"height": "40p",
"font": "Open-Sans-bold",
"color": "242a27",
"max_size": 24,
"min_size": 20
},
{
"name": "role",
"kind": "text",
"text": "{{person1_role}}",
"x": "center",
"y": "46p",
"width": "full",
"height": "32p",
"font": "Open-Sans",
"color": "6f7771",
"max_size": 16,
"min_size": 13
}
]
},
{
"name": "person-2-labels",
"kind": "image",
"x": "29p+2",
"y": "68p+3",
"width": "18p+4",
"height": "13p+2",
"slots": [
{
"name": "name",
"kind": "text",
"text": "{{person2_name}}",
"x": "center",
"y": "0p",
"width": "full",
"height": "40p",
"font": "Open-Sans-bold",
"color": "242a27",
"max_size": 24,
"min_size": 20
},
{
"name": "role",
"kind": "text",
"text": "{{person2_role}}",
"x": "center",
"y": "46p",
"width": "full",
"height": "32p",
"font": "Open-Sans",
"color": "6f7771",
"max_size": 16,
"min_size": 13
}
]
},
{
"name": "person-3-labels",
"kind": "image",
"x": "52p+6",
"y": "68p+3",
"width": "18p+4",
"height": "13p+2",
"slots": [
{
"name": "name",
"kind": "text",
"text": "{{person3_name}}",
"x": "center",
"y": "0p",
"width": "full",
"height": "40p",
"font": "Open-Sans-bold",
"color": "242a27",
"max_size": 24,
"min_size": 20
},
{
"name": "role",
"kind": "text",
"text": "{{person3_role}}",
"x": "center",
"y": "46p",
"width": "full",
"height": "32p",
"font": "Open-Sans",
"color": "6f7771",
"max_size": 16,
"min_size": 13
}
]
},
{
"name": "person-4-labels",
"kind": "image",
"x": "75p+10",
"y": "68p+3",
"width": "18p+4",
"height": "13p+2",
"slots": [
{
"name": "name",
"kind": "text",
"text": "{{person4_name}}",
"x": "center",
"y": "0p",
"width": "full",
"height": "40p",
"font": "Open-Sans-bold",
"color": "242a27",
"max_size": 24,
"min_size": 20
},
{
"name": "role",
"kind": "text",
"text": "{{person4_role}}",
"x": "center",
"y": "46p",
"width": "full",
"height": "32p",
"font": "Open-Sans",
"color": "6f7771",
"max_size": 16,
"min_size": 13
}
]
},
{
"name": "footer-rule",
"kind": "rect",
"image": "color:d9d6cd",
"x": "6p",
"y": "88p+2",
"width": "full-144",
"height": "0p+1"
},
{
"name": "footer",
"kind": "image",
"x": "87p+6",
"y": "90p+6",
"width": "6p+6",
"height": "4p+5",
"slots": [
{
"name": "footer-text",
"kind": "text",
"text": "{{footer}}",
"x": "center",
"y": "0p",
"width": "full",
"height": "86p",
"font": "Open-Sans-bold",
"color": "777d77",
"max_size": 15,
"min_size": 13
}
]
}
],
"values": {
"brand": "SABLON",
"headline": "The people behind it",
"person1_name": "Mei Lam",
"person1_role": "Platform engineering",
"person1_image": "https://images.unsplash.com/photo-1654765437547-6b572f52ee1a?auto=format&fit=crop&w=400&h=400&q=80",
"person2_name": "Jon Ho",
"person2_role": "Image pipeline",
"person2_image": "https://images.unsplash.com/photo-1595152772835-219674b2a8a6?auto=format&fit=crop&w=400&h=400&q=80",
"person3_name": "Ada Cheng",
"person3_role": "Developer experience",
"person3_image": "https://images.unsplash.com/photo-1764103372936-a2c6aceb4233?auto=format&fit=crop&w=400&h=400&q=80",
"person4_name": "Sam Yuen",
"person4_role": "Reliability",
"person4_image": "https://images.unsplash.com/photo-1633625576932-348e73c45e82?auto=format&fit=crop&w=400&h=400&q=80",
"footer": "imagor"
}
}
```

## comparison

![comparison](comparison.png)

*1200×630* — Two matched columns of points, for a before and after.

**Template** — copy this to iterate:

```json
{
"name": "Comparison card",
"note": "Two matched columns of points, for a before and after.",
"canvas": [
1200,
630
],
"background": "color:f7f5ef",
"slots": [
{
"name": "brand",
"kind": "text",
"text": "{{brand}}",
"x": "5p+4",
"y": "6p+6",
"font": "Open-Sans-bold-18",
"color": "213d36"
},
{
"name": "top_rule",
"kind": "rect",
"image": "color:c58e43",
"x": "5p+4",
"y": "13p",
"width": "3p+8",
"height": "0p+3"
},
{
"name": "headline",
"kind": "text",
"text": "{{headline}}",
"x": "5p+4",
"y": "16p+4",
"width": "full-128",
"height": "11p+1",
"font": "Open-Sans-bold",
"max_size": 45,
"min_size": 32,
"color": "182a27",
"align": "low"
},
{
"name": "left_panel",
"kind": "rect",
"image": "color:efede6",
"x": "5p+4",
"y": "34p+4",
"width": "43p+8",
"height": "53p"
},
{
"name": "right_panel",
"kind": "rect",
"image": "color:1d443c",
"x": "51p",
"y": "34p+4",
"width": "43p+8",
"height": "53p"
},
{
"name": "left_accent",
"kind": "rect",
"image": "color:c58e43",
"x": "5p+4",
"y": "34p+4",
"width": "0p+6",
"height": "53p"
},
{
"name": "right_accent",
"kind": "rect",
"image": "color:79c7a6",
"x": "51p",
"y": "34p+4",
"width": "0p+6",
"height": "53p"
},
{
"name": "left_label",
"kind": "text",
"text": "{{left_label}}",
"x": "7p+8",
"y": "38p+6",
"font": "Open-Sans-bold-24",
"color": "756f62"
},
{
"name": "right_label",
"kind": "text",
"text": "{{right_label}}",
"x": "53p+4",
"y": "38p+6",
"font": "Open-Sans-bold-24",
"color": "a5e4c6"
},
{
"name": "left_label_rule",
"kind": "rect",
"image": "color:d1cdc2",
"x": "7p+8",
"y": "44p+6",
"width": "39p",
"height": "0p+1"
},
{
"name": "right_label_rule",
"kind": "rect",
"image": "color:45685f",
"x": "53p+4",
"y": "44p+6",
"width": "39p",
"height": "0p+1"
},
{
"name": "left_bullet_1",
"kind": "rect",
"image": "color:c58e43",
"x": "7p+10",
"y": "50p+2",
"width": "0p+8",
"height": "1p+2"
},
{
"name": "left_point_1",
"kind": "text",
"text": "{{left_point_1}}",
"x": "9p+10",
"y": "49p+3",
"width": "36p+8",
"font": "Open-Sans-18",
"color": "3d403b"
},
{
"name": "right_bullet_1",
"kind": "rect",
"image": "color:79c7a6",
"x": "53p+6",
"y": "50p+2",
"width": "0p+8",
"height": "1p+2"
},
{
"name": "right_point_1",
"kind": "text",
"text": "{{right_point_1}}",
"x": "55p+6",
"y": "49p+3",
"width": "36p+8",
"font": "Open-Sans-18",
"color": "eef4ef"
},
{
"name": "left_bullet_2",
"kind": "rect",
"image": "color:c58e43",
"x": "7p+10",
"y": "62p+3",
"width": "0p+8",
"height": "1p+2"
},
{
"name": "left_point_2",
"kind": "text",
"text": "{{left_point_2}}",
"x": "9p+10",
"y": "61p+5",
"width": "36p+8",
"font": "Open-Sans-18",
"color": "3d403b"
},
{
"name": "right_bullet_2",
"kind": "rect",
"image": "color:79c7a6",
"x": "53p+6",
"y": "62p+3",
"width": "0p+8",
"height": "1p+2"
},
{
"name": "right_point_2",
"kind": "text",
"text": "{{right_point_2}}",
"x": "55p+6",
"y": "61p+5",
"width": "36p+8",
"font": "Open-Sans-18",
"color": "eef4ef"
},
{
"name": "left_bullet_3",
"kind": "rect",
"image": "color:c58e43",
"x": "7p+10",
"y": "74p+5",
"width": "0p+8",
"height": "1p+2"
},
{
"name": "left_point_3",
"kind": "text",
"text": "{{left_point_3}}",
"x": "9p+10",
"y": "73p+6",
"width": "36p+8",
"font": "Open-Sans-18",
"color": "3d403b"
},
{
"name": "right_bullet_3",
"kind": "rect",
"image": "color:79c7a6",
"x": "53p+6",
"y": "74p+5",
"width": "0p+8",
"height": "1p+2"
},
{
"name": "right_point_3",
"kind": "text",
"text": "{{right_point_3}}",
"x": "55p+6",
"y": "73p+6",
"width": "36p+8",
"font": "Open-Sans-18",
"color": "eef4ef"
},
{
"name": "footer",
"kind": "text",
"text": "{{footer}}",
"x": "5p+4",
"y": "91p+4",
"font": "Open-Sans-bold-14",
"color": "7b7e76"
}
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

## event

![event](event.png)

*1200×628* — Date block, title, venue. The open-house / meetup format.

**Template** — copy this to iterate:

```json
{
"name": "event",
"note": "Date block, title, venue. The open-house / meetup format.",
"canvas": [
1200,
628
],
"background": "color:f3f0e8",
"slots": [
{
"name": "date-card",
"kind": "rect",
"image": "color:174c43",
"x": "5p+4",
"y": "14p",
"width": "21p",
"height": "71p+6"
},
{
"name": "date-accent",
"kind": "rect",
"image": "color:e6a26b",
"x": "8p",
"y": "29p+2",
"width": "4p+4",
"height": "0p+4"
},
{
"name": "day",
"kind": "text",
"text": "{{day}}",
"x": "8p",
"y": "35p",
"font": "Open-Sans-bold",
"size": 118,
"color": "fffaf1"
},
{
"name": "month",
"kind": "text",
"text": "{{month}}",
"x": "8p+4",
"y": "55p+3",
"font": "Open-Sans-bold",
"size": 34,
"color": "d9e5d9"
},
{
"name": "title",
"kind": "text",
"text": "{{title}}",
"x": "31p",
"y": "28p+6",
"width": "63p+8",
"height": "27p+4",
"font": "Liberation-Serif",
"color": "173c36",
"max_size": 76,
"min_size": 44
},
{
"name": "title-rule",
"kind": "rect",
"image": "color:d47b54",
"x": "31p",
"y": "58p+2",
"width": "6p",
"height": "0p+4"
},
{
"name": "venue",
"kind": "text",
"text": "{{venue}}",
"x": "31p",
"y": "63p+5",
"font": "Open-Sans",
"size": 30,
"color": "49645d"
}
],
"values": {
"day": "14",
"month": "NOV",
"title": "Deterministic image pipelines",
"venue": "Harbour office, 6pm"
}
}
```

## launch-card

![launch-card](launch-card.png)

*1200×630* — Editorial split layout for a seasonal accessories collection announcement.

**Template** — copy this to iterate:

```json
{
"name": "launch-card",
"note": "Editorial split layout for a seasonal accessories collection announcement.",
"canvas": [
1200,
630
],
"background": "color:f5f1e9",
"slots": [
{
"name": "hero-photo",
"kind": "image",
"image": "{{photo}}",
"x": "61p+6",
"y": "8p+5",
"width": "32p",
"height": "82p+3",
"fit": true
},
{
"name": "brand",
"kind": "text",
"text": "{{brand}}",
"x": "7p+11",
"y": "14p+3",
"font": "Open-Sans",
"size": 17,
"color": "302a25"
},
{
"name": "brand-rule",
"kind": "rect",
"image": "color:a7654b",
"x": "7p+11",
"y": "22p",
"width": "4p+4",
"height": "0p+2"
},
{
"name": "headline",
"kind": "text",
"text": "{{headline}}",
"x": "7p+8",
"y": "32p+5",
"width": "49p+8",
"height": "19p",
"font": "Liberation-Serif",
"color": "302a25",
"align": "low",
"max_size": 76,
"min_size": 54
},
{
"name": "supporting",
"kind": "text",
"text": "{{supporting}}",
"x": "7p+11",
"y": "55p+2",
"width": "46p+3",
"font": "Open-Sans",
"size": 19,
"color": "746b61"
},
{
"name": "cta-button",
"kind": "rect",
"image": "color:302a25",
"x": "7p+11",
"y": "68p+5",
"width": "20p",
"height": "8p+4"
},
{
"name": "cta-label",
"kind": "text",
"text": "{{cta}}",
"x": "9p+8",
"y": "71p+4",
"width": "17p",
"font": "Open-Sans",
"size": 15,
"color": "f5f1e9"
},
{
"name": "availability",
"kind": "text",
"text": "{{availability}}",
"x": "7p+11",
"y": "86p+2",
"width": "46p+3",
"font": "Open-Sans",
"size": 15,
"color": "746b61"
}
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

## news-announcement

![news-announcement](news-announcement.png)

*1200×630* — A restrained, editorial announcement for a product return, pairing a prominent headline with a supporting quote and dateline.

**Template** — copy this to iterate:

```json
{
"name": "Maison editorial news announcement",
"note": "A restrained, editorial announcement for a product return, pairing a prominent headline with a supporting quote and dateline.",
"canvas": {
"width": 1200,
"height": 630,
"background": "color:f6f4ed"
},
"slots": [
{
"name": "kicker",
"kind": "text",
"text": "{{kicker}}",
"x": "7p",
"y": "12p",
"font": "Open-Sans-bold",
"size": 18,
"color": "526653"
},
{
"name": "kicker-rule",
"kind": "rect",
"image": "color:526653",
"x": "7p",
"y": "18p",
"width": "4p",
"height": "2"
},
{
"name": "headline",
"kind": "text",
"text": "{{headline}}",
"x": "7p",
"y": "22p",
"width": "86p",
"height": "120",
"font": "Open-Sans-bold",
"color": "20251f",
"max_size": 66,
"min_size": 56
},
{
"name": "quote-rule",
"kind": "rect",
"image": "color:9a7958",
"x": "7p",
"y": "46p",
"width": "3",
"height": "29"
},
{
"name": "quote",
"kind": "text",
"text": "{{quote}}",
"x": "9p",
"y": "46p",
"width": "84p",
"height": "102",
"font": "Liberation-Serif",
"color": "343b34",
"max_size": 32,
"min_size": 27
},
{
"name": "source",
"kind": "text",
"text": "{{source}}",
"x": "9p",
"y": "70p",
"font": "Open-Sans",
"size": 16,
"color": "697064"
},
{
"name": "footer-rule",
"kind": "rect",
"image": "color:d7d6cc",
"x": "7p",
"y": "87p",
"width": "86p",
"height": "1"
}
],
"values": {
"kicker": "SABLON",
"headline": "Hand-woven leather, back in stock",
"quote": "The same bag, made the same way, for nineteen years.",
"source": "Hong Kong, October 2026"
}
}
```

## og-modern

![og-modern](og-modern.png)

*1200×630* — A dark, restrained Open Graph card with a textured navy ground, split headline, and fitted product grid.

**Template** — copy this to iterate:

```json
{
"name": "og-modern",
"note": "A dark, restrained Open Graph card with a textured navy ground, split headline, and fitted product grid.",
"canvas": {
"width": 1200,
"height": 630,
"background": "color:142e2b"
},
"background": "color:090b10",
"slots": [
{
"name": "brand",
"kind": "text",
"text": "{{brand}}",
"x": "7p",
"y": "8p",
"font": "Open-Sans-bold-18",
"color": "F3F1EC"
},
{
"name": "brand_rule",
"kind": "rect",
"image": "color:C5A575",
"x": "7p",
"y": "14p",
"width": "3p",
"height": "2"
},
{
"name": "headline_lead",
"kind": "text",
"text": "{{headline_lead}}",
"x": "7p",
"y": "30p",
"width": "50p",
"height": "15p",
"font": "Liberation-Serif-bold",
"max_size": 76,
"min_size": 50,
"color": "F3F1EC",
"align": "low"
},
{
"name": "headline_accent",
"kind": "text",
"text": "{{headline_accent}}",
"x": "7p",
"y": "45p",
"width": "50p",
"height": "15p",
"font": "Liberation-Serif-bold",
"max_size": 76,
"min_size": 50,
"color": "C5A575",
"align": "low"
},
{
"name": "supporting",
"kind": "text",
"text": "{{supporting}}",
"x": "7p",
"y": "69p",
"width": "48p",
"height": "7p",
"font": "Open-Sans",
"max_size": 24,
"min_size": 18,
"color": "D2D0C8",
"align": "low"
},
{
"name": "domain",
"kind": "text",
"text": "{{domain}}",
"x": "7p",
"y": "87p",
"font": "Open-Sans-16",
"color": "A5A6AA"
},
{
"name": "tile1",
"kind": "rect",
"image": "color:1d3b34",
"x": "61p",
"y": "22p",
"width": "10p",
"height": "24p",
"slots": [
{
"name": "tile1_photo",
"kind": "image",
"image": "{{photo1}}",
"x": "8p",
"y": "7p",
"width": "84p",
"height": "76p",
"fit": true
},
{
"name": "tile1_label",
"kind": "text",
"text": "01",
"x": "center",
"y": "89p",
"width": "full",
"height": "14p",
"font": "Open-Sans-bold-12",
"color": "C5A575",
"align": "center"
}
]
},
{
"name": "tile2",
"kind": "rect",
"image": "color:1d3b34",
"x": "72p",
"y": "22p",
"width": "10p",
"height": "24p",
"slots": [
{
"name": "tile2_photo",
"kind": "image",
"image": "{{photo2}}",
"x": "8p",
"y": "7p",
"width": "84p",
"height": "76p",
"fit": true
},
{
"name": "tile2_label",
"kind": "text",
"text": "02",
"x": "center",
"y": "89p",
"width": "full",
"height": "14p",
"font": "Open-Sans-bold-12",
"color": "C5A575",
"align": "center"
}
]
},
{
"name": "tile3",
"kind": "rect",
"image": "color:1d3b34",
"x": "83p",
"y": "22p",
"width": "10p",
"height": "24p",
"slots": [
{
"name": "tile3_photo",
"kind": "image",
"image": "{{photo3}}",
"x": "8p",
"y": "7p",
"width": "84p",
"height": "76p",
"fit": true
},
{
"name": "tile3_label",
"kind": "text",
"text": "03",
"x": "center",
"y": "89p",
"width": "full",
"height": "14p",
"font": "Open-Sans-bold-12",
"color": "C5A575",
"align": "center"
}
]
},
{
"name": "tile4",
"kind": "rect",
"image": "color:1d3b34",
"x": "61p",
"y": "50p",
"width": "10p",
"height": "24p",
"slots": [
{
"name": "tile4_photo",
"kind": "image",
"image": "{{photo4}}",
"x": "8p",
"y": "7p",
"width": "84p",
"height": "76p",
"fit": true
},
{
"name": "tile4_label",
"kind": "text",
"text": "04",
"x": "center",
"y": "89p",
"width": "full",
"height": "14p",
"font": "Open-Sans-bold-12",
"color": "C5A575",
"align": "center"
}
]
},
{
"name": "tile5",
"kind": "rect",
"image": "color:1d3b34",
"x": "72p",
"y": "50p",
"width": "10p",
"height": "24p",
"slots": [
{
"name": "tile5_photo",
"kind": "image",
"image": "{{photo5}}",
"x": "8p",
"y": "7p",
"width": "84p",
"height": "76p",
"fit": true
},
{
"name": "tile5_label",
"kind": "text",
"text": "05",
"x": "center",
"y": "89p",
"width": "full",
"height": "14p",
"font": "Open-Sans-bold-12",
"color": "C5A575",
"align": "center"
}
]
},
{
"name": "tile6",
"kind": "rect",
"image": "color:1d3b34",
"x": "83p",
"y": "50p",
"width": "10p",
"height": "24p",
"slots": [
{
"name": "tile6_photo",
"kind": "image",
"image": "{{photo6}}",
"x": "8p",
"y": "7p",
"width": "84p",
"height": "76p",
"fit": true
},
{
"name": "tile6_label",
"kind": "text",
"text": "06",
"x": "center",
"y": "89p",
"width": "full",
"height": "14p",
"font": "Open-Sans-bold-12",
"color": "C5A575",
"align": "center"
}
]
}
],
"values": {
"brand": "SABLON",
"headline_lead": "Built for the",
"headline_accent": "long run",
"supporting": "Autumn 2026, available now",
"domain": "sablon.example",
"photo1": "https://images.unsplash.com/photo-1691480150204-66dd1eb77391?ixid=M3wxMTAwMjk5fDB8MXxzZWFyY2h8MXx8bGVhdGhlciUyMGJhZ3xlbnwwfDJ8fHwxNzkxODIyNzYxfDA&ixlib=rb-4.1.0?auto=format&fit=crop&w=700&q=80",
"photo2": "https://images.unsplash.com/photo-1719488134786-a4c273733931?ixid=M3wxMTAwMjk5fDB8MXxzZWFyY2h8Mnx8cGFja2FnaW5nJTIwYm94fGVufDB8Mnx8fDE3OTE4MjI3NjN8MA&ixlib=rb-4.1.0?auto=format&fit=crop&w=700&q=80",
"photo3": "https://images.unsplash.com/photo-1691480250099-a63081ecfcb8?ixid=M3wxMTAwMjk5fDB8MXxzZWFyY2h8Mnx8bGVhdGhlciUyMGJhZ3xlbnwwfDJ8fHwxNzkxODIyNzYxfDA&ixlib=rb-4.1.0?auto=format&fit=crop&w=700&q=80",
"photo4": "https://images.unsplash.com/photo-1644085159285-5fd924740cb3?ixid=M3wxMTAwMjk5fDB8MXxzZWFyY2h8NHx8Z2xhc3MlMjBib3R0bGV8ZW58MHwxfHx8MTc5MTgyMjc2NHww&ixlib=rb-4.1.0?auto=format&fit=crop&w=700&q=80",
"photo5": "https://images.unsplash.com/photo-1760624294582-5341f33f9fa4?ixid=M3wxMTAwMjk5fDB8MXxzZWFyY2h8Mnx8dG90ZSUyMGJhZ3xlbnwwfDJ8fHwxNzkxODIyNzYyfDA&ixlib=rb-4.1.0?auto=format&fit=crop&w=700&q=80",
"photo6": "https://images.unsplash.com/photo-1691480288782-142b953cf664"
}
}
```

## og-standard

![og-standard](og-standard.png)

*1200×630* — A text-first 1200x630 Open Graph layout with a small brand mark, prominent title, supporting description, and quiet domain footer.

**Template** — copy this to iterate:

```json
{
"name": "og-standard",
"note": "A text-first 1200x630 Open Graph layout with a small brand mark, prominent title, supporting description, and quiet domain footer.",
"canvas": [
1200,
630
],
"background": "color:f4f0e7",
"slots": [
{
"name": "brand",
"kind": "text",
"text": "{{brand}}",
"x": "7p+8",
"y": "11p+3",
"font": "Open-Sans-bold-24",
"color": "263a32"
},
{
"name": "brand_rule",
"kind": "rect",
"image": "color:c58b4b",
"x": "7p+8",
"y": "18p+3",
"width": "4p+6",
"height": "0p+3"
},
{
"name": "title",
"kind": "text",
"text": "{{title}}",
"x": "7p+8",
"y": "30p+3",
"width": "80p+10",
"height": "26p+2",
"font": "Liberation-Serif",
"color": "263a32",
"max_size": 82,
"min_size": 60
},
{
"name": "description",
"kind": "text",
"text": "{{description}}",
"x": "7p+8",
"y": "59p",
"width": "81p+8",
"height": "13p",
"font": "Open-Sans",
"color": "4e5b52",
"max_size": 32,
"min_size": 30
},
{
"name": "footer_rule",
"kind": "rect",
"image": "color:d8d0c1",
"x": "7p+8",
"y": "84p+2",
"width": "84p+8",
"height": "0p+1"
},
{
"name": "domain",
"kind": "text",
"text": "{{domain}}",
"x": "7p+8",
"y": "87p+4",
"font": "Open-Sans-20",
"color": "58645b"
}
],
"values": {
"brand": "SABLON",
"title": "The Autumn Edit is here",
"description": "Twelve pieces in vegetable-tanned leather, made in small batches",
"domain": "sablon.example"
}
}
```

## product-grid

![product-grid](product-grid.png)

*1800×942* — Lifestyle-led editorial product grid with a tall campaign panel and product captions nested in each image canvas.

**Template** — copy this to iterate:

```json
{
"name": "Product grid banner",
"note": "Lifestyle-led editorial product grid with a tall campaign panel and product captions nested in each image canvas.",
"canvas": {
"width": 1800,
"height": 942,
"background": "color:ffffff"
},
"slots": [
{
"name": "lifestyle_panel",
"kind": "image",
"image": "color:ffffff",
"x": "3p",
"y": "6p",
"width": "33p",
"height": "88p",
"slots": [
{
"name": "lifestyle_photo",
"kind": "image",
"image": "{{src_life}}",
"x": "0",
"y": "0",
"width": "full",
"height": "87p"
},
{
"name": "wordmark",
"kind": "text",
"text": "{{wordmark}}",
"x": "center",
"y": "90p",
"font": "Liberation-Serif-32",
"color": "000000"
}
]
},
{
"name": "p1_image",
"kind": "image",
"image": "color:ffffff",
"x": "38p",
"y": "6p",
"width": "19p",
"height": "42p",
"slots": [
{
"name": "p1_photo",
"kind": "image",
"image": "{{src_p1}}",
"x": "0",
"y": "0",
"width": "full",
"height": "70p",
"fit": true
},
{
"name": "p1_name",
"kind": "text",
"text": "{{p1_name}}",
"x": "center",
"y": "76p",
"width": "full-14",
"height": "8p",
"font": "Open-Sans",
"color": "777777",
"max_size": 14,
"min_size": 12
},
{
"name": "p1_price",
"kind": "text",
"text": "{{p1_price}}",
"x": "center",
"y": "85p",
"width": "full-14",
"height": "8p",
"font": "Open-Sans-bold",
"color": "000000",
"max_size": 16,
"min_size": 14
}
]
},
{
"name": "p2_image",
"kind": "image",
"image": "color:ffffff",
"x": "58p",
"y": "6p",
"width": "19p",
"height": "42p",
"slots": [
{
"name": "p2_photo",
"kind": "image",
"image": "{{src_p2}}",
"x": "0",
"y": "0",
"width": "full",
"height": "70p",
"fit": true
},
{
"name": "p2_name",
"kind": "text",
"text": "{{p2_name}}",
"x": "center",
"y": "76p",
"width": "full-14",
"height": "8p",
"font": "Open-Sans",
"color": "777777",
"max_size": 14,
"min_size": 12
},
{
"name": "p2_price",
"kind": "text",
"text": "{{p2_price}}",
"x": "center",
"y": "85p",
"width": "full-14",
"height": "8p",
"font": "Open-Sans-bold",
"color": "000000",
"max_size": 16,
"min_size": 14
}
]
},
{
"name": "p3_image",
"kind": "image",
"image": "color:ffffff",
"x": "78p",
"y": "6p",
"width": "19p",
"height": "42p",
"slots": [
{
"name": "p3_photo",
"kind": "image",
"image": "{{src_p3}}",
"x": "0",
"y": "0",
"width": "full",
"height": "70p",
"fit": true
},
{
"name": "p3_name",
"kind": "text",
"text": "{{p3_name}}",
"x": "center",
"y": "76p",
"width": "full-14",
"height": "8p",
"font": "Open-Sans",
"color": "777777",
"max_size": 14,
"min_size": 12
},
{
"name": "p3_price",
"kind": "text",
"text": "{{p3_price}}",
"x": "center",
"y": "85p",
"width": "full-14",
"height": "8p",
"font": "Open-Sans-bold",
"color": "000000",
"max_size": 16,
"min_size": 14
}
]
},
{
"name": "p4_image",
"kind": "image",
"image": "color:ffffff",
"x": "48p",
"y": "52p",
"width": "19p",
"height": "42p",
"slots": [
{
"name": "p4_photo",
"kind": "image",
"image": "{{src_p4}}",
"x": "0",
"y": "0",
"width": "full",
"height": "70p",
"fit": true
},
{
"name": "p4_name",
"kind": "text",
"text": "{{p4_name}}",
"x": "center",
"y": "76p",
"width": "full-14",
"height": "8p",
"font": "Open-Sans",
"color": "777777",
"max_size": 14,
"min_size": 12
},
{
"name": "p4_price",
"kind": "text",
"text": "{{p4_price}}",
"x": "center",
"y": "85p",
"width": "full-14",
"height": "8p",
"font": "Open-Sans-bold",
"color": "000000",
"max_size": 16,
"min_size": 14
}
]
},
{
"name": "p5_image",
"kind": "image",
"image": "color:ffffff",
"x": "68p",
"y": "52p",
"width": "19p",
"height": "42p",
"slots": [
{
"name": "p5_photo",
"kind": "image",
"image": "{{src_p5}}",
"x": "0",
"y": "0",
"width": "full",
"height": "70p",
"fit": true
},
{
"name": "p5_name",
"kind": "text",
"text": "{{p5_name}}",
"x": "center",
"y": "76p",
"width": "full-14",
"height": "8p",
"font": "Open-Sans",
"color": "777777",
"max_size": 14,
"min_size": 12
},
{
"name": "p5_price",
"kind": "text",
"text": "{{p5_price}}",
"x": "center",
"y": "85p",
"width": "full-14",
"height": "8p",
"font": "Open-Sans-bold",
"color": "000000",
"max_size": 16,
"min_size": 14
}
]
}
],
"values": {
"src_life": "https://images.unsplash.com/photo-1760624294469-550753ec203a?ixid=M3wxMTAwMjk5fDB8MXxzZWFyY2h8M3x8dG90ZSUyMGJhZ3xlbnwwfDJ8fHwxNzkxODIyNzYyfDA&ixlib=rb-4.1.0?auto=format&fit=crop&w=900&q=80",
"src_p1": "https://images.unsplash.com/photo-1691480150204-66dd1eb77391?ixid=M3wxMTAwMjk5fDB8MXxzZWFyY2h8MXx8bGVhdGhlciUyMGJhZ3xlbnwwfDJ8fHwxNzkxODIyNzYxfDA&ixlib=rb-4.1.0?auto=format&fit=crop&w=700&q=80",
"src_p2": "https://images.unsplash.com/photo-1719488134786-a4c273733931?ixid=M3wxMTAwMjk5fDB8MXxzZWFyY2h8Mnx8cGFja2FnaW5nJTIwYm94fGVufDB8Mnx8fDE3OTE4MjI3NjN8MA&ixlib=rb-4.1.0?auto=format&fit=crop&w=700&q=80",
"src_p3": "https://images.unsplash.com/photo-1691480250099-a63081ecfcb8?ixid=M3wxMTAwMjk5fDB8MXxzZWFyY2h8Mnx8bGVhdGhlciUyMGJhZ3xlbnwwfDJ8fHwxNzkxODIyNzYxfDA&ixlib=rb-4.1.0?auto=format&fit=crop&w=700&q=80",
"src_p4": "https://images.unsplash.com/photo-1644085159285-5fd924740cb3?ixid=M3wxMTAwMjk5fDB8MXxzZWFyY2h8NHx8Z2xhc3MlMjBib3R0bGV8ZW58MHwxfHx8MTc5MTgyMjc2NHww&ixlib=rb-4.1.0?auto=format&fit=crop&w=700&q=80",
"src_p5": "https://images.unsplash.com/photo-1760624294582-5341f33f9fa4?ixid=M3wxMTAwMjk5fDB8MXxzZWFyY2h8Mnx8dG90ZSUyMGJhZ3xlbnwwfDJ8fHwxNzkxODIyNzYyfDA&ixlib=rb-4.1.0?auto=format&fit=crop&w=700&q=80",
"wordmark": "SABLON",
"p1_name": "Leather satchel bag",
"p1_price": "HKD$18500",
"p2_name": "Ribboned gift box",
"p2_price": "HKD$11540",
"p3_name": "Grained leather satchel",
"p3_price": "HKD$12875",
"p4_name": "Amber glass bottle",
"p4_price": "HKD$13625",
"p5_name": "Soft tote, three colourways",
"p5_price": "HKD$15170"
}
}
```

## product-og

![product-og](product-og.png)

*1200×630* — An editorial product share card with a featured product photograph and concise offer details.

**Template** — copy this to iterate:

```json
{
"name": "Product OG image",
"note": "An editorial product share card with a featured product photograph and concise offer details.",
"canvas": {
"width": 1200,
"height": 630
},
"background": "color:f7f5f0",
"slots": [
{
"name": "photo-mat",
"kind": "rect",
"image": "color:ffffff",
"x": "51p+8",
"y": "6p+4",
"width": "44p+4",
"height": "86p+4"
},
{
"name": "featured-image",
"kind": "image",
"image": "{{featured_image}}",
"x": "52p+8",
"y": "7p+6",
"width": "42p+4",
"height": "83p+6"
},
{
"name": "collection-accent",
"kind": "rect",
"image": "color:9b8057",
"x": "7p",
"y": "22p+3",
"width": "2p+10",
"height": "0p+3"
},
{
"name": "collection",
"kind": "text",
"text": "{{collection}}",
"x": "7p",
"y": "26p+1",
"font": "Open-Sans-bold",
"size": 19,
"color": "806d52"
},
{
"name": "title",
"kind": "text",
"text": "{{title}}",
"x": "7p",
"y": "34p+4",
"width": "37p+6",
"height": "26p",
"font": "Liberation-Serif",
"color": "25231f",
"max_size": 62,
"min_size": 48,
"align": "low"
},
{
"name": "price",
"kind": "text",
"text": "{{price}}",
"x": "7p",
"y": "66p+6",
"font": "Open-Sans-bold",
"size": 38,
"color": "25231f"
},
{
"name": "cta-accent",
"kind": "rect",
"image": "color:9b8057",
"x": "7p",
"y": "77p+5",
"width": "2p+4",
"height": "0p+2"
},
{
"name": "cta",
"kind": "text",
"text": "{{cta}}",
"x": "7p",
"y": "81p+2",
"font": "Open-Sans",
"size": 20,
"color": "6e665a"
}
],
"values": {
"title": "Leather satchel bag",
"price": "HKD$18500",
"collection": "Autumn 2026",
"cta": "Free delivery",
"featured_image": "https://images.unsplash.com/photo-1691480150204-66dd1eb77391?ixid=M3wxMTAwMjk5fDB8MXxzZWFyY2h8MXx8bGVhdGhlciUyMGJhZ3xlbnwwfDJ8fHwxNzkxODIyNzYxfDA&ixlib=rb-4.1.0?auto=format&fit=crop&w=700&q=80"
}
}
```

## quote-card

![quote-card](quote-card.png)

*1200×630* — A pull quote on deep green: the quotation in serif, attributed below a rule.

**Template** — copy this to iterate:

```json
{
"name": "Quote card",
"note": "A pull quote on deep green: the quotation in serif, attributed below a rule.",
"source": "A daily quote card with a flat, typographic treatment (quote, attribution, role)",
"canvas": {
"width": 1200,
"height": 630,
"background": "color:142E2B"
},
"slots": [
{
"name": "panel",
"kind": "rect",
"image": "color:F4F0E6",
"x": "4p",
"y": "6p+6",
"width": "full-96",
"height": "full-88"
},
{
"name": "accent",
"kind": "rect",
"image": "color:C57B4A",
"x": "4p",
"y": "6p+6",
"width": "1p",
"height": "full-88"
},
{
"name": "mark",
"kind": "text",
"text": "\u201c",
"x": "11p+4",
"y": "13p",
"font": "Liberation-Serif",
"color": "C57B4A",
"size": 96
},
{
"name": "quote",
"kind": "text",
"text": "{{quote}}",
"x": "11p+8",
"y": "26p+2",
"width": "75p",
"height": "34p+6",
"font": "Liberation-Serif",
"color": "173C37",
"max_size": 72,
"min_size": 52
},
{
"name": "rule",
"kind": "rect",
"image": "color:C57B4A",
"x": "11p+8",
"y": "63p+5",
"width": "5p+4",
"height": "0p+4"
},
{
"name": "attribution",
"kind": "text",
"text": "{{attribution}}",
"x": "11p+8",
"y": "67p+4",
"font": "Open-Sans",
"color": "244F47",
"size": 24
},
{
"name": "role",
"kind": "text",
"text": "{{role}}",
"x": "11p+8",
"y": "73p+4",
"font": "Open-Sans",
"color": "687A72",
"size": 18
}
],
"values": {
"quote": "The same bytes in,\nthe same bytes out.",
"attribution": "Mei Lam, Harbour",
"role": "Platform engineering, Harbour"
}
}
```

## testimonial

![testimonial](testimonial.png)

*1200×628* — B2B customer quote with an inset forest panel, warm paper margins and restrained brass accents.

**Template** — copy this to iterate:

```json
{
"name": "testimonial",
"note": "B2B customer quote with an inset forest panel, warm paper margins and restrained brass accents.",
"canvas": [
1200,
628
],
"background": "color:f2efe6",
"slots": [
{
"name": "panel",
"kind": "rect",
"image": "color:142c29",
"x": "5p",
"y": "8p-2",
"width": "90p",
"height": "84p+4"
},
{
"name": "eyebrow-rule",
"kind": "rect",
"image": "color:c6a76a",
"x": "10p+6",
"y": "16p+3",
"width": "34",
"height": "2"
},
{
"name": "eyebrow",
"kind": "text",
"text": "CUSTOMER STORY",
"x": "15p",
"y": "14p+7",
"font": "Open-Sans-bold-14",
"color": "c6a76a"
},
{
"name": "quote-mark",
"kind": "text",
"text": "\u201c",
"x": "10p+6",
"y": "21p+2",
"font": "Liberation-Serif-72",
"color": "c6a76a"
},
{
"name": "quote",
"kind": "text",
"text": "{{quote}}",
"x": "10p+6",
"y": "27p",
"width": "80p",
"height": "35p+3",
"font": "Liberation-Serif",
"color": "f7f4ec",
"max_size": 62,
"min_size": 42
},
{
"name": "rule",
"kind": "rect",
"image": "color:c6a76a",
"x": "10p+6",
"y": "58p+4",
"width": "48",
"height": "2"
},
{
"name": "name",
"kind": "text",
"text": "{{name}}",
"x": "10p+6",
"y": "62p+1",
"font": "Open-Sans-bold-25",
"color": "ffffff"
},
{
"name": "company",
"kind": "text",
"text": "{{company}}",
"x": "10p+6",
"y": "69p",
"font": "Open-Sans-18",
"color": "b8c4bd"
}
],
"values": {
"quote": "We replaced a queue of image workers with one URL.",
"name": "Mei Lam",
"company": "Platform engineering, Harbour"
}
}
```

## tip

![tip](tip.png)

*1200×628* — One actionable tip per card. The educational format.

**Template** — copy this to iterate:

```json
{
"name": "tip",
"note": "One actionable tip per card. The educational format.",
"canvas": [
1200,
628
],
"background": "color:102c2a",
"slots": [
{
"name": "number",
"kind": "text",
"text": "{{number}}",
"x": "8p+4",
"y": "14p",
"font": "Open-Sans-bold-26",
"color": "e5a15b"
},
{
"name": "top-rule",
"kind": "rect",
"image": "color:456a60",
"x": "12p+10",
"y": "18p+3",
"width": "78p+10",
"height": "0p+2"
},
{
"name": "quote-accent",
"kind": "rect",
"image": "color:e5a15b",
"x": "8p+4",
"y": "30p+4",
"width": "0p+5",
"height": "22p+6"
},
{
"name": "tip",
"kind": "text",
"text": "{{tip}}",
"x": "11p+2",
"y": "31p+5",
"width": "80p+6",
"height": "35p+4",
"font": "Liberation-Serif",
"color": "fff7e8",
"max_size": 70,
"min_size": 48
},
{
"name": "mark",
"kind": "text",
"text": "{{mark}}",
"x": "8p+4",
"y": "81p+1",
"font": "Open-Sans-bold-22",
"color": "e5a15b"
}
],
"values": {
"number": "01",
"tip": "A signature covers the whole path, not the source.",
"mark": "imagor"
}
}
```
