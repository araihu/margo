# Margo Slide Decks v0.0.1 Implementation Plan

> For agentic workers: REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Each task is executed with TDD and a fresh verification checkpoint.

Goal: Implement the accepted Margo Marpit-compatible v0.0.1 deck profile in the isolated Margo worktree, including deterministic parsing, layout projection, accessible HTML, Mermaid/chart parity, runtime evidence, and PDF/CLI gates.

Architecture: Keep deck authoring in deck/, extend root margo runtime types only with profile-neutral v2 request/identity fields, and keep PDF media-box evidence in the PDF projection. Parse into an immutable normalized deck model, render each slide through the existing Margo compiler, then wrap fragments in a fixed logical canvas/stage with theme-owned CSS and deterministic task descriptors.

Tech Stack: Go 1.26.5, Goldmark 1.8.2, YAML v3, existing Margo compiler/render-plan/runtime APIs, embedded deck CSS/JavaScript, Chromium via the existing pdf/chromium engine, and table-driven Go tests.

Spec: docs/superpowers/specs/2026-08-19-margo-slide-decks-v001-design.md

## Global Constraints

- Scope is the versioned Margo Marpit-compatible profile v0.0.1, not universal Marpit or Marp Core compatibility.
- Scalar headingDivider N starts slides at H1 through HN; arrays match exact heading levels.
- Top-level CommonMark thematic breaks accept 0–3 leading spaces, with Setext H2 precedence; protected fences, lists, and blockquotes are not slide separators.
- Only built-in themes modern, goshtoso, and minimal, and only the closed layout catalog, are accepted.
- Fixed logical canvases are 1280×720 for 16:9, 960×720 for 4:3, or bounded custom geometry; screen stage scaling never changes logical measurements.
- Canonical validation uses chromium-deck-v1, pinned 1440×900 CSS pixels, device scale 1, zoom 1, and exact bundled Margo font faces.
- Font-bundle v1 uses the frozen prefix/NUL/face-order/uint64-length/raw-WOFF2 preimage and known-answer digest from the spec.
- Raw HTML, remote backgrounds, unsafe extension IDs, unsupported classes, invalid contrast, and invalid directives fail closed with stable diagnostics.
- Existing non-deck v1 runtime descriptors/reports remain wire-compatible; deck profile descriptors/reports use root-owned margo-runtime/v2 fields.
- PDF validation compares exact page count and all four MediaBox edges within 10 micrometres, with non-recursive evidence hashing.
- Do not merge, tag, release, publish, or commit without separate authorization; preserve unrelated worktree state.

---

### Task 1: Freeze root runtime v2 profile contracts

Files: create runtime_validation.go; modify runtime_descriptor.go, runtime_report.go, runtime_projection.go; test runtime_validation_test.go, runtime_descriptor_test.go, runtime_report_test.go.

Interfaces: produce RuntimeProtocolV2, RuntimeValidationRequest, RuntimeValidationIdentity, strict v2 descriptor/report validation, defensive clone support, and profile-aware runtime projection using existing runtime task IDs and diagnostics.

- [ ] Write failing tests for v2 request validation, unknown-field rejection, defensive clone, ready-report identity equality, failed-report identity omission, v1 compatibility, digest-bound task IDs, and missing/extra task reports.
- [ ] Run go test ./... -run Runtime -count=1 and confirm RED because the v2 contract does not exist.
- [ ] Implement root-owned request/identity structs, protocol branching, strict validation, clone logic, and profile checks without changing v1 behavior.
- [ ] Run focused tests, then GOWORK=off go test ./... -count=1.

### Task 2: Implement deterministic font-bundle identity and profile request

Files: create deck/font_bundle.go and deck/font_bundle_test.go; modify deck/render.go, deck/model.go, deck/doc.go.

Interfaces: produce FontBundleDigestV1(theme, bundle) and immutable deck validation-request resolution from the spec theme manifests.

- [ ] Write and run failing known-answer/order/framing tests for the exact 98-byte fixture.
- [ ] Implement the margo-font-bundle/v1 prefix, NUL framing, theme-row family order, ascending weights, big-endian uint64 lengths, and raw-byte hashing.
- [ ] Resolve ExpectedFontBundleDigest from the immutable lock and reject differing caller overrides.
- [ ] Run focused and package tests.

### Task 3: Build directives, slide splitting, and normalized layouts

Files: create deck/directives.go, deck/layout.go, deck/layout_test.go; modify deck/model.go, deck/parse.go, deck/parse_test.go.

Interfaces: produce typed global/local/spot directives, inherited resets, presenter notes, headingDivider semantics, CommonMark thematic-break scanning, and structural slots with source-order reading order.

- [ ] Write failing tests for scalar/array heading dividers, 0–3-space rulers, Setext precedence, inherited reset, background A/B reset, malformed directives, notes, and layout cardinalities.
- [ ] Run focused parser/layout tests and confirm RED.
- [ ] Implement protected-region scanning, heading-level splitting, typed directive YAML, explicit none clears, source positions, slot validation, stable IDs, and defensive accessors.
- [ ] Run focused tests and existing deck tests.

### Task 4: Render themes, geometry, layouts, and accessible HTML

Files: create deck/theme.go, deck/theme_test.go, deck/geometry.go, deck/geometry_test.go; modify deck/render.go, deck/page.go, deck/assets/deck.css, deck/assets/deck.js, deck/render_test.go.

Interfaces: produce deterministic theme/mode tokens, fixed canvas/stage wrappers, exact grid classes, localized labels, focus-safe controls, background semantics, and print reset behavior.

- [ ] Write failing tests for html lang, stage reservation, 16:9/4:3/custom geometry, token rows, structural DOM/read order, contrast rejection, background alternatives, and print restoration.
- [ ] Run focused render tests and confirm RED.
- [ ] Implement DeckGeometry, theme catalog/token tables, contrast calculation, grid classes, CSS variables, stage scaling, print reset, navigation labels, focus restoration, and noncanonical host-viewport diagnostics.
- [ ] Run focused render and asset tests.

### Task 5: Preserve feature parity and render-wide identity safety

Files: create deck/ids.go, deck/ids_test.go, deck/feature_parity_test.go; modify deck/render.go, render.go, runtime_projection.go, runtime_projection_test.go.

Interfaces: produce typed render-wide ID allocation, same-slide/slot reference validation, and integration coverage for Markdown, tables, footnotes, images, Mermaid, charts, code, and sanitized HTML/iframes using existing extensions.

- [ ] Write failing tests for repeated tables/Mermaid/charts, same-slide fragments, cross-slide/cross-slot rejection, allocator idempotence/injectivity, and unsafe extension IDs.
- [ ] Run focused tests and confirm RED.
- [ ] Implement allocator namespacing and reference validation at the deck boundary without changing ordinary document output.
- [ ] Add the integration fixture and run package/runtime projection tests.

### Task 6: Add screen/print runtime task envelopes and validators

Files: create deck/runtime.go, deck/runtime_test.go, deck/runtime_fixture_test.go; modify deck/render.go, runtime_descriptor.go, runtime_report.go, pdf/chromium/engine.go.

Interfaces: produce Result.RuntimeDescriptor, Result.ScreenRuntimeDescriptor, two mode-bound task kinds, canonical screen/print envelopes, overflow normalization, observed browser/profile identity, and separate embedded advisory behavior.

- [ ] Write failing tests for two-task descriptors, valid four-component IDs, canonical output hashes, logical coordinates, overflow diagnostics, mismatch ownership, and advisory isolation.
- [ ] Run focused runtime tests and confirm RED.
- [ ] Implement descriptor construction/dependency ordering, one-call terminal validation, canonical envelopes, stage-origin normalization, and root report projection.
- [ ] Run runtime and Chromium package tests while retaining v1 tests.

### Task 7: Implement PDF media-box evidence and deck CLI geometry

Files: create deck/pdf_evidence.go and deck/pdf_evidence_test.go; modify pdf/engine.go, pdf/chromium/engine.go, cmd/margo/deck.go, cmd/margo/deck_test.go, cmd/margo/pdf.go.

Interfaces: produce PDFMediaBoxMicrometers, PDFArtifactReport, exact four-edge evidence hashing, deck geometry precedence, and atomic HTML/PDF publication gates.

- [ ] Write failing tests for point conversion, edge tolerance, page count, 247-byte evidence, geometry conflicts, and atomic failure output.
- [ ] Run focused PDF/CLI tests and confirm RED.
- [ ] Implement evidence serialization/hash, media-box comparison, deck CLI flags, screen/print/PDF routing, and native-engine rejection for deck mode.
- [ ] Run focused PDF/CLI tests and the full repository suite.

### Task 8: Add fixtures, documentation, and final gates

Files: create deck/testdata fixtures; modify deck/doc.go, README.md, and deck-mode authority references in docs/GOSHTOSO_MARKDOWN_DESIGN.md.

Interfaces: produce representative 16:9/4:3 screen/print fixtures, compatibility corpus, integration deck, and user-facing API/CLI documentation.

- [ ] Write fixture-consumption tests and run them RED.
- [ ] Add fixtures and documentation with exact accepted syntax and no unsupported author CSS claims.
- [ ] Run gofmt, git diff --check, GOWORK=off go test ./... -count=1, and the browser/PDF gates from spec section 14.6.
- [ ] Stop with a clean verification report; do not commit or publish without authorization.

---

## Execution Notes

Execute tasks in order. Each task must complete its RED→GREEN cycle before the next task begins. If a focused test exposes a contract gap, revise this plan and re-run plan review before writing production code. Generated files remain generated through repository tooling; do not hand-edit generated templ output.
