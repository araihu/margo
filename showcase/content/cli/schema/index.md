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

`schema` writes version-matched embedded JSON Schema bytes to stdout. The
configuration kinds are `policy`, `document`, and `site`; output kinds include
`doctor-report`, `check-report`, `site-report`, `site-manifest`, `diagnostic`,
`runtime-descriptor`, `runtime-report`, `deck-layout-evidence`, and
`deck-pdf-artifact-report`. All use JSON Schema Draft 2020-12. The site schema
describes the top-level `site.yaml` shape for YAML-language-server and IDE
completion, while output schemas keep generated documentation and automation
contracts in sync with the binary that emits them.

Browse the [Schemas family](../../schemas/index.md) for a rendered tree of each
kind and links to the places that consume it.

## Input and output

The required positional input is one schema kind. The exact embedded schema
bytes go to stdout. Argument errors go to stderr.

## Examples

```sh
mkdir -p build
margo schema policy > build/margo-policy.schema.json
margo schema document > build/margo-document.schema.json
margo schema site > build/margo-site.schema.json
margo schema doctor-report > build/margo-doctor-report.schema.json
margo schema check-report > build/margo-check-report.schema.json
margo schema runtime-descriptor > build/margo-runtime-descriptor.schema.json
margo schema deck-pdf-artifact-report > build/margo-deck-pdf-artifact-report.schema.json
```

## Failures and diagnostics

An unknown kind fails with `cli.schema_invalid`. Missing or extra positional
arguments also exit `1`; because `schema` is a small exact-byte command, those
argument errors use Cobra's text error rather than the custom JSON diagnostic
projection.

## Limitations and care

The emitted bytes are the schemas embedded in the installed Margo version.
Capture them from the same binary used by CI when an editor or validator must
match that build. Attach `policy` to policy JSON, `document` to Markdown
frontmatter, and `site` to `site.yaml`. For example, a YAML language server can
use this external association (the schema files do not need to be copied into
the source documents):

```json
{
  "yaml.schemas": {
    "./.schemas/margo-site.schema.json": ["site.yaml"]
  }
}
```

Policy JSON uses the equivalent `json.schemas` association. Markdown
frontmatter needs an editor extension that exposes its YAML block (or a
frontmatter extraction step) before `margo-document.schema.json` can be
attached. Do not add a
top-level `$schema` property to a policy or site config: both are closed
runtime contracts and Margo intentionally rejects that property. The command
emits schemas; it does not validate an instance or perform cross-file checks.

The schemas include `x-margo-*` annotation keywords for targets, precedence,
limits, and security effects. Validators that reject unknown annotation
keywords (for example strict Ajv) must register those vocabulary names or
disable strict unknown-keyword checks. Generic JSON Schema also cannot express
all runtime semantics: Margo still enforces duplicate JSON keys, byte ceilings,
HTTPS origin-only iframe syntax (no path, query, fragment, credentials, or
wildcards), asset existence, links, locales, and theme availability. Run
`margo check guide.md --policy policy.json` and `margo site site.yaml` in
addition to editor validation.
