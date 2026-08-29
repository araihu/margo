---
title: Fenced types
description: The specialized Markdown fences that Margo compiles into semantic projections.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Fenced types

Margo keeps specialized Markdown blocks explicit. Each page documents the
fence's source syntax and shows the resulting projection.

## Built-in and optional fences

- [Mermaid](mermaid/index.md) — diagrams rendered as sanitized SVG.
- [Goshtoso charts](goshtoso-chart/index.md) — static SVG charts with accessible data.
- [JSON Schema](jsonschema/index.md) — versioned schema trees in documentation.
- [Code blocks](code/index.md) — highlighted source with copy controls.

Fences are parsed during compilation. Optional fences must be registered by the
host, and a target can reject a fence when its projection is unsupported.
