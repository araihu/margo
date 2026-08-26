# Margo Charts

`github.com/araihu/margo/charts` adds the `goshtosochart` fenced block to a
Margo compiler. It renders a static SVG and one adjacent accessible data table
by default. The package is optional and belongs to the root Margo Go module.

Register the extension before compiling Markdown that contains a chart:

```go
compiler := margo.New(margo.WithExtension(charts.Extension()))
```

Without that registration, the root compiler fails closed with
`extension.missing_integration`.

## First chart

Version 1 accepts YAML or JSON payloads for `bar`, `line`, `pie`, `doughnut`,
and `scatter` charts:

````markdown
```goshtosochart
schemaVersion: 1
type: bar
title: Weekly revenue
categories: [Mon, Tue, Wed]
series:
  - name: Revenue
    values: [12, 18, 21]
```
````

Static output does not require JavaScript. The default wrapper adds screen-only
expand and export controls; the SVG and exact-data table remain usable when
those controls are unavailable. Use
`charts.WithControlWrapper(false)` when the host needs static-only HTML.

Chart styles use Goshtoso tokens by default. A payload can choose a palette,
caller class, or explicit hex colors. `class` and `color` are mutually
exclusive on one series or slice.

## Interactive charts

Every v1 chart family can opt into Goshtoso Charts' interactive renderer:

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

Omit `renderer`, or set it to `static`, for server-rendered SVG. Interactive
output retains the accessible data table and supports PNG export. Margo disables
initial animation so export and print capture a complete frame.

When the host resolves dependencies through Margo's HTML requirement graph,
register the extension with:

```go
compiler := margo.New(margo.WithExtension(charts.Extension(
	charts.WithExternalizedControlRuntime(true),
)))
```

Standalone output can inline the declared Goshtoso and chart runtimes. A host
using local dependencies must serve the matching asset mounts. Chromium PDF
export waits for interactive initialization, requests PNG output, substitutes
that image for print, then prints the document.

Interactive limits: the default control wrapper is required. Per-series and
per-slice `class` is rejected because the interactive public APIs cannot
preserve it; palettes, root class, and explicit colors remain supported.
Interactive scatter requires exactly one point or value for every declared
category. Use the static renderer when a category contains multiple samples.

Target support is explicit: interactive charts work in standalone HTML, sites,
and standalone PDF (the PDF projection captures a printable raster). The
`margo deck` CLI target is static for both HTML and PDF, so
`margo check --target deck` rejects `renderer: interactive` with
`chart.renderer_target_unsupported`. Omit `renderer` or set it to `static` in a
deck. Library callers that preflight chart fences should register the same
extension with `margo.WithCheckExtension(charts.Extension())`.

## Accessible data in print

One formatted semantic accessible data table follows each chart in HTML.
Redundant chart-owned exact-value disclosures are suppressed so static and
interactive output expose the same table. Print and PDF hide that table by
default. Register
`charts.Extension(charts.WithPrintableAccessibleData(true))` to include it.

## Development-only renderer

The optimistic renderer is a repository development tool, not part of the
released `margo` CLI. Run it from the repository root:

```sh
GOWORK=off GOFLAGS=-mod=readonly \
  go run ./charts/tools/optimistic-renderer \
  --source testdata/markdown/margo-full-feature-set.md \
  --charts-source charts/testdata/markdown/optimistic-charts.md \
  --output /tmp/margo-optimistic-charts.html
```

No `go.work` file or independent chart module is required. The output embeds
the pinned chart-control runtime for offline inspection. Print CSS hides
controls and exact-data tables while retaining the chart.
