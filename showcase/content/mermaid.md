---
title: Mermaid diagrams
description: Turn a Mermaid fence into an accessible flow diagram with a readable source fallback.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Mermaid diagrams

Describe a diagram as Mermaid source inside Markdown. Margo keeps the source,
rendered figure, and accessible description together in the document.

## A publishing flow

```mermaid
flowchart LR
    source[Markdown source] --> check{Check compatibility}
    check -->|pass| render[Render once]
    check -->|findings| revise[Revise source]
    revise --> check
    render --> html[Standalone HTML]
    render --> site[Static site]
    render --> pdf[PDF or deck]
```

Readers can follow the rendered flow, inspect its source, or use the text
description when the browser runtime is unavailable.

## Choose an output

```sh
margo check guide.md
margo html guide.md --output guide.html
margo site docs --output-dir public
```

The Mermaid fence stays with the document when Margo renders standalone HTML, a
linked site, a PDF, or an experimental deck.

## What stays available

- The source stays versionable beside the prose it explains.
- The browser can render a visual diagram without asking the author to export
  an image first.
- The source disclosure and accessible description keep the relationship
  readable beyond the canvas.

Mermaid is ordinary authoring input. Raw HTML and iframe behavior remain behind
the explicit policy surface.
