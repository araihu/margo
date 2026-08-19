---
title: Margo
description: A visual guide to Margo's Markdown compiler, Go module, and publishing CLI.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Margo

Convert Markdown to `HTML`

![Margo mascot preparing a document](margo-mascot.png)

Margo is one Go module and one `margo` command for turning ordinary Markdown
into standalone HTML, linked static sites, PDF documents, and an experimental
presentation deck projection. This showcase is the public path through those
capabilities: each page is a small, runnable-looking example of one feature.

## One source, several projections

```mermaid
treeView-beta
  "ordinary Markdown"
    check ## compatibility findings
    html ## standalone HTML
    site ## linked HTML pages
    serve ## development preview and live reload
    pdf ## browser-backed PDF
    deck ## experimental deck projection
    mermaid ## rendered diagram plus source fallback
```

The compiler keeps the document content separate from the host-owned frame,
metadata, assets, and publication choices. The CLI adds explicit output and
diagnostic boundaries around the same module surface. This tree uses Mermaid's
[TreeView syntax](https://mermaid.js.org/syntax/treeView.html), so the visual
structure remains source text with a readable fallback.

## A predictable workflow

```sh
margo check guide.md
margo html guide.md --output guide.html
margo serve ./docs --open
margo pdf guide.md --output guide.pdf
```

For a larger publication, point `margo site` at a directory of Markdown files.
While writing, use `margo serve` for an in-memory preview with live reload; it
is a development tool, not a production server. For a programmatic integration,
import the root module and choose the package that owns the projection you need.

> This is a feature tour, not the project's internal decision log. For source,
> releases, and the complete public README, visit the
> [Margo repository](https://github.com/araihu/margo#readme).
