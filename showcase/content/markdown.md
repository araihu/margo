---
title: Markdown compiler
description: Compile ordinary Markdown once, then project it to the output you need.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Markdown compiler

Start with ordinary Markdown. Margo compiles the source into a semantic
document, renders it for a target, then lets the host choose the final page
shape.

## Compile in Go

```go
compiler := margo.New()

document, err := compiler.Compile(ctx, margo.Source{
    Name:    "guide.md",
    Content: markdown,
})
if err != nil {
    return err
}

rendered, err := compiler.Render(ctx, document)
if err != nil {
    return err
}

page, err := margo.RenderStandalone(rendered)
if err != nil {
    return err
}
```

The compiler owns Markdown parsing and document semantics. The application owns
routes, navigation, assets, and publication.

## The authoring surface

```markdown
---
title: A small guide
language: en
---

# A small guide

Write headings, links, tables, code, images, and Mermaid diagrams as Markdown.
```

Local images, tables, Mermaid, and code work without a policy file. Raw HTML and
iframe embeds require explicit host authorization; see [Policy](policy.md).

## Use the compiled document

- Render one source as HTML, a site, a PDF, or an experimental deck.
- Carry metadata with the document, then override it through an explicit API or
  CLI option.
- Run `margo.Check` for compatibility findings without producing an artifact.

These boundaries keep authoring input independent from application-owned
presentation and delivery.
