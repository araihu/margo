---
title: Goshtoso chart fence
description: Render a chart fence into static SVG with accessible exact data.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Goshtoso chart fence

The optional `goshtosochart` fence describes a chart in YAML. Register the
charts extension in a Go host; static SVG is the default, while interactive
HTML is an explicit capability.

## Source

````markdown
```goshtosochart
schemaVersion: 1
type: bar
renderer: static
title: Weekly output
categories: [Mon, Tue, Wed]
series:
  - name: Pages
    values: [3, 5, 4]
```
````

## Result

```goshtosochart
schemaVersion: 1
type: bar
renderer: static
title: Weekly output
categories: [Mon, Tue, Wed]
series:
  - name: Pages
    values: [3, 5, 4]
```

Static charts can include an exact-data table for screen readers and print
readers. Deck output keeps the static projection by contract.

<style>
.margo-chart-data { display: none; }
.chart-example--data .margo-chart-data { display: block; }
</style>

The examples below keep the visual projection prominent. The first chart also
exposes its exact-data table as an accessibility example.

<div class="chart-example--data">

```goshtosochart
schemaVersion: 1
type: bar
renderer: static
title: Exact values
categories: [Mon, Tue, Wed]
series:
  - name: Pages
    values: [3, 5, 4]
```

</div>

## Line chart

```goshtosochart
schemaVersion: 1
type: line
renderer: interactive
title: Weekly trend
categories: [Mon, Tue, Wed]
series:
  - name: Pages
    values: [3, 5, 4]
```

## Pie and doughnut charts

```goshtosochart
schemaVersion: 1
type: pie
renderer: interactive
title: Request share
slices:
  - name: API
    value: 60
  - name: UI
    value: 40
```

Change `type` to `doughnut` to use the same slice payload with a doughnut
projection.

## Scatter chart

```goshtosochart
schemaVersion: 1
type: scatter
renderer: static
title: Latency by region
categories: [us, eu, ap]
series:
  - name: P95 latency
    points:
      - category: us
        value: 120
      - category: eu
        value: 150
      - category: ap
        value: 180
```

## Options

Set `type`, `title`, `categories`, and `series` in the YAML payload. Choose
`renderer: static` for a portable SVG; `renderer: interactive` requires the
registered chart runtime and is not available in every target. See the
[Goshtoso chart schemas](../../schemas/goshtoso-chart/index.md) for the full
payload contracts.

The YAML block above is the complete source payload for this chart fence.
