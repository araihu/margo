# Margo Development Server and Live Reload Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `margo serve` as a development-only, in-memory static-site server with automatic port selection, recursive watching, and browser live reload.

**Architecture:** Cobra resolves a Markdown tree or YAML project into a CLI-owned builder. `internal/devserver` owns listener selection, immutable snapshots, HTTP/SSE behavior, rebuild coordination, and the recursive `fsnotify` adapter. Successful builds atomically replace the served snapshot; failed builds retain the last good result.

**Tech Stack:** Go 1.26.5, Cobra, `net/http`, `fsnotify`, Server-Sent Events, existing `site.Build`/`site.BuildConfig`, existing browser opener dependency.

**Spec:** `docs/superpowers/specs/2026-08-18-margo-serve-live-reload-design.md`

## Global Constraints

- This is a development-only server, not a production hosting contract.
- Default bind host is `127.0.0.1`; warn when binding a non-loopback host.
- Automatic ports are `8080`, `8000`, `3000`, `1313`, and `4000`, followed by an OS-selected port.
- An explicitly supplied port never falls back.
- Serve generated artifacts from memory; never write or mutate configured output.
- Reload browsers only after successful builds; retain the last good snapshot after later failures.
- Keep `/ssg` generic and keep Margo project/config policy in the CLI and `/site` layers.
- Add no public Margo library API.
- Implement every production behavior through a witnessed failing test first.

---

## File Map

- `internal/devserver/types.go`: core interfaces, options, build events, immutable snapshot shape.
- `internal/devserver/port.go`: ordered listener selection and URL construction.
- `internal/devserver/handler.go`: artifact routing, MIME handling, error page, reload injection.
- `internal/devserver/reload.go`: SSE generation broker and injected client.
- `internal/devserver/coordinator.go`: serialized builds, pending follow-up, snapshot swaps, shutdown.
- `internal/devserver/watcher.go`: recursive `fsnotify` source and debounce.
- `cmd/margo/serve.go`: Cobra command, project discovery/build adapter, development-only warnings.
- `cmd/margo/root.go`: register command and one injectable serve seam.
- `README.md`, `unified_docs_test.go`, `cmd/margo/root_test.go`: user-facing CLI contract.

### Task 1: Deterministic Listener Selection

**Files:**
- Create: `internal/devserver/port.go`
- Test: `internal/devserver/port_test.go`

**Interfaces:**
- Produces: `type ListenFunc func(network, address string) (net.Listener, error)`
- Produces: `func Listen(host string, port int, explicit bool, listen ListenFunc) (net.Listener, int, error)`
- Produces: `func URL(host string, port int, basePath string) string`

- [ ] **Step 1: Write failing listener-order and explicit-port tests**

Use fake listeners and a recording `ListenFunc`. Assert automatic attempts are
`127.0.0.1:8080`, `:8000`, `:3000`, `:1313`, `:4000`, then `:0`; assert the
reported port comes from the retained listener address. Assert explicit `4444`
attempts only `127.0.0.1:4444` and returns its bind error.

- [ ] **Step 2: Verify RED**

Run: `GOWORK=off go test ./internal/devserver -run 'TestListen|TestURL' -count=1`

Expected: FAIL because package/functions do not exist.

- [ ] **Step 3: Implement minimal listener selection**

Validate host, explicit ports `1..65535`, and non-nil `ListenFunc`. Keep the
first successful listener open. After candidate exhaustion call the same
function with `host:0` and derive the selected TCP port from `listener.Addr()`.
Bracket IPv6 hosts through `net.JoinHostPort`. Normalize base paths in `URL`.

- [ ] **Step 4: Verify GREEN**

Run: `GOWORK=off go test ./internal/devserver -run 'TestListen|TestURL' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/devserver/port.go internal/devserver/port_test.go
git commit -m "feat(cli): select development server ports"
```

### Task 2: Immutable Snapshot HTTP Handler and SSE Reload

**Files:**
- Create: `internal/devserver/types.go`
- Create: `internal/devserver/handler.go`
- Create: `internal/devserver/reload.go`
- Test: `internal/devserver/handler_test.go`
- Test: `internal/devserver/reload_test.go`

**Interfaces:**
- Produces: `func NewSnapshot(result site.Result) Snapshot`
- Produces: `type SnapshotStore` with `Replace(Snapshot) uint64`, `SetError(error)`, and read-only current-state access used by the handler.
- Produces: `func NewHandler(store *SnapshotStore, broker *Broker) http.Handler`
- Produces: `type Broker` with `Publish(uint64)` and `ServeHTTP(http.ResponseWriter, *http.Request)`.

- [ ] **Step 1: Write failing snapshot/handler tests**

Construct `site.Result` fixtures and assert:

```go
response := httptest.NewRecorder()
handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/docs/guide.html", nil))
```

Verify base-path root redirect, `index.html` resolution, exact artifact bytes,
MIME type, `Cache-Control: no-store`, 404 isolation, and HTML injection directly
before `</body>`. Assert the source snapshot still contains its original bytes.
Assert initial error pages HTML-escape build diagnostics.

- [ ] **Step 2: Verify handler RED**

Run: `GOWORK=off go test ./internal/devserver -run 'TestSnapshot|TestHandler' -count=1`

Expected: FAIL on missing snapshot/handler types.

- [ ] **Step 3: Implement snapshot store and handler**

Copy artifact paths and bytes in `NewSnapshot`. Store immutable state behind an
`atomic.Pointer`. Resolve only clean slash paths below the configured base path;
never access disk. Use `mime.TypeByExtension` with deterministic fallbacks for
HTML, CSS, JavaScript, JSON, XML, text, SVG, PNG, and JPEG. Inject only HTML that
contains `</body>`.

- [ ] **Step 4: Verify handler GREEN**

Run: `GOWORK=off go test ./internal/devserver -run 'TestSnapshot|TestHandler' -count=1`

Expected: PASS.

- [ ] **Step 5: Write failing SSE tests**

Connect an `httptest` request with a cancellable context. Assert the first event
is `event: ready` with current generation, `Publish(2)` yields `event: reload`,
and cancellation releases the subscriber. Assert the injected script uses
`EventSource("/.margo/live-reload")` and reloads only for reload events.

- [ ] **Step 6: Verify SSE RED**

Run: `GOWORK=off go test ./internal/devserver -run 'TestBroker|TestReload' -count=1`

Expected: FAIL on missing broker behavior.

- [ ] **Step 7: Implement SSE broker and injection client**

Use per-client buffered channels protected by a mutex. Send the ready event
immediately, fan out reload generations without blocking builds, set SSE and
no-cache headers, and remove subscribers on request cancellation.

- [ ] **Step 8: Verify task GREEN**

Run: `GOWORK=off go test ./internal/devserver -count=1`

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/devserver/types.go internal/devserver/handler.go internal/devserver/reload.go internal/devserver/*_test.go
git commit -m "feat(cli): serve immutable site snapshots"
```

### Task 3: Serialized Rebuild Coordinator

**Files:**
- Create: `internal/devserver/coordinator.go`
- Test: `internal/devserver/coordinator_test.go`

**Interfaces:**
- Produces: `type Builder interface { Build(context.Context) (Snapshot, error) }`
- Produces: `type ChangeSource interface { Changes() <-chan struct{}; Errors() <-chan error; Close() error }`
- Produces: `type BuildEvent struct { Generation uint64; Err error; Initial bool }`
- Produces: `func Run(ctx context.Context, options Options) error`

- [ ] **Step 1: Write failing coordinator tests**

Use a blocking fake builder and channel-backed change source. Assert the HTTP
server is reachable before a successful build, initial failure produces an
error snapshot, success publishes one generation, later failure preserves the
previous bytes, multiple changes during a build cause one follow-up build, no
builds overlap, watcher errors stop the run, and cancellation closes resources.

- [ ] **Step 2: Verify RED**

Run: `GOWORK=off go test ./internal/devserver -run 'TestRun' -count=1`

Expected: FAIL because coordinator is missing.

- [ ] **Step 3: Implement minimal coordinator**

Accept an already-bound listener, builder, change source, store, broker,
`Started func(string)`, and `BuildReported func(BuildEvent)`. Run builds in one
goroutine at a time. Track `building` and `pending` in the coordinator select
loop. Start one follow-up after completion when pending. Use `http.Server` and
context-driven `Shutdown`; always close the change source and listener.

- [ ] **Step 4: Verify GREEN and race safety**

Run: `GOWORK=off go test -race ./internal/devserver -run 'TestRun' -count=1`

Expected: PASS with no race reports.

- [ ] **Step 5: Commit**

```bash
git add internal/devserver/coordinator.go internal/devserver/coordinator_test.go
git commit -m "feat(cli): coordinate live site rebuilds"
```

### Task 4: Recursive Debounced File Watcher

**Files:**
- Create: `internal/devserver/watcher.go`
- Test: `internal/devserver/watcher_test.go`
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- Consumes: `ChangeSource`
- Produces: `func Watch(root string, ignored func(string) bool, debounce time.Duration) (ChangeSource, error)`

- [ ] **Step 1: Write failing watcher integration tests**

Use temporary directories. Assert one debounced change for repeated writes,
events from existing nested directories, events after creating a new directory,
no events below `.git`, no events when the dynamic ignore callback selects the
configured output directory, and clean channel closure.

- [ ] **Step 2: Verify RED**

Run: `GOWORK=off go test ./internal/devserver -run 'TestWatch' -count=1`

Expected: FAIL because watcher is missing.

- [ ] **Step 3: Implement recursive fsnotify adapter**

Promote `github.com/fsnotify/fsnotify v1.7.0` to a direct dependency. Walk and
watch real directories without following symlinks. Ignore `.git` and
`.worktrees` before invoking the project callback. On directory creation, add
the complete new subtree. Coalesce Create, Write, Remove, and Rename events with
a resettable timer and a buffered output signal.

- [ ] **Step 4: Verify GREEN**

Run: `GOWORK=off go test ./internal/devserver -run 'TestWatch' -count=1`

Expected: PASS.

- [ ] **Step 5: Verify full internal package**

Run: `GOWORK=off go test -race ./internal/devserver -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/devserver/watcher.go internal/devserver/watcher_test.go go.mod go.sum
git commit -m "feat(cli): watch development site sources"
```

### Task 5: Cobra Command and Project Adapter

**Files:**
- Create: `cmd/margo/serve.go`
- Create: `cmd/margo/serve_test.go`
- Modify: `cmd/margo/root.go`
- Modify: `cmd/margo/root_test.go`

**Interfaces:**
- Consumes: all `internal/devserver` interfaces.
- Produces: `type serveRequest struct { Input string; Host string; Port int; PortExplicit bool; Open bool }`
- Produces: `type serveFunc func(context.Context, serveRequest) error`
- Produces: `func newServeCommand(Dependencies) *cobra.Command`

- [ ] **Step 1: Write failing Cobra contract tests**

Inject a `serveFunc` into `Dependencies`. Assert no positional input becomes
`.`, one directory remains that directory, one YAML path remains explicit,
`--host`, `--port`, and `--open` populate the request, two inputs fail, explicit
port `0`/`65536` fails, root help lists `serve`, and serve help contains
`development` plus `not for production`.

- [ ] **Step 2: Verify Cobra RED**

Run: `GOWORK=off go test ./cmd/margo -run 'TestServeCommand|TestRootHelp' -count=1`

Expected: FAIL because serve is not registered.

- [ ] **Step 3: Implement command parsing and injection seam**

Add `ServeSite serveFunc` to `Dependencies`, default it in normalization, and
register `newServeCommand`. Use `cobra.MaximumNArgs(1)` through diagnostic
reporting. Determine explicit port with `command.Flags().Changed("port")`.

- [ ] **Step 4: Verify Cobra GREEN**

Run: `GOWORK=off go test ./cmd/margo -run 'TestServeCommand|TestRootHelp' -count=1`

Expected: PASS.

- [ ] **Step 5: Write failing project discovery/build tests**

Use temporary trees to assert omitted/directory input auto-selects root
`site.yaml`, explicit YAML works, raw Markdown trees use local assets, empty
trees return recoverable build errors, output exclusion follows loaded config,
and snapshots include `margo-manifest.json` without writing `dist`.

- [ ] **Step 6: Verify project adapter RED**

Run: `GOWORK=off go test ./cmd/margo -run 'TestResolveServeProject|TestServeProjectBuild' -count=1`

Expected: FAIL on missing resolver/builder.

- [ ] **Step 7: Implement project adapter and production serve function**

Resolve paths relative to `Dependencies.WorkingDirectory`. For config builds,
call `site.BuildConfig`; for raw trees, reuse `discoverSiteSources` and
`site.Build`. Convert results through `devserver.NewSnapshot` after appending
the CLI manifest. Maintain the current configured output path behind a mutex so
the watcher ignore callback follows successful config reloads.

Select a listener through `devserver.Listen`, create the watcher, then run the
coordinator. Print the chosen development URL, report builds/errors, warn for
non-loopback hosts, and invoke `browser.OpenURL` only when `--open` is set.

- [ ] **Step 8: Verify command package GREEN**

Run: `GOWORK=off go test ./cmd/margo -count=1`

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add cmd/margo/serve.go cmd/margo/serve_test.go cmd/margo/root.go cmd/margo/root_test.go
git commit -m "feat(cli): add development site server"
```

### Task 6: End-to-End Reload and Documentation

**Files:**
- Create: `cmd/margo/serve_e2e_test.go`
- Modify: `README.md`
- Modify: `unified_docs_test.go`

**Interfaces:**
- Consumes: completed `margo serve` command and dev-server internals.
- Produces: documented development workflow and cross-layer regression proof.

- [ ] **Step 1: Write failing end-to-end test**

Start the command with a temporary Markdown tree and test-only serve dependencies
that bind `127.0.0.1:0`. Fetch the initial HTML and open the SSE endpoint. Edit
Markdown, observe a later reload generation, fetch changed HTML, cancel context,
and assert clean shutdown plus no output directory.

- [ ] **Step 2: Verify E2E RED**

Run: `GOWORK=off go test ./cmd/margo -run 'TestServeEndToEnd' -count=1`

Expected: FAIL until all cross-layer details are connected.

- [ ] **Step 3: Finish cross-layer wiring**

Make only changes required by the failing E2E test. Preserve in-memory output,
development warnings, and explicit-port semantics.

- [ ] **Step 4: Verify E2E GREEN**

Run: `GOWORK=off go test -race ./cmd/margo -run 'TestServeEndToEnd' -count=1`

Expected: PASS.

- [ ] **Step 5: Write failing documentation assertions**

Extend `unified_docs_test.go` to require `margo serve [INPUT_DIR|CONFIG]`, both
directory/config examples, automatic port fallback, live reload after successful
builds, last-good behavior, loopback default, and explicit development-only /
not-for-production wording.

- [ ] **Step 6: Verify docs RED**

Run: `GOWORK=off go test . -run 'TestREADMEExplainsUnifiedCLIAndReleaseContract' -count=1`

Expected: FAIL on missing README text.

- [ ] **Step 7: Update README CLI reference**

Document command syntax, automatic discovery, port sequence/fallback,
`--open`, in-memory output, watch/reload behavior, build-error recovery,
loopback default, non-loopback warning, and development-only limitations.

- [ ] **Step 8: Verify docs and CLI GREEN**

Run: `GOWORK=off go test . ./cmd/margo -count=1`

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add cmd/margo/serve_e2e_test.go README.md unified_docs_test.go
git commit -m "docs: explain development site serving"
```

### Task 7: Full Verification and Showcase Launch

**Files:**
- Modify only if verification exposes a defect.

**Interfaces:**
- Consumes: complete feature.
- Produces: fresh repository and live-server evidence.

- [ ] **Step 1: Format and verify module state**

Run: `gofmt -w internal/devserver/*.go cmd/margo/serve*.go cmd/margo/root*.go`

Run: `GOWORK=off go mod tidy && GOWORK=off go mod verify`

Expected: formatting completes; module verification prints `all modules verified`.

- [ ] **Step 2: Run complete local gates**

Run: `GOWORK=off go test ./... -count=1`

Run: `GOWORK=off go vet ./...`

Run: `GOWORK=off go test -race ./internal/devserver ./cmd/margo -count=1`

Run: `git diff --check`

Expected: every command exits zero.

- [ ] **Step 3: Build CLI binary**

Run: `GOWORK=off go build -o /tmp/margo-serve-live-reload ./cmd/margo`

Expected: binary builds successfully.

- [ ] **Step 4: Launch Margo showcase with new command**

Run from the worktree:

```bash
/tmp/margo-serve-live-reload serve showcase.yaml
```

Expected: startup reports a loopback development URL, configured showcase loads
with HTTP 200, and the process remains running for user inspection.

- [ ] **Step 5: Verify live showcase behavior**

Fetch `/`, one configured content route, `llms.txt`, `sitemap.xml`, and the SSE
endpoint. Confirm HTML contains the development reload client while source
artifacts and Git status remain unchanged.

- [ ] **Step 6: Final commit if verification required fixes**

```bash
git add <only files changed by verified fixes>
git commit -m "fix(cli): finish development server verification"
```
