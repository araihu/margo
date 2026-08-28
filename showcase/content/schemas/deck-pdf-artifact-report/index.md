---
title: Deck PDF artifact report schema
description: The versioned schema for validated deck PDF output evidence.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Deck PDF artifact report schema

This schema describes media-box evidence and composition identity returned
after a deck PDF has been validated.

```jsonschema ref=margo://schema/v1/output/deck-pdf-artifact-report.json
```

## Used by

- The [`margo deck`](../../cli/deck/index.md) PDF workflow.
- CI and release checks that retain deck artifact provenance.
- `margo schema deck-pdf-artifact-report`, which emits the exact embedded bytes.
