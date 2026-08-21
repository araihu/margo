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

## Failures and diagnostics

`cli.format_invalid` reports a format other than `html` or `pdf`.
`cli.output_required` reports a PDF deck without an explicit destination.
`cli.deck_geometry_invalid` reports an incomplete or unsupported slide size;
`cli.deck_geometry_conflict` reports incompatible slide and legacy page flags.
`cli.deck_validator_unavailable` reports PDF validation without a supported
Chromium-compatible validator. Engine failures otherwise use the same `pdf.*`
codes as `pdf`. Invalid host chrome reports `deck.confidentiality_badge_invalid`
or `deck.pagination_icon_invalid`. Existing files require `--force`.

## Limitations and care

Deck projection does not expose the standalone `--title`, `--lang`,
`--base-url`, or `--relative-links` flags. Preset slide geometry cannot be
combined with custom width, height, or unit flags; custom geometry requires a
positive width and height plus an explicit unit. Legacy `--page-size` and
`--orientation` must match the selected slide canvas.

Arbitrary HTML and CSS, remote backgrounds, custom Marpit themes, and unknown
extension allocators remain rejected. PDF decks validate page count and page
edges against the logical canvas before publishing the destination.
