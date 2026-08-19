---
title: Charts
description: Add static SVG charts with accessible data tables and optional interactive controls.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Charts

Add charts through Margo's optional chart extension. The CLI registers it
automatically; Go applications opt in with `charts.Extension()`.

## A chart fence

```goshtosochart
schemaVersion: 1
type: line
renderer: static
title: Weekly signal
categories: [Mon, Tue, Wed, Thu]
series:
  - name: Requests
    values: [12, 18, 16, 24]
```

Static rendering produces an SVG followed by a semantic exact-data table. The
values remain available to readers and assistive technology without JavaScript.

## Go integration

```go
compiler := margo.New(
    margo.WithExtension(charts.Extension()),
)
```

The extension supports `bar`, `line`, `pie`, `doughnut`, and `scatter`. Static
SVG is the default. Set `renderer: interactive` to add Goshtoso chart controls;
the accessible data table remains in the output.

## Print behavior

Margo disables chart animation for deterministic capture. PDF output omits
exact-data tables by default; `--print-chart-data` includes them after each
chart.
