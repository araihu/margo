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

Import one root Go module, then select the package for the required output.
Your application retains ownership of URLs, navigation, storage, and
deployment.

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
| `github.com/araihu/margo/site` | `site.Build`, `site.LoadConfig`, and `site.BuildConfig` |
| `github.com/araihu/margo/ssg` | Layout-neutral frame and shell contracts |

```go
import (
    margo "github.com/araihu/margo"
    "github.com/araihu/margo/site"
)
```

The `margo` executable provides the command surface. Root-module packages
provide the library surface, so an application can depend only on the output
boundaries it uses.

## Compose at the edge

The module exposes semantic HTML and dependency requirements. Add a page shell,
trusted asset handlers, and publication identity in the host, or select a
supported shell such as the Goshtoso component docs shell used here.
