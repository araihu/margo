# PDF engine test matrix

This page records tested evidence for Margo's PDF engine behavior. It is not a minimum
or maximum browser-version policy: discovery reports the installed
version and attempts the selected engine instead of rejecting an unfamiliar
Chromium build number.

Observed on 2026-08-10:

| Engine or build | Host evidence | Automated result | Current release capability |
| --- | --- | --- | --- |
| Google Chrome 151.0.7922.77 | macOS 26.5.2 build 25F84, arm64, Go 1.26.5 | Installed-browser unit, CLI PDF, deck PDF, Mermaid runtime, and no-fallback tests passed | Available when explicitly selected or discovered |
| Chromium 142.0.7400.0 | macOS 26.5.2 build 25F84, arm64, Go 1.26.5 | Black-box article/deck HTML DOM tests and PDF artifact tests passed | Available when explicitly selected or discovered |
| WKWebView | macOS native slot | Capability probe only | compiled out pending official bridge and runner evidence |
| WebView2 | Windows native slot | Contract and cross-build boundary only | compiled out pending official bridge and runner evidence |
| WebKitGTK | opt-in Linux native slot | Contract and cross-build boundary only | compiled out pending official bridge, declared libraries, and runner evidence |
| Portable Linux and musl | `CGO_ENABLED=0` static Linux test binary compiled on the observed host | Native probe reports `pdf.native.compiled_out`; runner execution remains required | Installed Chromium only |

The automated browser checks assert:

- generated article and deck HTML load without external runtime requests;
- PNG, JPEG, WebP, GIF, and SVG images decode with non-zero dimensions;
- static chart SVG and accessible data tables remain present;
- embedded Mermaid reaches `margoRuntimeReady` and inserts SVG;
- deck Previous, Next, Print, ArrowLeft, ArrowRight, Home, End, and print CSS
  remain functional;
- generated PDF bytes begin with `%PDF-`, runtime reports are terminal and
  identity-bound, and engine path/version provenance is present;
- a render failure does not trigger fallback to another engine.

PDF typography, clipping, page breaks, and overall visual quality are reviewed
by a human against generated checkpoint artifacts. Automated success does not
infer that visual verdict.
