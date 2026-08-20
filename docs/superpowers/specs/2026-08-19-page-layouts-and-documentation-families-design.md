# Margo Page Layouts and Documentation Families Design

Date: 2026-08-19
Status: approved in chat; awaiting written-spec review

## Purpose

Add a reusable Margo convention for sites that combine a conversion-oriented
landing page with deeper documentation. A Markdown page selects a semantic page
layout while the trusted site configuration controls which concrete Margo frame
that name resolves to. Site-wide identity and navigation remain consistent
across layouts.

Use the convention to restructure the Margo showcase into three documentation
families:

- **Tour** is one Markdown-generated landing page that helps a potential user
  decide quickly whether Margo fits.
- **Module** begins the technical manual for integrating Margo into another Go
  project.
- **CLI** begins the technical manual for Margo commands, configuration,
  policies, limitations, and operational gotchas.

The first content increment ships the completed Tour plus concise Module and
CLI overview/outlines. It does not create empty technical chapter pages.

## Architectural Decision

A landing page does not require an external application shell. It still needs
site chrome and a structural page frame, but Margo can compose both from its
existing SSG frame contracts and public Goshtoso components.

Margo therefore distinguishes three concepts:

1. **Site chrome** owns brand, global category navigation, search, appearance
   controls, repository action, and footer.
2. **Page layout** is a semantic, author-selectable name such as `landing` or
   `docs`.
3. **Frame** is the trusted implementation selected by `site.yaml`, such as
   `top-main-footer` or `top-left-main-right-footer`.

Markdown never names an executable command, Go module, external shell, or raw
frame implementation. It may request only a layout profile allowlisted by the
site configuration.

The existing top-level `frame` and `shell` modes remain supported for existing
sites. Layout profiles form a new, mutually exclusive presentation mode.

## Configuration Contract

The Margo showcase uses this shape:

```yaml
layouts:
  default: docs
  profiles:
    landing:
      frame:
        builtin: top-main-footer
    docs:
      frame:
        builtin: top-left-main-right-footer

navigation:
  mode: file-tree
  families:
    - id: tour
      label: Tour
      source: .
      overview: index.md
      layout: landing
    - id: module
      label: Module
      source: module
      overview: module/index.md
      layout: docs
    - id: cli
      label: CLI
      source: cli
      overview: cli/index.md
      layout: docs
```

### Layout Profiles

`layouts.default` is required and must name one entry in `layouts.profiles`.
Profile names are trimmed, non-empty, unique YAML mapping keys. Each profile
selects one built-in frame through `frame.builtin`. Initial layout profiles do
not support command or Go-module frame distributions because the configured
site builder does not implement those distribution contracts.

The profile's frame is resolved and schema-validated before any page artifacts
are emitted. Frame option values may use the same published option paths and
validation already used by the legacy top-level frame configuration.

`layouts` cannot coexist with top-level `frame` or `shell`. This prevents two
presentation authorities from competing for the same output. A site without
`layouts` follows its existing frame or shell behavior byte-for-byte except for
unavoidable manifest/schema evolution explicitly covered by compatibility
tests.

### Page Layout Preference

A page may override its family or site default with closed Margo frontmatter:

```yaml
---
margo:
  site:
    layout: landing
---
```

`margo.site.layout` is an optional site-projection preference. It is normalized
into immutable document metadata. HTML fragment, standalone HTML, PDF, and deck
renderers ignore it; the configured site builder consumes it.

Layout resolution order is:

1. page `margo.site.layout`;
2. active navigation family's `layout`;
3. `layouts.default`.

Every selected name must exist in `layouts.profiles`. An unknown name is a
preflight error, not a fallback.

### Documentation Families

Each family has:

- a stable ID used for active-state identity;
- a human label rendered in the global secondary navbar row;
- a normalized source-directory prefix;
- an overview Markdown source inside that prefix; and
- an optional default layout profile.

Family IDs and normalized source roots must be unique. Source roots are matched
against locale-independent source paths. The most-specific directory prefix
wins, allowing `.` to be the fallback family while `module` and `cli` own their
subtrees. Matching is segment-aware: `cli` matches `cli/index.md`, not
`client/index.md`.

Every discovered public page must resolve to exactly one family when families
are configured. Each family's overview must exist, remain inside the family
source root, and resolve for every configured locale represented by that
family. Overview URLs are derived from normal Margo route and `base_path`
rules; config authors do not duplicate public URLs.

The first version requires layout-profile mode when families are configured.
Legacy single-frame and external-shell sites keep their current flat navigation
contract rather than receiving a partial family implementation.

## Frame and Binding Model

Layout-profile mode resolves a frame per page instead of one frame per site.
Frame identity, schema hash, resolved values, and bindings are computed for the
selected profile and recorded in deterministic route metadata.

The SSG binding vocabulary separates global and family-local navigation:

- `site_navigation` binds to `top-nav` and renders site chrome, including the
  secondary family row.
- `navigation` remains the local documentation navigation and binds to
  `left-nav` in documentation frames.

Built-in frames that contain `top-nav` accept one `site_navigation` binding.
Frames with `left-nav` continue to accept one `navigation` binding. This avoids
misusing breadcrumbs or rendering two indistinguishable `navigation` bindings.
Legacy binding defaults remain unchanged outside layout-profile mode.

The `landing` profile uses `top-main-footer`. It receives site navigation,
document content, and footer bindings. It receives no local navigation, TOC,
breadcrumbs, or pagination bindings.

The `docs` profile uses `top-left-main-right-footer`. It receives the same site
navigation and footer, plus active-family local navigation, breadcrumbs, a TOC,
and family-scoped pagination when neighbors exist.

## Site Chrome

Margo composes site chrome with public Goshtoso components rather than an App
Shell's private markup:

- Goshtoso `navbar.Navbar` provides brand, responsive primary controls, and the
  optional secondary row.
- `navbar.SecondaryConfig` renders Tour, Module, and CLI as primitive links with
  one `aria-current="location"` state.
- Goshtoso `sidebar.Sidebar` renders only the active family's local pages.
- Existing Margo theme, dark-mode, repository, footer, and global search
  behavior remain available through public component APIs and app-owned slots.

The brand link returns to the configured site home, which is the Tour. Family
links always open that family's overview. Search remains global and indexes all
published families. Mobile output exposes one primary navigation trigger and
does not pair a navbar menu with a second sidebar trigger for landing pages.

No App Shell private selector, DOM mutation, or route-specific CSS workaround
is allowed. Margo-owned frames expose semantic layout hooks such as
`data-margo-layout="landing"`; Margo's own stylesheet may target those hooks.
Goshtoso component internals remain opaque.

## Navigation Behavior

Global family order follows `navigation.families` configuration order. The
active family is derived from the page's resolved family, never inferred from
the current URL in browser JavaScript.

Local sidebar entries include only pages in the active family and locale. The
family overview is first; remaining pages retain deterministic file-tree order.
Module and CLI initially contain only their overview pages, so their sidebars
show a single Overview destination.

Pagination is family- and locale-scoped. It never moves from Tour to Module or
from Module to CLI. A one-page family renders no pagination.

Search, `sitemap.xml`, and `llms.txt` remain publication-global. Nested links,
canonical URLs, family overview URLs, and search destinations honor
`base_path` and locale routing.

## Margo Showcase Content

### Tour

`showcase/content/index.md` becomes the only Tour source and the site home. It
is still ordinary Markdown compiled by Margo. The landing profile changes its
frame and presentation, not its content format.

The page is designed for a three-to-five-minute adoption decision:

1. concise hero and value proposition;
2. one-source/several-projections visual;
3. short, polished examples of Markdown, HTML, sites, PDF, experimental decks,
   Mermaid, and charts;
4. brief trust section covering policy, deterministic artifacts, and offline
   behavior;
5. explicit "good fit / not a fit" guidance; and
6. final paths to the Module and CLI technical categories.

Tour has no sidebar, TOC rail, breadcrumbs, pagination, or page-action toolbar.
Its existing feature content is edited and consolidated rather than copied
verbatim into one oversized manual.

The former root feature sources are deleted:

```text
charts.md
cli.md
decks.md
determinism.md
html.md
markdown.md
mermaid.md
module.md
pdf.md
policy.md
site.md
```

Their old `.html` routes intentionally disappear and return 404. This design
does not add redirects or hidden compatibility pages.

### Module

`showcase/content/module/index.md` is a technical-category overview. It includes
a small compiling example, compiler/render lifecycle, public package map, host
ownership boundaries, extension and policy overview, and an ordered outline of
future chapters.

The outline covers installation/versioning, compiler lifecycle, source and
metadata, checks and diagnostics, rendering and HTML composition, site builds,
PDF engines, deck projection, chart extensions, policy/security,
concurrency/cancellation, determinism, and testing. No empty chapter files are
created in this increment.

The page retains Markdown/PDF page actions appropriate to technical
documentation.

### CLI

`showcase/content/cli/index.md` is a technical-category overview. It includes
installation and version checks, the stdin/stdout/stderr contract, exit and
diagnostic behavior, command map, configuration/policy layering, output
replacement safeguards, and an ordered outline of future command chapters.

The outline covers `check`, `html`, `pdf`, `deck`, `site`, `serve`, `doctor`,
`schema`, `version`, and `completion`, plus shared options, config schemas,
policies, limitations, and operational gotchas. No empty chapter files are
created in this increment.

The page retains Markdown/PDF page actions appropriate to technical
documentation.

## Metadata and Deterministic Identity

Configured `site.Page` and route manifest records gain resolved family and
layout identity. Layout profile frame name and schema hash participate in site
layout identity so two routes rendered through different profiles cannot appear
equivalent in the manifest.

Profile maps are processed in stable key order where ordering affects hashes.
Family navigation preserves declared sequence intentionally. Route discovery,
sidebar entries, search items, sitemap entries, and `llms.txt` remain
deterministically sorted under their documented contracts.

The showcase result contains exactly three HTML routes:

```text
/
/module/
/cli/
```

Assets, `sitemap.xml`, `llms.txt`, the site manifest, retained Markdown, and
pre-rendered PDF actions remain additional artifacts and do not count as HTML
routes.

## Error Handling

Configured-site preflight resolves every page's family, layout profile, frame,
schema, and bindings before artifact materialization. Any failure aborts the
build without a partial output set.

New diagnostics distinguish these cases:

- invalid or conflicting layout configuration;
- unknown default, family, or page layout;
- unsupported external frame distribution inside a profile;
- duplicate or invalid family identity;
- ambiguous/duplicate normalized family source root;
- missing, out-of-family, or locale-incomplete overview;
- public page not assigned to a family; and
- selected frame unable to accept required global or local navigation.

Configuration diagnostics identify `site.yaml` and the relevant YAML field.
Page-layout diagnostics identify the Markdown source and
`/margo/site/layout`. Messages include one actionable correction rather than
silently selecting another layout.

## Compatibility

- Sites without `layouts` and `navigation.families` retain current behavior.
- Existing top-level built-in frames remain supported.
- Existing `componentdocshell` sites remain supported.
- Existing generic static-site callers that do not use configured publication
  remain unchanged.
- `margo.site.layout` is ignored by non-site projections after schema-valid
  normalization.
- Legacy frame binding defaults do not change; new `site_navigation` defaults
  apply only to layout-profile mode.
- Existing route and artifact ordering remains stable unless the site opts into
  families/layout profiles.

## Testing Strategy

Implementation follows test-driven development.

### Configuration and Resolution

- Accept the approved layout/family example.
- Prove page, family, and site layout precedence.
- Reject unknown layouts at each authority level.
- Reject duplicate family IDs, duplicate normalized roots, invalid prefixes,
  and invalid overview paths.
- Prove segment-aware longest-prefix family resolution.
- Prove locale-independent family resolution and locale overview checks.
- Reject `layouts` combined with top-level `frame` or `shell`.
- Preserve existing configuration behavior when new fields are absent.

### Frame Composition

- Resolve and hash a distinct frame per layout profile.
- Bind `site_navigation` to top navigation in both landing and docs frames.
- Bind local `navigation` only in docs frames.
- Reject frames lacking required binding areas.
- Record resolved family/layout/frame identity in route manifests.
- Prove stable output across repeated identical builds.

### Navigation and Discovery

- Render Tour, Module, and CLI in declared secondary-row order.
- Render one correct `aria-current="location"` family state per route.
- Generate `base_path`-correct overview, sidebar, search, breadcrumb, and
  canonical links.
- Keep sidebar entries and pagination inside the active family and locale.
- Index every published family in global search, sitemap, and `llms.txt`.

### Showcase Contract

- Produce exactly `/`, `/module/`, and `/cli/` HTML routes.
- Produce none of the removed feature-route artifacts.
- Render Tour with no sidebar, TOC, breadcrumbs, pagination, or page actions.
- Render Module and CLI with docs layout and only their own local navigation.
- Keep all three pages Markdown-generated with exactly one document `h1`.
- Retain Mermaid and chart rendering on the consolidated Tour.

### Browser Acceptance

Browser tests cover 390 px and 1440 px under light and dark modes:

- Tour, Module, and CLI family navigation;
- visible and correct active states;
- keyboard navigation and focus visibility;
- one mobile navigation trigger;
- no Tour sidebar or empty-sidebar gutter;
- no horizontal overflow;
- readable landing-page hierarchy and examples;
- usable Module/CLI sidebar and TOC behavior; and
- no browser console errors.

## Verification Gates

Fresh completion evidence includes:

```sh
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
GOWORK=off go mod verify
GOWORK=off go test -race ./site -count=1
git diff --check
```

The showcase is rebuilt from `showcase.yaml`, checked for deterministic route
and artifact identity, served through `margo serve`, and inspected with the
browser acceptance matrix. Generated output is never hand-edited.

## Non-Goals

- Complete Module or CLI manuals in this increment
- Empty placeholder chapter pages
- Redirects for removed Tour feature routes
- Arbitrary page-selected commands, Go modules, shells, or raw frames
- External frame distribution support
- Per-page theme, brand, policy, or security authority
- Migrating existing sites automatically to layout profiles
- Replacing or removing `componentdocshell` support
- Releasing, tagging, pushing, merging, or deploying without separate
  authorization
