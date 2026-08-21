# Margo deck compositions R1

Status: design approved in chat; implementation starts only after this spec is
reviewed and accepted.

Date: 2026-08-19

## 1. Decision

Margo decks will gain a versioned `composition` directive. A composition names
authorial intent and resolves to a finite Margo catalog entry. The catalog entry
owns the structural layout, slot contract, visual variant, accessibility label,
and validation rules. Existing `class` and `layout/slot` syntax remains the
structural mechanism for the v0.0.1 profile and remains valid without a
composition.

The first catalog is `r1`. It is a Margo extension, not a claim of universal
Marpit or Marp Core compatibility. The catalog is closed: authors cannot add a
composition, slot, class, CSS rule, or layout through Markdown.

The public authoring surface is:

```yaml
---
marp: true
composition: media-split
---
```

or, for a slide-local override:

```markdown
<!-- composition: media-split -->
<!-- slot: media -->
![A descriptive alternative](media.png)
<!-- slot: content -->
## The result

Supporting evidence.
```

The frontmatter value establishes the inherited default. A body comment
overrides that default for following slides. A spot comment such as
`<!-- _composition: none -->` applies only to the slide currently being
scanned. `composition: none` clears the inherited composition.

For a multi-slot composition, slot markers are explicit. The composition may
imply the structural layout marker, but it never guesses content boundaries.
An explicit `<!-- layout: ... -->` is accepted only when it agrees with the
resolved composition. Existing v0.0.1 structural input keeps its exact rules.

## 2. Problem and goals

The reference deck contains recurring authorial patterns: a standard content
slide, a media split, a staged media slide, an agenda, a sequence of steps, a
highlight, a comparison grid, a hero, and an image grid. The existing Margo
catalog has useful layouts, but it makes intent implicit and cannot express the
new grid shapes without a stable contract.

This slice will:

1. expose the nine approved R1 composition names;
2. normalize each composition to a deterministic catalog entry;
3. preserve source, reading, keyboard, and print order;
4. validate required slots and explicit layout/class conflicts before render;
5. emit stable composition and slot metadata in HTML;
6. include composition identity in deterministic runtime evidence;
7. preserve all v0.0.1 decks that do not use `composition`;
8. provide tests and representative 16:9/4:3 evidence for every composition.

This slice does not attempt to reproduce the reference deck's 67 layouts. It
generalizes recurring patterns and keeps the reference files, media, fonts,
logos, and extracted elements outside the repository.

## 3. Non-goals and boundaries

The following remain outside R1:

- arbitrary author CSS, HTML, JavaScript, or custom composition registration;
- absolute positioning, masks, compound vector editing, or device mockups;
- transitions, fragments, presenter mode, or animation timing;
- round-trip PPTX import or export;
- copying a source template's assets, geometry, logos, or brand treatment;
- R2 data-storytelling compositions such as `chart-story`, `data-table`,
  `cycle`, and `timeline-fit`;
- literal browser 200% assistive-technology evidence and synthetic comparator
  provenance, which remain separate P2 work;
- changing ordinary Markdown rendering when deck mode is not active.

The visual guide at
`/private/tmp/margo-totvs-deck-audit-20260819/margo-decks-visual-reference.html`
is vocabulary-only evidence. It is not a product input and must not be copied
or versioned.

## 4. Versioning and compatibility

### 4.1 Catalog identity

The deck package owns:

```go
const CompositionCatalogVersion = "r1"
```

The value is part of the normalized composition manifest, the root HTML data
attribute, the deck layout task input digest, and the canonical layout
validation envelope. Changing slot names, cardinality, layout geometry,
variant semantics, or accessibility labels requires a new catalog version.

The source value remains the short composition name. Authors do not write a
catalog version in every slide. A future catalog can be selected by an
explicit profile gate; R1 is the only accepted catalog in this slice.

### 4.2 Old input

Input without `composition` follows the existing v0.0.1 path:

- no composition metadata is emitted on slides;
- existing style-only classes keep their current meaning;
- existing structural classes require their current layout marker and slot
  names;
- existing errors and runtime task kinds remain unchanged;
- existing fingerprints remain stable when the normalized source and render
  settings are unchanged.

Input with `composition` opts into the R1 catalog and receives the new
composition metadata and R1 task input identity.

### 4.3 Conflict policy

The resolver combines source declarations in this order:

1. the catalog default is empty;
2. frontmatter `composition` establishes the inherited default;
3. non-spot body comments replace the inherited default for following slides;
4. a spot comment replaces or clears the value for one slide;
5. an explicit class or layout marker is checked against the resolved entry.

The following are errors, never silent fallback:

- unknown or empty composition name;
- composition and explicit class resolving to different structural/style
  families;
- composition and explicit layout marker resolving to different layouts;
- a slot name not accepted by the composition;
- missing, duplicate, empty, or out-of-range composition slots;
- a composition block crossing a slide boundary;
- composition metadata placed inside a fenced code block being interpreted as
  a directive.

## 5. R1 catalog

Every entry has a stable name, an intent, a resolved layout family, a variant,
and a slot contract. Slot order is source order and is never changed by CSS.

| Name | Intent | Resolved family | Slots | Cardinality | Variant |
| --- | --- | --- | --- | --- | --- |
| `content` | Standard text, table, code, chart, or diagram slide. | default | body | implicit body | `content` |
| `agenda` | A roadmap or table of contents. | timeline | `item-1` ... `item-6` | 3-6 | `agenda` |
| `media-split` | Media beside explanatory content. | columns | `media`, `content` | exactly 2 | `split` |
| `media-stage` | Media-led split with a stronger media stage. | columns | `media`, `content` | exactly 2 | `stage` |
| `steps` | Ordered process or implementation sequence. | timeline | `step-1` ... `step-6` | 3-6 | `steps` |
| `highlight` | One conclusion, quote, or key takeaway. | section | body | implicit body | `highlight` |
| `compare-grid` | Symmetric comparison of two to four items. | grid | `item-1` ... `item-4` | 2-4 | `compare` |
| `hero` | Opening, closing, or high-emphasis statement. | lead | body | implicit body | `hero` |
| `image-grid` | Ordered collection of two to four images. | grid | `image-1` ... `image-4` | 2-4 | `image` |

`content`, `highlight`, and `hero` use the slide body and do not require slot
markers. Their normalized entry still contains a single logical `body` role
for manifests and accessibility reporting.

`agenda` uses the same ordered-list DOM family as `timeline`, but its variant
and localized accessible label are distinct. `steps` retains the existing
timeline step naming. Both are constrained to three through six items.

`media-split` and `media-stage` use the two-column geometry. Their semantic
slot names are `media` and `content`; the renderer emits those names in
`data-margo-slot` while preserving the resolved family in the layout class.

`compare-grid` and `image-grid` use the new controlled `grid` family. The grid
has two through four equal tracks, a catalog-owned gap, and no source-order
reordering. `compare-grid` slots are content cards. `image-grid` slots are
media cards; existing Margo image rendering remains responsible for local
asset policy and alternative text.

## 6. Source grammar

### 6.1 Frontmatter default

`composition` is accepted as a deck-owned frontmatter scalar:

```yaml
composition: content
```

The accepted values are the nine lowercase R1 names and `none`. Mixed-case
values are invalid rather than silently normalized. Empty, sequence, mapping,
alias, and unknown values fail with `deck.composition_invalid`.

### 6.2 Body and spot directives

The same scalar is accepted in a recognized HTML comment:

```markdown
<!-- composition: image-grid -->
```

The underscore form is slide-local:

```markdown
<!-- _composition: none -->
```

Unrecognized comments remain presenter notes under existing rules. Recognized
malformed comments fail at their source line.

### 6.3 Slot markers

Composition slots use the existing marker syntax:

```markdown
<!-- slot: image-1 -->
![First image](one.png)
<!-- slot: image-2 -->
![Second image](two.png)
```

For a composition, `<!-- layout: ... -->` and `<!-- /layout -->` are optional
when the composition itself supplies a structural family. If present, they
must match the resolved family. A composition cannot contain unmarked text
outside its body or slot contract.

The existing explicit structural form remains unchanged:

```markdown
<!-- class: columns -->
<!-- layout: columns -->
<!-- slot: left -->
...
<!-- slot: right -->
...
<!-- /layout -->
```

It does not become a composition implicitly. Authors must opt into
`composition` when they want R1 semantic metadata or composition variants.

## 7. Normalized model

The `deck/` package remains the single owner of composition normalization. The
model adds a defensive, immutable value alongside the existing directive and
layout state:

```go
type CompositionName string

type CompositionSlot struct {
    Name       string
    Role       string
    Required   bool
    SourceLine int
}

type CompositionSpec struct {
    CatalogVersion string
    Name           CompositionName
    LayoutClass    string
    Variant        string
    MinSlots       int
    MaxSlots       int
    Slots          []CompositionSlot
    BodyRole       string
}
```

These fields and the following invariants are fixed:

- an empty `CompositionSpec` means no composition and is distinct from
  `content` selected explicitly;
- every non-empty spec has catalog version `r1` and a known name;
- slot order is source order and source lines are retained for diagnostics;
- defensive copies are returned by exported accessors;
- normalized specs are immutable after parsing;
- layout normalization cannot mutate the source Markdown or directive state.

`DirectiveState` carries the requested composition name. `Slide` carries the
resolved spec, because an inherited composition may resolve differently on
each slide after a spot override. The public `Slide` accessor returns a
defensive copy, matching `Layout`, `Directives`, and `Notes`.

## 8. Resolution algorithm

Resolution happens after slide splitting and before fragment rendering:

1. parse and validate the source composition value;
2. apply inherited and spot state to the slide;
3. look up the R1 catalog entry;
4. determine whether the entry uses body content or slots;
5. normalize optional layout markers and explicit classes;
6. validate composition slots and content boundaries;
7. produce a `CompositionSpec` and a normalized `Layout` when structural;
8. render each body or slot fragment with the existing TargetDeck path;
9. emit the spec in HTML and include it in runtime identity.

The resolver never infers a composition from headings, image count, table
shape, or CSS. The source must opt in.

### 8.1 Class compatibility

If no class is supplied, the resolver supplies the catalog family's style or
structural class. If a class is supplied, it must equal the catalog family or
be the allowed `invert` modifier. A second structural or style class remains
invalid. A mismatch produces `deck.composition_conflict`.

### 8.2 Layout compatibility

If no layout marker is supplied, a structural composition supplies the family.
If a marker is supplied, its class must equal the family. A mismatch produces
`deck.composition_conflict`. Explicit v0.0.1 layout markers without a
composition continue through the existing validator.

### 8.3 Slot compatibility

The resolver validates the exact set and order for the selected entry. Empty
slots, duplicates, unknown names, missing required slots, and cardinality
outside the table produce composition-specific diagnostics with the original
slot line. The resolver does not reorder slots to satisfy the table.

## 9. HTML and accessibility contract

The root article receives:

```html
<article
  class="margo-deck"
  data-margo-composition-catalog="r1"
  data-margo-width="1280"
  data-margo-height="720">
```

Each composed slide receives:

```html
<section
  class="margo-deck__slide margo-deck__slide--columns"
  data-margo-slide="0"
  data-margo-composition="media-split"
  data-margo-composition-variant="split"
  aria-label="Slide 1 of 1">
  <div
    class="margo-layout margo-layout--columns"
    data-margo-composition-family="columns"
    data-margo-slot-count="2">
    <div
      class="margo-layout__slot margo-layout__slot--media"
      data-margo-slot="media"
      data-margo-slot-role="media">
      ...
    </div>
    <div
      class="margo-layout__slot margo-layout__slot--content"
      data-margo-slot="content"
      data-margo-slot-role="content">
      ...
    </div>
  </div>
</section>
```

The exact attribute order is not a public contract, but attribute names,
values, DOM nesting, source order, and roles are. Uncomposed v0.0.1 slides do
not gain a composition attribute.

Accessibility rules:

- source order is DOM order, reading order, and keyboard order;
- CSS cannot reorder composition slots;
- `agenda` and `steps` retain an ordered-list structure;
- `compare-grid` and `image-grid` use a labeled group when a group label is
  needed, without hiding individual slot content;
- image alternatives remain visible to assistive technology through the
  existing Margo image projection;
- media-stage decoration is not allowed to replace an informative alternative;
- composition labels are localized through the existing deck label catalog;
- focus, contrast, and print behavior inherit the existing theme contracts.

## 10. Geometry and CSS

R1 adds one controlled structural family:

| Family | Geometry | Gap | Slots |
| --- | --- | --- | --- |
| `grid` | equal tracks, 2-4 columns | catalog content gap | 2-4 |

The `grid` family uses the same fixed logical canvas, theme padding, card
tokens, print reset, and responsive stage as existing layouts. At 16:9 and
4:3, its logical dimensions remain unchanged. The browser stage may scale the
canvas; it may not change slot measurements for canonical validation.

Variants are selected by data attributes and owned by deck CSS:

- `media-stage` increases the media slot's visual emphasis without changing
  the two-column logical geometry;
- `agenda` uses the timeline rule and localized agenda marker treatment;
- `steps` uses the existing timeline marker treatment;
- `highlight` uses the section accent treatment with a bounded content width;
- `hero` uses the lead treatment and preserves the existing lead constraints;
- `compare-grid` uses card treatment and equal tracks;
- `image-grid` uses media containment, captions, and stable card heights.

No variant may introduce an unlisted absolute coordinate, reorder source
content, or change the logical canvas. Computed-style tests cover all three
themes, both color modes, both canonical sizes, and all R1 entries that render
CSS.

## 11. Runtime and deterministic identity

The existing root protocol remains `margo-runtime/v2`. R1 composition identity
is deck-owned evidence, not a new root protocol. The deck layout task input
digest must include:

- `CompositionCatalogVersion`;
- document fingerprint;
- theme and color mode;
- geometry and validation request;
- mode and overflow algorithm version;
- every slide's ID, resolved composition name, variant, class, layout family,
  and ordered slot names.

The current digest implementation must stop using slide IDs alone for this
input. Two decks with equal slide counts but different composition geometry
must not receive the same deck layout task digest.

The canonical layout validation envelope adds:

```json
{"compositionCatalogVersion":"r1"}
```

for R1 deck renders. The field is required for R1 envelopes and is included in
canonical JSON before hashing. Existing v0.0.1 envelopes without compositions
remain parseable under their existing contract; the implementation must not
silently reinterpret an old envelope as R1.

## 12. Diagnostics

R1 adds these stable diagnostic codes:

| Code | Condition |
| --- | --- |
| `deck.composition_invalid` | Name is empty, unknown, malformed, or uses an unsupported YAML type. |
| `deck.composition_conflict` | Explicit class or layout marker disagrees with the resolved composition. |
| `deck.composition_slots_required` | Required composition slots are missing or cardinality is outside the catalog range. |
| `deck.composition_slot_invalid` | Slot is empty, duplicated, unknown, or out of source order. |
| `deck.composition_catalog_mismatch` | Runtime evidence or requested catalog differs from the registered R1 catalog. |

Existing `deck.layout_invalid`, `deck.slot_invalid`,
`deck.layout_slots_required`, `deck.class_combination_invalid`, and background
diagnostics remain authoritative for uncomposed input. A composed input must
not fall through to a generic layout error when a composition-specific code
can identify the cause.

## 13. Files and ownership

The implementation plan will assign ownership as follows:

- `deck/model.go`: composition values, defensive copies, and slide accessors;
- `deck/directives.go` and `deck/parse.go`: source grammar, inheritance,
  frontmatter default, spot clear, and diagnostics;
- `deck/composition.go`: the closed R1 catalog and resolver;
- `deck/layout.go`: composition-aware slot normalization while preserving the
  v0.0.1 validator;
- `deck/render.go` and `deck/page.go`: DOM attributes, localized labels, and
  root catalog identity;
- `deck/assets/deck.css`: grid family and R1 variants;
- `deck/runtime.go`: digest inputs and canonical envelope composition identity;
- `deck/*_test.go` and `deck/testdata/`: red-green coverage and fixtures;
- `docs/GOSHTOSO_MARKDOWN_DESIGN.md` and `README.md`: public authoring and
  compatibility documentation.

No source reference files or temporary visual-guide files enter the repository.

## 14. Test and evidence contract

Implementation must follow red-green-refactor. Before production code for each
behavior, a focused test must fail for the missing behavior and then pass with
the smallest implementation.

Required focused coverage:

1. all nine names normalize to the expected family, variant, and slot table;
2. frontmatter default, body inheritance, spot override, and `none` clear;
3. malformed, unknown, sequence, mapping, and empty directive values fail;
4. implicit layout and explicit matching layout both work;
5. explicit mismatched class/layout fails with the stable composition code;
6. each slot cardinality boundary and invalid ordering is covered;
7. source order, semantic slot data attributes, and no composition metadata on
   old slides are asserted in rendered HTML;
8. defensive model copies cannot mutate the parsed document;
9. `grid` renders 2, 3, and 4 tracks without logical overflow;
10. every R1 entry renders under modern, goshtoso, and minimal themes in light
    and dark modes;
11. 16:9 and 4:3 runtime evidence includes catalog identity and deterministic
    task digests;
12. changing only composition identity changes the deck layout task digest;
13. old v0.0.1 fixtures remain green and retain their existing DOM contract;
14. representative HTML and PDF captures preserve slot order, alt text,
    captions, and one logical page per slide.

Final gates for the implementation cycle are:

```sh
GOWORK=off go test ./... -count=1
GOWORK=off go vet ./...
GOWORK=off go mod verify
gofmt -l .
git diff --check
node --check tools/marpit-compare/make-compare.mjs
```

The visual review must rerun the comparator at 16:9, 4:3, desktop, and narrow
viewport sizes. A green Go suite alone is not evidence of composition geometry
or accessibility.

## 15. Acceptance criteria

The R1 design is accepted when:

- the written contract contains no unresolved placeholder or contradictory
  compatibility rule;
- every catalog entry has an explicit family, variant, slot contract, and
  accessibility treatment;
- v0.0.1 input behavior is stated and covered;
- runtime and PDF identity include the catalog version and complete composition
  inputs;
- the implementation plan is split into independently testable cycles;
- no external reference asset is required to build or test the feature.

Implementation is not accepted merely because the parser recognizes a name.
The catalog, DOM, geometry, runtime identity, PDF evidence, and visual review
must agree.
