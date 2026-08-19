# Margo / Marpit visual comparison

`make-compare.mjs` builds a standalone review page from two directories of
`slide-NN.png` captures. It emits:

- `index.html`: paired captures, category index, search/filter, local review
  state keyed to the corpus/manifests, Margo-only/Marpit-only views and a
  paired 1:1 capture dialog;
- `comparison-manifest.json`: corpus, renderer, viewport, geometry, timestamp,
  SHA-256 manifests, and expected/observed font/runtime evidence.

Example:

```sh
node tools/marpit-compare/make-compare.mjs \
  --output /tmp/margo-pdf-compare \
  --margo /tmp/margo-pdf-compare/margo \
  --marpit /tmp/margo-pdf-compare/marpit \
  --corpus deck/testdata/compatibility.md \
  --viewport '1280×720' \
  --pdf-geometry '1280×720 CSS px' \
  --captured-at 2026-08-19T17:20:00Z \
  --font-bundle-digest da8c0b01236c31348701ed36c257ceed2c02898c0811adf4aac01e2b1ad4c8c0 \
  --font-bundle-observed-digest da8c0b01236c31348701ed36c257ceed2c02898c0811adf4aac01e2b1ad4c8c0 \
  --font-checks 6
```

Serve the output with any static server. The page has no external runtime
dependencies.
