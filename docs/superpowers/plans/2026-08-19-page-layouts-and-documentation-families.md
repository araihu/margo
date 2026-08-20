# Margo Page Layouts and Documentation Families Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add allowlisted semantic page layouts and documentation families, then restructure the Margo showcase into Tour, Module, and CLI routes with shared Margo-owned chrome.

**Architecture:** Preserve legacy single-frame and componentdocshell site modes. Add an opt-in layout-profile mode in `site/` that resolves a frame and family per Markdown page, binds a shared Goshtoso navbar plus family-local navigation to the selected SSG frame, and records resolved identity in route manifests. Add `margo.site.layout` as a site-only document preference; page content never selects raw shells or executable frame distributions.

**Tech Stack:** Go 1.26+, Margo Markdown/frontmatter compiler, JSON Schema v1, `ssg` frame contracts, Goshtoso `navbar`/`sidebar` components, YAML site config, `templ`, `chromedp` browser tests.

**Spec:** `docs/superpowers/specs/2026-08-19-page-layouts-and-documentation-families-design.md`

## Global Constraints

- Existing sites without `layouts` and `navigation.families` retain current behavior.
- `layouts` is mutually exclusive with top-level `frame` and `shell`.
- Page layout values are semantic allowlisted profile names; no arbitrary commands, Go modules, shells, or raw frames.
- Family matching is segment-aware and uses the most-specific locale-independent source prefix.
- Tour is one Markdown-generated landing page with no sidebar, TOC, breadcrumbs, pagination, or page-action toolbar.
- Module and CLI use docs layout and family-local navigation; no empty chapter pages are added.
- Removed Tour feature routes produce no artifacts and return 404.
- Global navbar/search/sitemap/llms discovery includes every configured family.
- No App Shell private selector, DOM mutation, or route-specific CSS workaround.
- Preflight resolves families, layouts, frames, schemas, and bindings before artifact materialization.
- Generated output is never hand-edited; use `GOWORK=off` for Go/module truth.
- No push, merge, release, tag, deployment, or external dependency release is implied.

---

### Task 1: Add the closed `margo.site.layout` document preference

**Files:**
- Modify: `schema/v1/document.json` in the closed `margo` namespace.
- Modify: `metadata.go` (`DocumentPreferences`, clone logic).
- Modify: `markdown.go` (`normalizeSourceMetadata`).
- Test: `frontmatter_test.go` and `editorial_metadata_test.go`.

**Interfaces:**
- Produces `margo.Metadata.Margo.Site.Layout string` (empty when omitted).
- Keeps `margo.page` and `margo.actions` parsing unchanged.
- Rejects unknown nested `margo.site` properties through the existing JSON Schema validator.

- [ ] **Step 1: Write failing metadata tests**

Add a valid-frontmatter case:

```go
func TestEditorialFrontmatterAcceptsSiteLayoutPreference(t *testing.T) {
	doc, err := New().Compile(context.Background(), Source{
		Name: "landing.md",
		Content: []byte("---\nmargo:\n  site:\n    layout: landing\n---\n# Landing\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.Metadata().Margo.Site.Layout; got != "landing" {
		t.Fatalf("layout = %q, want landing", got)
	}
}
```

Add an unknown-field case asserting `frontmatter.schema_invalid` and a
`/margo/site` pointer. Add a non-string `layout` case with the same diagnostic
family. Assert cloning retains the value without sharing mutable state.

- [ ] **Step 2: Run focused tests and verify failure**

Run:

```sh
GOWORK=off go test ./... -run 'TestEditorialFrontmatterAcceptsSiteLayoutPreference|TestFrontmatterRejectsUnknownMargoSiteField' -count=1
```

Expected: FAIL because `DocumentPreferences` has no `Site` field and the schema
rejects the new property.

- [ ] **Step 3: Implement the schema and metadata projection**

Add a closed `site` object beneath `margo` with a string `layout` property,
document its target as `site`, and add:

```go
type SitePreference struct {
	Layout string
}

type DocumentPreferences struct {
	Page    *PagePreference
	Actions *PageActions
	Site    *SitePreference
}
```

Parse `margo.site.layout` only when the value is a string after schema
validation. Clone the pointer in `Metadata.clone`. Keep non-Margo root metadata
in `Additional` exactly as before.

- [ ] **Step 4: Run focused tests and verify success**

Run:

```sh
GOWORK=off go test ./... -run 'TestEditorialFrontmatterAcceptsSiteLayoutPreference|TestFrontmatterRejectsUnknownMargoSiteField|TestEditorialFrontmatter' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```sh
git add schema/v1/document.json metadata.go markdown.go frontmatter_test.go editorial_metadata_test.go
git commit -m "feat: add site layout document preference"
```

### Task 2: Extend SSG frame schemas with global site navigation

**Files:**
- Modify: `ssg/builtin.go` for every built-in frame containing `top-nav`.
- Modify: `ssg/validate.go` only if profile validation needs a separate global-navigation requirement.
- Test: `ssg/builtin_test.go` and the relevant validation test file.

**Interfaces:**
- Adds payload kind `site_navigation`.
- `top-nav.Accepts` includes `site_navigation`; `left-nav` does not.
- Existing `navigation` defaults remain unchanged for legacy callers.
- Layout-profile mode can bind both `site_navigation` at `top-nav` and `navigation` at `left-nav` in docs frames.

- [ ] **Step 1: Write failing frame contract tests**

Add a test that creates a `site_navigation` binding for `top-main` and renders
it in `top-nav`. Add a `top-left-main-right-footer` test with both payload kinds
in their respective areas. Add a negative test placing `site_navigation` in
`left-nav` and assert the area error names the invalid kind/area.

Example binding setup:

```go
siteNav, err := NewAreaBinding(hash, "index.html", BindingSpec{
	Kind: "site_navigation", Area: "top-nav",
}, 0, templ.Raw(`<nav aria-label="main navigation">site</nav>`))
if err != nil {
	t.Fatal(err)
}
```

- [ ] **Step 2: Run focused SSG tests and verify failure**

Run:

```sh
GOWORK=off go test ./ssg -run 'TestBuiltin.*Navigation|TestValidateBindings.*Navigation' -count=1
```

Expected: FAIL because `site_navigation` is not accepted by built-in schemas.

- [ ] **Step 3: Implement the new payload kind**

Update the `top` `AreaDescriptor` in `ssg/builtin.go` to accept
`site_navigation`, add it to the `top-nav` binding order, and leave the
`BindingDefaults` map pointing legacy `navigation` to its existing area. Do not
make `site_navigation` a default payload for old frame builds.

- [ ] **Step 4: Run SSG tests and the existing frame suite**

Run:

```sh
GOWORK=off go test ./ssg -count=1
```

Expected: PASS, including stable schema/hash tests.

- [ ] **Step 5: Commit**

```sh
git add ssg/builtin.go ssg/validate.go ssg/builtin_test.go
git commit -m "feat: publish global site navigation binding"
```

### Task 3: Model and validate layout profiles and navigation families

**Files:**
- Modify: `site/config.go` (`Config`, `NavigationConfig`, validation, binding-kind allowlist).
- Create: `site/presentation.go` for pure profile/family types and resolution helpers.
- Test: `site/presentation_test.go`.

**Interfaces:**

Define the pure resolver surface before wiring the builder:

```go
type LayoutProfiles struct {
	Default  string                   `yaml:"default"`
	Profiles map[string]LayoutProfile `yaml:"profiles"`
}

type LayoutProfile struct {
	Frame LayoutSelection `yaml:"frame"`
}

type FamilyConfig struct {
	ID       string `yaml:"id"`
	Label    string `yaml:"label"`
	Source   string `yaml:"source"`
	Overview string `yaml:"overview"`
	Layout   string `yaml:"layout"`
}

type PagePresentation struct {
	FamilyID   string
	LayoutName string
	FrameName  string
	Frame      ssg.Frame
	Schema     ssg.FrameSchema
	Values     ssg.Values
	SchemaHash string
}

func resolveFamily(source string, locales LocaleConfig, families []FamilyConfig) (FamilyConfig, error)
func resolveLayout(metadata margo.Metadata, family FamilyConfig, layouts LayoutProfiles) (string, error)
```

`resolveFamily` performs segment-aware longest-prefix matching after
`sourceLocale` stripping. `resolveLayout` applies page, family, then site
default precedence and rejects empty/unknown names.

- [ ] **Step 1: Write failing resolver and config-validation tests**

Cover:

```go
func TestResolveFamilyUsesMostSpecificSegmentPrefix(t *testing.T) {
	families := []FamilyConfig{
		{ID: "tour", Source: "."},
		{ID: "cli", Source: "cli"},
	}
	got, err := resolveFamily("cli/index.md", LocaleConfig{Default: "en", Supported: []string{"en"}}, families)
	if err != nil || got.ID != "cli" {
		t.Fatalf("family = %+v, err = %v", got, err)
	}
	if got, err := resolveFamily("client.md", LocaleConfig{Default: "en", Supported: []string{"en"}}, families); err != nil || got.ID != "tour" {
		t.Fatalf("client fallback = %+v, err = %v", got, err)
	}
}
```

Add page-over-family-over-default precedence tests, duplicate ID/root tests,
missing overview/out-of-root tests, invalid family path tests, unknown profile
tests, and a `layouts` + top-level `frame`/`shell` conflict test. Assert
diagnostic codes and field pointers.

- [ ] **Step 2: Run focused tests and verify failure**

Run:

```sh
GOWORK=off go test ./site -run 'TestResolveFamily|TestResolveLayout|TestValidate.*Layout|TestValidate.*Family' -count=1
```

Expected: FAIL because the new YAML fields and resolver functions do not exist.

- [ ] **Step 3: Add configuration types and validation**

Add `Layouts LayoutProfiles` to `site.Config`, `Families []FamilyConfig` to
`NavigationConfig`, and validate only when `layouts` or families are present.
Reject layout-profile mode with top-level `Frame` or `Shell`; require a
non-empty default profile and one built-in frame selection per profile; call
the existing `validateLayoutSelection` with `shell=false` for profile frames.

Extend `knownBindingKind` with `site_navigation`. Normalize family roots with
the existing path rules, reject absolute/parent/empty roots, and validate
overview paths as normalized Markdown paths. Preserve legacy defaults when both
new fields are absent.

- [ ] **Step 4: Implement pure family/layout resolution**

Use locale-independent source paths and segment boundaries. Return one
diagnostic when no family matches, one when a selected layout is unknown, and
include the source/profile name in the hint. Keep family declaration order
available for navbar ordering; use source specificity only for lookup.

- [ ] **Step 5: Run focused and existing config tests**

Run:

```sh
GOWORK=off go test ./site -run 'TestResolveFamily|TestResolveLayout|TestValidate.*Layout|TestValidate.*Family|TestLoadConfigRejectsUnknownKeys' -count=1
GOWORK=off go test ./site -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```sh
git add site/config.go site/presentation.go site/presentation_test.go
git commit -m "feat: validate site layout profiles and families"
```

### Task 4: Resolve per-page frames and record presentation identity

**Files:**
- Modify: `site/build.go` (`Page`, `SiteManifest`, `builder` presentation state).
- Modify: `site/config_build.go` (`buildConfigured`, `preflightConfigured`, `renderConfiguredSource`).
- Modify: `site/presentation.go` with frame-profile preparation helpers.
- Test: `site/config_build_test.go` and `site/presentation_test.go`.

**Interfaces:**

Add manifest fields:

```go
type Page struct {
	// existing fields...
	Family string `json:"family,omitempty"`
	Layout string `json:"layout,omitempty"`
}
```

Add a builder map keyed by profile name and keep the existing single `frame`,
`frameSchema`, `frameValues`, and `frameHash` fields for legacy mode. In profile
mode, each prepared page carries a `PagePresentation` and render uses its
schema/hash/values.

- [ ] **Step 1: Write failing configured-build tests**

Create a temporary site with `index.md`, `module/index.md`, and `cli/index.md`
plus the approved `layouts`/`navigation.families` YAML. Assert each manifest
route has expected `Family`/`Layout`, repeated `BuildConfig` results are deeply
equal, and the landing/docs frame identities differ. Add a page frontmatter
override test and a preflight test proving an invalid selected profile emits no
HTML artifact.

- [ ] **Step 2: Run focused tests and verify failure**

Run:

```sh
GOWORK=off go test ./site -run 'TestBuildConfig.*Layout|TestBuildConfig.*Family|TestBuildConfig.*Presentation' -count=1
```

Expected: FAIL because `buildConfigured` currently creates one global frame and
`Page` has no presentation identity.

- [ ] **Step 3: Prepare profile frames before page preflight**

Add a helper that iterates declared profiles in sorted name order, resolves the
built-in frame, requests a schema with the configured theme/locale, validates it
under `ssg.DocsProfile`, resolves profile frame values, and computes
`SchemaHashForValues`. Store the result by profile name. Do not render or stage
partial artifacts until all profiles validate.

- [ ] **Step 4: Resolve each prepared page**

In `preflightConfigured`, after compiling metadata and before appending the
page, call `resolveFamily` and `resolveLayout`, attach `Family` and `Layout` to
the page, and retain the selected `PagePresentation` in `configuredPage`. For
legacy mode, retain current single-frame behavior and leave new fields empty.

- [ ] **Step 5: Render with the selected presentation**

Change the profile-mode branch of `renderConfiguredSource` to pass the selected
frame, schema hash, values, and per-page bindings into `FrameInput`. Keep the
componentdocshell branch untouched. Include family/layout identity in the site
manifest layout identity and per-route manifest output.

- [ ] **Step 6: Run focused and full site tests**

Run:

```sh
GOWORK=off go test ./site -run 'TestBuildConfig.*Layout|TestBuildConfig.*Family|TestBuildConfig.*Presentation' -count=1
GOWORK=off go test ./site -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit**

```sh
git add site/build.go site/config_build.go site/presentation.go site/config_build_test.go site/presentation_test.go
git commit -m "feat: resolve frames per configured page layout"
```

### Task 5: Render Margo-owned global and family-local navigation

**Files:**
- Create: `site/site_navigation.go` for Goshtoso navbar/search/global-family fragments.
- Modify: `site/config_build.go` (`bindingsForPage`, pagination/sidebar selection).
- Modify: `site/build.go` or the configured asset staging path for Goshtoso component assets required by navbar/sidebar/search.
- Modify: `site/config_build.go` stylesheet constants for semantic layout hooks and landing/docs frame presentation.
- Test: `site/config_build_test.go`.

**Interfaces:**

Define consumer-owned fragments with explicit presentation input:

```go
func (b *builder) siteNavigationFragment(page Page) (string, error)
func (b *builder) familyNavigationFragment(page Page) (string, error)
func (b *builder) bindingsForPage(prepared configuredPage) (map[string][]ssg.AreaBinding, error)
```

`bindingsForPage` must select `site_navigation` for profile-mode `top-nav`,
`navigation` for docs-mode `left-nav`, and omit local navigation for landing.
Legacy mode continues using the existing `navigationFragment` path.

- [ ] **Step 1: Write failing markup tests**

Build the three-page fixture and assert:

- global navbar contains Tour, Module, CLI in configured order;
- exactly one family link has `aria-current="location"` per page;
- Tour has no `left-nav` binding or sidebar markup;
- Module/CLI local navigation contains only the active family overview;
- docs pagination excludes pages from other families; and
- global search result markup includes all three page routes.

- [ ] **Step 2: Run focused tests and verify failure**

Run:

```sh
GOWORK=off go test ./site -run 'TestBuildConfigRenders.*Family|TestBuildConfigRenders.*Layout|TestBuildConfig.*Navigation' -count=1
```

Expected: FAIL because profile-mode bindings currently call the flat
`navigationFragment` and do not render Goshtoso navbar/sidebar components.

- [ ] **Step 3: Implement global navbar and search fragments**

Use public `navbar.Navbar`, `navbar.SecondaryConfig`, `navbar.SecondaryLink`,
and the existing search component configuration. Build root-relative links with
the same `base_path` and `relativeAssetPath` helpers used by current pages.
Set the current family link by location, preserve repository/theme controls,
and keep the search index global. Stage only public Goshtoso assets required by
these components through the existing asset mechanism.

- [ ] **Step 4: Implement active-family sidebar and scoped pagination**

Construct `sidebar.Item` values from pages matching `page.Family` and
`page.Locale`, with the family overview first and deterministic remaining
order. Replace the current locale-only pagination candidate list with the same
family filter in profile mode. Keep current legacy pagination unchanged.

- [ ] **Step 5: Add semantic layout styling**

Add Margo-owned selectors for `data-margo-layout="landing"` and
`data-margo-layout="docs"`. Landing styles must make the article full-width
within the selected frame and must not hide or target App Shell-private DOM.
Docs styles retain readable article measure, sidebar spacing, TOC rail, focus
states, and responsive behavior. Scope controls and links under Margo-owned
frame/layout classes.

- [ ] **Step 6: Run focused, asset, and full site tests**

Run:

```sh
GOWORK=off go test ./site -run 'TestBuildConfigRenders.*Family|TestBuildConfigRenders.*Layout|TestBuildConfig.*Navigation' -count=1
GOWORK=off go test ./site -count=1
```

Expected: PASS with no missing local Goshtoso assets and no private App Shell
selectors in profile-mode output or CSS.

- [ ] **Step 7: Commit**

```sh
git add site/site_navigation.go site/config_build.go site/build.go site/config_build_test.go
git commit -m "feat: render shared family navigation"
```

### Task 6: Convert configured publication routes and showcase content

**Files:**
- Modify: `showcase.yaml` with `layouts` and `navigation.families`.
- Modify: `showcase/content/index.md` as the consolidated Tour landing page.
- Create: `showcase/content/module/index.md`.
- Create: `showcase/content/cli/index.md`.
- Delete: `showcase/content/charts.md`, `cli.md`, `decks.md`, `determinism.md`, `html.md`, `markdown.md`, `mermaid.md`, `module.md`, `pdf.md`, `policy.md`, `site.md`.
- Test: extend `site/config_build_test.go` with the repository showcase fixture assertions.

**Interfaces:**
- `showcase.yaml` uses only the validated profile/family fields from Tasks 3–5.
- Tour links to `/module/` and `/cli/` through Markdown route rewriting, not hard-coded deployment URLs.
- Module/CLI frontmatter uses `margo.actions` for Markdown/PDF retention and does not select raw frames.

- [ ] **Step 1: Write the showcase contract test**

Add a test that builds the repository `showcase.yaml` and asserts exactly three
HTML route pages (`index.html`, `module/index.html`, `cli/index.html`), no old
feature HTML artifacts, family/layout manifest identity, sitemap/llms entries,
Tour landing markers, and Module/CLI outline headings.

- [ ] **Step 2: Run the contract test and verify failure**

Run:

```sh
GOWORK=off go test ./site -run 'TestBuildConfiguredShowcase.*' -count=1
```

Expected: FAIL because the current showcase config has one shell and eleven
feature sources.

- [ ] **Step 3: Write the consolidated Tour Markdown**

Keep one document-level `h1`, retain the mascot and the strongest Mermaid/chart
examples, shorten feature prose for a three-to-five-minute decision, add
good-fit/not-a-fit guidance, and finish with links to Module and CLI. Remove
Tour page actions from frontmatter.

- [ ] **Step 4: Write Module and CLI overview/outlines**

Create one overview Markdown file per category with a compiling example, the
approved technical sections, and explicit future chapter headings. Keep only
the overview page in each family; do not create empty child routes.

- [ ] **Step 5: Update `showcase.yaml` and remove retired sources**

Declare the two layout profiles and three families exactly as the spec. Remove
the eleven old root Markdown sources so their old HTML outputs are absent and
their server routes return 404.

- [ ] **Step 6: Run the showcase contract and source checks**

Run:

```sh
GOWORK=off go test ./site -run 'TestBuildConfiguredShowcase.*' -count=1
GOWORK=off go test ./site -count=1
git diff --check
```

Expected: PASS; no generated files are edited by hand.

- [ ] **Step 7: Commit**

```sh
git add showcase.yaml showcase/content/index.md showcase/content/module/index.md showcase/content/cli/index.md showcase/content
git commit -m "docs: organize showcase into tour module and cli"
```

### Task 7: Add browser acceptance for layout switching and responsive output

**Files:**
- Create: `site/layout_profile_browser_test.go` for profile-mode server/browser checks.
- Modify: `site/config_build_test.go` for semantic HTML assertions that do not require a browser.

**Interfaces:**
- Browser fixture builds a temporary layout-profile site and serves artifacts on a loopback `httptest` server.
- Assertions query Margo-owned layout hooks and public navbar/sidebar semantics, not private App Shell selectors.

- [ ] **Step 1: Write failing browser tests**

Cover 390 px and 1440 px at light/dark modes. Navigate `/`, `/module/`, and
`/cli/`; assert active family state, Tour absence of sidebar/TOC/pagination,
Module/CLI sidebar presence, one mobile navigation trigger, visible keyboard
focus, no horizontal overflow, and no console errors. Assert links switch
family through normal navigation and retain correct active state.

- [ ] **Step 2: Run the browser tests and verify failure**

Run:

```sh
GOWORK=off go test ./site -run 'TestLayoutProfileBrowser' -count=1 -v
```

Expected: FAIL until profile-mode markup and responsive styles exist.

- [ ] **Step 3: Implement only test seams required by the public output**

Use the existing browser test launcher and page helpers. Wait for stable
selectors (`data-margo-layout`, navbar family links, sidebar navigation) rather
than fixed sleeps. Capture browser console/page errors and fail the test with
the route and viewport.

- [ ] **Step 4: Run focused browser tests at both viewport classes**

Run:

```sh
GOWORK=off go test ./site -run 'TestLayoutProfileBrowser' -count=1 -v
```

Expected: PASS at both viewport sizes and color modes.

- [ ] **Step 5: Commit**

```sh
git add site/layout_profile_browser_test.go site/goshtoso_shell_browser_test.go site/config_build_test.go
git commit -m "test: cover layout profile browser behavior"
```

### Task 8: Run repository-wide verification and update public documentation

**Files:**
- Modify: `README.md` configured-site documentation and layout/family reference.
- Modify: `unified_docs_test.go` to assert the new public layout/family contract.
- Test: repository gates and showcase serve/browser verification.

**Interfaces:**
- README documents `layouts.default`, `layouts.profiles`, `navigation.families`, `margo.site.layout`, legacy compatibility, and Tour/Module/CLI showcase routes.
- Documentation names semantic layouts, not raw shell implementations in page content.

- [ ] **Step 1: Extend the existing unified-doc test**

Require the README/config reference to contain the approved YAML shape, page
precedence, family-prefix behavior, landing/docs distinction, and the explicit
404 behavior for retired Tour routes.

- [ ] **Step 2: Run documentation checks and verify failure**

Run:

```sh
GOWORK=off go test ./... -run 'TestREADMEExplainsUnifiedCLIAndReleaseContract' -count=1
```

Expected: FAIL until the new contract is documented.

- [ ] **Step 3: Update README and its assertions**

Describe the generic convention, fallback compatibility, and a minimal page
frontmatter example. Update the exact required strings in
`TestREADMEExplainsUnifiedCLIAndReleaseContract`; no generated documentation
file is part of this feature.

- [ ] **Step 4: Run final gates**

Run:

```sh
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
GOWORK=off go mod verify
GOWORK=off go test -race ./site -count=1
git diff --check
```

Build and serve the showcase:

```sh
GOWORK=off go run ./cmd/margo site ./showcase.yaml --output-dir /tmp/margo-showcase-dist
GOWORK=off go run ./cmd/margo serve ./showcase.yaml
```

Verify `/`, `/module/`, `/cli/` return 200, retired feature paths return 404,
and generated sitemap/llms/manifest identities match the showcase contract.

- [ ] **Step 5: Commit documentation and verification changes**

```sh
git add README.md schema site showcase.yaml
git commit -m "docs: document layout profile publication model"
```

## Execution Handoff

After plan approval, execute tasks in order with a fresh test cycle and commit
at every task boundary. Do not push, merge, tag, release, deploy, or delete
worktrees without separate authorization.
