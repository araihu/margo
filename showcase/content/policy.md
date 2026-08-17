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

Margo distinguishes ordinary Markdown from privileged capabilities. The normal
path needs no policy file. Raw HTML and iframe embeds require an explicit host
policy, so the same document cannot silently widen what a target is allowed to
load or publish.

## Check before you render

```sh
margo check guide.md --diagnostics json
margo html guide.md --policy policy.json --diagnostics json
```

Checks report a source, line or field pointer when available, plus a remediation
hint. JSON diagnostics make the result straightforward to consume in CI; text
diagnostics are the default for local work.

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

The document cannot change its own capabilities through frontmatter. A host
chooses the policy, the CLI validates its exact schema, and each output target
applies its own least-authoritative projection.

## A useful failure is part of the feature

If a link, image, SVG, heading sequence, policy field, or engine requirement is
invalid, Margo fails with a stable diagnostic code and a next action. That makes
the publishing boundary visible instead of turning a document build into a
best-effort guess.
