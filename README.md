# Margo

<p align="center">
  <img src="assets/margo-mascot.png" alt="Margo, a pink Go gopher holding a rendered document in a publishing atelier." width="480">
</p>

Margo turns Markdown into standalone HTML, linked static sites, PDF documents,
and presentation decks. It ships as one Go module and one `margo`
command. Applications keep ownership of URLs, navigation, storage, and
deployment.

## Local CI with Dagger

Margo's portable CI logic is exposed by the Dagger module and uses Dagger
v0.21.8. The same functions back the GitHub Actions adapters:

```sh
dagger call required
dagger call portable-release-shape
scripts/prepare-dagger-git.sh
dagger call snapshot --git-bundle=.dagger-git.bundle export --path=dist
dagger call musl
dagger call pages-site export --path=_site
```

Go modules and compiler outputs use separate Dagger cache volumes. Published
Pages and GitHub Releases remain explicit provider-side effects; the local
functions only validate or produce their input artifacts.

Local calls generate an isolated `local` execution nonce. CI adapters instead
write a validated `.dagger-ci-context.json`: pull requests receive per-PR
untrusted caches, while main and releases use distinct trusted domains. Test
and verification functions always execute; only dependency and compiler cache
volumes persist.

## Install

Margo requires Go 1.26.5 or newer. Install a root release with Go:

```sh
go install github.com/araihu/margo/cmd/margo@vX.Y.Z
```

Starting with `v0.0.3`, each root [GitHub Release](https://github.com/araihu/margo/releases)
contains `margo_VERSION_OS_ARCH` archives for Linux, macOS, and Windows on
amd64 and arm64, plus `checksums.txt` with SHA-256 digests. Unix archives use
`.tar.gz`; Windows archives use `.zip` and contain `margo.exe`.

Download the archive for your platform, verify `checksums.txt`, extract it,
and place `margo` or `margo.exe` on `PATH`. Release binaries use
`CGO_ENABLED=0`: they discover installed Chromium but do not download a browser
or load native WebKit libraries.

## Ordinary Markdown quick start

Create `proposal.md` with ordinary Markdown. Only generic metadata is needed:

```markdown
---
title: Proposta técnica
language: pt-BR
---

# Proposta técnica

The same source renders to every target.
```

Check it, then render HTML or PDF:

```sh
margo check proposal.md
margo html proposal.md --output proposal.html
margo pdf proposal.md --output proposal.pdf
```

No policy file is needed for ordinary Markdown, local images, Mermaid, tables,
or code. A policy is required only for privileged raw HTML or iframe embeds.

Page geometry is an optional document preference. Every field can still be
overridden by an explicit CLI or API value:

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

## Supported Go packages

Install every supported library package through the root module:

```sh
go get github.com/araihu/margo@vX.Y.Z
```

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
| `github.com/araihu/margo/site` | Build deterministic multi-page HTML sites from caller-supplied site-relative `[]site.Source`. Directory discovery and publication are CLI-only. | `site.Build` |

`cmd/margo` is the CLI program, not a library API. `internal/...` packages are
unsupported implementation details. `profiles/` and both
`tools/optimistic-renderer` commands are test and developer tools, not release
surface. The chart-aware renderer at `charts/tools/optimistic-renderer` has the
same status.

The root package is enough for a standalone page:

```go
compiler := margo.New()
document, err := compiler.Compile(ctx, margo.Source{Name: "hello.md", Content: markdown})
if err != nil {
	return err
}
rendered, err := compiler.Render(ctx, document)
if err != nil {
	return err
}
page, err := margo.RenderStandalone(rendered)
```

Library consumers can run the same preflight with `margo.Check`. Supply
`margo.WithCheckAssetReader` when checks must read local assets. Charts are
opt-in for libraries:

```go
compiler := margo.New(margo.WithExtension(charts.Extension()))
```

`RenderHTML` projects a rendered document to an `HTMLResult` whose fragment is
one semantic `article.margo-document`. `RenderHTMLPage` provides a generic page
shell without claiming a publication domain. `HTMLPageInput` gives the host
ownership of composition and dependency choices:

```go
page, err := margo.RenderHTMLPage(htmlResult, margo.HTMLPageInput{
	DependencyMode: margo.HTMLDependenciesLocal,
	Head:           siteMetadata(),
	Header:         siteNavigation(),
	BeforeContent:  documentContext(),
	Footer:         siteFooter(),
})
```

Use `HTMLDependenciesInline` for one self-contained page. With
`HTMLDependenciesLocal`, mount each handler at its own path:

```go
mux := http.NewServeMux()
mux.Handle("/assets/", goshtosoassets.Handler())
mux.Handle("/margo-assets/", margo.HTMLAssetHandler())
mux.Handle(chartassets.Prefix, chartassets.Handler()) // /charts/assets/
```

`goshtosoassets.Handler` owns `/assets/`; `margo.HTMLAssetHandler` owns only
`/margo-assets/`; `chartassets.Handler` owns `chartassets.Prefix`,
`/charts/assets/`. The Margo handler does not serve either dependency mount.

See [Host policy and natural iframe embeds](docs/policy.md), the generated
[policy reference](docs/reference/policy.md), and the generated
[document metadata reference](docs/reference/document-metadata.md).

## CLI reference

Run `margo --help` for the generated command reference. The command surface is:

```text
margo check INPUT
margo html INPUT [--output PATH|-] [--force]
margo site INPUT_DIR --output-dir OUTPUT_DIR [--assets local|inline]
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

`INPUT` for `check`, `html`, `pdf`, and `deck` is one path or `-` for stdin.
`site` takes a directory, not stdin. Commands write artifacts and command
reports to stdout; errors and diagnostics go to stderr. `--diagnostics text`
is the default and `--diagnostics json` selects JSON diagnostics. `check`
writes findings to stdout. Errors exit nonzero; warnings remain visible without
blocking `check`.

`html` writes HTML to stdout by default. `deck` defaults to `--format html`
and stdout. `pdf` and `deck --format pdf` require an explicit `--output PATH`
or `--output -` for binary stdout. `html`, `pdf`, and `deck` refuse an existing
output file unless `--force` is present. `site --output-dir` is required; the
destination must not already exist. `site --assets local` is the default and
keeps assets local; `--assets inline` embeds them.

All rendering commands accept `--policy FILE` for a trusted host policy.
`html` and `pdf` also accept `--title TEXT` and `--lang TAG`.
`margo schema policy` and `margo schema document` emit the exact embedded
Draft 2020-12 schema bytes for this Margo version.

### Check

`margo check INPUT [--policy FILE] [--diagnostics text|json]` checks Markdown
compatibility without rendering. It reports raw HTML, unavailable images,
incompatible SVG, invalid frontmatter, legacy Mermaid configuration, empty
image alternatives, missing document language, skipped headings, empty links,
and relative links. Each finding identifies source, line, field pointer, and a
remediation hint.

### HTML

`margo html INPUT [--output PATH|-] [--force] [--title TEXT] [--lang TAG]
[--policy FILE] [--diagnostics text|json]` renders one standalone HTML page.
The output default is `-`, so HTML goes to stdout unless `--output` names a
file.

### Site

`margo site INPUT_DIR --output-dir OUTPUT_DIR [--assets local|inline]
[--policy FILE] [--diagnostics text|json]` builds a linked site from recursive
`.md` and `.markdown` inputs. It maps pages to `.html`, validates Markdown
links and fragments, and writes `margo-manifest.json`. Output is staged beside
the destination and published by rename only after a successful build.

### PDF

`margo pdf INPUT --output PATH|- [--force] [--engine auto|chromium|native]
[--engine-path PATH] [--page-size A4|Letter] [--orientation portrait|landscape]
[--margin-top MM] [--margin-right MM] [--margin-bottom MM] [--margin-left MM]
[--relative-links strip|error|keep|resolve] [--base-url URL] [--title TEXT]
[--lang TAG] [--print-chart-data] [--policy FILE]
[--diagnostics text|json]` renders a PDF.

Defaults are `--engine auto`, A4 portrait, and readable document margins of
24 mm top, 22 mm right, 26 mm bottom, and 22 mm left unless `margo.page`
supplies a page preference. Explicit CLI flags override document preferences;
set all four margin flags to `0` for full bleed. The default
`--relative-links strip` keeps visible text while removing relative PDF link
targets. `--base-url URL` selects `resolve` when `--relative-links` was not set;
explicit `resolve` requires `--base-url`. `margo doctor` reports candidate
discovery and compiled capabilities. One formatted semantic exact-data table
follows each chart in HTML; redundant chart-owned disclosures are suppressed.
Exact-data tables are omitted from PDF by default. `--print-chart-data` adds
them to PDF output.

All v1 chart families (`bar`, `line`, `pie`, `doughnut`, and `scatter`) accept
`renderer: interactive`; omitting it preserves static SVG. Interactive scatter
accepts exactly one point or value per declared category. Multi-sample scatter
data remains available through the static renderer.

Current releases use installed Chromium. `auto` tries an explicit
`--engine-path`, `MARGO_CHROMIUM_PATH`, discovered Chromium-family executables,
then a native slot. Native backends are currently compiled out, so selecting
`--engine native` does not make WKWebView, WebView2, or WebKitGTK available.
Margo never downloads an engine or browser. A selected engine that fails does
not fall back to another engine.

### Deck

`margo deck INPUT [--format html|pdf] [--output PATH|-] [--force]
[--engine auto|chromium|native] [--engine-path PATH] [--page-size A4|Letter]
[--orientation portrait|landscape] [--margin-top MM] [--margin-right MM]
[--margin-bottom MM] [--margin-left MM] [--policy FILE]
[--diagnostics text|json]` renders a presentation deck.

Its defaults are HTML to stdout, `--engine auto`, A4, portrait, and zero
margins. PDF decks require `--format pdf` and an explicit output path or `-`.
PDF deck links use the same default `strip` policy as `margo pdf`.

### Doctor, version, and completion

`margo doctor [--diagnostics text|json]` reports PDF candidates and reasons.
`margo version` and `margo --version` print the version and compiled engine
capabilities without probing external engines. `margo help [command]` prints
command help. `margo completion SHELL [--no-descriptions]` prints shell
completion. `SHELL` is `bash`, `zsh`, `fish`, or `powershell`; the
`--no-descriptions` flag applies to all four generators.

## One module and release

Every supported package belongs to the root module and one root release tag.
The [ADR 0001](docs/decisions/0001-unified-module-and-cli.md) records this
decision and the current native-PDF boundary.

The historical submodule tags remain unchanged. Consumers that previously required
`github.com/araihu/margo/pdf`, `github.com/araihu/margo/charts`, or
`github.com/araihu/margo/cmd/margo` as independently versioned modules must
remove those old requirements and select the root Margo release. Otherwise an
old nested module can shadow the package supplied by the root module.

## Development and verification

Run the repository as one module:

```sh
GOWORK=off GOFLAGS=-mod=readonly go test -race ./...
GOWORK=off GOFLAGS=-mod=readonly go vet ./...
CGO_ENABLED=0 GOWORK=off GOFLAGS=-mod=readonly go build ./cmd/margo
```

The [PDF engine matrix](docs/testing/pdf-engine-matrix.md) records tested
browser versions as evidence. It is not a supported-version policy.

## Security

See [SECURITY.md](SECURITY.md) for private vulnerability reporting.

## License

Margo is licensed under the [MIT License](LICENSE).
