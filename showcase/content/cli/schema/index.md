---
title: schema
description: Write a version-matched embedded Margo JSON Schema to stdout.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# `margo schema`

## Purpose

`schema` writes version-matched embedded JSON Schema bytes to stdout. The only
schema kinds are `policy` and `document`; there is no `site` schema command.
Both schemas use JSON Schema Draft 2020-12.

## Input and output

The required positional input is one schema kind: `policy` or `document`. The
exact embedded schema bytes go to stdout. Argument errors go to stderr.

## Examples

```sh
margo schema policy > build/margo-policy.schema.json
margo schema document > build/margo-document.schema.json
```

## Failures and diagnostics

An unknown kind fails with `cli.schema_invalid`. Missing or extra positional
arguments also exit `1`; because `schema` is a small exact-byte command, those
argument errors use Cobra's text error rather than the custom JSON diagnostic
projection.

## Limitations and care

The emitted bytes are the schemas embedded in the installed Margo version.
Capture them from the same binary used by CI when an editor or validator must
match that build. The command does not validate a document or policy instance.
