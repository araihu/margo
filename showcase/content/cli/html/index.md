---
title: html
description: Render one Markdown document as standalone HTML with separate diagnostics.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# `margo html`

## Purpose

`html` renders one complete standalone HTML document.

## Input and output

Input is one Markdown path or `-` for stdin. Output defaults to `-`, so HTML
bytes go to stdout unless `--output PATH` selects a file. Diagnostics go to
stderr and never mix with HTML bytes.

`--title` overrides the document title and accepts at most 256 UTF-8 bytes.
`--lang` overrides the BCP 47 document language. Local images are materialized
for the standalone artifact; unsupported or external resource forms are
rejected instead of producing a silently broken page.

## Examples

```sh
margo html docs/guide.md \
  --title "Publishing guide" \
  --lang en \
  --output build/guide.html

margo html docs/guide.md | gzip -9 > build/guide.html.gz
```

Replace an existing output only when intended:

```sh
margo html docs/guide.md --output build/guide.html --force
```

## Failures and diagnostics

The file sink is atomic and refuses an existing destination with
`margo.atomic.destination_exists`. Invalid language or title metadata reports
`html.metadata_invalid`. Input, policy, parser, resource, and output failures
exit `1` without committing a partial destination.

## Limitations and care

`html` builds one page; it does not rewrite links across a directory or create
a site manifest. Review the [CLI overview](../index.md) before supplying a
trusted policy or using stdin with relative resources.
