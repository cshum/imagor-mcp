# The canvas that isn't there

*2 May to 13 October 2026. An issue filed in May, six days of building in October, and one bug that took both to close.*

What I want is to hand an agent a blank canvas and a brief, and get back a layout I would
publish.

imagor is an image-processing server I maintain. You give it a URL and it gives you an
image back. A URL like

```
/300x200/filters:quality(80)/photo.jpg
```

says *resize, and compress on the way out*. That is the smallest thing it does. A real one
looks more like this:

```
/1600x600/filters:image(580x600/photo.jpg,1020,0)
  :text(SPRING-SALE,78,150,serif-bold-25,e7b66c)
  :round_corner(24,24,173b32)/color:173b32
```

That one composes a photograph into a panel, draws text over it, masks the corners and
paints a ground. There are forty-five filters — compositing, type, watermarks, rounded
masks, colour profiles, PDF pages, region detection and redaction — and the whole language
is a path. Written out, it is a small program, and like most small programs it has a habit
of doing something reasonable-looking when you get it wrong.

What that adds up to is a claim I have not seen made in this space, and it is worth stating
before the failures, because the failures are in service of it.

The products this resembles are **editor-first**. A human designs a template in a
drag-and-drop canvas, and an API fills slots in that design. The canvas is the product.
This is **URL-first**: there is no canvas, no template document, no library. A layout *is*
a URL, and a URL is something an agent can write, check and rewrite. Given a brief with no
layout, no positions and no sizes, the agent designed three distinct banners across three
runs — hierarchy, colour, type pairing, invented copy — and zoomed into regions to read the
small text it had drawn.

**An agent can author a layout, not just fill one in.** That is the claim.

And the thing I keep coming back to is not a feature. It is a number: `200 OK`. What follows
is what it cost to be able to make that claim.

## 2 May — Someone else found it first

A user filed an issue against the server.

He had run the same animated GIF through a dozen filters and tabulated the results. `blur`, `sharpen`, `pixelate`, `saturation`, `hue` and `negative` returned the source unchanged — byte-identical to the passthrough. `brightness`, `contrast`, `grayscale`, `rotate`, `round_corner`, `rgb` and `format` all worked on the same input. He wrote the thing I had spent the project circling:

> *"The silent passthrough is hard to spot in client code: caller gets a 200 with valid GIF bytes and assumes it worked."*

I had been treating the silent success as something I kept discovering. Someone else had already described it precisely, with a table, and the table was better than mine.

I was offline when he filed it. It sat there for five months.

## September — Thirteen out of a hundred

Earlier this year an index scored the project on how well software agents could use it.
Thirteen out of a hundred, with the agent-specific part at zero. The finding was not that
the server lacked capability. It was that a machine had no typed way to write one of those
URLs, and no way to check one it had written — and their own note said the fix was easy,
because a transform grammar maps cleanly onto tool calls.

So I built the missing half: ten typed tools an agent can call to compose those URLs, a
document explaining the grammar, and — the part that turned out to matter — **the rendered
image back**, so the agent can look at what it built and correct it. There is also a
template layer: a whole layout described as data, rendered to a path, and read back into a
template you can edit rather than retype. The two directions do not agree exactly — a path
records the numbers a layout resolved to, not the forms it was written in — and I did not
find that out until I went looking.

What follows is those six days. One hundred and forty-eight commits, and their shape is the
shape of the story: seven on the first, one on the third, forty-nine on the fourth, sixty-nine
on the fifth, twenty-one on the last. Sixty-nine commits in a day is not a pace anybody
chooses; it is what the last day looks like when the checks start catching things.

## 8 October — The server says yes and is wrong

The server answers `200` with the image unchanged when a filter does not apply. Not an
error. Not a warning. A successful response containing the wrong picture.

I found this the way I found everything after it: by rendering the same composition twice,
once with the filter and once without, and subtracting one image from the other. When the
difference is zero, the filter did nothing, and the server told you it worked.

That single habit — render it twice, subtract — turned out to be the whole method. Almost
every bug in this project was invisible to reading and obvious to subtracting.

A template slot that named a font size had that size stripped out and replaced with a
hardcoded forty-eight. Every price label rendered at forty-eight points. The code looked
right. The path looked right. The server said `200`. It only became visible when I measured
the height of a price band against a reference image and got forty-two pixels where the
answer was seventeen.

And underneath all of it, in the server itself: `full-fit-in` was missing from the list of
words that get treated as commands rather than as the start of a source. A source whose
path happened to begin with those characters would be read as an instruction instead. Two
of twelve prefixes were unlisted. Nobody would notice, because the result is an image —
just not the one that was asked for.

## 12 October — The experiment that lied to me

The skill — the document an agent reads before composing anything — was getting long.
Every composition pays its length as a cost, so I wrote a shorter version and set out to
measure whether it was worse.

Six runs of the same task, same reference, same model, only the document differing. The
original averaged 9.42 with a range of 4.19. The shorter one averaged 9.06 with a range of
**0.38**.

That looked like a result. The shorter version was not just equal, it was *tighter* — it
raised the floor. I wrote it up that way, merged it, and moved on.

Then, later, I ran the shorter version three more times to check something else. The range
came back **8.25**. Twice the original's, including the worst single result in either
experiment.

The 0.38 was three runs. I had read a property out of a sample too small to have one. The
honest conclusion — the one now in the record — is that **six runs cannot separate these
two documents at all**. The shorter one ships because it is three to four times faster and
forty-five percent smaller. Not because it is better, which I do not know.

## 12 October — Everything that went wrong was mine

The first attempt at that experiment produced nothing usable, and every failure was in the
harness rather than the thing being tested. There were six of them, and I am listing them
because they are the part I would want to read.

The runs were being cut off at a ten-minute limit and my script was reading a URL out of a
truncated log — so it was scoring a **guess**. The task turned out to be unscorable,
because it named the wrong brand and pointed at a photograph that was not the one in the
reference; the model built exactly what it was asked for and I scored it against something
else. A stale copy of the server was still running from hours earlier, holding an old
environment, signing with a different key, so every URL it produced was rejected — and I
spent three rounds diagnosing the model before checking the process. My own prompt had
dropped a line that made the model iterate, so it composed once and stopped. My
URL-matching pattern stopped at a closing bracket, and every path contains one — so I
truncated every URL myself, fetched my own truncations, got 403 from all of them, and
reported that as the model failing.

Then I deleted the logs before anyone needed them.

None of those are interesting individually. What is interesting is that they are all the
same mistake: **a small sample or a partial read treated as a finding.** The pattern match
that read as behaviour. The three runs that read as a property. The single Dockerfile that
read as the whole system — I announced there were no fonts in the image, having stopped
reading one file short of the repository that actually installs them.

I would like to say I learned that once. Later in the same project I saw a card render navy
when it should have been green, decided the server was serving a cached copy, and went
looking into cache headers. Two lines I had already printed said the truth: the value I had
changed was an orphan, and the slot held a different URL directly. I had not read them.
Twice more in one evening I formed a hypothesis from a glance and started acting on it
before checking. It is not a lesson I have finished learning; it is a habit I now notice
faster, which is a different and lesser thing.

## 12 October — Teaching it to look

The part of this I am most pleased with started as an absence.

For most of the build, the verification tool returned **text**. It told the agent what the
server did with each filter, whether it applied or declined, what dimensions came back. And
the agent, reading that, composed blind. It iterated against a number.

When I gave it the rendered image back — scaled, with the scale stated — the behaviour
changed. It started checking its own work. And when I gave it a way to ask for one region
at full resolution, it worked out on its own that the scaled preview was for layout and the
region was for reading text. Nobody told it that. It had two tools and drew the distinction
itself.

The numbers are almost funny. A scaled preview costs about four hundred tokens. A region
costs fifty-two. The full render costs two thousand two hundred. So the cheapest way to
check a price label is not to look at everything — it is to crop the four hundred pixels
that contain it, at native resolution, where the text is sharper than the full render
anyway, because it is the same pixels with less around them.

That arithmetic is not a detail, it is the design. An agent that can afford to look at its
own output forty times while composing will look forty times. One that pays two thousand
tokens a glance looks once and hopes.

Then I asked it to design something from scratch. No layout, no positions, no sizes — a
shop, an offer, six photographs, and a blank canvas. It produced three distinct banners
across three runs. One dark with gold, one cream with copper, and one purely typographic
with no photographs at all — serif display type against a sans body, invented copy,
deliberate whitespace. It paired fonts. It made decisions. It rendered, looked, and revised
without being asked to.

I had expected to write a section on design principles afterwards. I did not, because the
design sense was not coming from the document. The document only had to supply the
mechanics and a way to see.

That is the whole thing working. Not an agent filling in a template somebody else designed,
but an agent given a blank canvas and a brief, producing work I would not have been
embarrassed to publish, and checking it as it went. Everything in this essay that reads
like a confession is the price of getting to that paragraph.

It is public and tagged. Ten tools, a skill served byte-identically from the binary, and
fourteen recipes in an example cookbook that CI renders at three canvas sizes on every
commit — so a recipe that stops working fails the build rather than sitting there teaching a
shape that no longer renders. It runs self-hosted, so there is no per-render fee and nothing
to sign up for, and it is built on a server that people were already deploying before any of
this existed.

None of that is the interesting part. It is the part that has to be true before the
interesting part counts.

## 13 October — The machine found what I did not

Later I stopped writing the examples by hand and started briefing a separate agent to
rebuild them from a written brief: the content fixed, the mechanics stated, the traps I had
already paid for written down so it would not rediscover them.

Two of those runs found gaps in my own tool that I had not.

The first hit a schema refusal — `unexpected additional properties` — and rather than
reporting a fault, it worked around the missing field and described the workaround as a
limitation in its closing summary. Reading that summary is how I learned my template format
could not express the technique the brief had asked for. The second found a second missing
field the same way.

Both were the same shape: a nested element carried fewer fields than a top-level one, and
nothing said which were absent. Every omission became a silent refusal, and a run that
wanted to use the technique simply could not, quietly, and said so only in passing.

The useful part is not that the tool had bugs. It is that **the thing I built to check its
own work is what surfaced them**, and that it reported them the way it did — as a
limitation it had worked around, not as a failure it had hit. A run that had stopped and
shouted would have been easier. One that adapts and mentions it in the last paragraph is
the one you have to actually read.

## 13 October — The typography that was never applied

Late in this, I noticed that every card in the example set looked like it used the same
font.

The set is built on a pairing: a serif for headlines, a sans for everything else. Twelve of
the fourteen cards name a serif. I had looked at these cards dozens of times and read the
pairing into them, because I knew it was supposed to be there.

So I rendered the same word — *illicit*, where a serif's shapes are unmistakable — in six
different families. Same size, same string, then a pixel diff. Every one came back
byte-identical. Not similar. Identical, same hash, same ink width.

The render host was a binary built for my laptop, and it did not carry any of the families
the recipes name. It had been substituting a system font, silently, for all of them. Weight
still applied, so the cards looked varied — bold headlines next to regular body text, which
is exactly what a working typographic system looks like. The serif headlines were not
serif. Nothing in the output said so, and nothing would have, ever.

The fix took a line: render against the image the recipes actually target, where the fonts
live. Every card changed. Judgements I had made about the composition of that set — which
headline was too heavy, which ground was too dark — had been made against the wrong
picture, and some of them were simply wrong.

That is the same failure as the filter that did nothing, one level up. Not a mistake in
what I asked for. A whole dimension of the work, quietly not applied, reported as success.

## 13 October — The filter that did nothing

When I came back to it in October, the fix turned out to be structural rather than local — not six filters to repair, but the reporting itself.

Every filter handler now says whether it acted. The dispatch loop distinguishes three outcomes instead of one:

```
filter             a handler ran and applied the filter
filter-declined    a handler ran and declined to
filter-unhandled   no handler matched the name
```

and the metadata endpoint reports it per filter, one entry per occurrence in URL order — because a filter can repeat, and its occurrences can disagree. `blur(5):blur()` reports one applied and one declined, which is a thing a single boolean per name could not have said.

That is the answer to the question at the top of this. The server that says yes now says what it did.

But a report is still something a caller has to read, and the tools I had built were reading it and returning success. A filter the server recognised and left the image unchanged came back in a note — *"1 recognised but left the image unchanged"* — which a caller checking for an error would never see. The tool had the information and handed it over as prose.

So that is now an error. A composition where any filter did nothing fails, names the filter, and says what imagor did instead. There is an opt-out for a filter that is deliberately conditional, and the walk descends into loaded paths, because a filter inside `image()` that did nothing is the same fault and harder to spot.

Two things came out of writing that, both worth more than the change. The first is that the note was counting declines rather than naming them — a test I wrote failed on it, and the right fix was the message, not the test, because identifying *which* filter did nothing is the entire point. The second is that the existing suite caught me: one test's fixture had `rotate()` declining, which it does by design since it needs an angle, so the new default failed it. That is the behaviour working. A test that breaks when you make the tool stricter was asserting the old silence.

## 13 October, after — What I keep coming back to

Every bug in this project had the same shape. The server said yes and was wrong.

A filter that did nothing. A font that was not installed, silently substituted. A crop in
the wrong units, returning the whole image. A preview of a different image from the one
described. A typographic system that was never applied at all. None of them errored. All of
them were indistinguishable from success, and the only reason any of them was caught is
that something was compared against something else.

And the failures were not all the server's. A check I wrote passed a card with a visible
collision, because it never substituted the recipe's values and then compared captions
against a cell's background rather than against the photograph inside it. It printed
`85 text slots checked, 0 overlaps`, and that number was true and meant nothing. I only
found out by making it fail on purpose. A test that has never failed is a test you have not
tested.

So the tool catches these by looking. That works, and it is the reason the claim at the top
of this piece is worth anything: an agent that can see its own output is an agent you can
hand a blank canvas to.

And a filter that did nothing is no longer something anyone has to notice. The server
reports what it did, the tool reads that report, and a composition where any filter
declined fails by default with the filter named. That is the central bug of this project,
closed — not by being careful, but by making the silence impossible.

What is left is narrower and more interesting. The tool can tell you that every filter ran.
It cannot tell you that the composition is the one you meant. That still needs something to
compare against — a reference image, a measurement, an eye — and where there is none, the
result is `200`, it looks plausible, and plausible is what every bug in this document looked
like.

That is a limit of the shape rather than of the work, and I do not think it is closable from
here. Verification is only as good as the thing you compare against. The tool can supply the
comparison and cannot supply the standard, and everything I have built this year has been
about moving the boundary between those two rather than pretending it is not there.

I have been building a server that says yes, and teaching an agent to notice when yes is not
an answer — which it now does, reliably, for every question it can be asked in the language
of the URL. What I have not built is a way to tell it what it should have seen.
