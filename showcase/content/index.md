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

Publish Markdown in the format you need.

![Margo mascot preparing a document](margo-mascot.png)

Margo turns Markdown source into standalone HTML, linked static sites, PDF
documents, and experimental presentation decks. Use the `margo` command for
publishing workflows or the Go module to integrate the same compiler into an
application.

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

Every projection starts with the same compiled document. The application or
site config still owns its frame, metadata, assets, routes, and publication
destination. This diagram uses Mermaid's
[TreeView syntax](https://mermaid.js.org/syntax/treeView.html), preserving a
readable source fallback with the visual.

## A predictable workflow

```sh
margo check guide.md
margo html guide.md --output guide.html
margo serve ./docs --open
margo pdf guide.md --output guide.pdf
```

Use `margo site` for a directory of Markdown files. While writing, run
`margo serve` for an in-memory preview with live reload. The development server
is not a production server. For programmatic use, import the root module and
choose the package for the required output.

> This showcase covers public features and runnable paths. For source, releases,
> and the complete README, visit the
> [Margo repository](https://github.com/araihu/margo#readme).
