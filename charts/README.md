# Margo Charts

`github.com/araihu/margo/charts` registers the `goshtosochart` fence for
server-side, static Goshtoso Charts SVG. The module is opt-in; the root module
continues to report `extension.missing_integration` when the registration is
not supplied.

Version 1 accepts `bar`, `line`, `pie`, `doughnut`, and `scatter` payloads in
YAML or JSON. Every rendered chart includes a complete adjacent data table.
The HTML control wrapper, expand action, verified SVG/PNG export actions, and
their versioned browser runtime are enabled by default:

```go
compiler := margo.New(margo.WithExtension(charts.Extension()))
```

That default preserves compatibility with consumers pinned to the previously
published Margo root: each enabled wrapper retains its exact upstream control
loader. A consumer compiling against the current editorial root externalizes
the shared runtime into Margo's typed requirement graph instead:

```go
compiler := margo.New(margo.WithExtension(
    charts.Extension(charts.WithExternalizedControlRuntime(true)),
))
```

The declarative path materializes reviewed bytes from the embedded Goshtoso and
Charts handlers without download. It loads Alpine Focus, Goshtoso first-party
behavior, Alpine, and the chart controls once in that order. Mount the resulting
local URLs at `/assets/` and `/charts/assets/`, or let
`margo.HTMLDependenciesInline` embed the same bytes.

When the wrapper is enabled, its action fieldset and expand modal are hidden by
the chart's print CSS, so browser PDF output contains only the chart and its
accessible data table while screen HTML keeps the controls. For a static-only
HTML input, opt out explicitly. This emits the same SVG and accessible table
without wrapper DOM, export actions, or runtime:

```go
compiler := margo.New(margo.WithExtension(charts.Extension(charts.WithControlWrapper(false))))
```

JavaScript is progressive enhancement: static SVGs and every adjacent
accessible data table remain in initial HTML when JavaScript is disabled. The
expand/export controls remain inert or hidden until their declared runtime is
available.

Chart appearance follows Goshtoso tokens by default. The optional `style`
object selects a token palette and/or adds a caller class; `style.colors` sets
explicit hex colors by series index. A series or pie slice can also provide its
own `class` or `color`:

```yaml
schemaVersion: 1
type: line
title: Revenue
style:
  palette: auto       # auto, araihu, bold, neutral, pastel, status
  class: finance-chart
  colors: ["", "#2563eb"]
categories: [Q1, Q2]
series:
  - name: Revenue
    class: revenue-series
    values: [12, 18]
  - name: Cost
    color: "#dc2626"
    values: [7, 9]
```

Resolution precedence, from fallback to strongest override: Goshtoso theme
tokens, CSS supplied by `style.class` or a series/slice class, then explicit
hex `color`/`style.colors`. `class` and `color` are mutually exclusive on one
series or slice. Blank entries in `style.colors` keep the corresponding theme
token.

For a human benchmark that combines the root corpus with the chart appendix,
use the module-local renderer. It enables the HTML chart controls and embeds
the pinned controls runtime in the output, so the file works without a server
or external requests:

```sh
go run ./charts/tools/optimistic-renderer \
  --source testdata/markdown/margo-full-feature-set.md \
  --charts-source charts/testdata/markdown/optimistic-charts.md \
  --output /tmp/margo-v0.0.1-optimistic-charts.html
```
Run this integration command from the repository root with the checkout's
`go.work` active so the renderer uses the local root module. Keep the
`GOWORK=off GOFLAGS=-mod=readonly` prefix for the independent module gates.
The generated HTML keeps the screen controls and print CSS behavior shown
above. The PDF path hides the controls while retaining the chart and its
accessible data table.

This checkout is a feature branch (`v0.0.1-dev`). Release provenance and
external publication remain separate integration work.
