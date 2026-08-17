---
title: Standalone HTML
description: Render one Markdown document as a self-contained HTML page or a local-asset page.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Standalone HTML

The HTML projection is the shortest path from a Markdown file to a browser
document. It can be emitted to stdout, written to a new file, or composed by a
Go host that wants to provide its own page shell.

## CLI

```sh
margo check proposal.md
margo html proposal.md --output proposal.html
```

The default output is stdout. An explicit output path creates a file, and Margo
does not replace an existing file unless `--force` is present.

## Host-owned composition

```go
htmlResult, err := margo.RenderHTML(rendered)
if err != nil {
    return err
}

page, err := margo.RenderHTMLPage(htmlResult, margo.HTMLPageInput{
    DependencyMode: margo.HTMLDependenciesLocal,
    Head:           siteMetadata(),
    Header:         siteNavigation(),
    Footer:         siteFooter(),
})
```

`RenderHTML` exposes the semantic document fragment and its dependency
requirements. `RenderHTMLPage` supplies a generic page shell without claiming
ownership of a publication domain. That boundary is what lets a documentation
site add its own navigation—or, as this showcase does, compose a Goshtoso
documentation shell.

## Local or inline assets

Choose local dependencies when a host will publish an asset directory:

```go
margo.HTMLDependenciesLocal
```

Choose inline dependencies when the artifact should carry its runtime and
styles with it:

```go
margo.HTMLDependenciesInline
```

The choice changes packaging, not the Markdown content.
