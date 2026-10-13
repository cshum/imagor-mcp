# Build log

What went wrong building this, in the order it went wrong.

Not a changelog — the commits carry that. This is the record of the mistakes, because they
turned out to be the interesting part: almost every one had the same shape, and none of
them looked like a mistake at the time.

## The shape

imagor answers `200` and is wrong. A filter that does nothing, a font that is not
installed, a crop in the wrong units — none of them error. They return an image, just not
the one that was asked for.

Everything below is a variation on that, including the mistakes that were mine rather than
the server's. **A wrong answer that looks right is the whole problem this tool exists to
address, and it turned out to be the whole problem in building it too.**

---

## I thought the ground had not changed

A card's background rendered navy when it should have been green. I had changed the value
that names the ground, so the change had not taken. The obvious explanation was caching:
the fixture had 404'd before it was deployed, so the server was serving a stale copy.

That theory is plausible, testable, and wrong. **Two lines I had already printed said so:**

```
values.texture -> texture-forest.svg     the value I changed
slot  texture  -> texture-navy.svg       a literal URL, which wins
```

The slot held the URL directly instead of a `{{texture}}` placeholder, so `values.texture`
was an orphan key nobody read. **The value looked live and was dead.**

**What it cost:** a detour into cache behaviour, and a fixture added for a problem that did
not exist. **What fixed it:** reading the two lines. **What I should have done first:**
diff the value against the slot, which takes one command.

## The typo that wasn't

A README said `FIXTURES`; the generator defines `FIXTURES`. I checked, saw a mismatch,
and set about fixing it — carefully, because the name was inside a shell heredoc where
quoting is easy to get wrong. I wrote a small script, built the search strings by
concatenation so they could not confuse each other, and ran it.

**It reported that the file was already correct.** It was. **I had misread my own grep
output** — the earlier command printed `FIXTURES` and I read it as `FIXTURES`.

**What it cost:** nothing, because the script read the real name from the generator instead
of trusting a literal. **What it would have cost:** a commit whose message described a typo
that never existed. **The lesson is not "check twice"** — it is that the verification step
is what caught it, not the care taken in the fix.

## The check that passed and proved nothing

The cookbook had a visible fault: a caption running into a tall bottle. So I wrote a check
that no caption overlaps a picture, and it passed.

It passed because it was comparing the wrong things, twice over.

**First**, the slot held `{{src_p1}}` and the check never substituted the recipe's values,
so no source resolved. With no source, the "ink box" fell back to the whole slot box — and
the check ran, found nothing, and reported success.

**Second**, even with values substituted it was comparing each caption against the cell's
own white background rectangle rather than against the photograph inside it. A cell in this
cookbook is a `color:ffffff` rect carrying a nested photo. **The rect is not ink.**

**Both faults were invisible.** The check printed `85 text slots checked, 0 overlaps` and
that number was true and meaningless.

**What it took to find:** making the check fail on purpose. **A test that has never failed
is a test you have not tested.**

## I measured the box and called it the picture

Before the ink comparison there was a simpler version: compare slot boxes. It reported
seven overlaps, all in a card whose render is correct.

The reasoning was that a fitted image centres in its box and leaves margins, so a caption
can overlap the box harmlessly. That is true — **and then I over-applied it.** To catch the
portrait case I added "a portrait *box* means a portrait *source*, so any overlap is a
fault."

**A box's shape says nothing about the source's.** A 700×700 photograph in a 342×396 box
letterboxes vertically and leaves the caption 54 pixels of margin. My rule flagged that
card ten times.

**What fixed it:** reading the source's aspect, which the URL carries when the width is set
and the height is not. **What it cost:** three attempts, because I kept trying to infer the
ink from something other than the source.

## Six fonts, one picture

A card's headline rendered identically in `Liberation-Serif`, `Liberation-Sans`,
`DejaVu-Serif`, `DejaVu-Sans` and `Open-Sans`. Byte-identical, same hash.

The render host was a binary built for macOS, which does not carry any of those families,
and substitutes an Apple face silently. **Every card in the cookbook had been rendering in
the same fallback font** — including the serif headlines the whole set is built around,
which were not serif.

**Nothing in the output said so.** The cards looked varied, because weight still applied.

**What fixed it:** rendering against the imagor image, where the fonts live. **What it
cost:** every visual judgement made before that point was made against the wrong picture.
**The rule that came out of it:** render with the image the recipes target, and say so in
the code, because the substitution is invisible.

## The fixture for a problem that was already solved

The green ground needed a fixture, so I added `texture-forest` — a gradient SVG, deployed,
verified.

Then the actual fix turned out to be `color:142e2b` on the canvas. **A background is a
colour, not an asset.** Every other card in the set already worked that way; this one had
been fetching an image to paint its own background, which is what put a network fetch
between the card and its colour in the first place.

The fixture was removed. **An unused fixture justified by a hypothetical future request is
the same dead weight as the orphaned value that started this.**

## What a machine found that I did not

Two gaps in the tool surfaced from a delegated run hitting schema refusals — not from
reading the code:

**`InnerSlot` had no `Fit` field.** A nested canvas could not fit an image, so the
technique the brief asked for could not be expressed. The run worked around it, documented
the workaround as a limitation, and carried on. **Only the log said so.**

**The image branch resolved `x` as a dimension where the text branch used a position**, so
a nested picture could not be centred — only placed by pixel count.

Both are the same shape: **a nested slot's field set was narrower than a top-level slot's,
and nothing said so.** Every field omitted became a silent refusal. Two of them shipped.

## The brief that carried a rule that was not true

A brief told the delegated run that a slot's `filters` field replaces its fit. **It does
not** — `fit` prepends `fit-in/` and adds `fill(none)`, then `filters` appends as its own
segment. The two combine.

I had written that rule from a probe that set one and not the other. **A trap in a brief
that is false is worse than no trap**, because the run trusts it — and the failure is a
wrong artefact, not an error.

**What fixed it:** reading the code that builds the path. **What it cost:** a corrected
brief and a rewritten rule, after the run had already been launched against the wrong one.

## And the small ones

**A generator that wrote but never deleted.** Removing a fixture from the table left its
`.svg` on disk and still served. The deployed set and the committed set disagreed, and
nothing said which was right.

**A hand-written copy of a generated table.** `references/sources.md` restated the
manifest and had already gone stale — it listed fixtures that had been removed. It is
generated now.

**A README that named a tool that does not exist** and a domain that is no longer used. The
example URL was the one line a reader would copy, and it 404'd.

**Three of the same fault, in three documents:** anything that restates a generated file
goes stale, and the fix is to generate it or delete it.

---

## The round trip that isn't

`parse_template` is meant to be the inverse of `render_template`: take a path and get back
something you can edit. It is documented as lossy in one direction — a path holds the numbers
a layout resolved to, not the forms it was written in, so `38p` comes back as `684`. What it
does not say, and what I did not know until I measured it, is that **a nested layout does not
come back at all.** Reading a path back flattened every nested slot into an unrelated
top-level one: a grid of six products, each with a photo, a name and a price inside it,
returned as six slots all called `image`.

Measured across the cookbook, `render → parse → render` was stable for one recipe out of five.
The paths that came out were longer than the ones that went in — 3,132 characters to 4,146 —
because each nested slot was re-expressed at the top level with absolute numbers.

I tried to fix it twice. Both attempts are reverted.

**The first** split the inner path by hand. It cut a URL source in half at its last slash —
`https://images.unsplash.com/photo-169…` came back as `photo-169…` — and read a dimension
segment of `176x176/smart` as a height. It also broke three existing tests.

**The second** did the right thing and handed the inner path to `imagorpath.Parse`, which
already knows every form a path takes. That fixed the URL and the `smart`, and `comparison`
became stable. But three recipes still moved, and the same three tests still failed.

**What the tests were telling me, and I did not read:** they assert that a slot's parts come
back *individually* — `img.Height == "full"`, `txt.Width == "600"` — which is the whole point
of the tool. A parser that returns a plausible-looking template is not enough; it has to return
the same template. I was checking that the path round-tripped and not that the fields did.

**The real bug, found on the way:** the emitter writes a text slot's box width as a *form* —
`full-144`, not `144`. The parser read it with `strconv.Atoi`, which failed silently and took
the align and the rest of the argument tail with it. One `if err == nil` around a value that
was never a number.

Two attempts, no fix, and the lesson is the one this document keeps arriving at. I was reading
the path — a plausible artifact that looked right — instead of the fields, which are the thing
the tool exists to produce.

---

## What the pattern is

Every entry above is a wrong answer that looked right. Not one of them errored. The
server, the test, the check, the brief and the README each reported success while being
wrong, and in every case **the only way to tell was to look at the thing itself** rather
than at what something said about it.

That is why the tool exists, and it is why this document exists next to it.
