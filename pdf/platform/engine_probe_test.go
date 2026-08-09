package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlatformToolchainLockRejectsUnrecordedRunner(t *testing.T) {
	t.Parallel()

	lockPath := filepath.Join(t.TempDir(), "platform-toolchain.lock")
	writePlatformTestFile(t, lockPath, `{
  "schemaVersion": "margo/pdf-platform-toolchain/v1",
  "go": {"version": "1.26.5"},
  "modules": [],
  "nodeHarness": {},
  "muambaTool": {},
  "runners": [],
  "networkPolicy": "no-download",
  "recordDigest": ""
}`)

	err := VerifyPlatformToolchain(lockPath, RunnerWindowsWebView2)
	requirePlatformErrorCode(t, err, "pdf.platform_runtime_evidence_required")
}

func TestPlatformToolchainLockRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	lockPath := filepath.Join(t.TempDir(), "platform-toolchain.lock")
	writePlatformTestFile(t, lockPath, `{
  "schemaVersion": "margo/pdf-platform-toolchain/v1",
  "go": {"version": "1.26.5"},
  "modules": [],
  "nodeHarness": {},
  "muambaTool": {},
  "runners": [],
  "networkPolicy": "no-download",
  "recordDigest": "",
  "downloadURL": "https://example.invalid/browser"
}`)

	err := VerifyPlatformToolchain(lockPath, RunnerWindowsWebView2)
	requirePlatformErrorCode(t, err, "pdf.platform_lock_invalid")
}

func TestPlatformBootstrapExecutesLockBoundOfflineProbe(t *testing.T) {
	if os.Getenv("MARGO_PDF_PROBE_EXECUTION") == "1" {
		t.Skip("parent-only bootstrap test")
	}

	lockPath := writeSyntheticToolchainLock(t)
	contractsPath := filepath.Join(t.TempDir(), "runner-contracts.json")
	writePlatformTestFile(t, contractsPath, validRunnerContractsJSON(`[
    "go", "test", "./platform", "-run", "^TestProbeSynthetic$", "-count=1"
  ]`))
	workingDirectory, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}

	result, err := Bootstrap(context.Background(), lockPath, contractsPath, RunnerWindowsWebView2, workingDirectory)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if result.ExitCode != 0 || !strings.Contains(result.Stdout, "ok") || result.SDKVersion != "sdk-test-1" || result.RuntimeVersion != "runtime-test-2" || !validSHA256(result.SourceDigest) {
		t.Fatalf("result = %+v", result)
	}
}

func TestProbeSynthetic(t *testing.T) {
	if os.Getenv("MARGO_PDF_PROBE_EXECUTION") != "1" {
		t.Skip("bootstrap-only probe")
	}
	for name, value := range map[string]string{
		"GOENV": "off", "GOFLAGS": "-mod=readonly", "GOPROXY": "off",
		"GOSUMDB": "off", "GOTOOLCHAIN": "local", "GOWORK": "off",
	} {
		if got := os.Getenv(name); got != value {
			t.Fatalf("%s = %q, want %q", name, got, value)
		}
	}
}

func writeSyntheticToolchainLock(t *testing.T) string {
	t.Helper()
	temporaryDirectory := t.TempDir()
	probePath := filepath.Join(temporaryDirectory, "platform", "engine_probe_test.go")
	if err := os.MkdirAll(filepath.Dir(probePath), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	probeBytes, err := os.ReadFile("engine_probe_test.go")
	if err != nil {
		t.Fatalf("ReadFile(engine_probe_test.go) error = %v", err)
	}
	if err := os.WriteFile(probePath, probeBytes, 0o600); err != nil {
		t.Fatalf("WriteFile(probe) error = %v", err)
	}
	probeDigest := sha256.Sum256(probeBytes)
	sdkEvidencePath := filepath.Join(temporaryDirectory, "sdk-evidence.txt")
	runtimeEvidencePath := filepath.Join(temporaryDirectory, "runtime-evidence.txt")
	writePlatformTestFile(t, sdkEvidencePath, "sdk-test-1\n")
	writePlatformTestFile(t, runtimeEvidencePath, "runtime-test-2\n")

	runners := make([]toolchainRunner, 0, len(knownRunnerIDs))
	for _, id := range []RunnerID{RunnerWindowsWebView2, RunnerDarwinWKWebView, RunnerLinuxWebKitGTK, RunnerChromiumCDP} {
		policy := "host-evidence-exact"
		if id == RunnerChromiumCDP {
			policy = "exact"
		}
		runners = append(runners, toolchainRunner{
			ID: id, Probe: "platform/engine_probe_test.go",
			SourceDigest:    hex.EncodeToString(probeDigest[:]),
			SDKEvidencePath: sdkEvidencePath, RuntimeEvidencePath: runtimeEvidencePath,
			VersionPolicy: policy,
		})
	}
	lock := toolchainLock{
		SchemaVersion: platformToolchainSchema,
		Go:            toolchainGo{Version: "1.26.5"},
		Modules: []toolchainModule{
			{Path: "github.com/araihu/margo", Version: "v0.0.1", Sum: "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", GoModSum: "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="},
			{Path: "github.com/chromedp/chromedp", Version: "v0.14.2", Sum: "h1:r3b/WtwM50RsBZHMUm9fsNhhzRStTHrKdr2zmwbZSzM=", GoModSum: "h1:rHzAv60xDE7VNy/MYtTUrYreSc0ujt2O1/C3bzctYBo="},
		},
		NodeHarness: nodeHarness{
			NodeVersion: "v26.5.0", NPMVersion: "11.17.0",
			DOMStrategy: "DOMParser/XMLSerializer/CSSOM+css-tree", CSSTreeVersion: "3.1.0",
			PlaywrightVersion: "1.52.0", ChromiumRevision: "1169",
			ChromiumArchiveSHA256: strings.Repeat("a", 64),
		},
		MuambaTool:    muambaTool{Path: "github.com/araihu/muamba/cmd/muamba", Version: "v0.0.3"},
		Runners:       runners,
		NetworkPolicy: "no-download",
	}
	payload, err := json.Marshal(lock)
	if err != nil {
		t.Fatalf("Marshal(preimage) error = %v", err)
	}
	recordDigest := sha256.Sum256(payload)
	lock.RecordDigest = hex.EncodeToString(recordDigest[:])
	encoded, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() error = %v", err)
	}
	lockPath := filepath.Join(temporaryDirectory, "platform-toolchain.lock")
	if err := os.WriteFile(lockPath, append(encoded, '\n'), 0o600); err != nil {
		t.Fatalf("WriteFile(lock) error = %v", err)
	}
	return lockPath
}

func writePlatformTestFile(t *testing.T, filePath, contents string) {
	t.Helper()
	if err := os.WriteFile(filePath, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", filePath, err)
	}
}

func requirePlatformErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), code) {
		t.Fatalf("error = %v, want code %q", err, code)
	}
}
