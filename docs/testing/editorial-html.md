# HTML browser evidence

Margo’s tagged HTML journey was tested on 2026-08-09 with Google Chrome
151.0.7922.77 on macOS 26.5.2 (build 25F84), arm64. This records gate evidence,
not a minimum, maximum, pinned, or otherwise required browser version.

The root-module package test composes the root renderer and optional charts
package in generated HTML shaped like downstream consumers:

- a Manja-owned page embeds `HTMLResult.Fragment()` while the host changes the
  Goshtoso theme and dark mode;
- a consumer-composed page supplies canonical, Open Graph, X/Twitter, article
  metadata, and byline through `HTMLPageInput` seams;
- local and inline dependency modes verify ordering, request counts, no external
  traffic, no failed requests, and no browser exceptions;
- tables cycle source, ascending, descending, and source order with `aria-sort`;
- charts keep static SVG and accessible data tables while the expand dialog opens
  and closes;
- disabled JavaScript keeps article text, table rows, SVGs, and chart data
  readable.

Run the tagged root-module package test from the repository root:

```sh
GOWORK=off GOFLAGS=-mod=readonly \
  go test -tags=editorial_e2e ./charts/e2e \
  -run TestGeneratedEditorialHTMLJourneys -count=1 -v
```

This command needs no `charts/go.mod`, no `go.work`, and no temporary
workspace. It discovers an installed Chromium-family browser. Set
`MARGO_CHROMIUM` to an explicit executable when discovery is unsuitable. A
missing Chromium skips this tagged gate. Untagged tests may opportunistically
launch installed Chromium and skip browser assertions when it is unavailable.
Chromium is not required for the ordinary suite.

## Goal audit

| Requirement | Evidence |
| --- | --- |
| Manja-compatible fragment | `/manja` initial HTML embeds the fragment in `.manja-markdown`; the browser preserves article identity while changing host theme and dark mode. |
| Consumer-composed page | `/guide` uses `RenderHTMLPage` with caller-owned `Head` and `BeforeContent` components plus a custom theme. |
| Goshtoso theme inheritance | The Manja journey compares token-backed foreground and background colors before and after theme and color-mode changes. |
| Table sorting | Browser clicks cover ascending, descending, and restored source order, including active `aria-sort`. |
| Chart controls | Charts retain static SVG and accessible data tables while the expand dialog opens and closes. |
| JavaScript-free readability | CDP disables scripts before navigation; article text, source-order table rows, SVGs, and chart data tables remain readable. |
| No duplicate runtime | Capability graph and browser network evidence assert each local requirement once, zero asset requests for inline mode, no external requests, and no duplicate DOM IDs. |
| PDF deferred | This slice accepts generated HTML only. PDF generation and visual correctness remain separate human review. |

Goshtoso Charts emits component-scoped CSS with static SVG. The adapter marks
each style with `data-margo-extension-style="charts"`; `RenderHTML` accepts
that trusted provenance, rejects unowned styles, and continues to reject every
fragment script. Host theme and color-mode ownership remain unchanged because
component CSS consumes Goshtoso tokens.
