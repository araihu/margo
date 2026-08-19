# Task 7 Report: Layout-kind Rendering and Asset Ownership

## Status

Complete. Commit: this report's containing commit, `refactor: render chrome by
layout kind`.

## Files

- `site/build.go`
- `site/config_build.go`
- `site/config_build_test.go`
- `site/goshtoso_shell.go`
- `site/layout.go`
- `site/page_actions.go`
- `site/page_actions_test.go`
- `site/site_navigation.go`

## TDD evidence

Layout-kind rendering RED:

```text
$ GOWORK=off go test ./site -run '^TestBuildConfigRendersLayoutKindsWithOwnedChrome$' -count=1
TestBuildConfigRendersLayoutKindsWithOwnedChrome: id=\"left-nav\" leaked into landing
FAIL github.com/araihu/margo/site
```

The first implementation compile exposed the retained legacy test contract:

```text
site/site_navigation_test.go:43:53: not enough arguments in call to builder.renderPageHead
```

The typed head renderer now accepts `ResolvedLayout`; the legacy-signature
wrapper remains until Task 8 so existing fixtures keep compiling.

Inline dependency ownership RED:

```text
$ GOWORK=off go test ./site -run '^TestBuildConfigStagesLayoutKindAssetsAndDependencies$' -count=1
TestBuildConfigStagesLayoutKindAssetsAndDependencies/inline: inline docs dependency missing data-margo-layout-dependency=\"goshtoso-navigation\"
FAIL github.com/araihu/margo/site
```

Artifact-versus-toolbar ownership RED:

```text
$ GOWORK=off go test ./site -run '^TestBuildConfigLayoutKindsPublishDeclaredMarkdownAndPDFWithoutLeakingActions$' -count=1
TestBuildConfigLayoutKindsPublishDeclaredMarkdownAndPDFWithoutLeakingActions: margo-page-actions action UI leaked into landing
FAIL github.com/araihu/margo/site
```

Typed public-route and deterministic-style identity REDs:

```text
TestBuildConfigRendersLayoutKindsWithOwnedChrome: typed Markdown link did not use the public docs route
TestBuildConfigStagesLayoutKindAssetsAndDependencies: asset mode changed typed style identity
FAIL github.com/araihu/margo/site
```

GREEN:

```text
$ GOWORK=off go test ./site -run 'Test(BuildConfig.*(LayoutKinds|Chrome|Assets|Dependencies)|BuildConfigPublishesDeclaredMarkdownAndPDFActions)' -count=1
ok github.com/araihu/margo/site 1.507s

$ GOWORK=off go test ./site -count=1
ok github.com/araihu/margo/site 15.069s

$ GOWORK=off go test ./... -count=1
all packages passed; site 22.082s; cmd/margo 27.714s

$ GOWORK=off go test -race ./site -count=1
ok github.com/araihu/margo/site 57.809s

$ GOWORK=off go vet ./...
(no output)

$ GOWORK=off go mod verify
all modules verified

$ git diff --check
(no output)
```

## Delivered behavior

- The typed layout registry now owns each kind's frame, renderer, and dependency
  metadata. Landing and article use builtin `main`; docs uses builtin
  `top-left-main-right-footer` with the docs frame profile.
- Configured typed pages resolve their frame and values during preflight, then
  dispatch through landing, article, or docs bindings per page. The existing
  profile/frame path remains beside it for Task 8 migration.
- Landing and article bind only the Markdown document. Landing receives one
  landing-owned wrapper/style hook. Neither kind receives site/family
  navigation, sidebar, TOC, breadcrumbs, pagination, docs toolbar, component
  shell, docs interactions, or Goshtoso navigation assets.
- Docs alone binds site navigation, Task 6 family navigation and sidebar,
  configured TOC, family-scoped pagination, optional locale/theme controls, and
  docs footer. Page-action injection and dependencies are docs-owned.
- Local builds stage the union of required typed artifacts once. Each page links
  only its active kind's CSS and scripts. Inline builds stage no layout-runtime
  artifacts and embed only the active kind's dependencies, including the pinned
  Goshtoso navigation runtime for docs.
- Shared, landing, and docs CSS ownership is separate. The typed landing CSS has
  no hide/override selectors for docs chrome.
- Landing/article Markdown and PDF output declarations remain honored without
  rendering or staging docs toolbar markup on those pages.
- Typed canonical URLs, rewritten Markdown links, navigation links, and family
  pagination use public directory routes.
- Typed document-style identity is independent of local versus inline asset
  delivery and includes the active layout dependency union.

## Self-review

- Registry render/dependency metadata participates in configured layout
  identity, so ownership changes invalidate the layout schema identity.
- Typed helpers consume the already-resolved per-page layout and Task 6 family
  index. They do not consult legacy family labels, overview prefixes, or profile
  presentations.
- Semantic Markdown dependencies remain page-specific in both asset modes.
  Local shared layout artifacts are deduplicated by the artifact store.
- Legacy fields, profile branches, styles, and fixtures remain intact by
  controller ruling; no existing assertion was weakened or removed.
- Existing `.impeccable/critique/*` reports remain untracked and untouched.
- The running server on `127.0.0.1:8080` was not stopped or restarted.
- No push, merge, release, publication, or deployment.

## Concerns

None within Task 7. Task 8 still owns migrating/removing the legacy profile
runtime, legacy style assembly, and legacy fixtures together.
