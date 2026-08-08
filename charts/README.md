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

When the wrapper is enabled, its action fieldset and expand modal are hidden by
the chart's print CSS, so browser PDF output contains only the chart and its
accessible data table while screen HTML keeps the controls. For a static-only
HTML input, opt out explicitly. This emits the same SVG and accessible table
without wrapper DOM, export actions, or runtime:

```go
compiler := margo.New(margo.WithExtension(charts.Extension(charts.WithControlWrapper(false))))
```

This checkout is a feature branch (`v0.0.1-dev`). Release provenance and
external publication remain separate integration work.
