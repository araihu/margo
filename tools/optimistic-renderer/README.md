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

Regenerate reproducible browser evidence with the tracked runner. It resolves
Playwright and css-tree only from the checked local installation, routes the
Mermaid module to the pinned local asset set, denies every other network
request, and binds evidence to the current HTML bytes and SHA-256:

```sh
. test/browser/.cache/node-env.checked.sh
cd test/browser
PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 PLAYWRIGHT_BROWSERS_PATH=0 \
  "$MARGO_NODE_BIN" generate-evidence.mjs \
  --root /absolute/path/to/repository \
  --html /absolute/path/to/output/html/margo-v0.0.1-optimistic.html \
  --mode light \
  --evidence /absolute/path/to/output/evidence/margo-v0.0.1-optimistic.json
```

Run again with `--mode dark` and the dark HTML for dark evidence. The runner
verifies Mermaid screen rendering, print TOC preparation, source disclosure
state, shared document/header/footer surfaces, and zero blocked requests or
console/page errors before writing evidence atomically.

For a human PDF artifact, use the checked Chromium executable and the local
Playwright installation after the M0 environment has been verified:

```sh
. test/browser/.cache/node-env.checked.sh
cd test/browser
PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 PLAYWRIGHT_BROWSERS_PATH=0 \
  "$MARGO_NODE_BIN" print-pdf.mjs \
  --html /absolute/path/to/output/html/margo-v0.0.1-optimistic.html \
  --output /absolute/path/to/output/pdf/margo-v0.0.1-optimistic.pdf \
  --mode light \
  --evidence /absolute/path/to/output/evidence/margo-v0.0.1-optimistic-pdf.json
```

The command blocks non-local requests, waits for fonts and images, emulates
print, calls `window.margoPreparePrintTOC()` before `page.pdf()`, and fails on
network or console errors. This explicit preparation is required for protected
lists, tables, Mermaid blocks, TOC fallback columns, and controlled page
continuation to use the same layout contract as the browser tests.
