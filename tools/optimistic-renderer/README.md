# Optimistic renderer

This developer tool turns one Markdown benchmark source into an offline,
standalone Margo HTML artifact. It is part of the root Margo module and is not
a released CLI surface.

```sh
GOWORK=off GOFLAGS=-mod=readonly \
  go run ./tools/optimistic-renderer \
  --source testdata/markdown/margo-full-feature-set.md \
  --output /tmp/margo-optimistic.html \
  --color-mode light
```

The tool uses the embedded Goshtoso stylesheet and Margo’s modern theme by
default. Use `--color-mode dark` for a dark artifact. Output is written through
a same-directory temporary file and renamed after successful rendering. A failed
render leaves no partial destination.

For a benchmark with optional charts, run the
[chart-aware renderer](../../charts/tools/optimistic-renderer/README.md) from
the repository root with the same `GOWORK=off GOFLAGS=-mod=readonly` prefix.
No `go.work` setup or independent module is required.

Browser and PDF evidence remain separate developer operations. Use the tracked
browser scripts only after their checked environment is available; do not use
ambient package installation, browser downloads, or fallback executables.
