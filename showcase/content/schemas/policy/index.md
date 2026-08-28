---
title: Policy schema
description: The versioned schema for trusted Margo host policy.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Policy schema

This schema describes the trusted host policy passed with `--policy`. It keeps
capabilities and resource ceilings explicit; document content cannot grant
itself these permissions.

```jsonschema ref=margo://schema/v1/policy.json
```

## Used by

- [`margo check`](../../cli/check/index.md) and the standalone render commands.
- [Policy and security guidance](https://github.com/araihu/margo/blob/v0.0.17/docs/policy.md).
- `margo schema policy`, which emits the exact embedded bytes.
