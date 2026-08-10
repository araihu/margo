# Margo unified module and CLI design

Date: 2026-08-10
Status: approved for implementation
Decision record: [ADR 0001](../../decisions/0001-unified-module-and-cli.md)

## Purpose

Margo ships as one Go module, one release, and one `margo` command. Library
consumers can use the Markdown-to-HTML core and the chart, PDF, and deck
packages independently. The command imports all of them and exposes the full
feature set.

This design replaces the repository's original independent release model for
`charts`, `pdf`, and `cmd/margo`. Existing submodule tags remain historical.
Future releases use only root tags such as `v0.1.0`.

## Goals

- One module path and dependency graph rooted at `github.com/araihu/margo`.
- One root version for core, charts, PDF, deck, and CLI.
- One install command:

  ```sh
  go install github.com/araihu/margo/cmd/margo@vX.Y.Z
  ```

- A Cobra CLI that generates standalone HTML, PDF documents, HTML decks, and
  PDF decks.
- Installed Chromium as the preferred PDF engine.
- Platform-native PDF fallback where the distributed binary includes it.
- Explicit engine override for nonstandard systems.
- No browser downloads, hidden installation, or silent recovery from render
  failures.
- Public documentation that states the release and engine support matrix.

## Non-goals

- Static-site routing, navigation, feeds, deployment, or content discovery.
- Automatic Chromium or native-runtime installation.
- One Linux native binary that works against every glibc, musl, WebKitGTK, and
  GTK combination.
- Silent fallback after an engine begins rendering.
- Complete Marpit directive or plugin compatibility.
- Batch and glob input in the first unified CLI release.
- Byte-identical PDF output across different engines.

## Repository and package boundaries

The repository becomes one Go module:

```text
github.com/araihu/margo
├── root package        Markdown compilation and semantic HTML
├── charts              Optional chart extension
├── deck                Deck parsing and HTML rendering
├── pdf                 Renderer-neutral contracts
│   ├── chromium        Installed Chromium engine
│   ├── native          Platform-native engine
│   └── engines         Discovery and selection
└── cmd/margo           Cobra CLI
```

Implementation removes these nested module files:

- `charts/go.mod`
- `charts/go.sum`
- `pdf/go.mod`
- `pdf/go.sum`
- `cmd/margo/go.mod`
- any `cmd/margo/go.sum` introduced before consolidation

Dependencies required by these packages move into root `go.mod` and `go.sum`.
CI rejects any nested `go.mod` below the repository root.

Package dependency direction is fixed:

```text
cmd/margo -> root, charts, deck, pdf/engines
deck      -> root
charts    -> root, goshtoso-charts
pdf/*     -> root or pdf contracts
root      -> no Margo subpackage
```

The root package never imports charts, deck, PDF, Cobra, or CLI code. Deck
produces HTML and does not import PDF. CLI composes deck HTML with a PDF engine
when PDF deck output is requested.

## CLI contract

The first unified command surface is:

```text
margo html INPUT --output OUTPUT
margo pdf INPUT --output OUTPUT
margo deck INPUT --format html|pdf --output OUTPUT
margo doctor
margo version
```

`INPUT` accepts one file path or `-` for standard input. HTML may write to
standard output when `--output -` is explicit. PDF output requires a file
unless the caller explicitly sets `--output -`; diagnostics never share the
artifact stream and always use standard error.

`html` defaults to `--output -`. `deck` defaults to `--format html` and
`--output -`. PDF output, including `deck --format pdf`, requires an explicit
`--output`; `--output -` is accepted for callers intentionally consuming binary
stdout. An existing filesystem destination is refused unless `--force` is set.
Each command accepts exactly one input. `--diagnostics text|json` controls only
standard error formatting.

The `html` command emits self-contained HTML by default. A later local-assets
mode can expose the existing requirement graph without changing the default.
The CLI registers charts and all supported core Markdown extensions. Library
consumers continue to register charts explicitly.

The CLI uses Cobra. Command constructors receive filesystem, standard stream,
clock, build-information, and engine-discovery dependencies. `main.go` only
constructs the production root command and executes it. Tests invoke commands
without starting subprocesses unless the test is explicitly black-box.

## Shared rendering flow

All commands use one core pipeline:

```text
Markdown source
  ├── html: core + charts -> standalone HTML
  ├── pdf:  core + charts -> standalone HTML -> selected engine -> PDF
  └── deck: split slides -> core + charts per slide -> deck HTML
                                           └── selected engine -> deck PDF
```

Compilation and rendering finish before artifact publication. Existing spool
and atomic sink contracts stage complete output, validate its digest, and
publish once. A compile, runtime, engine, or validation failure leaves no
partial final artifact.

HTML generation does not require an installed engine. Browser runtime tasks,
including Mermaid, remain embedded and progressive in standalone HTML. PDF
generation loads the same HTML, waits for every declared runtime task, and
prints only after the terminal report is ready.

Static chart output and accessible data remain present before JavaScript runs.
PDF and deck PDF output hide controls that have no printed meaning.

## Deck contract

Deck mode recognizes:

- one optional opening YAML frontmatter block for deck metadata;
- a line containing exactly `---` as a slide separator after frontmatter;
- separators only outside fenced code blocks;
- ordinary Margo Markdown, Mermaid, tables, code, images, and charts inside
  each slide.

Empty slides are invalid. Each slide receives a stable ordinal and identity.
Deck HTML uses an article containing accessible slide sections. Screen output
provides previous/next controls and ArrowLeft, ArrowRight, Home, and End keyboard
navigation. Navigation does not alter document content. Print CSS maps exactly
one slide to one page.

The first unified release does not interpret Marpit comments, arbitrary CSS,
presenter notes, transitions, fragments, or plugins. Unknown deck-specific
syntax stays ordinary Markdown or produces an explicit unsupported diagnostic;
it is never executed as HTML or CSS.

## PDF engine selection

Engine flags are shared by `pdf` and PDF-formatted `deck`:

```text
--engine auto|chromium|native
--engine-path PATH
```

`auto` is the default. Discovery order is:

1. `--engine-path`, validated as an installed Chromium-family executable.
2. `MARGO_CHROMIUM_PATH`.
3. Supported Chromium-family executables in `PATH` and known platform paths.
4. Native engine compiled into the current binary and available at runtime.
5. A typed failure containing every attempted engine and its reason.

`--engine chromium` selects Chromium only. `--engine native` selects the
platform-native engine only. Explicit selection never falls back.

Automatic fallback happens only while an engine is absent or unavailable. If
an engine starts loading or rendering and then fails, that error is final. The
CLI records the selected engine name, version, executable path when applicable,
and selection source in the export report.

No code path downloads a browser, native runtime, package, or helper binary.

## Native engine and release matrix

The binary name and root version stay identical across artifacts, but compiled
capabilities can differ by platform:

| Artifact | Chromium | Native fallback | Notes |
| --- | --- | --- | --- |
| macOS | Installed browser | WKWebView | Native bridge can use CGO and Apple frameworks |
| Windows | Installed browser | WebView2 | Requires WebView2 runtime |
| Linux portable, including musl | Installed browser | None | `CGO_ENABLED=0`; errors honestly when no browser exists |
| Linux WebKitGTK variant | Installed browser | WebKitGTK | CGO build with declared GTK, GLib, and WebKitGTK runtime requirements |

One release may contain multiple platform archives. Every archive contains a
binary named `margo` built from the same root tag. Package APIs and command
syntax do not vary by artifact.

The portable Linux artifact is the default Linux download. The WebKitGTK
variant is optional and named clearly in release assets. A glibc-linked native
artifact is not presented as musl-compatible.

Native implementation follows official platform APIs:

- macOS uses WKWebView PDF creation on the required main thread;
- Windows uses WebView2 and its installed runtime;
- Linux native uses WebKitGTK only when compiled and dynamically available.

References:

- [Go cgo documentation](https://pkg.go.dev/cmd/cgo)
- [WKWebView PDF creation](https://developer.apple.com/documentation/webkit/wkwebview/createpdf%28configuration%3Acompletionhandler%3A%29)
- [WebView2 Win32 initialization](https://learn.microsoft.com/en-us/microsoft-edge/webview2/get-started/win32)
- [WebKitGTK print operation](https://webkitgtk.org/reference/webkit2gtk/2.35.2/WebKitPrintOperation.html)
- [Alpine Linux musl documentation](https://wiki.alpinelinux.org/wiki/Musl)

## Discovery and diagnostics

`pdf/engines` owns one discovery result used by both command execution and
`margo doctor`. Each candidate records:

- engine identity;
- selection source;
- compiled-in status;
- discovered path;
- runtime version when obtainable without rendering;
- availability status;
- stable diagnostic code and human explanation.

Discovery distinguishes:

- not installed;
- compiled out;
- invalid explicit path;
- unsupported platform;
- runtime missing;
- probe failure;
- render failure.

The final unavailable error lists attempts in deterministic discovery order.
An invalid explicit path fails immediately because the user selected it; auto
discovery does not hide the mistake.

## Resource and security behavior

Engines render reviewed, finalized HTML. Navigation and subresource policies
block unexpected remote requests. Runtime tasks, fonts, images, Mermaid, and
layout readiness must finish within configured limits. Timeout, cancellation,
blocked request, runtime protocol mismatch, or engine process failure aborts
the output.

Chromium runs in an isolated temporary profile. The engine closes and reaps
its process on success, failure, and cancellation. Native engines release
webviews, callbacks, temporary files, and main-loop resources on every path.

The explicit engine path is never executed during generic option parsing. It
is normalized, checked as a regular executable file, probed with bounded
arguments, and then used only by the selected engine.

## Version and release behavior

`margo version` reads Go build information and reports:

- root module path;
- root version or a clear development marker;
- commit when available;
- Go version;
- operating system and architecture;
- compiled engine capabilities.

Future tags are root tags only. Existing `pdf/v0.0.1` and `pdf/v0.0.2` tags are
retained as immutable history. No replacement submodule tags are created.

Package import paths remain stable, but module requirements change. Consumers
must remove direct `require github.com/araihu/margo/pdf` or
`github.com/araihu/margo/charts` entries and require
`github.com/araihu/margo` instead. Otherwise Go's longest-module-prefix rule can
keep selecting an old submodule over the package in a newer root release.

The current platform lock and runner commands refer to the old PDF-module
boundary. Consolidation versions their schema forward, changes package-local
commands from `./platform` to `./pdf/platform`, and binds source provenance to
the root module without inventing a pseudo-version or module sum. Historical
root-transfer and submodule receipts remain immutable evidence.

The Goshtoso Charts dependency moves to the root graph at its exact verified
version. Consolidation does not add a local `replace`, fabricate a version, or
silently change that dependency while moving module metadata.

README installation examples always use the root version. README also includes
the engine matrix, `auto` order, no-download promise, musl limitation, explicit
override, and `doctor` command.

## Verification strategy

Implementation follows RED, GREEN, REFACTOR for each behavior. Required layers:

1. Module-boundary test rejects nested `go.mod` files.
2. Root readonly module verification and tests cover all packages.
3. Cobra unit tests cover help, input, output, diagnostics, and exit behavior.
4. Black-box tests build and execute the real `margo` binary.
5. HTML browser tests open generated standalone pages and validate semantics,
   charts, images, Mermaid readiness, and print preparation.
6. Chromium PDF tests verify page configuration, runtime completion, request
   blocking, process cleanup, and valid PDF structure.
7. Deck tests cover frontmatter, fenced-code separators, empty-slide rejection,
   accessibility, navigation, and one-slide-per-page print behavior.
8. Native tests run on their matching macOS, Windows, and WebKitGTK runners.
9. A musl container test proves the portable binary starts, finds an installed
   Chromium when supplied, reports native as compiled out, and fails honestly
   when no engine exists.
10. Release gates build every archive from one root tag and verify that
    `go install github.com/araihu/margo/cmd/margo@TAG` resolves the same module
    and command version.

PDF pixel and pagination quality receives human review in addition to
automated structural tests. Automated gates do not claim visual acceptance.

## Migration sequence

1. Consolidate module metadata and make root readonly gates cover every
   package. Version platform lock and runner contracts for the new root-relative
   paths.
2. Introduce Cobra command shell, version, and doctor without conversion
   commands.
3. Implement deck parsing and HTML rendering.
4. Implement Chromium discovery and PDF engine.
5. Implement platform-native engines and compiled-out capability reports.
6. Add HTML, PDF, and deck commands over the shared pipeline.
7. Add black-box, browser, platform, and musl verification.
8. Update README and release documentation.

Each step must leave root tests green and must not create a tag, release, or
publication. Integration and release remain separately authorized lifecycle
actions.
