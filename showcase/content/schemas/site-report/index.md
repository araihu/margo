---
title: Site report schema
description: The versioned JSON contract emitted by margo site.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Site report schema

This schema describes the report produced by
`margo site --diagnostics json`, including the staged publication result and
artifact counts.

```jsonschema ref=margo://schema/v1/output/site-report.json
```

## Used by

- The [`margo site`](../../cli/site/index.md) JSON output.
- Build pipelines that verify a publication before deployment.
- `margo schema site-report`, which emits the exact embedded bytes.
