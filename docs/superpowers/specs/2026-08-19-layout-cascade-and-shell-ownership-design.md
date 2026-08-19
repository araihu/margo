# Layout Cascade and Shell Ownership Design

## Status

Approved in chat on 2026-08-19. This design replaces the layout-profile and
source-prefix family model added earlier on `docs/expansion`.

## Goal

Keep Margo's no-config directory build as a plain Markdown-to-HTML tree
projection. Make every shell, navigation surface, and layout-specific asset an
explicit consequence of a configured layout. Isolate the Tour landing page
from the Module and CLI documentation shell.

## Non-goals

- New visual features.
- External or user-defined layout implementations.
- Push, merge, release, deployment, or publication.
- Changes to Markdown document semantics unrelated to layout selection.

## Plain no-config projection

`site.Build` remains the only no-config directory path. Source discovery,
source-to-output mapping, link and image rewriting, deterministic ordering,
asset modes, and manifest generation remain independent from configured
layouts.

The generated page is a complete HTML document containing the semantic
Markdown projection. It does not call the layout registry and does not emit a
frame, shell, navigation, sidebar, table of contents, breadcrumbs, pagination,
page-action toolbar, configured site stylesheet, or Goshtoso layout runtime.
Existing document feature artifacts may remain only when the semantic document
itself requires them; configured layout dependencies never enter this path.

Regression tests inspect both markup and artifact names so a configured shell
cannot leak into the no-config path.

## Configuration model

Configured sites expose one site layout selection:

```yaml
layout:
  kind: docs
  default:
    families: [default, module, cli]
    sidebar: true
    toc: true
    content:
      layout: article
  values:
    family: default
```

`layout.default` declares site defaults for the selected kind.
`layout.values` is the site-level override patch. Both are validated against
the selected kind's closed values schema. The effective site values are the
built-in kind defaults, then `default`, then `values`.

Supported first-version kinds:

- `article`: one-column semantic article, without navigation chrome.
- `landing`: one-column landing composition, without documentation chrome.
- `docs`: documentation shell with optional sidebar and table of contents,
  family navigation, and article content.

Each registry entry owns:

- its kind name;
- immutable built-in defaults;
- a closed schema for values;
- validation and normalization;
- rendering and dependency staging.

Registry tests validate built-in defaults through the same validator used for
site defaults, directory patches, and Markdown patches. Unknown kinds and
unknown values fail before artifacts are emitted.

## Typed cascade

Resolution order is:

```text
site defaults
  -> directory patches from source root to nearest directory
    -> Markdown frontmatter patch
```

Within one kind, maps merge recursively, scalars replace, and arrays replace
completely. Patches are applied in path order, never map iteration order.

Kind changes create a typed boundary. The new kind starts from its own
built-in defaults; values inherited from another kind do not cross that
boundary. Later value-only patches apply to the currently selected kind. If a
later patch selects a previously used kind, that kind's accumulated values are
available again. This makes a root page containing only
`layout.kind: landing` valid even when the site default is `docs`.

`default` is site-only. Directory and Markdown patches may contain only
`kind` and `values`.

## Directory patches

The reserved filename is `_layout.yaml`. The configured-source walk discovers
these files alongside Markdown inputs but never publishes them as content.
Each patch is keyed by its normalized directory. A page receives patches for
the source root and each ancestor directory, ending with its own directory.

The loader rejects symlinks, duplicate YAML keys, multiple documents,
non-mapping roots, unknown patch properties, invalid kinds, and invalid values.
Diagnostics identify the `_layout.yaml` source plus a YAML/JSON pointer. Patch
discovery and application use sorted normalized paths.

## Markdown patches

Top-level frontmatter may contain:

```yaml
layout:
  kind: landing
```

or:

```yaml
layout:
  values:
    family: module
```

The existing frontmatter parser already preserves consumer-owned root values
in `Metadata.Additional`. The site layer reads `Additional["layout"]`, validates
the closed patch shape, and attaches source path plus `/layout/...` pointer to
failures. Layout data remains a site concern; generic Markdown rendering does
not interpret it.

The old `margo.site.layout` profile selector is removed from showcase usage and
is not part of the new cascade.

## Family ownership

Family is not added to `Page` as a universal presentation requirement. It is
resolved only when the active layout kind is `docs` and recorded in configured
route identity only for docs pages.

Docs values define an ordered `families` array and one selected `family`.
Rules:

- `default` always exists, even when omitted from the configured array.
- Non-default selections must appear in the central ordered array.
- Directory and Markdown patches may select a family but cannot declare one.
- Missing selection resolves to `default`.
- One effective family suppresses the secondary family navbar.
- Multiple families render in configured order.
- Sidebar contents and pagination are restricted to the current docs family.
- Landing and article validation reject family values rather than silently
  turning family into a generic page concept.

Family overview links are derived deterministically from docs pages: the first
route in each family, with an `index.md` route ordered first when present.
Selecting an undeclared family or declaring a family with no docs page produces
a source-level diagnostic before rendering.

## Rendering boundary

`landing` renders a dedicated one-column composition containing only the
Markdown-generated article and landing-owned styling. It emits no family
navbar, docs sidebar, table of contents, breadcrumbs, pagination, or page
actions.

`article` renders the same semantic article boundary without landing-specific
visual rules or documentation chrome.

`docs` alone owns site navigation, optional family navbar, optional sidebar,
optional table of contents, family-scoped pagination, search, repository link,
and docs-specific assets. Module and CLI select independent families through
their directory patches. Docs shell structure never determines Tour geometry.

The implementation removes profile-mode branching and CSS selector repairs
whose only purpose was forcing landing behavior through the shared docs shell.

## Showcase migration

- `showcase.yaml` selects `layout.kind: docs`, declares ordered families, and
  provides docs defaults.
- `showcase/content/index.md` selects `layout.kind: landing` in frontmatter.
- `showcase/content/module/_layout.yaml` selects family `module`.
- `showcase/content/cli/_layout.yaml` selects family `cli`.
- Tour declares no family.

Module and CLI keep their existing Markdown content and docs behavior. Tour
remains Markdown-generated and visually isolated.

## Diagnostics and determinism

Preflight resolves every page's patch chain, kind, values, family, schema, and
bindings before emitting HTML. Failures return no partial artifacts.

Diagnostics use stable codes for unknown kind, unknown value, invalid patch,
undeclared family, and empty family. Configuration diagnostics point to
`site.yaml`; directory diagnostics point to `_layout.yaml`; Markdown
diagnostics point to the Markdown source.

Manifest layout identity hashes the ordered registry identity, validated site
defaults, directory patch paths and normalized values, and effective per-page
layout identity. Equivalent inputs produce identical output regardless of
filesystem enumeration order.

## Testing and verification

TDD order:

1. Plain no-config isolation.
2. Registry defaults and closed-schema failures.
3. Deep merge and array replacement.
4. Root-to-nearest directory cascade.
5. Markdown precedence and kind boundaries.
6. Family declaration, selection, ordering, and failure cases.
7. Tour chrome absence and Module/CLI docs behavior.
8. Browser checks at mobile, tablet, and desktop widths.

After Go changes, stop and restart `margo serve`; browser checks must not reuse
a stale binary. Final gates:

```sh
GOWORK=off go test ./... -count=1
GOWORK=off go test -race ./site -count=1
GOWORK=off go vet ./site
GOWORK=off go mod verify
git diff --check
```

No new visual feature enters scope until these gates pass.
