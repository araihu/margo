package platform

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoDownloadProbeForcesOfflineGoEnvironment(t *testing.T) {
	t.Parallel()

	contractsPath := filepath.Join(t.TempDir(), "runner-contracts.json")
	writePlatformTestFile(t, contractsPath, validRunnerContractsJSON(`[
    "go", "test", "./pdf/platform", "-run", "^TestProbeWindowsWebView2$", "-count=1"
  ]`))

	executor := &recordingExecutor{exitCode: 0}
	result, err := runProbe(context.Background(), contractsPath, RunnerWindowsWebView2, t.TempDir(), executor)
	if err != nil {
		t.Fatalf("runProbe() error = %v", err)
	}
	if result.RunnerID != RunnerWindowsWebView2 || result.ExitCode != 0 {
		t.Fatalf("result = %+v", result)
	}

	want := map[string]string{
		"GOENV":                     "off",
		"GOFLAGS":                   "-mod=readonly",
		"GOPROXY":                   "off",
		"GOSUMDB":                   "off",
		"GOTOOLCHAIN":               "local",
		"GOWORK":                    "off",
		"MARGO_PDF_PROBE_EXECUTION": "1",
		"MARGO_PDF_RUNNER_ID":       string(RunnerWindowsWebView2),
	}
	for name, value := range want {
		if got := environmentValue(executor.environment, name); got != value {
			t.Fatalf("%s = %q, want %q", name, got, value)
		}
	}
}

func TestNoDownloadProbeRejectsUnexpectedExitCode(t *testing.T) {
	t.Parallel()

	contractsPath := filepath.Join(t.TempDir(), "runner-contracts.json")
	writePlatformTestFile(t, contractsPath, validRunnerContractsJSON(`[
    "go", "test", "./pdf/platform", "-run", "^TestProbeWindowsWebView2$", "-count=1"
  ]`))

	_, err := runProbe(context.Background(), contractsPath, RunnerWindowsWebView2, t.TempDir(), &recordingExecutor{exitCode: 1})
	requirePlatformErrorCode(t, err, "pdf.platform_runtime_evidence_required")
}

func TestNoDownloadEnvironmentReplacesAmbientOverrides(t *testing.T) {
	t.Parallel()

	environment := offlineProbeEnvironment([]string{
		"GOPROXY=https://example.invalid/proxy",
		"goflags=-mod=mod",
		"GOTOOLCHAIN=auto",
		"KEEP=value",
	}, RunnerWindowsWebView2, "/tmp/sdk-evidence", "/tmp/runtime-evidence")
	for _, name := range []string{"GOPROXY", "GOFLAGS", "GOTOOLCHAIN"} {
		if count := environmentNameCount(environment, name); count != 1 {
			t.Fatalf("%s entries = %d, want 1: %v", name, count, environment)
		}
	}
	if got := environmentValue(environment, "GOPROXY"); got != "off" {
		t.Fatalf("GOPROXY = %q, want off", got)
	}
	if got := environmentValue(environment, "KEEP"); got != "value" {
		t.Fatalf("KEEP = %q, want value", got)
	}
}

type recordingExecutor struct {
	environment []string
	exitCode    int
}

func (executor *recordingExecutor) Run(_ context.Context, _ string, _ []string, environment []string) ([]byte, []byte, int, error) {
	executor.environment = append([]string(nil), environment...)
	return []byte("probe ok"), nil, executor.exitCode, nil
}

func environmentValue(environment []string, name string) string {
	prefix := name + "="
	for _, entry := range environment {
		if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix {
			return entry[len(prefix):]
		}
	}
	return ""
}

func environmentNameCount(environment []string, name string) int {
	count := 0
	for _, entry := range environment {
		entryName, _, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(entryName, name) {
			count++
		}
	}
	return count
}
