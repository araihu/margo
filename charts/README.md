# Margo Charts

`github.com/araihu/margo/charts` is an optional package in the root Margo Go
module. It registers the `goshtosochart` fence for Goshtoso Charts and
accessible adjacent data tables. Static SVG remains the default. The root compiler reports
`extension.missing_integration` until a consumer registers the extension.

```go
compiler := margo.New(margo.WithExtension(charts.Extension()))
```

Version 1 accepts `bar`, `line`, `pie`, `doughnut`, and `scatter` payloads in
YAML or JSON. The default wrapper includes screen-only expand and export
controls. Static SVG and its data table remain usable without JavaScript. Use
`charts.WithControlWrapper(false)` for static-only HTML, or
`charts.WithExternalizedControlRuntime(true)` when the host supplies the
reviewed Goshtoso and chart runtimes through Margo’s requirement graph.
One formatted semantic accessible data table follows each chart in HTML. Redundant
chart-owned exact-value disclosures are suppressed so static and interactive
renderers expose the same table surface. Tables are hidden from print by
default. Use `charts.WithPrintableAccessibleData(true)` to include them in
print/PDF output.

Chart styles use Goshtoso tokens by default. A payload can choose a palette,
caller class, or explicit hex colors. `class` and `color` are mutually
exclusive on one series or slice.

## Printable interactive proof of concept

Bar and Line payloads can select Goshtoso Charts' interactive implementation:

```yaml
schemaVersion: 1
type: line
renderer: interactive
title: Weekly revenue
categories: [Mon, Tue, Wed]
series:
  - name: Revenue
    values: [12, 18, 21]
```

Omit `renderer`, or set it to `static`, for the existing server-rendered SVG.
Interactive output keeps the accessible exact-data table and exposes PNG
export. The POC disables initial chart animation so export and print capture a
complete deterministic frame. Standalone HTML relocates provenance-marked chart
initialization into the reviewed requirement graph, preserving Margo's
script-free fragment contract. Chromium PDF export waits for initialization,
requests the chart's PNG export, substitutes that image for print, then prints
the document.

POC limits: only Bar and Line accept `renderer: interactive`; the default
control wrapper is required; per-series `class` is rejected because the
interactive public API cannot preserve it. Palettes, root class, and explicit
series colors remain supported. Pie, doughnut, and scatter remain static.

## Developer renderer

The chart-aware optimistic renderer is a developer tool, not a released CLI.
Run it from the repository root as part of the one root module:

```sh
GOWORK=off GOFLAGS=-mod=readonly \
  go run ./charts/tools/optimistic-renderer \
  --source testdata/markdown/margo-full-feature-set.md \
  --charts-source charts/testdata/markdown/optimistic-charts.md \
  --output /tmp/margo-optimistic-charts.html
```

No `go.work` file or independent chart module is required. The output embeds
the pinned chart-controls runtime for offline inspection. Print CSS hides the
controls and exact-data tables while retaining the chart.
