---
title: Policy and diagnostics
description: Keep privileged document features explicit and make failures actionable.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Policy and diagnostics

Ordinary Markdown needs no policy file. Raw HTML and iframe embeds require an
explicit host policy, preventing a document from widening what an output may
load or publish.

## Check before you render

```sh
margo check guide.md --diagnostics json
margo html guide.md --policy policy.json --diagnostics json
```

Checks identify the source and, when available, a line or field. They also
include a remediation hint. Text diagnostics suit local work; JSON diagnostics
provide structured CI input.

## Policy is host-owned

```json
{
  "schemaVersion": "margo-policy/v1",
  "rawHTML": "sanitized",
  "iframe": {
    "allowedOrigins": ["https://video.example.com"],
    "projections": {
      "html": "interactive",
      "site": "interactive",
      "pdf": "static-link",
      "deck": "deny"
    }
  }
}
```

A document cannot grant itself capabilities through frontmatter. The host
chooses the policy, the CLI validates its schema, and each output target applies
its configured projection.

## A useful failure is part of the feature

An invalid link, image, SVG, heading sequence, policy field, or engine
requirement fails with a stable diagnostic code and a next action. Margo does
not silently publish a partial or best-effort result.
