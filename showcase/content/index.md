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

## Start with the surface

| Feature | Output | Best first stop |
| --- | --- | --- |
| Markdown compiler | A semantic document model | [Markdown](markdown.md) |
| Standalone HTML | One portable HTML page | [HTML](html.md) |
| Static sites | Linked pages plus a manifest | [Static sites](site.md) |
| Development server | In-memory preview with live reload | [CLI workflows](cli.md#develop-with-live-reload) |
| PDF documents | Print-ready PDF bytes | [PDF](pdf.md) |
| Presentation decks | Experimental HTML/PDF projection | [Decks](decks.md) |
| Charts | Static SVG, accessible data, optional interaction | [Charts](charts.md) |
| Mermaid diagrams | Rendered flowcharts with a text fallback | [Mermaid](mermaid.md) |
| Policy and diagnostics | Actionable validation | [Policy](policy.md) |

Ready to write a site? Start with the
[live-reload development workflow](cli.md#develop-with-live-reload).

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
