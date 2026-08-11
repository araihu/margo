# Chart-aware optimistic renderer

This developer command renders the Margo optimistic benchmark with the optional
`goshtosochart` extension enabled. It is part of the root Margo module, not an
independent module or release artifact.

From the repository root:

```sh
GOWORK=off GOFLAGS=-mod=readonly \
  go run ./charts/tools/optimistic-renderer \
  --source testdata/markdown/margo-full-feature-set.md \
  --charts-source charts/testdata/markdown/optimistic-charts.md \
  --output /tmp/margo-optimistic-charts.html \
  --color-mode light
```

Use `--color-mode dark` for the dark projection. The chart appendix covers bar,
line, doughnut, and scatter charts, including token palettes, caller classes,
and explicit hexadecimal colors. Generated HTML embeds the pinned control
runtime, so `Expand` and capability-derived `Export` controls work offline.
Print CSS hides controls while preserving SVG and the accessible data table.

Run this command with `GOWORK=off GOFLAGS=-mod=readonly`. No `go.work` setup,
nested `go.mod`, or `v0.0.1-dev` release claim applies.
