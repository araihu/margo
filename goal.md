# Goal: Margo / Marpit slide decks

> Historical development record from 2026-08-19. The original session intended
> this file to remain local and untracked, but it is present in repository
> history. Status, evidence and authorization statements below describe that
> session, not current repository state or permission for new work.

## Iterative record

The original session treated this as a living goal, updated after each relevant
decision, implementation, evidence item or review round. It was intended to:

- retain the current objective and next verifiable result;
- record new decisions and evidence without losing useful context;
- move items between open, in progress, blocked and complete;
- update the review loop and exit criteria when scope changed;
- remain local and untracked throughout the work.

## Context

- Worktree: `/private/tmp/margo-marpit-decks-v001`
- Branch: `feat/marpit-decks-v001`
- Cycle date: 2026-08-19
- Published commit: `335d9892deb937b1516304aa5dd65acb9bd53a90`
- Recorded state: v0.0.1 slice implemented, reviewed, committed and pushed to the branch; a new cycle began after handoff from session `01a01874-d5e8-74a3-9338-191cc438a0f0`.
- Delivery boundary at that time: preserve the isolated worktree. The previous slice's commit and push were authorized and complete. Any new commit or push required its own authorization. PR creation, merge, tags, releases and deployment required explicit authorization.

## Active cycle after handoff

### Current objective at that checkpoint

Define and implement the next Margo decks increment on the published baseline,
without weakening Marpit compatibility, determinism, accessibility or approved
runtime/PDF contracts.

### Frozen baseline

- [x] Marpit-compatible v0.0.1 profile implemented;
- [x] layouts, themes, charts, Mermaid, media, notes and standalone HTML;
- [x] 16:9/4:3 geometry, overflow, print/PDF and runtime reports;
- [x] r6 comparator with 30 Margo/Marpit pairs and reproducible review;
- [x] Impeccable R5 and Checklist Design R5 without P0/P1 findings;
- [x] Go tests, vet, modules, formatting, diff and JavaScript syntax checks passed;
- [x] commit and remote branch verified at the same SHA;
- [x] no TOTVS PDF, PPTX, asset or string tracked.

### Scope decision

The selected next increment was **Compositions R1**. Explicit approval in that
session authorized implementation and verification of the slice, not a commit,
push, PR, merge, tag, release or deployment.

- [x] **Compositions R1:** turn the reference's generic vocabulary into Margo
  presets (`content`, `agenda`, `media-split`, `media-stage`, `steps`,
  `highlight`, `compare-grid`, `hero`, `image-grid`) without copying source
  layouts or assets.
- [ ] **P2 evidence debt:** run actual browser zoom at 200%, test VoiceOver/NVDA,
  correct the PT-BR placeholder language, and record reproducible corpus bytes
  and sidecars.
- [ ] **Comparator provenance:** replace synthesized identity/status in
  `make-compare.mjs` with evidence from the real runtime, validated through the
  canonical envelope.
- [ ] **Data storytelling R2:** specify `chart-story`, `data-table`, `cycle`
  and `timeline-fit`; implement only after closing the R1 composition contract.

### Approved architectural decision

Compositions R1 would introduce a versioned `composition` directive, separate
from `class` and `layout/slot`. The directive expresses intent; the internal
catalog resolves structural layout, slot contract, visual variant, content
rules and accessibility requirements. The DOM receives
`data-margo-composition` and remains deterministic.

v0.0.1 input without `composition` remains compatible. R1 adds only the
versioned directive, accepts no arbitrary HTML/CSS, copies no TOTVS layouts
and imports no reference assets. The specification fixes names, aliases,
cardinalities, fallback behavior, diagnostics, profile version and print/PDF
behavior.

### Next verifiable result

- [x] choose one primary workstream;
- [x] record the contract, affected files, compatibility and non-goals in
  `docs/superpowers/specs/2026-08-19-margo-compositions-r1-design.md`;
- [x] review and approve the architectural specification;
- [x] write the implementation plan in
  `docs/superpowers/plans/2026-08-19-margo-compositions-r1.md`;
- [x] write failing tests for the selected behavior;
- [x] implement the smallest increment that passes those tests;
- [x] repeat Go, runtime/PDF and visual review gates proportional to risk;
- [x] update this record with evidence and remaining work;
- [x] stop before commit, push, PR, merge, tag, release or deployment without
  new explicit authorization.

### Active cycle exit criteria

The selected workstream needed a documented contract, reproducible tests,
matching visual or runtime evidence, no P0/P1 regressions and a clear inventory
of remaining out-of-scope work. Passing gates did not authorize publication.

### Auxiliary synchronization boundary

An external workstream inventoried a TOTVS template: PDF/PPTX, 118 slides,
67 layouts, 5 masters and 174 media files. This input was only a vocabulary
and composition/layout reference. Its files, PDFs, PPTX, media and assets
were not to be imported, copied or tracked in the worktree.

Final synchronization confirmed the generalization rule: do not replicate
67 layouts. The slice grouped composition families into eight R1 presets
(`content`, `agenda`, `media-split`/`media-stage`, `steps`, `highlight`,
`compare-grid`, `hero`, `image-grid`). Data storytelling (`chart-story`,
`data-table`, `cycle`, `timeline-fit`) remained R2. `theme` meant global
tokens/fonts/chrome; `layout` meant geometry/slots; `composition` meant
intent + layout + rules; `variant` meant visual treatment; `asset` meant
media or decoration; `motion` remained a separate runtime. The v0.0.1 public
contract remained `class` + `<!-- layout/slot -->`, with no new public
directive in that baseline cycle.

Agendas, media cropping/focal points, grids of 3-4 images, steps, highlights,
3-4-column comparisons, heroes, multiple charts, tables/merges, radial gauges
and other identified gaps were roadmap items, not delivered support. Object
timelines, transitions, absolute positioning, composite masks/vectors, device
mockups, brand ribbons and PPTX round trips remained outside Markdown/v0.0.1.
No logo, photo, icon, font or asset was extracted or imported from the reference.

A self-contained visual guide also existed at
`/private/tmp/margo-totvs-deck-audit-20260819/margo-decks-visual-reference.html`.
It could inform documentation and vocabulary: 25 presets, a matrix, tokens,
boundaries and roadmap. It remained local/temporary, was not to be tracked or
promoted by inference, and supplied no TOTVS strings, images, SVGs or elements
to the product.

## Objectives

1. Implement Margo slide decks with an explicitly versioned Marpit/Marp-compatible profile.
2. Support the approved layouts and themes, including columns, sidebar, compare, metrics, timeline and demo.
3. Support the slice's content extensions: charts, Mermaid, code, images/backgrounds, speaker notes and local/global directives.
4. Produce deterministic standalone HTML and PDF with fixed geometry, overflow validation, print/PDF evidence and runtime identity.
5. Preserve semantic accessibility: stable reading order, localized labels, visible focus, keyboard navigation, text alternatives and AA contrast.
6. Compare Margo visually with equivalent Marpit output in 16:9 and 4:3, on desktop and narrow viewports.
7. Preserve existing runtime contracts: root descriptors/reports, task IDs, diagnostics and canonical hashes.

## Specification review result

An independent review loop used development and design subagents. Blockers
were revised until the specification milestone was considered acceptable.

Contract decisions closed:

- correct scalar and array `headingDivider` semantics;
- CommonMark separators with 0-3 spaces and Setext H2 precedence;
- inherited directive reset/clear and atomic background + alternative association;
- logical/visual geometry, scaling, print reset and normalized measurement space;
- structural catalog, DOM, reading order and per-layout combinations;
- exhaustive contrast checks by theme/mode;
- canonical font bundle, weights, licenses and digest with reproducible preimage;
- profile-neutral root descriptor/report, strict v2 schemas and valid screen/print tasks;
- non-recursive PDF evidence envelope and known hashes;
- separate `deck.font_bundle_mismatch` and `deck.validator_profile_mismatch` diagnostics.

## Visual implementation review result

### Verdict

**Partial, 15/32 (Poor).** Two heuristics were N/A, #5 and #9. There was no
P0 finding, but three P1 findings remained.

The content was recognizably Margo through its editorial teal accent,
runtime/PDF, charts and evidence. The comparator still behaved like a generic
gallery. At 390 px, each pair stayed side by side in roughly 158x90 thumbnails,
making slide text unreadable.

### Objective evidence

- 30 comparison cards and 60/60 images loaded;
- 960x540 images;
- tested at 1280x720 and 390x844;
- no console errors, duplicate IDs or horizontal overflow;
- `lang=pt-BR`, one H1, 30 H2s, `header/main/article` and nonempty `alt`;
- observed contrast: body 13.91:1, paragraph 5.53:1, labels 4.80:1;
- Impeccable detector: exit 2, 2 warnings, 0 errors, both concerning `Inter`
  in the comparison tool and of low relevance to the product;
- no overlay was injected because the browser interface was read-only;
  inspection used the DOM, metrics, screenshots and console.

### Priority blockers

The three P1 items below record the historical R1 round. They were fixed in
that cycle and reassessed in R2.

#### P1: mobile inspection

Add stacked mode, 1:1 zoom, fullscreen, slide navigation and synchronized
pan/zoom. Also validate 200% zoom.

#### P1: typography and theme fidelity

Embed versioned WOFF2, validate computed fonts and address serif fallback on
"modern" slides, which should use Margo Sans/UI sans.

#### P1: provenance and meaning of differences

Show source SHA, renderer/version, viewport and PDF geometry, timestamp,
difference captions, per-slide status, notes and delta mode.

#### P2: review organization

Group the corpus into Foundations, Layouts, Media, Charts and Runtime/PDF.
Provide an index, anchors, progress and an unreviewed-item filter. Reduce
shadows and chrome.

#### P2: capture semantics

Use `figure/figcaption`, difference summaries and links to source HTML/PDF
instead of relying only on generic `alt` text.

### R2 review after the first fix

- Impeccable Assessment A: **26/40, Acceptable**, compared with 15/32 in R1.
  The comparator now looked like a Margo/Marpit tool, with provenance, groups,
  filters, state and zoom.
- Impeccable Assessment B: detector ran once. Its only warning was a false
  positive for the closed dialog's image. The in-app browser became
  unavailable during the round, so reproducible local CDP evidence
  supplemented the live R2 evidence.
- Checklist Design: R1 responsiveness was fixed, but the audit found
  insufficient focus/border contrast, mixed language without associated
  `lang`, local state without corpus identity, and a manifest without
  font/runtime identity.

Fixes applied after R2:

- [x] local state now uses manifest + corpus identity; `needs attention` is
  counted separately and reset has confirmation + undo;
- [x] 1:1 zoom opens Margo and Marpit together, stacked on narrow viewports;
- [x] borders and focus rings use tokens with contrast >= 3:1 and controls
  have a minimum 44px target;
- [x] the comparator document uses `lang="en"` and marks its introduction and
  caption with `lang="pt-BR"`;
- [x] the manifest publishes expected/observed font bundles, checks and
  `margo/runtime-report/v2` validation identity;
- [x] Margo captures were regenerated with the real WOFF2 bundle in
  `/tmp/margo-pdf-compare/margo-r3-final4` at 960x540, removing the serif
  fallback seen in the previous round;
- [x] the deck keyboard handler does not intercept arrow keys inside
  controls, links or chart widgets;
- [x] extreme reflow simulated through CDP at 390px and an equivalent of
  200% has no horizontal overflow beyond the vertical scrollbar.

## Review loop performed

### 1. Specification and contract

- Initial layout and Marpit compatibility brainstorming.
- Isolated worktree created for the slice.
- Parallel specification review by development and design subagents.
- Iteration on Marpit semantics, geometry, accessibility, fonts, runtime
  reports, IDs and PDF evidence.
- Specification milestone accepted before implementation.

### 2. Slice implementation

- Deck parser/renderer, themes, layouts and directives.
- Charts, Mermaid and the other planned extensions.
- Navigation runtime, localized controls, focus and overflow.
- Standalone HTML, PDF and validation evidence.
- Compatibility corpus and Margo/Marpit comparison.

### 3. Visual comparison

- Equivalent Marpit output generated.
- Comparison page at
  `http://127.0.0.1:8765/margo-pdf-compare/index.html?cb=20260819v3`.
- Desktop and narrow-viewport inspection.

### 4. Review with `$impeccable`

- Context preflight with `context.mjs`.
- Assessment A in a subagent: specificity, Nielsen heuristics, cognitive
  load, emotional journey, personas and priorities.
- Assessment B in a subagent: detector, DOM, metrics, screenshots, console
  and responsive behavior.
- Critique saved at
  `/private/tmp/margo-marpit-decks-v001/.impeccable/critique/2026-08-19T16-52-25Z__index-html.md`.
- Trend: first record for that target, with no comparable trend yet.

### 5. Review with `$using-checklist-design`

Independent audit based on Checklist Design records:

- [Carousel](https://www.checklist.design/design-system/carousel): partial;
- [Button](https://www.checklist.design/design-system/button): partial;
- [Card](https://www.checklist.design/design-system/card): consistent on
  desktop, failed mobile readability;
- [Typography](https://www.checklist.design/design-system/typography): scale
  present, insufficient zoom/narrow-viewport support;
- [Accessibility](https://www.checklist.design/design-system/accessibility):
  good semantics and contrast, incomplete assistive-technology/focus matrix.

## Progress during that cycle

Implemented after the 15/32 review:

- [x] reproducible generator in `tools/marpit-compare/make-compare.mjs`;
- [x] Foundations, Layouts, Media, Charts and Runtime/PDF groups;
- [x] index, search, category filter and local review state;
- [x] Margo-only/Marpit-only views and 1:1 capture dialog;
- [x] provenance with corpus/manifest SHA-256, renderer, viewport, PDF geometry
  and timestamp;
- [x] stacked narrow-viewport layout; CDP evidence: viewport 390,
  `scrollWidth=390`, 366px card, 336px capture;
- [x] explicit sans/mono fallback to prevent accidental serif fallback when
  interactive font faces are absent;
- [x] replace font-bundle placeholders with real versioned WOFF2, embed
  aliases/license, validate `document.fonts` and observed digest;
- [x] connect font evidence to Chromium v2 `RuntimeReport` through `FontChecks`
  and `FontBundleDigest`, failing with a stable diagnostic on mismatch;
- [x] bind review state to corpus/manifest identity, separate `reviewed` from
  `needs attention`, and provide confirmation + undo;
- [x] open zoom as a paired Margo/Marpit comparison with valid placeholders,
  captions and contextual alt text;
- [x] raise border/focus contrast and use 44px targets;
- [x] regenerate 30 Margo captures with real WOFF2 and publish observed
  font/runtime identity in the manifest;
- [x] protect the keyboard handler around controls, links and interactive widgets;
- [x] crop each capture to logical `.margo-deck__slide`, removing stage,
  shadow and border, and freeze 30 960x540 PNGs; `capture-report.json`
  records zero overflow at 16:9;
- [x] fit charts to the fixed canvas and validate 30/30 slides without
  overflow at 16:9 and 4:3 in `/tmp/margo-exhaustive-4by3-final3`;
- [x] add per-slide manifests, `capture-report.json` and
  `runtime-report-v2.json` with browser/engine/platform/font identity and
  reproducible hashes;
- [x] add default `fit pair` mode and a 1:1 toggle in the paired dialog;
- [x] add review JSON export with identity, counts, states, timestamp and
  evidence sidecars;
- [x] align taxonomy with the real corpus: Media 13-19, Diagrams/Charts 20-26
  and Runtime/PDF 27-30;
- [x] repeat Assessment A, Assessment B and Checklist Design on the new page.

### 6. Required next cycle

1. Complete: rerun Assessment A, Assessment B and Checklist Design on r6.
2. Complete: revalidate 16:9, 4:3, 1280x720, 390x844, the equivalent of 200%,
   print/PDF, modal, export and structural accessibility. Keep CDP evidence
   when the in-app browser is unavailable.
3. Complete: record visual/functional approval and close the goal without
   merge, push, release or deployment.

## Exit criteria

The goal required a comparator with readable mobile and desktop inspection,
explicit differences/provenance, deterministic HTML/PDF/runtime contracts,
and approval from both Impeccable subagents plus Checklist Design without
open P1 findings. It remained open until that checkpoint, even when code and
CDP gates passed.

## R3/R4 checkpoint: post-review fixes

- The visual P0 was closed: all 30 Margo PNGs in `margo-r3-final4` are
  960x540, captured only from the logical slide. `capture-report.json`
  reports `overflow=false` for every 16:9 slide.
- The same overflow matrix passed at 4:3 for 30/30 slides in
  `/tmp/margo-exhaustive-4by3-final3`.
- The current comparator's CDP audit, `cb=20260819r5`, found 30 cards,
  60 figures/60 images, 60/60 images loaded after individual scrolling,
  no broken images, no console errors and no horizontal overflow at 1280 or
  390. At the equivalent of 200%, `scrollWidth=204` versus `clientWidth=195`
  represented only the expected vertical scrollbar.
- The paired dialog opens in `fit-pair` with both images and toggles to 1:1.
  Focus returns to the originating button. Reset + undo restores state.
  CDP exercised JSON export and downloaded
  `margo-marpit-review-337ca30e59b9.json` with `total=30`, `needsAttention=1`,
  sidecars and `runtimeEvidenceSHA256`.
- Checklist R4 found a P1 selector defect: `data-zoom-title` also appeared on
  buttons. The generator now uses `#zoom-title`, preserves all 60 images
  after opening the modal and updates the dialog's associated label.
- The equivalent of 200% had 9px of extra document width from flexible fields
  and long hashes. Inputs, selects, toolbar and notes now have width limits
  and wrapping. r6 measures `195/195` without horizontal document overflow.
- Impeccable R4 approved the comparator without P0/P1 after capture, fit-pair
  and export fixes. The previous Checklist R4 was invalidated by the stale
  selector. Checklist R5 and Impeccable R5 approved r6 without P0/P1.
- `GOWORK=off go test ./... -count=1` passed after chart and comparison changes.
  The browser served `index.html` with HTTP 200 and bytes identical to the
  local file.

## R5 checkpoint: slice completion

- Impeccable R5: **approved, no P0/P1**. Served r6 matched local `index.html`,
  SHA-256 `8974ffd145e5071e387bc37856b156b5e14355c69c5c0d63c8c973837b0a60ea`.
- Checklist Design R5: **no remaining P1**. Carousel, Button and Card passed.
  Typography, Accessibility and Provenance remained partial only because of
  P2 evidence gaps.
- The selector defect was fixed with `#zoom-title`: opening the modal keeps
  60/60 images and updates `aria-labelledby` for the selected slide.
- Fresh CDP confirmed fit-pair at roughly 586/586 px, 1:1 at 960/960 px,
  restored focus, no errors/exceptions and a 195 px equivalent with no
  document overflow: `inner/client/scroll/body = 195`.
- Actual export validated: parseable JSON, consistent counts/identity/sidecars,
  SHA-256 `7d4e7e53daa1c8c009e18e5f6e3b892010a9cf51777415bfcb395139db54d576`.
- Margo evidence: 30 960x540 PNGs, 30/30 slides without overflow at 16:9 and
  4:3, 60 assets with intact sizes and hashes, manifests and runtime report
  recomputable within the comparator envelope.
- Explicit P2 limits: no literal 200% browser zoom or VoiceOver/NVDA run;
  corpus bytes do not accompany sidecars; `make-compare.mjs` synthesizes
  comparative report identity/status from arguments; a Portuguese placeholder
  still inherits `lang="en"`. These gaps did not block the slice but
  remained evidence backlog.

## Compositions R1 checkpoint: approved implementation

The user approved execution after specification and plan review. Implementation
was completed in the isolated worktree without promoting TOTVS files or
creating another commit or push.

### Delivered contract

- closed `r1` catalog with `content`, `agenda`, `media-split`, `media-stage`,
  `steps`, `highlight`, `compare-grid`, `hero` and `image-grid`; `none` clears
  inherited composition;
- frontmatter, body and spot directive with inheritance/clear and stable
  `deck.composition_*` diagnostics;
- implicit layouts or compatible explicit markers, semantic slots in source
  order, and the `grid` family controlled only through composition;
- HTML with catalog, composition, variant, family and slot identity,
  localized labels and CSS limited to R1 tokens/families;
- canonical task digest and validation envelope include catalog and
  composition inputs; PDF evidence preserves the legacy path and records R1
  identity when used;
- self-contained Markdown fixture covering all nine names, comparable
  manifest, public documentation and a `Compositions R1` generator category.

### Gates executed

All passed:

```text
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
GOWORK=off go mod verify              # all modules verified
gofmt -l .                             # empty output
git diff --check
node --check tools/marpit-compare/make-compare.mjs
GOWORK=off go test ./deck ./pdf ./pdf/chromium ./cmd/margo -count=1
```

### Visual and comparison evidence

Temporary artifacts remained outside the repository at
`/tmp/margo-compositions-r1-eNjPpk/`:

- six HTML files: modern, goshtoso and minimal at 16:9 and 4:3, plus six
  Chromium PDFs. Each PDF has 10 pages; 16:9 measures `960 × 540 pt` and
  4:3 measures `720 × 540 pt`;
- reviewed PNGs of content, compare-grid and 4:3 minimal in `png/`;
- comparator generated in `compare/`, with `catalogVersion: r1`, manifest
  SHA-256, nine R1 category cards and no deliberately uncomposed slide;
- Playwright/CDP checked desktop `1280/1280` and mobile `390/390` document
  widths, 9 R1 cards, 18 composition labels, category filter, Margo-only,
  paired zoom, 1:1 toggle, focus restoration and dialog closure.

### Status and remaining work

The R1 cycle was complete for implementation and proportional review, with
no open P0/P1. P2 items remained: literal 200% zoom, VoiceOver/NVDA, complete
corpus sidecars, real runtime identity capture in the comparator and a dedicated
PT-BR placeholder. The next data-storytelling increment remained R2 and was
not presented as current support.

The published baseline remained
`335d9892deb937b1516304aa5dd65acb9bd53a90`; any new commit, push, PR, merge,
tag, release or deployment required its own authorization.

## Visual checkpoint: Margo chrome and cover

- [x] visible pagination reduced to an integer, `1`, `2`, etc., always
  anchored to the logical canvas's bottom-right corner;
- [x] header/footer removed from content flow, positioned near corners and
  left-aligned, without screen or print overlap;
- [x] cover/lead/hero uses a three-row vertical container, with block and
  text centered on the canvas;
- [x] PDF regenerated and checked: 42 pages, `960 × 540 pt`, Tagged;
  pages 1, 2, 10, 13 and 42 reviewed visually;
- [x] `GOWORK=off go test ./... -count=1` and `git diff --check` passed;
- [ ] commit/push remained out of scope without new authorization.

## Marpit-equivalent checkpoint: exhaustive corpus

- [x] equivalent output generated from the same 42-slide narrative corpus
  using `@marp-team/marp-core` 4.4.0 on `@marp-team/marpit` 3.2.2;
- [x] frontmatter converted to `marp: true`, standard theme, pagination,
  header/footer and `size: 16:9`; Margo classes projected to Marpit `_class`;
- [x] content, tables, lists, code, notes, images and roadmap preserved;
  `layout/slot/composition` without a native equivalent were flattened in
  source order, without arbitrary CSS or absolute positioning;
- [x] Mermaid and Goshtoso remained code fences in the pure baseline, with
  limits stated in the deck; no external plugin was presented as native support;
- [x] temporary HTML, Markdown source and PDF in
  `/tmp/margo-marpit-equivalent/`; only existing Margo assets were staged
  locally, without copying or tracking TOTVS material;
- [x] Chromium PDF checked: 42 pages, `960 × 540 pt`, Tagged, 1,973 words
  inside the MediaBox, 42 HTML sections and four images with `alt`;
- [x] visual samples reviewed: centered cover, agenda, media stage, metrics,
  timeline, Mermaid, chart fence, matrices, inverted theme and final slide;
- [ ] commit/push publication remained outside this checkpoint and required
  new explicit authorization.

## Base slice status

Implementation, visual comparison, runtime/PDF, structural accessibility and
independent review objectives were complete in the isolated worktree. The slice
was published at commit `335d9892deb937b1516304aa5dd65acb9bd53a90` on
`feat/marpit-decks-v001`. The session recorded `goal.md` as local and untracked.
Its active cycle was described at the start of the original document; it
conferred no implicit authorization for another commit, push, PR, merge, tag,
release, deployment or import of TOTVS material.
