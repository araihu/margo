# Margo deck compositions R1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the approved R1 `composition` directive and nine deterministic Margo deck compositions while preserving all v0.0.1 decks that do not opt in.

**Architecture:** Keep composition normalization inside `deck/`. Parse an inherited or slide-local composition name, resolve it through a closed R1 catalog, and normalize it into an immutable composition spec plus an existing or newly controlled layout family. Render semantic composition and slot metadata into the fixed deck DOM, then include the catalog and resolved slide inputs in the deck runtime/PDF evidence digest.

**Tech Stack:** Go, `gopkg.in/yaml.v3`, existing Margo TargetDeck renderer, embedded deck CSS/JavaScript, Chromium deck validation, canonical JSON, and the existing Marpit comparator.

**Spec:** `docs/superpowers/specs/2026-08-19-margo-compositions-r1-design.md`

## Global Constraints

- Preserve the existing v0.0.1 `class` plus `layout/slot` contract for input that does not use `composition`.
- Accept only the closed R1 names: `content`, `agenda`, `media-split`, `media-stage`, `steps`, `highlight`, `compare-grid`, `hero`, and `image-grid`, plus `none` for clearing.
- Keep `CompositionCatalogVersion` exactly `r1` in normalized manifests, root HTML metadata, task input digests, and R1 validation envelopes.
- Never infer a composition from headings, images, tables, CSS, or rendered HTML.
- Preserve source, DOM, reading, keyboard, and print order; CSS must not reorder slots.
- Reject arbitrary HTML/CSS/JavaScript, custom composition registration, external template assets, and copied TOTVS material.
- Every production behavior follows RED, a targeted failing test, then GREEN, minimal implementation, then refactor while green.
- Do not create commits, push, PRs, merges, tags, releases, or deploys during execution unless separately authorized.
- Use the existing isolated worktree `/private/tmp/margo-marpit-decks-v001`; do not create a second worktree.

---

### Task 1: Freeze the R1 catalog and immutable model

**Files:**
- Create: `deck/composition.go`
- Create: `deck/composition_test.go`
- Modify: `deck/model.go`
- Test: `deck/composition_test.go`

**Interfaces:**
- Consumes: validated lowercase composition names from the parser in later tasks.
- Produces: `CompositionCatalogVersion`, `CompositionName`, `CompositionSlot`, `CompositionSpec` (including `MinSlots` and `MaxSlots`), `ResolveComposition`, and defensive `Slide.Composition()` / `DirectiveState.Composition` accessors for later tasks.

- [x] **Step 1: Write the failing catalog and clone tests.** Add tests named `TestCompositionCatalogR1ContainsEveryApprovedName`, `TestResolveCompositionRejectsUnknownName`, `TestResolveCompositionMapsGridVariants`, and `TestCompositionSpecCloneDoesNotShareSlots`. Assert all nine names, `r1`, exact family/variant/cardinality, `deck.composition_invalid`, and mutation isolation.

- [x] **Step 2: Run the focused tests and verify RED.**

  Run: `GOWORK=off go test ./deck -run 'Test(Composition|ResolveComposition)' -count=1`

  Expected: compilation or assertion failure because the R1 types and resolver do not exist. Do not add production code before observing this failure.

- [x] **Step 3: Add the closed catalog and model types.** Define the exact fields from the spec. Register `content`, `agenda`, `media-split`, `media-stage`, `steps`, `highlight`, `compare-grid`, `hero`, and `image-grid`; encode body versus structural slots, family, variant, and cardinality. Make `ResolveComposition("none")` return an empty spec and unknown values return `deck.composition_invalid`.

- [x] **Step 4: Add defensive copies.** Add composition state to `DirectiveState` and the normalized spec to `Slide`. Extend `cloneDirectiveState`, `Document.Slides`, and the new accessor so returned slot slices cannot mutate the parsed document.

- [x] **Step 5: Run the focused tests and verify GREEN.**

  Run: `GOWORK=off go test ./deck -run 'Test(Composition|ResolveComposition)' -count=1`

  Expected: all focused tests pass with no warnings.

- [x] **Step 6: Refactor without changing behavior.** Keep the catalog table readable and centralize cardinality checks so later parser/layout code calls one resolver instead of duplicating R1 names.

**Checkpoint:** The catalog is finite, versioned, and independently testable; no parser or renderer behavior has changed yet. Focused and package tests pass.

### Task 2: Parse frontmatter, inherited, local, and spot composition directives

**Files:**
- Modify: `deck/directives.go`
- Modify: `deck/parse.go`
- Modify: `deck/model.go`
- Modify: `deck/layout_test.go`
- Create: `deck/composition_parse_test.go`

**Interfaces:**
- Consumes: `ResolveComposition` and `CompositionName` from Task 1.
- Produces: frontmatter `composition`, body `composition`, spot `_composition`, `none` clear behavior, and source-line diagnostics for layout integration.

- [x] **Step 1: Write failing parser tests.** Add `TestParseCompositionFrontmatterDefault`, `TestParseCompositionBodyInheritanceAndSpotClear`, `TestParseCompositionRejectsMalformedValues`, and `TestParseCompositionInsideFenceRemainsMarkdown`. Assert the frontmatter default is inherited, a body comment changes subsequent slide state, `_composition: none` clears one slide, sequence/mapping/empty/mixed-case/unknown values fail with `deck.composition_invalid`, and a fenced comment is rendered as code rather than parsed.

- [x] **Step 2: Run the focused parser tests and verify RED.**

  Run: `GOWORK=off go test ./deck -run 'TestParseComposition' -count=1`

  Expected: failure because frontmatter treats the new key as unrelated and body comments are not recognized as composition state.

- [x] **Step 3: Register the directive grammar.** Add `composition` to the local recognized directive set, validate only lowercase R1 names or `none`, and keep `composition` out of the existing global body scan so local and spot semantics remain possible.

- [x] **Step 4: Parse the frontmatter default explicitly.** In `parseMetadata`, special-case the deck-owned frontmatter key, validate it with the same node validator, and apply it to the inherited directive state. Do not change how unrelated host frontmatter is preserved.

- [x] **Step 5: Apply body and spot state.** Extend directive application, cloning, inheritance, and clear semantics. A non-spot body event updates inherited state at its source position; a spot event affects only the current slide. Ensure recognized malformed comments fail with the original line number.

- [x] **Step 6: Run the focused parser tests and verify GREEN.**

  Run: `GOWORK=off go test ./deck -run 'TestParseComposition' -count=1`

  Expected: all parser tests pass, including the fenced-code guard.

- [x] **Step 7: Run existing directive/layout regressions.**

  Run: `GOWORK=off go test ./deck -run 'TestParseDirectiveStateNotesAndBackgroundReset|TestParseStructuralLayoutSlots|TestParseRejectsMalformedDirectivesAndLayoutCardinality' -count=1`

  Expected: existing v0.0.1 tests remain green.

**Checkpoint:** Source composition state is available to the slide resolver without changing ordinary Markdown or old deck directives. Focused parser tests and v0.0.1 directive/layout regressions pass.

### Task 3: Resolve composition slots and implicit layouts

**Files:**
- Modify: `deck/layout.go`
- Modify: `deck/parse.go`
- Modify: `deck/directives.go`
- Create: `deck/composition_layout_test.go`
- Modify: `deck/layout_test.go`

**Interfaces:**
- Consumes: parsed composition state and catalog entries from Tasks 1-2.
- Produces: `Layout` plus `CompositionSpec` for every composed slide, explicit matching-marker support, implicit family support, and stable composition diagnostics.

- [x] **Step 1: Write failing normalization tests.** Add `TestParseCompositionImplicitLayout`, `TestParseCompositionAcceptsMatchingLayoutMarker`, `TestParseCompositionRejectsClassLayoutConflict`, `TestParseCompositionRejectsCompositionSlotErrors`, and `TestParseCompositionBodyCompositionsDoNotRequireSlots`. Cover media split, agenda, steps, compare-grid, image-grid, hero, highlight, and content.

- [x] **Step 2: Run the focused layout tests and verify RED.**

  Run: `GOWORK=off go test ./deck -run 'Test(ParseComposition|CompositionLayout)' -count=1`

  Expected: failure because the existing builder rejects slots outside an explicit layout and does not know the `grid` family or semantic composition names.

- [x] **Step 3: Add composition-aware builder state.** Let a composed structural slide collect explicit slots without requiring an initial `layout` marker. Keep the existing explicit builder path unchanged for uncomposed v0.0.1 slides.

- [x] **Step 4: Normalize the resolved family.** For a composition without a marker, create the catalog family. For a matching marker, accept it. For a mismatched marker or explicit class, return `deck.composition_conflict` with the source location.

- [x] **Step 5: Validate exact semantic slots.** Validate body compositions as one body role. Validate media split/stage as `media`, `content`; agenda as `item-1` through `item-6`; steps as `step-1` through `step-6`; compare-grid as `item-1` through `item-4`; image-grid as `image-1` through `image-4`. Enforce the catalog cardinality and source order without renaming or reordering input.

- [x] **Step 6: Add the controlled `grid` family to the layout catalog.** Keep it inaccessible as an arbitrary author class in v0.0.1; only the R1 resolver may produce it. Make explicit `class: grid` without `composition` fail as an unsupported v0.0.1 class.

- [x] **Step 7: Run the focused tests and verify GREEN.**

  Run: `GOWORK=off go test ./deck -run 'Test(ParseComposition|CompositionLayout)' -count=1`

  Expected: all composition normalization tests pass.

- [x] **Step 8: Run all existing layout tests.**

  Run: `GOWORK=off go test ./deck -run 'TestParseStructuralLayoutSlots|TestParseRejectsMalformedDirectivesAndLayoutCardinality|TestParseCross' -count=1`

  Expected: old structural markers, slot names, and cross-boundary protections remain green.

**Checkpoint:** Every R1 source shape normalizes to one immutable composition spec and, where needed, one controlled structural layout.

### Task 4: Render composition metadata, semantics, and visual variants

**Files:**
- Modify: `deck/render.go`
- Modify: `deck/page.go`
- Modify: `deck/assets/deck.css`
- Create: `deck/composition_render_test.go`
- Modify: `deck/render_test.go`

**Interfaces:**
- Consumes: normalized `CompositionSpec` and `Layout` from Task 3.
- Produces: root catalog metadata, composed slide/slot attributes, localized labels, controlled grid CSS, and R1 variant selectors.

- [x] **Step 1: Write failing HTML and CSS tests.** Add `TestRenderCompositionDataAttributesAndSemanticSlots`, `TestRenderUncomposedSlideHasNoCompositionMetadata`, `TestRenderCompositionLabelsFollowSlideLanguage`, and `TestDeckCSSDeclaresR1GridAndVariants`. Assert root `data-margo-composition-catalog="r1"` only for an R1 result, semantic slot attributes in source order, no metadata on old input, and the `grid`, `media-stage`, `agenda`, `highlight`, `hero`, `compare-grid`, and `image-grid` hooks.

- [x] **Step 2: Run the focused render tests and verify RED.**

  Run: `GOWORK=off go test ./deck -run 'Test(RenderComposition|DeckCSS)' -count=1`

  Expected: failure because the current article has no composition data attributes and no R1 grid/variant CSS.

- [x] **Step 3: Render root and slide metadata.** Add the catalog data attribute when any R1 composition is active, add per-slide name and variant, and preserve the old article output for a deck with no composition. Keep attribute values escaped and deterministic.

- [x] **Step 4: Render semantic slot metadata.** Emit `data-margo-slot`, `data-margo-slot-role`, and the resolved family on composed layout nodes. Preserve ordered-list semantics for agenda and steps, and use the existing `role="group"` pattern for card grids with localized labels.

- [x] **Step 5: Add localized composition labels.** Extend the existing deck label catalog for agenda, steps, compare-grid, image-grid, and the body compositions. Keep `lang` inherited from the slide and avoid changing labels for uncomposed slides.

- [x] **Step 6: Add controlled CSS.** Implement equal-track `grid` geometry for 2-4 slots, plus variant selectors for media-stage, agenda, highlight, hero, compare-grid, and image-grid. Use existing theme tokens, card padding, gap, fixed logical canvas, print reset, and no source-order reordering.

- [x] **Step 7: Run focused render tests and verify GREEN.**

  Run: `GOWORK=off go test ./deck -run 'Test(RenderComposition|DeckCSS)' -count=1`

  Expected: all HTML, localization, and CSS assertions pass.

- [x] **Step 8: Run existing render and accessibility regressions.**

  Run: `GOWORK=off go test ./deck -run 'TestRender|TestRenderKeeps|TestRenderNamespaces' -count=1`

  Expected: existing target projections, IDs, chart parity, and runtime requirements remain green.

**Checkpoint:** R1 composition intent is visible in a stable semantic DOM and the visual system renders every catalog family through bounded CSS.

### Task 5: Bind composition identity to runtime and PDF evidence

**Files:**
- Modify: `deck/runtime.go`
- Modify: `deck/pdf_evidence.go`
- Create: `deck/composition_runtime_test.go`
- Modify: `deck/runtime_test.go`
- Modify: `deck/pdf_evidence_test.go`

**Interfaces:**
- Consumes: resolved composition specs from Task 3 and rendered result metadata from Task 4.
- Produces: catalog-aware layout task digests, R1 envelope field, legacy envelope behavior, and stable catalog mismatch diagnostics.

- [x] **Step 1: Write failing runtime tests.** Add `TestLayoutTaskDigestIncludesCompositionIdentity`, `TestCanonicalLayoutEnvelopeIncludesCompositionCatalogVersion`, `TestLegacyLayoutEnvelopeWithoutCompositionCatalogRemainsValid`, and `TestCompositionCatalogMismatchFailsClosed`. Assert two equal-length decks with different composition names produce different task input hashes, R1 canonical JSON includes `compositionCatalogVersion`, old envelopes remain accepted under their old path, and a wrong R1 version emits `deck.composition_catalog_mismatch`.

- [x] **Step 2: Run focused runtime tests and verify RED.**

  Run: `GOWORK=off go test ./deck -run 'Test(LayoutTask|CanonicalLayout|LegacyLayout|CompositionCatalog)' -count=1`

  Expected: failure because the current digest includes only slide IDs and the envelope has no catalog field.

- [x] **Step 3: Include complete slide composition inputs in the digest.** Extend the digest projection with catalog version, resolved composition, variant, class, family, and ordered slots. Keep the mode, geometry, validation request, document fingerprint, and overflow version in the canonical input.

- [x] **Step 4: Extend the canonical envelope.** Add the R1 catalog version to `LayoutValidationEnvelope`, validate `r1` for composed results, serialize it before hashing, and preserve a compatibility path for old uncomposed envelopes.

- [x] **Step 5: Validate mismatch diagnostics.** Reject evidence requesting an unsupported catalog or an R1 envelope whose catalog identity differs from the rendered result. Do not downgrade a mismatch to a generic layout failure.

- [x] **Step 6: Run focused runtime tests and verify GREEN.**

  Run: `GOWORK=off go test ./deck -run 'Test(LayoutTask|CanonicalLayout|LegacyLayout|CompositionCatalog)' -count=1`

  Expected: all digest, envelope, and mismatch assertions pass.

- [x] **Step 7: Run the full runtime/PDF package regressions.**

  Run: `GOWORK=off go test ./deck ./pdf ./pdf/chromium ./cmd/margo -count=1`

  Expected: all existing descriptor, report, PDF evidence, and CLI tests remain green.

**Checkpoint:** A composition change is observable in deterministic runtime identity, while old runtime artifacts retain their compatibility path.

### Task 6: Add fixtures, documentation, and comparator coverage

**Files:**
- Create: `deck/testdata/compositions-r1.md`
- Create: `deck/testdata/compositions-r1.manifest.json`
- Create: `deck/composition_fixture_test.go`
- Create: `docs/reference/deck-compositions-r1.md`
- Modify: `tools/marpit-compare/make-compare.mjs`
- Modify: `tools/marpit-compare/README.md`

**Interfaces:**
- Consumes: the complete R1 implementation from Tasks 1-5.
- Produces: self-contained fixtures with no external reference assets, public authoring documentation, comparator categories/manifests for R1, and reviewable examples.

- [x] **Step 1: Write the fixture test first.** Add `TestCompositionR1FixtureCoversEveryCatalogEntry`. It must load `deck/testdata/compositions-r1.md`, render every composition under the existing deck compiler, assert one or more slides per entry, and reject missing catalog names.

- [x] **Step 2: Run the fixture test and verify RED.**

  Run: `GOWORK=off go test ./deck -run 'TestCompositionR1Fixture' -count=1`

  Expected: failure because the fixture and test do not exist.

- [x] **Step 3: Add self-contained Markdown fixtures.** Use text, local data URIs or existing test-safe assets, tables, code, charts, and Mermaid only where the current renderer supports them. Include frontmatter default, local override, spot clear, body compositions, every structural slot boundary, and no TOTVS material.

- [x] **Step 4: Add the fixture loader and assertions.** Keep the test deterministic, assert catalog names and resolved families, and use real Margo rendering rather than mocks.

- [x] **Step 5: Update public docs.** Document the R1 directive, examples, slot contracts, compatibility behavior, diagnostics, and explicit non-goals. State that the reference deck is vocabulary-only and that no source assets are shipped.

- [x] **Step 6: Extend comparator coverage.** Add a `Compositions R1` category, composition name/variant fields, and manifest identity. Ensure generated comparator output remains self-contained and does not import the external visual guide or its assets.

- [x] **Step 7: Run fixture, documentation, and generator checks.**

  Run: `GOWORK=off go test ./deck -run 'TestCompositionR1Fixture' -count=1`

  Run: `node --check tools/marpit-compare/make-compare.mjs`

  Expected: fixture and syntax checks pass; comparator metadata lists all nine entries.

**Checkpoint:** Authors have a reproducible, generic R1 reference corpus and the docs explain the exact contract without versioning the source template.

### Task 7: Run exhaustive gates and visual evidence

**Files:**
- Modify: `goal.md` only for evidence and status; keep it untracked.
- Evidence outside repository: generated comparator/PDF/PNG sidecars under `/tmp`.

**Interfaces:**
- Consumes: all implementation and fixture outputs from Tasks 1-6.
- Produces: requirement-by-requirement evidence, updated goal state, and a review packet. It does not publish Git state.

- [x] **Step 1: Run the full Go suite.**

  Run: `GOWORK=off go test ./... -count=1`

  Expected: every package passes, including all old and R1 deck tests.

- [x] **Step 2: Run static and module gates.**

  Run: `GOWORK=off go vet ./...`

  Run: `GOWORK=off go mod verify`

  Run: `gofmt -l .`

  Run: `git diff --check`

  Run: `node --check tools/marpit-compare/make-compare.mjs`

  Expected: no vet output, verified modules, no unformatted Go files, no diff errors, and valid generator syntax.

- [x] **Step 3: Render and inspect all compositions.** Generate 16:9 and 4:3 HTML/PDF evidence across modern, goshtoso, and minimal themes. Verify one logical page per slide, no horizontal or vertical logical overflow, stable source order, required alternatives, and catalog identity.

- [x] **Step 4: Exercise comparator review.** Serve the generated comparator, inspect desktop and narrow viewport modes, open paired and 1:1 views, verify all R1 images/manifests load, and record console, focus, and overflow evidence.

- [x] **Step 5: Update `goal.md`.** Record the exact commands, evidence paths, remaining P2 gaps, and whether the R1 cycle is complete. Do not stage the goal file.

- [x] **Step 6: Stop at the authorization boundary.** Report the diff, gates, and evidence. Do not commit, push, open a PR, merge, tag, release, or deploy without a separate explicit instruction.

**Checkpoint:** Completion is proven only when catalog, parser, DOM, CSS, runtime/PDF identity, fixtures, docs, and visual evidence agree.

## Self-review against the spec

| Spec requirement | Plan coverage |
| --- | --- |
| Nine closed R1 entries and `r1` identity | Tasks 1 and 5 |
| Frontmatter, body, spot, and `none` grammar | Task 2 |
| v0.0.1 compatibility and conflict policy | Tasks 2 and 3 |
| Explicit and implicit layout/slot normalization | Task 3 |
| Grid geometry and all visual variants | Task 4 |
| Semantic DOM and localized accessibility labels | Task 4 |
| Runtime digest and canonical PDF envelope | Task 5 |
| Stable diagnostics | Tasks 1-3 and 5 |
| Fixtures, docs, and comparator | Task 6 |
| Full tests and visual evidence | Task 7 |
| No external reference assets | Global constraints and Task 6 |

The plan contains no unresolved implementation placeholder. Every production
behavior has a named failing test, a RED command, a minimal implementation
step, a GREEN command, and a checkpoint before the next concern.
