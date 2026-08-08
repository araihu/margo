# Chart-aware optimistic renderer

This module-local developer command renders the Margo optimistic benchmark with
the interactive `goshtosochart` extension enabled. The root renderer stays
extension-neutral because `github.com/araihu/margo/charts` is an optional Go
module.

From the repository root, run through the checkout workspace:

```sh
go run ./charts/tools/optimistic-renderer \
  --source testdata/markdown/margo-full-feature-set.md \
  --charts-source charts/testdata/markdown/optimistic-charts.md \
  --output /tmp/margo-v0.0.1-optimistic-charts.html \
  --color-mode light
```

Keep the checkout's `go.work` active for this integration artifact so the
renderer uses the local root module. The independent module gates still use
`GOWORK=off GOFLAGS=-mod=readonly`.

Use `--color-mode dark` for the dark projection. The chart appendix covers bar,
line, doughnut, and scatter, including theme tokens, caller classes, and
explicit hexadecimal colors. The command embeds the pinned chart-controls
runtime in the HTML, so expand/fullscreen/export actions work offline. Print
CSS hides those controls from PDF output while retaining the SVG and accessible
data table.
