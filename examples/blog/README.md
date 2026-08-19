# Blog example

Generate two blog-style pages and their local assets from Margo's public
contracts:

```sh
GOWORK=off GOFLAGS=-mod=readonly go run ./examples/blog \
  -out examples/blog/generated
```

The command writes:

- `index.html`, composed directly with `RenderHTMLPage`;
- `field-notes.html`, with canonical, Open Graph, X/Twitter, and article
  metadata composed through consumer-owned components;
- `assets/`, containing the embedded stylesheet and image formats used by both
  pages.

For a local development preview:

```sh
python3 -m http.server 8080 --directory examples/blog/generated
```

Open <http://127.0.0.1:8080/> and verify both pages and their images load. This
Python server is for local inspection, not production hosting.

## What the example demonstrates

The publication adapter in `site/publication.go` intentionally belongs to this
example. Routes, SEO metadata, article identity, navigation, and site ownership
remain consumer policy; Margo supplies semantic fragments, dependency
requirements, and page composition seams. The `margo.invalid` URLs are
deliberate placeholders, not deployable metadata.

The checked assets cover AVIF, WebP, JPEG, PNG, and GIF. The article uses an
AVIF/WebP/JPEG `<picture>` hero plus PNG and GIF figures. Unit tests verify file
signatures and HTML references; the tagged browser test decodes every format
with the installed Chromium-family browser.

This directory is an integration example, not a site generator or production
blog template. Change its publication adapter and metadata before reusing the
pattern in a public site.

## Verification

Run the focused example tests from the repository root:

```sh
GOWORK=off GOFLAGS=-mod=readonly go test ./examples/blog/...
```
