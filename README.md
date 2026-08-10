# Margo

<p align="center">
  <img src="assets/margo-mascot.png" alt="Margo, a pink Go gopher holding a rendered document in a publishing atelier." width="480">
</p>

Margo is a Go document engine for turning Markdown into reusable, semantic
HTML. It can produce an embeddable article, a complete static page, or
print-ready HTML for an explicitly selected PDF engine.

Use Margo as the rendering layer for documentation sites, blogs, reports,
previews, and export tools. Your application keeps control of routes, URLs,
navigation, site metadata, storage, and deployment.

## What it provides

- Markdown with tables, task lists, footnotes, autolinks, syntax-highlighted
  code, and Mermaid source handling.
- YAML frontmatter for title, description, language, slug, authors, dates, and
  tags.
- Semantic HTML that remains readable before JavaScript runs.
- Embeddable fragments and complete pages from the same compiled document.
- Inline dependencies for self-contained output, or local asset URLs for web
  applications.
- Caller-owned composition points for `<head>` content, headers, content before
  the article, and footers.
- Immutable compile and render results, diagnostics, resource limits, and
  deterministic fingerprints.
- Renderer-neutral PDF contracts without downloading or bundling a browser.

## Quick start

Margo requires Go 1.26.5 or newer.

```sh
go get github.com/araihu/margo@v0.0.2
```

This program writes one self-contained HTML file:

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/araihu/margo"
)

const source = `---
title: Hello, Margo
description: A small standalone document.
---

# Hello, Margo

Markdown goes in. A complete HTML page comes out.

## Included by default

- semantic headings
- readable typography
- print styles
`

func main() {
	ctx := context.Background()
	compiler := margo.New()

	document, err := compiler.Compile(ctx, margo.Source{
		Name:    "hello.md",
		Content: []byte(source),
	})
	if err != nil {
		log.Fatal(err)
	}

	rendered, err := compiler.Render(ctx, document)
	if err != nil {
		log.Fatal(err)
	}

	page, err := margo.RenderStandalone(rendered, margo.WithTableOfContents())
	if err != nil {
		log.Fatal(err)
	}

	output, err := os.Create("hello.html")
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	if err := page.Render(ctx, output); err != nil {
		log.Fatal(err)
	}
}
```

Run it and open `hello.html` in a browser:

```sh
go run .
```

## Choose the output you need

### Embed an article

Project a rendered document to `HTMLResult`, then render its fragment inside an
existing page:

```go
htmlResult, err := margo.RenderHTML(rendered)
if err != nil {
	return err
}
return htmlResult.Fragment().Render(ctx, writer)
```

The fragment contains one `article.margo-document`. It doesn't own the page
shell, navigation, color mode, or site metadata. `HTMLResult` also exposes
normalized metadata, plain text, diagnostics, dependencies, and a fingerprint.

### Build a complete page

`RenderHTMLPage` adds a generic document shell. Use inline dependencies for a
self-contained file:

```go
page, err := margo.RenderHTMLPage(htmlResult, margo.HTMLPageInput{
	Theme:           margo.ThemeModern,
	ColorMode:       margo.ColorModeLight,
	DependencyMode: margo.HTMLDependenciesInline,
})
```

For a web application, choose `HTMLDependenciesLocal`, inspect
`htmlResult.Requirements()`, and serve each requirement at its declared local
URL. The page API also accepts caller-owned `Head`, `Header`, `BeforeContent`,
and `Footer` components, plus a custom theme stylesheet.

```go
page, err := margo.RenderHTMLPage(htmlResult, margo.HTMLPageInput{
	Theme:           siteTheme,
	ColorMode:       margo.ColorModeLight,
	DependencyMode: margo.HTMLDependenciesLocal,
	ThemeStylesheet: siteStyles,
	Head:            siteMetadata(htmlResult.Metadata()),
	Header:          siteNavigation(),
	BeforeContent:   documentContext(),
	Footer:          siteFooter(),
})
```

Mount local handlers at their owning prefixes. The chart handler is needed
only when the chart extension is registered:

```go
mux.Handle("/assets/", goshtosoassets.Handler())
mux.Handle("/margo-assets/", margo.HTMLAssetHandler())
mux.Handle("/charts/assets/", chartassets.Handler())
```

The handlers already understand those prefixes, so don't strip them again.

### Generate static sites

Margo renders pages but doesn't prescribe a site generator. A consumer can map
source files to routes, supply canonical and social metadata through `Head`,
and write the rendered components to any output directory.

The checked [blog example](examples/blog/README.md) does exactly that. It builds
a landing page and an article with a custom theme, public metadata, and AVIF,
WebP, JPEG, PNG, and GIF images.

```sh
GOWORK=off GOFLAGS=-mod=readonly go run ./examples/blog \
  -out examples/blog/generated
```

### Integrate a PDF engine

The optional `pdf` module defines `Engine`, `Request`, `Result`, `EngineInfo`,
`ExportReport`, and validated physical page settings. It doesn't download a
browser or silently choose one. Engine selection and browser installation stay
with the application:

```sh
go get github.com/araihu/margo/pdf@v0.0.2
```

The installed browser version is recorded as runtime evidence, not enforced as
a compatibility requirement. Project gates validate generated HTML in a real
browser; PDF output remains a separate visual-review surface.

## Rendering model

```text
Markdown source
    | Compile
immutable Document
    | Render
immutable RenderResult
    | RenderHTML
HTMLResult
    | Fragment, RenderHTMLPage, or consumer-selected PDF engine
output artifact
```

Compile once and project the result as needed. Margo doesn't infer publication
semantics from a document: a blog post, API guide, invoice, and internal report
all use the same core contracts.

Raw HTML fails closed by default. Hosts that allow it must set an explicit
policy, and the document must request the matching sanitized capability.

## Packages and modules

| Import path | Status | Purpose |
| --- | --- | --- |
| `github.com/araihu/margo` | Released | Compiler, semantic renderer, HTML fragments, complete pages, and standalone output |
| `github.com/araihu/margo/pdf` | Released separately | PDF engine contracts, page configuration, runtime evidence, and platform probes |
| `github.com/araihu/margo/charts` | Optional repository module | Validated chart fences, accessible data, and static SVG output |
| `github.com/araihu/margo/deck` | Reserved | Package boundary for future static deck support |

Margo currently uses Goshtoso for its embedded visual primitives. That is an
implementation dependency; consumers integrate through Margo's public document
and page APIs and don't need prior knowledge of the wider Arai Hû ecosystem.

## Development and verification

Each Go module is tested independently with workspace mode disabled:

```sh
GOWORK=off GOFLAGS=-mod=readonly go test ./...
(cd charts && GOWORK=off GOFLAGS=-mod=readonly go test ./...)
(cd pdf && GOWORK=off GOFLAGS=-mod=readonly go test ./...)
```

Generated HTML also has browser-level checks for semantics, interaction,
images, accessibility behavior, and print preparation. See
[HTML browser evidence](docs/testing/editorial-html.md) and the
[print contrast checker](docs/CONTRAST_LINT.md).

## Security

See [SECURITY.md](SECURITY.md) for private vulnerability reporting.

## License

Margo is licensed under the [MIT License](LICENSE).
