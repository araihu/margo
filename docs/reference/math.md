# Math in Markdown

Enable math with `--math` on `check`, `html`, `pdf`, `site`, `serve`, or `deck`:

```sh
margo html guide.md --math --output guide.html
margo pdf guide.md --math --output guide.pdf
margo serve ./docs --math
margo deck slides.md --math --format pdf --output slides.pdf
```

Math is disabled by default. Site YAML and document frontmatter do not enable
it. Configured sites use the command's flag or the supplied Go compiler.
`--math=false` retains ordinary Markdown interpretation.

## Authoring

Use single dollars for inline equations:

```markdown
The energy is $E = mc^2$.
```

Use double dollars for a display equation on its own line, either compact or
multiline:

```markdown
$$E = mc^2$$

$$
\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}
$$
```

Fenced blocks with language `math` also produce display equations. Write the
expression without dollar delimiters inside the fence:

````markdown
```math
\begin{aligned}
a &= b + c \\
d &= e + f
\end{aligned}
```
````

Backtick and tilde math fences work in lists and blockquotes. Close every math
fence and display block explicitly. Ordinary `tex`, `latex`, and other code
fences continue to show source code. With math disabled, `math` fences also
remain ordinary code unless a host extension owns them.

Inline math works in headings, paragraphs, lists, blockquotes, table cells,
and link labels. Display math works in document bodies, lists, blockquotes,
slide bodies, and composition slots. Deck headers, footers, and presenter
notes keep their existing interpretation.

Inside an equation, Markdown formatting is not interpreted. In table cells,
use `\vert` or `\mid` for a vertical bar, because an unescaped Markdown pipe
still separates columns.

## Dollar signs and boundaries

An inline opener must have non-whitespace immediately after it. A closer must
have non-whitespace immediately before it and cannot be followed by a digit.
An inline equation must close on the same source line. Literal dollars can be
escaped with `\$`; code spans and ordinary code blocks are protected.

| Source | Result |
| --- | --- |
| `$x^2$` | Inline equation. |
| `Price: $20 and $30.` | Ordinary currency text. |
| `$ x$`, `$x $`, `$x$2` | Ordinary text. |
| `$x$ $y$` | Two inline equations. |
| `$x$$y$` | Ordinary text; adjacent dollar runs are not inline delimiters. |
| `$$x$$` on a line by itself | Display equation. |
| `Text $$x$$ text` | Ordinary text in Margo's profile. |

Dollar handling differs across Markdown implementations. Margo does not claim
complete GitHub or Pandoc grammar compatibility. `\(...\)` and `\[...\]` are
not math delimiters in this version.

## Go API and embedded applications

```go
compiler := margo.New(margo.WithMath(true))
document, err := compiler.Compile(ctx, source)
// Handle err, then use compiler.Render and your chosen output projection.

findings, err := margo.Check(ctx, source, margo.WithCheckMath(true))
```

Pass this compiler through `site.Request.Compiler`,
`site.ConfigRequest.Compiler`, or `deck.Render`. Deck rendering derives its
parsing mode from `compiler.MathEnabled()`. For parsing without rendering:

```go
document, err := deck.ParseWithOptions(name, markdown, deck.WithMath(true))
```

The existing `deck.Parse` function retains its disabled default. The core math
capability reserves the `math` fence while enabled; a host extension cannot
also claim it.

Embedded HTML hosts should render the result's declared dependencies through
`margo.RenderHTMLDependencies` or `margo.RenderHTMLPage` and mount the declared
asset handlers. `margo.HTMLAssetHandler()` serves first-party math assets and
the bundled KaTeX engine under `/margo-assets/`.

Await `window.margoRuntimeReady` when measurements depend on completed math and
Mermaid rendering. Call `window.margoRunMath()` after mounting new Margo
content. It starts a shared execution snapshot; completed, unchanged equations
are reused. Margo's configured documentation shell does this after HTMX swaps.
Use a scoped render ID allocator when composing independently rendered
fragments in one document.

## Rendering and validation

Margo bundles KaTeX 0.19.1, its stylesheet, and 20 WOFF2 font faces. Stylesheets
contain the fonts as data URLs in both dependency modes, so local assets need
no additional font-relative URL rewriting. Standalone HTML works offline when
opened as a file. The build and generated publication do not fetch KaTeX or
download a browser.

HTML generation requires Go. Typeset equations require browser JavaScript;
before rendering, or when JavaScript is disabled, escaped TeX remains visible.
Successful rendering produces HTML plus MathML and hides the duplicate source.
This does not promise accessible mathematical tagging in exported PDF.

`margo check --math` checks delimiters, source locations, and source limits in
Go. It emits `check.math_runtime_validation_required` when math is present.
It does not validate the full TeX expression. HTML/site/deck HTML builds also
leave expression validation to the browser. PDF export requires successful
typesetting, font loading, and runtime evidence before printing.

Failures remain visible beside the source. PDF export rejects invalid
equations, missing engine/font resources, task identity mismatches, output
limits, and print overflow. Wide display equations scroll with keyboard access
in HTML. They are not silently cropped or scaled in PDF or decks.

## Supported expressions and limits

The supported language is the pinned KaTeX math subset under Margo's fixed
strict configuration. Fractions, roots, integrals, matrices, cases, and aligned
equations are supported. External resources, arbitrary HTML commands, `.tex`
document input/import, TeX packages, labels/references, automatic numbering,
and persistent macros between expressions are outside this version's scope.
Local macro definitions do not survive into the next equation or document.

| Resource | Limit |
| --- | --- |
| One normalized equation payload | 16 KiB |
| Equations per document, composed page, or deck | 1,024 |
| Combined equation payload | 1 MiB |
| Generated HTML/MathML per equation | 256 KiB |
| Combined generated math output | 8 MiB, also bounded by the effective output policy |
| Macro expansions per equation | 1,000 |
| User-specified size | 20 em |

The [design spec](../superpowers/specs/2026-11-05-margo-latex-math-design.md)
records the exact grammar, diagnostic codes, and renderer comparison.
