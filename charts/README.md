# Margo Charts

`github.com/araihu/margo/charts` registers the `goshtosochart` fence for
server-side, static Goshtoso Charts SVG. The module is opt-in; the root module
continues to report `extension.missing_integration` when the registration is
not supplied.

```go
compiler := margo.New(margo.WithExtension(charts.Extension()))
```

Version 1 accepts `bar`, `line`, `pie`, `doughnut`, and `scatter` payloads in
YAML or JSON. Every rendered chart includes a complete adjacent data table;
controls, export buttons, hydration, and browser runtime are disabled.

This checkout is a feature branch (`v0.0.1-dev`). Release provenance and
external publication remain separate integration work.
