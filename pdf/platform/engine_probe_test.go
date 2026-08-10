package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPlatformToolchainLockRejectsUnrecordedRunner(t *testing.T) {
	t.Parallel()

	lockPath := filepath.Join(t.TempDir(), "platform-toolchain.lock")
	writePlatformTestFile(t, lockPath, `{
  "schemaVersion": "margo/pdf-platform-toolchain/v2",
  "modulePath": "github.com/araihu/margo",
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
  "schemaVersion": "margo/pdf-platform-toolchain/v2",
  "modulePath": "github.com/araihu/margo",
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

func TestPlatformToolchainLockRejectsWrongRootIdentity(t *testing.T) {
	t.Parallel()

	modules := []toolchainModule{
		{
			Path:     "github.com/araihu/margo",
			Version:  "v0.0.0-20260808231103-000000000000",
			Sum:      "h1:SFWnf6NlyC5IDfK+vvImwpVkuRlp6DdmYZ5wV9Mviuo=",
			GoModSum: "h1:t5vzt4j6VTYIJsreX2+d/Cr37vftL1Cd0PiXIcre11U=",
		},
		{
			Path:     "github.com/chromedp/chromedp",
			Version:  "v0.14.2",
			Sum:      "h1:r3b/WtwM50RsBZHMUm9fsNhhzRStTHrKdr2zmwbZSzM=",
			GoModSum: "h1:rHzAv60xDE7VNy/MYtTUrYreSc0ujt2O1/C3bzctYBo=",
		},
	}

	err := validateModules(modules)
	requirePlatformErrorCode(t, err, "pdf.platform_lock_invalid")
}

func TestPlatformToolchainLockRepositoryIdentity(t *testing.T) {
	t.Parallel()

	lockPath := moduleArgumentPath(*platformToolchainLockArgument)
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("ReadFile(repository lock) error = %v", err)
	}
	var lock toolchainLock
	if err := decodeStrictJSON(data, &lock); err != nil {
		t.Fatalf("decodeStrictJSON(repository lock) error = %v", err)
	}
	if err := validateToolchainLock(lock, ".."); err != nil {
		t.Fatalf("validateToolchainLock(repository lock) error = %v", err)
	}
	for _, runner := range lock.Runners {
		_, sdkError := os.Stat(runner.SDKEvidencePath)
		_, runtimeError := os.Stat(runner.RuntimeEvidencePath)
		err := VerifyPlatformToolchain(lockPath, runner.ID)
		if sdkError == nil && runtimeError == nil {
			if err != nil {
				t.Fatalf("VerifyPlatformToolchain(%q) error = %v", runner.ID, err)
			}
			continue
		}
		requirePlatformErrorCode(t, err, "pdf.platform_runtime_evidence_required")
	}
}

func TestPlatformToolchainLockRepositoryBootstrapAvailableRunner(t *testing.T) {
	if os.Getenv("MARGO_PDF_PROBE_EXECUTION") == "1" {
		t.Skip("parent-only repository bootstrap test")
	}

	lockPath, err := filepath.Abs(moduleArgumentPath(*platformToolchainLockArgument))
	if err != nil {
		t.Fatalf("Abs(repository lock) error = %v", err)
	}
	contractsPath, err := filepath.Abs(moduleArgumentPath(*runnerContractsArgument))
	if err != nil {
		t.Fatalf("Abs(repository contracts) error = %v", err)
	}
	workingDirectory, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Abs(repository working directory) error = %v", err)
	}

	runnerID := hostProbeRunner()
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("ReadFile(repository lock) error = %v", err)
	}
	var lock toolchainLock
	if err := decodeStrictJSON(data, &lock); err != nil {
		t.Fatalf("decodeStrictJSON(repository lock) error = %v", err)
	}
	selected, found := findRunner(lock.Runners, runnerID)
	if !found {
		t.Fatalf("runner %q is absent from repository lock", runnerID)
	}
	if _, err := os.Stat(selected.SDKEvidencePath); err != nil {
		t.Skipf("runner %q SDK evidence unavailable: %v", runnerID, err)
	}
	if _, err := os.Stat(selected.RuntimeEvidencePath); err != nil {
		t.Skipf("runner %q runtime evidence unavailable: %v", runnerID, err)
	}
	result, err := Bootstrap(context.Background(), lockPath, contractsPath, runnerID, workingDirectory)
	if err != nil {
		t.Fatalf("Bootstrap(repository, %q) error = %v", runnerID, err)
	}
	if result.SDKVersion == "" || result.RuntimeVersion == "" || !validSHA256(result.SourceDigest) {
		t.Fatalf("result = %+v", result)
	}
}

func TestPlatformBootstrapExecutesLockBoundOfflineProbe(t *testing.T) {
	if os.Getenv("MARGO_PDF_PROBE_EXECUTION") == "1" {
		t.Skip("parent-only bootstrap test")
	}

	lockPath := writeSyntheticToolchainLock(t)
	contractsPath, err := filepath.Abs("runner-contracts.json")
	if err != nil {
		t.Fatalf("Abs(runner-contracts.json) error = %v", err)
	}
	workingDirectory, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}

	result, err := Bootstrap(context.Background(), lockPath, contractsPath, hostProbeRunner(), workingDirectory)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if result.ExitCode != 0 || !strings.Contains(result.Stdout, "ok") || strings.Contains(result.Stdout, "no tests to run") || result.SDKVersion != "sdk-test-1" || result.RuntimeVersion != "runtime-test-2" || !validSHA256(result.SourceDigest) {
		t.Fatalf("result = %+v", result)
	}
}

func TestPlatformBootstrapReportsChromiumVersionWithoutEnforcingBrowserBuild(t *testing.T) {
	if os.Getenv("MARGO_PDF_PROBE_EXECUTION") == "1" {
		t.Skip("parent-only bootstrap test")
	}

	lockPath := writeSyntheticToolchainLock(t)
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("ReadFile(lock) error = %v", err)
	}
	var lock toolchainLock
	if err := decodeStrictJSON(data, &lock); err != nil {
		t.Fatalf("decodeStrictJSON(lock) error = %v", err)
	}
	for index := range lock.Runners {
		if lock.Runners[index].ID == RunnerChromiumCDP {
			lock.Runners[index].VersionPolicy = "tested-version-reported"
		}
	}
	lock.RecordDigest = ""
	payload, err := json.Marshal(lock)
	if err != nil {
		t.Fatalf("Marshal(preimage) error = %v", err)
	}
	digest := sha256.Sum256(payload)
	lock.RecordDigest = hex.EncodeToString(digest[:])
	encoded, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(lock) error = %v", err)
	}
	if err := os.WriteFile(lockPath, append(encoded, '\n'), 0o600); err != nil {
		t.Fatalf("WriteFile(lock) error = %v", err)
	}

	contractsPath, err := filepath.Abs("runner-contracts.json")
	if err != nil {
		t.Fatalf("Abs(runner-contracts.json) error = %v", err)
	}
	workingDirectory, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}
	result, err := Bootstrap(context.Background(), lockPath, contractsPath, RunnerChromiumCDP, workingDirectory)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if result.RuntimeVersion != "runtime-test-2" {
		t.Fatalf("RuntimeVersion = %q, want reported version %q", result.RuntimeVersion, "runtime-test-2")
	}
}

func hostProbeRunner() RunnerID {
	switch runtime.GOOS {
	case "windows":
		return RunnerWindowsWebView2
	case "darwin":
		return RunnerDarwinWKWebView
	case "linux":
		return RunnerLinuxWebKitGTK
	default:
		return RunnerChromiumCDP
	}
}

func TestPlatformBootstrapFailsClosedForWrongHostRunner(t *testing.T) {
	if os.Getenv("MARGO_PDF_PROBE_EXECUTION") == "1" {
		t.Skip("parent-only bootstrap test")
	}

	lockPath := writeSyntheticToolchainLock(t)
	contractsPath, err := filepath.Abs("runner-contracts.json")
	if err != nil {
		t.Fatalf("Abs(runner-contracts.json) error = %v", err)
	}
	workingDirectory, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}

	wrongRunner := RunnerWindowsWebView2
	if runtime.GOOS == "windows" {
		wrongRunner = RunnerLinuxWebKitGTK
	}
	_, err = Bootstrap(context.Background(), lockPath, contractsPath, wrongRunner, workingDirectory)
	requirePlatformErrorCode(t, err, "pdf.platform_runtime_evidence_required")
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

func TestProbeWindowsWebView2(t *testing.T) {
	requireHostEvidenceProbe(t, RunnerWindowsWebView2, "windows")
}

func TestProbeDarwinWKWebView(t *testing.T) {
	requireHostEvidenceProbe(t, RunnerDarwinWKWebView, "darwin")
}

func TestProbeLinuxWebKitGTK(t *testing.T) {
	requireHostEvidenceProbe(t, RunnerLinuxWebKitGTK, "linux")
}

func TestProbeChromiumCDP(t *testing.T) {
	requireHostEvidenceProbe(t, RunnerChromiumCDP, "")
}

func requireHostEvidenceProbe(t *testing.T, runnerID RunnerID, requiredGOOS string) {
	t.Helper()
	if os.Getenv("MARGO_PDF_PROBE_EXECUTION") != "1" {
		t.Skip("bootstrap-only host evidence probe")
	}
	if got := RunnerID(os.Getenv("MARGO_PDF_RUNNER_ID")); got != runnerID {
		t.Fatalf("runner ID = %q, want %q", got, runnerID)
	}
	if requiredGOOS != "" && runtime.GOOS != requiredGOOS {
		t.Fatalf("pdf.platform_runtime_evidence_required: runner %q requires GOOS %q, got %q", runnerID, requiredGOOS, runtime.GOOS)
	}
	for _, environmentName := range []string{"MARGO_PDF_SDK_EVIDENCE_PATH", "MARGO_PDF_RUNTIME_EVIDENCE_PATH"} {
		evidencePath := os.Getenv(environmentName)
		if !filepath.IsAbs(evidencePath) {
			t.Fatalf("%s must be absolute", environmentName)
		}
		if _, err := readEvidenceVersion(evidencePath); err != nil {
			t.Fatalf("%s is invalid: %v", environmentName, err)
		}
	}
}

func writeSyntheticToolchainLock(t *testing.T) string {
	t.Helper()
	temporaryDirectory := t.TempDir()
	moduleDirectory := filepath.Join(temporaryDirectory, "pdf")
	probePath := filepath.Join(moduleDirectory, "platform", "engine_probe_test.go")
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
	sdkEvidencePath := filepath.Join(moduleDirectory, "sdk-evidence.txt")
	runtimeEvidencePath := filepath.Join(moduleDirectory, "runtime-evidence.txt")
	writePlatformTestFile(t, sdkEvidencePath, "sdk-test-1\n")
	writePlatformTestFile(t, runtimeEvidencePath, "runtime-test-2\n")

	runners := make([]toolchainRunner, 0, len(knownRunnerIDs))
	for _, id := range []RunnerID{RunnerWindowsWebView2, RunnerDarwinWKWebView, RunnerLinuxWebKitGTK, RunnerChromiumCDP} {
		runners = append(runners, toolchainRunner{
			ID: id, Probe: "pdf/platform/engine_probe_test.go",
			SourceDigest:    hex.EncodeToString(probeDigest[:]),
			SDKEvidencePath: "pdf/sdk-evidence.txt", RuntimeEvidencePath: "pdf/runtime-evidence.txt",
			VersionPolicy: "tested-version-reported",
		})
	}
	lock := toolchainLock{
		SchemaVersion: platformToolchainSchema,
		ModulePath:    "github.com/araihu/margo",
		Go:            toolchainGo{Version: "1.26.5"},
		Modules: []toolchainModule{
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
	lockPath := filepath.Join(moduleDirectory, "platform-toolchain.lock")
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
