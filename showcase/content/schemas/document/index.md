---
title: Document schema
description: The versioned schema for Markdown frontmatter and page actions.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Document schema

This schema describes the generic metadata and Margo-owned rendering
preferences accepted in Markdown frontmatter. Consumer-owned metadata remains
available outside the Margo namespace.

```jsonschema ref=margo://schema/v1/document.json
```

## Used by

- [`margo check`](../../cli/check/index.md) when validating Markdown.
- [`margo html`](../../cli/html/index.md), [`margo pdf`](../../cli/pdf/index.md),
  and [`margo deck`](../../cli/deck/index.md).
- `margo schema document`, which emits the exact embedded bytes.
