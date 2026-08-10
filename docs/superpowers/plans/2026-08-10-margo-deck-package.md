# Margo Deck Package Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a consumer-neutral `deck` package that turns Margo Markdown into accessible standalone HTML with one printable page per slide.

**Architecture:** Deck parsing is a small line-state machine that preserves opening YAML frontmatter and splits only exact `---` lines outside fenced code. Rendering compiles each slide through the caller-provided Margo compiler, wraps immutable HTML projections in semantic sections, and owns only deck navigation and print behavior.

**Tech Stack:** Go 1.26.5, Margo root APIs, templ, embedded CSS and JavaScript, standard `testing` package

## Global Constraints

- `deck` imports the Margo root package; it does not import `pdf` or CLI packages.
- Slide input supports ordinary Margo Markdown, Mermaid, tables, code, images, and caller-registered charts.
- Only one optional opening YAML frontmatter block is deck metadata; an exact `---` outside fenced code splits slides.
- Empty slides are rejected; separators inside backtick or tilde fenced code are preserved.
- Deck HTML uses an article with accessible slide sections, previous/next controls, ArrowLeft, ArrowRight, Home, and End.
- Print CSS maps exactly one slide to one page and hides screen-only controls.
- The first release does not execute Marpit directives, arbitrary CSS, presenter notes, transitions, fragments, or plugins.
- HTML generation requires no browser and never downloads runtime tooling.
- Every gate runs with `GOWORK=off GOFLAGS=-mod=readonly`.
- Do not push, merge, tag, release, or publish while executing this plan.

---

## File structure

- `deck/model.go`: immutable `Document`, `Slide`, and `Metadata` values.
- `deck/parse.go`: frontmatter/fence-aware slide splitter.
- `deck/parse_test.go`: parser boundaries and diagnostics.
- `deck/render.go`: public render API, per-slide core composition, and merged runtime projection.
- `deck/page.go`: semantic shell assembly.
- `deck/assets/deck.css`, `deck/assets/deck.js`, `deck/assets.go`: embedded presentation behavior.
- `deck/render_test.go`: semantic, navigation, print, charts, Mermaid, and image tests.

### Task 1: Parse opening metadata and exact slide separators

**Files:**
- Create: `deck/model.go`
- Create: `deck/parse.go`
- Create: `deck/parse_test.go`
- Modify: `deck/doc.go`

**Interfaces:**
- Consumes: `[]byte` Markdown and a source name.
- Produces: `func Parse(name string, source []byte) (*Document, error)`, `func (d *Document) Metadata() Metadata`, `func (d *Document) Slides() []Slide`, `func (s Slide) Ordinal() int`, and `func (s Slide) Markdown() []byte`.

- [ ] **Step 1: Write failing parser tests**

```go
func TestParseSplitsOnlyExactSeparatorsOutsideFences(t *testing.T) {
	source := []byte("---\ntitle: Demo\n---\n# One\n\n```text\n---\n```\n---\n# Two\n")
	doc, err := Parse("demo.md", source)
	if err != nil { t.Fatal(err) }
	if got, want := doc.Metadata().Title, "Demo"; got != want { t.Fatalf("title = %q", got) }
	slides := doc.Slides()
	if len(slides) != 2 { t.Fatalf("slides = %d", len(slides)) }
	if !bytes.Contains(slides[0].Markdown(), []byte("```text\n---\n```")) { t.Fatal("fenced separator was removed") }
	if slides[0].Ordinal() != 1 || slides[1].Ordinal() != 2 { t.Fatal("unstable ordinals") }
}

func TestParseRejectsEmptySlides(t *testing.T) {
	_, err := Parse("empty.md", []byte("# One\n---\n  \n"))
	if got := diagnosticCode(err); got != "deck.slide_empty" { t.Fatalf("code = %q", got) }
}
```

- [ ] **Step 2: Run the parser tests and verify missing API failure**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./deck -run 'TestParse' -count=1`

Expected: FAIL because `Parse` and the model types do not exist.

- [ ] **Step 3: Implement the immutable model**

```go
type Metadata struct { Title, Description string }
type Slide struct { ordinal int; markdown []byte }
type Document struct { name string; metadata Metadata; slides []Slide }

func (s Slide) Ordinal() int { return s.ordinal }
func (s Slide) Markdown() []byte { return append([]byte(nil), s.markdown...) }
func (d *Document) Metadata() Metadata { if d == nil { return Metadata{} }; return d.metadata }
func (d *Document) Slides() []Slide {
	if d == nil { return nil }
	result := make([]Slide, len(d.slides))
	for i, slide := range d.slides { result[i] = Slide{ordinal: slide.ordinal, markdown: slide.Markdown()} }
	return result
}
```

- [ ] **Step 4: Implement the line-state parser**

Track opening YAML separately, recognize matching ````` or `~~~` fence markers with at least three marker bytes, and split only when `line == "---"` after removing `\r`. Return `deck.frontmatter_invalid`, `deck.fence_unclosed`, or `deck.slide_empty` diagnostics with source line numbers.

- [ ] **Step 5: Run parser and race tests**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test -race ./deck -run 'TestParse' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit the parser**

```bash
git add deck/doc.go deck/model.go deck/parse.go deck/parse_test.go
git commit -m "feat(deck): parse markdown slides"
```

### Task 2: Render semantic slide fragments through the core compiler

**Files:**
- Create: `deck/render.go`
- Create: `deck/render_test.go`

**Interfaces:**
- Consumes: `func (*margo.Compiler) Compile(context.Context, margo.Source) (*margo.Document, error)`, `func (*margo.Compiler) Render(context.Context, *margo.Document, ...margo.RenderOption) (*margo.RenderResult, error)`, and `func margo.RenderHTML(*margo.RenderResult, ...margo.HTMLOption) (*margo.HTMLResult, error)`.
- Produces: `type RenderInput struct { Name string; Markdown []byte; Theme margo.ThemeName; ColorMode margo.ColorMode }`, `type Result`, `func Render(context.Context, *margo.Compiler, RenderInput) (*Result, error)`, `func (r *Result) HTML() []byte`, `func (r *Result) SlideCount() int`, `func (r *Result) DocumentFingerprint() margo.DocumentFingerprint`, and `func (r *Result) RuntimeDescriptor(margo.RenderInstanceID) (margo.RuntimeDescriptor, error)`.

- [ ] **Step 1: Write a failing semantic render test**

```go
func TestRenderProducesAccessibleSections(t *testing.T) {
	compiler := margo.New()
	result, err := Render(context.Background(), compiler, RenderInput{Name: "deck.md", Markdown: []byte("# One\n---\n# Two\n")})
	if err != nil { t.Fatal(err) }
	html := string(result.HTML())
	for _, fragment := range []string{`<article class="margo-deck"`, `role="region"`, `aria-label="Slide 1 of 2"`, `aria-label="Slide 2 of 2"`, `<h1>One</h1>`, `<h1>Two</h1>`} {
		if !strings.Contains(html, fragment) { t.Fatalf("HTML missing %q", fragment) }
	}
	if result.SlideCount() != 2 { t.Fatalf("count = %d", result.SlideCount()) }
}
```

- [ ] **Step 2: Run and verify the render API is absent**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./deck -run TestRenderProducesAccessibleSections -count=1`

Expected: FAIL because `Render`, `RenderInput`, and `Result` do not exist.

- [ ] **Step 3: Implement per-slide composition**

For each parsed slide, compile `margo.Source{Name: fmt.Sprintf("%s#slide-%d", input.Name, ordinal), Content: slide.Markdown()}`, render it, convert with `margo.RenderHTML`, and render the component into a private buffer. Build the final document only after every slide succeeds; copy final bytes into `Result`. The private slide instance IDs are `ri-` plus the zero-based slide ordinal padded to eight base-36 characters.

```go
type RenderInput struct {
	Name string
	Markdown []byte
	Theme margo.ThemeName
	ColorMode margo.ColorMode
}

type Result struct {
	html []byte
	slideCount int
	fingerprint margo.DocumentFingerprint
	slideResults []*margo.RenderResult
}
func (r *Result) HTML() []byte { if r == nil { return nil }; return append([]byte(nil), r.html...) }
func (r *Result) SlideCount() int { if r == nil { return 0 }; return r.slideCount }
func (r *Result) DocumentFingerprint() margo.DocumentFingerprint { if r == nil { return margo.DocumentFingerprint{} }; return r.fingerprint }
```

Compute the deck fingerprint as `sha256("margo/deck/v1\n" + source)` converted to `margo.DocumentFingerprint`. Retain immutable per-slide render results. `RuntimeDescriptor(instance)` projects each slide with its private valid instance and calls `margo.ComposeRuntimeDescriptors(r.fingerprint, instance, descriptors...)`; nil results return `deck.result_required`.

- [ ] **Step 4: Add runtime composition, failure atomicity, and nil-compiler cases**

Add a two-slide Mermaid test that calls `result.RuntimeDescriptor("ri-00000042")`, expects two rebased tasks and `descriptor.DocumentFingerprint == result.DocumentFingerprint()`, and passes `margo.ValidateRuntimeDescriptor`. Also assert `deck.compiler_required` for a nil compiler and no `Result` when the second slide contains an invalid Mermaid block.

- [ ] **Step 5: Run focused and package tests**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test -race ./deck -count=1`

Expected: PASS.

- [ ] **Step 6: Commit semantic rendering**

```bash
git add deck/render.go deck/render_test.go
git commit -m "feat(deck): render semantic slide content"
```

### Task 3: Embed navigation and one-slide-per-page behavior

**Files:**
- Create: `deck/assets/deck.css`
- Create: `deck/assets/deck.js`
- Create: `deck/assets.go`
- Create: `deck/page.go`
- Modify: `deck/render.go`
- Modify: `deck/render_test.go`

**Interfaces:**
- Consumes: ordered rendered slide fragments from Task 2.
- Produces: complete offline HTML whose only client behavior is deck navigation.

- [ ] **Step 1: Write failing behavior-contract tests**

```go
func TestRenderEmbedsNavigationAndPrintContract(t *testing.T) {
	result := mustRenderDeck(t, "# One\n---\n# Two\n")
	html := string(result.HTML())
	for _, value := range []string{"data-margo-deck-previous", "data-margo-deck-next", "ArrowLeft", "ArrowRight", "Home", "End", "@media print", "break-after: page", "window.print"} {
		if !strings.Contains(html, value) { t.Fatalf("HTML missing %q", value) }
	}
}
```

- [ ] **Step 2: Run and verify missing assets fail the test**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./deck -run TestRenderEmbedsNavigationAndPrintContract -count=1`

Expected: FAIL because navigation and print assets are absent.

- [ ] **Step 3: Implement minimal navigation**

`deck.js` must keep one zero-based index, set `hidden` and `aria-current="page"`, focus the newly active section, clamp previous/next, and map only `ArrowLeft`, `ArrowRight`, `Home`, and `End`. It must not fetch or import any resource.

- [ ] **Step 4: Implement print CSS**

`deck.css` must apply `break-after: page` and `page-break-after: always` to all slides except the last, remove `hidden` during print, hide controls, and avoid fixed screen dimensions in print media.

- [ ] **Step 5: Run package tests**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test -race ./deck -count=1`

Expected: PASS.

- [ ] **Step 6: Commit deck behavior**

```bash
git add deck/assets/deck.css deck/assets/deck.js deck/assets.go deck/page.go deck/render.go deck/render_test.go
git commit -m "feat(deck): add accessible navigation and print layout"
```

### Task 4: Prove images, Mermaid, and opt-in charts survive deck composition

**Files:**
- Create: `deck/integration_test.go`

**Interfaces:**
- Consumes: `charts.Extension()` registered through `margo.WithExtension` and the public `deck.Render` API.
- Produces: cross-package unit/E2E proof without browser or PDF tooling.

- [ ] **Step 1: Add integration fixtures and assertions**

Test one two-slide source containing PNG, JPEG, WebP, GIF, SVG, Mermaid, a table, fenced Go code, and one `goshtosochart`. Construct the compiler with `margo.New(margo.WithExtension(charts.Extension()))`. Assert the HTML contains five `<img` references with unchanged relative URLs, one `margo-mermaid`, one semantic table, highlighted code markup, and the chart's static SVG/accessibility table.

- [ ] **Step 2: Run deck integration tests**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./deck -run 'TestRenderPreservesPopularImagesAndExtensions' -count=1`

Expected: PASS.

- [ ] **Step 3: Run the full repository gate**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1 && git diff --check`

Expected: PASS.

- [ ] **Step 4: Commit integration proof**

```bash
git add deck/integration_test.go
git commit -m "test(deck): cover images diagrams and charts"
```

### Task 5: Deck checkpoint

**Files:**
- Verify: `deck/` and unchanged root package boundaries.

**Interfaces:**
- Consumes: Tasks 1-4.
- Produces: an independently reviewable deck package.

- [ ] **Step 1: Run final gates and record identity**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test -race ./deck -count=1 && GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1 && git diff --check && git status --short --branch && git rev-parse HEAD HEAD^{tree}`

Expected: all gates pass and the worktree is clean at the recorded HEAD/tree.
