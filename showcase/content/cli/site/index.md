---
title: site
description: Build and validate a linked static site from Markdown sources or a Margo config.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# `margo site`

## Purpose

`site` builds linked HTML pages from a Markdown directory or a `.yaml` or
`.yml` site config. Directory mode recursively discovers regular `.md` and
`.markdown` files, rewrites valid Markdown links, and validates anchors and
local images. It does not read stdin.

## Input and output

Directory mode requires `--output-dir`. The destination must not already
exist. `--assets local` copies validated assets; `inline` embeds supported
images. On success, the new directory contains page artifacts, required runtime
assets, and `margo-manifest.json`.

The report on stdout contains `margo-site-report/v1`, artifact count, manifest
digest, page records, and an optional policy digest. Build and publication
failures go to stderr.

Config mode takes source, output, assets, site identity, base URL or path,
layout, navigation, locale, theme, and publication settings from the config.
Without `--output-dir`, configured output is resolved beside that file. An
explicit `--output-dir` changes only the publication destination.

## Examples

```sh
margo site ./docs \
  --output-dir ./build/site-2026-08-20 \
  --assets local \
  --diagnostics json > build/site-report.json
```

```sh
margo site ./site.yaml --diagnostics text > build/site-report.txt
```

## Failures and diagnostics

| Code | Meaning |
| --- | --- |
| `site.output_required` | Directory mode has no `--output-dir` |
| `site.output_exists` | Destination already exists |
| `site.sources_empty` | No public Markdown source was found |
| `site.config_invalid` | YAML or closed config fields are invalid |
| `site.identity_required` | Configured public identity is incomplete |
| `site.link_missing` | A Markdown or generated link target is absent |
| `site.anchor_missing` | A target fragment has no matching heading |
| `site.asset_external` | A site image is not a local site asset |
| `site.asset_outside_root` | An asset escapes the source root |
| `site.output_collision` | Sources map to the same output path |
| `site.artifact_collision` | Generated artifact paths conflict |

Margo builds and validates the complete result in a sibling staging directory,
then publishes by a no-replace rename. A failure exits `1` and leaves an
existing destination untouched.

## Limitations and care

Use a new versioned destination for every build; there is no `--force` for a
site tree. A site build is filesystem publication only. It does not deploy,
upload, tag, or release the generated directory.
