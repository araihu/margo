---
title: Runtime report schema
description: The versioned schema for terminal browser validation evidence.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Runtime report schema

This schema describes terminal browser-runtime evidence collected while a
document or deck is being validated.

```jsonschema ref=margo://schema/v1/output/runtime-report.json
```

## Used by

- The Go module's PDF and deck integrations.
- Artifact reports that record task status, layout, and identity.
- `margo schema runtime-report`, which emits the exact embedded bytes.
