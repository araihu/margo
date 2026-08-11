# Margo Charts

`github.com/araihu/margo/charts` is an optional package in the root Margo Go
module. It registers the `goshtosochart` fence for static Goshtoso Charts SVG
and accessible adjacent data tables. The root compiler reports
`extension.missing_integration` until a consumer registers the extension.

```go
compiler := margo.New(margo.WithExtension(charts.Extension()))
```

Version 1 accepts `bar`, `line`, `pie`, `doughnut`, and `scatter` payloads in
YAML or JSON. The default wrapper includes screen-only expand and export
controls. Static SVG and its data table remain usable without JavaScript. Use
`charts.WithControlWrapper(false)` for static-only HTML, or
`charts.WithExternalizedControlRuntime(true)` when the host supplies the
reviewed control runtime through Margo’s requirement graph.

Chart styles use Goshtoso tokens by default. A payload can choose a palette,
caller class, or explicit hex colors. `class` and `color` are mutually
exclusive on one series or slice.

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
controls while retaining the chart and accessible data table.
