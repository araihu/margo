---
title: Structural layouts
description: The six closed structural layouts in the Margo deck profile.
language: en
theme: modern
colorMode: light
size: 16:9
---

<!-- _class: columns -->
<!-- layout: columns -->
<!-- slot: left -->
# Context

The left column carries the premise or primary evidence.
<!-- slot: right -->
# Decision

The right column carries the consequence or supporting action.
<!-- /layout -->

---

<!-- _class: sidebar -->
<!-- layout: sidebar -->
<!-- slot: main -->
## Main

Use the main slot for the thesis, evidence, and reading flow.
<!-- slot: rail -->
### Rail

Use the rail for definitions, context, or a short callout.
<!-- /layout -->

---

<!-- _class: compare -->
<!-- layout: compare -->
<!-- slot: left -->
## Option A

- controlled authoring
- deterministic output
<!-- slot: right -->
## Option B

- familiar Markdown input
- bounded visual vocabulary
<!-- /layout -->

---

<!-- _class: metrics -->
<!-- layout: metrics -->
<!-- slot: metric-1 -->
### 6

structural layouts
<!-- slot: metric-2 -->
### 2

two-column variants
<!-- slot: metric-3 -->
### 3–6

timeline slots
<!-- slot: metric-4 -->
### 0

arbitrary CSS rules
<!-- /layout -->

---

<!-- _class: timeline -->
<!-- layout: timeline -->
<!-- slot: step-1 -->
### Input

Write Markdown and closed directives.
<!-- slot: step-2 -->
### Normalize

Resolve the class, slots, and source order.
<!-- slot: step-3 -->
### Publish

Render HTML or validate a PDF projection.
<!-- /layout -->

---

<!-- _class: demo -->
<!-- layout: demo -->
<!-- slot: code -->
### Source

```markdown
<!-- composition: media-split -->
<!-- slot: media -->
Evidence placeholder
<!-- slot: content -->
## Decision
Keep the source order explicit.
```
<!-- slot: result -->
### Result

The same source can produce a navigable HTML deck and a validated PDF.
<!-- /layout -->
