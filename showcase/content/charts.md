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

Charts are an optional Margo extension. The CLI registers the chart extension
for you; Go applications opt in with `charts.Extension()`.

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

Static rendering produces an SVG and an adjacent semantic exact-data table.
The table keeps the values available to readers and assistive technology even
when JavaScript is disabled.

## Go integration

```go
compiler := margo.New(
    margo.WithExtension(charts.Extension()),
)
```

The extension supports `bar`, `line`, `pie`, `doughnut`, and `scatter` families.
Static SVG remains the default. Set `renderer: interactive` when the host also
wants Goshtoso chart controls; the accessible data table remains part of the
output contract.

## Print behavior

Chart animation is disabled for deterministic capture. PDF output omits exact
data tables by default; `--print-chart-data` includes them for a data-forward
print artifact.
