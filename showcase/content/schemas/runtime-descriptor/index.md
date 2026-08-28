---
title: Runtime descriptor schema
description: The versioned schema for browser task input identity.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Runtime descriptor schema

This schema describes the immutable task descriptor supplied to the browser
runtime when a rendered document needs runtime validation.

```jsonschema ref=margo://schema/v1/output/runtime-descriptor.json
```

## Used by

- The Go module's PDF and deck integrations.
- Browser validation tasks that accompany rendered HTML.
- `margo schema runtime-descriptor`, which emits the exact embedded bytes.
