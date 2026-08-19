---
title: Static sites
description: Build linked HTML pages from a Markdown directory with route and asset validation.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Static sites

The site projection turns a directory of Markdown files into a linked HTML
publication. Source paths become `.html` routes, local assets are copied or
embedded, and Markdown links are checked against the discovered page set.

## Build a directory

```sh
margo site ./content --output-dir ./dist --assets local
```

Use `--assets inline` when a page should carry its generated dependencies in
the document. The output directory must be new; the site is staged and only
published after the build succeeds.

## Preview while writing

`margo serve` is the development companion to the publication command. It can
start from the same config or directly from a Markdown directory:

```sh
margo serve ./content
margo serve ./showcase.yaml --open
```

The preview is served from an immutable in-memory snapshot. Margo recursively
watches pages and local assets, serializes rebuilds, and signals the browser to
reload only after a successful replacement. If a build fails, the last good
snapshot remains available while the terminal shows the diagnostics.

This server is a local development tool, not a production deployment target.

## What the builder verifies

| Input concern | Site result |
| --- | --- |
| Markdown page path | A deterministic `.html` route |
| Relative Markdown link | A rewritten route or an actionable error |
| Local image | A copied asset or an inline data URL |
| Page metadata | Canonical, description, and social tags |
| Build output | `margo-manifest.json` with artifact digests |

## A site has a composition boundary

Margo supplies the article and route data. The host selects the visual frame or
shell, maps navigation, and chooses its identity assets. This showcase uses
the public Goshtoso `componentdocshell` for the header, sidebar, responsive
drawer, dark-mode control, and on-page table of contents.

The content remains plain Markdown. The shell is a presentation choice applied
after the document has been rendered.
