# Editorial HTML browser evidence

Margo's tagged editorial HTML journey was tested on 2026-08-09 with Google
Chrome 151.0.7922.77 on macOS 26.5.2 (build 25F84), arm64. This records the
environment used for the gate; it is not a minimum, maximum, pinned, or
otherwise required browser version for users.

The gate composes the current root and charts modules in a temporary Go
workspace and exercises generated HTML shaped like both downstream consumers:

- a Manja-owned page embeds `EditorialResult.Fragment()` and changes the host
  Goshtoso theme and dark mode without replacing the article;
- a public article page from `RenderPublication` verifies initial canonical,
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
