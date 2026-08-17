---
title: Go module
description: Integrate Margo as one root module with focused packages for each output boundary.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Go module

Margo ships as one root Go module. Applications can use the compiler directly,
select an output package, and keep URL, navigation, storage, and deployment
ownership in the host application.

## Install once

```sh
go get github.com/araihu/margo@vX.Y.Z
```

## Pick the package that matches the job

| Package | Entry point |
| --- | --- |
| `github.com/araihu/margo` | `margo.New`, `Compile`, and `Render` |
| `github.com/araihu/margo/charts` | `charts.Extension` |
| `github.com/araihu/margo/deck` | `deck.Render` |
| `github.com/araihu/margo/pdf` | `pdf.Engine.Export` |
| `github.com/araihu/margo/site` | `site.Build` |

```go
import (
    margo "github.com/araihu/margo"
    "github.com/araihu/margo/site"
)
```

The CLI package is the executable surface; the root module packages are the
library surface. This split keeps integrations explicit without requiring a
second module for every projection.

## Compose at the edge

The module exposes semantic HTML and dependency requirements. The application
can then add its own page shell, trusted asset handlers, and publication
identity—or select a supported shell such as the Goshtoso component docs shell
used by this showcase.
