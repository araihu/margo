# Unified layout model

Configured publications select one trusted layout kind in `site.yaml`:

```yaml
layout:
  kind: docs
  default:
    families: [module, cli]
    sidebar: true
    toc: true
    content:
      layout: article
  values:
    family: default
```

`layout.default` is available only in site configuration. It establishes the
site defaults for the selected kind. `layout.values` is the site-level override
patch. The supported kinds are `article`, `landing`, and `docs`; only `docs`
owns navigation, search, sidebar, table of contents, pagination, page actions,
and documentation families.

## Cascade

Each page resolves its layout in this order:

1. Site defaults and site values.
2. `_layout.yaml` patches from the source root through the nearest directory.
3. The page's top-level `layout` frontmatter patch.

Directory and Markdown patches accept only `kind` and `values`. Within one
kind, maps merge recursively, scalar values replace, and arrays replace
completely. A kind change starts from that kind's built-in defaults; values do
not leak across kind boundaries. Returning to a previously selected kind
restores that kind's accumulated values.

For example, `module/_layout.yaml` selects a docs family:

```yaml
values:
  family: module
```

The reserved `_layout.yaml` file participates in preflight but never becomes a
published artifact. A landing page selects its kind directly:

```yaml
---
layout:
  kind: landing
---
```

## Documentation families

Docs families are centrally declared by `layout.default.families`. `default`
always exists. Directory and Markdown patches may select a declared family but
cannot declare one. Declaration order controls secondary navigation. Each
non-default family must resolve to at least one docs page, and its overview is
the first family route, preferring an `index.md` route. Sidebar contents and
pagination stay inside the active docs family. Article and landing pages never
carry family identity.

Unknown kinds, unknown values, invalid patch shapes, undeclared families, and
empty families fail during preflight before output is emitted. Patch discovery
and application use normalized sorted paths, so equivalent inputs produce the
same manifest identity.

Top-level `frame` and `shell` remain compatibility paths for configured sites
that do not select `layout`. They are mutually exclusive with typed `layout`.
