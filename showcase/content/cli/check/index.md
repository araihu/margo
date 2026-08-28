---
title: check
description: Validate a Markdown document for a specific Margo target without rendering it.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# `margo check`

## Purpose

`check` validates one document without producing HTML, PDF, or site files. Its
`--target` value is `html`, `site`, `pdf`, or `deck`; the default is `html`.
Target selection matters because a feature can be safe for an interactive page
but unsuitable for a static or printable projection.

## Input and output

Input is one Markdown path or `-` for stdin. Findings and the final error and
warning counts go to stdout in text mode. JSON mode emits an object with
`diagnostics`, `errors`, `warnings`, and an optional policy digest. Each finding
can carry a source, line, column, field pointer, and remediation hint.

The JSON output contract is rendered directly from the versioned schema shipped
with Margo:

```jsonschema ref=margo://schema/v1/output/check-report.json
```

See the [check report schema reference](../../schemas/check-report/index.md)
for the full tree and its other consumers.

Command failures that prevent a report go to stderr. See the [CLI overview](../index.md)
for shared input limits, policy rules, and stream behavior.

Raw HTML and iframe markup are denied by default. Pass
`--allow-unsafe-html` when checking a trusted document that intentionally uses
arbitrary HTML; the alias `--allow-raw-html` is equivalent.

## Examples

```sh
margo check docs/guide.md --target html
margo check docs/guide.md --target pdf --diagnostics json \
  > build/check-pdf.json
cat docs/guide.md | margo check - --target deck
```

For `--target site`, ordinary relative Markdown links are accepted by the
single-document check because the multi-page site build resolves and validates
them after indexing all source documents. Other link diagnostics, such as an
empty destination or an unsupported scheme, still apply to every target.

The CLI registers its chart checker automatically. Interactive Goshtoso Charts
are valid for `html`, `site`, and standalone `pdf`; the `deck` target is a
static projection for both HTML and PDF deck artifacts. Therefore
`margo check --target deck` rejects `renderer: interactive` before rendering and
points to `/renderer`; omit the field or set it to `static`.

## Failures and diagnostics

Warnings remain visible but do not fail the command. Any error finding writes
the complete report, then exits `1` with internal status `check.failed`.

| Code | Typical correction |
| --- | --- |
| `frontmatter.schema_invalid` | Correct closed frontmatter fields and types |
| `check.language_missing` | Add a BCP 47 `language` value |
| `check.image_alt_empty` | Add meaningful image alternative text |
| `check.asset_missing` | Restore or correct a local image path |
| `check.svg_incompatible` | Use a supported static SVG |
| `check.heading_level_skipped` | Restore a sequential heading outline |
| `check.link_destination_empty` | Give the link a destination |
| `check.link_relative` | Review target-specific relative-link behavior; ordinary site links are resolved by `margo site` |
| `check.raw_html` | Remove raw HTML or supply trusted policy authority |
| `mermaid.configuration_forbidden` | Remove legacy Mermaid configuration |
| `chart.renderer_target_unsupported` | Use `renderer: static` for `margo deck`; use `margo html`, `margo site`, or `margo pdf` for interactive charts |

## Limitations and care

`check` does not prove that a Chromium executable can launch or export a PDF.
Use `doctor` for discovery and run the real rendering command in the target
environment. A single-document check is also not a complete multi-page site
build with cross-page link and anchor validation.
