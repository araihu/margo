---
title: Go module
description: Integrate Margo into a Go application with one compiler and focused output packages.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Go module

The Margo root module turns Markdown into an immutable compiled document. Your
application chooses the output boundary and keeps ownership of its URLs,
navigation, assets, storage, and deployment.

## A small compiling example

This complete program compiles and renders one source. A host can replace the
standalone page with its own HTML composition or site frame.

```go
package main

import (
    "context"
    "log"

    margo "github.com/araihu/margo"
)

func main() {
    compiler := margo.New()
    document, err := compiler.Compile(context.Background(), margo.Source{
        Name:    "guide.md",
        Content: []byte("# Guide\n\nHello from Margo.\n"),
    })
    if err != nil {
        log.Fatal(err)
    }
    rendered, err := compiler.Render(context.Background(), document)
    if err != nil {
        log.Fatal(err)
    }
    page, err := margo.RenderStandalone(rendered)
    if err != nil {
        log.Fatal(err)
    }
    _ = page // Render the component in the host-owned HTTP response.
}
```

## Compiler and render lifecycle

`Compile` parses Markdown, normalizes closed metadata, evaluates the host policy,
and freezes the source into an immutable document. `Render` projects that
document for a target and exposes semantic HTML plus dependency requirements.
Reuse one compiler across requests; it is safe for concurrent compile and render
calls. Pass a canceled context to stop work at the API boundary.

## Public package map

| Package | Responsibility |
| --- | --- |
| `github.com/araihu/margo` | `New`, `Compile`, `Render`, metadata, and HTML projections |
| `github.com/araihu/margo/charts` | Optional static SVG and interactive chart extension |
| `github.com/araihu/margo/deck` | Experimental HTML/PDF presentation projection |
| `github.com/araihu/margo/pdf` | Renderer-neutral PDF request and engine contracts |
| `github.com/araihu/margo/site` | Linked-site build, config, routes, and publication artifacts |
| `github.com/araihu/margo/ssg` | Layout-neutral frame, shell, schema, and binding contracts |

Use only the packages your host needs. The `margo` executable is a separate
command surface built on these same boundaries.

## Host ownership boundaries

Margo owns Markdown semantics, diagnostics, target projection, and declared
dependency requirements. The host owns page shells, navigation, canonical URLs,
asset staging, authentication, persistence, and publication. `RenderHTML`
returns a semantic fragment; `RenderHTMLPage` supplies a generic page shape when
that is useful, while a configured site may compose its own frame.

This separation lets a Go service adopt the compiler without importing a
particular visual shell or deployment system.

## Extensions and policy

Optional capabilities are registered by the host. For example, chart support is
explicit:

```go
import (
    margo "github.com/araihu/margo"
    "github.com/araihu/margo/charts"
)

compiler := margo.New(
    margo.WithExtension(charts.Extension()),
)
```

A host policy controls privileged raw HTML and iframe behavior. Frontmatter can
carry document preferences and site publication actions, but it cannot grant
itself capabilities. Check documents before rendering when CI needs structured
diagnostics.

## Future chapters

The technical manual will expand in this order:

1. Installation and versioning
2. Compiler lifecycle
3. Source and metadata
4. Checks and diagnostics
5. Rendering and HTML composition
6. Site builds
7. PDF engines
8. Deck projection
9. Chart extensions
10. Policy and security
11. Concurrency and cancellation
12. Determinism
13. Testing

Only this overview ships in the first increment; chapter routes are added when
their content is ready.
