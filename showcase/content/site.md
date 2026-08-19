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

Build a linked HTML site from a directory of Markdown files. Margo maps source
paths to `.html` routes, packages local assets, and validates Markdown links
against the discovered pages.

## Build a directory

```sh
margo site ./content --output-dir ./dist --assets local
```

Use `--assets inline` to place generated dependencies in each page. The output
directory must be new. Margo stages the complete site before publishing it.
Pass a YAML config instead when the publication needs a shell, navigation,
identity, canonical URLs, or social metadata.

## Preview while writing

Use `margo serve` while editing. It accepts either a Markdown directory or the
same site config used for a build:

```sh
margo serve ./content
margo serve ./showcase.yaml --open
```

The server watches pages and local assets, rebuilds into memory, and reloads the
browser after a successful build. After a failed build, it reports diagnostics
and continues serving the last successful snapshot.

This server is a local development tool, not a production deployment target.

## What the builder verifies

| Input concern | Site result |
| --- | --- |
| Markdown page path | A deterministic `.html` route |
| Relative Markdown link | A rewritten route or an actionable error |
| Local image | A copied asset or an inline data URL |
| Configured publication metadata | Canonical, description, and social tags |
| Build output | `margo-manifest.json` with artifact digests |

## A site has a composition boundary

Margo supplies article and route data. The host selects the page shell,
navigation, and identity assets. This showcase uses the public Goshtoso
`componentdocshell` for its header, sidebar, responsive drawer, dark-mode
control, and on-page table of contents.

The Markdown source remains independent from that presentation choice.
