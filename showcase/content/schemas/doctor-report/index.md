---
title: Doctor report schema
description: The versioned JSON contract emitted by margo doctor.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Doctor report schema

This schema describes the complete report produced by
`margo doctor --diagnostics json`, including renderer candidates and their
availability.

```jsonschema ref=margo://schema/v1/output/doctor-report.json
```

## Used by

- The [`margo doctor`](../../cli/doctor/index.md) JSON output.
- Automation that records local PDF renderer availability.
- `margo schema doctor-report`, which emits the exact embedded bytes.
