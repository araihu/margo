---
name: Margo
description: Quiet technical editorial system for deterministic HTML and PDF
colors:
  accent: "var(--color-primary)"
  surface: "var(--color-surface)"
  surface-alt: "var(--color-surface-alt)"
  text: "var(--color-on-surface)"
  text-strong: "var(--color-on-surface-strong)"
  outline: "var(--color-outline)"
  accent-dark: "var(--color-primary-dark)"
  surface-dark: "var(--color-surface-dark)"
  surface-dark-alt: "var(--color-surface-dark-alt)"
  text-dark: "var(--color-on-surface-dark)"
  text-dark-strong: "var(--color-on-surface-dark-strong)"
  outline-dark: "var(--color-outline-dark)"
typography:
  display:
    fontFamily: "var(--document-font-heading)"
    fontSize: "var(--text-4xl)"
    fontWeight: "var(--font-weight-bold)"
    lineHeight: "var(--text-4xl--line-height)"
    letterSpacing: "-0.02em"
  headline:
    fontFamily: "var(--document-font-heading)"
    fontSize: "var(--text-2xl)"
    fontWeight: "var(--font-weight-bold)"
    lineHeight: "var(--text-2xl--line-height)"
    letterSpacing: "-0.02em"
  title:
    fontFamily: "var(--document-font-heading)"
    fontSize: "var(--text-xl)"
    fontWeight: "var(--font-weight-bold)"
    lineHeight: "var(--text-xl--line-height)"
    letterSpacing: "-0.02em"
  body:
    fontFamily: "var(--document-font-body)"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: "var(--document-line-height)"
  label:
    fontFamily: "var(--document-font-body)"
    fontSize: "var(--text-sm)"
    fontWeight: "var(--font-weight-medium)"
    lineHeight: "var(--text-sm--line-height)"
rounded:
  standard: "var(--radius-radius)"
spacing:
  base: "var(--spacing)"
  prose: "calc(var(--spacing) * 4)"
  section: "calc(var(--spacing) * 10)"
components:
  document-surface:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.text}"
    typography: "{typography.body}"
    rounded: "{rounded.standard}"
    padding: "clamp(calc(var(--spacing) * 5), 4vw, calc(var(--spacing) * 12))"
  figure-caption:
    textColor: "{colors.text}"
    typography: "{typography.label}"
    padding: "calc(var(--spacing) * 2)"
  data-table:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.text}"
    typography: "{typography.label}"
    rounded: "{rounded.standard}"
    padding: "calc(var(--spacing) * 2)"
---

<!--
margo-visual-spec-version: 1.1.0
status: approved
effective-date: 2026-08-17
-->

# Design System: Margo

**Visual specification:** 1.1.0

**Status:** Approved foundation

**Visitor mode:** Read

## Overview

**Creative North Star: "Quiet Technical Editorial"**

Margo turns one semantic Markdown document into trustworthy HTML and later PDF
projections. Its visual system must make technical material comfortable to read
without disguising evidence as decoration. The default feels like maintained
project documentation: calm, precise, legible, and intentionally restrained.

HTML is the canonical visual surface. PDF inherits its hierarchy, typography,
captions, tables, figures, and color semantics. Print may change pagination,
ink behavior, and page furniture; it must not invent a second aesthetic or a
different information hierarchy.

Margo owns a shared semantic visual foundation with three expressions:

1. **Editorial default:** relaxed reading rhythm and restrained standalone
   furniture. This is the canonical fallback and PDF base.
2. **Reference density:** a compatible option for documentation-heavy hosts,
   with tighter rhythm and stronger wayfinding.
3. **Publication expression:** host-owned art direction through bounded Margo
   document tokens and Goshtoso themes. It may be more expressive, but cannot
   change document meaning or accessibility behavior.

Goshtoso remains visual authority for primitives, themes, color modes, and
shared components. Margo adds only document roles, prose rhythm, responsive
reading behavior, pagination, and bounded print furniture.

**Key Characteristics:**

- One semantic source and one visual hierarchy across HTML and PDF.
- Quiet surfaces, strong type hierarchy, readable measure, sparse accent.
- Flat composition using spacing, rules, and tonal contrast instead of cards.
- Captions, tables, charts, and diagrams treated as first-class reading content.
- Content-driven responsiveness; fitting inside a viewport is not enough.
- Light and dark modes preserve hierarchy and geometry.
- Offline output remains legible without animation or browser runtime behavior.

### Specification versioning

This contract uses semantic versioning independently from Margo package
versions:

- **Major:** changes ownership, default visual register, semantic roles, or an
  established HTML/print behavior incompatibly.
- **Minor:** adds compatible roles, variants, components, or acceptance cells.
- **Patch:** clarifies language or corrects a rule without changing intended
  rendered behavior.

Every normative change must update the version marker, effective date, and the
sidecar generated timestamp. A document-version bump does not authorize a Git
tag, package release, dependency update, or deployment.

## Colors

Margo does not own a parallel palette. Every default color resolves through
Goshtoso semantic tokens recorded in frontmatter. Host themes may override the
bounded token surface; Margo components must continue using semantic roles.

### Primary

- **Document Accent:** reserved for links, focus, selected state, and small
  navigational emphasis. Accent never becomes decoration around ordinary prose.

### Neutral

- **Reading Surface:** canonical page background for document content.
- **Alternate Surface:** supporting background for disclosures, code-adjacent
  UI, table headers, and bounded component states.
- **Reading Text:** body copy and secondary document furniture.
- **Strong Reading Text:** headings, captions, labels, and important metadata.
- **Document Outline:** rules, table borders, and structural boundaries.

Light and dark projections must use corresponding Goshtoso families. Dark mode
is a real projection, not an inverted screenshot: backgrounds, text, outlines,
figures, diagrams, and print output must preserve readable contrast.

**The Semantic Color Rule.** Margo-authored CSS uses semantic tokens. Literal
colors require a reviewed, narrowly scoped interoperability reason.

**The Sparse Accent Rule.** Accent communicates action, navigation, focus, or
state. It does not decorate headings, page borders, charts, or metadata by
default.

**The Parity Rule.** Light and dark modes may change values, never hierarchy,
geometry, element visibility, or document meaning.

### Margo theme: mangrove daylight

The mascot supplies the color relationship, not a page background: blue-green
teal anchors the system, analogous mint carries the quiet supporting surfaces,
and the coral/berry counterpoint marks feedback and focus. Warm paper keeps the
light projection connected to the illustration without lowering reading
contrast. The dark projection moves that same relationship onto teal night and
uses mint as its readable action color.

The concrete theme lives in `themes/margo.css` with its required semantic
catalog in `themes/margo.tokens.yaml`. Coral and amber are never used for body
copy or ordinary links; they remain state-bound or decorative. Every text and
focus pairing is recorded for both modes so the mascot-derived palette cannot
silently become an accessibility regression.

## Typography

**Display Font:** Goshtoso document heading family

**Body Font:** Goshtoso document body family

**Label Font:** Goshtoso document body family

**Character:** Workhorse typography with deliberate editorial hierarchy. Type
must remain familiar enough for technical reading while spacing, weight, and
measure provide identity. Margo never introduces a reflexive serif/sans pairing
as a substitute for art direction.

### Hierarchy

- **Display:** one H1 per document; maximum measure 24ch; balanced wrapping.
- **Headline:** H2 sections; maximum measure 32ch; more space above than below.
- **Title:** H3 sections; maximum measure 40ch; visibly subordinate to H2.
- **Deep headings:** H4-H6 remain distinguishable through size, weight, rhythm,
  or label treatment. They must not collapse into visually identical body text.
- **Body:** 1rem default with the active document line-height; sustained prose
  measure 65-75ch.
- **Lead:** document purpose or summary directly below the title; one step more
  prominent than body copy without becoming a second heading.
- **Metadata:** compact but readable; never below 0.75rem and never used as the
  only explanation of document status.
- **Caption:** compact, visually secondary, and consistently positioned after
  the figure. Static and interactive chart captions use identical typography.
- **Code:** preserves the Goshtoso code component and theme. Inline code must
  remain distinct without overwhelming prose.

Display and section roles must adapt when their content no longer fits cleanly.
Responsive scaling is driven by wrapping and hierarchy, not device labels.
Text must remain usable at 200% browser zoom.

**The Reading Measure Rule.** Ordinary prose stays within 65-75ch. Tables,
charts, diagrams, media, and code may use wider bounded regions.

**The Heading Distance Rule.** A heading always has more space above than
below. Its relationship to following content must be visually obvious.

**The Caption Equality Rule.** A caption describes the figure regardless of
renderer. Static and interactive charts cannot expose different caption styles
or semantics.

## Layout

The editorial default is a centered document canvas with bounded reading
measure inside a wider media-capable surface. The modern theme currently caps
the document surface at 72rem; minimal caps it at 64rem. These outer widths do
not override the 65-75ch prose measure.

### Standalone entry

- Document title, lead, and essential metadata precede exhaustive navigation on
  screen.
- At narrow reading widths, the title and purpose must appear in the first
  viewport under ordinary browser chrome.
- Header and footer reflow into a calm single-column or wrapped composition;
  they never create two competing columns of fragments.
- A wide-screen page boundary may clarify the standalone artifact. At narrow
  widths the border and radius must not consume scarce reading space.
- A screen watermark is hidden by default. Any enabled watermark must occupy
  reserved non-content space and can never overlap text or controls. Print may
  use configured page furniture.

### Table of contents

- Desktop may use balanced columns when the list remains scannable.
- Narrow screens use native progressive disclosure.
- Initial narrow state exposes top-level sections; deeper headings remain
  available on demand.
- Links have visible affordance, comfortable targets, deterministic anchors,
  and current-location feedback when runtime support exists.
- A long document supplies a return-to-contents path without trapping keyboard
  or screen-reader users.
- Print uses complete deterministic contents independent from screen collapse.

### Responsive containment

- No document-level horizontal overflow at 390px or wider acceptance cells.
- Tables, code, and wide diagrams may own local horizontal overflow.
- Local overflow requires a visible cue; content cannot appear accidentally
  cropped.
- Interactive targets are at least 44px in one dimension or receive equivalent
  safe spacing.
- Diagrams preserve readable labels. A tiny fit-to-width projection is not
  considered responsive; use minimum canvas width, local scrolling, expansion,
  zoom, or a narrow alternative.

### HTML and print

- HTML defines semantic order and default visibility.
- Print removes controls with no paper meaning.
- Print restores deterministic table order and repeats table headers.
- Pagination avoids stranded headings and protects meaningful figures where
  practical.
- Print-specific rules may adjust margins, breaks, page furniture, ink, and
  overflow. They cannot introduce captions, data, or hierarchy absent from
  canonical HTML.

**The HTML-First Rule.** If content or semantics matter in PDF, they must exist
in canonical HTML first.

**The Content Breakpoint Rule.** Add a responsive change when content wraps,
clips, competes, or loses hierarchy; do not add arbitrary device categories.

**The Local Overflow Rule.** Wide content owns its overflow. The page never
does.

## Elevation & Depth

Margo is flat by default. Depth comes from whitespace, tonal surfaces, borders,
and reading order. Default document content has no diffuse shadow, glass,
gradient, or floating card treatment. Modal chart expansion may use the shared
Goshtoso overlay model because it represents real interaction state.

Host-owned publication themes may add photographic or material depth around
the document. They cannot apply decorative elevation to every paragraph,
section, table, or figure.

**The Structural Depth Rule.** A boundary must explain structure or state. If
removing a border or surface does not reduce comprehension, omit it.

## Shapes

Margo inherits Goshtoso's standard radius and outline tokens. Corners are
compact and functional. Ordinary prose never sits inside decorative cards.

- Document boundaries use the standard radius only on sufficiently wide
  standalone surfaces.
- Tables, disclosures, code, figures, and controls use shared component
  geometry.
- Pills are reserved for true tags, compact statuses, or filters. Metadata does
  not become a row of decorative capsules by habit.
- Chart, table, and diagram containers use the same border language when they
  need a boundary.

**The One Geometry Rule.** Static and interactive forms of the same content use
the same caption, table, border, and spacing geometry.

## Components

### Document shell

The shell establishes identity, provenance, theme, and print furniture without
competing with reading. Header, status, contents, article, and footer follow
semantic document order. Decorative backdrop art remains subtle and cannot
cover content.

### Metadata and status

Status labels use plain language. Terms such as "human review" must distinguish
pending, completed, or required state. Metadata remains subordinate to title
and lead.

### Table of contents

Contents are real navigation, not a decorative index. Use semantic `nav`, a
visible title, deterministic anchors, keyboard-safe links, progressive
disclosure on narrow screens, and complete print output.

### Figures and captions

- Every meaningful figure uses semantic figure/caption structure or an
  equivalent accessible relationship.
- Captions follow figures and share one visual role across images, charts, and
  diagrams.
- Captions describe content; they do not repeat the nearest heading without
  adding meaning.
- Meaningful SVG receives an accessible name and, when useful, description.
  Redundant SVG is hidden from the accessibility tree when adjacent semantic
  content communicates the same information.

### Static and interactive charts

- Both renderers expose the same figure, caption, and exact-data table semantics
  in HTML.
- The exact-data table is the only HTML data disclosure. Do not also show an
  "Exact category values" accordion.
- The exact-data table remains visible in HTML for static and interactive
  charts.
- Print hides chart data tables by default. The printable-accessibility flag
  includes them; this flag is off by default.
- Print hides expand, export, sorting, and other controls with no paper meaning.
- Static and interactive captions are visually identical.
- SVG and PNG exports preserve figure identity and do not replace semantic HTML
  before print preparation requires a stable image.
- Interactive chart controls use shared Goshtoso components, labels, focus
  behavior, and minimum target sizing.
- Every generated ID is unique within the document.

The target contract is explicit: standalone HTML, sites, and standalone PDF may
use the interactive renderer (PDF captures a printable raster), while the
`deck` target is static in both its HTML and PDF projections. Deck compatibility
checks reject `renderer: interactive` before rendering so controls are never
silently removed from an author-requested interactive chart.

### Tables

- Use semantic caption, header cells, scopes, and source order.
- Headers are visually distinct through weight and restrained tonal contrast.
- Numeric alignment follows meaning rather than blanket centering.
- Screen may scroll locally; print restores wrapping, full width, repeated
  headers, and deterministic source order.
- Sort controls expose state accessibly and return to source order for print.

### Mermaid and technical diagrams

- Diagrams have unique accessible names and useful text or source fallbacks.
- Narrow layouts keep labels readable through local scroll, expansion, zoom, or
  a simplified vertical projection.
- Runtime-pending and failed states remain understandable without an empty box.
- Diagram color follows active Goshtoso semantic roles in light, dark, and
  print.

### Code blocks

- Use the shared Goshtoso code component and selected code theme.
- Copy actions remain keyboard reachable, visibly focused, and comfortably
  sized.
- Horizontal overflow stays local and visibly discoverable.
- Print removes copy controls and wraps source only where the print contract
  requires it.

### Details and disclosures

Native disclosure is preferred. Summary remains a clear action with visible
focus. A disclosure must not duplicate permanently visible content.

### Focus and motion

All interactive elements use visible `:focus-visible` treatment based on the
semantic accent. Initial chart animation remains disabled for deterministic
capture. Optional motion must respect reduced-motion preferences and cannot be
required to understand document content.

## Do's and Don'ts

### Do:

- **Do** begin every visual decision in semantic HTML and project it to print.
- **Do** reuse Goshtoso tokens and components before adding Margo document roles.
- **Do** make title, purpose, and reading path clear in the first mobile
  viewport.
- **Do** keep prose within 65-75ch while letting evidence use bounded wider
  regions.
- **Do** give charts, diagrams, tables, captions, and controls equal attention
  in light, dark, mobile, desktop, and print review.
- **Do** preserve exact-data tables in HTML and gate their print visibility with
  the accessibility flag, off by default.
- **Do** validate the 390px and 1440px acceptance cells in both color modes,
  plus print output from the same HTML identity.
- **Do** require semantic headings, unique IDs, visible focus, useful names,
  AA text contrast, and usable output without runtime behavior.

### Don't:

- **Don't** create a second visual system that fights Goshtoso.
- **Don't** solve hierarchy with decorative cards, diffuse shadows, glass,
  gradients, pills, or ornamental chrome.
- **Don't** place exhaustive contents before title and purpose on narrow screens.
- **Don't** let watermarks or fixed furniture overlap document content.
- **Don't** call a diagram responsive when its labels became unreadable.
- **Don't** expose both a data accordion and a visible exact-data table.
- **Don't** style static and interactive chart captions differently.
- **Don't** add PDF-only content semantics absent from HTML.
- **Don't** use screenshot polish as substitute for keyboard, screen-reader,
  console, responsive, and printed evidence.
