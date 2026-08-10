# ADR 0001: Unified module, release, and CLI

Date: 2026-08-10
Status: accepted for implementation

## Context

Margo currently contains separate Go modules for core, charts, PDF, and the
command. This makes one repository behave like several products: packages can
drift between versions, the PDF module needs its own tags, and the empty command
module cannot install a complete tool from the root release.

The intended product is one document engine. Library consumers choose the
packages they need. Command users install one `margo` binary containing HTML,
PDF, deck, chart, and Mermaid support.

PDF generation also needs a portable discovery policy. Chromium should be used
when installed. Platform-native web engines provide a fallback where the
distributed artifact supports them. Musl and minimal containers must not be
forced to consume a glibc or WebKitGTK-linked binary.

## Decision

Margo will use one Go module at `github.com/araihu/margo`, one root release tag,
and one binary named `margo`.

`charts`, `pdf`, `deck`, and `cmd/margo` are packages in the root module. Their
nested module files are removed. Future releases do not create `charts/`,
`pdf/`, or `cmd/margo/` tags. Historical tags remain unchanged.

The complete CLI installs with:

```sh
go install github.com/araihu/margo/cmd/margo@vX.Y.Z
```

The CLI uses Cobra and exposes `html`, `pdf`, `deck`, `doctor`, and `version`.
Core stays consumer-neutral and does not import optional packages. CLI owns
composition of core, charts, deck, and PDF engines.

PDF engine selection defaults to `auto`. It checks an explicit path, an
environment path, installed Chromium-family browsers, then a compiled and
available native engine. Discovery may fall back; rendering may not. Explicit
engine selection disables fallback. Failure reports every attempted engine.
Margo never downloads a browser or native runtime.

Current release capability is installed Chromium discovery on Linux, macOS,
and Windows. Portable binaries use `CGO_ENABLED=0`; the native engine slots are
compiled out and report that state through `doctor`.

Target native capabilities, after their platform runners provide verifiable
backend evidence, are:

- macOS: installed Chromium with WKWebView fallback;
- Windows: installed Chromium with WebView2 fallback;
- portable Linux and musl: installed Chromium, no native fallback;
- optional Linux WebKitGTK artifact: installed Chromium with WebKitGTK
  fallback and declared dynamic dependencies.

Every artifact uses the same root tag and contains a binary named `margo`.

## Consequences

Advantages:

- one dependency graph and version;
- one installation path;
- no submodule release ordering;
- CLI and libraries cannot accidentally ship different Margo versions;
- engine behavior is visible through `doctor` and deterministic diagnostics;
- musl users receive a portable default artifact.

Costs:

- root module consumers download metadata for dependencies used by optional
  packages, although Go links only imported packages;
- platform release builds differ in compiled engine capability;
- macOS and Linux-native engines need CGO and platform libraries;
- native PDF output can differ from Chromium output;
- existing consumers of independently tagged PDF modules must migrate to a
  root release containing this decision;
- an old direct PDF or charts module requirement can shadow the package in a
  newer root module until the consumer removes that requirement;
- platform lock and runner-contract schemas need a new root-relative version.

## Superseded decisions

This ADR supersedes the independent module and release sequence in
`docs/GOSHTOSO_MARKDOWN_DESIGN.md`, including root-first then charts/PDF/CLI
submodule tags. It also supersedes native-first CLI engine selection. That file
remains historical design context and is not rewritten to disguise the change.

The detailed implementation contract is
`docs/superpowers/specs/2026-08-10-margo-unified-cli-design.md`.
