---
title: Check report schema
description: The versioned JSON contract emitted by margo check.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Check report schema

This schema describes the complete report produced by
`margo check --diagnostics json`, including findings and summary counts.

```jsonschema ref=margo://schema/v1/output/check-report.json
```

## Used by

- The [`margo check`](../../cli/check/index.md) JSON output.
- CI checks that consume compatibility findings deterministically.
- `margo schema check-report`, which emits the exact embedded bytes.
