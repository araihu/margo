# Margo PDF Engines Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement deterministic discovery plus installed-Chromium and capability-gated native PDF engines without any download path.

**Architecture:** `pdf/engines` inventories candidates and selects exactly one engine before rendering; fallback is permitted only while a candidate is absent or unavailable. `pdf/chromium` owns CDP lifecycle and runtime completion, while `pdf/native` exposes one build-tag-selected backend or an honest compiled-out capability.

**Tech Stack:** Go 1.26.5, chromedp v0.14.2, CDP, WKWebView, WebView2, WebKitGTK, CGO build tags, JSON Schema contracts, standard `testing` package

## Global Constraints

- Engine flags are `auto`, `chromium`, and `native`; explicit selection never falls back.
- Auto discovery order is explicit `--engine-path`, `MARGO_CHROMIUM_PATH`, PATH/known Chromium locations, compiled native, then typed failure.
- An invalid explicit path fails immediately; a render failure after selection never falls back.
- Margo never downloads Chromium, Playwright, a native runtime, a package, or a helper binary.
- Chromium version is discovered and reported; policy documents state tested versions rather than rejecting other installed versions.
- Portable Linux and musl use `CGO_ENABLED=0` and Chromium only.
- macOS native uses WKWebView on the required main thread; Windows native uses installed WebView2; optional Linux native uses WebKitGTK with declared dynamic dependencies.
- Platform contracts use a root-relative schema version and commands rooted at `./pdf/platform`; no self-referential module pseudo-version or fabricated sum is allowed.
- PDF bytes require a human visual review; automated tests prove contracts, structure, runtime completion, and engine provenance.
- Every Go gate runs with `GOWORK=off GOFLAGS=-mod=readonly`.
- Do not push, merge, tag, release, or publish while executing this plan.

---

## File structure

- `pdf/engines/discovery.go`, `selection.go`, `diagnostic.go`: deterministic candidate inventory and selection.
- `pdf/chromium/engine.go`, `browser.go`, `runtime.go`: installed-browser CDP engine.
- `pdf/native/native.go`: stable public capability/constructor.
- `pdf/native/native_stub_unsupported.go`, `native_stub_nocgo.go`, `native_stub_linux.go`: unsupported, CGO-disabled, and portable-Linux implementations.
- `pdf/native/native_darwin.go`, `native_windows.go`, `native_linux.go`: tagged platform backends.
- `pdf/platform/runner-contracts.json`, `platform-toolchain.lock`, `bootstrap.go`: version-2 root-relative verification contracts.
- `pdf/e2e/`: browser and PDF structural tests.

### Task 1: Version the root-relative platform contract

**Files:**
- Modify: `pdf/platform/runner-contracts.json`
- Modify: `pdf/platform-toolchain.lock`
- Modify: `pdf/platform/bootstrap.go`
- Modify: `pdf/platform/runner_probe_test.go`
- Modify: `pdf/platform/engine_probe_test.go`
- Modify: `pdf/platform/no_download_probe_test.go`

**Interfaces:**
- Consumes: the unified root module and platform runner IDs `windows-webview2/v2`, `darwin-wkwebview/v2`, `linux-webkitgtk/v2`, `chromium-cdp/v2`.
- Produces: schema `margo/pdf-platform-contracts/v2` with commands rooted at `./pdf/platform` and no frozen self-module sum.

- [ ] **Step 1: Rewrite probe expectations first**

Assert `schemaVersion == "margo/pdf-platform-contracts/v2"`, every command begins `go test ./pdf/platform`, the lock has `modulePath == "github.com/araihu/margo"`, and the JSON contains neither `/private/tmp/` nor `github.com/araihu/margo v0.0.1` nor a root module sum.

- [ ] **Step 2: Run probes and verify v1 contract failure**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./pdf/platform -run 'TestRunnerContracts|TestPlatformToolchainLock' -count=1`

Expected: FAIL because the current files are v1 and contain submodule-relative commands/self-module evidence.

- [ ] **Step 3: Write the v2 contract and bootstrap parser**

The lock records `goVersion: "1.26.5"`, `modulePath: "github.com/araihu/margo"`, exact externally verifiable engine dependency versions, and tested Chromium versions as evidence fields. It does not claim a minimum/maximum Chromium support range.

- [ ] **Step 4: Run all platform probes**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./pdf/platform -count=1`

Expected: PASS without network or downloads.

- [ ] **Step 5: Commit the versioned platform contract**

```bash
git add pdf/platform-toolchain.lock pdf/platform/runner-contracts.json pdf/platform/bootstrap.go pdf/platform/*_test.go
git commit -m "build(pdf): version unified platform contracts"
```

### Task 2: Implement deterministic discovery and strict fallback boundaries

**Files:**
- Create: `pdf/engines/discovery.go`
- Create: `pdf/engines/selection.go`
- Create: `pdf/engines/diagnostic.go`
- Create: `pdf/engines/discovery_test.go`
- Create: `pdf/engines/selection_test.go`
- Modify: `pdf/engine.go`
- Modify: `pdf/engine_test.go`

**Interfaces:**
- Consumes: `pdf.Engine` and native `Capability()` from Task 4.
- Produces: `type Mode string`, `const ModeAuto, ModeChromium, ModeNative`, `type Request struct { Mode Mode; EnginePath string }`, `type Candidate`, `type Discovery`, `type Probe`, `func Discover(context.Context, Request, Probe) (Discovery, error)`, `func (Discovery) Select() (pdf.Engine, Candidate, error)`, and extended `pdf.EngineInfo{Name, Version, Path, Source}` provenance.

- [ ] **Step 1: Write failing discovery-order tests**

```go
func TestDiscoverUsesDeterministicAutoOrder(t *testing.T) {
	probe := fakeProbe{envPath: "/env/chrome", pathBrowsers: []string{"/path/chromium"}, nativeAvailable: true}
	discovery, err := Discover(context.Background(), Request{Mode: ModeAuto, EnginePath: "/explicit/chrome"}, probe)
	if err != nil { t.Fatal(err) }
	if got, want := discovery.Sources(), []string{"flag", "environment", "path", "native"}; !slices.Equal(got, want) { t.Fatalf("sources = %v", got) }
}

func TestSelectionDoesNotFallbackAfterRenderStarts(t *testing.T) {
	selected, _, err := successfulDiscovery(t).Select()
	if err != nil { t.Fatal(err) }
	if got := selected.Name(); got != "chromium" { t.Fatalf("selected = %q", got) }
	if err := selected.Export(context.Background(), validRequest()); diagnosticCode(err) != "pdf.chromium.render_failed" { t.Fatalf("error = %v", err) }
	if successfulDiscovery(t).nativeExports != 0 { t.Fatal("native engine was invoked after render failure") }
}
```

- [ ] **Step 2: Run tests and verify missing package failure**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./pdf/engines -count=1`

Expected: FAIL because `Discover`, modes, and candidate types do not exist.

- [ ] **Step 3: Implement candidate records and discovery**

```go
type Candidate struct {
	Name string `json:"name"`
	Source string `json:"source"`
	Compiled bool `json:"compiled"`
	Path string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
	Available bool `json:"available"`
	Code string `json:"code,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type Discovery struct { candidates []Candidate; engines []pdf.Engine }
func (d Discovery) Candidates() []Candidate { return append([]Candidate(nil), d.candidates...) }

type Probe struct {
	LookupEnv func(string) (string, bool)
	LookPath func(string) (string, error)
	Stat func(string) (fs.FileInfo, error)
	ChromiumVersion func(context.Context, string) (string, error)
	ChromiumEngine func(string) (pdf.Engine, error)
	Native func(context.Context) (pdf.Engine, Candidate)
	KnownPaths []string
	GOOS string
}
```

Validate explicit paths as executable regular files and Chromium-family binaries. Treat flag/env errors as explicit configuration failures; PATH and known-location misses are unavailable candidates. Preserve attempt order in the final `pdf.engine_unavailable` diagnostic.

Extend `pdf.EngineInfo` with `Path string` and `Source string`; `Source` is one of `flag`, `environment`, `path`, `known-location`, or `native`. Validation requires name, version, and source; Chromium results require the normalized executable path, while native results leave path empty. `Discovery.Select` decorates the selected engine so every returned `pdf.Result.Engine` contains the selected candidate provenance.

- [ ] **Step 4: Add explicit-mode and invalid-path tests**

Cover `auto`, `chromium`, `native`, unknown mode, invalid flag path, unavailable native runtime, duplicate executable discovery, and deterministic JSON candidate projection.

- [ ] **Step 5: Run package tests with the race detector**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test -race ./pdf/engines -count=1`

Expected: PASS.

- [ ] **Step 6: Commit discovery**

```bash
git add pdf/engines pdf/engine.go pdf/engine_test.go
git commit -m "feat(pdf): discover installed rendering engines"
```

### Task 3: Implement the installed-Chromium engine

**Files:**
- Create: `pdf/chromium/engine.go`
- Create: `pdf/chromium/browser.go`
- Create: `pdf/chromium/runtime.go`
- Create: `pdf/chromium/engine_test.go`
- Create: `pdf/e2e/chromium_test.go`

**Interfaces:**
- Consumes: `pdf.Engine`, `pdf.Request`, `pdf.Result`, `margo.ValidateRuntimeDescriptor`, and an explicit installed executable path.
- Produces: `type Config struct { ExecutablePath string; Timeout time.Duration }`, `func New(Config) (*Engine, error)`, `func (*Engine) Name() string`, `func (*Engine) Version(context.Context) (string, error)`, and `func (*Engine) Export(context.Context, pdf.Request) (pdf.Result, error)`.

- [ ] **Step 1: Write failing constructor and validation tests**

```go
func TestNewRequiresAbsoluteExecutable(t *testing.T) {
	_, err := New(Config{ExecutablePath: "chromium"})
	if got := diagnosticCode(err); got != "pdf.chromium.path_invalid" { t.Fatalf("code = %q", got) }
}

func TestExportValidatesBeforeLaunching(t *testing.T) {
	engine := testEngine(t)
	request := validRequest()
	request.Page.Size = "Legal"
	_, err := engine.Export(context.Background(), request)
	if got := diagnosticCode(err); got != "pdf.page_size_unsupported" { t.Fatalf("code = %q", got) }
	if engine.launches != 0 { t.Fatal("browser launched for invalid request") }
}
```

- [ ] **Step 2: Run and verify missing engine failure**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./pdf/chromium -count=1`

Expected: FAIL because the package implementation does not exist.

- [ ] **Step 3: Implement browser lifecycle and print mapping**

Launch only the configured executable with an isolated temporary user-data directory, no remote debugging listener exposed beyond the child process, and no download behavior. Load finalized HTML from a controlled local origin, deny external navigation/subrequests according to the Margo resource policy, wait for the declared runtime report, then call CDP `Page.printToPDF` with exact A4/Letter, orientation, and millimeter-to-inch margin mapping.

- [ ] **Step 4: Validate results before returning**

Require `%PDF-` bytes, a terminal runtime report matching request descriptor/execution ID, and non-empty engine version. Return cloned bytes and `pdf.EngineInfo{Name: "chromium", Version: version}`. Any failure after launch uses a `pdf.chromium.*` render diagnostic and never invokes another engine.

- [ ] **Step 5: Add installed-browser E2E with an honest skip**

The test locates Chromium through the same read-only probe. If absent, call `t.Skip("installed Chromium unavailable")`; if present, render HTML containing text, PNG, JPEG, WebP, GIF, SVG, a table, Mermaid, and a chart. Assert the result begins `%PDF-`, has at least one page, the runtime report is terminal, blocked external requests are reported, and engine version is recorded.

- [ ] **Step 6: Run unit and E2E gates**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test -race ./pdf/chromium -count=1 && GOWORK=off GOFLAGS=-mod=readonly go test ./pdf/e2e -run TestChromium -count=1 -v`

Expected: unit PASS; E2E PASS when Chromium is installed or explicit SKIP when absent.

- [ ] **Step 7: Commit Chromium support**

```bash
git add pdf/chromium pdf/e2e/chromium_test.go
git commit -m "feat(pdf): export with installed chromium"
```

### Task 4: Add the native capability boundary and portable stub

**Files:**
- Create: `pdf/native/native.go`
- Create: `pdf/native/native_stub_unsupported.go`
- Create: `pdf/native/native_stub_nocgo.go`
- Create: `pdf/native/native_stub_linux.go`
- Create: `pdf/native/native_stub_test.go`

**Interfaces:**
- Consumes: `pdf.Engine`.
- Produces: `type Capability struct { Name string; Compiled bool; Available bool; Code string; Reason string }`, `func Probe(context.Context) Capability`, and `func New() (pdf.Engine, error)`.

`native_stub_test.go` uses `//go:build !cgo || (!darwin && !windows && !linux) || (linux && !margo_webkitgtk)` so the compiled-out assertion never runs against an included Darwin or Windows backend.

- [ ] **Step 1: Write the CGO-disabled contract test**

```go
func TestPortableBuildReportsNativeCompiledOut(t *testing.T) {
	capability := Probe(context.Background())
	if capability.Compiled { t.Fatal("portable build claims native engine") }
	if capability.Code != "pdf.native.compiled_out" { t.Fatalf("code = %q", capability.Code) }
	if _, err := New(); diagnosticCode(err) != "pdf.native.compiled_out" { t.Fatalf("error = %v", err) }
}
```

- [ ] **Step 2: Run under CGO-disabled Linux semantics and verify failure**

Run: `CGO_ENABLED=0 GOOS=linux GOWORK=off GOFLAGS=-mod=readonly go test ./pdf/native -run TestPortableBuildReportsNativeCompiledOut -count=1`

Expected: FAIL because `Probe` and `New` do not exist.

- [ ] **Step 3: Implement stable API and build-tagged stub**

`native.go` contains only platform-neutral types. Use `//go:build !darwin && !windows && !linux` for unsupported platforms, `//go:build (darwin || linux) && !cgo` for CGO-disabled Darwin/Linux, and `//go:build linux && cgo && !margo_webkitgtk` for portable Linux. Each stub returns `pdf.native.compiled_out` and contains no dynamic loading or subprocess behavior. Darwin implementation uses `darwin && cgo`, Windows uses `windows`, and optional Linux uses `linux && cgo && margo_webkitgtk`.

- [ ] **Step 4: Run portable gates**

Run: `CGO_ENABLED=0 GOWORK=off GOFLAGS=-mod=readonly go test ./pdf/native -count=1 && CGO_ENABLED=0 GOOS=linux GOWORK=off GOFLAGS=-mod=readonly go test ./pdf/native -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the capability boundary**

```bash
git add pdf/native/native.go pdf/native/native_stub_unsupported.go pdf/native/native_stub_nocgo.go pdf/native/native_stub_linux.go pdf/native/native_stub_test.go
git commit -m "feat(pdf): define native engine capability"
```

### Task 5: Implement and runner-test native backends

**Files:**
- Create: `pdf/native/native_darwin.go`
- Create: `pdf/native/native_windows.go`
- Create: `pdf/native/native_linux.go`
- Create: `pdf/native/native_darwin_test.go`
- Create: `pdf/native/native_windows_test.go`
- Create: `pdf/native/native_linux_test.go`
- Modify: `pdf/platform/runner-contracts.json`

**Interfaces:**
- Consumes: `pdf.Request`, `pdf.Result`, and the stable native API from Task 4.
- Produces: build-tag-selected WKWebView, WebView2, and WebKitGTK engines with identical `pdf.Engine` behavior.

- [ ] **Step 1: Add platform mapping tests before bridge code**

For each backend, test PageConfig-to-native units, runtime unavailability, cancellation, invalid HTML, `%PDF-` validation, exact engine identity, and cleanup after failure. Darwin tests also assert main-thread dispatch; Windows tests distinguish WebView2 runtime absence; Linux tests distinguish missing WebKitGTK shared libraries.

- [ ] **Step 2: Run each platform runner expecting missing implementation**

Run the exact v2 commands from `pdf/platform/runner-contracts.json` on matching hosts.

Expected: FAIL because backend constructors/bridges are absent.

- [ ] **Step 3: Implement official API bridges**

Use WKWebView PDF creation on the macOS main thread, WebView2 environment/controller initialization on Windows, and WebKitGTK print operation on the opt-in Linux build. Each bridge loads finalized local HTML, enforces subresource policy, waits for runtime terminal state, and returns no bytes before validation completes.

- [ ] **Step 4: Run platform unit and E2E runners**

Run every v2 runner command on its declared platform. Record exact OS, architecture, Go version, native runtime/library version, and test result in the runner evidence, without claiming support for untested versions.

Expected: PASS on declared runners; absence of required runtime is a typed test fixture, not a fabricated success.

- [ ] **Step 5: Commit native backends**

```bash
git add pdf/native pdf/platform/runner-contracts.json
git commit -m "feat(pdf): add platform native exporters"
```

### Task 6: Prove portable musl and no-download behavior

**Files:**
- Create: `pdf/e2e/musl_test.go`
- Modify: `pdf/platform/no_download_probe_test.go`

**Interfaces:**
- Consumes: the unified root module, discovery, Chromium engine, and native stub.
- Produces: release-relevant proof that portable Linux does not require CGO or hidden browser installation.

- [ ] **Step 1: Add a musl container gate script to the test**

The test runs an Alpine container with the repository mounted read-only, `CGO_ENABLED=0`, `GOWORK=off`, and `GOFLAGS=-mod=readonly`; it compiles `./cmd/margo`, runs `margo doctor --diagnostics json`, and asserts native is `compiled_out`. A no-browser execution of `margo pdf` must fail with `pdf.engine_unavailable` and create no output file.

- [ ] **Step 2: Scan production code for downloader signatures**

Extend the probe to reject Playwright installation APIs, `chromedp/headless-shell`, `http.Get`/download helpers in `pdf/`, package-manager invocations, and writes to browser cache locations. Allow network APIs only where they enforce resource blocking or local CDP control.

- [ ] **Step 3: Run portable and repository gates**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./pdf/platform ./pdf/engines ./pdf/chromium ./pdf/native ./pdf/e2e -count=1 && git diff --check`

Expected: PASS; installed-browser E2E may explicitly SKIP only when its prerequisite is absent.

- [ ] **Step 4: Commit portable proof**

```bash
git add pdf/e2e/musl_test.go pdf/platform/no_download_probe_test.go
git commit -m "test(pdf): prove portable no-download behavior"
```

### Task 7: PDF engine checkpoint

**Files:**
- Verify: `pdf/`, `go.mod`, and `go.sum`.

**Interfaces:**
- Consumes: Tasks 1-6.
- Produces: an independently reviewable engine slice.

- [ ] **Step 1: Run final gates and capture hashes**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./pdf/... -count=1 && GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1 && git diff --check && shasum -a 256 go.mod go.sum pdf/platform-toolchain.lock pdf/platform/runner-contracts.json && git status --short --branch && git rev-parse HEAD HEAD^{tree}`

Expected: all gates pass, hashes and HEAD/tree are recorded, and no uncommitted files remain.
