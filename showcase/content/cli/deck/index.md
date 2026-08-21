---
title: deck
description: Project Markdown as an experimental HTML or PDF presentation.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# `margo deck`

## Purpose

`deck` projects one Markdown source as an experimental presentation. Slides
are separated by thematic breaks.

## Input and output

Input is a path or `-` for stdin. The default format is HTML and the default
output is stdout. `--format pdf` requires an explicit `--output PATH` or
`--output -`. HTML or PDF bytes go to the selected output; failures go to
stderr.

PDF decks reuse the PDF engine, page, image-overflow, and atomic output
contracts. Their page margins default to zero, and relative PDF links use
`strip`. Page metadata in the document still applies unless an explicit page
flag overrides it.

## Examples

Create and render a two-slide example:

```sh
cat > slides.md <<'MARKDOWN'
---
title: Release review
language: en
---

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
  --engine auto
```

## Failures and diagnostics

`cli.format_invalid` reports a format other than `html` or `pdf`.
`cli.output_required` reports a PDF deck without an explicit destination.
Engine and geometry failures use the same `pdf.*` codes as `pdf`. Existing
files require `--force`.

## Limitations and care

Deck projection remains experimental. It does not expose the standalone
`--title`, `--lang`, `--base-url`, `--relative-links`, or
`--print-chart-data` flags. Validate both HTML slide behavior and the final PDF
before relying on it for a presentation.
