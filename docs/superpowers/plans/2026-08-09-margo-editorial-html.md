# Margo Editorial HTML Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Goshtoso-compatible editorial HTML Margo's first-class output, with an embeddable Manja fragment, a complete araihu.com publication page, inherited themes, functional table sorting, functional chart controls, and generated-HTML unit/E2E proof.

**Architecture:** Root compilation renders every extension once at its Markdown position, then projects immutable editorial metadata and an ordered HTML requirement graph into `EditorialResult`. Hosts consume the fragment plus requirements; complete pages compose the same fragment, local or inline requirements, publication metadata, and caller-owned theme/brand material. The separately versioned charts module declares requirements through the existing extension capability field so its ordinary `GOWORK=off` gate does not require a fabricated root version.

**Tech Stack:** Go 1.26.5, Goldmark, templ v0.3.1020, Goshtoso v0.1.2, Goshtoso Charts pinned in `charts/go.mod`, `golang.org/x/net/html`, browser-native JavaScript, chromedp v0.14.2 for tagged cross-module E2E, installed Chromium/Chrome with tested version recorded as evidence.

## Global Constraints

- Approved design: `docs/superpowers/specs/2026-08-09-margo-editorial-html-design.md`.
- Root never imports `github.com/araihu/margo/charts` or `github.com/araihu/goshtoso-charts`.
- Fragment output contains one `.margo-document` article and no document shell, stylesheet, script, theme, or color-mode ownership.
- Extension output appears once at its Markdown source position. Registered fences never remain as code blocks.
- Fragment presentation inherits the host's Goshtoso theme and `.dark` state. Custom theme names match `^[a-z][a-z0-9-]{0,63}$`.
- Tables and charts remain readable and accessible without JavaScript. Loaded requirements enable client sorting and chart controls.
- Margo assets use `/margo-assets/`; Goshtoso keeps `/assets/`; Goshtoso Charts keeps `/charts/assets/`.
- Public metadata appears once in initial HTML. Article pages use `og:type=article`; documentation pages use `og:type=website`.
- No PDF engine, PDF collector, PDF export, deck, live OpenAPI execution, proxy/Try It surface, release, tag, push, merge, deployment, or external consumer edit.
- Preserve root `go.mod`, root `go.sum`, and root toolchain lock. Task 9 alone may add exact chromedp v0.14.2 to `charts/go.mod` and `charts/go.sum` for tagged E2E.
- No `replace`, fabricated pseudo-version, local tag, or committed `go.work`. Cross-module gates create a temporary workspace and delete only that temporary directory through normal test cleanup.
- Every root command runs with `GOWORK=off GOFLAGS=-mod=readonly`. Every ordinary charts command runs with the same environment. Only the explicitly named cross-module E2E command sets `GOWORK` to a temporary file.
- Before and after every module gate, record `git hash-object go.mod go.sum` for that module and fail on drift.
- Generated `*_templ.go` files are regenerated with `GOWORK=off GOFLAGS=-mod=readonly go tool templ generate`; never hand-edit them.
- Strict RED-GREEN-REFACTOR: prove intended failure, add minimum behavior, prove focused green, refactor, repeat green byte-for-byte, commit exact owned paths.
- Each task ends with `git diff --check`, a path-scoped staged manifest, and one reviewable commit. Never stage unrelated paths.
- Browser gate records installed browser version; it does not reject a user browser solely for differing from that tested version.

## File Map

### Root module

- `render_plan.go`: source-position extension slots and private per-slot execution.
- `render.go`: semantic renderer inserts pre-rendered extension slots.
- `metadata.go`, `markdown.go`: normalized editorial frontmatter values and defensive copies.
- `html_requirement.go`: public requirement types, validation, merge, ordering, and capability envelope.
- `editorial.go`: immutable editorial result, fragment/plain-text projection, title policy, and options.
- `editorial_fingerprint.go`: canonical editorial identity.
- `publication.go`, `publication.templ`, `publication_templ.go`: complete page contract and direct templ composition.
- `social.go`, `social.templ`, `social_templ.go`: article/document head metadata and compatibility delegation.
- `table_adapter.go`: inert progressive-sort markup.
- `assets/table-sort.js`: table enhancement runtime.
- `assets/document.css`: fragment/editorial styles and inserted sort controls.
- `assets.go`: `/margo-assets/` handler and JS embedding.
- `standalone.go`, `standalone.templ`, `standalone_templ.go`: compatibility wrapper over editorial/publication path.

### Charts module

- `charts/extension.go`: requirement capability declarations without new root API usage.
- `charts/controls.go`: suppress declared per-chart loader tags at the extension boundary.
- `charts/e2e/editorial_html_test.go`: tagged generated-HTML browser journeys.
- `charts/go.mod`, `charts/go.sum`: exact chromedp v0.14.2 E2E dependency only.

### Documentation and fixtures

- `testdata/markdown/editorial-article.md`: complete editorial source fixture.
- `charts/testdata/markdown/editorial-charts.md`: chart source-order and controls fixture.
- `docs/testing/editorial-html.md`: consumer contract, gates, and tested browser evidence.
- `README.md`, `charts/README.md`: public entry points and integration examples.

---

### Task 1: Render extensions once at Markdown source position

**Files:**
- Modify: `render_plan.go`
- Modify: `render.go`
- Modify: `render_test.go`
- Modify: `extension_test.go`

**Interfaces:**
- Consumes: existing `ExtensionRegistration`, `ExtensionSession`, `ExtensionNode`, `renderPlan`, and Goldmark fenced-code AST.
- Produces: private `plannedExtensionNode`, `executeRenderPlanSlots(context.Context, renderPlan) ([][]byte, error)`, and slot-aware `markdownRenderer.extensionSlots`.

- [ ] **Step 1: Write failing source-order and single-render tests**

Add a deterministic extension fixture that writes one figure. Assert `before`, figure, and `after` byte indexes are increasing; assert the registered fence source and duplicate appendix output are absent.

```go
type extensionSessionFunc func(context.Context, ExtensionNode, io.Writer) error

func (fn extensionSessionFunc) Render(ctx context.Context, node ExtensionNode, out io.Writer) error {
    return fn(ctx, node, out)
}

func TestExtensionRendersOnceAtMarkdownPosition(t *testing.T) {
    compiler := New(WithExtension(ExtensionRegistration{
        Identity: ExtensionIdentity{Name: "demo", Version: "v1"},
        Fences: []string{"demo"},
        Factory: func(RenderContext) (ExtensionSession, error) {
            return extensionSessionFunc(func(_ context.Context, _ ExtensionNode, out io.Writer) error {
                _, err := io.WriteString(out, `<figure data-demo="true">rendered</figure>`)
                return err
            }), nil
        },
    }))
    document, err := compiler.Compile(context.Background(), Source{
        Name: "ordered.md",
        Content: []byte("before\n\n```demo\npayload\n```\n\nafter\n"),
    })
    if err != nil { t.Fatal(err) }
    result, err := compiler.Render(context.Background(), document)
    if err != nil { t.Fatal(err) }
    markup := renderComponent(t, result.Content())
    before := strings.Index(markup, "<p>before</p>")
    figure := strings.Index(markup, `<figure data-demo="true">rendered</figure>`)
    after := strings.Index(markup, "<p>after</p>")
    if before < 0 || figure <= before || after <= figure { t.Fatalf("source order lost: %s", markup) }
    if strings.Count(markup, `data-demo="true"`) != 1 || strings.Contains(markup, "payload") {
        t.Fatalf("extension duplicated or fence leaked: %s", markup)
    }
}
```

- [ ] **Step 2: Prove RED**

Run:

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'TestExtension(RendersOnceAtMarkdownPosition|FailureWritesNoResult)' -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: FAIL because current output keeps the fence as a code block and appends extension bytes after `</article>`.

- [ ] **Step 3: Add source slots and private per-slot spooling**

Change `renderPlan.nodes` to `[]plannedExtensionNode`. Assign the global source-order slot while walking the AST and store it as an unexported Goldmark attribute. Render sessions into one buffer per slot before serializing the semantic article.

```go
const extensionSlotAttribute = "margo-extension-slot"

type plannedExtensionNode struct {
    node              ExtensionNode
    registrationIndex int
    slot              uint32
}

func extensionSlot(node *ast.FencedCodeBlock) (uint32, bool) {
    value, ok := node.AttributeString(extensionSlotAttribute)
    slot, valid := value.(uint32)
    return slot, ok && valid
}
```

In `markdownRenderer.renderNode`, replace owned fences with their spooled bytes:

```go
case *goldast.FencedCodeBlock:
    if slot, ok := extensionSlot(value); ok {
        if int(slot) >= len(r.extensionSlots) { return fmt.Errorf("extension.slot_missing") }
        _, err := r.out.Write(r.extensionSlots[slot])
        return err
    }
```

- [ ] **Step 4: Prove GREEN and failure atomicity**

Run:

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'TestExtension|TestSemanticRender' -count=1
GOWORK=off GOFLAGS=-mod=readonly go test -race . -run 'TestConcurrentExtensionSessions|TestExtensionRendersOnceAtMarkdownPosition' -count=20
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: PASS; no extension failure returns a `RenderResult`, every figure is inside the article, and concurrent renders keep independent slots.

- [ ] **Step 5: REFACTOR and repeat GREEN**

Extract slot cloning and bounds checks without changing public types. Repeat Step 4 exactly. Expected: PASS.

- [ ] **Step 6: Commit checkpoint**

```bash
git diff --check
git add -- render_plan.go render.go render_test.go extension_test.go
test "$(git diff --cached --name-only | LC_ALL=C sort)" = "$(printf '%s\n' extension_test.go render.go render_plan.go render_test.go | LC_ALL=C sort)"
git diff --cached --check
git commit -m "fix: preserve extension source order"
```

---

### Task 2: Normalize editorial metadata and defensive copies

**Files:**
- Modify: `metadata.go`
- Modify: `markdown.go`
- Modify: `frontmatter_test.go`
- Create: `editorial_metadata_test.go`
- Create: `testdata/markdown/editorial-article.md`

**Interfaces:**
- Consumes: existing `Metadata`, `frontmatterResult.values`, and `sourceNormalization`.
- Produces: deep-copying metadata fields `Language`, `Slug`, `Authors`, `PublishedAt`, `ModifiedAt`, `Tags`; normalized RFC 3339 date strings.

- [ ] **Step 1: Write failing normalization tests**

Use exact top-level keys and mutation checks.

```go
func TestEditorialFrontmatterNormalizesMetadata(t *testing.T) {
    document, err := New().Compile(context.Background(), Source{Name: "post.md", Content: []byte(`---
title: Durable HTML
description: One semantic source.
language: pt-BR
slug: durable-html
authors: ["Arai Hû"]
publishedAt: "2026-08-09T12:00:00-03:00"
modifiedAt: "2026-08-09T15:00:00Z"
tags: [Go, HTML]
---
Body
`)})
    if err != nil { t.Fatal(err) }
    metadata := document.Metadata()
    if metadata.Language != "pt-BR" || metadata.Slug != "durable-html" { t.Fatalf("metadata = %#v", metadata) }
    metadata.Authors[0] = "mutated"
    metadata.Tags[0] = "mutated"
    again := document.Metadata()
    if again.Authors[0] != "Arai Hû" || again.Tags[0] != "Go" { t.Fatal("metadata aliases caller") }
}
```

Add table tests rejecting non-string lists, invalid RFC 3339, invalid language, and invalid slug with exact `frontmatter.editorial.*` diagnostic codes.

- [ ] **Step 2: Prove RED**

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'TestEditorial(Frontmatter|Metadata)' -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: compile failure because `Metadata` lacks editorial fields.

- [ ] **Step 3: Add minimum parsing and cloning**

Extend `Metadata` and deep-copy lists:

```go
type Metadata struct {
    Name, BaseURL, Title, Description string
    Language, Slug                    string
    Authors                           []string
    PublishedAt, ModifiedAt           string
    Tags                              []string
}

func (m Metadata) clone() Metadata {
    m.Authors = append([]string(nil), m.Authors...)
    m.Tags = append([]string(nil), m.Tags...)
    return m
}
```

Accept only string scalars/lists. Normalize dates with `time.Parse(time.RFC3339, value).Format(time.RFC3339)`. Validate language with `^[A-Za-z]{2,8}(?:-[A-Za-z0-9]{1,8})*$` and slug with `^[a-z0-9]+(?:-[a-z0-9]+)*$`.

- [ ] **Step 4: Prove GREEN**

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'Test(EditorialFrontmatter|EditorialMetadata|Frontmatter)' -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: PASS with stable diagnostics and defensive copies.

- [ ] **Step 5: REFACTOR and repeat GREEN**

Move scalar/list/date helpers next to Markdown normalization, retain exact codes, repeat Step 4. Expected: PASS.

- [ ] **Step 6: Commit checkpoint**

```bash
git diff --check
git add -- metadata.go markdown.go frontmatter_test.go editorial_metadata_test.go testdata/markdown/editorial-article.md
git diff --cached --check
git commit -m "feat: normalize editorial metadata"
```

---

### Task 3: Add immutable ordered HTML requirements

**Files:**
- Create: `html_requirement.go`
- Create: `html_requirement_test.go`
- Modify: `extension.go`
- Modify: `registry.go`
- Modify: `render_plan.go`
- Modify: `document.go`
- Modify: `result.go`
- Modify: `compiler.go`
- Modify: `fingerprint_test.go`

**Interfaces:**
- Consumes: `AssetRef`, `ExtensionIdentity.Capabilities`, `renderPlan` used registrations, `Document`, and `RenderResult` cloning rules.
- Produces: `HTMLRequirementKind`, `HTMLRequirement`, `HTMLRequirements.List`, `HTMLRequirementCapability`, strict capability decoding, ordered merge, and result-carried requirements.

- [ ] **Step 1: Write failing graph and capability tests**

Cover cloning, stable order, deduplication, conflict, missing dependency, cycle, and capability round trip.

```go
func TestHTMLRequirementsMergeOrdersAndDeduplicates(t *testing.T) {
    merged, err := mergeHTMLRequirements([]HTMLRequirement{
        {ID: "runtime", Kind: HTMLScript, LocalURL: "/margo-assets/runtime.js", LoadAfter: []string{"styles"}},
        {ID: "styles", Kind: HTMLStylesheet, LocalURL: "/margo-assets/document.css"},
        {ID: "styles", Kind: HTMLStylesheet, LocalURL: "/margo-assets/document.css"},
    })
    if err != nil { t.Fatal(err) }
    list := merged.List()
    if len(list) != 2 || list[0].ID != "styles" || list[1].ID != "runtime" { t.Fatalf("order = %#v", list) }
    list[0].ID = "mutated"
    if merged.List()[0].ID != "styles" { t.Fatal("requirements alias caller") }
}
```

- [ ] **Step 2: Prove RED**

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'TestHTMLRequirement|TestExtensionRequirement' -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: compile failure because requirement types do not exist.

- [ ] **Step 3: Implement value validation and topological merge**

Define exact kinds and defensive projection:

```go
type HTMLRequirementKind string

const (
    HTMLStylesheet  HTMLRequirementKind = "stylesheet"
    HTMLScript      HTMLRequirementKind = "script"
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

type HTMLRequirements struct { requirements []HTMLRequirement }
func (r HTMLRequirements) List() []HTMLRequirement { return cloneHTMLRequirements(r.requirements) }
```

Use stable lexical selection among currently dependency-free nodes. Validate ID with `^[a-z][a-z0-9.-]{0,127}$`; URLs must be absolute-path local URLs or HTTPS; integrity is empty or `sha384-` base64; inline content must pass `AssetRef.validate()`.

- [ ] **Step 4: Add the separately-versioned capability envelope**

Encode canonical field names with standard `encoding/json`, then base64url without padding:

```go
const htmlRequirementCapabilityPrefix = "margo-html-requirement/v1:"

type htmlRequirementCapabilityValue struct {
    ID        string              `json:"id"`
    Kind      HTMLRequirementKind `json:"kind"`
    LocalURL  string              `json:"localURL,omitempty"`
    Integrity string              `json:"integrity,omitempty"`
    LoadAfter []string            `json:"loadAfter,omitempty"`
    Inline    *htmlRequirementCapabilityAsset `json:"inline,omitempty"`
}

type htmlRequirementCapabilityAsset struct {
    Path      string `json:"path"`
    MediaType string `json:"mediaType"`
    SHA256    string `json:"sha256"`
    Content   []byte `json:"content"`
}

func htmlRequirementCapabilityAssetFrom(asset AssetRef) *htmlRequirementCapabilityAsset {
    if asset.Path == "" && asset.MediaType == "" && asset.SHA256 == "" && len(asset.Content) == 0 { return nil }
    return &htmlRequirementCapabilityAsset{
        Path: asset.Path, MediaType: asset.MediaType, SHA256: asset.SHA256,
        Content: append([]byte(nil), asset.Content...),
    }
}

func HTMLRequirementCapability(requirement HTMLRequirement) (string, error) {
    if err := requirement.validate(); err != nil { return "", err }
    value := htmlRequirementCapabilityValue{
        ID: requirement.ID, Kind: requirement.Kind, LocalURL: requirement.LocalURL,
        Integrity: requirement.Integrity, LoadAfter: append([]string(nil), requirement.LoadAfter...),
        Inline: htmlRequirementCapabilityAssetFrom(requirement.Inline),
    }
    data, err := json.Marshal(value)
    if err != nil { return "", err }
    return htmlRequirementCapabilityPrefix + base64.RawURLEncoding.EncodeToString(data), nil
}
```

The decoder rejects unknown JSON fields, a second JSON value, malformed base64, invalid asset bytes or hashes, and invalid requirements. Existing non-prefixed capabilities remain opaque. Reject any inline asset over `4 << 20` bytes and any merged requirement set over `8 << 20` inline bytes. Capability-carried bytes are part of compiler identity; no capability fetches a URL.

- [ ] **Step 5: Attach only used extension requirements**

Decode capability requirements while freezing registry identity. When `buildRenderPlan` records the first node for a registration, merge that registration's requirements into `plan.htmlRequirements`. Copy the merged value into `Document` and `RenderResult` private fields; add package-private accessors for `RenderEditorial`.

- [ ] **Step 6: Prove GREEN and fingerprint sensitivity**

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'Test(HTMLRequirement|ExtensionRequirement|CompilerConfig)' -count=1
GOWORK=off GOFLAGS=-mod=readonly go test -race . -run 'TestHTMLRequirements|TestExtensionRequirement' -count=20
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: PASS; unused extensions contribute nothing, used equivalent requirements dedupe, mutations do not escape, changed capability bytes change compiler identity.

- [ ] **Step 7: REFACTOR and repeat GREEN**

Separate validation, capability transport, and graph ordering helpers inside `html_requirement.go`; repeat Step 6. Expected: PASS.

- [ ] **Step 8: Commit checkpoint**

```bash
git diff --check
git add -- html_requirement.go html_requirement_test.go extension.go registry.go render_plan.go document.go result.go compiler.go fingerprint_test.go
git diff --cached --check
git commit -m "feat: declare editorial HTML requirements"
```

---

### Task 4: Add immutable EditorialResult and fragment contract

**Files:**
- Create: `editorial.go`
- Create: `editorial_fingerprint.go`
- Create: `editorial_test.go`
- Modify: `fingerprint.go`
- Modify: `result.go`

**Interfaces:**
- Consumes: `RenderResult.Content`, normalized `Metadata`, private result requirements, diagnostics, canonical JSON, and `templ.Component`.
- Produces: `EditorialResult`, `EditorialMetadata`, `EditorialOption`, `WithEditorialHeader`, `RenderEditorial`, `PlainText`, `Requirements`, `Diagnostics`, and `EditorialFingerprint`.

- [ ] **Step 1: Write failing fragment, title, plain-text, and identity tests**

```go
func TestRenderEditorialProducesHostOwnedFragment(t *testing.T) {
    result := mustRenderSource(t, "---\ntitle: Host title\n---\n\nBody with **meaning**.\n")
    editorial, err := RenderEditorial(result)
    if err != nil { t.Fatal(err) }
    markup := renderComponent(t, editorial.Fragment())
    for _, forbidden := range []string{"<!doctype", "<html", "<head", "<body", "<script", "<style", "data-theme="} {
        if strings.Contains(strings.ToLower(markup), forbidden) { t.Fatalf("fragment owns shell %q: %s", forbidden, markup) }
    }
    if strings.Count(markup, `<article class="margo-document">`) != 1 { t.Fatalf("article contract: %s", markup) }
    if editorial.PlainText() != "Body with meaning." { t.Fatalf("plain = %q", editorial.PlainText()) }
}
```

Add tests for header insertion only under `WithEditorialHeader`, metadata/H1 conflict warning, nil result, defensive slices, equivalent fingerprint, and fingerprint changes for metadata/requirements/options.

- [ ] **Step 2: Prove RED**

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'TestRenderEditorial|TestEditorialFingerprint' -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: compile failure because editorial APIs do not exist.

- [ ] **Step 3: Implement immutable projection and plain text**

Render the content once to private bytes. Parse one fragment with `x/net/html`, reject document-shell nodes, extract first H1 and plain text while skipping `script`, `style`, and `template`, then construct a bytes-backed component.

```go
type EditorialResult struct {
    fragmentBytes []byte
    plainText string
    metadata EditorialMetadata
    requirements HTMLRequirements
    diagnostics []Diagnostic
    fingerprint EditorialFingerprint
}

func (r *EditorialResult) Fragment() templ.Component {
    data := append([]byte(nil), r.fragmentBytes...)
    return templ.ComponentFunc(func(_ context.Context, out io.Writer) error { _, err := out.Write(data); return err })
}
```

`WithEditorialHeader` inserts one semantic article header only when no body H1 exists. A metadata/body H1 mismatch appends `editorial.title_conflict` with `SeverityInfo`.

- [ ] **Step 4: Add canonical editorial fingerprint**

Hash `margo/editorial/v1\n` plus canonical JSON containing fragment SHA-256, normalized metadata, ordered requirement identities/hashes, and `header` option. Exclude caller header/footer publication components.

- [ ] **Step 5: Prove GREEN and race safety**

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'TestRenderEditorial|TestEditorialFingerprint' -count=1
GOWORK=off GOFLAGS=-mod=readonly go test -race . -run 'TestEditorial' -count=20
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: PASS with no shell ownership and immutable repeated output.

- [ ] **Step 6: REFACTOR and repeat GREEN**

Keep HTML traversal private and single-pass; repeat Step 5. Expected: PASS.

- [ ] **Step 7: Commit checkpoint**

```bash
git diff --check
git add -- editorial.go editorial_fingerprint.go editorial_test.go fingerprint.go result.go
git diff --cached --check
git commit -m "feat: expose editorial HTML fragments"
```

---

### Task 5: Add progressive Markdown table sorting and Margo asset mount

**Files:**
- Modify: `assets.go`
- Modify: `assets_test.go`
- Modify: `assets/document.css`
- Create: `assets/table-sort.js`
- Modify: `table_adapter.go`
- Modify: `table_adapter_test.go`
- Modify: `render_plan.go`
- Modify: `html_requirement_test.go`

**Interfaces:**
- Consumes: Goshtoso table markup, root AST table nodes, `HTMLRequirement`, and `/margo-assets/` contract.
- Produces: `EditorialAssetHandler`, `margo.document.styles`, conditional `margo.table-sort`, inert `data-margo-table-sort`, and progressive browser controls.

- [ ] **Step 1: Write failing requirement and markup tests**

Assert a document without a table has only style requirements; one with a table adds exactly one sort requirement and inert wrapper data.

```go
func TestMarkdownTableDeclaresProgressiveSort(t *testing.T) {
    result := mustRenderSource(t, "| Name | Count |\n|---|---:|\n| Item 10 | 10 |\n| Item 2 | 2 |\n")
    editorial, err := RenderEditorial(result)
    if err != nil { t.Fatal(err) }
    markup := renderComponent(t, editorial.Fragment())
    if !strings.Contains(markup, `data-margo-table-sort="natural"`) { t.Fatalf("missing sort marker: %s", markup) }
    if strings.Contains(markup, `<button`) { t.Fatalf("server emitted JS-only control: %s", markup) }
    requireRequirementIDs(t, editorial.Requirements(), "goshtoso.styles", "margo.document.styles", "margo.table-sort")
}
```

- [ ] **Step 2: Prove RED**

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'TestMarkdownTableDeclaresProgressiveSort|TestEditorialAssetHandler' -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: FAIL because no requirement graph entry, JS asset, or new mount exists.

- [ ] **Step 3: Embed and serve Margo editorial assets**

Expand the embed pattern to `assets/*.css assets/*.js assets/*.svg`. Add constants `/margo-assets/document.css` and `/margo-assets/table-sort.js`; extend `assetMediaType` so `.js` returns `application/javascript`. `EditorialAssetHandler` strips `/margo-assets/` itself; test both paths return correct content types and `/assets/` is not accepted by the new handler.

- [ ] **Step 4: Add inert table data and conditional requirements**

Render:

```html
<div data-table-client-sort="true" data-margo-table-sort="natural">
```

During AST planning, every document gets `goshtoso.styles` from `goshtoso/assets.DefaultRuntimeManifest().Stylesheet` plus inline bytes from `goshtoso/assets.StylesCSS()`, and `margo.document.styles` from `EmbeddedAsset("document.css")`; at least one Markdown table adds `margo.table-sort` once. Use `/margo-assets/` URLs and inline embedded bytes with computed SHA-256.

- [ ] **Step 5: Implement progressive runtime**

`assets/table-sort.js` must:

```javascript
(() => {
  "use strict";
  const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: "base" });
  const sourceRows = (tbody) => Array.from(tbody.rows);
  const compare = (a, b) => collator.compare(a.trim(), b.trim());
  const apply = (table, tbody, rows, active) => {
    const ordered = rows.slice().sort((left, right) => {
      const sourceOrder = Number(left.dataset.margoSourceIndex) - Number(right.dataset.margoSourceIndex);
      if (active.state === "source") return sourceOrder;
      const value = compare(left.cells[active.column]?.textContent || "", right.cells[active.column]?.textContent || "");
      if (value === 0) return sourceOrder;
      return active.state === "descending" ? -value : value;
    });
    ordered.forEach((row) => tbody.append(row));
    Array.from(table.tHead.rows[0].cells).forEach((header) => header.removeAttribute("aria-sort"));
    if (active.state !== "source") table.tHead.rows[0].cells[active.column].setAttribute("aria-sort", active.state);
  };
  const installButton = (table, tbody, rows, cell, column, active) => {
    const label = cell.textContent.trim();
    const button = document.createElement("button");
    button.type = "button";
    button.className = "margo-table-sort-button";
    button.textContent = label;
    cell.textContent = "";
    cell.append(button);
    button.addEventListener("click", () => {
      const next = active.column !== column || active.state === "source" ? "ascending" : active.state === "ascending" ? "descending" : "source";
      active.state = next;
      active.column = next === "source" ? -1 : column;
      apply(table, tbody, rows, active);
      button.focus();
    });
  };
  const initialize = (root) => {
    if (root.dataset.margoTableSortReady === "true") return;
    const table = root.querySelector("table");
    const tbody = table && table.tBodies[0];
    if (!table || !tbody || !table.tHead) return;
    const rows = sourceRows(tbody);
    const active = { state: "source", column: -1 };
    rows.forEach((row, index) => { row.dataset.margoSourceIndex = String(index); });
    Array.from(table.tHead.rows[0].cells).forEach((cell, column) => installButton(table, tbody, rows, cell, column, active));
    let printState = { state: "source", column: -1 };
    addEventListener("beforeprint", () => {
      printState = { ...active };
      active.state = "source";
      active.column = -1;
      apply(table, tbody, rows, active);
    });
    addEventListener("afterprint", () => {
      active.state = printState.state;
      active.column = printState.column;
      apply(table, tbody, rows, active);
    });
    root.dataset.margoTableSortReady = "true";
  };
  const scan = () => document.querySelectorAll('[data-margo-table-sort="natural"]').forEach(initialize);
  document.addEventListener("DOMContentLoaded", scan, { once: true });
  document.addEventListener("htmx:afterSettle", scan);
  if (document.readyState !== "loading") scan();
})();
```

- [ ] **Step 6: Prove focused GREEN**

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'Test(MarkdownTable|EditorialAsset|EmbeddedAsset|HTMLRequirement)' -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: PASS; behavior execution remains for tagged E2E in Task 9.

- [ ] **Step 7: REFACTOR and repeat GREEN**

Keep public constants and script selectors stable; remove duplicate style rules; repeat Step 6. Expected: PASS.

- [ ] **Step 8: Commit checkpoint**

```bash
git diff --check
git add -- assets.go assets_test.go assets/document.css assets/table-sort.js table_adapter.go table_adapter_test.go render_plan.go html_requirement_test.go
git diff --cached --check
git commit -m "feat: sort editorial tables progressively"
```

---

### Task 6: Compose complete publication pages and metadata directly

**Files:**
- Create: `publication.go`
- Create: `publication.templ`
- Generate: `publication_templ.go`
- Create: `publication_test.go`
- Modify: `social.go`
- Modify: `social.templ`
- Generate: `social_templ.go`
- Modify: `theme.go`
- Modify: `theme_test.go`

**Interfaces:**
- Consumes: `EditorialResult`, `HTMLRequirements`, `AuthorityRecord`, `SocialImage`, templ, Goshtoso asset manifest.
- Produces: `PublicationKind`, `HTMLDependencyMode`, `PublicationInput`, `RenderPublication`, arbitrary safe theme names, direct article/document head tags.

- [ ] **Step 1: Write failing publication contract tests**

Cover article/document OG type, one metadata set, canonical derivation, arbitrary `araihu` theme, inline/local dependency modes, exact ordering, private URL omission, and zero bytes on invalid input.

```go
func loadPublicationAuthority(t *testing.T) AuthorityRecord {
    t.Helper()
    data, err := os.ReadFile("testdata/authority/record.json")
    if err != nil { t.Fatal(err) }
    record, err := VerifyAuthorityRecord(data)
    if err != nil { t.Fatal(err) }
    return record
}

func mustEditorialFixture(t *testing.T) *EditorialResult {
    t.Helper()
    source, err := os.ReadFile("testdata/markdown/editorial-article.md")
    if err != nil { t.Fatal(err) }
    editorial, err := RenderEditorial(mustRenderSource(t, string(source)))
    if err != nil { t.Fatal(err) }
    return editorial
}

func testThemeStylesheet() AssetRef {
    content := []byte(`[data-theme="araihu"]{--color-surface:#fff;--color-on-surface:#111}`)
    digest := sha256.Sum256(content)
    return AssetRef{Path: "themes/araihu.css", MediaType: "text/css", SHA256: hex.EncodeToString(digest[:]), Content: content}
}

func TestRenderPublicationComposesArticleInInitialHTML(t *testing.T) {
    authority := loadPublicationAuthority(t)
    editorial := mustEditorialFixture(t)
    page, err := RenderPublication(editorial, PublicationInput{
        Mode: PublicationPublic, Kind: PublicationArticle,
        Authority: authority, RoutePath: authority.Routes.Representative,
        SiteName: "Arai Hû", Locale: "pt_BR",
        Image: SocialImage{URL: string(authority.CanonicalOrigin) + authority.Routes.Preview, MIMEType: authority.Asset.MIMEType, Width: authority.Asset.Width, Height: authority.Asset.Height, Alt: "Editorial preview fixture."},
        Theme: ThemeName("araihu"), ColorMode: ColorModeDark,
        DependencyMode: HTMLDependenciesInline, ThemeStylesheet: testThemeStylesheet(),
    })
    if err != nil { t.Fatal(err) }
    markup := renderComponent(t, page)
    for _, want := range []string{`<!doctype html>`, `data-theme="araihu"`, `class="dark"`, `property="og:type" content="article"`, `rel="canonical" href="https://margo.invalid/guide"`, `<address`, `<time`, `data-margo-requirement=`} {
        if !strings.Contains(markup, want) { t.Fatalf("missing %q: %s", want, markup) }
    }
    if strings.Count(markup, `name="description"`) != 1 { t.Fatalf("duplicate metadata: %s", markup) }
}
```

- [ ] **Step 2: Prove RED**

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'TestRenderPublication|TestPublicationInput|TestCustomThemeName' -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: compile failure because publication API and arbitrary theme validation do not exist.

- [ ] **Step 3: Add exact publication types and validation**

```go
type PublicationKind string
const (
    PublicationDocument PublicationKind = "document"
    PublicationArticle  PublicationKind = "article"
)

type HTMLDependencyMode string
const (
    HTMLDependenciesLocal  HTMLDependencyMode = "local"
    HTMLDependenciesInline HTMLDependencyMode = "inline"
)

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

Validate all fields before constructing the templ component. Derive canonical URL from verified origin plus exact listed route. Validate custom theme regex and materialized custom theme CSS. Tests use only `testdata/authority/record.json`; do not create a production-looking authority identity for araihu.com.

- [ ] **Step 4: Compose templ head/body without string replacement**

`publication.templ` owns one document shell. `social.templ` accepts OG type and optional article fields. Render local requirements as one `<link>`/`<script defer>` each; render inline requirements as escaped-safe `<style>`/`<script>` bytes after replacing `</script` with `<\/script`. Every dependency element carries `data-margo-requirement="<requirement ID>"`. Put Goshtoso CSS before Margo CSS and custom theme CSS last. Render non-empty authors, dates, and tags once as semantic `<address>`, `<time>`, and list markup.

- [ ] **Step 5: Regenerate templ output**

```bash
before=$(git hash-object go.mod go.sum)
templates_before=$(git hash-object publication.templ social.templ)
GOWORK=off GOFLAGS=-mod=readonly go tool templ generate
templates_after=$(git hash-object publication.templ social.templ)
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
test "$templates_before" = "$templates_after"
```

Expected: `publication_templ.go` created and `social_templ.go` updated; source `.templ` files unchanged by generation.

- [ ] **Step 6: Prove GREEN**

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'Test(RenderPublication|PublicationInput|CustomThemeName|Social)' -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: PASS; metadata is initial, unique, escaped, authority-bound, and correctly typed.

- [ ] **Step 7: REFACTOR and repeat GREEN**

Share metadata validation without retaining HTML replacement. Repeat Steps 5 and 6. Expected: PASS and no generated drift.

- [ ] **Step 8: Commit checkpoint**

```bash
git diff --check
git add -- publication.go publication.templ publication_templ.go publication_test.go social.go social.templ social_templ.go theme.go theme_test.go
git diff --cached --check
git commit -m "feat: compose editorial publication pages"
```

---

### Task 7: Rebuild standalone and social compatibility over editorial output

**Files:**
- Modify: `standalone.go`
- Modify: `standalone.templ`
- Generate: `standalone_templ.go`
- Modify: `standalone_test.go`
- Modify: `social.go`
- Create: `social_test.go`

**Interfaces:**
- Consumes: `RenderEditorial`, `RenderPublication`, `PublicationInput`, existing standalone options/brand/TOC/print script.
- Produces: backward-compatible `RenderStandalone`, `Standalone`, and `RenderSocialStandalone` delegating to shared editorial composition.

- [ ] **Step 1: Write failing shared-path tests**

Assert standalone fragment bytes and requirements match `RenderEditorial`, the built-in `minimal` theme remains compatible, and social standalone uses direct head composition without replacing `<title>`. Arbitrary consumer themes are exercised through `RenderPublication` in Tasks 6 and 9.

```go
func TestStandaloneUsesEditorialFragmentExactlyOnce(t *testing.T) {
    result := mustRenderSource(t, "# Shared\n\nBody\n")
    editorial, err := RenderEditorial(result)
    if err != nil { t.Fatal(err) }
    fragment := renderComponent(t, editorial.Fragment())
    standalone, err := RenderStandalone(result, WithStandaloneTheme(ThemeMinimal))
    if err != nil { t.Fatal(err) }
    markup := renderComponent(t, standalone)
    if strings.Count(markup, fragment) != 1 { t.Fatalf("fragment count != 1: %s", markup) }
}
```

- [ ] **Step 2: Prove RED**

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'TestStandaloneUsesEditorial|TestRenderSocialStandaloneDelegates' -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: FAIL because standalone has an independent assembly path and closed themes.

- [ ] **Step 3: Delegate while preserving options**

Map title, description, theme, color mode, TOC, brand, asset overrides, and print preparation into editorial/publication configuration. Keep private standalone output free of canonical/social URLs. Preserve deterministic `ri-00000000`, fingerprint attributes, brand furniture, TOC, and print script.

`RenderSocialStandalone` maps existing `SocialRenderInput` to `PublicationInput`; public output uses exact supplied validated metadata for compatibility, private output delegates without URL metadata. Delete only the old string-replacement branch.

- [ ] **Step 4: Regenerate templ and prove GREEN**

```bash
before=$(git hash-object go.mod go.sum)
templates_before=$(git hash-object standalone.templ social.templ)
GOWORK=off GOFLAGS=-mod=readonly go tool templ generate
templates_after=$(git hash-object standalone.templ social.templ)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'Test(Standalone|RenderSocialStandalone|Social)' -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
test "$templates_before" = "$templates_after"
```

Expected: PASS; standalone behavior remains, shared fragment appears once, generated files are current.

- [ ] **Step 5: Run complete root compatibility gate**

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1
GOWORK=off GOFLAGS=-mod=readonly go test -race . -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: PASS.

- [ ] **Step 6: REFACTOR and repeat GREEN**

Remove dead duplicate head/style helpers while retaining exported compatibility APIs. Repeat Steps 4 and 5. Expected: PASS.

- [ ] **Step 7: Commit checkpoint**

```bash
git diff --check
git add -- standalone.go standalone.templ standalone_templ.go standalone_test.go social.go social_test.go
git diff --cached --check
git commit -m "refactor: share editorial standalone path"
```

---

### Task 8: Declare and externalize chart control requirements compatibly

**Files:**
- Modify: `charts/extension.go`
- Modify: `charts/extension_test.go`
- Modify: `charts/controls.go`
- Modify: `charts/controls_test.go`
- Create: `charts/testdata/markdown/editorial-charts.md`

**Interfaces:**
- Consumes: existing `ExtensionIdentity.Capabilities`, standard-library JSON/base64, Goshtoso runtime manifest roles, `goshtoso-charts/assets.ControlRuntimeURL`, and root capability protocol.
- Produces: chart capability envelopes with materialized inline bytes, exact-loader suppression when declared, one requirement set per used charts extension, and an explicitly preserved legacy optimistic-renderer fallback for its pinned old-root identity.

- [ ] **Step 1: Write failing capability and loader tests**

Assert enabled wrapper registration includes four decodable capability envelopes; disabled wrapper includes none. Render four charts and assert no per-chart control loader remains when externalization is enabled, while static SVG/data tables remain.

```go
type requirementCapabilityValue struct {
    ID        string   `json:"id"`
    Kind      string   `json:"kind"`
    LocalURL  string   `json:"localURL,omitempty"`
    Integrity string   `json:"integrity,omitempty"`
    LoadAfter []string `json:"loadAfter,omitempty"`
    Inline    *requirementCapabilityAsset `json:"inline,omitempty"`
}

type requirementCapabilityAsset struct {
    Path      string `json:"path"`
    MediaType string `json:"mediaType"`
    SHA256    string `json:"sha256"`
    Content   []byte `json:"content"`
}

func decodeRequirementCapabilities(t *testing.T, capabilities []string) []requirementCapabilityValue {
    t.Helper()
    const prefix = "margo-html-requirement/v1:"
    values := make([]requirementCapabilityValue, 0, len(capabilities))
    for _, capability := range capabilities {
        if !strings.HasPrefix(capability, prefix) { continue }
        data, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(capability, prefix))
        if err != nil { t.Fatal(err) }
        var value requirementCapabilityValue
        decoder := json.NewDecoder(bytes.NewReader(data))
        decoder.DisallowUnknownFields()
        if err := decoder.Decode(&value); err != nil { t.Fatal(err) }
        if err := decoder.Decode(new(any)); err != io.EOF { t.Fatalf("trailing capability data: %v", err) }
        values = append(values, value)
    }
    return values
}

func requireCapabilityIDs(t *testing.T, values []requirementCapabilityValue, expected ...string) {
    t.Helper()
    got := make([]string, len(values))
    for index := range values { got[index] = values[index].ID }
    if !slices.Equal(got, expected) { t.Fatalf("capability IDs = %v, want %v", got, expected) }
}

func TestExtensionDeclaresEnabledControlRequirements(t *testing.T) {
    registration := Extension(WithExternalizedControlRuntime(true))
    capabilities := decodeRequirementCapabilities(t, registration.Identity.Capabilities)
    if len(capabilities) != 4 { t.Fatalf("requirements = %#v", capabilities) }
    requireCapabilityIDs(t, capabilities,
        "goshtoso.runtime.alpine-focus", "goshtoso.runtime.first-party",
        "goshtoso.runtime.alpine", "goshtoso-charts.controls")
}
```

- [ ] **Step 2: Prove RED under ordinary charts identity**

```bash
cd charts
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'TestExtensionDeclares|TestChartControlLoader' -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: compile failure because `WithExternalizedControlRuntime` does not exist.

- [ ] **Step 3: Add extension-local capability encoder**

Add `externalizedControlRuntime bool` to `chartRenderOptions`. When true, append four `margo-html-requirement/v1:` base64url JSON strings to capabilities. Build values from Goshtoso/Goshtoso Charts exported manifest URLs, roles, and integrity; do not call new root APIs.

```go
type requirementCapabilityValue struct {
    ID        string   `json:"id"`
    Kind      string   `json:"kind"`
    LocalURL  string   `json:"localURL,omitempty"`
    Integrity string   `json:"integrity,omitempty"`
    LoadAfter []string `json:"loadAfter,omitempty"`
    Inline    *requirementCapabilityAsset `json:"inline,omitempty"`
}

type requirementCapabilityAsset struct {
    Path      string `json:"path"`
    MediaType string `json:"mediaType"`
    SHA256    string `json:"sha256"`
    Content   []byte `json:"content"`
}

func requirementCapability(value requirementCapabilityValue) string {
    data, err := json.Marshal(value)
    if err != nil { panic(err) }
    return "margo-html-requirement/v1:" + base64.RawURLEncoding.EncodeToString(data)
}
```

Materialize each local URL with its owning embedded handler using `httptest.NewRequest` and `httptest.NewRecorder`; require HTTP 200, compute the SHA-256 over the returned immutable bytes, use `application/javascript`, and strip only the exact owning mount prefix to form the `AssetRef` path. `Extension` stores any materialization error in the frozen factory closure and contributes no partial capability list; that factory returns the deterministic error before rendering a node. It never downloads. Declare requirements only when both the control wrapper and externalization are enabled. Exact load order: focus; first-party after focus; Alpine after first-party; controls after Alpine.

- [ ] **Step 4: Suppress only declared upstream loader tags**

Extend the private chart writer to remove exact `<script src="` + `assets.ControlRuntimeURL` + `" defer></script>` sequences only when externalization is true. Any changed or malformed loader remains visible and fails a focused test rather than being broadly stripped. Default false preserves old-root tool behavior.

- [ ] **Step 5: Preserve the pinned old-root optimistic renderer path**

Do not change `charts/tools/optimistic-renderer` in this task. It compiles under `GOWORK=off` against the pinned published root, which cannot consume the new capability envelope. It therefore keeps default non-externalized charts plus its existing exact-loader inline fallback. Task 9 proves the new declarative path independently with `WithExternalizedControlRuntime(true)` in the temporary current-root workspace. Removal of the legacy fallback requires a later verified charts root-version update; do not fabricate that identity here.

- [ ] **Step 6: Prove GREEN under GOWORK=off**

```bash
cd charts
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run 'Test(ExtensionDeclares|ChartControl)' -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: PASS without root module or charts module drift.

- [ ] **Step 7: REFACTOR and repeat GREEN**

Share chart option hashing with the new boolean so configuration identity changes. Repeat Step 6. Expected: PASS.

- [ ] **Step 8: Commit checkpoint**

```bash
git diff --check
git add -- charts/extension.go charts/extension_test.go charts/controls.go charts/controls_test.go charts/testdata/markdown/editorial-charts.md
git diff --cached --check
git commit -m "feat(charts): declare editorial control runtime"
```

---

### Task 9: Prove Manja and araihu.com generated HTML in Chromium

**Files:**
- Create: `charts/e2e/editorial_html_test.go`
- Modify: `charts/go.mod`
- Modify: `charts/go.sum`
- Create: `docs/testing/editorial-html.md`

**Interfaces:**
- Consumes: current root and charts modules through temporary workspace, `RenderEditorial`, `RenderPublication`, `EditorialAssetHandler`, Goshtoso and charts asset handlers, chromedp.
- Produces: tagged consumer-shaped browser proof and tested-browser evidence.

- [ ] **Step 1: Write tagged failing E2E journeys**

Create `charts/e2e/editorial_html_test.go` with `//go:build editorial_e2e`. Start one `httptest.Server` mounting `/assets/`, `/margo-assets/`, and `/charts/assets/`. Serve:

- `/manja`: a host-owned Goshtoso page embedding `EditorialResult.Fragment()` inside `.manja-markdown`;
- `/guide`: `RenderPublication` in local dependency mode with theme `araihu`, article metadata, one sortable table, and four charts using externalized runtime;
- `/guide-inline`: the same verified editorial result in inline dependency mode, with no runtime asset request.

Load `testdata/authority/record.json` through `VerifyAuthorityRecord`; use its listed representative route for canonical assertions. The additional private inline route is functional fixture evidence and does not fabricate an araihu.com authority identity.

Use chromedp to assert:

```go
func TestGeneratedEditorialHTMLJourneys(t *testing.T) {
    browserPath := requireInstalledChromium(t)
    t.Logf("tested browser: %s", chromiumVersion(t, browserPath))
    ctx, cancel := chromedp.NewExecAllocator(context.Background(), append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath(browserPath))...)
    defer cancel()
    ctx, cancel = chromedp.NewContext(ctx)
    defer cancel()
    runManjaFragmentJourney(t, ctx)
    runPublicationJourney(t, ctx, "/guide")
    runInlinePublicationJourney(t, ctx, "/guide-inline")
}
```

The Manja journey changes `<html data-theme>` and `.dark` without replacing the article, then asserts computed token-backed colors change. The publication journey clicks a sort header through source/ascending/descending/source, checks `aria-sort`, opens/closes a chart modal, and confirms four static SVGs plus four accessible tables remain.

- [ ] **Step 2: Add exact chromedp dependency**

Record charts module hashes, then perform the only permitted charts module write:

```bash
cd charts
pre_mod=$(git hash-object go.mod)
pre_sum=$(git hash-object go.sum)
GOWORK=off GOFLAGS=-mod=mod go get github.com/chromedp/chromedp@v0.14.2
GOWORK=off GOFLAGS=-mod=readonly go mod verify
post_mod=$(git hash-object go.mod)
post_sum=$(git hash-object go.sum)
test "$pre_mod" != "$post_mod"
test "$pre_sum" != "$post_sum"
```

Expected: exact v0.14.2 direct requirement and verified sums; no root module file changes.

- [ ] **Step 3: Prove RED in a temporary workspace**

From repository root:

```bash
workspace_dir=$(mktemp -d)
trap 'find "$workspace_dir" -depth -delete' EXIT
repo_root=$(pwd -P)
(cd "$workspace_dir" && GOWORK=off go work init "$repo_root" "$repo_root/charts")
root_before=$(git hash-object go.mod go.sum)
charts_before=$(git -C charts hash-object go.mod go.sum)
GOWORK="$workspace_dir/go.work" GOFLAGS=-mod=readonly go test -tags=editorial_e2e ./charts/e2e -run TestGeneratedEditorialHTMLJourneys -count=1 -v
root_after=$(git hash-object go.mod go.sum)
charts_after=$(git -C charts hash-object go.mod go.sum)
test "$root_before" = "$root_after"
test "$charts_before" = "$charts_after"
```

Expected: FAIL first on missing browser journey behavior or incomplete dependency assembly, never on a fabricated module identity.

- [ ] **Step 4: Complete browser assertions and network evidence**

Listen to `network.EventRequestWillBeSent` and `runtime.EventExceptionThrown`. For `/guide`, assert each local requirement URL loads once. For `/guide-inline`, assert no `/assets/`, `/margo-assets/`, or `/charts/assets/` request occurs after navigation. In both runs, assert no external URL is requested, no request fails, no exception/page error occurs, and duplicate DOM IDs are absent. Fetch initial HTML separately and parse it to prove canonical/OG/X/article metadata before browser execution.

Add a JavaScript-disabled allocator run that asserts article text, semantic table rows, four SVGs, and four chart data tables are visible/readable while sort/expand controls are absent or inert.

- [ ] **Step 5: Prove GREEN and record browser evidence**

Repeat the Step 3 command. Expected: PASS. Copy the exact logged browser product/version and operating system into `docs/testing/editorial-html.md`; state that this is the tested environment, not a required user browser version.

- [ ] **Step 6: Prove ordinary module isolation**

```bash
root_before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1
root_after=$(git hash-object go.mod go.sum)
test "$root_before" = "$root_after"

charts_before=$(git -C charts hash-object go.mod go.sum)
(cd charts && GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1)
charts_after=$(git -C charts hash-object go.mod go.sum)
test "$charts_before" = "$charts_after"
```

Expected: PASS; tagged cross-module tests do not contaminate ordinary module gates.

- [ ] **Step 7: REFACTOR and repeat GREEN**

Extract browser helpers only inside `charts/e2e`; repeat Steps 3, 5, and 6. Expected: PASS with unchanged browser assertions.

- [ ] **Step 8: Commit checkpoint**

```bash
git diff --check
git add -- charts/e2e/editorial_html_test.go charts/go.mod charts/go.sum docs/testing/editorial-html.md
git diff --cached --check
git commit -m "test: verify editorial HTML consumers"
```

---

### Task 10: Publish integration guidance and run final gates

**Files:**
- Create: `editorial_docs_test.go`
- Modify: `README.md`
- Modify: `charts/README.md`
- Modify: `PRODUCT.md`
- Modify: `docs/testing/editorial-html.md`
- Modify: `docs/superpowers/specs/2026-08-09-margo-editorial-html-design.md` only if implementation reality requires a non-behavioral clarification

**Interfaces:**
- Consumes: completed APIs, asset mounts, browser evidence, Manja/araihu.com fixtures.
- Produces: exact consumer examples, migration notes, final verification record.

- [ ] **Step 1: Write documentation contract test**

Create `editorial_docs_test.go` in root. It reads README/testing docs and asserts exact public symbols, three asset mounts, two output modes, tested-browser wording, and no claim that user Chromium is pinned.

```go
func TestEditorialDocumentationNamesPublicContract(t *testing.T) {
    readme := readRepoFile(t, "README.md")
    for _, want := range []string{"RenderEditorial", "RenderPublication", "EditorialAssetHandler", "/assets/", "/margo-assets/", "/charts/assets/"} {
        if !strings.Contains(readme, want) { t.Fatalf("README missing %q", want) }
    }
}
```

- [ ] **Step 2: Prove RED**

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run TestEditorialDocumentationNamesPublicContract -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: FAIL because README lacks the new contract.

- [ ] **Step 3: Add exact consumer examples**

README must show:

```go
compiler := margo.New()
document, err := compiler.Compile(ctx, margo.Source{Name: "description.md", Content: source})
if err != nil { return err }
rendered, err := compiler.Render(ctx, document)
if err != nil { return err }
editorial, err := margo.RenderEditorial(rendered)
if err != nil { return err }
return editorial.Fragment().Render(ctx, writer)
```

Also show handlers at exact mounts and `RenderPublication` for an article. Explain host theme inheritance, requirement satisfaction, JavaScript fallbacks, charts externalization, and downstream Manja/araihu.com ownership.

- [ ] **Step 4: Prove docs GREEN**

```bash
before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go test . -run TestEditorialDocumentationNamesPublicContract -count=1
after=$(git hash-object go.mod go.sum)
test "$before" = "$after"
```

Expected: PASS.

- [ ] **Step 5: Run generation and full module gates**

```bash
root_before=$(git hash-object go.mod go.sum)
GOWORK=off GOFLAGS=-mod=readonly go tool templ generate
git diff --exit-code -- publication_templ.go social_templ.go standalone_templ.go
GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1
GOWORK=off GOFLAGS=-mod=readonly go test -race . -count=1
root_after=$(git hash-object go.mod go.sum)
test "$root_before" = "$root_after"

charts_before=$(git -C charts hash-object go.mod go.sum)
(cd charts && GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1)
charts_after=$(git -C charts hash-object go.mod go.sum)
test "$charts_before" = "$charts_after"
```

Expected: PASS with no module drift or generated templ drift.

- [ ] **Step 6: Run final cross-module browser gate**

```bash
workspace_dir=$(mktemp -d)
trap 'find "$workspace_dir" -depth -delete' EXIT
repo_root=$(pwd -P)
(cd "$workspace_dir" && GOWORK=off go work init "$repo_root" "$repo_root/charts")
root_before=$(git hash-object go.mod go.sum)
charts_before=$(git -C charts hash-object go.mod go.sum)
GOWORK="$workspace_dir/go.work" GOFLAGS=-mod=readonly go test -tags=editorial_e2e ./charts/e2e -run TestGeneratedEditorialHTMLJourneys -count=1 -v
root_after=$(git hash-object go.mod go.sum)
charts_after=$(git -C charts hash-object go.mod go.sum)
test "$root_before" = "$root_after"
test "$charts_before" = "$charts_after"
```

Expected: PASS with tested browser version matching the documentation record for this implementation run.

- [ ] **Step 7: Audit goal requirements**

Record direct evidence in `docs/testing/editorial-html.md` for each row:

| Requirement | Evidence |
|---|---|
| Manja-compatible fragment | initial HTML and browser fixture |
| araihu.com blog page | publication fixture and social metadata assertions |
| Goshtoso theme inheritance | live built-in/custom plus light/dark computed styles |
| Table sorting | four-state browser journey and ARIA assertions |
| Chart controls | expand/close journey with SVG/data fallback |
| JavaScript-free readability | disabled-JS browser journey |
| No duplicate runtime | network request counts and requirement graph tests |
| PDF deferred | explicit scope statement; no PDF path changed |

- [ ] **Step 8: Commit final checkpoint**

```bash
git diff --check
git add -- README.md charts/README.md PRODUCT.md docs/testing/editorial-html.md editorial_docs_test.go
if ! git diff --quiet -- docs/superpowers/specs/2026-08-09-margo-editorial-html-design.md; then
  git add -- docs/superpowers/specs/2026-08-09-margo-editorial-html-design.md
fi
git diff --cached --check
git commit -m "docs: document editorial HTML integration"
```

- [ ] **Step 9: Freeze review identity without integration**

```bash
git status --short --branch
git diff --check
git rev-parse HEAD HEAD^{tree}
git log --oneline --decorate -12
```

Expected: clean `feat/html-editorial-contract`, exact checkpoint history available for independent review. Do not push, merge, tag, release, publish, deploy, or remove the worktree.

## Spec Coverage Audit

| Approved design section | Implemented by |
|---|---|
| One render, two projections | Tasks 4, 6, 7 |
| Extension source position | Task 1 |
| Host-owned theme | Tasks 4, 6, 9 |
| Typed requirements | Tasks 3, 5, 8 |
| Static fallback and client behavior | Tasks 5, 8, 9 |
| Editorial metadata/title policy | Tasks 2, 4 |
| Publication head | Task 6 |
| Manja and araihu.com boundaries | Tasks 9, 10 |
| Errors, immutability, identity | Tasks 2-6 |
| Unit and generated-HTML E2E | Tasks 1-9 |
| Tested browser evidence, not policy pin | Tasks 9, 10 |
| PDF deferred | Global constraints and Task 10 audit |
