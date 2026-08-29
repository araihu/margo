---
title: Schemas
description: Versioned Margo JSON Schemas, rendered as trees for inspection.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Schemas

Margo ships version-matched JSON Schemas for configuration, diagnostics, CLI
reports, runtime evidence, and chart payloads. Each schema has its own page
with a rendered property tree for inspection.

## Configuration schemas

- [Policy](policy/index.md) — trusted host capabilities and resource limits.
- [Document](document/index.md) — Markdown frontmatter and page actions.
- [Site](site/index.md) — the closed `site.yaml` publication contract.
- [Goshtoso charts](goshtoso-chart/index.md) — chart payload schemas for bar,
  line, pie/doughnut, and scatter charts.

## Output and runtime schemas

- [Diagnostic](diagnostic/index.md) — one structured diagnostic object.
- [Check report](check-report/index.md) — `margo check --diagnostics json` output.
- [Doctor report](doctor-report/index.md) — `margo doctor --diagnostics json` output.
- [Site report](site-report/index.md) — configured site build results.
- [Site manifest](site-manifest/index.md) — generated route and asset manifest.
- [Runtime descriptor](runtime-descriptor/index.md) — browser task input identity.
- [Runtime report](runtime-report/index.md) — browser validation evidence.
- [Deck layout evidence](deck-layout-evidence/index.md) — deck screen/print layout evidence.
- [Deck PDF artifact report](deck-pdf-artifact-report/index.md) — validated deck PDF output.

The [`margo schema`](../cli/schema/index.md) command emits the exact bytes for
each kind from the installed binary. The trees on these pages are generated
from those same embedded documents, so they stay aligned with the release.
