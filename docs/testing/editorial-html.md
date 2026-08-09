# HTML browser evidence

Margo's tagged HTML journey was tested on 2026-08-09 with Google
Chrome 151.0.7922.77 on macOS 26.5.2 (build 25F84), arm64. This records the
environment used for the gate; it is not a minimum, maximum, pinned, or
otherwise required browser version for users.

The gate composes the current root and charts modules in a temporary Go
workspace and exercises generated HTML shaped like both downstream consumers:

- a Manja-owned page embeds `HTMLResult.Fragment()` and changes the host
  Goshtoso theme and dark mode without replacing the article;
- a public article page from `webpublication.Render` verifies initial canonical,
  Open Graph, X/Twitter, and article metadata;
- local and inline dependency modes verify request counts, ordering, no
  external traffic, no failed requests, and no browser exceptions;
- a Markdown table cycles source, ascending, descending, and source order with
  `aria-sort` updates;
- four charts keep four static SVG projections and four accessible data tables
  while an expand modal opens and closes;
- a CDP context with script execution disabled keeps article text, table rows,
  SVGs, and accessible chart data readable without progressive sort controls.

Run the browser gate from a temporary workspace containing the root and charts
modules:

```sh
workspace_dir=$(mktemp -d)
repo_root=$(pwd -P)
(cd "$workspace_dir" && GOWORK=off go work init "$repo_root" "$repo_root/charts")
GOWORK="$workspace_dir/go.work" GOFLAGS=-mod=readonly \
  go test -tags=editorial_e2e ./charts/e2e \
  -run TestGeneratedEditorialHTMLJourneys -count=1 -v
find "$workspace_dir" -depth -delete
```

The test discovers an installed Chromium-family browser. Set `MARGO_CHROMIUM`
to an explicit executable when discovery is not appropriate. A missing browser
skips only the tagged browser gate; ordinary unit tests remain browser-free.

## Goal audit

| Requirement | Evidence |
| --- | --- |
| Manja-compatible fragment | `/manja` initial HTML embeds the exact fragment in `.manja-markdown`; the browser preserves article identity while changing host theme and dark mode. |
| araihu.com blog page | `/guide` uses optional `webpublication.Render` around `RenderHTMLPage`, with a custom `araihu` theme and a verified fixture authority; initial canonical, Open Graph, X/Twitter, and article fields are asserted. |
| Goshtoso theme inheritance | The Manja journey compares computed token-backed foreground/background colors before and after live `modern` to `dracula` plus light/dark changes. |
| Table sorting | Browser clicks cover ascending, descending, and restored source order after the initial source state, including active `aria-sort`. |
| Chart controls | Four charts retain four static SVGs and four accessible data tables while the expand dialog opens and closes. |
| JavaScript-free readability | CDP disables script execution before navigation; article text, two source-order table rows, four SVGs, and four chart data tables remain readable. |
| No duplicate runtime | Capability graph tests assert order and identity; browser network evidence asserts each local requirement once, zero asset requests for inline mode, no external requests, and no duplicate DOM IDs. |
| PDF deferred | This slice changes and accepts generated HTML only. PDF generation and visual correctness remain deferred to separate human review. |

Goshtoso Charts currently emits component-scoped CSS with its static SVG. The
adapter marks every such style with `data-margo-extension-style="charts"`;
`RenderHTML` accepts only that explicit trusted provenance, rejects
unowned styles, and continues to reject every fragment script. Host theme and
color-mode ownership remain unchanged because the component CSS consumes
Goshtoso tokens.
