---
title: Site manifest schema
description: The versioned schema for the generated margo-manifest.json file.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Site manifest schema

This schema describes `margo-manifest.json`, the exact publication identity
and route/artifact manifest emitted by a configured site build.

```jsonschema ref=margo://schema/v1/output/site-manifest.json
```

## Used by

- [`margo site`](../../cli/site/index.md) publication output.
- Deployment tooling that needs route and asset provenance.
- `margo schema site-manifest`, which emits the exact embedded bytes.
