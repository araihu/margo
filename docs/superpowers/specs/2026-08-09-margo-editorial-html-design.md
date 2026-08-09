# Margo editorial HTML design

**Status:** Approved by the user on 2026-08-09.

**API boundary amendment (approved 2026-08-09):** The technical distinction is
fragment versus complete HTML document, not editorial versus publication. The
normative root API is now `RenderHTML`/`HTMLResult` for the immutable fragment
projection and `RenderHTMLPage`/`HTMLPageInput` for a generic document shell.
The root page contract has no authority record, route, canonical URL, social
metadata, or article kind. Consumers compose those concerns through generic
`Head`, `Header`, `BeforeContent`, and `Footer` components; Margo provides no
publication-domain adapter. References below to `EditorialResult`,
`RenderEditorial`, `PublicationInput`, and `RenderPublication` record the
original design and are superseded by this amendment. PDF consumers may use
the semantic render or generic HTML contract without accepting public-web
policy.

**Primary goal:** Make Goshtoso-compatible HTML the first-class Margo product
for documentation and blogging. Manja consumes safe fragments inside its
existing documentation shell. The Arai Hû site consumes complete static
publication pages. Both outputs come from one semantic render and preserve
Goshtoso theme and client behavior.

## Context

Margo already compiles Markdown into semantic HTML, renders Markdown tables
through Goshtoso, supports the optional charts module, and can assemble an
offline standalone document. Those pieces do not yet form a coherent consumer
contract:

- `RenderResult.Content()` is an embeddable article but exposes no HTML
  requirements;
- `RenderStandalone` owns a full shell, embeds CSS, restricts themes to a small
  closed list, and carries print-specific behavior;
- registered extension fences remain visible as code blocks in the article and
  their rendered output is appended after the article instead of appearing at
  the source position;
- Markdown tables carry `data-table-client-sort` but the pinned Goshtoso table
  and Margo output contain no client-sort controls or runtime, so the marker is
  not current behavioral proof;
- chart-aware HTML becomes functional through tool-specific byte rewriting;
- social publication metadata is added by rewriting already-rendered HTML;
- Manja needs a fragment inside its own Goshtoso page, while araihu.com needs a
  complete generated page with route metadata.

The HTML product must make these use cases explicit without coupling the root
module to the optional charts module.

## Decisions

### 1. One semantic render, two first-class projections

The compiler continues to produce one immutable `RenderResult`. A new
editorial projection converts that result into an immutable `EditorialResult`.
It exposes:

```go
type EditorialResult struct {
    fragment     templ.Component
    plainText    string
    metadata     EditorialMetadata
    requirements HTMLRequirements
    diagnostics  []Diagnostic
    fingerprint  EditorialFingerprint
}

func RenderEditorial(result *RenderResult, options ...EditorialOption) (*EditorialResult, error)
func (r *EditorialResult) Fragment() templ.Component
func (r *EditorialResult) PlainText() string
func (r *EditorialResult) Metadata() EditorialMetadata
func (r *EditorialResult) Requirements() HTMLRequirements
func (r *EditorialResult) Diagnostics() []Diagnostic
func (r *EditorialResult) Fingerprint() EditorialFingerprint
```

`Fragment()` is the canonical host-embeddable output. It contains one
`<article class="margo-document">` and no `<!doctype>`, `<html>`, `<head>`,
`<body>`, host stylesheet, script, theme attribute, or color-mode class. It
inherits all page-level presentation from the host. The trusted Charts adapter
is the sole scoped-style exception: upstream static SVG components retain
their required token-based CSS, marked
`data-margo-extension-style="charts"`. Unowned styles remain invalid.

Complete pages use the same `EditorialResult`:

```go
func RenderPublication(editorial *EditorialResult, input PublicationInput) (templ.Component, error)
```

`RenderPublication` emits a complete static HTML document with initial head
metadata, resolved dependencies, publication chrome, and the exact fragment.
It never reparses Markdown or renders extension nodes a second time.

`RenderStandalone` remains a compatibility and preview API. Its implementation
will delegate to the editorial projection and publication assembler instead of
maintaining a separate HTML path. Print/PDF preparation remains a standalone
option and is not part of the fragment contract.

### 2. Extensions render once at their source position

The render plan assigns every registered fence a stable source-order slot. It
renders each extension node once into a private per-slot buffer before the
semantic article is serialized. When the Markdown renderer reaches an owned
fence, it writes that slot's output instead of rendering the fence as a code
block. Unregistered ordinary code fences remain code blocks.

This preserves Markdown order and keeps every chart inside the article. A
failed extension produces no `RenderResult`; no partial article or publication
bytes escape. Extension sessions remain isolated per render operation, and
per-extension node order remains source order even when multiple extension
families are registered.

### 3. Host-owned shell and theme

Embedded output never selects a theme. Manja's existing `<html data-theme>` and
light/dark state remain authoritative. Margo document CSS stays scoped below
`.margo-document` and uses Goshtoso semantic tokens; it must not reset host
tokens or hard-code a palette.

Complete pages accept a `ThemeName` matching `^[a-z][a-z0-9-]{0,63}$` rather
than a closed enum. Built-in themes such as `modern`,
`minimal`, and `news`, plus consumer themes such as `araihu` and `manja`, are
therefore valid. Margo validates syntax, not whether a consumer stylesheet
defines the name. The caller owns custom theme CSS and loads it after Goshtoso
base CSS.

Color mode remains separate from theme. A complete page may select light or
dark initially. A fragment never emits `.dark` or `data-color-mode` and follows
the host document element.

### 4. Typed HTML requirements

Client behavior cannot depend on a tool scanning and rewriting rendered HTML.
Every root feature and optional extension declares requirements before output
assembly. The result carries only requirements used by that document.

```go
type HTMLRequirementKind string

const (
    HTMLStylesheet HTMLRequirementKind = "stylesheet"
    HTMLScript     HTMLRequirementKind = "script"
    HTMLRuntimeRole HTMLRequirementKind = "runtime-role"
)

type HTMLRequirement struct {
    ID        string
    Kind      HTMLRequirementKind
    LocalURL  string
    Integrity string
    LoadAfter []string
    Inline    AssetRef
}

type HTMLRequirements struct {
    requirements []HTMLRequirement
}

func (r HTMLRequirements) List() []HTMLRequirement
```

Requirement IDs are stable public identities. Initial IDs are:

- `goshtoso.styles`, satisfied by the host's Goshtoso stylesheet;
- `margo.document.styles`, the scoped editorial stylesheet;
- `margo.table-sort`, Margo's progressive client-sort runtime;
- `goshtoso.runtime.alpine-focus`, `goshtoso.runtime.first-party`, and
  `goshtoso.runtime.alpine`, required by enabled chart controls in that order;
- `goshtoso-charts.controls`, the upstream chart control runtime, loaded after
  the three Goshtoso roles.

Exact URLs, hashes, embedded bytes, and ordering come from the owning module's
reviewed asset API. Margo does not copy third-party version strings into source.

An `ExtensionRegistration` may contribute HTML requirements. The registry
snapshots and validates them with the extension identity. Requirements are
attached to a document only when its render plan uses that extension. Root
never imports `github.com/araihu/margo/charts` or
`github.com/araihu/goshtoso-charts`.

Requirement merging is fail-closed:

- the same ID and same immutable identity deduplicates;
- the same ID with different kind, URL, hash, bytes, or ordering is an error;
- missing dependencies and ordering cycles are errors;
- unsafe IDs or URLs are errors;
- all errors occur before any publication bytes are written.

The fragment does not emit requirements. The host reads
`Requirements()` and satisfies them through its existing Goshtoso head and
asset handlers. The publication assembler can render local tags or inline exact
materialized bytes. Offline standalone uses inline mode. Normal static sites
use versioned local URLs and their handlers.

Margo-owned local requirement URLs use `/margo-assets/`; they never compete
with Goshtoso's `/assets/` or Goshtoso Charts' `/charts/assets/` mounts. A new
self-stripping editorial asset handler serves that prefix. The existing
`AssetHandler` remains available for compatibility but is not used by new host
integration examples.

### 5. Functional behavior with static fallbacks

Generated HTML is useful without JavaScript:

- tables remain semantic, readable tables in source order;
- every chart retains accessible static SVG plus its data table;
- chart actions are absent when the chart extension is configured static;
- Mermaid retains its reviewed source/fallback state until its own runtime
  contract completes.

When declared requirements are loaded:

- Markdown table headers sort client-side, update `aria-sort`, remain keyboard
  operable, and restore source order for printing;
- chart expand, close, and other enabled controls execute through the upstream
  Goshtoso Charts control runtime;
- runtime dependencies load once in declared order even when one document uses
  multiple tables or charts.

Table sorting is progressive enhancement owned by Margo for this Markdown
adapter. Server output keeps Goshtoso's table structure plus inert sort keys.
`margo.table-sort` inserts native header buttons only after its runtime starts,
then owns stable natural ordering, `aria-sort`, keyboard activation, and print
restoration. JavaScript-disabled output therefore retains plain semantic
headers without inert controls. This design does not claim that the currently
pinned Goshtoso module provides a client-sort API it does not contain.

The charts module declares control requirements when its wrapper is enabled.
It declares none when `WithControlWrapper(false)` is used. The current
upstream wrapper emits one control `<script>` per chart. The Margo charts
adapter removes only that exact upstream-owned loader tag while privately
rendering each chart and replaces it with the single
`goshtoso-charts.controls` requirement. It does not scan or rewrite the
completed document. The optimistic renderer's `inlineChartControlRuntime`
transformation becomes unnecessary and will be removed after the editorial
assembler passes the same browser journey.

### 6. Editorial metadata and visible article structure

`EditorialMetadata` extends the current title/description projection with the
bounded fields required by documentation and blogs:

```go
type EditorialMetadata struct {
    Title       string
    Description string
    Language    string
    Slug        string
    Authors     []string
    PublishedAt string
    ModifiedAt  string
    Tags        []string
}
```

Dates use normalized RFC 3339 strings so metadata stays stable across time
zones and serialization. Lists are copied defensively. Frontmatter accepts the
exact top-level keys `title`, `description`, `language`, `slug`, `authors`,
`publishedAt`, `modifiedAt`, and `tags`. Unknown generic top-level metadata
continues to be accepted; unknown `goshtoso` configuration remains an error.

Visible heading policy:

- an existing first body `<h1>` remains the visible article title;
- when the body has no `<h1>` and metadata has a title, publication output
  inserts that title in an article header;
- a fragment does not invent a heading unless requested with an explicit
  editorial option, allowing Manja to keep its surrounding OpenAPI heading;
- head title and social title use metadata title, falling back to the first
  body `<h1>`;
- conflicting metadata and first-body titles produce a stable warning
  diagnostic, not duplicate headings.

Publication article chrome may render authors, publication/modification dates,
and tags using semantic `<address>`, `<time>`, and list markup. Empty fields
emit nothing.

### 7. Publication head contract

`PublicationInput` distinguishes a documentation page from a blog article:

```go
type PublicationInput struct {
    Mode            PublicationMode
    Kind            PublicationKind
    Authority       AuthorityRecord
    RoutePath       string
    SiteName        string
    Locale          string
    Image           SocialImage
    Theme           ThemeName
    ColorMode       ColorMode
    DependencyMode  HTMLDependencyMode
    ThemeStylesheet AssetRef
    Header          templ.Component
    Footer          templ.Component
}
```

Title, description, language, and article fields come from the immutable
editorial result. Canonical URL derives from the verified authority origin and
`RoutePath`; callers cannot supply a second conflicting canonical value.

All public metadata appears exactly once in initial HTML:

- title and description;
- canonical URL;
- Open Graph URL, type, title, description, site name, locale, and image fields;
- explicit X/Twitter Card fields;
- for articles, published time, modified time, authors, and tags when supplied.

Blog pages emit `og:type=article`; documentation pages emit
`og:type=website`. Existing authority validation remains mandatory for public
URLs. Private/local pages omit canonical and social URLs. Publication metadata
is rendered directly by templ composition; string replacement over completed
HTML is removed.

### 8. Consumer boundaries

#### Manja

Manja replaces its private Goldmark adapter with a Margo-backed adapter that
returns the fragment HTML and plain-text projection expected by its existing
port. Manja keeps ownership of:

- page layout and OpenAPI information architecture;
- `.manja-markdown` compatibility wrapper during migration;
- Goshtoso head dependencies, asset handlers, theme, dark mode, search, and
  navigation;
- source-specific sanitization policy.

The Margo repository first proves this seam through a Manja-shaped external
consumer fixture. Direct Manja repository changes are a downstream integration
slice after the Margo API is reviewed.

#### araihu.com

The site passes localized publication metadata, the `araihu` theme, its custom
theme stylesheet, and route authority into `RenderPublication`. Its static
builder writes one complete page per locale/route and serves versioned assets.
The site retains ownership of locale routing, project navigation, organization
branding, and deployment.

The Margo repository first proves this seam through an araihu.com-shaped static
site fixture. Direct site repository changes remain a downstream integration
slice.

### 9. Errors and immutability

Public constructors validate complete inputs before returning a component.
Rendering never returns a partially written head or dependency list after a
configuration error. Nil results, invalid metadata, unsafe theme names,
dependency conflicts, missing inline bytes, invalid integrity, and unsatisfied
publication authority use stable diagnostic codes.

The initial stable codes are `editorial.result_required`,
`editorial.metadata_invalid`, `editorial.title_conflict`,
`editorial.theme_invalid`, `html.requirement_invalid`,
`html.requirement_conflict`, `html.requirement_dependency_missing`,
`html.requirement_cycle`, and the existing `publication.*` authority codes.

All metadata slices, requirement slices, inline bytes, and option maps are
snapshotted. Equivalent inputs yield identical fragment bytes, requirement
ordering, metadata projection, and editorial fingerprint. Host mutation after
construction cannot change output.

The editorial fingerprint covers the semantic fragment bytes, normalized
editorial metadata, ordered requirement identities, and relevant editorial
options. It does not include host-only layout components that are outside the
fragment contract.

## Testing and acceptance

Implementation follows strict RED-GREEN-REFACTOR. Unit tests prove:

- fragment boundaries and semantic article structure;
- metadata normalization, copying, title resolution, and diagnostics;
- theme inheritance and acceptance of safe custom names;
- requirement derivation, ordering, deduplication, conflict rejection, and
  extension isolation;
- direct templ head composition with one complete social set;
- deterministic bytes and fingerprints;
- standalone compatibility over the shared editorial path;
- charts static/enabled requirement differences.

Generated-HTML E2E tests run against a local server and the installed Chromium.
Documentation records the exact Chromium version used by the gate; browser
version is evidence, not a user compatibility pin. Journeys cover:

1. Manja-shaped fragment inside a Goshtoso shell.
2. araihu.com-shaped complete article under built-in and custom themes.
3. Light/dark host changes without regenerating the fragment.
4. Client table sorting, keyboard activation, `aria-sort`, and source-order
   restoration.
5. Chart controls expand/close with static SVG and accessible data preserved.
6. JavaScript-disabled readable fallback.
7. One load per dependency, correct ordering, no blocked requests, no browser
   console/page errors, and no duplicate IDs.
8. Initial-HTML canonical/Open Graph/X/article metadata.

PDF rendering and PDF visual correctness are not acceptance gates for this
slice. Generated PDFs remain a later human-review projection over the same
editorial HTML.

## Implementation slices

1. Normalize editorial metadata and title policy in the root module.
2. Preserve extension output once at its Markdown source position.
3. Add immutable HTML requirements and extension contribution seams.
4. Add `EditorialResult` and host-embeddable fragment contract.
5. Add Margo's progressive table-sort markup/runtime requirements.
6. Project and deduplicate chart control requirements from the charts module.
7. Compose publication head/body directly and generalize theme validation.
8. Rebuild standalone output over the shared editorial assembler.
9. Add Manja-shaped and araihu.com-shaped consumer fixtures.
10. Add generated-HTML unit/E2E gates and tested-browser documentation.
11. Remove obsolete optimistic-renderer byte rewriting.

Each slice receives exact owned files, forbidden files, commands, hashes, and
review gates in the implementation plan. Root module files remain unchanged
unless an independently justified dependency change is required; no replace,
pseudo-version, tag, release, push, merge, or external consumer edit is implied
by this design.

## Out of scope

- PDF engines, browser collectors, PDF export, or PDF visual acceptance.
- Slide/deck rendering.
- Live OpenAPI request execution or a Manja proxy/Try It surface.
- A new client framework or Margo-owned global theme switcher.
- Loading arbitrary runtime plugins from Markdown.
- Direct deployment, release, tag, push, merge, or publication.
