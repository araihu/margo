# Margo Static Documentation Site

**Status:** Draft for independent review
**Date:** 2026-08-15
**Scope:** Documentation-site architecture through frame/shell composition and distribution.
**Out of scope:** implementation, blog/editorial/landing-page art direction, and deployment.

## 1. Product use cases

Margo has three HTML/PDF-related use cases:

1. **PDF generation.** Existing HTML-to-PDF output is already a strong baseline.
   Future work is limited to small print corrections and parity fixes.
2. **Static documentation sites.** This is the active slice. Margo must build its
   own documentation site from Markdown as a dogfooding benchmark.
3. **Blog, editorial, and landing pages.** These require a more expressive visual
   language and remain roadmap/backlog work. They must not distort the documentation
   defaults.

The documentation site is a reader-first, multi-page static publication. It must
support deep links, cross-page references, responsive reading, accessible Margo
content, light/dark themes, local assets, route metadata, and later PDF handoff.

It is not a marketing landing page, an authoring IDE, or a generic application
dashboard.

## 2. Architectural split

The work is split into two packages:

```text
/ssg   minimal, extensible static-site-generation engine
/site  the Margo documentation site, composed using /ssg
```

The existing generic `site` builder is the candidate for the `/ssg` package. The
new `/site` package/application is the Margo-specific consumer and dogfooding
benchmark. This is a conceptual split first: the existing public
`github.com/araihu/margo/site` import remains a compatibility facade while the
new `/ssg` package is introduced. The Margo publication command/configuration
must use a distinct application-owned path, so the facade and the publication
site are not ambiguous Go packages.

### 2.1 `/ssg`

`/ssg` owns the invariants shared by every static site:

- read and index a caller-supplied `fs.FS`;
- normalize source paths and derive output routes;
- detect locale variants and build a neutral route tree;
- compile Markdown through Margo;
- render the semantic Margo article fragment;
- rewrite internal links and local assets;
- derive and validate per-route metadata;
- provide neutral route, navigation, locale, and theme data to `/site` binding
  providers;
- validate the resulting semantic bindings before passing them to the selected
  frame or shell;
- collect frame/shell/site assets;
- emit the document, its single validated `<head>`, deterministic HTML
  artifacts, and `margo-manifest.json`.

`/ssg` does not own logos, product identity, sidebar markup, frame/shell layout, or
Goshtoso-specific visual composition. It should not import a concrete Goshtoso
App Shell.

### 2.2 `/site`

`/site` owns the Margo documentation publication:

- source FS and output destination;
- Margo site identity, logo, icon, and social image;
- base URL and locale defaults;
- homepage selection;
- theme catalog and custom CSS;
- frame selection and frame values;
- semantic binding providers for navigation, breadcrumbs, pagination, controls,
  and footer content;
- optional opinionated shell selection and shell values;
- the repository documentation corpus used as the benchmark.

`/site` selects a frame by default, or an explicitly configured opinionated
shell, and passes it the site/page data produced by `/ssg`. In frame mode,
`/site` owns semantic binding providers: it turns neutral navigation and route
data into fragments for the selected areas. The selected layout decides only
where those fragments appear; `/site` does not maintain a second navigation
tree or ask the frame to interpret component properties.

## 3. Composition boundary

The pipeline is:

```text
site.yaml
  -> /site configuration
  -> /ssg FS/index/route/locale/link/metadata pipeline
  -> Margo semantic article
  -> neutral route/navigation/theme data
  -> /site semantic binding providers
  -> frame or shell schema/capability discovery
  -> semantic area bindings
  -> selected frame or shell
  -> static HTML + assets + manifest
```

The selected frame or shell declares the areas into which the downstream
consumer may inject semantic content. A frame contributes only structural
divisions, responsive behavior, and client-side interaction primitives. A shell
may additionally compose headers, sidebars, mobile navigation, breadcrumbs,
footers, theme controls, or a completely different arrangement. None of those
visual regions are part of the shared Margo contract. Neither layout kind may
reimplement Markdown compilation, link validation, locale matching, metadata
validation, document `<head>` generation, or manifest generation.

## 4. Frame and shell contract

### 4.1 Frame layer

A `frame` is the minimal structural layout primitive. It declares the areas
available for downstream injection and supplies only:

- structural divisions and responsive behavior;
- stable area IDs and semantic roles;
- HTMX/Alpine event hooks and target attributes;
- the normalized swap semantics defined below;
- declared runtime/style resources required for those primitives.

A frame does not choose Goshtoso buttons, sidebars, tables, cards, headers, or
other product components. It may render containers and behavior hooks, but its
visual vocabulary is deliberately empty. Theme and locale context may affect
structural CSS or directionality only; frames do not own localized UI labels.
Margo uses frames as its default layout layer.

Frame resources are limited to the responsive, HTMX, and Alpine primitives
declared by the frame. They cannot introduce a component library or an
opinionated product UI through the resource catalog.

The initial builtin frame catalog follows the supplied base compositions. The
labels identify areas, not components:

| Frame | Areas |
| --- | --- |
| `main` | `main-content` |
| `top-main` | `top-nav`, `main-content` |
| `top-main-footer` | `top-nav`, `main-content`, `footer` |
| `top-left-main-footer` | `top-nav`, `left-nav`, `main-content`, `footer` |
| `top-left-main-right-footer` | `top-nav`, `left-nav`, `main-content`, `right-nav`, `footer` |
| `main-footer` | `main-content`, `footer` |

`main-content` is the conventional documentation host; the root documentation
profile assigns the shared `document` role to exactly one selected area. Child
frame instances remain structural and cannot inherit that role implicitly. The
other area IDs remain frame-specific. The catalog is extensible, but a new frame
must preserve the same schema and interaction contract.

The builtin catalog keeps payload semantics capability-based rather than
component-based. A frame may advertise `navigation` for `top-nav` or `left-nav`,
`toc` for `right-nav`, and `footer` for `footer`; these are accepted payload
kinds, not required widgets. The Margo documentation profile requires the
selected root layout to expose at least one `navigation`-accepting area. The
minimal `main` frame remains valid for generic `/ssg` sites, but fails the
documentation profile because it has nowhere to place the required navigation
binding.

The builtin visual contract is structural but not undefined. Each frame
publishes wide, mid, and narrow source order, a bounded `main-content` reading
measure, a semantic gap token, named sticky stacking order, and typed placement
and collapse rules through `FrameSchema.Layout`:

| Frame | Wide composition | Narrow composition |
| --- | --- | --- |
| `main` | `main-content` | `main-content` |
| `top-main` | `top-nav`, then `main-content` | `top-nav`, then `main-content` |
| `top-main-footer` | `top-nav`, `main-content`, `footer` | same source order, one column |
| `main-footer` | `main-content`, `footer` | same source order, one column |
| `top-left-main-footer` | `top-nav`, `left-nav`, `main-content`, `footer` | `top-nav`, `left-nav` drawer/stack, `main-content`, `footer` |
| `top-left-main-right-footer` | `top-nav`, `left-nav`, `main-content`, `right-nav`, `footer` | `top-nav`, `left-nav` drawer/stack, `main-content`, `right-nav` stack, `footer` |

Wide sidebars remain bounded around the 65–75ch reading measure. Narrow behavior
may use a `/site` binding provider's drawer fragment, but frame order and target
hooks remain deterministic. Frame schemas must publish the transformation; a
custom frame cannot claim responsive support while merely shrinking unreadable
columns.

The builtin capability matrix is normative. Every builtin publishes the same
capability fields and contract semantics (`Accepts`, `Multiple`, `MaxBindings`,
`BindingDefaults`, `BindingOrder`, and ordered slots), with values appropriate
to its areas. The `/site` documentation profile loads this matrix before accepting
the YAML example in Section 6:

| Frame | `document` area | `pagination` slot | `navigation` area(s) | `toc` area | `footer` area | deterministic order inside `top-nav` |
| --- | --- | --- | --- | --- | --- | --- |
| `main` | `main-content` | `main-content:after-article` | none (invalid for the docs profile) | none | none | — |
| `top-main` | `main-content` | `main-content:after-article` | `top-nav` | none | none | `navigation`, `breadcrumbs`, `theme_controls`, `locale_controls` |
| `top-main-footer` | `main-content` | `main-content:after-article` | `top-nav` | none | `footer` | `navigation`, `breadcrumbs`, `theme_controls`, `locale_controls` |
| `main-footer` | `main-content` | `main-content:after-article` | none (invalid for the docs profile) | none | `footer` | — |
| `top-left-main-footer` | `main-content` | `main-content:after-article` | `left-nav` and optional `top-nav` | none | `footer` | `navigation`, `breadcrumbs`, `theme_controls`, `locale_controls` |
| `top-left-main-right-footer` | `main-content` | `main-content:after-article` | `left-nav` and optional `top-nav` | `right-nav` | `footer` | `navigation`, `breadcrumbs`, `theme_controls`, `locale_controls` |

Each `main-content` area accepts exactly one `document` binding and one optional
`pagination` binding in the declared `after-article` slot; the slot renders
inside the single `<main>` after the article and exposes a distinct
`label.article_navigation` landmark (English fallback: “Article navigation”).
`left-nav`, `right-nav`, and `footer` are
non-multiple and have maximum one area binding. `top-nav` is multiple only for
the listed provider kinds, with a maximum of one binding per kind and four total
bindings; all other kinds are rejected. The
matrix is part of the builtin schema, not an inference from area names, and
custom frames must publish equivalent limits and ordering. A build fails when
the configured mapping cannot be represented by this capability contract.

The six builtin layout descriptors are fixed for v1; `mid` is not an
implementation-defined third mode. The dimensions shown below are the resolved
projection of the `modern` token catalog (for example,
`layout.sidebar.inline-size-wide` resolves to `16rem` and the reading measure
token resolves to `75ch`). A custom theme may replace those token values only
within the catalog's declared range; the frame never receives an unbounded raw
length. The table is therefore a stable fallback/rendered example, not a second
raw-length contract:

| Frame | wide columns / rows | mid columns / rows | narrow collapse |
| --- | --- | --- | --- |
| `main` | `minmax(0,75ch)` / `1fr` | `minmax(0,75ch)` / `1fr` | `none` |
| `top-main` | `minmax(0,1fr)` / `auto 1fr` | `minmax(0,1fr)` / `auto 1fr` | `top-nav:stack-before` |
| `top-main-footer` | `minmax(0,1fr)` / `auto 1fr auto` | `minmax(0,1fr)` / `auto 1fr auto` | `top-nav:stack-before`, `footer:stack-after` |
| `main-footer` | `minmax(0,1fr)` / `1fr auto` | `minmax(0,1fr)` / `1fr auto` | `footer:stack-after` |
| `top-left-main-footer` | `16rem minmax(0,75ch)` / `auto 1fr auto` | `14rem minmax(0,68ch)` / `auto 1fr auto` | `left-nav:drawer-inline-start`, `top-nav:stack-before`, `footer:stack-after` |
| `top-left-main-right-footer` | `16rem minmax(0,75ch) 16rem` / `auto 1fr auto` | `14rem minmax(0,65ch) 14rem` / `auto 1fr auto` | `left-nav:drawer-inline-start`, `top-nav:stack-before`, `right-nav:stack-after`, `footer:stack-after` |

The table is a human-readable projection of the typed descriptor, not a second
layout contract. Grid indices are one-based; `RowEnd` and `ColumnEnd` are
exclusive; every span must remain within the declared tracks; regions may not
overlap; and `SourceOrder` must enumerate every parent area exactly once in
DOM/read order. Child mounts are nested under their host and use the host
area's `BindingOrder` for internal order. Holes require an explicit
empty track declaration. CSS
placement must never change focus, keyboard, or assistive-technology order.
The `mid`/`narrow` transitions use the named `ContentBreakpoint` intervals in
the schema, with minimum inclusive and maximum exclusive CSS pixels.

For the builtin `modern` projection, `main` uses `grid.track_tokens.main-wide`
at wide/mid and `grid.track_tokens.narrow-content` at narrow; `top-main`,
`top-main-footer`, and `main-footer` use `grid.track_tokens.fluid` at wide/mid;
`top-left-main-footer` uses the sidebar token plus `main-wide`/`main-mid`; and
`top-left-main-right-footer` uses the sidebar token plus
`main-wide`/`main-three-column-mid`. These are frame-owned placements that
reference theme track tokens; the theme cannot add, remove, or reorder areas.

### 4.2 Opinionated shell layer

A `shell` is an opinionated composition built by a shell author. It may use a
frame internally, fill frame areas with Goshtoso or application components,
reserve areas for its own UI, and expose fewer downstream areas than the frame
it uses. Its exposed schema is therefore a new contract, not a transparent
view of its internal frame.

The first Goshtoso documentation shell remains available as an optional
opinionated shell. It is not the default Margo layout; the default is a frame.
Consumers that need a component-rich navigation, search, theme controls, or
other product-specific furniture may opt into a shell explicitly.

### 4.3 Shared area interaction semantics

Frames and shells use the same area-level response replacement semantics. The
default is `innerHTML`, matching HTMX:

```text
innerHTML  -> replace the target element's inner HTML
outerHTML  -> replace the entire target element
beforebegin -> insert before the target element
afterbegin  -> insert before the target's first child
beforeend   -> insert after the target's last child
afterend    -> insert after the target element
```

Each declared area has a canonical interaction target, trigger list, and default
`swap` value. `Target` defaults to the area's stable DOM ID; trigger names are
frame-declared hooks scoped to that area. A frame emits the corresponding
`hx-target`, `hx-trigger`, and `hx-swap` attributes plus neutral data/event
hooks. Alpine integration consumes the same enum rather than inventing a second
replacement vocabulary. The frame supplies behavior primitives, not an
application state machine.

The omitted `Swap` value is normalized to `innerHTML` before schema hashing.
Runtime responses may override the request default only through the
`HX-Reswap` response header, and only when the value belongs to the area's
normalized `AllowedSwaps` set (which is itself a subset of the six values
above). An invalid or area-disallowed response override is rejected by the
runtime adapter before DOM mutation and cannot change the target or escape the
frame subtree. Response overrides are runtime state; they are not
`AreaBinding` data and do not change the static schema hash.

Area schemas may narrow `AllowedSwaps` below the six-value global set. When it
is omitted, `/ssg` fills it with the target-legal set for that area and hashes
the normalized list. A root mount therefore never receives `beforebegin` or
`afterend` merely because the field was omitted. The boundary rules are strict:

- `beforebegin` and `afterend` are forbidden when the target is the root of a
  frame mount, because they could insert outside the authorized subtree;
- `beforebegin` and `afterend` are also forbidden for an area with
  `Live: polite` or `Live: assertive`; live updates must use the stable target.
- `outerHTML` is valid only when the response contains one root element with
  the same qualified target ID and re-declares the frame hooks; a one-shot area
  must declare that lifecycle explicitly;
- `innerHTML`, `afterbegin`, and `beforeend` preserve the target and its hooks;
- every swap records a deterministic focus policy: retain focus when its node
  survives, move to the replacement's first focusable descendant when declared,
  otherwise return to the triggering control or the nearest stable area target;
  an `area` fallback is focusable by native semantics or receives temporary
  `tabindex="-1"` for the move, with that temporary attribute removed after
  focus; if a triggering control was replaced, the nearest stable area target
  is the next fallback;
- areas with meaningful updates declare `Live: polite` or `Live: assertive`;
  `off` is the default. The adapter must not announce the same update twice.

These policies apply identically to builtin, command, and Go-module layouts and
are covered by keyboard, VoiceOver/Safari, and NVDA/Chrome acceptance runs.

### 4.4 Structural frame composition

A frame may contain child frames. Composition remains structural: a child frame
does not become a shell merely because it is mounted inside another frame. The
selected frame is the root of an acyclic composition tree; each child is mounted
into a parent-declared mount area.

The declarative representation is a `FrameComposition`: every `FrameNode` has a
stable instance ID, parent mount ID, distribution selector, structural values,
and recursively declared children. `/ssg` resolves this tree bottom-up. It
discovers and validates a child, renders it once, freezes its fragment and
schema hash, then passes that fragment through the parent's dedicated
`ChildrenByMount` channel. A child mount is not an `AreaBinding` and never
competes with a semantic provider for the same area. The root receives its own
resolved semantic bindings from `/site` and renders last. The same normalized
tree is sent to command-mode frame instances through `RootCompositionHash`,
`InstanceID`, `CompositionPath`, and child-frame inputs.

Composition rules:

- mount IDs are owned by the parent, live in a reserved `mount:<id>` namespace,
  and become qualified DOM IDs and event targets nested inside their declared
  `HostArea`, so two child frames cannot collide;
- every mount declares a host area, target, and whether it is
  exclusive; v1 mounts are exclusive by default, so their host area cannot also
  receive a semantic binding. A non-exclusive mount is invalid unless its
  schema publishes an explicit `BindingOrder` entry containing
  `mount:<id>` plus every compatible semantic kind;
- a child frame may address only its own subtree; a parent may explicitly export
  a target when cross-frame behavior is required;
- `/ssg` discovers and validates every child schema, merges and deduplicates
  resources, and includes the composition tree, child identities, and resolved
  values in the root schema hash;
- child frames receive their own namespaced values, child-frame inputs, and
  semantic bindings; a parent cannot
  silently mutate a child's internal options;
- composition cycles, duplicate mount IDs, mount/area collisions, unsupported
  child contracts, wrong-kind child inputs, and unresolved required mounts fail
  the build;
- the root composition has exactly one `Role: document` area. Child frames are
  structural subframes and cannot add a second document host, `<main>`, or skip
  link.

Example nested configuration:

```yaml
frame:
  command: ./bin/composite-frame
  protocol: margo.ssg.frame/v1
  children:
    utility:
      builtin: top-main
      values:
        areas:
          top-nav:
            sticky:
              enabled: true
              edge: block-start
```

`children.<mount>` is valid only when the parent schema declares that dedicated
mount (the example assumes `utility` is an exclusive mount hosted by a
structural area). Child values are validated against the child schema and
cannot be addressed by the parent's unqualified option paths. A builtin that
does not publish a mount rejects `children` rather than silently mounting into
an area such as `top-nav`.

The conventional `main-content` name does not itself create a document role. The
documentation profile assigns `Role: document` once, on the root host selected
for the route. This keeps the builtin frame catalog composable without creating
duplicate document landmarks.

Frame areas may opt into sticky positioning as a structural behavior. Sticky is
off by default and is expressed through typed options, not arbitrary CSS:

- `top-nav` may use logical `edge: block-start`;
- `footer` may use logical `edge: block-end`;
- an optional logical offset and content breakpoint may be declared;
- breakpoint names describe content behavior (`narrow`, `mid`, `wide`) rather
  than device brands; each frame publishes the thresholds it supports;
- each frame publishes a resolved `MinimumReadingBlock` token/value and a
  deterministic `StickyOrder`; sticky is disabled before the remaining reading
  block falls below that value;
- the frame must reserve the sticky area so headings and anchor targets are not
  obscured;
- the parent frame owns the scroll context for nested sticky areas;
- cumulative top/bottom offsets include parent and child sticky stacks plus safe
  areas; `scroll-margin-block` and content padding use the same resolved stack;
- when combined sticky occupancy leaves less than the frame's declared minimum
  reading block, lower-priority sticky areas disable in deterministic order
  (`footer`, then nested child areas) before the primary top navigation;
- sticky areas use local overflow for long navigation and never trap keyboard
  focus;
- print styles disable sticky positioning without changing source order or
  semantics.

`top`/`bottom` are accepted only as compatibility aliases for
`block-start`/`block-end` and are normalized before schema hashing. In RTL
layouts, navigation drawers use logical `inline-start`/`inline-end` rather than
physical left/right edges.

### 4.5 Property and binding channels

Frame customization has three deliberately separate channels:

1. `frame.values` configures structural behavior: area options, breakpoints,
   sticky positioning, event primitives, and child-frame values.
2. `/site` binding providers configure semantic content: navigation,
   breadcrumbs, pagination, theme/language controls, footer content, and other
   payloads. Providers resolve their component properties before producing the
   frozen `AreaBinding`; the frame only places the resulting fragment.
3. `shell.values` configures opinionated shell components and remains a shell
   concern. Those values never become frame options.

Structural values are static for a build. Omitted values resolve to the frame's
declared defaults before schema hashing; explicit defaults and omitted defaults
therefore produce the same hash. Unknown paths, invalid types, enum values,
breakpoints, or sticky edges fail with the selected layout identity and full
property path. Page-specific or runtime state does not enter `frame.values`; it
uses the binding provider plus HTMX/Alpine state instead.

The frame contract must expose option descriptors sufficient for tooling and
diagnostics: path, type, default, allowed values or range, and a concise
description. A generic map remains the transport shape, but it is not an
unvalidated escape hatch.

Example structural customization:

```yaml
frame:
  builtin: top-left-main-footer
  values:
    areas:
      top-nav:
        sticky:
          enabled: true
          edge: block-start
          offset: 0
      left-nav:
        collapse_at: narrow
```

### 4.6 Semantic binding resolution

`/site` providers publish a finite semantic kind and optional typed `props`; the
provider resolves those props into the frozen fragment before `/ssg` creates an
`AreaBinding`. The frame never receives provider props and never interprets
component configuration.

Binding targets resolve deterministically in this order:

1. an explicit `/site` mapping for the provider kind, composition path, and
   optional declared slot;
2. the selected frame schema's `BindingDefaults` entry;
3. the only area whose `Accepts` list contains that kind.

If two or more candidate areas remain without an explicit mapping or schema
default, the build fails with the provider kind, composition path, and candidate
IDs. A target must accept the provider kind, and its `MaxBindings` and
`MaxBindingsByKind` limits must not be exceeded. A slot must be declared by the
target area and accept the provider kind; slots are rendered in their declared
source order and do not count as a second area. `BindingOrder` is a total order
per area; providers of the same kind are ordered by their canonical route key
before the final order is hashed.
In frame mode, the documentation profile requires exactly one
`navigation` binding and exactly one `document` binding; breadcrumbs, pagination,
theme controls, locale controls, TOC, and footer are optional only when their
providers are not enabled by the selected site configuration. In shell mode, the
shell may render navigation internally from `ShellInput.Navigation`, but it must
expose one active navigation outcome and still preserve the one `/ssg`-owned
document binding.

The `/site` configuration may make the mapping explicit:

```yaml
bindings:
  navigation:
    area: left-nav
    props:
      label: Contents
  breadcrumbs:
    area: top-nav
  pagination:
    area: main-content
    slot: after-article
  theme_controls:
    area: top-nav
  footer:
    area: footer
```

Nested targets use the same mapping with a qualified path. For example,
`composition_path: [utility, controls]` and `area: main-content` addresses the
`main-content` area of that child instance, not the root area with the same
name. An omitted path means the root instance. The path is part of the
canonical binding key and manifest, so a provider cannot accidentally collide
with a child frame.

Mapping keys and provider props are canonicalized before the manifest is written.
Unknown provider kinds, invalid props, duplicate non-multiple targets, and
provider fragments without their declared semantic kind fail the build. This
keeps the same provider behavior across builtin frames, command frames, and
Go-module frames.

`templ.Component` is the renderable output, but it is not sufficient as the full
public contract: a component alone has no typed area discovery, capability
negotiation, or access to route metadata, navigation, brand assets, or the Margo
article.

The conceptual contract is schema-first and deliberately does not prescribe a
visual frame or shell composition:

```go
type SwapMode string

const (
	SwapInnerHTML  SwapMode = "innerHTML"
	SwapOuterHTML  SwapMode = "outerHTML"
	SwapBeforeBegin SwapMode = "beforebegin"
	SwapAfterBegin  SwapMode = "afterbegin"
	SwapBeforeEnd   SwapMode = "beforeend"
	SwapAfterEnd    SwapMode = "afterend"
)

type AreaDescriptor struct {
	ID           string   // frame/shell-owned identifier
	Role         string   // optional shared semantic role; docs requires "document"
	Required     bool
	Multiple     bool
	MaxBindings  int      // zero means unlimited only when Multiple is true
	MaxBindingsByKind map[string]int // zero/absent means use MaxBindings
	Accepts      []string // payload kinds accepted by this area
	Slots        []SlotDescriptor // ordered semantic slots nested inside this area
	Target       string   // stable local DOM target; defaults to ID
	Triggers     []string // frame-declared HTMX/Alpine event hooks
	AllowedSwaps []SwapMode // defaults to the target-legal set when omitted
	Live         string   // off, polite, or assertive; off by default
	Focus        string   // retain, first-focusable, trigger, or area
	Swap         SwapMode // innerHTML by default; one of the six HTMX-compatible modes
}

type SlotDescriptor struct {
	ID      string
	Accepts []string
	Order   int // one-based order inside the area; unique and contiguous
}

type FrameOptionDescriptor struct {
	Path        string // e.g. "areas.top-nav.sticky.enabled"
	Type        string // boolean, enum, length, breakpoint, or number
	Default     any
	Allowed     []string
	Min         *float64
	Max         *float64
	Description string
}

type FrameMountDescriptor struct {
	ID         string // parent-owned mount ID in the mount:<id> namespace
	HostArea   string // parent structural area containing the mount target
	Target     string // stable qualified child target within HostArea
	Required   bool
	Exclusive  bool   // v1 defaults true; exclusive mounts cannot share HostArea bindings
	Contract   string // margo.ssg.frame/v1
}

type FrameRegion struct {
	Area         string // parent area ID; child mounts are nested inside this region
	RowStart     int
	RowEnd       int
	ColumnStart  int
	ColumnEnd    int
	Collapse     string // none, stack-before, stack-after, drawer-inline-start, drawer-inline-end
}

type FramePlacement struct {
	Rows        []string
	Columns     []string
	Regions     []FrameRegion // declared in DOM/source order
	SourceOrder []string      // area IDs, same order as DOM/source order
}

type FrameLayoutDescriptor struct {
	Wide                FramePlacement
	Mid                 FramePlacement
	Narrow              FramePlacement
	MainMeasure         string // semantic token or bounded ch measure
	GapToken            string // semantic spacing token
	MinimumReadingBlock string // resolved semantic token or bounded length
	StickyOrder         []string
	Breakpoints         []ContentBreakpoint
	DrawerInlineSize    string // semantic token or bounded logical length
	DrawerMaxInlineSize string // semantic token or bounded logical length
}

type ContentBreakpoint struct {
	Name          string
	MinCSSPx      int  // inclusive
	MaxCSSPx      *int // exclusive; nil means no upper bound
}

type FrameSchema struct {
	Contract        string // margo.ssg.frame/v1
	Areas           []AreaDescriptor
	Options         []FrameOptionDescriptor
	Mounts          []FrameMountDescriptor
	Layout          FrameLayoutDescriptor
	BindingDefaults map[string]string // payload kind -> area ID; slot defaults are profile-defined
	BindingOrder    map[string][]string // area ID -> provider-kind/mount order
	Resources       []ResourceRequirement
}

type FrameContext struct {
	Locale           string
	Direction        string // resolved ltr or rtl
	Theme            ThemeContext
	Profile          string // e.g. "margo-docs"
	Root             bool
	InstanceID       string
	CompositionPath  []string
}

type FrameInput struct {
	SchemaHash         string // instance schema hash
	RootCompositionHash string
	InstanceID         string
	CompositionPath    []string
	Bindings           map[string][]AreaBinding
	ChildrenByMount    map[string]FrameChildBinding
}

type FrameChildBinding struct {
	MountID          string
	InstanceID       string
	CompositionPath  []string
	SchemaHash       string
	Digest           string
	Fragment         templ.Component
}

type FrameOutput struct {
	Fragment   templ.Component // structural fragment, without document wrappers
	Assets     AssetSet
	SchemaHash string
}

type Frame interface {
	Schema(FrameContext) (FrameSchema, error)
	Render(FrameInput) (FrameOutput, error)
}

type ShellSchema struct {
	Contract  string // margo.ssg.shell/v1
	Areas     []AreaDescriptor
	Locales   []string
	Labels    map[string]map[string]string // locale -> label key -> localized value
	Resources []ResourceRequirement
}

type AreaBinding struct {
	Kind             string
	CompositionPath []string       // empty for root; qualified for child frames
	Slot             string         // empty for an area binding; e.g. document:after-article
	Token            string         // deterministic comment-marker identity
	Digest           string         // canonical digest of the bound payload
	Component        templ.Component // serialized HTML fragment at a process boundary
}

type BindingSpec struct {
	Kind             string
	CompositionPath []string       // empty for root; qualified child mount path
	Area             string
	Slot             string         // optional declared semantic slot within Area
	Props            map[string]any // resolved by the /site provider, never by the frame
}

type FrameNode struct {
	InstanceID string
	MountID    string
	Selector   string // builtin, command, or go_module identity
	Values     map[string]any
	Children   []FrameNode
}

type FrameComposition struct {
	Root FrameNode
}

type ShellContext struct {
	Locale    string
	Direction string // resolved ltr or rtl
	Theme     ThemeContext
}

type ShellInput struct {
	Route      Route
	Metadata   PageMetadata
	Brand      Brand
	Navigation NavigationTree
	Locale     LocaleContext
	Theme      ThemeContext
	Assets     AssetSet
	SchemaHash string
	Bindings   map[string][]AreaBinding // keyed by selected-layout AreaDescriptor.ID
}

type ShellOutput struct {
	Fragment   templ.Component // shell-owned fragment, without document wrappers
	Assets     AssetSet
	SchemaHash string
}

type Shell interface {
	Schema(ShellContext) (ShellSchema, error)
	Render(ShellInput) (ShellOutput, error)
}
```

`FrameInput` intentionally contains only the resolved schema hash, frozen area
bindings, and frozen child fragments addressed by declared mounts. Route
metadata, brand, navigation, and page context are resolved by `/site` binding
providers before frame rendering; passing them directly to a frame would turn
the structural contract into a shell. `ShellInput` retains the broader page
context because an opinionated shell may compose those concerns.

Binding resolution is instance-aware. Every semantic target is the pair
`(CompositionPath, AreaID, Slot)`; root bindings use an empty path, and child
bindings use the qualified mount path. An empty slot addresses the area itself;
a non-empty slot addresses a schema-declared semantic slot inside that area.
`/site` resolves providers for each target before the bottom-up render, so a
child can host a `toc` or `footer` without colliding with a root area of the same
name. A child mount itself is addressed only by `ChildrenByMount`, never by this
semantic key.

The concrete API may use a factory or function pair, but the data boundary must
remain equivalent for frames and shells. `AreaDescriptor.ID` belongs to the
selected frame or shell and is not a cross-layout visual name. Shared semantic
roles are intentionally tiny: for the documentation profile, exactly one
declared area must have `Role: document`, be required, not be multiple, and
accept the `document` payload kind. No other area may require a downstream
binding in that profile; frame/shell-owned defaults are internal to the layout
and are not declared as required areas. All other areas and roles are
layout-specific. A frame or shell with no `document` area cannot be selected
for the documentation profile.

`/ssg` canonicalizes and validates the discovered schema before creating
bindings: the contract is supported; area IDs, targets, triggers, swap/focus/
live policies, multiplicity/`MaxBindings`, capability and ordered slot lists, binding
order, option descriptors, mount host/target/exclusivity plus complete
`BindingOrder` coverage, and resources
are non-empty, unique, and valid; in offline mode its complete resource closure
contains no unresolved remote URL; a shell's
locale and label catalogs are also valid; and the documentation-profile
invariants above hold. It records
the canonical schema hash in the manifest
and passes that hash to the render session. Unknown area IDs, missing required
areas, multiplicity or maximum violations, unsupported payload/slot kinds,
duplicate/non-contiguous slot orders,
mount/area collisions, invalid structural values, unresolved mounts, or a render
result with a different schema hash fail the build. Optional areas with no
binding are omitted. The selected layout
chooses the placement, markup, accessibility landmarks, and responsive behavior
of every area, subject to the profile's semantic invariants.

`FrameSchema.Layout` must list each declared parent area exactly once per supported
responsive mode, with explicit grid rows, columns, region spans, and a
collapse strategy (`none`, `stack-before`, `stack-after`,
`drawer-inline-start`, or `drawer-inline-end`). It must keep `main-content`
within the declared reading measure and reference semantic spacing tokens. The
resolved `MinimumReadingBlock`, `StickyOrder`, and every published breakpoint
threshold are part of the layout schema and hash. Raw lengths are allowed only
for an option whose descriptor explicitly permits them; frame defaults use the
selected theme's semantic tokens and named content breakpoints.

`ContentBreakpoint` intervals are sorted, non-overlapping, and cover every
supported layout state; `MinCSSPx` is inclusive and `MaxCSSPx` is exclusive.
Every `FrameRegion` uses one-based tracks with exclusive end indices, stays
within its placement's row/column count, and appears once in `SourceOrder`.
Child mounts do not create a second top-level region: their `HostArea` region
contains the qualified mount target, and the host area's `BindingOrder`
determines the mount's internal DOM/focus order. The build proves that the child target is
a descendant of `HostArea`, that exclusive mounts do not share the host's
semantic binding, and that non-exclusive mounts follow the host's declared
`BindingOrder`. Overlapping regions, duplicate source-order entries, missing
areas/mounts, and implicit CSS reordering fail schema validation.

`MainMeasure`, `GapToken`, `MinimumReadingBlock`, typography references, and
spacing references resolve against the selected theme's published semantic
token catalog. `/site` validates that every token exists and that the catalog
publishes a readable type scale, line-height, minimum text size, responsive
behavior, and 65–75ch content measure. A raw length must state why it cannot be
expressed as a token and is still included in the schema hash. The frame may
place those resolved values, but does not invent a second token vocabulary.

`Schema(FrameContext)` receives the resolved root/profile/instance context and
direction before canonicalization. `/ssg` canonicalizes the BCP 47 locale to
`Direction: ltr` or `Direction: rtl` using its locale data, then calls the frame
once for each instance with `Profile: "margo-docs"` and `Root: true` only for the
root; child schemas receive the qualified instance path and `Root: false`. The
`document` role, direction-sensitive placement, and other profile invariants are
therefore validated before the schema hash is frozen; no caller may mutate a
schema after hashing to add a document host or recompute direction locally.

Margo renders each binding once into an isolated HTML-fragment context, freezes
the resulting canonical fragment bytes, and computes
`sha256("margo.ssg.area-payload/v1\\0" + canonical_fragment_bytes)`. It derives
a deterministic `AreaBinding.Token` as
`sha256("margo.ssg.area-marker/v1\\0" + canonical_json(schema_hash, route,
composition_path, area_id, slot, ordinal, kind, payload_digest))`. It wraps the frozen binding in paired HTML
comment sentinels carrying that token. The selected layout must preserve each
pair exactly once; `/ssg` reparses the marked subtree as an HTML fragment with the
same parser/canonicalizer, compares it with the declared payload digest, then
removes the sentinels. The comments are layout-neutral and never become DOM
boxes or final artifact bytes. This proves that the required `document` payload
was neither dropped, duplicated, nor changed by the selected layout.

For the documentation profile, `/ssg` materializes the frozen `document` binding
inside the single root `<main>` host and emits one skip link targeting that
stable ID before the selected layout fragment. These are shared semantic
primitives, not frame components. The same post-render validation must prove that
the marked `document` subtree is a descendant of the single rendered `<main>`,
that the document subtree contains the single `h1`, and that the skip link
resolves to that same `<main>` ID. A selected layout cannot satisfy the profile
with an empty `main` beside an unrelated document area.

Before this check, `/ssg` rejects any returned fragment containing a doctype or
`<html>`, `<head>`, `<title>`, or `<body>` wrapper in builtin, Go-module, and
command modes. A selected frame or shell may render any internal elements it
needs, but it may not smuggle a second document into the outer Margo document.
It must preserve the `/ssg`-owned document `<main>` host, its stable ID, and the
single skip link; a layout may add landmarks around them but cannot replace or
duplicate those primitives.

Schema hashing uses Margo's canonical JSON routine with sorted object keys and a
domain-separated SHA-256 preimage (`margo.ssg.layout-schema/v1`). The canonical
`Contract` value distinguishes `margo.ssg.frame/v1` from
`margo.ssg.shell/v1`; resource maps, and shell label maps when present, therefore
have one deterministic representation. Frame option descriptors and mount
descriptors are canonicalized in the same schema. An empty resource list is
valid; every declared resource still needs its own valid identity and placement.

The schema is session-scoped by the effective frame/shell identity, immutable
layout values, locale, theme name, and color mode. Equivalent contexts may reuse
a hash; a multilingual or theme-varying build records a sorted context-to-hash
map in the manifest and passes the matching hash to each render. The Go-module
constructor's `values` and the command process's initial configuration are the
only layout-value sources; the context does not duplicate them.

`LocaleContext` contains the current canonical BCP 47 tag, supported tags,
alternate logical routes, and labels resolved by `/site` from the selected
shell's `ShellSchema.Labels` catalog when a shell is selected. `ThemeContext`
contains the selected theme, available themes, `allow_switch_theme`, and
`color_mode`. A shell can render language/theme controls from these contexts
without inventing parallel configuration. In frame mode, `/site` resolves the
same contexts into semantic control bindings. A frame remains label-free and
locale-agnostic; `/ssg` supplies localized document and navigation data.
A shell must expose supported locales and localized label values through
`ShellSchema` before page rendering; missing labels fail the build instead of
being silently filled by the shell.

`ResourceRequirement` is a declarative, validated dependency record rather than
raw head HTML:

```go
type ResourceRequirement struct {
	Placement  string // "head" or "body-end"
	Kind       string // "stylesheet", "script", "module", or "preload"
	URL        string
	Inline     string
	Integrity  string
	Attributes map[string]string
}
```

`/ssg` validates ordering, URL/base-path resolution, duplicate identities,
allowed attributes, and integrity policy, then emits the corresponding
`<link>`/`<script>` tags. Exactly one of `URL` and `Inline` is allowed. Inline
content is legal only for `script` or `module`; `stylesheet` and `preload`
requirements use `URL`, and every declared resource has a valid identity and
placement. `preload` and `stylesheet` use `head`; `script` and `module` may use
`head` or `body-end`. Inline content is restricted to deterministic frame/shell
bootstrap scripts and its hash is recorded in the manifest. Resources are
discovered by the selected layout's `Schema(context)` after the resolved
site/locale/theme context is known, then frozen under the schema hash for the
render session. Page metadata remains exclusively owned by `/ssg`.

The builtin frame catalog is Margo-owned and has no Goshtoso component
dependency. The first Goshtoso adapter is owned by the `/site` publication application
(within the root Margo Go module) and must explicitly depend on the
`github.com/araihu/goshtoso-app-shells` module. It adapts the
`componentdocshell` package to the schema-first contract: the upstream shell
declares its areas and renders a shell-owned fragment after receiving bindings.
It must not call a complete-document layout, because `/ssg` owns the outer
document and metadata head. The current public `Layout`, `Fragment`, and `Head`
APIs are not yet sufficient for this adapter: the selected upstream version is
a prerequisite and must export (1) area schema/capability discovery with
localized label values, (2) a
fragment renderer with no `<html>`, `<head>`, `<title>`, or `<body>` wrapper,
and (3) an ordered structured resource catalog, including the first-paint
bootstrap currently emitted inline. The exact module version is pinned by the
root Margo Go module at implementation time and recorded in build provenance;
until that upstream contract exists, `goshtoso-docs` is not declared
implemented.

The adapter maps Margo's semantic payloads to the shell-declared areas. It does
not assume that a shell has a header, sidebar, main column, or any other named
visual region.

`/ssg` remains independent of the concrete Goshtoso package. Goshtoso provides
the first optional shell implementation and its visual primitives, not the SSG
engine's route, content, or document-head model. `/ssg` wraps the returned
`Fragment` in the outer document and owns the single server-rendered `<head>`.

## 5. Frame and shell distribution modes

Go cannot portably import and execute an arbitrary source module at runtime.
Therefore frame or shell selection has three explicit modes. The same modes,
values, schema discovery, hashing, and validation apply to either layout kind;
the selected kind determines whether the process implements `ssg.Frame` or
`ssg.Shell` and whether it advertises `margo.ssg.frame/v1` or
`margo.ssg.shell/v1`.

### 5.1 Builtin frame or shell

```yaml
frame:
  builtin: top-left-main-footer
```

Builtin frames require no consumer-side Go installation. They are compiled into
the Margo distribution and versioned with the frame contract. The initial
builtin frame catalog is the six structural compositions in Section 4. An
opinionated builtin shell is selected explicitly:

```yaml
shell:
  builtin: goshtoso-docs
```

`/site` documentation builds default to the builtin `top-left-main-footer` frame
when neither `frame` nor `shell` is configured, because the profile requires a
navigation-capable area. A shell is never selected implicitly. The generic
`/ssg` API has no implicit layout; callers may select the minimal `main` frame
explicitly for sites without the documentation profile.

### 5.2 Prebuilt frame or shell command or bundle

```yaml
shell:
  command: ./bin/my-docs-shell
  protocol: margo.ssg.shell/v1
  values:
    sidebar_style: compact
```

For a structural frame, the same shape uses `frame` and
`protocol: margo.ssg.frame/v1`; the executable must expose only frame areas and
primitive behavior.

The frame or shell author may implement the templ design in Go and publish a
prebuilt executable or verified bundle. The consumer runs Margo, not Go. The
process protocol is versioned, and its request shape follows the selected
contract:

```text
margo.ssg.frame/v1 request:
  root_composition_hash, instance_id, composition_path
  schema_hash, child-frame bindings, semantic bindings

margo.ssg.shell/v1 request:
  route, metadata, brand, navigation, locale, theme, assets
  schema_hash, bindings
```

A frame command receives only the serialized equivalent of `FrameInput`: the
root composition hash, instance identity/path, instance schema hash, child-frame
bindings, and frozen semantic bindings. It does not receive route metadata,
brand, navigation, or page context. A shell command receives the broader equivalent of
`ShellInput` because it may compose those concerns. In both modes, the
`document` payload travels only as the binding for the schema-declared
`document` area. The layout returns its structural or opinionated fragment (for
example `fragment_html`) plus declared assets and the same schema hash. `/ssg`
still validates the bindings, writes the outer document, and emits each schema
resource at its declared `head` or `body-end` placement.

Protocol negotiation must fail with a Margo diagnostic that names the requested
and advertised protocol versions. Untrusted layout downloads are out of scope;
the command/bundle must be supplied or verified by the site owner.

### 5.3 Go-module development mode

```yaml
shell:
  go_module:
    import: github.com/acme/docs-shell
    version: v1.2.0
    constructor: New
    values:
      sidebar_style: compact
```

This mode is for frame or shell authors and CI that accept a Go build step. The
expected constructor follows the selected layout kind:

```go
func New(values ssg.Values) (ssg.Frame, error) // frame
func New(values ssg.Values) (ssg.Shell, error) // shell
```

`margo ssg gen site.yaml` generates a small typed Go driver that pins the module
and asserts the selected layout contract. `margo ssg build site.yaml` is the
Margo-owned build path: it compiles/runs that driver, recognizes the generated
contract probe, and reports a constructor mismatch before or alongside compiler output.
If a consumer runs `go build` directly, only the normal compiler diagnostic is
available; the large Margo diagnostic is guaranteed by `margo ssg build`.

```text
MARGO_SSG_LAYOUT_CONTRACT_MISMATCH

Configured layout: github.com/acme/docs-shell@v1.2.0
Expected contract: margo.ssg.frame/v1 for `frame`, or margo.ssg.shell/v1 for `shell`

Expected constructor for a selected shell:
func New(values ssg.Values) (ssg.Shell, error)

For a selected frame, the expected constructor is:
func New(values ssg.Values) (ssg.Frame, error)

The configured layout exposes an incompatible constructor.
Use a release implementing the selected frame/shell contract,
or select a compatible Margo version.
```

Exactly one of `builtin`, `command`, or `go_module` is allowed under the
selected `frame` or `shell` object. `frame` and `shell` are mutually exclusive;
configuring both fails validation. Frame `values` are validated against the
selected `FrameSchema.Options` descriptors; shell-specific `values` remain
opaque to `/ssg` and are validated by the selected shell. Unknown keys at the
top-level site YAML still fail config validation.

A frame is a structural primitive; a shell is a page compositor, not a second
SSG. Code generation exists only to bridge Go module distribution; it does not
move routing or content ownership out of `/ssg`.

## 6. Declarative site configuration

The normal `/site` path is intentionally declarative:

```yaml
version: 1

source: docs
output: dist
assets: local
offline: true
base_path: /

site:
  name: Margo
  description: Documentation for Margo
  base_url: https://margo.araihu.com
  home: index.md
  logo: assets/logo.svg
  icon: assets/favicon.svg
  social_image:
    path: assets/social/margo-docs-v1.png
    alt: Margo documentation preview

frame:
  builtin: top-left-main-footer

locales:
  default: en
  supported:
    - en
    - pt-BR

navigation:
  mode: file-tree
  exclude:
    - drafts/**
    - internal/**

bindings:
  navigation:
    area: left-nav
  breadcrumbs:
    area: top-nav
  pagination:
    area: main-content
    slot: after-article
  theme_controls:
    area: top-nav
  footer:
    area: footer

themes:
  - name: my_custom_theme
    css_url: themes/my-custom.css
    token_catalog: themes/my-custom.tokens.yaml

custom_css:
  - css_url: themes/site-overrides.css
  - css_url: themes/docs-tweaks.css

theme:
  builtin: true
  name: modern
  allow_switch_theme: true
  color_mode: system
```

Rules:

- `logo` and `icon` are required site identity inputs;
- `offline` defaults to `true`; the default HTML/PDF profile must not emit live
  network dependencies;
- all relative paths (`source`, `output`, assets, frame/shell command, CSS, and
  token catalogs) resolve
  from the directory containing `site.yaml`; only an absolute `http` or `https`
  URL is an exception;
- relative assets are copied and rewritten; an external CSS URL is fetched and
  vendored during the build, rewritten to a local immutable asset, and recorded
  with its integrity digest. A fetch failure is a build error. An active remote
  CSS URL is permitted only in an explicitly non-offline profile and is never
  used by the default HTML/PDF path;
- in offline mode, resource closure is transitive: `/ssg` resolves and
  vendorizes every `@import`, CSS `url(...)` (including fonts and images),
  module/script dependency, frame/shell resource, and token-catalog asset;
  rewrites each reference to a local digest-addressed asset, records original
  URL/local path/digest/dependencies in the manifest, and rejects any unresolved
  remote URL or programmatic runtime network call;
- offline output emits a restrictive policy manifest and CSP equivalent to
  `default-src 'self'; connect-src 'none'; img-src 'self' data:; font-src
  'self'; style-src 'self' 'sha256-…'; script-src 'self' 'sha256-…'` (with
  hashes/nonces for the declared inline bootstrap). Programmatic `fetch`,
  WebSocket, EventSource, and runtime `import()` are rejected in the offline
  profile; all resources are loaded by ordinary same-origin document requests
  from the declared closure. The build's browser journey allows only those
  same-origin closure requests and fails on any external or undeclared request.
- `base_url` is an origin (`https://host`), not a path-bearing URL;
- `base_path` is `/` by default, is normalized without a trailing slash except
  for root `/`, and is applied to every public route, asset URL, canonical URL,
  and layout link without duplication;
- `frame` and `shell` are mutually exclusive layout selectors; omitting both
  selects the builtin `top-left-main-footer` frame for `/site` documentation,
  while selecting a shell never also selects a top-level frame; generic `/ssg`
  callers must select their layout explicitly;
- the selected frame/shell is the only layout authority for area placement;
  `/site` passes navigation and page data but does not maintain a second layout
  tree;
- `bindings` maps semantic provider kinds to area IDs; provider props are
  validated by `/site` and resolved before the frame boundary;
- `custom_css` loads after the selected theme, in declaration order; vendored
  CSS is local and deterministic before first paint and print;
- unknown YAML keys fail validation rather than being silently ignored;
- config version is independent from the Margo package version;
- the config and selected frame/shell identity participate in deterministic build
  provenance.

## 7. Locale convention

The engine should model locales now, even when the first site has one language.
Recommended source convention:

```text
docs/article/foo.md          # default locale
docs/pt-br/article/foo.md   # translated route
```

Locale tags use canonical BCP 47 spelling (`en`, `pt-BR`). The locale segment is
the first source-tree segment and is compared case-insensitively; the output
directory key is normalized to lowercase (`pt-br`). The default locale remains
unprefixed; alternate locales receive a locale-prefixed route. A language
switcher is rendered only when a matching logical route exists in another
locale. A first source segment that collides with a supported locale but is
intended as ordinary content is a build error, not an implicit fallback.

Locale metadata, canonical URLs, alternate links, and route navigation must agree.
The engine should reject a frontmatter language that conflicts with the path
locale. The first active Margo site may set `supported: [en]`; multilingual
output is enabled for frames without layout labels. When an opinionated shell
is selected, it must supply labels for every configured locale. Translation
fallback and the exact pretty-URL mapping remain open design decisions for the
next section of the spec.

## 8. Navigation contract

`navigation.mode: file-tree` is deterministic and needs no hand-maintained
sidebar. `exclude` removes matching source files and their nodes from both the
build and navigation; it does not merely hide them.

- `index.md` is the landing page for its directory and does not create a second
  child node;
- directories with descendants but no `index.md` are structural, non-clickable
  group nodes;
- page labels resolve as frontmatter `title`, then the page's document `h1`; a
  public page with no usable label fails the build. Structural group labels use
  a humanized directory name; a future directory-metadata file may override it;
- every public Markdown source must contain exactly one document-level `h1`;
  `/ssg` fails with a source diagnostic when it is missing or duplicated rather
  than synthesizing or letting the shell rewrite the article semantics;
- siblings sort by normalized output route after Unicode NFC normalization and
  bytewise UTF-8 comparison, with a directory landing page before its
  descendants;
- node IDs derive from normalized logical routes and are stable across builds;
- the selected layout or semantic navigation payload owns markup and
  `aria-current`; it receives enough route state to mark the active page,
  breadcrumbs, and previous/next links consistently.

## 9. Theme contract

`themes` is the custom theme catalog. Each entry has a unique `name`, a
`css_url`, and, for custom themes, a structured `token_catalog` sidecar. The
catalog is coupled to the vendored CSS digest; CSS alone cannot claim the Margo
theme contract.

`theme` selects behavior:

- `builtin: true` includes the Goshtoso/Margo builtin catalog;
- `builtin: false` disables builtin choices except for mandatory `modern` fallback;
- `name` selects the initial theme and defaults to `modern`;
- `allow_switch_theme` controls whether the active layout exposes a theme
  selector: `/site` supplies it as a semantic binding in frame mode, while the
  shell owns its component in shell mode; it defaults to `false`;
- `color_mode` is exactly `system`, `light`, or `dark`, and defaults to `system`;
- custom and builtin names share one namespace; duplicate names fail validation
  rather than silently shadowing a theme.

An unknown or unavailable configured `theme.name` is a build-time configuration
error. A stale value found only in the user's persisted `margo.theme` key is a
runtime input and falls back to the validated configured theme without failing
the document build.

Every custom token catalog publishes the semantic typography roles `display`,
`headline`, `title`, `body`, `label`, and `caption` (font family and loading
policy, size, weight, line-height, letter spacing where applicable, and
responsive behavior),
layout/component spacing tokens,
named content breakpoints and grid tracks, reading measure, logical drawer size,
semantic light/dark color tokens (`accent`, `surface`, `surface-alt`, `text`,
`text-strong`, `outline`, and `focus-ring`), interactive states (default, hover, pressed,
focus, disabled, selected), feedback states, and contrast pairs. It also
publishes the minimum text size and the 44px project touch-target token. `/site`
validates references from `FrameSchema.Layout` and frame options against this
catalog before hashing; missing tokens, unresolved fonts, or a catalog digest
that does not match its CSS fail the build. Builtin themes ship the same catalog
in the Margo distribution.

The contrast catalog must include every required role pair in both color modes:
body text and strong text on the primary and alternate surfaces, accent text on
the primary and alternate surfaces, and outline/non-text UI on the primary and
alternate surfaces. Focus rings must meet 3:1 against the page background, the
focus-state background, and the alternate surface; text pairs meet 4.5:1; other
non-text pairs meet 3:1. The semantic
`caption` role is an explicit alias of `label`, matching the approved
`figure-caption` and `data-table` foundation; it inherits the label's font, 500
weight, size, and line-height. The semantic `active` state is an explicit alias
of `selected` for this
documentation profile: navigation items marked `aria-current` use
`token://states.<mode>.selected` and do not introduce a second visual token.

The sidecar is a deterministic data document, not an opaque CSS convention. Its
versioned minimum shape is:

```yaml
schema: margo.theme.tokens/v1
css_digest: sha256-...
fonts:
  - {name: body, source: assets/fonts/body.woff2, weights: [400, 500, 700], display: swap}
typography:
  display: {font_family: "...", size: "...", weight: 700, line_height: "...", letter_spacing: "...", responsive: {mode: fixed}}
  headline: {font_family: "...", size: "...", weight: 700, line_height: "...", letter_spacing: "...", responsive: {mode: fixed}}
  title: {font_family: "...", size: "...", weight: 700, line_height: "...", letter_spacing: "...", responsive: {mode: fixed}}
  body:
    font_family: "..."
    size: "..."
    line_height: "..."
    weight: 400
    responsive: {mode: fixed, narrow: {size: "...", line_height: "..."}, mid: {size: "...", line_height: "..."}, wide: {size: "...", line_height: "..."}}
  label: {font_family: "...", size: "...", weight: 500, line_height: "...", responsive: {mode: fixed}}
  caption: {alias_of: label}
minimum_text_size: "1rem"
touch_target: {min_css_px: 44}
reading_measure: {token: "token://layout.tokens.main-measure", min_ch: 65, max_ch: 75}
layout:
  tokens:
    main-measure: {type: length, value: "75ch", min: "65ch", max: "75ch"}
    main-measure-mid: {type: length, value: "68ch", min: "65ch", max: "75ch"}
    main-three-column-measure: {type: length, value: "65ch", min: "65ch", max: "75ch"}
    spacing-base: {type: length, value: "0.25rem", min: "0.25rem", max: "0.5rem"}
    spacing-prose: {type: length, value: "1rem", min: "0.75rem", max: "2rem"}
    spacing-section: {type: length, value: "2.5rem", min: "1.5rem", max: "6rem"}
    gap: {type: length, value: "1.5rem", min: "0.5rem", max: "3rem"}
    component-padding: {type: length, value: "1rem", min: "0.5rem", max: "2rem"}
    sidebar-inline-size-wide: {type: length, value: "16rem", min: "12rem", max: "20rem"}
    sidebar-inline-size-mid: {type: length, value: "14rem", min: "12rem", max: "18rem"}
    drawer-inline-size: {type: length, value: "18rem", min: "16rem", max: "22rem"}
spacing:
  scale: ["0.25rem", "0.5rem", "0.75rem", "1rem", "1.5rem", "2rem"]
  semantic:
    base: "token://layout.tokens.spacing-base"
    prose: "token://layout.tokens.spacing-prose"
    section: "token://layout.tokens.spacing-section"
    layout-section-gap: "token://layout.tokens.gap"
    component-padding: "token://layout.tokens.component-padding"
breakpoints:
  - {name: narrow, min_css_px: 0, max_css_px: 720}
  - {name: mid, min_css_px: 720, max_css_px: 1100}
  - {name: wide, min_css_px: 1100}
grid:
  track_tokens:
    main-wide: {type: track, function: minmax, min: 0, max: "token://layout.tokens.main-measure"}
    main-mid: {type: track, function: minmax, min: 0, max: "token://layout.tokens.main-measure-mid"}
    main-three-column-mid: {type: track, function: minmax, min: 0, max: "token://layout.tokens.main-three-column-measure"}
    narrow-content: {type: track, function: minmax, min: 0, max: "token://layout.tokens.main-measure"}
    fluid: {type: track, function: minmax, min: 0, max: "1fr"}
  reference_syntax: token://grid.track_tokens/<name>
drawer: {inline_size: "token://layout.tokens.drawer-inline-size", max_inline_size: "min(22rem, 85vi)"}
colors:
  light: {accent: "...", surface: "...", surface-alt: "...", text: "...", text-strong: "...", outline: "...", focus-ring: "..."}
  dark: {accent: "...", surface: "...", surface-alt: "...", text: "...", text-strong: "...", outline: "...", focus-ring: "..."}
states:
  light:
    default: {background: "...", foreground: "..."}
    hover: {background: "...", foreground: "..."}
    pressed: {background: "...", foreground: "..."}
    focus: {background: "...", foreground: "..."}
    disabled: {background: "...", foreground: "..."}
    selected: {background: "...", foreground: "..."}
  dark:
    default: {background: "...", foreground: "..."}
    hover: {background: "...", foreground: "..."}
    pressed: {background: "...", foreground: "..."}
    focus: {background: "...", foreground: "..."}
    disabled: {background: "...", foreground: "..."}
    selected: {background: "...", foreground: "..."}
feedback:
  light:
    success: {foreground: "...", background: "...", cue: "icon-or-text"}
    warning: {foreground: "...", background: "...", cue: "icon-or-text"}
    error: {foreground: "...", background: "...", cue: "icon-or-text"}
    info: {foreground: "...", background: "...", cue: "icon-or-text"}
  dark:
    success: {foreground: "...", background: "...", cue: "icon-or-text"}
    warning: {foreground: "...", background: "...", cue: "icon-or-text"}
    error: {foreground: "...", background: "...", cue: "icon-or-text"}
    info: {foreground: "...", background: "...", cue: "icon-or-text"}
contrast_pairs:
  required_modes: [light, dark]
  required_states: [default, hover, pressed, focus, disabled, selected]
  required_feedback: [success, warning, error, info]
  required_non_text: [focus-ring]
  required_focus_surfaces: [page-background, focus-state-background, alternate-background]
  required_role_pairs: [text/surface, text-strong/surface, text/surface-alt, text-strong/surface-alt, accent/surface, accent/surface-alt, outline/surface, outline/surface-alt]
  entries:
    - {mode: light, state: default, foreground: "token://states.light.default.foreground", background: "token://states.light.default.background", ratio: 4.5}
    - {mode: light, state: hover, foreground: "token://states.light.hover.foreground", background: "token://states.light.hover.background", ratio: 4.5}
    - {mode: light, state: pressed, foreground: "token://states.light.pressed.foreground", background: "token://states.light.pressed.background", ratio: 4.5}
    - {mode: light, state: focus, foreground: "token://states.light.focus.foreground", background: "token://states.light.focus.background", ratio: 4.5}
    - {mode: light, state: disabled, foreground: "token://states.light.disabled.foreground", background: "token://states.light.disabled.background", ratio: 4.5}
    - {mode: light, state: selected, foreground: "token://states.light.selected.foreground", background: "token://states.light.selected.background", ratio: 4.5}
    - {mode: dark, state: default, foreground: "token://states.dark.default.foreground", background: "token://states.dark.default.background", ratio: 4.5}
    - {mode: dark, state: hover, foreground: "token://states.dark.hover.foreground", background: "token://states.dark.hover.background", ratio: 4.5}
    - {mode: dark, state: pressed, foreground: "token://states.dark.pressed.foreground", background: "token://states.dark.pressed.background", ratio: 4.5}
    - {mode: dark, state: focus, foreground: "token://states.dark.focus.foreground", background: "token://states.dark.focus.background", ratio: 4.5}
    - {mode: dark, state: disabled, foreground: "token://states.dark.disabled.foreground", background: "token://states.dark.disabled.background", ratio: 4.5}
    - {mode: dark, state: selected, foreground: "token://states.dark.selected.foreground", background: "token://states.dark.selected.background", ratio: 4.5}
    - {mode: light, feedback: success, foreground: "token://feedback.light.success.foreground", background: "token://feedback.light.success.background", ratio: 4.5}
    - {mode: light, feedback: warning, foreground: "token://feedback.light.warning.foreground", background: "token://feedback.light.warning.background", ratio: 4.5}
    - {mode: light, feedback: error, foreground: "token://feedback.light.error.foreground", background: "token://feedback.light.error.background", ratio: 4.5}
    - {mode: light, feedback: info, foreground: "token://feedback.light.info.foreground", background: "token://feedback.light.info.background", ratio: 4.5}
    - {mode: dark, feedback: success, foreground: "token://feedback.dark.success.foreground", background: "token://feedback.dark.success.background", ratio: 4.5}
    - {mode: dark, feedback: warning, foreground: "token://feedback.dark.warning.foreground", background: "token://feedback.dark.warning.background", ratio: 4.5}
    - {mode: dark, feedback: error, foreground: "token://feedback.dark.error.foreground", background: "token://feedback.dark.error.background", ratio: 4.5}
    - {mode: dark, feedback: info, foreground: "token://feedback.dark.info.foreground", background: "token://feedback.dark.info.background", ratio: 4.5}
    - {mode: light, non_text: focus-ring, surface: page-background, foreground: "token://colors.light.focus-ring", background: "token://colors.light.surface", ratio: 3.0}
    - {mode: light, non_text: focus-ring, surface: focus-state-background, foreground: "token://colors.light.focus-ring", background: "token://states.light.focus.background", ratio: 3.0}
    - {mode: light, non_text: focus-ring, surface: alternate-background, foreground: "token://colors.light.focus-ring", background: "token://colors.light.surface-alt", ratio: 3.0}
    - {mode: dark, non_text: focus-ring, surface: page-background, foreground: "token://colors.dark.focus-ring", background: "token://colors.dark.surface", ratio: 3.0}
    - {mode: dark, non_text: focus-ring, surface: focus-state-background, foreground: "token://colors.dark.focus-ring", background: "token://states.dark.focus.background", ratio: 3.0}
    - {mode: dark, non_text: focus-ring, surface: alternate-background, foreground: "token://colors.dark.focus-ring", background: "token://colors.dark.surface-alt", ratio: 3.0}
    - {mode: light, role: text/surface, foreground: "token://colors.light.text", background: "token://colors.light.surface", ratio: 4.5}
    - {mode: light, role: text-strong/surface, foreground: "token://colors.light.text-strong", background: "token://colors.light.surface", ratio: 4.5}
    - {mode: light, role: text/surface-alt, foreground: "token://colors.light.text", background: "token://colors.light.surface-alt", ratio: 4.5}
    - {mode: light, role: text-strong/surface-alt, foreground: "token://colors.light.text-strong", background: "token://colors.light.surface-alt", ratio: 4.5}
    - {mode: light, role: accent/surface, foreground: "token://colors.light.accent", background: "token://colors.light.surface", ratio: 4.5}
    - {mode: light, role: accent/surface-alt, foreground: "token://colors.light.accent", background: "token://colors.light.surface-alt", ratio: 4.5}
    - {mode: light, role: outline/surface, foreground: "token://colors.light.outline", background: "token://colors.light.surface", ratio: 3.0}
    - {mode: light, role: outline/surface-alt, foreground: "token://colors.light.outline", background: "token://colors.light.surface-alt", ratio: 3.0}
    - {mode: dark, role: text/surface, foreground: "token://colors.dark.text", background: "token://colors.dark.surface", ratio: 4.5}
    - {mode: dark, role: text-strong/surface, foreground: "token://colors.dark.text-strong", background: "token://colors.dark.surface", ratio: 4.5}
    - {mode: dark, role: text/surface-alt, foreground: "token://colors.dark.text", background: "token://colors.dark.surface-alt", ratio: 4.5}
    - {mode: dark, role: text-strong/surface-alt, foreground: "token://colors.dark.text-strong", background: "token://colors.dark.surface-alt", ratio: 4.5}
    - {mode: dark, role: accent/surface, foreground: "token://colors.dark.accent", background: "token://colors.dark.surface", ratio: 4.5}
    - {mode: dark, role: accent/surface-alt, foreground: "token://colors.dark.accent", background: "token://colors.dark.surface-alt", ratio: 4.5}
    - {mode: dark, role: outline/surface, foreground: "token://colors.dark.outline", background: "token://colors.dark.surface", ratio: 3.0}
    - {mode: dark, role: outline/surface-alt, foreground: "token://colors.dark.outline", background: "token://colors.dark.surface-alt", ratio: 3.0}
```

Additional semantic tokens may be added, but required names and value types are
versioned. `schema` selects the catalog validator. The catalog's `css_digest`
must match the normalized vendored CSS and its transitive style closure;
breakpoints/grid, drawer bounds, font sources, and token values are canonicalized
with sorted object keys and normalized CSS lengths before hashing. The catalog,
CSS, fonts, and referenced assets are one hashed theme identity.
Frame schema fields use `token://...` references; a raw CSS length is accepted
only through an explicit `css://...` value with the descriptor's declared range.
`/site` resolves every frame token reference against `layout.tokens` and
`grid.track_tokens` before schema hashing, while the frame alone supplies the
track arrangement and area spans. A custom theme may change bounded token
values but cannot silently change the number or semantic identity of a frame's
tracks. `reading_measure`,
`spacing.semantic`, and `drawer.inline_size` are aliases to those canonical
layout tokens, not independent values; the validator rejects a sidecar whose
alias bounds or resolved value disagree. A `type: track` entry has a typed
function (`minmax`), numeric minimum, and token/track maximum; it is not an
arbitrary CSS string. The resolved modern tracks therefore remain
`minmax(0, 75ch)`, `minmax(0, 68ch)`, `minmax(0, 65ch)`, or
`minmax(0, 1fr)` according to the frame-owned mapping.

Theme switching changes theme/color values, never document hierarchy or semantics.

The first paint is deterministic and does not depend on a remote stylesheet or
external client JavaScript. `/site` emits a small local (inline or vendored)
bootstrap before theme CSS. Its precedence is fixed:

1. the theme name is `margo.theme` only when `allow_switch_theme` is true and
   the stored value is in the validated catalog; otherwise it is `theme.name`;
2. `color_mode: light` or `dark` always wins over the OS preference;
3. only `color_mode: system` consults `prefers-color-scheme`.

Unknown or stale stored values and storage failures fall back to the validated
configured theme. A custom theme absent from the validated catalog is a build
error when configured in YAML; it is only a runtime fallback when it came from
stale persisted storage. The bootstrap sets a `data-theme`/color-mode
attribute before the first theme rule is applied, persists changes only when
`allow_switch_theme` is true, and has a no-JavaScript fallback to the configured
default. A system-preference listener may update the attribute only while
`color_mode: system`; it never changes document semantics and is not required
for the first paint.
The bootstrap, selected theme CSS, and custom CSS are all local for the default
offline HTML/PDF profile and their digests are recorded in the manifest.

## 10. Metadata and social contract

Every public route must emit metadata in initial generated HTML, without client
JavaScript. Route frontmatter must override site defaults where supplied; when
it does not, deterministic route-derived values are required so that one
generic homepage description is not copied to every page.

- route-specific title;
- route-specific description;
- canonical URL;
- `og:url`, type, title, description, site name, image, dimensions, MIME type,
  and alt text;
- explicit X/Twitter card, title, description, image, and alt text;
- locale and alternate-locale metadata when multilingual output is enabled.

Page frontmatter supplies route-specific values. Site YAML supplies identity and
safe defaults. `social_image` is a structured path/alt input; `/ssg` derives and
validates MIME type and dimensions from the copied file, then emits an absolute
HTTPS URL under `base_url` and `base_path`. The selected frame or shell cannot
write the document `<head>`; `/ssg` renders the shared head primitive exactly
once and rejects duplicate or missing required tags before emitting an artifact.
`/ssg` also owns the outer document language and direction: every artifact emits
`<html lang="<canonical BCP 47>" dir="<ltr|rtl>">`. Frames and shells inherit
this resolved direction and must not infer it independently.

## 11. Documentation-shell and PDF baseline

The first `goshtoso-docs` shell must provide a responsive header, file-tree
sidebar, breadcrumbs, previous/next links, and conditional
theme/language controls. Search and editorial landing-page composition are
optional and remain outside this slice. UI labels must come from the
`ShellSchema.Labels` catalog; the English-only benchmark is valid until another
catalog is supplied. The shell may choose any visual landmarks, subject to the
documentation profile's required `document` area and the acceptance invariants
for one `main`, one `h1`, and the single `/ssg`-owned skip link. The shell must
preserve and position that primitive, never emit a second one or replace its
target ID.

In frame mode, `/site` binding providers supply the file-tree navigation,
breadcrumbs, previous/next links, and theme/language controls as semantic
fragments. Previous/next is the normative `pagination` provider and is resolved
into the builtin `main-content:after-article` slot, so its `<nav
aria-label="label.article_navigation">` (localized by `/site`, English fallback
`Article navigation`) remains inside the document's single
`<main>`. A custom frame may expose another declared pagination slot, but it may
not place article navigation outside the document host without an explicit
profile change. The selected root frame must expose a `navigation`-accepting area;
frame hooks may enhance that fragment into a mobile drawer, but the static
navigation remains available without JavaScript. The `/site` navigation provider
owns the `<nav>` landmark, its localized accessible name, the active item, and
the labeled drawer trigger/close controls; the frame owns only placement,
qualified targets, responsive collapse, and event primitives. Every navigation
landmark on the page (file tree, breadcrumbs, TOC, and article navigation) has a
distinct localized accessible name. The drawer contract includes a logical
`inline-start` placement, a published `drawer-inline-size` token with a
`max-inline-size: min(22rem, 85vi)` bound, and a labeled trigger with
`aria-expanded`/`aria-controls`, an initial focus target, visible focus, Escape,
close-button and overlay dismissal, focus restoration, background `inert`/scroll
containment when modal, reduced-motion behavior, safe-area padding, and an
overlay that does not obscure the reading path. A non-modal drawer must state
that choice explicitly and retain an equivalent keyboard path. RTL fixtures must
verify that the drawer follows `inline-start` rather than a physical left edge.

The semantic Margo `document` payload is shared by HTML and PDF generation. PDF
consumes that payload directly and does not depend on optional frame/shell areas
or a layout fragment. In the HTML site, layout chrome may be hidden by print
CSS, but it must not rewrite document semantics. Existing PDF visual/readability
gates remain green while this site slice evolves. `/ssg` records the canonical
document payload digest in the manifest; HTML and PDF projections must report
that same digest before rendering. `/ssg` also emits a
`document_style_digest` that covers one canonical shared content-style set: the
post-cascade rules that can reach the semantic article, typography and
theme-token projection, table/chart/caption rules, local font assets, and
`custom_css` (plus its transitively vendored imports/assets) classified as
content-affecting. The digest is keyed by the selected theme name and effective
color mode. Rules classified as HTML-only chrome or PDF-only projection are
excluded from this shared set and receive separate `html_projection_style_digest`
or `pdf_projection_style_digest` values. HTML and PDF must report the same
`document_style_digest`; each may additionally report its projection digest, but
neither may substitute a different content style. An ambiguous custom selector
is content-affecting and therefore enters the shared digest.
Print fixtures compare headings, captions, tables, charts, wide-content
overflow, style digest, and sticky-disabled output against the HTML source
identity.

## 12. Acceptance gates

### 12.1 Checklist-backed design and accessibility requirements

The following requirements are traced to [Checklist Design: Accessibility](https://www.checklist.design/design-system/accessibility),
[Spacing / Grid](https://www.checklist.design/design-system/spacing-and-grid),
[Typography](https://www.checklist.design/design-system/typography),
[Color System](https://www.checklist.design/design-system/color-system), and
[Drawer](https://www.checklist.design/design-system/drawer):

- target WCAG AA; verify 4.5:1 contrast for normal text and 3:1 for large text
  and UI components;
- provide a visible high-contrast focus indicator, documented keyboard patterns,
  ARIA roles/states, and screen-reader verification for navigation and controls;
- use named content breakpoints, semantic spacing tokens, responsive type, a
  readable minimum size, and usable behavior at 200% browser zoom;
- project requirement: keep primary controls and drawer triggers at least 44px
  by 44px (or provide an equivalent adjacent target), with the size token
  published by the theme;
- when a mobile navigation binding becomes a drawer, define edge, width,
  overlay, header/close action, scrollable content, Escape dismissal, and focus
  restoration; static navigation remains usable without JavaScript;
- frame options must preserve reading measure, anchor visibility, local overflow,
  and source order across narrow, wide, dark, reduced-motion, and print states.

The documentation slice is ready for implementation review only when a fixture
site proves all of the following:

- the Margo site is generated from YAML, Markdown, and declared assets, with no
  hand-maintained sidebar;
- frame-mode fixtures use `/site` semantic binding providers for navigation and
  controls, while the frame remains component-free; the selected root exposes
  one navigation-capable area;
- homepage and at least one nested route build under both `/` and a non-root
  `base_path`;
- repeated builds are byte-deterministic and record config/frame-or-shell identity
  in the
  manifest;
- nested frame composition rejects cycles and duplicate mount IDs, namespaces
  event targets, rejects child mounts into undeclared/occupied areas, keeps
  `ChildrenByMount` separate from semantic bindings, rejects child document
  roles, and produces exactly one document host and skip link;
- root and child fixtures may use the same area IDs but must resolve distinct
  `(CompositionPath, AreaID, Slot)` keys; wrong-kind providers, slot violations,
  `MaxBindings`/`MaxBindingsByKind` overflow, and non-deterministic
  `BindingOrder` fail identically in builtin, command, and Go-module modes;
- omitted structural frame values normalize to declared defaults before hashing;
  invalid values fail with a fully qualified property path;
- content and links work with JavaScript disabled;
- generated pages contain exactly one `h1`, one `main`, exactly one `/ssg`-owned
  skip link, and an `aria-current` active navigation item; the bound `document`
  payload digest
  validates exactly once before its marker is removed; mobile navigation
  and every file-tree, breadcrumb, TOC, article-navigation, and drawer
  landmark has a distinct localized accessible name; mobile navigation
  has an accessible name, `aria-expanded`/`aria-controls`, visible focus,
  overlay/button/Escape dismissal, focus restoration, background isolation when
  modal, and a no-JavaScript fallback;
- 390px and 1440px viewports pass with builtin theme, custom theme, light/dark
  modes, and declared custom CSS; every published `narrow`/`mid`/`wide`
  threshold is also exercised just below, exactly at, and just above the
  threshold, including `prefers-reduced-motion: reduce`;
- an RTL locale passes with `/ssg`-resolved `Direction: rtl`, logical drawer and
  sticky offsets, `<html lang>`/`dir` correctness, distinct landmark names, and
  no frame-local BCP 47 inference;
- first paint is checked with JavaScript disabled, no stored theme, an allowlisted
  stored theme, an unknown stored theme, fixed `light`/`dark` modes, system
  light/dark preference, storage failure, and two distinct unavailable-theme
  cases: a configured unavailable theme must fail the build, while a stored
  unavailable theme falls back to the validated configured theme; OS preference
  is consulted only for `color_mode: system`;
- named content breakpoints, 200% zoom, sticky-area offsets, local overflow, and
  print disabling of sticky behavior preserve reading order and anchor visibility;
  fixtures cover top+bottom sticky together, nested sticky frames, long
  navigation, short viewports, safe-area insets, and the automatic lower-priority
  fallback;
- required title/description/canonical/Open Graph/X metadata appears exactly
  once, with a published preview image whose MIME, dimensions, alt, and HTTPS
  URL validate;
- internal links, local assets, locale alternates, and excluded paths validate;
- pagination renders through the declared `main-content:after-article` slot and
  remains inside the document `<main>`; a frame without that slot rejects the
  docs profile unless it publishes an equivalent slot;
- custom themes validate their token-catalog schema and CSS digest, including
  semantic text/spacing/color/state tokens, light/dark pairs, font assets,
  breakpoint tracks, and contrast pairs for all supported states;
- selected, active, error, warning, and focus states retain a text, icon,
  outline, pattern, or other non-color cue; `active` uses the normative
  `selected` token alias; a color-blindness fixture verifies
  that meaning is not conveyed by color alone;
- an offline build resolves the complete resource closure, rewrites all
  `@import`, `url(...)`, font, image, module, frame, shell, and token-catalog
  dependencies locally, records their original URL/local digest/dependency
  edges, emits `connect-src 'none'`/same-origin CSP policy, rejects unresolved
  dynamic imports and network-capable runtime paths, and makes zero network
  requests outside the declared same-origin closure during the browser journey;
- builtin, command, and Go-module frame modes receive equivalent frame-only
  requests; shell-only route/brand/navigation context never crosses the frame
  protocol boundary;
- request swaps default to normalized `innerHTML`; runtime response overrides
  accept only values in the target area's normalized `AllowedSwaps` subset of
  the six declared HTMX-compatible values and reject globally valid but
  area-disallowed `HX-Reswap` values; boundary restrictions, target preservation, focus
  restoration, and `aria-live` policy are verified for every allowed swap;
  `beforebegin`/`afterend` are always rejected for live areas;
- the HTML and PDF projections report the same canonical document payload digest
  and `document_style_digest`; optional HTML/PDF projection style digests may
  differ only for classified chrome/interactive rules. The same semantic article
  and content styles remain printable and the existing PDF gates stay green;
- the typed grid descriptors pass below/exactly/above every breakpoint with
  one-based exclusive-end spans, no overlaps/holes beyond declared tracks, and
  DOM/source order equal to keyboard and assistive-technology order;

The accessibility gate also requires an axe-equivalent scan with zero
unwaived A/AA violations (regardless of an internal priority label), WCAG AA
contrast for every text, UI, focus, and interactive-state pair in every light/
dark theme, visible keyboard focus, a clean browser console, and usable
primary/sidebar links at 390px with JavaScript disabled. Any waiver must name
the criterion, rationale, affected surface, compensating behavior, and manual
reviewer. It includes VoiceOver on Safari and NVDA on Chrome for
navigation, drawer states, live updates, and swap focus behavior. JavaScript
may enhance the mobile drawer, but the static navigation remains reachable
without it.

## 13. Current boundary

This draft defines the use-case split, `/ssg` versus `/site` ownership, declarative
site shape, locale direction, theme direction, frame/shell data contract, and
frame/shell distribution modes.

It does not yet define:

- exact Go interfaces and package paths;
- the frame/shell command wire format and asset transfer format;
- pretty URL rules;
- translation fallback policy;
- shell module trust/signature policy;
- implementation or migration steps.

Those decisions require approval of this boundary first.
