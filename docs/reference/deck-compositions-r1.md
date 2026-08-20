# Margo deck compositions R1

Margo deck compositions are a closed, versioned vocabulary for expressing
common presentation structures in Markdown. The R1 catalog is intentionally
small: it captures recurring patterns without importing a source presentation,
its images, its fonts, or arbitrary author CSS.

The deck renderer owns composition normalization. A composition selects a
catalog entry, a visual variant, a layout family, a slot contract, and the
runtime identity used by screen and print validation.

## Opt in

Use a scalar `composition` directive in frontmatter to establish a default:

```yaml
---
title: Product overview
composition: content
---
```

The same directive can change the inherited value for following slides:

```markdown
<!-- composition: media-split -->
```

Prefix a directive key with `_` to affect only the current slide:

```markdown
<!-- _composition: none -->
```

The accepted values are the nine lowercase R1 names plus `none`. Values are
case-sensitive. Empty, unknown, sequence, mapping, and alias values fail with
`deck.composition_invalid`. Comments inside fenced code blocks remain
Markdown and are never interpreted as directives.

## R1 catalog

| Name | Intent | Family | Variant | Slots | Cardinality |
| --- | --- | --- | --- | --- | --- |
| `content` | Standard text, table, code, chart, or diagram slide | default body | `content` | body | implicit |
| `agenda` | Roadmap or table of contents | `timeline` | `agenda` | `item-1` through `item-6` | 3 to 6 |
| `media-split` | Media beside explanatory content | `columns` | `split` | `media`, `content` | exactly 2 |
| `media-stage` | Media-led split with a stronger media stage | `columns` | `stage` | `media`, `content` | exactly 2 |
| `steps` | Ordered process or implementation sequence | `timeline` | `steps` | `step-1` through `step-6` | 3 to 6 |
| `highlight` | One conclusion, quote, or key takeaway | `section` | `highlight` | body | implicit |
| `compare-grid` | Symmetric comparison of two to four items | `grid` | `compare` | `item-1` through `item-4` | 2 to 4 |
| `hero` | Opening, closing, or high-emphasis statement | `lead` | `hero` | body | implicit |
| `image-grid` | Ordered collection of two to four media cards | `grid` | `image` | `image-1` through `image-4` | 2 to 4 |

`content`, `highlight`, and `hero` use the ordinary slide body. Their
normalized catalog entries still carry a body role for manifests and
accessibility reporting. Structural compositions use explicit slot markers;
the composition supplies the family, so the initial `layout` marker and the
closing marker are optional.

## Slot syntax

```markdown
<!-- composition: media-split -->
<!-- slot: media -->
Media, image, or diagram content
<!-- slot: content -->
## Explanation
Supporting copy stays in source order.
```

The optional explicit layout form is accepted only when it agrees with the
catalog family:

```markdown
<!-- composition: compare-grid -->
<!-- layout: grid -->
<!-- slot: item-1 -->
First option
<!-- slot: item-2 -->
Second option
<!-- /layout -->
```

The resolver never guesses slots from headings, image count, tables, or CSS.
Slots remain in source order in the normalized model, DOM, reading order, and
keyboard order. CSS does not reorder them.

## Class and layout compatibility

When a composition has a style or structural family and no class is supplied,
the resolver adds the family class. An explicit class must equal that family
or be the `invert` modifier. A second style or structural class is a
`deck.composition_conflict`.

The `grid` family is controlled by the R1 resolver. `class: grid` without a
composition is not a supported v0.0.1 class. Existing v0.0.1 structural
layouts such as `columns`, `sidebar`, `compare`, `metrics`, `timeline`, and
`demo` retain their explicit marker and slot rules when no composition is
selected.

## Normalized model

The `deck` package exposes defensive copies of the normalized values:

```go
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

`CompositionCatalogVersion` is `r1`. An empty `CompositionSpec` means no
composition and is distinct from an explicitly selected `content` entry.
`Slide.Composition()` and `Document.Slides()` clone slot slices so callers
cannot mutate parsed state.

## HTML contract

An R1 render marks the root and every composed slide:

```html
<article class="margo-deck" data-margo-composition-catalog="r1">
  <section
    data-margo-composition="media-split"
    data-margo-composition-variant="split">
    <div
      class="margo-layout margo-layout--columns"
      data-margo-composition-family="columns">
      <div
        class="margo-layout__slot margo-layout__slot--media"
        data-margo-slot="media"
        data-margo-slot-role="media">
        ...
      </div>
    </div>
  </section>
</article>
```

Uncomposed v0.0.1 slides do not gain composition attributes. `agenda` and
`steps` use an ordered list. `compare-grid` and `image-grid` expose a labeled
group. Labels follow the slide language (`en`, `pt`, or `es`) through the
existing deck label catalog.

## CSS families

R1 CSS is bounded to the deck asset bundle. The `grid` family uses two, three,
or four equal tracks selected by the normalized slot count. `media-stage`
gives the media track more room; `compare-grid` uses card treatment;
`image-grid` centers media; `highlight` and `hero` select their existing
section and lead treatments through composition attributes. The fixed logical
canvas, theme tokens, card padding, print reset, and no-reordering rule remain
in force.

## Backdrop images

`backgroundImage` reserves one bounded backdrop layer behind the semantic slide
content. Local assets and approved gradient tokens may use finite
`backgroundPosition`, `backgroundRepeat`, and `backgroundSize` values:

```markdown
<!-- backgroundImage: assets/backdrop.svg -->
<!-- backgroundPosition: top-right -->
<!-- backgroundSize: cover -->
<!-- backgroundDecorative: true -->
```

Decorative backdrops emit `aria-hidden="true"`; informative local images require
`backgroundAlt` and emit a labeled image role. A slide never relies on a
backdrop alone for meaning, and the renderer never imports the reference deck's
backdrop assets.

## Watermark image and opacity (roadmap)

A watermark is host-owned furniture, not slide content. R1 has no authoring
directive for an image watermark. The planned contract accepts one trusted
local or embedded brand image plus a bounded theme opacity; arbitrary CSS
opacity is not part of the input surface.

Screen watermarks remain off by default. When enabled, the image is decorative,
`aria-hidden="true"`, `pointer-events: none`, and cannot overlap text or
controls. Print may show it as reserved page furniture, never as an overlay
that changes reading flow. The renderer does not import the reference deck's
watermark image.

## Runtime and PDF identity

The layout task input digest includes the catalog version and, for every
composed slide, the resolved name, variant, class, family, and ordered slot
names. Therefore changing only `media-split` to `media-stage` changes the task
identity.

`LayoutValidationEnvelope.CompositionCatalogVersion` accepts `r1` and rejects
other non-empty versions with `deck.composition_catalog_mismatch`. An empty
field remains the legacy envelope path for v0.0.1 evidence. PDF callers that
have resolved composition specs can use
`BuildPDFArtifactReportWithComposition`; the legacy four-argument builder is
byte-compatible when no composition identity is supplied.

## Diagnostics

| Code | Meaning |
| --- | --- |
| `deck.composition_invalid` | Name is empty, unknown, malformed, or uses an unsupported YAML type |
| `deck.composition_conflict` | Explicit class or layout marker disagrees with the catalog family |
| `deck.composition_slots_required` | Required slots are missing or cardinality is outside the catalog range |
| `deck.composition_slot_invalid` | Slot is empty, duplicated, unknown, or out of source order |
| `deck.composition_catalog_mismatch` | Runtime or PDF evidence names a catalog other than registered R1 |

Existing `deck.layout_invalid`, `deck.slot_invalid`,
`deck.layout_slots_required`, and `deck.class_combination_invalid` remain the
authoritative diagnostics for uncomposed v0.0.1 input.

## Icon reference syntax (roadmap)

The media system reserves a compact inline token for icons:

```markdown
Mês/ano :icon-name-here
```

`icon-name-here` is a lower-kebab symbol resolved from the embedded Goshtoso
catalog or from an explicitly declared iconpack. The eventual renderer must
emit an accessible inline SVG, preserve source text order, and fail closed for
unknown or ambiguous symbols. This is a roadmap contract; R1 does not yet
resolve icon tokens or import an iconpack.

## Pagination position and icon placement

Margo pins the page ordinal to the bottom-right corner on every paginated
slide. The reference template may show an ordinal in the top-right corner, but
that placement is not imported into the Margo contract. A host-provided
Goshtoso catalog icon joins the same bottom-right cluster, immediately before
or after the integer:

```text
before: :icon-name-here 6
after:  6 :icon-name-here
```

`before` and `after` are explicit placement values; the renderer does not infer
placement from language, direction, or theme. Decorative icons stay hidden from
assistive technology; informative icons require an accessible label. R1's host
API resolves exact symbols from the embedded Goshtoso Heroicons catalog and
embeds the sprite in the static page:

```go
deck.WithPaginationIcon(deck.PaginationIconConfig{
    Symbol: "hi-16-solid-clock", Placement: deck.PaginationIconBefore,
    Decorative: true,
})
```

The `Mês/ano :icon-name-here` author token and declared iconpacks remain
roadmap work; R1 fails closed for symbols outside the embedded catalog.

## Confidentiality badge

Confidentiality is host-owned chrome. When configured, render the Goshtoso
`badge.Badge` component immediately before the numeric ordinal in the same
bottom-right cluster; do not create a parallel badge primitive in Margo:

```go
badge.Badge(badge.Config{
    Label:      "Confidencial",
    Tone:       badge.ToneWarning,
    Appearance: badge.AppearanceSoft,
    Size:       badge.SizeSM,
})
```

The visible label remains the accessible status. Hosts configure it through
`deck.WithConfidentialityBadge`; the CLI exposes the same host setting through
`--confidentiality-badge`. Margo renders Goshtoso Badge before the ordinal and
does not expose an author-authored badge directive or import the reference
seal.

## Fixtures and review

The self-contained fixture is
[`deck/testdata/compositions-r1.md`](../../deck/testdata/compositions-r1.md).
It covers every catalog entry, a frontmatter default, a local override, a
spot clear, implicit structural layouts, body compositions, and every slot
contract. It contains no reference-deck assets.

The optional Margo/Marpit comparator accepts a composition manifest with
`--composition-manifest`. The manifest is metadata only; the comparator never
copies or imports the reference deck, its fonts, or its images. See
[`tools/marpit-compare/README.md`](../../tools/marpit-compare/README.md) for
the capture workflow.

## Scope boundary

R1 does not reproduce the reference deck's full layout count. It does not
register arbitrary compositions, accept arbitrary CSS, import PPTX files,
ship TOTVS assets, infer storytelling layouts, or implement R2 chart-story,
data-table, metric-wall, or dashboard compositions. Those are separate design
and approval work.
