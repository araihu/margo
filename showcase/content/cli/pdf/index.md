---
title: pdf
description: Export a Markdown document to PDF through an installed local rendering engine.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# `margo pdf`

## Purpose

`pdf` renders one Markdown document, then exports it through a selected local
PDF engine.

## Input and output

Input is one path or `-` for stdin. Unlike `html`, the command requires an
explicit `--output PATH` or `--output -`. PDF bytes are binary and go only to
that destination; diagnostics stay on stderr.

The document defaults are A4 portrait with margins of 24 mm top, 22 mm right,
26 mm bottom, and 22 mm left. `--image-overflow limit` caps print images to the
safe projection; `allow` permits overflow. `--print-chart-data` includes the
accessible exact-data tables that PDF otherwise omits after charts.

## Examples

Run a compatibility check and engine probe before conversion:

```sh
margo check docs/guide.md --target pdf
margo doctor
margo pdf docs/guide.md \
  --output build/guide.pdf \
  --page-size A4 \
  --orientation portrait
```

Document frontmatter can declare page size, orientation, image overflow, and
individual margins. Explicit CLI flags override those preferences. Setting all
four margin flags to `0` requests a full-bleed page:

```sh
margo pdf docs/guide.md \
  --output build/guide-full-bleed.pdf \
  --margin-top 0 \
  --margin-right 0 \
  --margin-bottom 0 \
  --margin-left 0
```

Relative PDF links default to `strip`: link text remains, but the relative
target is removed. `--base-url` implies `resolve` unless `--relative-links` was
explicitly set. Use an absolute public HTTP or HTTPS URL; loopback and
unspecified hosts are rejected.

```sh
margo pdf docs/guide.md \
  --output build/guide-public.pdf \
  --base-url https://docs.example.com/manual/
```

## Failures and diagnostics

| Code | Meaning |
| --- | --- |
| `cli.output_required` | No PDF destination was selected |
| `pdf.engine_mode_invalid` | Engine is not `auto`, `chromium`, or `native` |
| `pdf.engine_path_invalid` | Explicit executable is unusable |
| `pdf.engine_unavailable` | No requested candidate is available |
| `pdf.native.compiled_out` | This build has no verified native backend |
| `pdf.page_size_unsupported` | Size is not A4 or Letter |
| `pdf.orientation_unsupported` | Orientation is unsupported |
| `pdf.margin_invalid` | A margin is negative or non-finite |
| `cli.relative_link_base_required` | `resolve` has no `--base-url` |
| `pdf.relative_link_base_invalid` | Base URL is unsafe or malformed |
| `pdf.relative_link_forbidden` | `error` policy found a relative link |

Existing output requires `--force`. All command failures exit `1` and report to
stderr without replacing a protected destination.

## Limitations and care

Engine mode is `auto`, `chromium`, or `native`. Auto discovery checks an
explicit `--engine-path`, `MARGO_CHROMIUM_PATH`, executables on `PATH`, known
platform locations, then the native slot. Margo never downloads a browser. A
selected engine that fails does not fall back to another candidate. `doctor`
reports candidates but cannot guarantee a later document-specific render will
succeed.
