# Layout Cascade and Shell Ownership Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Preserve Margo's no-config Markdown-to-HTML projection while replacing layout profiles and source-prefix families with typed, schema-validated `landing`, `article`, and `docs` layout cascades.

**Architecture:** `site.Build` remains a layout-free projection path. `site.BuildConfig` discovers Markdown and `_layout.yaml` inputs, resolves site, directory, and frontmatter patches through a closed layout registry, preflights all family and rendering state, then dispatches to a kind-owned renderer. Only the `docs` renderer owns family navigation and documentation chrome.

**Tech Stack:** Go, `gopkg.in/yaml.v3`, Margo compiler/render APIs, built-in `ssg.Frame` implementations, Goshtoso components, `chromedp`, standard Go tests.

**Spec:** [`docs/superpowers/specs/2026-08-19-layout-cascade-and-shell-ownership-design.md`](../specs/2026-08-19-layout-cascade-and-shell-ownership-design.md)

## Global Constraints

- Work only in `/Users/guilhermecastro/.codex/worktrees/margo-docs-expansion` on `docs/expansion`.
- Read and obey `AGENTS.md`; keep Caveman full active.
- Preserve unrelated changes and every untracked `.impeccable/critique/*` report.
- Do not push, merge, release, publish, or deploy.
- Use `GOWORK=off` for Go dependency and test truth.
- Add one failing test before each behavior change; run the focused red test, implement the minimum code, then rerun it green.
- Keep directory and page preflight deterministic: sort paths explicitly and emit no HTML before all layouts and families validate.
- Restart `margo serve` after Go changes before browser verification.
- Do not add visual features or compensate with shared-shell breakpoint selectors.

---

## File Map and Interfaces

### New files

- `site/layout.go`
  - Defines `LayoutKind`, `LayoutConfig`, `LayoutPatch`, `ResolvedLayout`, registry entries, schemas, typed cascade state, deep merge, validation, and deterministic identity.
- `site/layout_patch.go`
  - Loads `_layout.yaml`, decodes Markdown `Metadata.Additional["layout"]`, attaches source/pointer diagnostics, and computes root-to-nearest patch chains.
- `site/layout_test.go`
  - Unit tests for registry defaults, closed schemas, merge semantics, kind boundaries, family declaration/selection, and identity.
- `site/layout_patch_test.go`
  - Unit tests for patch parsing, path ordering, duplicate keys, multiple documents, reserved files, and source diagnostics.
- `site/config_layout_test.go`
  - Configuration-loading tests for the target `layout` shape and source-level validation errors.

### Reworked files

- `site/config.go`
  - Replaces `frame`, `shell`, `layouts`, and navigation-owned family declarations with one `layout` field.
  - Retains unrelated site, locale, theme, binding, and publication validation.
- `site/presentation.go`
  - Removes `LayoutProfiles`, `LayoutProfile`, `FamilyConfig`, prefix matching, and profile identity.
  - Keeps general presentation diagnostics and small route helpers still used by layout resolution.
- `site/config_build.go`
  - Discovers Markdown plus patch files, resolves every page layout during preflight, renders by kind, and stages only kind-owned dependencies.
- `site/build.go`
  - Removes profile state from `builder` and ensures `Build` never injects configured layout chrome or its assets.
- `site/site_navigation.go`
  - Makes navigation consume resolved docs families rather than configuration source prefixes.
- `site/goshtoso_shell.go`
  - Removes profile-mode conditions; docs calls its components explicitly.
- `site/page_actions.go`
  - Keeps toolbar implementation reusable, but only configured docs rendering invokes or stages it.
- `site/config_build_test.go`, `site/presentation_test.go`, `site/site_navigation_test.go`, `site/page_actions_test.go`
  - Replace profile fixtures and assertions with typed-layout fixtures and no-leakage assertions.
- `site/layout_profile_browser_test.go`
  - Rename to `site/layout_browser_test.go` and verify landing/docs boundaries at mobile, tablet, and desktop widths.
- `showcase.yaml`
  - Selects `layout.kind: docs`, sets docs defaults, and declares ordered docs families.
- `showcase/content/index.md`
  - Selects `layout.kind: landing` in frontmatter.
- `showcase/content/module/_layout.yaml`, `showcase/content/cli/_layout.yaml`
  - Select docs families without defining them.

### Core types

```go
type LayoutKind string

const (
	LayoutArticle LayoutKind = "article"
	LayoutLanding LayoutKind = "landing"
	LayoutDocs    LayoutKind = "docs"
)

type LayoutConfig struct {
	Kind    LayoutKind     `yaml:"kind"`
	Default map[string]any `yaml:"default"`
	Values  map[string]any `yaml:"values"`
}

type LayoutPatch struct {
	Kind   LayoutKind
	Values map[string]any
	Source string
	Base   string
}

type ResolvedLayout struct {
	Kind       LayoutKind
	Values     map[string]any
	Family     string
	FrameName  string
	Frame      ssg.Frame
	FrameSchema ssg.FrameSchema
	FrameValues ssg.Values
	SchemaHash string
	Identity   string
}
```

`LayoutPatch.Base` is the diagnostic pointer prefix: `/layout` for Markdown,
`/` for `_layout.yaml`, and `/layout` for `site.yaml`.

The internal schema supports the value forms needed by v1:

```go
type layoutValueType uint8

const (
	layoutObject layoutValueType = iota
	layoutBool
	layoutString
	layoutStringList
)

type layoutValueSchema struct {
	Type       layoutValueType
	Properties map[string]layoutValueSchema
	Enum       []string
}
```

Registry-owned closed schemas:

```text
article.values
  content.layout: enum [article]

landing.values
  content.layout: enum [article]

docs.values
  families: array[string]       site default only
  family: string                site/directory/Markdown selection
  sidebar: bool
  toc: bool
  content.layout: enum [article]
```

Built-in defaults:

```text
article: {content: {layout: article}}
landing: {content: {layout: article}}
docs:    {families: [default], family: default, sidebar: true,
          toc: true, content: {layout: article}}
```

`layoutCascade` stores one accumulated value map per kind plus the active kind.
A kind patch activates that bucket; a value-only patch mutates the active
bucket. Each bucket begins with its registry defaults. This preserves values
when switching away and back without leaking values across kinds.

---

## Task 1: Lock the Plain No-config Projection Contract

**Files:**

- Modify: `site/build_test.go`
- Modify: `site/build.go`
- Modify: `site/page_actions_test.go`

- [ ] Replace `TestBuildLocalSitePublishesDeclaredMarkdownActions` with a regression test named `TestBuildLocalSiteProjectsPlainHTMLWithoutLayoutChrome`.

```go
func TestBuildLocalSiteProjectsPlainHTMLWithoutLayoutChrome(t *testing.T) {
	result, err := Build(context.Background(), Request{
		Sources: []Source{{Path: "guide.md", Content: []byte(`---
title: Guide
margo:
  actions:
    markdown: true
---
# Guide

Plain projection.
`)}},
		Compiler: margo.New(), Assets: AssetsLocal,
	})
	if err != nil { t.Fatal(err) }
	page := artifactContent(t, result, "guide.html")
	for _, forbidden := range []string{
		"margo-page-actions", "margo-breadcrumbs", "margo-pagination",
		`id="left-nav"`, `id="right-nav"`, "data-margo-layout",
		"margo-assets/site.css", "goshtoso",
	} {
		if strings.Contains(page, forbidden) { t.Fatalf("plain HTML contains %q: %s", forbidden, page) }
	}
	for _, forbiddenArtifact := range []string{
		pageActionsScriptPath, pageActionsStylePath, pageActionsIconSpritePath,
		"margo-assets/site.css",
	} {
		if artifactExists(result, forbiddenArtifact) { t.Fatalf("unexpected artifact %q", forbiddenArtifact) }
	}
	}
}
```

- [ ] Run the focused test and confirm it fails on page-action markup/assets.

```sh
GOWORK=off go test ./site -run TestBuildLocalSiteProjectsPlainHTMLWithoutLayoutChrome -count=1
```

- [ ] In `site.Build`, stop calling page-action DOM injection and stop staging page-action CSS, JS, and icon sprite. Preserve source retention when frontmatter requests Markdown output.

- [ ] Update no-config page-action tests so toolbar behavior is asserted only through `BuildConfig`; retain tests for Markdown/PDF output artifacts.

- [ ] Run the focused no-config and page-action tests.

```sh
GOWORK=off go test ./site -run 'TestBuild(LocalSiteProjectsPlainHTMLWithoutLayoutChrome|ConfigPublishesDeclaredMarkdownAndPDFActions)' -count=1
```

- [ ] Run all `Build` tests; the configured fixtures remain useful guards against accidental shared helpers.

```sh
GOWORK=off go test ./site -run '^TestBuild' -count=1
```

- [ ] Commit the contract change.

```sh
git add site/build.go site/build_test.go site/page_actions_test.go
git commit -m "fix: keep no-config site output plain"
```

## Task 2: Add the Closed Layout Registry and Typed Cascade

**Files:**

- Create: `site/layout.go`
- Create: `site/layout_test.go`

- [ ] Add table-driven tests proving all three registry defaults validate through the public patch validator.

```go
func TestLayoutRegistryDefaultsValidate(t *testing.T) {
	for _, kind := range []LayoutKind{LayoutArticle, LayoutLanding, LayoutDocs} {
		t.Run(string(kind), func(t *testing.T) {
			entry, ok := builtinLayoutRegistry().lookup(kind)
			if !ok { t.Fatalf("missing kind %q", kind) }
			if _, err := entry.validateValues(entry.defaults, layoutValueSiteDefault, "/layout/default"); err != nil {
				t.Fatal(err)
			}
		})
	}
}
```

- [ ] Add failure tests for unknown kind, unknown value, wrong scalar type, wrong array element type, and invalid enum. Assert diagnostic code and pointer.

```go
func TestLayoutRegistryRejectsUnknownValue(t *testing.T) {
	_, err := resolveSiteLayout(LayoutConfig{
		Kind: LayoutDocs,
		Default: map[string]any{"sidebaar": true},
	}, "site.yaml")
	requirePresentationDiagnostic(t, err, "site.layout_value_unknown", "site.yaml", "/layout/default/sidebaar")
}
```

- [ ] Add deep-merge tests proving nested maps merge, scalars replace, and arrays replace completely.

```go
func TestLayoutValuesMergeMapsAndReplaceArrays(t *testing.T) {
	base := map[string]any{
		"families": []any{"default", "module"},
		"content": map[string]any{"layout": "article"},
	}
	patch := map[string]any{
		"families": []any{"cli"},
		"sidebar": false,
	}
	got := mergeLayoutValues(base, patch)
	assertLayoutValues(t, got, map[string]any{
		"families": []any{"cli"}, "sidebar": false,
		"content": map[string]any{"layout": "article"},
	})
}
```

- [ ] Add a typed-boundary test for `docs -> landing -> docs` proving landing never sees docs values and the second docs selection retrieves its earlier bucket.

- [ ] Implement `LayoutKind`, registry entries, schema validator, normalized map/list copying, deep merge, `layoutCascade`, and deterministic canonical-value hashing in `site/layout.go`.

- [ ] Use sorted property names when validating and hashing map values. Do not depend on Go map iteration.

- [ ] Run the registry tests.

```sh
GOWORK=off go test ./site -run 'TestLayout(Registry|Values|Cascade)' -count=1
```

- [ ] Commit the registry and cascade.

```sh
git add site/layout.go site/layout_test.go
git commit -m "feat: add typed layout registry"
```

## Task 3: Replace Profile Configuration with One Layout Selection

**Files:**

- Modify: `site/config.go`
- Create: `site/config_layout_test.go`
- Modify: `site/presentation.go`
- Modify: `site/presentation_test.go`

- [ ] Add config-loading tests for the target YAML shape and its normalized effective docs defaults.

```go
func TestLoadConfigAcceptsTypedLayoutSelection(t *testing.T) {
	root := t.TempDir()
	filename := filepath.Join(root, "site.yaml")
	writeConfigFile(t, filename, `version: 1
source: docs
output: dist
site:
  name: Example
  logo: assets/logo.svg
  icon: assets/icon.svg
  social_image:
    path: assets/social.png
    alt: Example documentation preview
  home: index.md
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
`)
	config, err := LoadConfig(filename)
	if err != nil { t.Fatal(err) }
	if config.Layout.Kind != LayoutDocs { t.Fatalf("kind = %q", config.Layout.Kind) }
}
```

- [ ] Add config diagnostics for unknown layout kind, unknown default/value key, invalid value type, duplicate family, and empty family identifier. Each must point into `site.yaml` under `/layout/...`.

- [ ] Change `Config` to expose `Layout LayoutConfig` with YAML tag `layout`, and remove `Frame`, `Shell`, `Layouts`, `layoutsPresent`, and `Navigation.Families`.

- [ ] Keep `Navigation.Mode` and `Navigation.Exclude`; family declarations live only at `layout.default.families`.

- [ ] Replace `validateLayoutProfiles` and `validateFamilyConfigs` with `validateSiteLayout`, using the same registry validator as runtime patches.

- [ ] Normalize the docs family list in declared order, implicitly insert `default` once, and reject duplicates after trimming. Preserve the configured order of non-default families.

- [ ] Remove `LayoutProfiles`, `LayoutProfile`, `FamilyConfig`, `resolveFamily`, `resolveLayout`, `configUsesLayoutProfiles`, `prepareFramePresentations`, and `profileLayoutIdentity` from `site/presentation.go`.

- [ ] Retain `newPresentationDiagnostic`, pointer/source attachment helpers, locale route helpers, and any code still shared by the new resolver.

- [ ] Update config and presentation tests to assert the new diagnostics. Delete tests whose only contract was source-prefix family inference or `margo.site.layout` profile selection.

- [ ] Run focused configuration tests.

```sh
GOWORK=off go test ./site -run 'Test(LoadConfig|LayoutConfig|PresentationDiagnostic)' -count=1
```

- [ ] Commit the configuration migration.

```sh
git add site/config.go site/config_layout_test.go site/presentation.go site/presentation_test.go
git commit -m "refactor: make layout selection site owned"
```

## Task 4: Discover and Validate Directory Layout Patches

**Files:**

- Create: `site/layout_patch.go`
- Create: `site/layout_patch_test.go`
- Modify: `site/config_build.go`

- [ ] Add a discovery test with root, `module`, and `module/advanced` patch files. Assert normalized sorted paths and that none is returned as a Markdown source.

```go
func TestDiscoverConfiguredInputsSeparatesLayoutPatches(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "index.md"), "# Home\n")
	writeConfigFile(t, filepath.Join(root, "_layout.yaml"), "values:\n  toc: false\n")
	writeConfigFile(t, filepath.Join(root, "module", "_layout.yaml"), "values:\n  family: module\n")
	inputs, err := discoverConfiguredInputs(context.Background(), root, nil)
	if err != nil { t.Fatal(err) }
	assertSourcePaths(t, inputs.Sources, []string{"index.md"})
	assertPatchPaths(t, inputs.Patches, []string{"_layout.yaml", "module/_layout.yaml"})
}
```

- [ ] Add parser tests rejecting a symlink, duplicate YAML key, multiple documents, non-mapping root, `default`, unknown root property, invalid kind, unknown value, and directory-owned `families`.

- [ ] Assert diagnostics use the patch filename as `Source` and exact pointers such as `/values/sidebar` and `/default`.

- [ ] Implement `configuredInputs`, change the source walk to recognize only exact `_layout.yaml` basenames, and reject symlinked patch files explicitly instead of skipping them silently.

- [ ] Decode patch YAML through `yaml.Node` before typed conversion so duplicate keys and pointer positions remain detectable. Require one document and a mapping root.

- [ ] Parse only `kind` and `values`; reject `default` in directory patches with `site.layout_patch_invalid`.

- [ ] Build `layoutPatchChain(sourcePath, patches)` from `.` through each ancestor directory. Sort normalized patch paths once, then return only exact ancestor matches.

- [ ] Run patch tests.

```sh
GOWORK=off go test ./site -run 'Test(DiscoverConfiguredInputs|DirectoryLayoutPatch|LayoutPatchChain)' -count=1
```

- [ ] Commit directory patch support.

```sh
git add site/layout_patch.go site/layout_patch_test.go site/config_build.go
git commit -m "feat: discover directory layout patches"
```

## Task 5: Parse Frontmatter Patches and Resolve Every Page

**Files:**

- Modify: `site/layout_patch.go`
- Modify: `site/layout_patch_test.go`
- Modify: `site/config_build.go`
- Modify: `site/config_build_test.go`
- Modify: `site/build.go`

- [ ] Add frontmatter tests for a kind-only patch, values-only patch, unknown root property, unknown value, directory/Markdown family declaration, and non-mapping `layout` value.

```go
func TestFrontmatterLayoutPatchUsesMarkdownSourcePointers(t *testing.T) {
	metadata := margo.Metadata{Additional: map[string]any{
		"layout": map[string]any{"values": map[string]any{"family": "missing"}},
	}}
	patch, err := layoutPatchFromMetadata(metadata, "guide.md")
	if err != nil { t.Fatal(err) }
	if patch.Source != "guide.md" || patch.Base != "/layout" { t.Fatalf("patch = %+v", patch) }
}
```

- [ ] Implement `layoutPatchFromMetadata`; leave generic Markdown parsing unchanged and read only `Metadata.Additional["layout"]` in `site`.

- [ ] Add an integration test with site docs defaults, a root directory patch, a nested directory patch, and a Markdown patch. Assert site < root < nearest < Markdown precedence.

- [ ] Add an integration test for `docs -> landing` at root `index.md` and `docs -> landing -> docs` in a nested page. Assert kind-specific values do not leak.

- [ ] Replace `configuredPage.presentation` with `configuredPage.layout ResolvedLayout`.

- [ ] In `preflightConfigured`, compile the document, resolve site layout, apply the page's ordered directory chain, apply the frontmatter patch, validate the effective values, then store kind/layout identity on the page.

- [ ] Set `Page.Layout` for all configured pages. Set `Page.Family` only when kind is `docs`; leave it empty for `landing` and `article`.

- [ ] Include ordered patch sources and normalized effective layout values in per-page identity. Fold sorted per-page identities into `SiteManifest.LayoutSchemaHash` only after all pages preflight.

- [ ] Reorder `buildConfigured`: index sources, preflight all pages/layouts/families, validate home, stage required assets, then render. No layout-dependent asset or HTML may be emitted before preflight succeeds.

- [ ] Add an assertion that an invalid final Markdown patch returns zero artifacts.

- [ ] Run focused cascade/preflight tests.

```sh
GOWORK=off go test ./site -run 'Test(BuildConfig.*(Cascade|Frontmatter|KindBoundary|Preflight)|FrontmatterLayoutPatch)' -count=1
```

- [ ] Commit page layout resolution.

```sh
git add site/layout_patch.go site/layout_patch_test.go site/config_build.go site/config_build_test.go site/build.go
git commit -m "feat: resolve layouts per configured page"
```

## Task 6: Make Family Context Docs-only and Deterministic

**Files:**

- Modify: `site/layout.go`
- Modify: `site/layout_test.go`
- Modify: `site/config_build.go`
- Modify: `site/config_build_test.go`
- Modify: `site/site_navigation.go`
- Modify: `site/site_navigation_test.go`

- [ ] Add unit tests proving `default` always exists, undeclared explicit family fails at the selecting source, and directory/Markdown patches cannot set `families`.

- [ ] Add preflight tests proving a declared explicit family with no docs page fails with `site.family_empty`, while an unused implicit `default` family is valid.

- [ ] Add family order tests proving the effective docs family list follows central configuration order and never source traversal order.

- [ ] During completed page preflight, build `docsFamilies` from docs pages only. Track locale plus family and select each overview deterministically: `index.md` first, then route order.

- [ ] Validate every non-default configured family has at least one docs page. Attach the error to `site.yaml` at `/layout/default/families/<index>`.

- [ ] Replace `familyConfig`, `familyOverviewPage`, and source-prefix family logic in `site/site_navigation.go` with lookups over the preflighted docs-family index.

- [ ] Keep `familyPages` restricted to pages whose resolved kind is `docs`, locale matches, and family matches.

- [ ] Render no secondary family navbar when the effective family list has one entry. When multiple families have docs pages, render links in configured order with exactly one `aria-current="location"`.

- [ ] Keep pagination family-scoped and suppress it for families with fewer than two pages.

- [ ] Run focused family tests.

```sh
GOWORK=off go test ./site -run 'Test(Layout.*Family|BuildConfig.*Family|SiteNavigation.*Family)' -count=1
```

- [ ] Commit docs-family ownership.

```sh
git add site/layout.go site/layout_test.go site/config_build.go site/config_build_test.go site/site_navigation.go site/site_navigation_test.go
git commit -m "refactor: keep family context inside docs layout"
```

## Task 7: Dispatch Rendering and Assets by Layout Kind

**Files:**

- Modify: `site/config_build.go`
- Modify: `site/build.go`
- Modify: `site/goshtoso_shell.go`
- Modify: `site/site_navigation.go`
- Modify: `site/page_actions.go`
- Modify: `site/config_build_test.go`
- Modify: `site/page_actions_test.go`

- [ ] Add one configured fixture containing landing, article, and docs pages. Assert all three keep Markdown-generated article content.

- [ ] Add exact negative assertions for landing and article:

```go
for _, forbidden := range []string{
	`data-margo-family-navigation`, `id="left-nav"`, `id="right-nav"`,
	"margo-breadcrumbs", "margo-pagination", "margo-page-actions",
	"data-toc-heading", "component-doc-shell",
} {
	if strings.Contains(page, forbidden) { t.Fatalf("%s leaked into %s", forbidden, route) }
}
```

- [ ] Add positive docs assertions for site navigation, family navigation when multiple families exist, configured sidebar, configured TOC, page actions, and family-scoped pagination.

- [ ] Implement registry render entries:

```text
landing -> builtin frame main; article binding only; landing-owned CSS hook
article -> builtin frame main; article binding only
docs    -> builtin frame top-left-main-right-footer; docs bindings
```

- [ ] Replace `profileMode`, `presentations`, global `frame`, and layout-name branching in `builder` with per-page `ResolvedLayout` dispatch.

- [ ] Split `bindingsForPage` into `landingBindings`, `articleBindings`, and `docsBindings`. The first two add only `document/main-content`; `docsBindings` alone adds navigation, sidebar, TOC, pagination, footer, and page actions.

- [ ] Make `renderPageHead` accept the resolved layout. Always render semantic document requirements, but add configured site CSS, docs interaction scripts, Goshtoso navigation dependencies, and page-action dependencies only when the active registry entry owns them.

- [ ] Replace `configuredSiteStylesheet(profileMode bool)` with kind-owned CSS assembly. Delete selectors whose only purpose is hiding docs chrome inside landing.

- [ ] Remove `profileMode` conditionals in `site/goshtoso_shell.go`; invoke Goshtoso shell helpers only from the docs renderer.

- [ ] Stage shared assets once if at least one resolved page requires them. For inline mode, embed only dependencies required by that page's layout.

- [ ] Run rendering and asset tests.

```sh
GOWORK=off go test ./site -run 'Test(BuildConfig.*(LayoutKinds|Chrome|Assets|Dependencies)|BuildConfigPublishesDeclaredMarkdownAndPDFActions)' -count=1
```

- [ ] Run the full `site` package to expose stale profile assumptions.

```sh
GOWORK=off go test ./site -count=1
```

- [ ] Commit renderer separation.

```sh
git add site/config_build.go site/build.go site/goshtoso_shell.go site/site_navigation.go site/page_actions.go site/config_build_test.go site/page_actions_test.go
git commit -m "refactor: render chrome by layout kind"
```

## Task 8: Remove Legacy Profile Contracts and Migrate Showcase

**Files:**

- Modify: `showcase.yaml`
- Modify: `showcase/content/index.md`
- Create: `showcase/content/module/_layout.yaml`
- Create: `showcase/content/cli/_layout.yaml`
- Modify: `site/config_build_test.go`
- Modify: `site/presentation_test.go`
- Modify: `site/site_navigation_test.go`
- Modify: `README.md`
- Modify: `docs/unified-model.md`

- [ ] Replace showcase profile/family configuration with:

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

- [ ] Add `layout.kind: landing` to `showcase/content/index.md` frontmatter without changing Tour content.

- [ ] Add `showcase/content/module/_layout.yaml` and `showcase/content/cli/_layout.yaml` selecting `module` and `cli` respectively.

- [ ] Update the showcase publication-contract test to assert:

```text
index.md        -> layout landing, family empty
module/index.md -> layout docs, family module
cli/index.md    -> layout docs, family cli
```

- [ ] Assert the build emits no artifact named `_layout.yaml` or ending in `/_layout.yaml`.

- [ ] Delete old tests and helpers that mention `layouts.profiles`, `navigation.families`, source-prefix inference, or `margo.site.layout`. Replace any still-useful behavior with typed-layout assertions.

- [ ] Update README and unified-model configuration examples to document `layout.kind/default/values`, `_layout.yaml`, frontmatter patches, merge rules, and docs-only family ownership.

- [ ] Search for stale public examples and identifiers.

```sh
rg -n 'layouts:|layouts\.profiles|navigation\.families|margo\.site\.layout|profileMode|LayoutProfiles|FamilyConfig' --glob '!docs/superpowers/specs/**' --glob '!docs/superpowers/plans/**' .
```

- [ ] Remove or migrate every runtime/public-doc hit. Historical design text may remain only where explicitly described as replaced behavior.

- [ ] Build the showcase twice and compare results through the deterministic integration test.

```sh
GOWORK=off go test ./site -run TestBuildConfiguredShowcasePublicationContract -count=2
```

- [ ] Commit showcase and public documentation migration.

```sh
git add showcase.yaml showcase/content/index.md showcase/content/module/_layout.yaml showcase/content/cli/_layout.yaml site/config_build_test.go site/presentation_test.go site/site_navigation_test.go README.md docs/unified-model.md
git commit -m "docs: migrate showcase to layout cascade"
```

## Task 9: Replace Browser Acceptance with Layout-boundary Checks

**Files:**

- Rename: `site/layout_profile_browser_test.go` -> `site/layout_browser_test.go`
- Modify: `site/layout_browser_test.go`
- Modify: browser fixture helpers under `site/*_browser_test.go` only as required

- [ ] Rename the browser test and fixture names from profile terminology to layout terminology.

- [ ] Update its fixture to use the approved `layout` config and `_layout.yaml` patches.

- [ ] Test these routes at 390px mobile, 820px tablet, and 1440px desktop widths:

```text
/            landing; no family/docs chrome; one-column content
/module/     docs; module family; sidebar/TOC behavior follows values
/cli/        docs; cli family; sidebar/TOC behavior follows values
```

- [ ] For Tour, query absence of family links, docs sidebar, TOC area, breadcrumbs, pagination, docs page actions, and docs shell data attributes. Assert article width never overflows the viewport.

- [ ] For Module and CLI, assert the active family link, family-scoped sidebar, TOC at supported widths, and pagination when the family has multiple pages.

- [ ] Assert normal navigation between Module and CLI updates the active family without imposing docs state on Tour.

- [ ] Run the browser acceptance test with an installed Chromium-family browser.

```sh
GOWORK=off go test ./site -run TestLayoutBrowser -count=1 -v
```

- [ ] If Go code changed after the last serve start, stop the stale server, rebuild, and restart `margo serve` before any manual browser inspection.

```sh
GOWORK=off go run ./cmd/margo serve ./showcase.yaml
```

- [ ] Manually inspect Tour, Module, and CLI at the same three widths. Record only failures that are architectural or regressions; do not add visual features.

- [ ] Commit browser acceptance updates.

```sh
git add site/layout_browser_test.go
git commit -m "test: cover layout boundaries in browsers"
```

## Task 10: Final Review and Verification

**Files:**

- Review: all files changed since `0ccff86`
- Preserve: `.impeccable/critique/*`

- [ ] Inspect worktree ownership before final checks.

```sh
git status --short
git diff --stat 0ccff86..HEAD
```

- [ ] Confirm every untracked `.impeccable/critique/*` report remains present and untouched.

- [ ] Search for forbidden leakage and stale architecture identifiers.

```sh
rg -n 'profileMode|LayoutProfiles|LayoutProfile|resolveFamily|margo\.site\.layout|navigation\.families' site showcase.yaml showcase/content README.md docs/unified-model.md
rg -n 'margo-page-actions|margo-breadcrumbs|margo-pagination|data-margo-family-navigation' site/build_test.go
```

- [ ] Run the full test suite.

```sh
GOWORK=off go test ./... -count=1
```

- [ ] Run the race-enabled site package.

```sh
GOWORK=off go test -race ./site -count=1
```

- [ ] Run static analysis.

```sh
GOWORK=off go vet ./site
```

- [ ] Verify module content.

```sh
GOWORK=off go mod verify
```

- [ ] Check whitespace errors.

```sh
git diff --check 0ccff86..HEAD
```

- [ ] Run the showcase browser acceptance once more against a freshly restarted server.

```sh
GOWORK=off go test ./site -run TestLayoutBrowser -count=1 -v
```

- [ ] Review the final diff for unrelated edits, partial artifacts, regenerated files, and accidental `.impeccable` staging.

```sh
git status --short
git diff --name-status 0ccff86..HEAD
git diff --check
```

- [ ] If verification found a defect, add the smallest failing regression test, fix it, rerun the affected focused test, then rerun all final gates.

- [ ] Stop when local implementation and verification are complete. Do not push, merge, release, publish, or deploy.
