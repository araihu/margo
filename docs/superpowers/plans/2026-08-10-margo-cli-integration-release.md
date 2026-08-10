# Margo CLI Integration and Release Verification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire HTML, PDF, deck, doctor, and version into one `margo` binary and produce release-ready verification and documentation without publishing a release.

**Architecture:** Commands share input, diagnostics, engine flags, and artifact publication adapters but own their individual output rules. The CLI composes root + charts + deck + PDF packages, spools every complete artifact, and publishes through existing atomic file/stdout sinks; black-box tests validate the installed command surface.

**Tech Stack:** Go 1.26.5, Cobra v1.10.2, Margo core/charts/deck/PDF packages, chromedp-backed E2E, Alpine musl container, GitHub Actions

## Global Constraints

- One binary named `margo` exposes `html`, `pdf`, `deck`, `doctor`, and `version`.
- Input is exactly one path or `-`; no batch or glob input in the first release.
- `html` defaults to stdout; `deck` defaults to HTML/stdout; PDF and deck PDF require explicit `--output`, while explicit `--output -` allows binary stdout.
- Diagnostics always use stderr and support exactly `--diagnostics text|json`; artifact stdout is never mixed with diagnostics.
- Existing destinations are refused unless `--force` is set.
- CLI registers charts; library consumers remain opt-in.
- Shared engine flags are `--engine auto|chromium|native` and `--engine-path PATH`.
- Discovery may fall back; rendering may not. Margo performs no browser/runtime downloads.
- Future installation is `go install github.com/araihu/margo/cmd/margo@vX.Y.Z` from one root tag.
- README must state the capability matrix, tested Chromium version evidence, CGO/musl consequences, and historical submodule migration.
- PDF visual quality is approved by a human; automation covers HTML semantics, PDF structure, runtime completion, and provenance.
- Every Go gate runs with `GOWORK=off GOFLAGS=-mod=readonly`.
- This plan does not authorize push, merge, tag, release, or publication.

---

## File structure

- `cmd/margo/io.go`: one-input reader and spool-to-sink publisher.
- `cmd/margo/resources.go`: local image materialization for standalone artifacts.
- `cmd/margo/diagnostics.go`: text/JSON stderr projection.
- `cmd/margo/compiler.go`: CLI compiler with chart registration.
- `cmd/margo/html.go`, `pdf.go`, `deck.go`, `doctor.go`: real subcommands.
- `cmd/margo/engine_flags.go`: shared engine options.
- `cmd/margo/*_test.go`: in-process command tests.
- `cmd/margo/e2e_test.go`: black-box binary tests.
- `README.md`: first-contact product and support documentation.
- `.github/workflows/ci.yml`: command E2E and portable build gates.

### Task 1: Add shared input, diagnostics, and atomic publication adapters

**Files:**
- Create: `cmd/margo/io.go`
- Create: `cmd/margo/diagnostics.go`
- Create: `cmd/margo/io_test.go`
- Create: `cmd/margo/diagnostics_test.go`
- Modify: `cmd/margo/root.go`

**Interfaces:**
- Consumes: `margo.NewSpool`, `margo.AtomicFileSink`, `margo.StdoutSink`, Cobra streams, and `type SourceReader interface { ReadFile(string) ([]byte, error) }`.
- Produces: `func readInput(context.Context, SourceReader, io.Reader, string) (margo.Source, error)`, `func publish(context.Context, []byte, outputOptions, io.Writer) (margo.CommitResult, error)`, `type diagnosticFormat string`, and `func writeDiagnostic(io.Writer, diagnosticFormat, error) error`.

- [ ] **Step 1: Write failing stream and destination tests**

```go
func TestPublishRefusesExistingDestinationWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "page.html")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil { t.Fatal(err) }
	_, err := publish(context.Background(), []byte("new"), outputOptions{Path: path}, io.Discard)
	if got := diagnosticCode(err); got != "margo.atomic.destination_exists" { t.Fatalf("code = %q", got) }
	if got := mustRead(t, path); got != "old" { t.Fatalf("destination changed to %q", got) }
}

func TestJSONDiagnosticWritesOneObjectToStderr(t *testing.T) {
	var out bytes.Buffer
	err := writeDiagnostic(&out, diagnosticJSON, errors.New("pdf.engine_unavailable: no engine"))
	if err != nil { t.Fatal(err) }
	if got := out.String(); got != "{\"code\":\"pdf.engine_unavailable\",\"message\":\"no engine\"}\n" { t.Fatalf("got %q", got) }
}
```

- [ ] **Step 2: Run and verify missing adapters**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./cmd/margo -run 'TestPublish|TestJSONDiagnostic' -count=1`

Expected: FAIL because the adapter functions/types do not exist.

- [ ] **Step 3: Implement complete-before-publish behavior**

Write bytes into a private `margo.Spool`, use its digest, then select `margo.StdoutSink{Writer: stdout}` for `Path == "-"` or `&margo.AtomicFileSink{Target: Path, Force: Force}` otherwise. Never write diagnostics from `publish`.

Add `SourceReader` to `Dependencies`; production uses `os.ReadFile` through a one-method adapter, and tests use an in-memory implementation. `readInput` maps `-` to the injected stdin, checks context while reading, and sets `margo.Source.Name` to `<stdin>` or the exact input path.

- [ ] **Step 4: Implement stable diagnostic projection**

Accept only `text` and `json`; invalid values return `cli.diagnostics_invalid`. JSON is one object with stable `code`, `message`, and optional structured fields. Text is `code: message\n`. Diagnostic writes go only to the Cobra error stream.

- [ ] **Step 5: Run focused tests**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test -race ./cmd/margo -run 'TestPublish|TestReadInput|Test.*Diagnostic' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit shared adapters**

```bash
git add cmd/margo/io.go cmd/margo/diagnostics.go cmd/margo/io_test.go cmd/margo/diagnostics_test.go cmd/margo/root.go
git commit -m "feat(cli): add safe input and output adapters"
```

### Task 2: Implement the HTML command with charts enabled

**Files:**
- Create: `cmd/margo/compiler.go`
- Create: `cmd/margo/html.go`
- Create: `cmd/margo/resources.go`
- Create: `cmd/margo/html_test.go`
- Create: `cmd/margo/resources_test.go`
- Modify: `cmd/margo/root.go`

**Interfaces:**
- Consumes: `charts.Extension()`, `margo.New`, `Compile`, `Render`, `RenderStandalone`, and Task 1 adapters.
- Produces: `func newCompiler() *margo.Compiler`, `func materializeLocalImages([]byte, string, string) ([]byte, error)`, and `func newHTMLCommand(Dependencies) *cobra.Command`.

- [ ] **Step 1: Write a failing HTML command E2E-style unit test**

```go
func TestHTMLCommandRendersChartsToStdout(t *testing.T) {
	input := "# Report\n\n```goshtosochart\ntype: bar\nseries:\n  - name: A\n    values: [1]\n```\n"
	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(testDependencies(strings.NewReader(input), &stdout, &stderr))
	cmd.SetArgs([]string{"html", "-"})
	if err := cmd.ExecuteContext(context.Background()); err != nil { t.Fatal(err) }
	for _, fragment := range []string{"<!doctype html>", "<h1>Report</h1>", "<svg", "<table"} {
		if !strings.Contains(strings.ToLower(stdout.String()), strings.ToLower(fragment)) { t.Fatalf("missing %q", fragment) }
	}
	if stderr.Len() != 0 { t.Fatalf("stderr = %q", stderr.String()) }
}
```

- [ ] **Step 2: Run and verify placeholder failure**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./cmd/margo -run TestHTMLCommandRendersChartsToStdout -count=1`

Expected: FAIL with `margo.command_not_implemented: html`.

- [ ] **Step 3: Implement HTML rendering**

Construct `margo.New(margo.WithExtension(charts.Extension()))`, read one input, compile/render, assemble the default standalone page with inline dependencies, render into a private buffer, materialize local images, then publish. Flags are `--output` default `-`, `--force`, and `--diagnostics` default `text`.

`materializeLocalImages` parses finalized HTML with `golang.org/x/net/html`, resolves relative image paths against the input file's parent or the injected working directory for stdin, rejects traversal and symlink escape outside that root, MIME-sniffs PNG/JPEG/WebP/GIF/SVG, enforces the effective resource byte limit, and rewrites each image to a `data:` URL. SVG passes a closed static-image XML allowlist that rejects scripts, event attributes, foreign objects, external references, and active URLs. Existing safe `data:` URLs pass through; `http`, `https`, `file`, protocol-relative, and unknown schemes return `cli.resource_external` because the default artifact promises no external fetch.

- [ ] **Step 4: Add path/stdin/error tests**

Cover file input, explicit stdout, destination refusal, `--force`, PNG/JPEG/WebP/GIF/SVG data-URL materialization, traversal and remote-resource rejection, invalid Markdown/extension diagnostic in text and JSON, two positional inputs rejected by Cobra, and proof that compile or resource failure writes zero artifact bytes.

- [ ] **Step 5: Run command tests**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test -race ./cmd/margo -run 'TestHTML' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit HTML command**

```bash
git add cmd/margo/compiler.go cmd/margo/html.go cmd/margo/resources.go cmd/margo/html_test.go cmd/margo/resources_test.go cmd/margo/root.go
git commit -m "feat(cli): generate standalone html"
```

### Task 3: Implement shared engine flags, PDF command, and doctor

**Files:**
- Create: `cmd/margo/engine_flags.go`
- Create: `cmd/margo/pdf.go`
- Create: `cmd/margo/doctor.go`
- Create: `cmd/margo/pdf_test.go`
- Create: `cmd/margo/doctor_test.go`
- Modify: `cmd/margo/root.go`

**Interfaces:**
- Consumes: `pdf/engines.Discover`, `pdf.Engine.Export`, `func (*margo.RenderResult) RuntimeDescriptor(margo.RenderInstanceID) (margo.RuntimeDescriptor, error)`, `func (*margo.RenderResult) DocumentFingerprint() margo.DocumentFingerprint`, and Task 1 adapters.
- Produces: `type engineFlags struct { Mode string; Path string }`, `func newPDFCommand(Dependencies) *cobra.Command`, and `func newDoctorCommand(Dependencies) *cobra.Command`; this task extends `Dependencies` with `EngineProbe engines.Probe` and `NextExecutionID func() margo.ExecutionID`.

- [ ] **Step 1: Write failing explicit-output and doctor tests**

```go
func TestPDFRequiresExplicitOutput(t *testing.T) {
	cmd := NewRootCommand(testDependencies(strings.NewReader("# Page\n"), io.Discard, io.Discard))
	cmd.SetArgs([]string{"pdf", "-"})
	if err := cmd.ExecuteContext(context.Background()); diagnosticCode(err) != "cli.output_required" { t.Fatalf("error = %v", err) }
}

func TestDoctorJSONUsesDiscoveryOrder(t *testing.T) {
	deps := testDependencies(strings.NewReader(""), io.Discard, io.Discard)
	deps.EngineProbe = deterministicUnavailableProbe()
	var stdout bytes.Buffer
	deps.Stdout = &stdout
	cmd := NewRootCommand(deps)
	cmd.SetArgs([]string{"doctor", "--diagnostics", "json"})
	if err := cmd.ExecuteContext(context.Background()); err != nil { t.Fatal(err) }
	assertCandidateOrder(t, stdout.Bytes(), []string{"environment", "path", "native"})
}
```

- [ ] **Step 2: Run and verify placeholders fail**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./cmd/margo -run 'TestPDF|TestDoctor' -count=1`

Expected: FAIL because real PDF and doctor commands are absent.

- [ ] **Step 3: Implement PDF composition**

Render and materialize the same standalone HTML as `html`, allocate one ID with `margo.NewInstanceAllocator().Next()`, call `renderResult.RuntimeDescriptor(instance)`, derive an `margo.ExecutionID` from a dependency-injected execution-ID source, discover/select once, export once, validate `%PDF-`/runtime/provenance, then publish. Map page flags `--page-size A4|Letter`, `--orientation portrait|landscape`, and four non-negative millimeter margins into `pdf.PageConfig`.

- [ ] **Step 4: Implement doctor as discovery-only**

`doctor` must never render or mutate the host. It reports every candidate, compiled-in state, discovered path, runtime version when probeable, diagnostic code/reason, build version, OS/architecture, and CGO/native capability in deterministic text or JSON.

- [ ] **Step 5: Add selection and no-fallback tests**

Cover `auto`, explicit Chromium, explicit native, invalid `--engine-path`, environment path, unavailable engines, successful binary stdout with diagnostics on stderr, and a Chromium render failure proving native export count remains zero.

- [ ] **Step 6: Run focused tests**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test -race ./cmd/margo -run 'TestPDF|TestDoctor|TestEngine' -count=1`

Expected: PASS.

- [ ] **Step 7: Commit PDF and doctor commands**

```bash
git add cmd/margo/engine_flags.go cmd/margo/pdf.go cmd/margo/doctor.go cmd/margo/pdf_test.go cmd/margo/doctor_test.go cmd/margo/root.go
git commit -m "feat(cli): generate pdf and report engines"
```

### Task 4: Implement HTML and PDF deck command modes

**Files:**
- Create: `cmd/margo/deck.go`
- Create: `cmd/margo/deck_test.go`
- Modify: `cmd/margo/root.go`

**Interfaces:**
- Consumes: `deck.Render`, `func (*deck.Result) RuntimeDescriptor(margo.RenderInstanceID) (margo.RuntimeDescriptor, error)`, `func (*deck.Result) DocumentFingerprint() margo.DocumentFingerprint`, shared compiler, shared engine flags, PDF engine selection, and publication adapters.
- Produces: `func newDeckCommand(Dependencies) *cobra.Command`.

- [ ] **Step 1: Write failing default and PDF-output tests**

```go
func TestDeckDefaultsToHTMLStdout(t *testing.T) {
	var stdout bytes.Buffer
	cmd := NewRootCommand(testDependencies(strings.NewReader("# One\n---\n# Two\n"), &stdout, io.Discard))
	cmd.SetArgs([]string{"deck", "-"})
	if err := cmd.ExecuteContext(context.Background()); err != nil { t.Fatal(err) }
	if !strings.Contains(stdout.String(), `class="margo-deck"`) { t.Fatal("deck HTML missing") }
}

func TestDeckPDFRequiresOutput(t *testing.T) {
	cmd := NewRootCommand(testDependencies(strings.NewReader("# One\n"), io.Discard, io.Discard))
	cmd.SetArgs([]string{"deck", "-", "--format", "pdf"})
	if err := cmd.ExecuteContext(context.Background()); diagnosticCode(err) != "cli.output_required" { t.Fatalf("error = %v", err) }
}
```

- [ ] **Step 2: Run and verify placeholder failure**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./cmd/margo -run 'TestDeck' -count=1`

Expected: FAIL with the deck placeholder diagnostic.

- [ ] **Step 3: Implement both formats**

`--format` accepts only `html` and `pdf`, default `html`. Both modes call `materializeLocalImages` on `deck.Result.HTML()` before publication or export. HTML publishes those standalone bytes. PDF requires explicit output, uses the same engine flags/discovery as `pdf`, prints the deck HTML with one-slide-per-page CSS, and never makes `deck` import `pdf`.

- [ ] **Step 4: Add chart/image/runtime and failure tests**

Cover popular image formats, charts registered by the CLI, Mermaid runtime completion in PDF mode, empty-slide diagnostics, output refusal/force, binary stdout, and render failure without fallback.

- [ ] **Step 5: Run command tests**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test -race ./cmd/margo -run 'TestDeck' -count=1`

Expected: PASS.

- [ ] **Step 6: Remove the final placeholder helper and commit**

```bash
git add cmd/margo/deck.go cmd/margo/deck_test.go cmd/margo/root.go
git commit -m "feat(cli): generate html and pdf decks"
```

### Task 5: Add black-box CLI and generated-HTML browser tests

**Files:**
- Create: `cmd/margo/e2e_test.go`
- Create: `testdata/cli/article.md`
- Create: `testdata/cli/deck.md`
- Create: `testdata/cli/images/sample.png`
- Create: `testdata/cli/images/sample.jpg`
- Create: `testdata/cli/images/sample.webp`
- Create: `testdata/cli/images/sample.gif`
- Create: `testdata/cli/images/sample.svg`

**Interfaces:**
- Consumes: built `./cmd/margo` binary and an installed Chromium probe.
- Produces: process-level proof for stdin/stdout/stderr, files, HTML behavior, charts, images, Mermaid, and PDF structure.

- [ ] **Step 1: Add black-box artifact tests**

Build `margo` into `t.TempDir()`, then run `html`, `pdf`, `deck --format html`, `deck --format pdf`, `doctor`, and `version`. Assert exit status, separate streams, exact overwrite behavior, `%PDF-`, and no partial destination after injected invalid input.

- [ ] **Step 2: Add browser HTML tests**

Open generated article and deck HTML in installed Chromium. Assert headings, links, tables, PNG/JPEG/WebP/GIF/SVG dimensions, static chart SVG/accessibility table, Mermaid terminal state, previous/next buttons, all four keys, and print CSS. If Chromium is absent, HTML file/DOM parser tests still run and only browser-dependent assertions explicitly skip.

- [ ] **Step 3: Run black-box tests**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./cmd/margo -run 'TestCLI|TestGeneratedHTML' -count=1 -v`

Expected: PASS; browser-only assertions may explicitly SKIP if no installed browser exists.

- [ ] **Step 4: Commit E2E proof**

```bash
git add cmd/margo/e2e_test.go testdata/cli
git commit -m "test(cli): verify generated html and pdf artifacts"
```

### Task 6: Rewrite README for first-contact users and migration truth

**Files:**
- Modify: `README.md`
- Create: `docs/testing/pdf-engine-matrix.md`

**Interfaces:**
- Consumes: implemented command help, engine discovery output, ADR 0001, and observed runner/browser versions.
- Produces: public documentation matching shipped behavior without requiring Goshtoso ecosystem context.

- [ ] **Step 1: Add README contract assertions**

Extend a docs test to require: product-first description; library and CLI quick starts; all five commands; one root install path; chart opt-in example for library consumers; engine selection order; no-download promise; native capability table; CGO/musl explanation; explicit output/force behavior; diagnostics stream contract; historical submodule migration warning; and links to ADR/testing evidence.

- [ ] **Step 2: Run and verify documentation test failure**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test . ./cmd/margo -run 'Test.*Docs|TestREADME' -count=1`

Expected: FAIL because README lacks the unified release/CLI contract.

- [ ] **Step 3: Rewrite README and evidence page**

Lead with “Margo turns Markdown into standalone HTML, PDF documents, and presentation decks.” Explain consumer-neutral packages before mentioning related Arai Hû projects. Record exact tested Chromium, OS, native runtime, and Go versions in `docs/testing/pdf-engine-matrix.md`; state they are evidence, not a rejection policy for other Chromium versions.

- [ ] **Step 4: Run docs and help consistency gates**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test . ./cmd/margo -run 'Test.*Docs|TestREADME|TestRootHelp' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit documentation**

```bash
git add README.md docs/testing/pdf-engine-matrix.md
git commit -m "docs: explain unified margo engine and cli"
```

### Task 7: Add release-shape CI without publishing

**Files:**
- Modify: `.github/workflows/ci.yml`
- Create: `scripts/verify-release-shape.sh`

**Interfaces:**
- Consumes: one root module and the release capability matrix.
- Produces: CI builds named `margo` for macOS, Windows, portable Linux/musl, plus optional Linux WebKitGTK when its runner is configured.

- [ ] **Step 1: Implement release-shape verifier**

The script accepts an output directory and asserts every binary/archive derives from one provided root version, is named `margo` (or `margo.exe`), contains no nested module metadata, and produces matching `margo version` output when executable on the host. It performs no upload.

- [ ] **Step 2: Add CI build matrix**

Build portable Linux with `CGO_ENABLED=0`, keep native variants on explicit runners with their platform dependencies, run `go version -m` on produced binaries, invoke the verifier, and run the musl container test. Do not create tags, releases, or uploaded public assets.

- [ ] **Step 3: Run local portable release gate**

Run: `tmpdir=$(mktemp -d) && CGO_ENABLED=0 GOWORK=off GOFLAGS=-mod=readonly go build -trimpath -o "$tmpdir/margo" ./cmd/margo && scripts/verify-release-shape.sh "$tmpdir" && rm -f "$tmpdir/margo" && rmdir "$tmpdir"`

Expected: PASS and the explicit temporary directory is removed.

- [ ] **Step 4: Commit CI release shape**

```bash
git add .github/workflows/ci.yml scripts/verify-release-shape.sh
git commit -m "ci: verify unified margo release shape"
```

### Task 8: Final release-candidate checkpoint and human PDF review

**Files:**
- Verify: repository root, generated temporary artifacts, documentation, and CI configuration.

**Interfaces:**
- Consumes: all prior unified-module, deck, PDF-engine, and CLI tasks.
- Produces: an immutable, unmerged checkpoint suitable for independent and human visual review.

- [ ] **Step 1: Run automated gates**

Run: `test "$(find . -name go.mod -not -path './.git/*' -print | sort)" = './go.mod' && test -z "$(gofmt -l $(find . -name '*.go' -not -path './.git/*'))" && GOWORK=off GOFLAGS=-mod=readonly go vet ./... && GOWORK=off GOFLAGS=-mod=readonly go test -race ./... -count=1 && git diff --check`

Expected: PASS.

- [ ] **Step 2: Generate human-review samples outside the repository**

Build the checkpoint binary, render the article and deck fixtures to a fresh `mktemp -d` directory using installed Chromium or the explicitly selected native engine, and record command, engine name/version, OS, architecture, output SHA-256, and page count. Do not commit generated PDFs.

- [ ] **Step 3: Perform human visual review**

Review both PDFs for missing charts/images, clipped content, incorrect page breaks, navigation chrome in print, font failure, and Mermaid incompletion. Record ACCEPT or REJECT bound to the exact HEAD/tree and artifact hashes; automation does not infer this verdict.

- [ ] **Step 4: Capture the final identity**

Run: `git status --short --branch && git rev-parse HEAD HEAD^{tree} && shasum -a 256 go.mod go.sum pdf/platform-toolchain.lock pdf/platform/runner-contracts.json`

Expected: clean worktree and immutable evidence ready for independent review. Stop before push, merge, tag, release, or publication.

- [ ] **Step 5: Record the public-install gate as pending authorization**

If no authorized immutable root tag exists, record `BLOCKED_RELEASE_IDENTITY` with the checkpoint HEAD/tree; do not create a tag, local proxy, pseudo-version, or replacement module to simulate it. After a separately authorized root tag is publicly visible, require the control plane to provide `MARGO_AUTHORIZED_TAG`, create an explicit temporary `MARGO_INSTALL_BIN=$(mktemp -d)`, run `GOBIN="$MARGO_INSTALL_BIN" GOWORK=off go install "github.com/araihu/margo/cmd/margo@${MARGO_AUTHORIZED_TAG}"` from outside every Margo checkout, execute `"$MARGO_INSTALL_BIN/margo" version`, verify its module version and command version match `MARGO_AUTHORIZED_TAG`, then remove that exact temporary directory before any release acceptance.
