---
title: Goshtoso chart schemas
description: The chart payload schemas shipped by the Goshtoso Charts extension.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Goshtoso chart schemas

The Goshtoso Charts extension validates each `goshtosochart` fence against a
versioned schema. The chart families are `bar`, `line`, `pie`/`doughnut`, and
`scatter`; all require `schemaVersion: 1` and a title.

## Source example

```yaml
schemaVersion: 1
type: bar
renderer: static
title: Weekly output
categories: [Mon, Tue, Wed]
series:
  - name: Pages
    values: [3, 5, 4]
```

## Schema options

Every family accepts `renderer: static` or `renderer: interactive` where the
target supports it. Shared styling includes a palette, custom class, and color
list. Bar and line charts use `categories` plus numeric `values`; pie charts
use non-negative `slices`; scatter charts use points or matrix values.

## Bar schema

```jsonschema
{
  "title": "Goshtoso bar chart",
  "type": "object",
  "required": ["schemaVersion", "type", "title", "categories", "series"],
  "properties": {
    "schemaVersion": {"const": 1},
    "type": {"const": "bar"},
    "renderer": {"enum": ["static", "interactive"]},
    "title": {"type": "string"},
    "categories": {"type": "array"},
    "series": {"type": "array"}
  }
}
```

## Line schema

```jsonschema
{
  "title": "Goshtoso line chart",
  "type": "object",
  "required": ["schemaVersion", "type", "title", "categories", "series"],
  "properties": {
    "schemaVersion": {"const": 1},
    "type": {"const": "line"},
    "renderer": {"enum": ["static", "interactive"]},
    "title": {"type": "string"},
    "categories": {"type": "array"},
    "series": {"type": "array"}
  }
}
```

## Pie and doughnut schema

```jsonschema
{
  "title": "Goshtoso pie chart",
  "type": "object",
  "required": ["schemaVersion", "type", "title", "slices"],
  "properties": {
    "schemaVersion": {"const": 1},
    "type": {"enum": ["pie", "doughnut"]},
    "renderer": {"enum": ["static", "interactive"]},
    "title": {"type": "string"},
    "slices": {"type": "array"}
  }
}
```

## Scatter schema

```jsonschema
{
  "title": "Goshtoso scatter chart",
  "type": "object",
  "required": ["schemaVersion", "type", "title", "categories", "series"],
  "properties": {
    "schemaVersion": {"const": 1},
    "type": {"const": "scatter"},
    "renderer": {"enum": ["static", "interactive"]},
    "title": {"type": "string"},
    "categories": {"type": "array"},
    "series": {"type": "array"}
  }
}
```

The canonical schema sources live with the extension:

- [`bar.json`](https://github.com/araihu/margo/blob/v0.0.17/charts/schema/v1/bar.json)
- [`line.json`](https://github.com/araihu/margo/blob/v0.0.17/charts/schema/v1/line.json)
- [`pie.json`](https://github.com/araihu/margo/blob/v0.0.17/charts/schema/v1/pie.json)
- [`scatter.json`](https://github.com/araihu/margo/blob/v0.0.17/charts/schema/v1/scatter.json)
