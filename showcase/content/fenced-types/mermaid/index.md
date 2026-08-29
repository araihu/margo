---
title: Mermaid fence
description: Render a Mermaid diagram as a sanitized SVG projection.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Mermaid fence

Use a `mermaid` fence for diagrams. Margo assigns deterministic IDs, applies a
known Mermaid runtime, and sanitizes the resulting SVG before publication.

## Source

````markdown
```mermaid
flowchart LR
  source[Markdown] --> compile[Compile]
  compile --> output[Semantic output]
```
````

## Result

```mermaid
flowchart LR
  source[Markdown] --> compile[Compile]
  compile --> output[Semantic output]
```

## Options

The fence body is Mermaid syntax. Choose the diagram family in the first line
(`flowchart`, `sequenceDiagram`, and others supported by the pinned runtime).
Margo owns deterministic IDs and SVG sanitization; arbitrary runtime
configuration is not accepted in the document body.
