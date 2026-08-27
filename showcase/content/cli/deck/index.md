---
title: deck
description: Render a versioned Marpit-compatible Markdown presentation as HTML or PDF.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# `margo deck`

## Purpose

`deck` renders one Markdown source through Margo's versioned Marpit-compatible
deck profile. Slides can be separated by top-level thematic breaks or supported
heading-divider pagination.

## Input and output

Input is a path or `-` for stdin. The default format is HTML and the default
output is stdout. `--format pdf` requires an explicit `--output PATH` or
`--output -`. HTML or PDF bytes go to the selected output; failures go to
stderr.

PDF decks reuse the PDF engine, page, image-overflow, and atomic output
contracts. Their page margins default to zero, and relative PDF links use
`strip`. Page metadata in the document still applies unless an explicit page
or slide-geometry flag overrides it.

## Examples

Create and render a two-slide example:

```sh
cat > slides.md <<'MARKDOWN'
---
title: Release review
language: en
---

<!-- paginate: true -->

# Release review

One source, two projections.

---

# Next step

Verify the artifact before publication.
MARKDOWN

margo check slides.md --target deck
margo deck slides.md --output build/slides.html
margo deck slides.md \
  --format pdf \
  --output build/slides.pdf \
  --slide-size 16:9 \
  --engine auto
```

Preset geometry selects a logical canvas: `16:9` is 1280×720 and `4:3` is
960×720. Custom geometry requires all four declarations:

```sh
margo deck slides.md \
  --slide-size custom \
  --slide-width 1920 \
  --slide-height 1080 \
  --slide-unit px \
  --output build/slides-wide.html
```

Host-owned chrome can add a bounded confidentiality label and an icon from the
Goshtoso catalog beside enabled page ordinals:

```sh
margo deck slides.md \
  --confidentiality-badge "Internal" \
  --pagination-icon hi-16-solid-clock \
  --pagination-icon-placement before \
  --pagination-icon-decorative \
  --output build/slides-internal.html
```

For an informative icon, omit `--pagination-icon-decorative` and provide
`--pagination-icon-label`. `--print-chart-data` includes accessible exact-data
tables after supported charts.

### Charts in decks

Deck HTML and PDF are static slide projections. Use the default renderer or
write `renderer: static`; `renderer: interactive` is intentionally rejected for
the `deck` target because browser chart controls are not part of the deck
artifact contract. Run the target check first to get the same actionable
diagnostic that rendering would produce:

~~~sh
cat > slides.md <<'MARKDOWN'
# Revenue review

```goshtosochart
schemaVersion: 1
type: line
renderer: static
title: Weekly revenue
categories: [Mon, Tue, Wed]
series:
  - name: Revenue
    values: [12, 18, 21]
```
MARKDOWN

margo check slides.md --target deck
margo deck slides.md --format html --output build/slides.html
margo deck slides.md --format pdf --output build/slides.pdf --slide-size 16:9
~~~

The interactive renderer remains available for standalone HTML, sites, and
`margo pdf`; PDF rasterizes the interactive chart for print. For a deck, keep
the exact-data table with `--print-chart-data` when the printed artifact needs
the tabular fallback. PDF decks apply compact print styling and let the table
reflow to its natural height, keeping all declared rows visible in the fixed
slide canvas. If a larger table still cannot fit, Margo reports the affected
slide and suggests reducing the chart data or choosing a larger slide size.

## Structural layouts

Structural layouts are a closed Markdown contract. A slide uses a class marker,
the matching `layout` marker, source-ordered `slot` markers, and a closing
`/layout` marker. Use the `_class` spelling when the class should apply only to
the current slide; an unprefixed `class` directive is inherited by following
slides until it changes. There must be no unmarked body text inside a
structural layout, and every slot must contain non-empty Markdown.

| Layout | Slot names, in source order | Cardinality |
| --- | --- | --- |
| `columns` | `left`, `right` | exactly 2 |
| `sidebar` | `main`, `rail` | exactly 2 |
| `compare` | `left`, `right` | exactly 2 |
| `metrics` | `metric-1` through `metric-4` | 3 to 4 |
| `timeline` | `step-1` through `step-6` | 3 to 6 |
| `demo` | `code`, `result` | exactly 2 |

The following is one complete deck containing every structural layout. The
comments are part of the deck source; keep each marker on its own line and do
not change its name. The `_class` markers make each slide independent when this
file is copied into another deck.

~~~sh
mkdir -p build
cat > build/structural-layouts.md <<'MARKDOWN'
---
title: Structural layouts
description: The six closed structural layouts in the Margo deck profile.
language: en
lang: en
theme: modern
colorMode: light
size: 16:9
---

<!-- _class: columns -->
<!-- layout: columns -->
<!-- slot: left -->
# Context

The left column carries the premise or primary evidence.
<!-- slot: right -->
# Decision

The right column carries the consequence or supporting action.
<!-- /layout -->
<!-- speaker note: compare the two columns in source order. -->

---

<!-- _class: sidebar -->
<!-- layout: sidebar -->
<!-- slot: main -->
## Main

Use the main slot for the thesis, evidence, and reading flow.
<!-- slot: rail -->
### Rail

Use the rail for definitions, context, or a short callout.
<!-- /layout -->
<!-- speaker note: keep the rail short enough to remain subordinate. -->

---

<!-- _class: compare -->
<!-- layout: compare -->
<!-- slot: left -->
## Option A

- controlled authoring
- deterministic output
<!-- slot: right -->
## Option B

- familiar Markdown input
- bounded visual vocabulary
<!-- /layout -->
<!-- speaker note: state the decision criteria before comparing options. -->

---

<!-- _class: metrics -->
<!-- layout: metrics -->
<!-- slot: metric-1 -->
### 6

structural layouts
<!-- slot: metric-2 -->
### 2

two-column variants
<!-- slot: metric-3 -->
### 3–6

timeline slots
<!-- slot: metric-4 -->
### 0

arbitrary CSS rules
<!-- /layout -->
<!-- speaker note: read the metrics left to right. -->

---

<!-- _class: timeline -->
<!-- layout: timeline -->
<!-- slot: step-1 -->
### Input

Write Markdown and closed directives.
<!-- slot: step-2 -->
### Normalize

Resolve the class, slots, and source order.
<!-- slot: step-3 -->
### Publish

Render HTML or validate a PDF projection.
<!-- /layout -->
<!-- speaker note: pause after each step before advancing. -->

---

<!-- _class: demo -->
<!-- layout: demo -->
<!-- slot: code -->
### Source

```markdown
<!-- composition: media-split -->
<!-- slot: media -->
![Evidence](assets/evidence.svg)
<!-- slot: content -->
## Decision
Keep source order explicit.
```
<!-- slot: result -->
### Result

The same source can produce a navigable HTML deck and a validated PDF.
<!-- /layout -->
<!-- speaker note: presenter notes are retained by the Go parser, not painted on the slide. -->
MARKDOWN

margo check build/structural-layouts.md --target deck --diagnostics json
margo deck build/structural-layouts.md --output build/structural-layouts.html
~~~

The sample carries both `language` and the deck directive `lang`: the generic
Margo check uses `language`, while deck labels and localized chrome use `lang`.
For a deck that is rendered only through the Go API, `lang` is the deck-local
language control.

The `layout` and `/layout` markers are structural markers, not free-form CSS.
`columns` and `compare` intentionally share the `left`/`right` slot contract,
while `sidebar`, `metrics`, `timeline`, and `demo` use the names in the table.
Slot order is preserved in the normalized model, HTML reading order, keyboard
order, and PDF projection; CSS never reorders slots.

### Composition presets are separate

The optional `composition` directive selects the versioned R1 catalog and has
different slot contracts. For example, `media-split` uses `media` and
`content`, `compare-grid` uses `item-1` through `item-4`, and `steps` uses
`step-1` through `step-6`. Do not substitute those names into an uncomposed
`columns`, `compare`, or `timeline` layout. See the
[composition reference](https://github.com/araihu/margo/blob/v0.0.17/docs/reference/deck-compositions-r1.md) for
the complete preset catalog and its cardinalities.

## Themes, directives, and presenter notes

Deck themes are closed: `modern`, `goshtoso`, and `minimal`. `colorMode` is
`light` or `dark`, and `lang` is a BCP 47-style language tag. Global directives
can be placed in frontmatter or in an unprefixed HTML comment outside a fenced
code block:

```yaml
---
title: Themed deck
lang: pt-BR
theme: goshtoso
colorMode: dark
headingDivider: 2
size: 4:3
---
```

Local directives are `paginate`, `header`, `footer`, `class`, `color`,
`backgroundColor`, `backgroundImage`, `backgroundPosition`,
`backgroundRepeat`, `backgroundSize`, `backgroundDecorative`, `backgroundAlt`,
and `composition`. An unprefixed local directive is inherited by subsequent
slides; prefix it with `_` for the current slide only, for example
`<!-- _class: invert -->` or `<!-- _composition: none -->`. Colors use finite
theme tokens, and backgrounds use a local asset or an approved gradient token.
Remote URLs, arbitrary CSS, `style`, custom Marpit themes, and unknown
extension allocators are rejected.

The `color` and `backgroundColor` directives accept only these finite tokens:

| Kind | Tokens |
| --- | --- |
| Text (`color`) | `surface`, `surface-alt`, `ink`, `ink-muted`, `accent`, `accent-strong`, `positive`, `warning`, `negative`, `info`, `transparent` |
| Background (`backgroundColor`) | `surface`, `surface-alt`, `accent`, `accent-strong`, `positive`, `warning`, `negative`, `info`, `transparent` |
| Gradient background (`backgroundImage`) | `gradient-blue`, `gradient-violet`, `gradient-sunset`, `gradient-forest` |

`color` and `backgroundColor` resolve against the selected theme and color mode;
the token names do not expose raw CSS values. `backgroundImage` may instead be
a normalized local asset path. If a token fails contrast validation, keep the
same semantic role and switch `colorMode`, theme, or the `invert` class rather
than adding arbitrary CSS.

An HTML comment that is not a recognized directive is a presenter note:

```markdown
<!-- speaker note: explain why the right column follows the left. -->
```

Notes stay in source order and are available through `deck.Slide.Notes()` in
the Go API. The CLI deck projection does not render a presenter pane or expose
notes as visible slide content; keep the comments in the source when a host
application needs to read them.

## Failures and diagnostics

Run the check before rendering to get source locations and recovery hints:

```sh
margo check build/structural-layouts.md --target deck --diagnostics json
```

| Code | Meaning | Recovery |
| --- | --- | --- |
| `deck.layout_invalid` | Unknown, mismatched, nested, or unclosed structural markers. | Use one matching `layout`/`_class` pair and close it with `<!-- /layout -->`. |
| `deck.layout_slots_required` | A structural layout has too few or too many slots. | Use the exact slot names and cardinality in the table above. |
| `deck.slot_invalid` | A slot is duplicated, unknown, or out of order. | Keep each slot once and preserve the required sequence. |
| `deck.class_unsupported` | The class is outside the closed layout/style vocabulary. | Choose a documented class such as `columns`, `sidebar`, `compare`, `metrics`, `timeline`, or `demo`. |
| `deck.class_combination_invalid` | Multiple structural/style classes were combined. | Select one structural class, with only the supported `invert` modifier when needed. |
| `deck.directive_invalid` | A directive value or type is outside its contract. | Use the accepted theme, color, geometry, pagination, and background values. |
| `deck.directive_unsupported` | An author-authored style or other unsupported directive was used. | Remove it; arbitrary CSS and custom Marpit themes are not part of the profile. |
| `deck.background_invalid` | A background is remote, malformed, or uses an unsupported value. | Use a local asset or one of the approved gradient tokens. |
| `deck.fence_unclosed` | A Markdown fence was not closed. | Close the fence before the next slot or slide separator. |
| `deck.composition_invalid` | A composition name or value is outside the R1 catalog. | Use a documented lowercase composition name or `none`. |
| `deck.composition_conflict` / `deck.composition_slot_invalid` | A composition disagrees with its class or slot contract. | Use the composition's own family and ordered slot names. |
| `cli.format_invalid` | A format other than `html` or `pdf` was selected. | Pass `--format html` or `--format pdf`. |
| `cli.output_required` | PDF output has no explicit destination. | Pass `--format pdf --output PATH` or `--output -`. |
| `cli.deck_geometry_invalid` | Slide size or custom geometry is incomplete or unsupported. | Use `16:9`, `4:3`, or positive custom width, height, and unit values. |
| `cli.deck_geometry_conflict` | Slide geometry conflicts with legacy page flags. | Use slide geometry or page geometry, not both. |
| `cli.deck_validator_unavailable` | PDF validation has no supported Chromium-compatible validator. | Install/select a supported local Chromium and retry, or render HTML. |
| `deck.confidentiality_badge_invalid` / `deck.pagination_icon_invalid` | Host chrome options are invalid. | Use bounded badge text or a catalog icon; informative icons need a label. |

Existing output files are protected; add `--force` only when replacement is
intentional. Engine failures otherwise use the same `pdf.*` diagnostics as
`margo pdf`.

## Limitations and care

Deck projection does not expose the standalone `--title`, `--lang`,
`--base-url`, or `--relative-links` flags. Preset slide geometry cannot be
combined with custom width, height, or unit flags; custom geometry requires a
positive width and height plus an explicit unit. Legacy `--page-size` and
`--orientation` must match the selected slide canvas.

Arbitrary HTML and CSS, remote backgrounds, custom Marpit themes, and unknown
extension allocators remain rejected. PDF decks validate page count and page
edges against the logical canvas before publishing the destination.
