---
title: Diagnostic schema
description: The versioned schema for one structured Margo diagnostic.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Diagnostic schema

This schema describes one stable diagnostic object: its code, severity,
location, message, and remediation hint.

```jsonschema ref=margo://schema/v1/output/diagnostic.json
```

## Used by

- JSON command failures and compatibility findings.
- The [`margo check`](../../cli/check/index.md) and [`margo doctor`](../../cli/doctor/index.md) reports.
- `margo schema diagnostic`, which emits the exact embedded bytes.
