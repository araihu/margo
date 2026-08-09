# Blog example

This example generates two blog-style pages from Margo's public contracts:

- `index.html` uses `RenderHTMLPage` without public-web metadata;
- `field-notes.html` adds canonical, Open Graph, X/Twitter, and article metadata through consumer-local components passed to Margo's generic page seams.

The publication adapter in `site/publication.go` intentionally belongs to this
example. It demonstrates flexibility without making routes, SEO, article
identity, or site ownership part of Margo's API.

The checked source assets cover AVIF, WebP, JPEG, PNG, and GIF. The article uses an AVIF/WebP/JPEG `<picture>` hero plus PNG and GIF figures. Unit tests verify file signatures and HTML references; the tagged browser test decodes every format with the installed Chromium-family browser.

Generate the local preview:

```sh
GOWORK=off GOFLAGS=-mod=readonly go run ./examples/blog -out examples/blog/generated
```

Then serve the generated directory with any static file server. For example:

```sh
python3 -m http.server 8080 --directory examples/blog/generated
```
