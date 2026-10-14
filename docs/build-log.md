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

I tried to fix it three times.

**The first** split the inner path by hand. It cut a URL source in half at its last slash —
`https://images.unsplash.com/photo-169…` came back as `photo-169…` — and read a dimension
segment of `176x176/smart` as a height. It also broke three existing tests.

**The second** did the right thing and handed the inner path to `imagorpath.Parse`, which
knows every form a path takes. That fixed the URL and the `smart`, and `comparison` became
stable. But three recipes still moved, and the same three tests still failed. `Parse` is right
for imagor's own vocabulary and this is not imagor's vocabulary: the dimension segment here is
`full` and `full-144`, forms the template layer resolves *before* a path is written, and
imagor has never heard of them.

**The third started from the instrument instead of the parser.** A test that walks
render → parse and checks **every field by name**, so a failure says *which* one rather than
"the path changed". It named five failures in one run, and four were the same one: the
boundary between a slot's own filters and its source was being guessed. It is now found by
walking the path once at parenthesis depth zero — a source begins at a scheme, or failing
that at the last slash that is not inside a filter.

**The lesson is the one this document keeps arriving at, and I had to be shown it twice.** I
was comparing *paths* — a plausible artifact that looked right — instead of the *fields*,
which are the thing the tool exists to produce. The tests were asserting the fields all
along, and I was not reading them. Two attempts "made progress" and failed the same three
tests because I had no instrument that could tell me which field broke.

**Two other bugs fell out of the third attempt.** A text slot's box width is written as a
form, `full-144`, and the parser read it with `strconv.Atoi`, which failed silently and
dropped the align and the rest of the argument tail with it. And a filter that became a child
was also left in the slot's own `Filters`, so the path carried it twice.

**What still does not survive is what the doc comment always said.** A box width and height
are resolved to pixels before a path is written, so they come back as numbers. The tests now
say so where they assert it, rather than leaving it to be discovered.

---

---

## The instruction that contradicted the tool it named

`build_imagor_url` ended by telling the caller to "call `verify_imagor_url` on the result". The
result of building a URL is a URL. `verify_imagor_url` takes parameters, and says so in its own
description: the same typed parameters as `build_imagor_url`, not a URL.

So the tool that pointed at it was contradicting it. A real agent in a real session followed
the instruction rather than the warning — it passed the composed path as `image`, the server
built `unsafe/<that path>/<the rendered url>`, and imagor answered `404`. The agent then said it
could not confirm the preview, which was true.

**What it cost:** a round trip, and a wrong conclusion in a session that was otherwise working.
**What fixed it:** "pass the same parameters to `verify_imagor_url`". **Why it was not caught:**
every test asserted what the tools do. None asserted what they say.

## The document that arrived as one line

`get_imagor_skill` returned the skill. The tool worked, every byte was there, and it was
unreadable: a tool result's text content is the JSON of the structured output unless you set it
yourself, so 18,720 characters of markdown arrived with no line breaks at all and 351 escaped
`\n`. A caller in a real session stopped reading it and wrote a Python one-liner to recover the
lines.

**What fixed it:** setting the content explicitly — which `preview.go` had been doing for its
own result from the start, with the reason in a comment: *"Content is set explicitly rather than
left for the SDK, which would otherwise fill it with the JSON of the structured output alone."*
Three more tools got the same treatment. **What it cost:** a round trip in a real session, and
the discovery that one file had known the rule for months while three others quietly broke it.

## A tool that did nothing the tool beside it did not

`build_imagor_path` documented itself as the unsigned path, for a caller that signs with its own
secret. Its input schema was byte-identical to `build_imagor_url`'s. It called the same builder
and then blanked two fields. And its `path` field held the same value, because `build_imagor_url`
already returns the bare path — the prefix lives in `url`, not in `path`, in both modes.

It was a subset of what the neighbouring tool already returned, and it cost 4,225 characters of
schema that every session pays whether or not it composes anything. Fixed cost went from 24,710
characters to 20,161 when it was removed, and nothing was lost.

**How it was found:** by measuring the schemas rather than by reading the tools. It had been
there from the beginning, and nothing about it looked wrong.

## The feature I announced missing, twice

I told the author the server does not return an image, and that adding one was a real change
worth making. It does: a base64 JPEG in an MCP image content block, capped at 1 MiB, scaled to
768 on the long edge, with the MIME type sniffed rather than trusted, and a sentence reporting
the scale so a caller knows which coordinate system it is looking at.

I had probed `render_template`, got one text block, and generalised. `render_template` returns
only a URL **by design** — compiling is local and instant, fetching is `verify_imagor_url`'s job.
I checked the compile tool and reported on the fetch tool.

**What it cost:** a wrong claim made twice, and part of a session spent planning a feature that
already existed. **The shape:** one observation is not a property. This is the third place I have
made that mistake — in the code, in the test harness, and now in a description of the tool.

## The measurement nobody made

Configuration said that turning the cache on made repeated looks at a composition much cheaper,
with numbers: 386 ms for the first preview and 25 ms for a repeat, on a 1600×600 composition.

Nothing in this project had measured that. And imagor's own page says the cache key is **the
image path only** — filter parameters are not part of it — so the saving is not re-decoding a
source, which applies to any request that reuses it at any size, not to repeating an identical
composition. The sentence described something the cache does not do.

**What fixed it:** deleting the section and linking imagor's page, which also documents two
variables our copy had left out. **The numbers were not replaced with other numbers**, because
the four-cell measurement that would have said whether they described the right thing needed
Docker, and Docker was not running.

## The config that was already correct

A session began failing with `connection refused` against `192.168.0.124:8210`. That address was
one I had configured myself while testing, so the report was a stale config, and I corrected it
and said so.

It was not the config. Both files had already been fixed and grep found the address in neither.
What was running was the process: a test binary launched hours earlier, holding the old address
in its environment, being talked to by a client that does not re-read its config for a server
that is already running.

**What it cost:** a confident wrong diagnosis. **What fixed it:** killing the process I had
started. **The lesson:** editing a config does not reach a process that is already running. A test
binary left running is a service holding an old environment, and a client can still be talking to
it.

## And more of the small ones

**A check that read an error message as a result.** Fourteen cookbook recipes timed through the
CLI, all reported passing. Every "path" was 213 characters, because 213 characters was the CLI
complaining that no mode was set, and the check was for length rather than for shape. The second
attempt reported fourteen failures, because the CLI genuinely needs `IMAGOR_BASE_URL` — fitting
text means rendering a probe. The third attempt was the first to say anything true.

**A download instruction that did not work.** Fetch the binary with `curl`, `chmod +x imagor-mcp`,
then point a client at `"command": "imagor-mcp"`. `chmod` marks a file executable where it sits;
it does not put it on `PATH`. The next two steps fail with `spawn ENOENT`, which a client reports
as a server that would not start. A step that looks complete and is not is worse than no step,
because the reader stops looking.

**A heading over the wrong sentence.** `docs/tools.md` had a `## The cookbook` heading above a
line about the tool list, and stated the same fact twice fifty lines apart in two different
words.

## What the pattern is

Every entry above is a wrong answer that looked right. Not one of them errored. The
server, the test, the check, the brief and the README each reported success while being
wrong, and in every case **the only way to tell was to look at the thing itself** rather
than at what something said about it.

The later entries move the fault from the tool to the words about it: a description that
contradicted the tool it named, a document delivered as one line of JSON, a heading over the
wrong sentence, a measurement nobody took, a config blamed for a process. They fail the same
way. Nothing checks a claim, so a wrong one looks the same as a right one. The fix is the rest
of this document's: compare the claim against the thing.

That is why the tool exists, and it is why this document exists next to it.
