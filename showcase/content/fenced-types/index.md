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
- [Math and LaTeX](math/index.md) — inline expressions and display equations, including `math` fences, enabled with `--math`.
- [Goshtoso charts](goshtoso-chart/index.md) — static SVG charts with accessible data.
- [JSON Schema](jsonschema/index.md) — versioned schema trees in documentation.
- [Code blocks](code/index.md) — highlighted source with copy controls.

Fences are parsed during compilation. Enable core math with `--math` or
`margo.WithMath(true)`; register extension fences with the host. A target can
reject a fence when its projection is unsupported.
