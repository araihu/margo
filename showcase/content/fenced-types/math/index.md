---
title: Math and LaTeX
description: Write inline equations, display formulas, and math fences in Markdown with bundled offline KaTeX.
language: en
margo:
  actions:
    markdown: true
    pdf: true
---

# Math and LaTeX

Keep equations next to the explanation, such as $E = mc^2$, in the same
Markdown source. Enable `--math` to typeset inline expressions, display
equations, and `math` fences with bundled KaTeX.

$$
\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}
$$

These are live Margo equations. Compare the authoring source and rendered
results below, or use the page menu to download this guide as a PDF.

## Enable math

Math is opt-in for each compiler or command. Use the same flag for checks,
HTML, sites, live previews, PDFs, and decks:

```sh
margo check notes.md --math
margo html notes.md --math --output notes.html
margo pdf notes.md --math --output notes.pdf
margo site ./docs --math --output-dir ./dist
margo serve ./docs --math
margo deck slides.md --math --format pdf --output slides.pdf
```

Site YAML and document frontmatter do not enable math. For this repository's
showcase, run `go run ./cmd/margo serve showcase.yaml --math --allow-unsafe-html`.

## Inline expressions

Use single dollars to put an equation in a sentence:

```markdown
Energy and mass are related by $E = mc^2$.
For a right triangle, $a^2 + b^2 = c^2$.
```

Energy and mass are related by $E = mc^2$.
For a right triangle, $a^2 + b^2 = c^2$.

Inline math also works in headings, lists, link labels, and tables:

| Quantity | Expression |
| --- | --- |
| Circle area | $A = \pi r^2$ |
| Quadratic roots | $x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}$ |
| Finite sum | $\sum_{i=1}^{n} i = \frac{n(n+1)}{2}$ |

## Display equations

Put double dollars on their own lines for a centered equation. A compact
`$$expression$$` line works too.

```markdown
$$
\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}
$$
```

$$
\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}
$$

## Fenced math

Use a `math` fence for multiline formulas. Write the expression directly,
without dollar delimiters inside the fence:

````markdown
```math
\begin{aligned}
f(x) &= x^2 + 2x + 1 \\
     &= (x + 1)^2
\end{aligned}
```
````

```math
\begin{aligned}
f(x) &= x^2 + 2x + 1 \\
     &= (x + 1)^2
\end{aligned}
```

Matrices use the same fence and engine:

```math
A = \begin{pmatrix}1 & 2 \\ 3 & 4\end{pmatrix},
\qquad \det(A) = -2
```

Backtick and tilde fences both work, including inside lists and blockquotes.
Fences named `tex` or `latex` remain highlighted source code.

## Literal dollars and Markdown

Currency such as $20 and $30 stays text. Escape a literal dollar with `\$`.
Code spans such as `$x^2$`, ordinary code fences, and image alt text do not
activate math.

An inline opener must be followed by non-whitespace. A closer must have
non-whitespace before it and cannot be followed by a digit. Close inline math
on the same line, and close every display block and math fence explicitly.
Markdown formatting is opaque inside an equation. In a table-cell formula,
use `\vert` or `\mid` for a vertical bar; a Markdown pipe still separates cells.

## Go applications

```go
compiler := margo.New(margo.WithMath(true))
document, err := compiler.Compile(ctx, source)
// Handle err, then render with this compiler and the chosen projection.

findings, err := margo.Check(ctx, source, margo.WithCheckMath(true))
```

Pass the compiler to `site.BuildConfig` or `deck.Render`. Embedded HTML hosts
mount the result's declared dependencies and await `window.margoRuntimeReady`
before measuring equations or Mermaid diagrams. See the
[Go module guide](../../module/index.md#math-in-markdown) for the integration boundary.

## Offline rendering and validation

KaTeX, styles, and all required fonts ship with Margo. Published equations need
no CDN. HTML typesetting runs in browser JavaScript and includes MathML;
without JavaScript, readable TeX source remains visible.

`margo check --math` validates delimiters and source limits, and reports
`check.math_runtime_validation_required`. Full expression and font validation
happens in the browser. Invalid expressions retain their source and show an
error. PDF export requires successful typesetting and rejects missing fonts
or equations that would be clipped. Wide HTML display equations scroll locally
with keyboard access; print output does not silently shrink or crop them.

The limits are 16 KiB per equation, 1,024 equations and 1 MiB of combined
equation source per publication, 256 KiB of generated output per equation,
and 8 MiB of combined math output within the effective output budget.

This is LaTeX **math inside Markdown**. Standalone `.tex` documents, TeX
packages, file imports, TikZ, equation labels/references, automatic numbering,
persistent macros, and extra delimiters such as `\(...\)` remain outside
this feature's scope.
