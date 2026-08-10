# Margo

<p align="center">
  <img src="assets/margo-mascot.png" alt="Margo, a pink Go gopher holding a rendered document in a publishing atelier." width="480">
</p>

Margo turns Markdown into standalone HTML, PDF documents, and presentation decks.

It is both a consumer-neutral Go document engine and one `margo` command. Margo
owns compilation, semantic rendering, embedded runtime assets, PDF engine
discovery, and artifact safety. Your application still owns routes, URLs,
navigation, publication metadata, storage, and deployment.

## Install the command

Margo requires Go 1.26.5 or newer. A released version installs from one root
tag:

```sh
go install github.com/araihu/margo/cmd/margo@vX.Y.Z
```

The one binary contains the full command surface:

```sh
margo html article.md --output article.html
margo pdf article.md --output article.pdf
margo deck talk.md --output talk.html
margo deck talk.md --format pdf --output talk.pdf
margo doctor
margo version
```

`INPUT` is exactly one file path or `-` for stdin. `html` and HTML decks
default to artifact stdout. PDF output requires an explicit path; `--output -`
allows binary stdout. An existing file is refused unless `--force` is present.
Diagnostics go to stderr and artifacts go to stdout or their destination;
choose deterministic text or JSON with `--diagnostics text|json`.

## PDF engines

Use `--engine auto|chromium|native` and, for unusual installations or
containers, `--engine-path PATH`. Auto discovery checks:

1. the explicit `--engine-path`;
2. `MARGO_CHROMIUM_PATH`;
3. Chromium-family executables on `PATH` and known platform locations;
4. a native engine compiled into that platform build.

Discovery can move past an unavailable candidate. Once an engine is selected,
rendering never falls back: a failed Chromium render cannot silently become a
different native PDF. `margo doctor --diagnostics json` reports the exact
candidate order, path, version, compiled state, availability, and reason.

Margo never downloads a browser, native runtime, helper binary, or package at
runtime. It uses installed Chromium when available. Native WKWebView, WebView2,
and opt-in WebKitGTK slots are capability-gated and currently report
`pdf.native.compiled_out` until their matching platform runners provide
verifiable backend evidence. If no usable engine exists, PDF commands fail
before publishing an output.

Portable Linux and musl builds use `CGO_ENABLED=0`; they support installed
Chromium and intentionally contain no native WebKitGTK dependency. This keeps
the default binary suitable for minimal containers. A future WebKitGTK build
is a distinct CGO-enabled artifact with declared dynamic libraries, not an
implicit dependency of the portable binary.

The [PDF engine matrix](docs/testing/pdf-engine-matrix.md) records versions
actually tested. Those observations are evidence, not a policy that rejects
other Chromium versions.

## Use the Go engine

All packages share the same module version and root release:

```sh
go get github.com/araihu/margo@vX.Y.Z
```

This program writes one standalone HTML page:

```go
package main

import (
	"context"
	"os"

	"github.com/araihu/margo"
)

func main() {
	ctx := context.Background()
	compiler := margo.New()
	document, err := compiler.Compile(ctx, margo.Source{
		Name:    "hello.md",
		Content: []byte("# Hello, Margo\n\nMarkdown goes in. HTML comes out.\n"),
	})
	if err != nil {
		panic(err)
	}
	rendered, err := compiler.Render(ctx, document)
	if err != nil {
		panic(err)
	}
	page, err := margo.RenderStandalone(rendered, margo.WithTableOfContents())
	if err != nil {
		panic(err)
	}
	output, err := os.Create("hello.html")
	if err != nil {
		panic(err)
	}
	defer output.Close()
	if err := page.Render(ctx, output); err != nil {
		panic(err)
	}
}
```

### Opt into charts

The CLI registers charts. Library consumers choose them explicitly:

```go
import (
	"github.com/araihu/margo"
	"github.com/araihu/margo/charts"
)

compiler := margo.New(margo.WithExtension(charts.Extension()))
```

Charts produce static SVG plus an accessible data table. Interactive controls
can be externalized through the requirement graph when a host needs them.

### Embed a fragment or compose a page

`RenderHTML` projects a rendered document to an `HTMLResult`. Its fragment is
one semantic `article.margo-document` and does not claim a blog, editorial, or
publication domain:

```go
htmlResult, err := margo.RenderHTML(rendered)
if err != nil {
	return err
}
return htmlResult.Fragment().Render(ctx, writer)
```

`RenderHTMLPage` provides a generic page shell. `HTMLPageInput` exposes
caller-owned composition points and dependency policy:

```go
page, err := margo.RenderHTMLPage(htmlResult, margo.HTMLPageInput{
	Theme:           margo.ThemeModern,
	ColorMode:       margo.ColorModeLight,
	DependencyMode: margo.HTMLDependenciesLocal,
	Head:            siteMetadata(),
	Header:          siteNavigation(),
	BeforeContent:   documentContext(),
	Footer:          siteFooter(),
})
```

Use `HTMLDependenciesInline` for a self-contained page or
`HTMLDependenciesLocal` when an application serves reviewed assets. Relevant
handlers include `HTMLAssetHandler` and these owning prefixes:

```go
mux.Handle("/assets/", goshtosoassets.Handler())
mux.Handle("/margo-assets/", margo.HTMLAssetHandler())
mux.Handle("/charts/assets/", chartassets.Handler())
```

The handlers already understand `/assets/`, `/margo-assets/`, and
`/charts/assets/`; do not strip those prefixes again.

## Rendering model

```text
Markdown source
    | Compile
immutable Document
    | Render
immutable RenderResult
    | project
HTML fragment, standalone page, deck, or renderer-neutral PDF request
    | publish complete artifact
stdout or atomic file destination
```

Compile and render results are immutable. Runtime descriptors bind Mermaid
work to a document and render instance. Standalone HTML embeds the locked
Mermaid browser bytes and exposes a terminal `margoRuntimeReady` promise; PDF
engines wait for the same work before printing. Raw HTML fails closed unless a
host explicitly enables the matching sanitized policy.

Margo can be the rendering engine inside a static-site generator, preview
service, report system, or export tool. Site crawling, route generation,
canonical URLs, feeds, and deployment remain consumer concerns. The checked
[blog example](examples/blog/README.md) demonstrates that boundary.

## One module and release

`github.com/araihu/margo`, `/charts`, `/deck`, `/pdf`, and `/cmd/margo` are
packages in one Go module. One future root tag versions the libraries and
binary together. The architecture decision is recorded in
[ADR 0001](docs/decisions/0001-unified-module-and-cli.md).

The historical submodule tags are not rewritten. Consumers that previously
required `github.com/araihu/margo/pdf`, `github.com/araihu/margo/charts`, or
`github.com/araihu/margo/cmd/margo` as independently versioned modules must
remove those old requirements and select a root Margo release. Otherwise an
old nested module can shadow the package supplied by the root module.

## Development and verification

Run the repository as one module:

```sh
GOWORK=off GOFLAGS=-mod=readonly go test -race ./...
GOWORK=off GOFLAGS=-mod=readonly go vet ./...
CGO_ENABLED=0 GOWORK=off GOFLAGS=-mod=readonly go build ./cmd/margo
```

Tests cover command process boundaries, atomic publication, generated HTML in
an installed browser, charts, popular image formats, Mermaid completion, deck
navigation, print CSS, PDF structure, and selected-engine provenance. PDF
visual quality remains a human review gate.

## Security

See [SECURITY.md](SECURITY.md) for private vulnerability reporting.

## License

Margo is licensed under the [MIT License](LICENSE).
