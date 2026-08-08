# Optimistic renderer

This developer tool turns one Markdown benchmark source into an offline,
standalone Margo HTML artifact. It uses the embedded Goshtoso stylesheet and
Margo's modern theme by default, and accepts an explicit light or dark color
mode.

```sh
GOWORK=off GOFLAGS=-mod=readonly \
  go run ./tools/optimistic-renderer \
  --source testdata/markdown/margo-full-feature-set.md \
  --output /tmp/margo-v0.0.1-optimistic.html \
  --color-mode light
```

The output is written through a same-directory temporary file, synced, and
renamed only after the component rendered successfully. A failed render leaves
no partial destination and no `.margo-render-*` temporary file.

PDF printing and contrast/browser evidence remain M0-owned operations. Feed
the generated absolute HTML path to `test/browser/run-playwright.sh` with its
checked environment file; do not use ambient npm, browser downloads, or a
fallback executable.
