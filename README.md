# Margo

<p align="center">
  <img src="assets/margo-mascot.png" alt="Margo, a pink Go gopher holding a rendered document in a publishing atelier." width="480">
</p>

Margo turns Markdown into standalone HTML, linked static sites, PDF documents,
and versioned Margo Marpit-compatible presentation decks. Use one `margo` command from a terminal
or the root Go module from an application. Applications retain ownership of
URLs, navigation, storage, and deployment.

## Install

Margo requires Go 1.27.0 or newer:

```sh
go install github.com/araihu/margo/cmd/margo@latest
# Or pin a root release:
go install github.com/araihu/margo/cmd/margo@vX.Y.Z
```

Starting with `v0.0.3`, each root
[GitHub Release](https://github.com/araihu/margo/releases) provides prebuilt
archives for Linux, macOS, and Windows on amd64 and arm64. Download the archive
for your platform, verify it against `checksums.txt`, extract it, and place
`margo` or `margo.exe` on `PATH`.

Release binaries use `CGO_ENABLED=0`. They discover an installed Chromium when
PDF output needs it; they do not download a browser or load native WebKit
libraries.

## Start with the CLI

Create `docs/index.md`:

```markdown
---
title: My first Margo site
language: en
---

# My first Margo site

Edit this file and watch the browser reload.
```

Start the development server:

```sh
margo serve ./docs --open
```

Margo recursively discovers Markdown, builds the site in memory, chooses an
available local port, and reloads connected browsers after successful changes.
The development server is not for production.

Build the same tree for publication when it is ready:

```sh
margo site ./docs --output-dir ./dist
```

The destination must not already exist. A successful build writes linked HTML
pages, local assets, and `margo-manifest.json` to `dist`.

For one document instead of a site:

```sh
margo check docs/index.md
margo html docs/index.md --output index.html
margo pdf docs/index.md --output document.pdf
```

`html` produces a standalone page. PDF output requires a supported installed
Chromium executable; `margo doctor` reports available engines.

## Start with the Go library

Install the root module:

```sh
go get github.com/araihu/margo@latest
```

Compile Markdown, render it, and write a standalone HTML page:

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/araihu/margo"
)

func main() {
	ctx := context.Background()
	compiler := margo.New()

	document, err := compiler.Compile(ctx, margo.Source{
		Name:    "hello.md",
		Content: []byte("---\ntitle: Hello\nlanguage: en\n---\n\n# Hello\n"),
	})
	if err != nil {
		log.Fatal(err)
	}

	rendered, err := compiler.Render(ctx, document)
	if err != nil {
		log.Fatal(err)
	}

	page, err := margo.RenderStandalone(rendered)
	if err != nil {
		log.Fatal(err)
	}
	if err := page.Render(ctx, os.Stdout); err != nil {
		log.Fatal(err)
	}
}
```

Use `margo.Check` for the same preflight available through `margo check`.
Supply `margo.WithCheckAssetReader` when checks must read local assets. Charts
remain opt-in for library consumers:

```go
compiler := margo.New(margo.WithExtension(charts.Extension()))
```

## Build static sites

`margo site` and `margo serve` accept either a Markdown directory or a site
configuration file.

### Use the defaults

Point either command at a directory tree to discover `.md` and `.markdown`
files recursively:

```sh
margo serve ./docs
margo site ./docs --output-dir ./dist
```

Directory builds map source files to `.html`, validate Markdown links and
fragments, and use local assets by default. Use `--assets inline` when each
generated page should embed its dependencies.

### Configure a publication

Add `site.yaml` when the site needs explicit identity, navigation, themes,
layout composition, locales, a base URL, or a base path. `serve` automatically
uses `site.yaml` from its input directory. Name the config explicitly with
`site`:

```sh
margo serve .
margo serve ./site.yaml --open
margo site ./site.yaml
```

Configured sites take their source, output, asset, and publication settings
from the file and also produce `sitemap.xml` and `llms.txt`. The configured
output defaults to `dist` when omitted. See
[`showcase.yaml`](showcase.yaml) for a complete configuration.

Site builds project `authors`, `publishedAt`, `modifiedAt`, and `tags` into
semantic article metadata and into each page record in the site report and
`margo-manifest.json`. Archive, tag, RSS, and Atom pages remain consumer-owned;
the deterministic route records are the input for those indexes.

### Semantic layouts and documentation families

Configured sites can opt into semantic page layouts and documentation families.
The site selects one trusted layout kind: `article`, `landing`, or `docs`.
Page content never names a raw frame, shell, executable command, or Go module.
Only `docs` owns navigation chrome, search, sidebars, tables of contents,
pagination, and documentation families.

The configuration shape is:

```yaml
layout:
  kind: docs
  default:
    families: [module, cli]
    sidebar: true
    toc: true
    content:
      layout: article
  values:
    family: default
```

`layout.default` declares site-only defaults for the selected kind.
`layout.values` applies the site-level override patch. A directory can select a
declared docs family in `_layout.yaml`:

```yaml
values:
  family: module
```

The reserved `_layout.yaml` file is discovered from the source root through the
page's nearest directory and is never published. A Markdown page can apply the
final patch through top-level frontmatter:

```yaml
---
layout:
  kind: landing
---
```

Resolution order is site defaults, directory patches from root to nearest, then
Markdown frontmatter. Within one kind, maps merge recursively, scalars replace,
and arrays replace completely. Changing `kind` creates a typed boundary: values
from another kind do not cross it. Unknown kinds, properties, values, or docs
families fail in preflight before artifacts are emitted.

Docs families are declared centrally by `layout.default.families`. Directory
and Markdown patches can select a family but cannot declare one. `default`
always exists, family order controls secondary navigation, and non-default
families must own at least one docs page. Landing and article pages have no
family identity.

The `landing` layout is for a conversion-oriented page: it has no sidebar,
table of contents, breadcrumbs, pagination, or page-action toolbar. The `docs`
layout provides family-local navigation, document context, and scoped
pagination when neighbors exist. The Margo showcase publishes Tour at `/`,
Module at `/module/`, the CLI overview at `/cli/`, and one CLI command page
under `/cli/COMMAND/` for each documented command. Static artifacts remain
directory `index.html` files, while public links, canonicals, search, family navigation, sitemap, `llms.txt`,
and rewritten Markdown use the directory routes, including base-path and locale prefixes.
The former root feature pages are retired. Retired Tour feature routes
return HTTP 404, produce no artifacts, and have no redirect or hidden
compatibility page.

Sites without `layout` retain existing top-level `frame` or `shell` behavior.
Existing `componentdocshell` consumers remain supported. Typed `layout` is
mutually exclusive with those top-level presentation authorities.

### Add page actions

Pages can opt into source and download controls through frontmatter:

```yaml
margo:
  actions:
    markdown: true
    pdf: true
```

`markdown: true` retains the source beside the generated page and adds Copy
page and View as Markdown actions. `pdf: true` also publishes a pre-rendered
PDF and adds Download PDF. PDF publication implies Markdown retention.

Use the browser's current page instead of publishing a PDF artifact:

```yaml
margo:
  actions:
    pdf: client
```

Client printing follows the active site theme. Pre-rendered PDFs use the
materialized document brand and stay independent of the surrounding site
shell. Generated actions refer only to same-site artifacts.

### Development server behavior

`margo serve [INPUT_DIR|CONFIG] [--host HOST] [--port PORT] [--open]` builds,
watches, and serves a site from memory with live reload. With no input it uses
the current directory. A directory containing `site.yaml` uses that config;
another directory uses Margo's default linked-site output. Explicit config
files must end in `.yaml` or `.yml`.

The server binds `127.0.0.1` by default. Without `--port`, it tries
8080, 8000, 3000, 1313, and 4000 before asking the operating system for any
available port. After the preferred list is exhausted, the
operating system selects any available port. An explicit port is strict and
fails when it cannot be bound. `--open` opens the chosen URL in the default
browser.

Margo watches configured Markdown, YAML, CSS, images, and other local assets
recursively. Conventional sibling directories used for build outputs, logs,
reports, and temporary artifacts (including `build/`) are ignored for
configured sites. A successful rebuild atomically replaces the in-memory
snapshot and reloads connected browsers. A failed rebuild prints diagnostics
and keeps serving the last successful site. The configured output directory is
excluded from watching and is never written by `serve`.

The development server has no TLS, authentication, authorization, rate
limiting, or deployment contract. Binding a non-loopback `--host` exposes the
content and prints a warning. Do not use it in production.

## Reference

- Run `margo --help` or `margo help COMMAND` for current CLI help.
- Read [Host policy and natural iframe embeds](docs/policy.md) before enabling
  privileged raw HTML or iframe embeds.
- Use the generated [policy schema](docs/reference/policy.md) and
  [document metadata schema](docs/reference/document-metadata.md) as exact
  configuration references.
- See the [PDF engine matrix](docs/testing/pdf-engine-matrix.md) for recorded
  browser test evidence. It is not a supported-version policy.
- Read [ADR 0001](docs/decisions/0001-unified-module-and-cli.md) for the
  unified-module decision and native-PDF boundary.

## CLI commands

```text
margo check INPUT [--target html|site|pdf|deck]
margo html INPUT [--output PATH|-] [--force]
margo site INPUT_DIR|CONFIG [--output-dir OUTPUT_DIR] [--assets local|inline]
margo serve [INPUT_DIR|CONFIG] [--host HOST] [--port PORT] [--open]
margo pdf INPUT --output PATH|- [PDF flags]
margo deck INPUT [--format html|pdf] [--output PATH|-] [PDF flags]
margo doctor
margo version
margo --version
margo help [command]
margo completion SHELL [--no-descriptions]
margo schema policy
margo schema document
```

`INPUT` for `check`, `html`, `pdf`, and `deck` is a path or `-` for stdin.
`site` takes a directory or YAML configuration, not stdin. Commands write
artifacts and command reports to stdout; errors and diagnostics go to stderr.
`--diagnostics text` is the default, while `--diagnostics json` selects JSON.
Errors exit nonzero. Warnings remain visible without blocking `check`.

`html` writes to stdout by default. `deck` defaults to HTML on stdout. `pdf`
and `deck --format pdf` require `--output PATH` or `--output -`. `html`, `pdf`,
and `deck` refuse to replace existing output unless `--force` is present.
Directory-based `site` builds require `--output-dir`; configured builds use the
config's output when the flag is omitted. Site output must not already exist.

All rendering commands accept `--policy FILE` for a trusted host policy.
Ordinary Markdown, local images, Mermaid, tables, and code need no policy.
`html` and `pdf` also accept `--title TEXT` and `--lang TAG`. `margo schema
policy` and `margo schema document` emit the exact embedded Draft 2020-12
schema bytes for the installed Margo version.

### Check

`margo check INPUT [--target html|site|pdf|deck] [--policy FILE]
[--diagnostics text|json]` checks Markdown compatibility without rendering.
The target defaults to HTML. It reports raw HTML, unavailable images,
incompatible SVG, invalid frontmatter, legacy Mermaid configuration, empty
image alternatives, missing document language, skipped headings, empty links,
and relative links for standalone targets. With `--target site`, ordinary
relative Markdown links are left to the multi-page site build, which resolves
and validates them after indexing all source documents. Findings identify the
source, line, field pointer, and a remediation hint.

### HTML

`margo html INPUT [--output PATH|-] [--force] [--title TEXT] [--lang TAG]
[--policy FILE] [--diagnostics text|json]` renders one standalone HTML page.
The output default is `-`.

### Site

`margo site INPUT_DIR|CONFIG [--output-dir OUTPUT_DIR] [--assets local|inline]
[--policy FILE] [--diagnostics text|json]` builds a linked site. Output is
staged beside the destination and published by rename only after a successful
build.

### PDF

`margo pdf INPUT --output PATH|- [--force] [--engine auto|chromium|native]
[--engine-path PATH] [--page-size A4|Letter] [--orientation portrait|landscape]
[--margin-top MM] [--margin-right MM] [--margin-bottom MM] [--margin-left MM]
[--image-overflow limit|allow]
[--relative-links strip|error|keep|resolve] [--base-url URL] [--title TEXT]
[--lang TAG] [--print-chart-data] [--policy FILE]
[--diagnostics text|json]` renders a PDF.

Defaults are `--engine auto`, A4 portrait, and readable document margins of 24 mm top,
22 mm right, 26 mm bottom, and 22 mm left unless `margo.page` supplies a page
preference. Explicit flags override document preferences. To remove all
margins, set all four margin flags to `0` for full bleed.
`--image-overflow limit` is the default;
`--image-overflow allow` permits images to exceed the printable content box.

The default `--relative-links strip` keeps visible text while removing relative
PDF link targets. `--base-url URL` selects `resolve` unless
`--relative-links` was set; explicit `resolve` requires `--base-url`.

All v1 chart families (`bar`, `line`, `pie`, `doughnut`, and `scatter`) accept
`renderer: interactive`; omitting it preserves static SVG. Interactive scatter
accepts one point or value per declared category. Multi-sample scatter remains
available through the static renderer. One formatted semantic exact-data table
follows each chart in HTML. Those tables are omitted from PDF by default;
`--print-chart-data` includes them.

For corporate PDF branding, use a configured site with `site.name`, a local SVG
`site.logo`, and `margo.actions.pdf: true`; the resulting pre-rendered page PDF
uses that name and logo. The complete, copyable configuration and the boundary
between pre-rendered PDFs, browser printing, and standalone `margo pdf` are in
the [`margo pdf` branding guide](showcase/content/cli/pdf/index.md#corporate-branding).

Current releases use installed Chromium. `auto` tries an explicit
`--engine-path`, `MARGO_CHROMIUM_PATH`, discovered Chromium-family executables,
then a native slot. Native backends are compiled out, so selecting
`--engine native` does not make WKWebView, WebView2, or WebKitGTK available.
Margo never downloads a browser. A selected engine that fails does not fall
back to another engine.

Page geometry can also be declared in document metadata:

```yaml
margo:
  page:
    size: Letter
    orientation: landscape
    margins:
      top: 12
      right: 0
      bottom: 12
      left: 0
```

Margin sides are independent. Omit a side to retain its target default or set
it to `0` for full bleed on that edge.

### Deck

`margo deck INPUT [--format html|pdf] [--output PATH|-] [--force]
[--engine auto|chromium|native] [--engine-path PATH] [--page-size A4|Letter]
[--orientation portrait|landscape] [--margin-top MM] [--margin-right MM]
[--margin-bottom MM] [--margin-left MM] [--image-overflow limit|allow]
[--slide-size 16:9|4:3|custom] [--slide-width N --slide-height N]
[--slide-unit px|mm|cm|in|pt|pc|Q] [--print-chart-data] [--policy FILE]
[--diagnostics text|json]` renders the versioned Margo
Marpit-compatible v0.0.1 deck profile.

Its defaults are HTML to stdout, `--engine auto`, A4, portrait, and zero
margins. PDF decks require `--format pdf` and an explicit output path or `-`.
PDF deck links use the same default `strip` policy as `margo pdf`.
`--image-overflow limit` is the default for PDF decks.
Deck PDF validation requires an installed Chromium-compatible engine; selecting
`--engine native` fails with `cli.deck_validator_unavailable` instead of
claiming visual validation.

Deck authoring accepts YAML frontmatter, top-level CommonMark thematic breaks,
heading-divider pagination, local/spot directives, presenter-note comments,
the built-in `modern`, `goshtoso`, and `minimal` themes, and the closed
`columns`, `sidebar`, `compare`, `metrics`, `timeline`, and `demo` layout
catalog. Mermaid, tables, code, images, and supported Goshtoso charts keep the
same accessible extension projections used by the other Margo targets. Layout
classes and slot names are validated before rendering; arbitrary HTML/CSS,
remote backgrounds, custom Marpit themes, and unregistered extension ID
allocators are rejected with diagnostics.

See the [`margo deck` structural-layout guide](showcase/content/cli/deck/index.md#structural-layouts)
for a complete copyable deck covering every layout, exact slot cardinalities,
presenter-note scope, and recovery guidance.

`--slide-size 16:9` selects a 1280x720 logical canvas and `--slide-size 4:3`
selects 960x720. For custom geometry, pass positive dimensions with
`--slide-width`, `--slide-height`, and `--slide-unit`; slide geometry cannot be
combined with the legacy document `--page-size` or `--orientation` flags.
HTML uses a responsive visual stage while retaining logical coordinates. PDF
decks compare every page MediaBox edge and page count against the selected
canvas before publication.

### Doctor, version, and completion

`margo doctor [--diagnostics text|json]` reports PDF engine candidates and
reasons. `margo version` and `margo --version` print the version and compiled
engine capabilities without probing external engines. `margo completion SHELL
[--no-descriptions]` prints completion for `bash`, `zsh`, `fish`, or
`powershell`. `SHELL` is `bash`, `zsh`, `fish`, or `powershell`; the
`--no-descriptions` flag applies to all four generators.

## Go packages

Every supported package ships in the root module:

| Import path | Purpose | Primary entrypoint |
| --- | --- | --- |
| `github.com/araihu/margo` | Compile Markdown and project rendered documents to HTML. | `margo.New`, then `Compile` and `Render` |
| `github.com/araihu/margo/assets` | Serve and inspect embedded Muamba runtime assets. | `assets.MuambaHTTPHandler` |
| `github.com/araihu/margo/charts` | Register optional static and printable interactive Goshtoso chart fences. | `charts.Extension` |
| `github.com/araihu/margo/deck` | Parse and render accessible HTML presentation decks. | `deck.Render` |
| `github.com/araihu/margo/pdf` | Define PDF engine, request, page, and link-policy contracts. | `pdf.Engine.Export` |
| `github.com/araihu/margo/pdf/chromium` | Export Margo HTML through an explicitly selected installed Chromium executable. | `chromium.New` |
| `github.com/araihu/margo/pdf/engines` | Discover and select PDF engine candidates. | `engines.Discover` |
| `github.com/araihu/margo/pdf/native` | Expose the stable platform-native capability boundary. | `native.Probe` |
| `github.com/araihu/margo/pdf/platform` | Verify locked platform probe contracts for native-engine work. | `platform.Bootstrap` |
| `github.com/araihu/margo/ssg` | Define and validate layout-neutral frame, shell, composition, binding, and resource contracts. | `ssg.ResolveComposition` |
| `github.com/araihu/margo/site` | Build deterministic multi-page HTML sites from caller-supplied site-relative sources or a validated config. | `site.Build`, `site.LoadConfig`, `site.BuildConfig` |

`cmd/margo` is the CLI program, not a library API. `internal/...` packages are
unsupported implementation details. `profiles/`, `tools/optimistic-renderer`,
and `charts/tools/optimistic-renderer` are test and developer tools, not release
surface. Directory discovery and publication are CLI-only; `site.Build`
receives caller-supplied `[]site.Source` values.

`margo.RenderHTML` projects a rendered document to an `HTMLResult` containing
one semantic `article.margo-document`. `margo.RenderHTMLPage` provides a
generic page shell without claiming a publication domain. `HTMLPageInput`
leaves composition and dependency choices with the host:

```go
page, err := margo.RenderHTMLPage(htmlResult, margo.HTMLPageInput{
	DependencyMode: margo.HTMLDependenciesLocal,
	Head:           siteMetadata(),
	Header:         siteNavigation(),
	BeforeContent:  documentContext(),
	Footer:         siteFooter(),
})
```

Use `HTMLDependenciesInline` for a self-contained page. With
`HTMLDependenciesLocal`, mount each handler at its own path:

```go
mux := http.NewServeMux()
mux.Handle("/assets/", goshtosoassets.Handler())
mux.Handle("/margo-assets/", margo.HTMLAssetHandler())
mux.Handle(chartassets.Prefix, chartassets.Handler()) // /charts/assets/
```

`goshtosoassets.Handler` owns `/assets/`; `margo.HTMLAssetHandler` owns only
`/margo-assets/`; `chartassets.Handler` owns `/charts/assets/`.
The Margo handler does not serve either dependency mount.

## Releases and module history

Every supported package belongs to the root module and one root release tag.
Starting with `v0.0.3`, releases include `margo_VERSION_OS_ARCH` archives and
SHA-256 digests in `checksums.txt`. Unix archives use `.tar.gz`; Windows
archives use `.zip` and contain `margo.exe`.

The historical submodule tags remain unchanged. Consumers that required
`github.com/araihu/margo/pdf`, `github.com/araihu/margo/charts`, or
`github.com/araihu/margo/cmd/margo` as separately versioned modules must remove
those old requirements and select a root Margo release. An old nested module
can otherwise shadow the package supplied by the root module.

## Contribute and verify

Run the repository as one module:

```sh
GOWORK=off GOFLAGS=-mod=readonly go test -race ./...
GOWORK=off GOFLAGS=-mod=readonly go vet ./...
CGO_ENABLED=0 GOWORK=off GOFLAGS=-mod=readonly go build ./cmd/margo
```

### Local CI with Dagger

Margo's portable CI logic uses Dagger v0.21.8. These functions also back the
GitHub Actions adapters:

```sh
dagger call required
dagger call portable-release-shape
scripts/prepare-dagger-git.sh
dagger call snapshot --git-bundle=.dagger-git.bundle export --path=dist
dagger call musl
dagger call pages-site export --path=_site
```

Go modules and compiler outputs use separate Dagger cache volumes. Published
Pages and GitHub releases remain explicit provider-side effects; local
functions only validate or produce their input artifacts.

Local calls generate an isolated `local` execution nonce. CI adapters write a
validated `.dagger-ci-context.json`: pull requests receive per-PR untrusted
caches, while main and releases use separate trusted domains. Tests and
verification always run; only dependency and compiler cache volumes persist.

## Security

See [SECURITY.md](SECURITY.md) for private vulnerability reporting.

## License

Margo is licensed under the [MIT License](LICENSE).
