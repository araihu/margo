# Margo LaTeX Math v1 Design

**Date:** 2026-11-05
**Status:** Implemented and validated in the working tree.
**Scope revision:** 2026-11-05 renderer comparison adds compact display
blocks and fenced `math` blocks; `.tex` input/import remains deferred. See
section 14 for evidence and the compatibility boundary.
**Scope:** Opt-in LaTeX math in Markdown, rendered with bundled KaTeX across
embedded HTML, standalone HTML, sites, PDF, and slide bodies/composition slots.
**Implementation status:** See [math reference](../../reference/math.md) for
the authoring/API contract and section 15 for validation evidence.

## 1. Product contract

An author enables math once for a compiler or CLI invocation and uses ordinary
Markdown with inline and display equations:

```markdown
The energy is $E = mc^2$.

$$
\int_0^\infty e^{-x^2}\,dx = \frac{\sqrt{\pi}}{2}
$$
```

Margo parses the delimiters and preserves the expression as semantic source.
Bundled KaTeX performs expression validation and typesetting in the browser.
All JavaScript, styles, and fonts needed for this feature work offline. Root
HTML generation remains a Go operation that requires neither Node.js nor
Chromium. PDF export continues using the explicitly selected installed Chromium
engine. Deck HTML follows the same browser-validation boundary as other HTML.

With math disabled, existing parsing and semantic output behavior remain
unchanged and no math dependencies or tasks are emitted. Shared runtime
implementation changes may update dependency bytes and their versioned hashes.
Enabling math in a document with no recognized equations adds no KaTeX
dependencies or tasks.

The initial release does not add `.tex` document input, TeX package loading,
filesystem inclusion, shell commands, TikZ, automatic equation numbering,
equation labels/references, a custom macro configuration API, persistent
macros between expressions, or additional delimiters such as `\(...\)` and
`\[...\]`. Fences with language `math` render as display math when enabled;
`tex`, `latex`, and other code fences retain their existing meaning.
The supported expression subset is the pinned KaTeX release under Margo's
fixed configuration; this is not a full LaTeX compatibility promise.

## 2. Existing architecture and required changes

The current root pipeline is `Compile -> Render -> output projection`:

- `markdown.go` constructs the Goldmark parser without math support.
- `compiler.go` normalizes source through a source-only internal seam; math
  requires passing immutable parser configuration into normalization.
- `extension.go` and `render_plan.go` register detached fenced-block payloads.
  Inline math cannot be implemented solely through this extension mechanism.
- `render.go` owns semantic node rendering; custom nodes need explicit cases.
- `HTMLRequirements`, Muamba-managed assets, and site artifacts already supply
  the dependency and offline-delivery boundaries.
- `runtime_projection.go` currently projects Mermaid compiled tasks.
- `mermaid.go` assigns `margoRuntimeReady` from a Mermaid-specific executor.
- `pdf/chromium/engine.go` collects Mermaid SVG output and synthesizes reports
  for other task kinds; math needs its own actual execution evidence.
- `deck/parse.go` scans directives and slide boundaries before compiling each
  slide. Those scans must understand protected display-math regions.

Math is an internal core capability enabled by a compiler option. Keep the
public fenced extension contract unchanged. Introduce private math AST nodes,
compiled task data, parser configuration, and shared delimiter scanning.

## 3. Activation and public API

Proposed root API:

```go
compiler := margo.New(margo.WithMath(true))
findings, err := margo.Check(ctx, source, margo.WithCheckMath(true))
enabled := compiler.MathEnabled()
```

Both options accept a boolean and default to false. `MathEnabled` returns the
frozen configuration value, allowing deck parsing to use exactly the compiler's
mode. A nil compiler reports false. Math is a compile-time interpretation
choice; a render option cannot change it after compilation.

Add `--math` (boolean, default false) to `check`, `html`, `pdf`, `site`, `serve`,
and `deck`. `--math=false` explicitly disables it. The same resolved value must
reach preflight, compiler creation, deck scanning, development rebuilds, and
site-generated PDF actions. Example commands:

```sh
margo check guide.md --math
margo html guide.md --math --output guide.html
margo pdf guide.md --math --output guide.pdf
margo site ./docs --math --output-dir ./dist
margo serve ./docs --math
margo deck slides.md --math --output slides.html
```

Existing CLI output-selection flags remain authoritative; math adds no new
output format. Existing programmatic `site.Request.Compiler` and
`site.ConfigRequest.Compiler` carry the setting. Site YAML and document
frontmatter receive no new math activation field in v1. Configured CLI sites
use the invocation's `--math` flag. Raw HTML opt-in does not enable math or
relax math restrictions.

Preserve the existing `deck.Parse(name, source)` signature and disabled default.
Add `deck.ParseWithOptions(name, source, ...deck.ParseOption)` and
`deck.WithMath(bool)` for hosts that inspect decks without rendering.
`deck.Render` derives its parsing mode from `compiler.MathEnabled()`; callers
do not need a second setting. CLI deck preflight uses the same mode.

## 4. Delimiter grammar

### 4.1 Inline math

An inline expression uses a single unescaped dollar on each side:

- A delimiter must be a run of exactly one dollar. Runs of two or more are
  never inline delimiters.
- The opening dollar must be followed by a non-whitespace character.
- The closing dollar must be preceded by a non-whitespace character and must
  not be immediately followed by an ASCII digit.
- The payload must be nonempty and close on the same logical source line.
- An odd run of immediately preceding backslashes escapes a dollar; an even
  run does not. Delimiters are removed, but payload backslashes are preserved.
- Scan to the first unescaped dollar run on that line. It must be a single
  dollar satisfying the closing rules; otherwise the candidate opener remains
  literal. Do not skip an ineligible run to search for a later closer. A dollar
  without an eligible partner remains text without a math diagnostic.
- Once opened, the payload is opaque TeX source: Markdown emphasis, backticks,
  entities, links, and HTML are not parsed inside it.

Examples, with math enabled:

| Source | Interpretation |
| --- | --- |
| `$x^2$` | One inline expression. |
| `$2 + 2 = 4$` | One inline expression; numeric opening content is valid. |
| `Price: $20 and $30.` | Currency text; the second dollar cannot close before `3`. |
| `\$20` | Ordinary escaped dollar. |
| `$ x$`, `$x $`, `$x$2` | Ordinary Markdown text. |
| `$x` | Ordinary Markdown text. |
| `$$x$$` on its own line | Compact display block; never inline math. |
| `Text $$x$$ text`, `$$$x$$$` | Ordinary Markdown text in this profile. |

These rules reduce common currency collisions but cannot distinguish every
prose convention. Authors escape literal dollars when their text matches the
math grammar. Whitespace uses Go's `unicode.IsSpace` rune classification.
Tests must pin tokenization of adjacent expressions, punctuation, Unicode
whitespace, and backslash parity. In particular, `$x$ $y$` is two expressions;
`$x$$y$` is literal because the central run cannot close or open inline math.

### 4.2 Display math

Accept compact display blocks such as `$$E = mc^2$$` on their own line, with
up to three leading spaces and optional trailing spaces/tabs. The opening and
closing runs must each contain exactly two dollars; the first unescaped
closing run must leave only trailing spaces/tabs. Trim boundary spaces/tabs
around the compact payload. Reject a recognized whitespace-only payload as
`math.empty_expression`. Four-space indented code takes precedence. A dollar
pair in the middle of a prose line does not create a display block in v1.

A display block opens with a line containing exactly `$$`, with up to three
leading spaces and optional trailing spaces/tabs, for multiline expressions.
A matching line closes it.
Within a list or blockquote, apply this rule after normal container prefixes
and container indentation are consumed. Four-space indented code takes
precedence. The opening line may interrupt a paragraph, like a fenced block.

The body may span lines and contain blank lines. Preserve body content after
container-prefix removal, normalize CRLF to LF, and exclude the delimiter
lines. A whitespace-only body is `math.empty_expression`. An opening display
block without a closer before its Markdown container ends or EOF is
`math.delimiter_unclosed`; point to its opening dollar. Once deck regions have
been classified, each block belongs wholly to one slide/composition slot.

Inside the block, every line is TeX source, including lines spelling `---`,
headings, backtick fences, HTML comments, or deck directives. Delimiter closing
is lexical and is not affected by TeX comments or braces. Escaping a dollar or
using at least four leading spaces prevents a body line from closing the block.

### 4.3 Fenced math blocks

Accept CommonMark fenced blocks whose language is exactly lowercase `math`
as display math, including tilde fences and nested list/blockquote fences:

````markdown
```math
\begin{aligned}
a &= b + c \\
d &= e + f
\end{aligned}
```
````

The body contains an expression without additional dollar delimiters. Preserve
the fenced payload after the normal container/fence indentation handling.
Normalize line endings and apply the same empty-body, source, output, engine,
font, runtime, and publication-budget rules as dollar-delimited display blocks.
Require an explicit closing math fence; an absent closer produces
`math.delimiter_unclosed`. Do not interpret info-string suffixes as engine
options. The ordinary Markdown fence parser protects all content in the body.

When enabled, `math` is reserved by the core capability. A host extension also
claiming that fence produces configuration error `math.fence_conflict` rather
than silently shadowing either renderer. With math disabled, existing fence
ownership and code rendering remain unchanged. A `tex` or `latex` fence always
remains source code under this feature.

Lower dollar display blocks and math fences to the same internal display-math
representation. The additional authoring form adds no independent engine,
import protocol, or runtime task kind.

### 4.4 Markdown integration

Recognize inline math in paragraphs, headings, list items, blockquotes, table
cells, and link labels. Recognize display blocks in document bodies, list
items, blockquotes, slide bodies, and composition slots.

Do not recognize math in code spans, ordinary code blocks, link destinations/titles,
image destinations/alt text, frontmatter, raw HTML tokens/blocks, or comments.
Deck header/footer directive strings and presenter notes keep their existing
interpretation and do not activate math in v1.

Goldmark table splitting remains authoritative: use `\vert` or `\mid` for a
vertical bar inside a table-cell equation; an unescaped Markdown pipe still
separates cells. Do not preprocess the whole Markdown source with regexes.

Plain-text projections such as heading labels, TOC entries, search records,
and inferred descriptions use the TeX payload without delimiters. They must
not include duplicate HTML/MathML text after browser typesetting. Heading ID
generation stays deterministic using the existing slug algorithm over the
heading's source text, with golden fixtures for headings containing math.

## 5. Compiled representation and identity

Each private math node records inline/display mode, normalized payload, and
original-source spans. Record source byte offsets through frontmatter removal,
deck splitting, and composition extraction; diagnostics resolve to the
original file, line, and column using the existing diagnostic convention.

During compilation, allocate a stable math ordinal in source order and freeze
a task descriptor containing the grammar/profile version, mode, payload,
KaTeX version and asset digest, and fixed engine configuration digest.
Project its canonical input hash through the existing runtime task-ID grammar
with task kind `math`. Equal equations retain distinct occurrence identities.

The compiler fingerprint includes math activation, grammar/profile version,
limits, KaTeX asset identity, and engine configuration. Omit math-specific keys
when disabled. Existing fingerprint rules still track changes to shared
runtime dependencies, including the coordinator refactor. Existing
compiler/document mismatch checks continue to apply.

Rendering produces a stable inline `span` or block `div` containing an escaped
source fallback and private execution metadata. Source is stored as text or
safe encoded data, never interpolated into executable JavaScript. Use the
existing render-wide allocator for IDs in composed documents and decks.
Any engine-generated IDs/references must also be scoped or rejected by the
profile; engine defaults cannot create duplicate IDs across host fragments.

## 6. Engine profile and resource limits

Bundle one exact KaTeX release with its license and all required WOFF2 faces.
Record exact version, acquired URLs, checksums, and generated embedding in the
existing Muamba pipeline. Do not depend on Mermaid's transitive KaTeX chunk:
Margo needs an independently versioned engine, stylesheet, and font contract.
Select and record the exact engine pin with the profile in PR 1; acquire and
verify it in PR 2 before browser fixtures are accepted. There is no runtime
lookup of a latest version.

Margo uses KaTeX's HTML plus MathML output. Its browser API and rendering
options are documented upstream: [KaTeX API](https://katex.org/docs/api) and
[output options](https://katex.org/docs/options).

Freeze the following Margo profile:

| Setting | v1 value |
| --- | --- |
| Display mode | Derived only from the parsed node. |
| Output | `htmlAndMathml`. |
| Invalid expression handling | `throwOnError: true`; Margo catches and reports failures. |
| Strictness | `strict: "error"`. |
| Trust | Deny every trust-sensitive operation; record denied attempts as failures. |
| Macro expansion limit | `maxExpand: 1000`. |
| User-specified size ceiling | `maxSize: 20` em. |
| Macro lifetime | Fresh macro state for each expression and each execution. |

Local definitions within one expression may work through the pinned engine;
they do not persist into later equations, slides, documents, or reruns.
No source/API hook supplies executable callbacks, arbitrary engine options,
external resources, HTML attributes, labels/references, or custom CSS. Detect
trust denials explicitly: a command rendered as red unsupported text is not a
successful task. These controls follow the engine's documented boundaries:
[KaTeX security](https://katex.org/docs/security).

Enforce forbidden features after macro expansion at the engine boundary;
matching source substrings alone is insufficient. Unsupported commands may
instead fail through the engine's normal expression-error path.

Apply fixed root limits initially; no new user-tunable limit API is introduced:

| Resource | Limit |
| --- | --- |
| Payload per equation | 16 KiB of UTF-8 after line-ending normalization. |
| Equations per publication | 1,024. |
| Combined equation payload per publication | 1 MiB. |
| Generated HTML/MathML per equation | 256 KiB of UTF-8. |
| Combined generated math output per publication | 8 MiB, also bounded by remaining effective output budget. |

For a deck or composed page, publication totals include every slide and slot;
splitting must not reset the limits. Independent site pages have independent
budgets. Enforce source limits before launching engine work. Check generated
output in a detached render target before inserting it. Count dynamic output
alongside existing rendered content against the effective output ceiling;
do not treat browser mutation as an unlimited second output channel.

PDF execution obeys the existing export timeout and context cancellation.
Browser execution yields between equations; bounded engine work and source
limits constrain each synchronous call. These limits are not a claim of a
preemptible JavaScript call in an ordinary reader's browser.

## 7. Dependency delivery and HTML behavior

Emit three ordered requirements only when a compiled document contains math:

1. `margo.math.styles`: pinned CSS and font delivery.
2. `margo.math.engine`: pinned KaTeX runtime.
3. `margo.math.execute`: Margo executor after the engine and shared coordinator.

Requirement merging deduplicates matching identities across fragments and
fails on conflicting versions/content. The shared runtime coordinator is
included whenever math or Mermaid requires it, using one common identity.

Local mode writes/serves self-contained CSS with the WOFF2 resources embedded
as data URLs. Inline mode embeds the same CSS. This keeps transitive font
resources complete without adding a new public dependency-resource kind or
CSS-relative font rewriting. Respect site base paths, nested routes, and the
host's dependency mounting API. A
standalone HTML file opened through `file://` needs no asset server or network.
Build-time acquisition may use the network; published artifacts do not.

Render only Margo-generated math markers. Do not run KaTeX's document-wide
auto-render extension over arbitrary host text, Mermaid output, or raw HTML.
An embedded host can explicitly rerun the executor after mounting new Margo
content. Execution is scoped by registered render instance, idempotent for
completed input, and does not install a document-wide mutation observer.

Until successful typesetting, escaped TeX remains readable. After success,
expose the MathML representation and hide the redundant source fallback from
visual display and the accessibility tree. If JavaScript is unavailable, the
fallback remains. HTML v1 therefore requires JavaScript for typeset equations;
it does not promise pre-rendered math or accessible math structure without it.
PDF v1 does not claim tagged, accessible mathematical PDF output.

On runtime failure, keep source visible and show an adjacent accessible error
message; do not rely only on color, hover text, or the console. Assign source
and errors through text APIs. Upstream documents why exception text must be
escaped: [KaTeX error handling](https://katex.org/docs/error).

## 8. Runtime completion and export evidence

Replace per-feature assignment of `margoRuntimeReady` with a shared coordinator
used by math and Mermaid. Register all expected tasks before publishing a
readiness snapshot. The snapshot represents every registered task in that
execution; one executor completing must not overwrite another executor's
pending or failed state. A rerun creates a new execution snapshot. Theme-driven
Mermaid rerenders participate in the same contract.

Each math task runs through pending, running, and succeeded/failed states.
Success requires a valid engine result, output-size checks, mounted HTML plus
MathML, and positively verified bundled fonts. Request/load the required font
faces after typesetting, await `document.fonts.ready`, and verify the required
faces loaded from the bundle. A resolved fonts-ready promise or a fallback
glyph alone is insufficient. Include faces needed by initially hidden slides.

Reuse runtime protocol v1 for ordinary documents and v2 for deck validation;
the existing task-kind grammar accepts `math`. No wire-format change is
required. Emit existing `FontCheck` evidence and keep deck font-bundle identity
consistent with the added math dependencies. The effective deck font-bundle
digest includes math fonts when math is present; a supplied validation request
that pins a different bundle fails explicitly.

The Chromium adapter must collect outputs keyed by task identity instead of
assuming one positional Mermaid SVG array. Math evidence is the UTF-8 KaTeX
HTML/MathML string returned by `renderToString`, captured before insertion
into Margo's wrapper. Hash/count those exact bytes; do not synthesize success
from layout dimensions, source text, or task kind. Mermaid retains SVG
evidence. Every expected math task must have exactly one matching result.

Ordering is typesetting -> required-font checks -> layout/overflow measurement
-> print. Deck layout tasks depend on every composed content task. PDF export
fails before publication on missing, duplicate, failed, mismatched, or timed
out math tasks, missing fonts, blocked external requests, or output-limit
violations. No partial PDF is published and no engine fallback occurs.

Preserve per-task failure evidence in failed runtime reports. Unknown task kinds
must not receive synthesized success. Existing supported layout task handling
remains explicit. Root HTML generation and ordinary site building do not launch
Chromium to prove equation validity; their success means structurally valid
artifacts were built. Site-requested PDF actions still validate each exported
PDF. Browser failures remain visible to the reader. Strict expression validation
happens during browser execution and is mandatory for PDF export. Existing
CLI deck HTML output does not invoke Chromium, so math preserves that behavior.

## 9. Deck scanning and layout

Share delimiter classification between Markdown parsing and deck scanning.
With math enabled, every pre-compilation scan must protect display bodies:
global directive discovery, thematic/heading slide division, notes, references,
composition syntax, and slot/layout boundaries. Classify display-math spans
before interpreting deck syntax. A line inside a closed display block is TeX
even if it resembles a slot boundary or slide separator. Deck scanning resumes
after its closing delimiter. An unclosed block errors before splitting; no
partially interpreted deck is returned. Ordinary Markdown container termination
still applies. This rule deliberately makes the delimiter, rather than an
apparent directive inside its body, authoritative.

`deck.Parse` retains its old interpretation. `deck.ParseWithOptions` and
`deck.Render` with math enabled must agree on slide count, extracted source,
and original-source error locations.

Inline equations follow body typography without shrinking the whole paragraph.
Display equations have theme-compatible spacing and inherit text color.
Constrain CSS overrides to Margo-owned wrappers so KaTeX's font metrics survive
the Goshtoso prose styles. Check Modern, Goshtoso, and Minimal in both modes.

On HTML pages, wide display equations scroll locally with keyboard access and
an accessible region label when overflow actually occurs. Print and deck
projections remove scrolling/clipping. An overwide or overheight equation in
PDF fails with `math.print_overflow`; in decks it participates in existing
terminal deck overflow validation. Do not silently scale down or crop math.
Keep a display equation together on a printed page whenever it fits.

## 10. Check and diagnostic contract

`margo.Check(..., WithCheckMath(true))` and `margo check --math` remain read-only
Go preflight. They use the same node grammar, positions, source limits, and
deck protection rules as compilation. They do not emulate KaTeX in Go, inspect
balanced TeX braces heuristically, or claim complete expression validation.
An unmatched inline dollar remains ordinary text.

When recognized math is present, emit one warning per input,
`check.math_runtime_validation_required`, explaining that expression validity
and fonts are checked in the browser/PDF execution. Warning semantics and exit
status follow existing Check behavior. No `--math` means no math diagnostics.

| Diagnostic | Stage | Meaning |
| --- | --- | --- |
| `math.delimiter_unclosed` | Compile/check/deck parse | Display opener or math fence has no closer in its container. |
| `math.empty_expression` | Compile/check | Display payload contains only whitespace. |
| `math.fence_conflict` | Configuration | Enabled core math and a host extension both claim the math fence. |
| `math.expression_too_large` | Compile/check | One normalized payload exceeds 16 KiB. |
| `math.document_limit_exceeded` | Compile/check/composition | Equation count or combined payload exceeds publication limits. |
| `math.expression_invalid` | Browser/export | Engine parse failure or strict-mode rejection. |
| `math.command_forbidden` | Browser/export | Trust-sensitive operation or a profile-forbidden feature was attempted. |
| `math.output_limit_exceeded` | Browser/export | Per-equation, combined, or effective output limit exceeded. |
| `math.runtime_unavailable` | Browser/export | Bundled engine or required execution capability is unavailable. |
| `math.font_unavailable` | Browser/export | A required pinned font face did not load. |
| `math.print_overflow` | PDF | Equation exceeds the printable content box. |
| `check.math_runtime_validation_required` | Check warning | Go preflight cannot validate typesetting. |

Reuse existing runtime identity, report, timeout, blocked-request, and deck
overflow diagnostics for those failures. Expression errors include original
source positions; translate an engine offset only when it is available from
the pinned release through a verified mapping, otherwise point to the opening
delimiter. Do not invent precise offsets from an error-message substring.

## 11. Acceptance matrix

Each row requires executable evidence in the indicated layer. Browser and PDF
checks belong in the repository's existing harness and required CI path.

| Scenario | Required result/evidence |
| --- | --- |
| Math disabled, including dollar-heavy existing fixtures | Existing Markdown semantics and task kinds preserved; no math assets/tasks. Shared-runtime identity changes follow existing fingerprint rules. |
| Enabled with no recognized math | No KaTeX assets/tasks; Go projection checks. |
| Inline delimiters, currency, escaping, Unicode whitespace, adjacent equations | Exact tokenization and preserved payload. Go parser corpus. |
| Display interruption, CRLF, blank lines, list/quote containers | Exact payload and source positions. Go parser corpus. |
| Compact `$$expression$$` on a line, prose-embedded double dollars, triple dollars | Standalone compact display accepted; prose/triple forms retain their documented literal behavior. |
| Backtick/tilde math fences, nesting, empty/unclosed fences, conflicting host ownership | One display-math representation with consistent diagnostics and budgets; explicit conflict failure. |
| Unclosed, empty, and oversized display math | Matching Compile/Check diagnostics and original locations. |
| Code, HTML, image alt, link targets, frontmatter, notes | Protected content receives no math execution. Go/browser fixtures. |
| Tables and links containing equations | Correct cell/label semantics; TeX pipe rule documented and exercised. |
| Heading math, TOC, search, metadata | Deterministic IDs and one plain-source projection; no duplicate math text. |
| Fractions, roots, integrals, matrices, aligned expressions, common Greek symbols | Valid HTML plus MathML and bundled font use. Browser/PDF corpus. |
| Invalid command, invalid syntax, forbidden inclusion/HTML/link/reference features | Failed task and visible escaped source/error; PDF rejection. |
| Script-like payload and error messages | No executable injection or unexpected external requests. Browser checks. |
| Macro loops and generated-output limits | Bounded failure and no successful PDF. |
| Oversized dimensions | Engine size ceiling applies; remaining print/deck overflow fails rather than cropping. |
| Local macro definition followed by another expression/document/rerun | No macro state leakage. Browser checks. |
| Equal expressions across fragments/slides/slots | Distinct task occurrences, scoped IDs, one shared dependency set. |
| Math and Mermaid with controlled completion/failure order | Readiness waits for both and preserves failures. Browser/export integration. |
| Duplicate, missing, wrong-input, or unknown task result | Export rejects evidence and publishes no PDF. Runtime integration. |
| Artificially delayed or broken math font, including hidden slides | No early ready/layout/print; missing font fails. Browser/export integration. |
| Standalone `file://`, local/inline site assets, nested site base path | Typeset math with all network access blocked. Browser integration. |
| Math body containing rulers, headings, directive comments, slot syntax | Stable deck state and slide count, or deterministic unclosed-container error. |
| Wide equation on mobile, in PDF, and in a slide | Accessible HTML scrolling; print/deck overflow rejection; no cropping. |
| JavaScript disabled or engine bundle missing | Readable fallback; missing-engine execution reports failure. |
| Theme/mode matrix at 390 px and 1440 px | Readable baseline/spacing, visible focus, usable overflow, no console errors on valid corpus. |
| Combined limits spread across slides/slots | Publication-wide budget holds across compilation boundaries. |
| Canceled or timed-out PDF export | Browser cleanup, failed operation, and no committed artifact. |
| Equivalent CLI and Go settings | Equivalent semantic nodes, dependency requirements, and runtime task inputs. |

The release milestone is one mixed math/Mermaid corpus exercising standalone
HTML, embedded fragments, local/inline sites, document PDF, deck HTML, and deck
PDF entirely offline. HTML accessibility is checked in the DOM/accessibility
tree; mathematical accessibility of exported PDF is outside the v1 guarantee.

## 12. Delivery sequence

### PR 1: Grammar and compilation

Add immutable parser configuration, activation options/flags, math nodes,
compact/multiline dollar display parsing and fenced-math normalization,
the exact engine/profile identity, source mapping, fixed source limits,
task inputs/fingerprints, Check findings,
plain-text projection rules, and math-aware deck pre-scans. Include the parser
and deck protection corpora. Do not advertise complete math output yet.

### PR 2: Bundled engine and HTML/site integration

Pin/acquire KaTeX and fonts, implement conditional dependencies and transitive
font artifacts, safe placeholders, the fixed engine profile, HTML plus MathML,
fallback/errors, output budgets, accessible overflow, and shared runtime
coordination with Mermaid. Cover `file://`, composed hosts, nested sites,
reruns, and the theme/mode matrix.

### PR 3: Export evidence and release documentation

Collect task-keyed engine evidence, verify math fonts, enforce completion before
measurement/printing, integrate deck font identity and overflow, and cover
terminal failures and atomic publication. Update README, CLI help, public API
docs, Check documentation, security guidance, and offline dependency examples.
Run the existing required pipeline plus the new math browser/PDF cases.

PRs may land incrementally behind the disabled default. Full LaTeX math
support is announced only when every output in the release milestone passes.

## 13. Decisions reserved for later work

Server-side pre-rendering would improve first paint and no-JavaScript output,
but requires a separate execution/dependency design for the Go library and
portable binary. It is not required for this browser-rendered v1.

Additional delimiter dialects, declarative site activation, configured or
persistent macros, chemical notation extensions, labels/cross-references,
tagged mathematical PDF, and native PDF engine support require separate
contracts and fixtures. Record the exact KaTeX release pin in PR 1 and measured
bundle-size cost in PR 2; they do not reopen the chosen browser architecture.

## 14. Renderer comparison and scope rationale

Reviewed official documentation on 2026-11-05. This is a targeted comparison
of authoring features, not a claim that implementations share identical
parsing, defaults, output architecture, or extension capabilities.

| Markdown-focused tool | Documented inline math | Documented display math | Source |
| --- | --- | --- | --- |
| GitHub | Dollar delimiters and a dollar/backtick variant. | Double dollars, including compact blocks, and math fences. | [GitHub math guide](https://docs.github.com/en/get-started/writing-on-github/working-with-advanced-formatting/writing-mathematical-expressions) |
| GitLab | Dollars and a dollar/backtick variant; KaTeX subset. | Double dollars and math fences. | [GitLab math equations](https://docs.gitlab.com/user/markdown/#math-equations) |
| Docusaurus | Dollars through its math plugin configuration. | Math fences and multiline double dollars. | [Docusaurus equations](https://docusaurus.io/docs/markdown-features/math-equations) |
| VitePress | Dollars; explicitly opt-in. | Double dollars; compact form shown. | [VitePress equations](https://vitepress.dev/guide/markdown#math-equations) |
| Marp | Dollars using Pandoc-style authoring. | Double dollars, compact and multiline. | [Marp math typesetting](https://marp.app/docs/guide/math-typesetting) |
| Material for MkDocs with Arithmatex | Dollars and parenthesized backslash delimiters. | Double dollars and additional configurable block forms. | [Material configuration](https://squidfunk.github.io/mkdocs-material/reference/math/) and [Arithmatex grammar](https://facelessuser.github.io/pymdown-extensions/extensions/arithmatex/) |
| Hugo | Configurable delimiters; its example prefers parenthesized backslashes to avoid currency ambiguity. | Configurable double dollars/bracketed backslashes. | [Hugo mathematics](https://gohugo.io/content-management/mathematics/) |

All seven document inline and display math within Markdown. Three explicitly
document math fences, making fences a useful compatibility addition rather
than a universal requirement. Compact double-dollar display syntax also has
clear precedent. These observations support the three-form Margo profile:
single-dollar inline, double-dollar display (compact or multiline), and fenced
`math` display. The `.md` document remains the unit of authoring and publication.

Standalone `.tex` input and importing equation snippets are absent from these
guides' core math contracts. This is a documentation observation, not proof that
their extensible platforms cannot include external files. Pandoc separately
documents LaTeX input as part of a broader document-converter architecture:
[Pandoc input formats and Markdown math](https://pandoc.org/MANUAL.html).
That broader role is not required to deliver Margo's Markdown math feature.

Reading one expression from a file is technically small, but a useful Markdown
snippet-import feature also needs include syntax, resource resolution,
dependency tracking, rebuild invalidation, and source diagnostics. Defer it
until an authoring use case justifies that contract; if added, design it as
reusable local content inclusion rather than treating `.tex` as a document
format or exposing TeX's `\input` command.

The surveyed tools differ on whitespace/blank lines, dollar ambiguity,
backslash delimiters, prose-embedded double dollars, engine options, and macro
persistence. Preserve the exact rules in section 4 and test published
cross-tool examples; do not advertise complete GitHub/Pandoc compatibility.
In particular, Margo accepts blank lines inside multiline math, reserves double
dollars for display blocks, and isolates macro state by expression. Academic
systems can go further: [MyST](https://myst-parser.readthedocs.io/en/latest/syntax/math.html)
adds equation labels/references, which remain a separate publishing feature.

## 15. Implementation validation

Validated locally on 2026-11-05:

- Full Go suite: `go test -p 2 ./... -count=1`, including Chromium rendering,
  PDF exports, offline assets, mixed Mermaid/math, table sorting, deck source
  locations, and navigation/recovery in both documentation shells.
- Focused race checks for concurrent compilation/rendering, parser grammar,
  source limits, and deck math.
- `go vet ./...` and a CLI build with `CGO_ENABLED=0`.
- `go generate ./...`, Muamba asset registry generation, and
  `go tool muamba verify katex` for all 23 pinned files.
- JavaScript syntax checks and `git diff --check`.
- CLI smoke exports for standalone HTML, site HTML, PDF, deck HTML, and deck
  PDF; both PDF outputs had valid PDF signatures and nonempty contents.

The full suite used a browser-readable `TMPDIR` because the installed Snap
Chromium cannot access the host's default temporary directory. These are local
validation results; the container CI pipeline has not been run.

## 16. Implementation review and visual inspection

Reviewed the implementation against sections 3–11 on 2026-11-05. The following
findings were reproduced and corrected in the working tree:

| Priority | Finding | Correction and regression evidence |
| --- | --- | --- |
| P1 | Valid slides showed an overflow alert because hidden MathML, positioned glyph metrics, and clipped radical SVG paths were treated as visible content. | Measure visible math bounds, ignore the accessible duplicate and SVG path internals, and revalidate after runtime completion. `TestMathDeckScreenIgnoresAccessibleDuplicate` covers desktop/mobile navigation and still rejects real visual overflow. |
| P1 | A math opener immediately after a deck slot marker could disappear into CommonMark's HTML-block classification, allowing slide/slot syntax inside the equation to escape protection. | Discover math spans with deck comments neutralized, then retain original content. `TestMathDeckProtectsBlocksImmediatelyAfterSlotMarkers`. |
| P1 | Print validation could miss tall inline equations and equations clipped by a slide or scrolling ancestor. | Measure intrinsic height and clipping bounds; reject before printing and preserve task evidence with original source diagnostics. `TestMathPDFEvidenceAndFailures/inline-height` and `TestMathDeckPrintRejectsClippedInlineEquation`. |
| P2 | Payload matching failed to map deck footnotes and could confuse repeated equations or trimmed slot columns. | Retain scanner-owned source positions through extraction and defensive copies; leave public layout structs unchanged. `TestMathDeckRemapsReorderedAndRepeatedEquations`. |
| P2 | Whole-file Markdown classification could miss math source budgets spread across structural slots. | Validate actual fragments with a shared publication accumulator. `TestMathDeckSourceBudgetIncludesAllSlots`. |
| P2 | Opaque math parsing changed image alt formatting and could consume an image's closing bracket. | Protect image-label source ranges before parsing math using the ordinary Markdown grammar. `TestMathImageAltRetainsOrdinaryMarkdown`. |
| P2 | Display equations were absent from search and description plain-text projections. | Count display-math wrappers as semantic text while retaining delimiter-free source. `TestMathPlainTextIncludesAllFormsOnce`. |
| P2 | Footnote relocation assigned math task ordinals in DOM order rather than source order. | Freeze source-ordered tasks and collect browser evidence by occurrence identity within each composed article. `TestMathOrdinalsFollowSourceAfterFootnoteRelocation` and PDF footnote coverage. |
| P2 | Failure messages had insufficient contrast in dark mode; long invalid expressions could overflow mobile viewports. | Add readable dark error colors and source/error wrapping, with browser contrast and viewport checks. |
| P2 | Browser tests could silently skip in the browser-free Go CI container, and pinned KaTeX verification was only a manual check. | Add a mandatory host-browser step to required CI that rejects missing/skipped math coverage, and verify locked KaTeX assets in the Dagger required script. |

Visual inspection covered Modern, Goshtoso, and Minimal in light/dark at 390 px
and 1440 px, desktop/mobile documentation pages, all three mixed math/Mermaid
slides, document/deck PDF rasterizations, the no-JavaScript source fallback,
and visible invalid/forbidden-expression failures. The corrected valid fixtures
had readable baselines, intact fractions/radicals/matrices/aligned expressions,
and no false slide overflow alerts. Wide HTML display math retained local
keyboard scrolling; oversized print cases were rejected.

The full local Go suite, focused race checks, vet, asset verification, generated
assets, CLI document/deck PDF exports, Dagger module unit tests, and workflow
syntax checks passed. The new required browser selection was also exercised
locally with no skipped math tests. GitHub Actions and the Dagger container
pipeline remain to be run by CI.
