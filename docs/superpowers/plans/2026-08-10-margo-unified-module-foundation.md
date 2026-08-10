# Margo Unified Module Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert Margo to one Go module and add a testable Cobra command shell without changing document rendering behavior.

**Architecture:** The root module becomes the only dependency and release boundary. `cmd/margo` contains dependency-injected Cobra constructors while `main.go` remains a minimal process adapter; the root package continues to import no optional Margo subpackages.

**Tech Stack:** Go 1.26.5, Go modules, Cobra v1.10.2, GitHub Actions, standard `testing` package

## Global Constraints

- Module path is exactly `github.com/araihu/margo`; no nested `go.mod` or `go.sum` survives.
- Preserve `github.com/araihu/goshtoso-charts v0.0.2-0.20260803224432-297df2f562e8` exactly; do not add `replace`, a fabricated pseudo-version, proxy, tag, release, or identity.
- One future root tag versions core, charts, deck, PDF, and `cmd/margo`; historical submodule tags are not rewritten.
- The root package must not import `charts`, `deck`, `pdf`, Cobra, or `cmd/margo`.
- Margo never downloads Chromium, a native runtime, a package, or a helper binary.
- Every Go gate runs with `GOWORK=off GOFLAGS=-mod=readonly` after dependencies are resolved intentionally.
- Do not push, merge, tag, release, or publish while executing this plan.

---

## File structure

- `go.mod`, `go.sum`: the only module dependency graph.
- `module_layout_test.go`: executable guard against nested modules and root-to-optional imports.
- `charts/go.mod`, `charts/go.sum`, `pdf/go.mod`, `pdf/go.sum`, `cmd/margo/go.mod`: removed module boundaries.
- `.github/workflows/ci.yml`: one-module CI from repository root.
- `cmd/margo/root.go`: dependency-injected Cobra root constructor and shared streams.
- `cmd/margo/version.go`: Go build metadata, compiled capabilities, and `version` command.
- `cmd/margo/main.go`: production adapter only.
- `cmd/margo/root_test.go`, `cmd/margo/version_test.go`: in-process command contract tests.

### Task 1: Freeze the one-module layout as an executable contract

**Files:**
- Create: `module_layout_test.go`
- Test: `module_layout_test.go`

**Interfaces:**
- Consumes: repository filesystem rooted by the test working directory.
- Produces: `TestRepositoryHasOneGoModule` and `TestRootPackageDoesNotImportOptionalPackages`.

- [ ] **Step 1: Write the failing module-layout tests**

```go
package margo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryHasOneGoModule(t *testing.T) {
	var nested []string
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil { return err }
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "vendor") { return filepath.SkipDir }
		if path != "go.mod" && entry.Name() == "go.mod" { nested = append(nested, filepath.ToSlash(path)) }
		return nil
	})
	if err != nil { t.Fatal(err) }
	if len(nested) != 0 { t.Fatalf("nested modules: %v", nested) }
}

func TestRootPackageDoesNotImportOptionalPackages(t *testing.T) {
	entries, err := filepath.Glob("*.go")
	if err != nil { t.Fatal(err) }
	for _, path := range entries {
		data, err := os.ReadFile(path)
		if err != nil { t.Fatal(err) }
		for _, forbidden := range []string{"github.com/araihu/margo/charts", "github.com/araihu/margo/deck", "github.com/araihu/margo/pdf", "github.com/spf13/cobra"} {
			if strings.Contains(string(data), forbidden) { t.Fatalf("%s imports %s", path, forbidden) }
		}
	}
}
```

- [ ] **Step 2: Run the focused test and verify the intended failure**

Run: `GOWORK=off go test . -run 'TestRepositoryHasOneGoModule|TestRootPackageDoesNotImportOptionalPackages' -count=1`

Expected: `TestRepositoryHasOneGoModule` fails and lists `charts/go.mod`, `pdf/go.mod`, and `cmd/margo/go.mod`.

- [ ] **Step 3: Commit only the red contract**

```bash
git add module_layout_test.go
git commit -m "test: freeze unified module layout"
```

### Task 2: Consolidate dependencies at the root

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Delete: `charts/go.mod`
- Delete: `charts/go.sum`
- Delete: `pdf/go.mod`
- Delete: `pdf/go.sum`
- Delete: `cmd/margo/go.mod`
- Test: `module_layout_test.go`

**Interfaces:**
- Consumes: imports already present below `charts/` and `pdf/` and Cobra v1.10.2.
- Produces: one root graph resolving `github.com/araihu/margo/{charts,pdf,cmd/margo}` as packages in the current module.

- [ ] **Step 1: Record module-file hashes before mutation**

Run: `shasum -a 256 go.mod go.sum charts/go.mod charts/go.sum pdf/go.mod pdf/go.sum cmd/margo/go.mod`

Expected: seven hashes recorded in the task log before consolidation.

- [ ] **Step 2: Add exact external dependencies to the root graph**

Run: `GOWORK=off go get github.com/araihu/goshtoso-charts@v0.0.2-0.20260803224432-297df2f562e8 github.com/chromedp/chromedp@v0.14.2 github.com/chromedp/cdproto@v0.0.0-20250724212937-08a3db8b4327 github.com/spf13/cobra@v1.10.2`

Expected: root `go.mod` gains exact requirements and contains no `replace` directive.

- [ ] **Step 3: Remove the nested module files**

```diff
*** Delete File: charts/go.mod
*** Delete File: charts/go.sum
*** Delete File: pdf/go.mod
*** Delete File: pdf/go.sum
*** Delete File: cmd/margo/go.mod
```

- [ ] **Step 4: Normalize the root graph**

Run: `GOWORK=off go mod tidy`

Expected: root `go.mod` and `go.sum` cover all repository packages, retain the exact Goshtoso Charts version, and contain no self-requirement on `github.com/araihu/margo`.

- [ ] **Step 5: Run the one-module and package gates**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1`

Expected: PASS, including both module layout tests.

- [ ] **Step 6: Verify graph invariants and hashes**

Run: `test "$(find . -name go.mod -not -path './.git/*' -print | sort)" = './go.mod' && ! rg '^replace ' go.mod && ! rg 'github.com/araihu/margo v' go.mod && rg 'github.com/araihu/goshtoso-charts v0.0.2-0.20260803224432-297df2f562e8' go.mod && shasum -a 256 go.mod go.sum`

Expected: every predicate succeeds and two post-consolidation hashes are recorded.

- [ ] **Step 7: Commit the module consolidation**

```bash
git add go.mod go.sum module_layout_test.go charts/go.mod charts/go.sum pdf/go.mod pdf/go.sum cmd/margo/go.mod
git commit -m "build: unify margo module graph"
```

### Task 3: Add the dependency-injected Cobra root and version command

**Files:**
- Create: `cmd/margo/root.go`
- Create: `cmd/margo/version.go`
- Create: `cmd/margo/root_test.go`
- Create: `cmd/margo/version_test.go`
- Modify: `cmd/margo/main.go`

**Interfaces:**
- Consumes: `github.com/spf13/cobra.Command`.
- Produces: `type Dependencies`, `type BuildInfo`, `func NewRootCommand(Dependencies) *cobra.Command`, and `func Execute(context.Context, Dependencies) error`.

- [ ] **Step 1: Write failing help and version tests**

```go
package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRootHelpListsCompleteSurface(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(Dependencies{Stdout: &stdout, Stderr: &stderr, Build: testBuildInfo()})
	cmd.SetArgs([]string{"--help"})
	if err := cmd.ExecuteContext(context.Background()); err != nil { t.Fatal(err) }
	for _, name := range []string{"html", "pdf", "deck", "doctor", "version"} {
		if !strings.Contains(stdout.String(), name) { t.Fatalf("help missing %q", name) }
	}
}

func TestVersionWritesOnlyStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(Dependencies{Stdout: &stdout, Stderr: &stderr, Build: testBuildInfo()})
	cmd.SetArgs([]string{"version"})
	if err := cmd.ExecuteContext(context.Background()); err != nil { t.Fatal(err) }
	want := "margo v0.1.0\nmodule github.com/araihu/margo\ncommit abc123\ngo go1.26.5\nplatform darwin/arm64\nengines chromium,native\n"
	if got := stdout.String(); got != want { t.Fatalf("got %q want %q", got, want) }
	if stderr.Len() != 0 { t.Fatalf("stderr = %q", stderr.String()) }
}

func testBuildInfo() BuildInfo {
	return BuildInfo{Module: "github.com/araihu/margo", Version: "v0.1.0", Commit: "abc123", GoVersion: "go1.26.5", GOOS: "darwin", GOARCH: "arm64", Engines: []string{"chromium", "native"}}
}
```

- [ ] **Step 2: Run tests and verify missing constructors fail compilation**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./cmd/margo -run 'TestRootHelpListsCompleteSurface|TestVersionWritesOnlyStdout' -count=1`

Expected: FAIL because `Dependencies`, `BuildInfo`, and `NewRootCommand` do not exist.

- [ ] **Step 3: Implement the command shell**

```go
type BuildInfo struct {
	Module string
	Version string
	Commit string
	GoVersion string
	GOOS string
	GOARCH string
	Engines []string
}

type Dependencies struct {
	Stdin io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Build BuildInfo
}

func NewRootCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{Use: "margo", SilenceUsage: true, SilenceErrors: true}
	cmd.SetIn(deps.Stdin)
	cmd.SetOut(deps.Stdout)
	cmd.SetErr(deps.Stderr)
	cmd.AddCommand(newPlaceholderCommand("html"), newPlaceholderCommand("pdf"), newPlaceholderCommand("deck"), newPlaceholderCommand("doctor"), newVersionCommand(deps))
	return cmd
}

func Execute(ctx context.Context, deps Dependencies) error {
	return NewRootCommand(deps).ExecuteContext(ctx)
}
```

`newPlaceholderCommand` must return a command whose `RunE` returns `margo.command_not_implemented: <name>`; the later command-integration plan deletes it as each real command lands.

- [ ] **Step 4: Keep `main.go` as the process boundary**

```go
func main() {
	build := ReadBuildInfo()
	err := Execute(context.Background(), Dependencies{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr, Build: build})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

`ReadBuildInfo` reads `runtime/debug.ReadBuildInfo`, normalizes an absent module version to `dev`, reads VCS revision when present, records `runtime.Version`, `runtime.GOOS`, `runtime.GOARCH`, and receives compiled engine capability names from the production dependency assembly.

- [ ] **Step 5: Run focused and full tests**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./cmd/margo ./... -count=1`

Expected: PASS.

- [ ] **Step 6: Commit the CLI shell**

```bash
git add cmd/margo/main.go cmd/margo/root.go cmd/margo/version.go cmd/margo/root_test.go cmd/margo/version_test.go
git commit -m "feat(cli): add unified command shell"
```

### Task 4: Project runtime tasks into renderer-neutral descriptors

**Files:**
- Create: `runtime_projection.go`
- Create: `runtime_projection_test.go`
- Modify: `compiler.go`
- Modify: `result.go`

**Interfaces:**
- Consumes: compiled Mermaid task descriptors retained in `Document.plan`, `RenderInstanceID`, and `RuntimeDescriptor` validation.
- Produces: `func (r *RenderResult) DocumentFingerprint() DocumentFingerprint`, `func (r *RenderResult) RuntimeDescriptor(RenderInstanceID) (RuntimeDescriptor, error)`, and `func ComposeRuntimeDescriptors(DocumentFingerprint, RenderInstanceID, ...RuntimeDescriptor) (RuntimeDescriptor, error)`.

- [ ] **Step 1: Write failing projection and composition tests**

```go
func TestRenderResultProjectsRuntimeDescriptor(t *testing.T) {
	result := mustRenderSource(t, "```mermaid\ngraph TD; A-->B\n```\n")
	descriptor, err := result.RuntimeDescriptor("ri-00000000")
	if err != nil { t.Fatal(err) }
	if descriptor.DocumentFingerprint != result.DocumentFingerprint() { t.Fatal("fingerprint mismatch") }
	if len(descriptor.Tasks) != 1 || descriptor.Tasks[0].Kind != "mermaid" { t.Fatalf("tasks = %#v", descriptor.Tasks) }
	if err := ValidateRuntimeDescriptor(descriptor); err != nil { t.Fatal(err) }
}

func TestComposeRuntimeDescriptorsRebasesTaskIdentities(t *testing.T) {
	first := mustRuntimeDescriptor(t, "ri-00000000", "mermaid")
	second := mustRuntimeDescriptor(t, "ri-00000001", "mermaid")
	fingerprint := DocumentFingerprint(sha256.Sum256([]byte("deck")))
	merged, err := ComposeRuntimeDescriptors(fingerprint, "ri-00000002", first, second)
	if err != nil { t.Fatal(err) }
	if len(merged.Tasks) != 2 { t.Fatalf("tasks = %d", len(merged.Tasks)) }
	if merged.Tasks[0].ID == first.Tasks[0].ID || merged.Tasks[1].ID == second.Tasks[0].ID { t.Fatal("task IDs were not rebased") }
	if err := ValidateRuntimeDescriptor(merged); err != nil { t.Fatal(err) }
}
```

- [ ] **Step 2: Run tests and verify missing methods fail compilation**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test . -run 'TestRenderResultProjectsRuntimeDescriptor|TestComposeRuntimeDescriptors' -count=1`

Expected: FAIL because the three public projection APIs do not exist.

- [ ] **Step 3: Retain immutable runtime task templates in `RenderResult`**

Add private `runtimeTaskTemplate { kind string; inputSHA256 string; dependsOn []int }`. During `Compiler.Render`, canonicalize each compiled Mermaid descriptor, hash its canonical bytes, and store templates plus `document.documentFingerprint`. Plain Markdown stores an explicit empty template slice.

```go
type RenderResult struct {
	content templ.Component
	metadata Metadata
	assets AssetSet
	diagnostics []Diagnostic
	htmlRequirements HTMLRequirements
	documentFingerprint DocumentFingerprint
	runtimeTasks []runtimeTaskTemplate
}
```

- [ ] **Step 4: Implement descriptor projection**

`RuntimeDescriptor(instance)` validates the instance, generates IDs in the existing `ri-...:<kind>:<eight-digit-ordinal>:<sha256>` grammar, maps template dependencies, sorts dependency IDs, validates the completed descriptor, and returns defensive slices. Nil results return `runtime.result_required`.

- [ ] **Step 5: Implement deterministic composition**

`ComposeRuntimeDescriptors` validates every input, rejects a zero destination fingerprint, rebases every task into the destination instance using one ordinal counter per kind, rewrites intra-descriptor dependencies through an old-ID/new-ID map, preserves descriptor argument order, validates the result, and never aliases input slices.

- [ ] **Step 6: Run runtime and repository tests**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test . -run 'Test.*Runtime|TestRenderResultProjects|TestComposeRuntime' -count=1 && GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1`

Expected: PASS.

- [ ] **Step 7: Commit the runtime projection seam**

```bash
git add compiler.go result.go runtime_projection.go runtime_projection_test.go
git commit -m "feat: project render runtime descriptors"
```

### Task 5: Replace multi-module CI with root-only gates

**Files:**
- Modify: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: the single root `go.mod` and all packages below it.
- Produces: CI that runs formatting, vet, tests, and the nested-module guard from repository root.

- [ ] **Step 1: Change the CI job contract**

Replace the module-discovery loop with these commands:

```yaml
- name: Verify module layout
  run: test "$(find . -name go.mod -not -path './.git/*' -print | sort)" = './go.mod'
- name: Test
  env:
    GOWORK: "off"
    GOFLAGS: "-mod=readonly"
  run: go test ./... -count=1
- name: Vet
  env:
    GOWORK: "off"
    GOFLAGS: "-mod=readonly"
  run: go vet ./...
```

- [ ] **Step 2: Run the local CI equivalents**

Run: `test "$(find . -name go.mod -not -path './.git/*' -print | sort)" = './go.mod' && test -z "$(gofmt -l $(find . -name '*.go' -not -path './.git/*'))" && GOWORK=off GOFLAGS=-mod=readonly go vet ./... && GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1`

Expected: PASS with no formatting output.

- [ ] **Step 3: Commit the CI boundary**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: test margo as one module"
```

### Task 6: Foundation checkpoint

**Files:**
- Verify: all files changed by Tasks 1-5.

**Interfaces:**
- Consumes: the complete foundation slice from Tasks 1-5.
- Produces: an identity-bound checkpoint ready for independent review.

- [ ] **Step 1: Run final gates**

Run: `GOWORK=off GOFLAGS=-mod=readonly go test ./... -count=1 && GOWORK=off GOFLAGS=-mod=readonly go vet ./... && git diff --check`

Expected: PASS.

- [ ] **Step 2: Record the immutable checkpoint**

Run: `git status --short --branch && git rev-parse HEAD HEAD^{tree} && shasum -a 256 go.mod go.sum`

Expected: clean worktree, exact HEAD/tree, and module hashes captured for review.
