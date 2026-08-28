---
title: Site schema
description: The versioned schema for the closed site.yaml contract.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Site schema

This schema describes the top-level `site.yaml` configuration used to build a
linked publication: source and output paths, identity, locales, navigation,
layouts, and theme settings.

```jsonschema ref=margo://schema/v1/site.json
```

## Used by

- [`margo site`](../../cli/site/index.md) and [`margo serve`](../../cli/serve/index.md).
- YAML language-server and editor associations for `site.yaml`.
- `margo schema site`, which emits the exact embedded bytes.
