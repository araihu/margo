---
title: JSON Schema fence
description: Render a versioned JSON Schema as an indented property tree.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# JSON Schema fence

The `jsonschema` fence validates an inline schema or a `ref=` path and renders
its properties as an indented tree.

## Source

````markdown
```jsonschema
{
  "title": "Widget",
  "type": "object",
  "required": [
    "id"
  ],
  "properties": {
    "id": {
      "type": "string"
    },
    "labels": {
      "type": "array",
      "items": {
        "type": "string"
      }
    }
  }
}
```
````

## Result

```jsonschema
{
  "title": "Widget",
  "type": "object",
  "required": [
    "id"
  ],
  "properties": {
    "id": {
      "type": "string"
    },
    "labels": {
      "type": "array",
      "items": {
        "type": "string"
      }
    }
  }
}
```

## Options

Use an inline JSON object, `ref=path/to/schema.json`, or an embedded
`ref=margo://schema/v1/...` contract. A `#/pointer` fragment selects a nested
schema. The result is always a validated, indented property tree.
