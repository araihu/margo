---
title: Code fence
description: Highlight source code with copy controls and readable overflow behavior.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Code fence

Ordinary fenced code blocks preserve source text and receive syntax highlighting
when the language is known. The document shell adds a copy control; long inline
code uses ellipsis with the full value available on hover.

## Source

````markdown
```go
document, err := compiler.Compile(ctx, source)
```
````

## Result

```go
document, err := compiler.Compile(ctx, source)
```

## Options

Set the language tag (`go`, `sh`, `yaml`, and so on) to select highlighting.
Omit it for plain text. Block fences receive copy controls; inline code keeps
one line and uses ellipsis with the full value available on hover.
