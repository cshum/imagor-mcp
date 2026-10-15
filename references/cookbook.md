# Cookbook

Ready-made layouts. **Read this to choose one, then fetch it by name.**
A recipe is the channel between you and `render_template` - it is not the product,
and it is not something to reproduce by hand.

```
get_cookbook_recipe  name: "event"      ->  the template, ready for render_template
```

Every recipe renders at three canvas sizes in CI, so one that no longer works cannot
reach this list. Their images come from `s.imagor.net` and render as they are.

| name | what it is | size | values to fill |
|---|---|---|---|
| `announcement` | Headline, one supporting line, one visual. The tech/startup announcement card. | 1200x628 | `eyebrow`, `headline`, `mark`, `supporting` |
| `avatar-card` | A landscape profile card with a circular portrait and a left-aligned identity block. | 1200x630 | `handle`, `name`, `role` |
| `circles` | A row of contributors as circular portraits with names and roles beneath. The portraits should be images on a plain or transparent ground - a source with its own background shows that background as a square block around the circle, which is the source's shape and not a fault in the corner. | 1200x630 | `brand`, `footer`, `headline`, `person1_image`, `person1_name`, `person1_role`, `person2_image`, `person2_name`, `person2_role`, `person3_image`, `person3_name`, `person3_role`, `person4_image`, `person4_name`, `person4_role` |
| `comparison` | Two matched columns of points, for a before and after. | 1200x630 | `brand`, `footer`, `headline`, `left_label`, `left_point_1`, `left_point_2`, `left_point_3`, `right_label`, `right_point_1`, `right_point_2`, `right_point_3` |
| `event` | Date block, title, venue. The open-house / meetup format. | 1200x628 | `day`, `month`, `title`, `venue` |
| `launch-card` | Editorial split layout for a seasonal accessories collection announcement. | 1200x630 | `availability`, `brand`, `cta`, `headline`, `photo`, `supporting` |
| `news-announcement` | A restrained, editorial announcement for a product return, pairing a prominent headline with a supporting quote and dateline. | 1200x630 | `headline`, `kicker`, `quote`, `source` |
| `og-modern` | A dark, restrained Open Graph card with a textured navy ground, split headline, and fitted product grid. | 1200x630 | `brand`, `domain`, `headline_accent`, `headline_lead`, `photo1`, `photo2`, `photo3`, `photo4`, `photo5`, `photo6`, `supporting` |
| `og-standard` | A text-first 1200x630 Open Graph layout with a small brand mark, prominent title, supporting description, and quiet domain footer. | 1200x630 | `brand`, `description`, `domain`, `title` |
| `product-grid` | Lifestyle-led editorial product grid with a tall campaign panel and product captions nested in each image canvas. | 1800x942 | `p1_name`, `p1_price`, `p2_name`, `p2_price`, `p3_name`, `p3_price`, `p4_name`, `p4_price`, `p5_name`, `p5_price`, `src_life`, `src_p1`, `src_p2`, `src_p3`, `src_p4`, `src_p5`, `wordmark` |
| `product-og` | An editorial product share card with a featured product photograph and concise offer details. | 1200x630 | `collection`, `cta`, `featured_image`, `price`, `title` |
| `quote-card` | A pull quote on deep green: the quotation in serif, attributed below a rule. | 1200x630 | `attribution`, `quote`, `role` |
| `testimonial` | B2B customer quote with an inset forest panel, warm paper margins and restrained brass accents. | 1200x628 | `company`, `name`, `quote` |
| `tip` | One actionable tip per card. The educational format. | 1200x628 | `mark`, `number`, `tip` |

## Choosing one

Start from the closest recipe and change the values rather than composing from
nothing: they carry spacing and type sizes that took iteration to get right. If none
fits, build from scratch - the skill covers the path grammar and the method.
