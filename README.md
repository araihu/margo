# Margo

<p align="center">
  <img src="assets/margo-mascot.png" alt="Margo, a pink Go gopher holding a rendered document in a publishing atelier." width="480">
</p>

Margo turns Markdown into standalone HTML, linked static sites, PDF documents,
and presentation decks.

It is both a consumer-neutral Go document engine and one `margo` command. Margo
owns compilation, semantic rendering, embedded runtime assets, deterministic
site path mapping, PDF engine discovery, and artifact safety. Your application
still owns public URLs, navigation design, storage, and deployment.

## Install the command

Margo requires Go 1.26.5 or newer. A released version installs from one root
tag. Choose either Go installation or a ready-to-run release binary.

### Install with Go

```sh
go install github.com/araihu/margo/cmd/margo@vX.Y.Z
```

### Download a release binary

Starting with `v0.0.3`, each root
[GitHub Release](https://github.com/araihu/margo/releases) includes portable
`margo_VERSION_OS_ARCH` archives for Linux, macOS, and Windows on amd64 and
arm64, plus `checksums.txt` with SHA-256 digests. Unix archives use `.tar.gz`;
Windows archives use `.zip` and contain `margo.exe`.

Download the archive for your platform, verify its entry in `checksums.txt`,
extract it, and place `margo` (or `margo.exe`) on `PATH`. Release binaries use
`CGO_ENABLED=0`: they discover an installed Chromium but do not acquire a
browser or load native WebKit libraries.

The one binary contains the full command surface:

```sh
margo check article.md
margo html article.md --output article.html
margo site docs/ --output-dir public/ --assets local
margo pdf article.md --output article.pdf
margo deck talk.md --output talk.html
margo deck talk.md --format pdf --output talk.pdf
margo doctor
margo version
margo --version
```

Single-document `INPUT` is exactly one file path or `-` for stdin. `html` and
HTML decks default to artifact stdout. PDF output requires an explicit path; `--output -`
allows binary stdout. An existing file is refused unless `--force` is present.
Conversion failures go to stderr and artifacts go to stdout or their
destination. `check` reports findings on stdout. Choose deterministic text or
JSON with `--diagnostics text|json`.

Run `margo check INPUT` before rendering an unfamiliar document. It reports
raw HTML, missing or remote images, incompatible SVG, invalid frontmatter,
legacy Mermaid configuration, empty image alternatives, missing document
language, skipped heading levels, empty link destinations, and relative links.
Every finding includes the source, line, field pointer, and a remediation
hint. Errors make the command exit nonzero; accessibility and link-policy
warnings remain visible without blocking the check.

## Build a static site

`margo site INPUT_DIR --output-dir OUTPUT_DIR` discovers `.md` and `.markdown`
files recursively, preserves their subdirectories, and maps each extension to
`.html`. Relative Markdown links are rewritten to the mapped pages, including
queries and fragments. Missing pages or heading fragments fail with an
actionable diagnostic instead of publishing a broken site.

The output directory must not already exist. Margo builds a sibling staging
directory and publishes it with one rename, so a failed build cannot leave a
partial destination. Every successful build contains `margo-manifest.json`
with sorted paths and exact-byte SHA-256 digests. Text and JSON command output
list every source-to-page mapping and the aggregate manifest identity.

Use `--assets local` (the default) to share Margo runtimes and copy validated
source images once. Use `--assets inline` for self-contained pages with no
separate asset artifacts:

```sh
margo site docs/ --output-dir public/ --assets local
margo site docs/ --output-dir offline/ --assets inline --diagnostics json
```

Source symlinks are not followed. Output and asset collisions fail even when
they differ only by letter case, keeping the result portable across common
filesystems.

## Markdown metadata

Margo accepts YAML frontmatter fields `title`, `description`, `language`,
`slug`, `authors`, `publishedAt`, `modifiedAt`, and `tags`. Dates use RFC 3339;
`authors` and `tags` are lists of strings; `language` uses a BCP 47-style tag.
The HTML title precedence is frontmatter `title`, the first H1, then the source
filename without its Markdown extension. Standalone output defaults the
document language to `en` when `language` is absent.

For one-off publication metadata, `margo html` and `margo pdf` accept
`--title TEXT` and `--lang TAG`; these override frontmatter in the generated
HTML shell and PDF metadata. Run `margo check` first: it warns when language is
missing instead of guessing editorial content.

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
`margo version` and `margo --version` report compiled engine capabilities;
they do not perform external engine discovery.

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

### PDF link policy

PDF rendering strips relative anchor targets by default. This preserves the
visible link text without publishing the renderer's temporary localhost origin
as a dead annotation. Fragment links, `http(s)`, `mailto`, and `tel` targets are
preserved.

Use a public base URL to resolve relative targets before printing:

```sh
margo pdf guide.md --output guide.pdf --base-url https://docs.example.com/manual/
```

`--relative-links strip|error|keep|resolve` makes the policy explicit.
`resolve` requires `--base-url`; supplying `--base-url` alone selects it.
`error` rejects the first relative target. `keep` is an explicit escape hatch
and can expose an engine-local URL in the resulting PDF, so it is unsuitable
for distributed artifacts.

Library consumers set `pdf.Request.RelativeLinks` and `pdf.Request.BaseURL`.
The zero-value policy is also `strip`.

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

Library consumers can run the same read-only preflight with `margo.Check`.
Supply `margo.WithCheckAssetReader` when local asset existence and SVG
compatibility should be checked; without a reader, source-only checks still
run.

### Authorize trusted embeds

Margo denies raw HTML and remote embeds by default. A host that intentionally
needs remote media can register the typed `embed` extension; the document can
then request only an allowed kind and exact HTTPS origin. It cannot supply HTML
or widen the host policy:

```go
import (
	"github.com/araihu/margo"
	"github.com/araihu/margo/embed"
)

trustedEmbeds := embed.Extension(embed.Policy{
	Projection:     embed.ProjectionInteractive,
	AllowedKinds:   []embed.Kind{embed.KindIframe},
	AllowedOrigins: []string{"https://video.example.com"},
	IframeSandbox:  []embed.SandboxToken{embed.SandboxAllowPresentation},
})
compiler := margo.New(margo.WithExtension(trustedEmbeds))
checks, err := margo.Check(ctx, source, margo.WithCheckExtension(trustedEmbeds))
```

The matching Markdown fence contains typed data, not markup:

````markdown
```trusted-embed
kind: iframe
url: https://video.example.com/watch/123
title: Architecture overview
width: 800
height: 450
```
````

The command uses the same model through a trusted operator-selected JSON file:

```sh
margo check article.md --policy margo-policy.json
margo html article.md --policy margo-policy.json --output article.html
margo site docs/ --policy margo-policy.json --output-dir public/
margo pdf article.md --policy margo-policy.json --output article.pdf
margo deck talk.md --policy margo-policy.json --output talk.html
```

See [Host policy and trusted embeds](docs/policy.md) for the policy schema,
target projections, sanitized raw-HTML handshake, CSP obligations, and audit
identity.

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
packages in one Go module. One root tag versions the libraries, `go install`,
and every GoReleaser binary together. The architecture decision is recorded in
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
