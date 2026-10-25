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

In HTML documents and sites, each rendered diagram has **Zoom in**, **Zoom out**,
and **Fit diagram** controls. **Expand**, or a double-click on the diagram,
opens the same diagram in a large modal;
**Fullscreen** uses the whole screen, with a viewport-sized modal when native
fullscreen is unavailable. Zoom and pan remain usable in both views. Close the
expanded view or press `Escape` to return; closing restores the original view
and focus. In native fullscreen, `Escape` first exits fullscreen.

The information button opens Goshtoso's operation help on demand.
Use `Ctrl` + wheel over a diagram to zoom around the pointer. Without `Ctrl`,
the wheel scrolls normally. Focus the diagram and use `+` / `-` to zoom, `0`
or `Home` to fit, and arrow keys to pan. You can also drag with the left or
middle mouse button, or swipe on a touch screen after zooming.
Each diagram keeps its own view. Page scrolling
and browser zoom remain available outside the diagram.

The complete source remains available to assistive technology. Without
JavaScript, the source disclosure remains available. Printed pages and PDF
exports use the full static diagram and omit the interactive controls.

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
