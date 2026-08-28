---
title: Deck layout evidence schema
description: The versioned schema for deck screen and print layout evidence.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Deck layout evidence schema

This schema describes quantized screen or print-DOM geometry captured while a
deck is validated.

```jsonschema ref=margo://schema/v1/output/deck-layout-evidence.json
```

## Used by

- The [`margo deck`](../../cli/deck/index.md) layout validation workflow.
- Deck artifact reports and CI checks for overflow or geometry regressions.
- `margo schema deck-layout-evidence`, which emits the exact embedded bytes.
