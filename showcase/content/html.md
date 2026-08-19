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

Render a Markdown file as a browser-ready HTML document. Write it to stdout,
create a file, or compose it inside a host-owned page shell.

## CLI

```sh
margo check proposal.md
margo html proposal.md --output proposal.html
```

Output goes to stdout by default. An explicit path creates a file; `--force` is
required to replace one.

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

`RenderHTML` returns the semantic document fragment and its dependency
requirements. `RenderHTMLPage` adds a generic page shell. A host can instead
compose its own navigation and identity, as this showcase does with a Goshtoso
documentation shell.

## Local or inline assets

Choose local dependencies when the host publishes an asset directory:

```go
margo.HTMLDependenciesLocal
```

Choose inline dependencies when the HTML document must carry its runtime and
styles:

```go
margo.HTMLDependenciesInline
```

Dependency mode changes packaging, not the Markdown source.
