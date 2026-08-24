// Package platform verifies the locked platform probe contract without
// selecting, downloading, or implementing a PDF engine.
package platform

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	platformToolchainSchema = "margo/pdf-platform-toolchain/v2"
	runnerContractsSchema   = "margo/pdf-platform-contracts/v2"
)

// RunnerID is one locked native or installed-browser probe identity.
type RunnerID string

const (
	RunnerWindowsWebView2 RunnerID = "windows-webview2/v2"
	RunnerDarwinWKWebView RunnerID = "darwin-wkwebview/v2"
	RunnerLinuxWebKitGTK  RunnerID = "linux-webkitgtk/v2"
	RunnerChromiumCDP     RunnerID = "chromium-cdp/v2"
)

var knownRunnerIDs = map[RunnerID]struct{}{
	RunnerWindowsWebView2: {},
	RunnerDarwinWKWebView: {},
	RunnerLinuxWebKitGTK:  {},
	RunnerChromiumCDP:     {},
}

var lockedRunnerProbePatterns = map[RunnerID]string{
	RunnerWindowsWebView2: "^TestProbeWindowsWebView2$",
	RunnerDarwinWKWebView: "^TestProbeDarwinWKWebView$",
	RunnerLinuxWebKitGTK:  "^TestProbeLinuxWebKitGTK$",
	RunnerChromiumCDP:     "^TestProbeChromiumCDP$",
}

// RunnerContract describes one reviewed probe command. Slices returned by
// Runner are copies, so callers cannot mutate the loaded contract.
type RunnerContract struct {
	Command          []string `json:"command"`
	ExpectedExitCode int      `json:"expectedExitCode"`
	OwnedSourcePaths []string `json:"ownedSourcePaths"`
	OwnedProbePaths  []string `json:"ownedProbePaths"`
}

// RunnerContracts is an immutable set of validated runner contracts.
type RunnerContracts struct {
	runners map[RunnerID]RunnerContract
}

// Runner returns a defensive copy of a validated runner contract.
func (contracts RunnerContracts) Runner(id RunnerID) (RunnerContract, bool) {
	contract, ok := contracts.runners[id]
	if !ok {
		return RunnerContract{}, false
	}
	return cloneRunnerContract(contract), true
}

// ProbeResult records process evidence without exposing mutable byte slices.
type ProbeResult struct {
	RunnerID       RunnerID
	ExitCode       int
	Stdout         string
	Stderr         string
	SDKVersion     string
	RuntimeVersion string
	SourceDigest   string
}

type runnerContractsDocument struct {
	SchemaVersion string                      `json:"schemaVersion"`
	Runners       map[RunnerID]RunnerContract `json:"runners"`
}

// LoadRunnerContracts strictly decodes and validates no-download probe
// commands. Only package-local Go tests are executable probe commands in v1.
func LoadRunnerContracts(contractsPath string) (RunnerContracts, error) {
	data, err := os.ReadFile(contractsPath)
	if err != nil {
		return RunnerContracts{}, platformError("pdf.platform_contract_invalid", err.Error())
	}
	var document runnerContractsDocument
	if err := decodeStrictJSON(data, &document); err != nil {
		return RunnerContracts{}, platformError("pdf.platform_contract_invalid", err.Error())
	}
	if document.SchemaVersion != runnerContractsSchema || len(document.Runners) != len(knownRunnerIDs) {
		return RunnerContracts{}, platformError("pdf.platform_contract_invalid", "runner contract schema or runner set is invalid")
	}

	validated := make(map[RunnerID]RunnerContract, len(document.Runners))
	for id, contract := range document.Runners {
		if _, ok := knownRunnerIDs[id]; !ok {
			return RunnerContracts{}, platformError("pdf.platform_contract_invalid", fmt.Sprintf("runner %q is not recorded by v2", id))
		}
		if err := validateRunnerContract(id, contract); err != nil {
			return RunnerContracts{}, err
		}
		validated[id] = cloneRunnerContract(contract)
	}
	return RunnerContracts{runners: validated}, nil
}

func validateRunnerContract(id RunnerID, contract RunnerContract) error {
	if !isPackageLocalGoTestProbe(contract.Command) {
		return platformError("pdf.platform_download_forbidden", "probe command must be one locked package-local go test")
	}
	if contract.Command[4] != lockedRunnerProbePatterns[id] {
		return platformError("pdf.platform_contract_invalid", fmt.Sprintf("runner %q probe command does not match its identity", id))
	}
	if contract.ExpectedExitCode != 0 {
		return platformError("pdf.platform_contract_invalid", "probe expected exit code must be zero")
	}
	if err := validateOwnedPaths(contract.OwnedSourcePaths); err != nil {
		return err
	}
	if err := validateOwnedPaths(contract.OwnedProbePaths); err != nil {
		return err
	}
	return nil
}

func isPackageLocalGoTestProbe(command []string) bool {
	if len(command) != 6 || command[0] != "go" || command[1] != "test" || command[2] != "./pdf/platform" || command[3] != "-run" || command[5] != "-count=1" {
		return false
	}
	pattern := command[4]
	if !strings.HasPrefix(pattern, "^TestProbe") || !strings.HasSuffix(pattern, "$") {
		return false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(pattern, "^TestProbe"), "$")
	if name == "" {
		return false
	}
	for _, character := range name {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' {
			return false
		}
	}
	return true
}

func validateOwnedPaths(paths []string) error {
	if len(paths) == 0 {
		return platformError("pdf.platform_contract_invalid", "owned path set must not be empty")
	}
	seen := make(map[string]struct{}, len(paths))
	for _, filePath := range paths {
		if !safeRelativePath(filePath) {
			return platformError("pdf.platform_contract_invalid", fmt.Sprintf("owned path %q escapes the repository", filePath))
		}
		if _, duplicate := seen[filePath]; duplicate {
			return platformError("pdf.platform_contract_invalid", fmt.Sprintf("owned path %q is duplicated", filePath))
		}
		seen[filePath] = struct{}{}
	}
	return nil
}

func safeRelativePath(filePath string) bool {
	if filePath == "" || strings.ContainsAny(filePath, "\\:") || filepath.IsAbs(filePath) || path.Clean(filePath) != filePath || filePath == "." || strings.HasPrefix(filePath, "../") {
		return false
	}
	for _, character := range filePath {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func cloneRunnerContract(contract RunnerContract) RunnerContract {
	contract.Command = append([]string(nil), contract.Command...)
	contract.OwnedSourcePaths = append([]string(nil), contract.OwnedSourcePaths...)
	contract.OwnedProbePaths = append([]string(nil), contract.OwnedProbePaths...)
	return contract
}

type probeExecutor interface {
	Run(context.Context, string, []string, []string) ([]byte, []byte, int, error)
}

type operatingSystemExecutor struct{}

func (operatingSystemExecutor) Run(ctx context.Context, workingDirectory string, command, environment []string) ([]byte, []byte, int, error) {
	process := exec.CommandContext(ctx, command[0], command[1:]...)
	process.Dir = workingDirectory
	process.Env = environment
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	process.Stdout = &stdout
	process.Stderr = &stderr
	err := process.Run()
	if err == nil {
		return stdout.Bytes(), stderr.Bytes(), 0, nil
	}
	var exitError *exec.ExitError
	if ok := errors.As(err, &exitError); ok {
		return stdout.Bytes(), stderr.Bytes(), exitError.ExitCode(), nil
	}
	return stdout.Bytes(), stderr.Bytes(), -1, err
}

func runProbe(ctx context.Context, contractsPath string, runnerID RunnerID, workingDirectory string, executor probeExecutor) (ProbeResult, error) {
	return runProbeWithEvidence(ctx, contractsPath, runnerID, workingDirectory, "", "", executor)
}

func runProbeWithEvidence(ctx context.Context, contractsPath string, runnerID RunnerID, workingDirectory, sdkEvidencePath, runtimeEvidencePath string, executor probeExecutor) (ProbeResult, error) {
	if ctx == nil || executor == nil {
		return ProbeResult{}, platformError("pdf.platform_contract_invalid", "probe context and executor are required")
	}
	workingDirectoryInfo, err := os.Stat(workingDirectory)
	if !filepath.IsAbs(workingDirectory) || err != nil || !workingDirectoryInfo.IsDir() {
		return ProbeResult{}, platformError("pdf.platform_contract_invalid", "probe working directory must be an existing absolute directory")
	}
	contracts, err := LoadRunnerContracts(contractsPath)
	if err != nil {
		return ProbeResult{}, err
	}
	contract, ok := contracts.Runner(runnerID)
	if !ok {
		return ProbeResult{}, platformError("pdf.platform_runtime_evidence_required", fmt.Sprintf("runner %q is not recorded", runnerID))
	}
	stdout, stderr, exitCode, err := executor.Run(ctx, workingDirectory, contract.Command, offlineProbeEnvironment(os.Environ(), runnerID, sdkEvidencePath, runtimeEvidencePath))
	if err != nil {
		return ProbeResult{}, platformError("pdf.platform_runtime_evidence_required", err.Error())
	}
	result := ProbeResult{RunnerID: runnerID, ExitCode: exitCode, Stdout: string(stdout), Stderr: string(stderr)}
	if exitCode != contract.ExpectedExitCode {
		return ProbeResult{}, platformError("pdf.platform_runtime_evidence_required", fmt.Sprintf("runner %q probe exited with %d", runnerID, exitCode))
	}
	return result, nil
}

// Bootstrap verifies host evidence and the lock-bound probe mapping before it
// executes one package-local probe with download paths disabled.
func Bootstrap(ctx context.Context, lockPath, contractsPath string, runnerID RunnerID, workingDirectory string) (ProbeResult, error) {
	if err := VerifyPlatformToolchain(lockPath, runnerID); err != nil {
		return ProbeResult{}, err
	}
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return ProbeResult{}, platformError("pdf.platform_lock_invalid", err.Error())
	}
	var lock toolchainLock
	if err := decodeStrictJSON(data, &lock); err != nil {
		return ProbeResult{}, platformError("pdf.platform_lock_invalid", err.Error())
	}
	selected, found := findRunner(lock.Runners, runnerID)
	if !found {
		return ProbeResult{}, platformError("pdf.platform_runtime_evidence_required", fmt.Sprintf("runner %q is not recorded", runnerID))
	}
	contracts, err := LoadRunnerContracts(contractsPath)
	if err != nil {
		return ProbeResult{}, err
	}
	contract, found := contracts.Runner(runnerID)
	if !found || !containsPath(contract.OwnedProbePaths, selected.Probe) {
		return ProbeResult{}, platformError("pdf.platform_contract_invalid", fmt.Sprintf("runner %q probe is not bound to its lock", runnerID))
	}
	repositoryRoot := repositoryRootForLockDirectory(filepath.Dir(lockPath))
	sdkEvidencePath := filepath.Join(repositoryRoot, filepath.FromSlash(selected.SDKEvidencePath))
	runtimeEvidencePath := filepath.Join(repositoryRoot, filepath.FromSlash(selected.RuntimeEvidencePath))
	result, err := runProbeWithEvidence(ctx, contractsPath, runnerID, workingDirectory, sdkEvidencePath, runtimeEvidencePath, operatingSystemExecutor{})
	if err != nil {
		return ProbeResult{}, err
	}
	if strings.Contains(result.Stdout, "[no tests to run]") {
		return ProbeResult{}, platformError("pdf.platform_contract_invalid", fmt.Sprintf("runner %q probe test does not exist", runnerID))
	}
	sdkVersion, err := readEvidenceVersion(sdkEvidencePath)
	if err != nil {
		return ProbeResult{}, platformError("pdf.platform_runtime_evidence_required", fmt.Sprintf("runner %q SDK evidence is invalid", runnerID))
	}
	runtimeVersion, err := readEvidenceVersion(runtimeEvidencePath)
	if err != nil {
		return ProbeResult{}, platformError("pdf.platform_runtime_evidence_required", fmt.Sprintf("runner %q runtime evidence is invalid", runnerID))
	}
	result.SDKVersion = sdkVersion
	result.RuntimeVersion = runtimeVersion
	result.SourceDigest = selected.SourceDigest
	return result, nil
}

func containsPath(paths []string, target string) bool {
	for _, filePath := range paths {
		if filePath == target {
			return true
		}
	}
	return false
}

func readEvidenceVersion(evidencePath string) (string, error) {
	data, err := os.ReadFile(evidencePath)
	if err != nil || len(data) == 0 || len(data) > 4096 || !utf8.Valid(data) {
		return "", fmt.Errorf("evidence is unavailable or malformed")
	}
	version := strings.TrimSpace(string(data))
	if version == "" || strings.ContainsAny(version, "\r\n") {
		return "", fmt.Errorf("evidence must contain one version line")
	}
	return version, nil
}

func offlineProbeEnvironment(environment []string, runnerID RunnerID, sdkEvidencePath, runtimeEvidencePath string) []string {
	locked := map[string]string{
		"GOENV": "off", "GOFLAGS": "-mod=readonly", "GOPROXY": "off",
		"GOSUMDB": "off", "GOTOOLCHAIN": "local", "GOWORK": "off",
		"MARGO_PDF_PROBE_EXECUTION": "1", "MARGO_PDF_RUNNER_ID": string(runnerID),
		"MARGO_PDF_SDK_EVIDENCE_PATH": sdkEvidencePath, "MARGO_PDF_RUNTIME_EVIDENCE_PATH": runtimeEvidencePath,
	}
	filtered := make([]string, 0, len(environment)+len(locked))
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		remove := false
		for lockedName := range locked {
			if strings.EqualFold(name, lockedName) {
				remove = true
				break
			}
		}
		if !remove {
			filtered = append(filtered, entry)
		}
	}
	for _, name := range []string{"GOENV", "GOFLAGS", "GOPROXY", "GOSUMDB", "GOTOOLCHAIN", "GOWORK", "MARGO_PDF_PROBE_EXECUTION", "MARGO_PDF_RUNNER_ID", "MARGO_PDF_SDK_EVIDENCE_PATH", "MARGO_PDF_RUNTIME_EVIDENCE_PATH"} {
		filtered = append(filtered, name+"="+locked[name])
	}
	return filtered
}

type toolchainLock struct {
	SchemaVersion string            `json:"schemaVersion"`
	ModulePath    string            `json:"modulePath"`
	Go            toolchainGo       `json:"go"`
	Modules       []toolchainModule `json:"modules"`
	NodeHarness   nodeHarness       `json:"nodeHarness"`
	MuambaTool    muambaTool        `json:"muambaTool"`
	Runners       []toolchainRunner `json:"runners"`
	NetworkPolicy string            `json:"networkPolicy"`
	RecordDigest  string            `json:"recordDigest,omitempty"`
}

type toolchainGo struct {
	Version string `json:"version"`
}

type toolchainModule struct {
	Path     string `json:"path"`
	Version  string `json:"version"`
	Sum      string `json:"sum"`
	GoModSum string `json:"goModSum"`
}

type nodeHarness struct {
	NodeVersion           string `json:"nodeVersion"`
	NPMVersion            string `json:"npmVersion"`
	DOMStrategy           string `json:"domStrategy"`
	CSSTreeVersion        string `json:"cssTreeVersion"`
	PlaywrightVersion     string `json:"playwrightVersion"`
	ChromiumRevision      string `json:"chromiumRevision"`
	ChromiumArchiveSHA256 string `json:"chromiumArchiveSHA256"`
}

type muambaTool struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

type toolchainRunner struct {
	ID                  RunnerID `json:"id"`
	Probe               string   `json:"probe"`
	SourceDigest        string   `json:"sourceDigest"`
	SDKEvidencePath     string   `json:"sdkEvidencePath"`
	RuntimeEvidencePath string   `json:"runtimeEvidencePath"`
	VersionPolicy       string   `json:"versionPolicy"`
}

// VerifyPlatformToolchain validates a selected runner against a strict lock.
// Missing or unavailable host evidence fails closed without downloading it.
func VerifyPlatformToolchain(lockPath string, runnerID RunnerID) error {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return platformError("pdf.platform_lock_invalid", err.Error())
	}
	var lock toolchainLock
	if err := decodeStrictJSON(data, &lock); err != nil {
		return platformError("pdf.platform_lock_invalid", err.Error())
	}
	if lock.SchemaVersion != platformToolchainSchema || lock.NetworkPolicy != "no-download" {
		return platformError("pdf.platform_lock_invalid", "platform lock schema or network policy is invalid")
	}

	selected, found := findRunner(lock.Runners, runnerID)
	if !found {
		return platformError("pdf.platform_runtime_evidence_required", fmt.Sprintf("runner %q is not recorded", runnerID))
	}
	if err := validateToolchainLock(lock, filepath.Dir(lockPath)); err != nil {
		return err
	}
	repositoryRoot := repositoryRootForLockDirectory(filepath.Dir(lockPath))
	if _, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(selected.SDKEvidencePath))); err != nil {
		return platformError("pdf.platform_runtime_evidence_required", fmt.Sprintf("runner %q SDK evidence is unavailable", runnerID))
	}
	if _, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(selected.RuntimeEvidencePath))); err != nil {
		return platformError("pdf.platform_runtime_evidence_required", fmt.Sprintf("runner %q runtime evidence is unavailable", runnerID))
	}
	return nil
}

func validateToolchainLock(lock toolchainLock, lockDirectory string) error {
	if lock.Go.Version != "1.27.0" {
		return platformError("pdf.platform_lock_invalid", "Go version must be 1.27.0")
	}
	if lock.ModulePath != "github.com/araihu/margo" {
		return platformError("pdf.platform_lock_invalid", "root module path is invalid")
	}
	if err := validateModules(lock.Modules); err != nil {
		return err
	}
	if lock.NodeHarness.NodeVersion != "v26.5.0" || lock.NodeHarness.NPMVersion != "11.17.0" || lock.NodeHarness.DOMStrategy == "" || lock.NodeHarness.CSSTreeVersion != "3.1.0" || lock.NodeHarness.PlaywrightVersion != "1.55.1" || lock.NodeHarness.ChromiumRevision != "1193" || !validSHA256(lock.NodeHarness.ChromiumArchiveSHA256) {
		return platformError("pdf.platform_lock_invalid", "node harness identity is incomplete or unsupported")
	}
	if lock.MuambaTool.Path != "github.com/araihu/muamba/cmd/muamba" || lock.MuambaTool.Version != "v0.0.3" {
		return platformError("pdf.platform_lock_invalid", "Muamba tool identity is invalid")
	}
	if err := validateLockedRunners(lock.Runners, lockDirectory); err != nil {
		return err
	}
	recordedDigest := lock.RecordDigest
	lock.RecordDigest = ""
	payload, err := json.Marshal(lock)
	if err != nil {
		return platformError("pdf.platform_lock_invalid", err.Error())
	}
	digest := sha256.Sum256(payload)
	if !validSHA256(recordedDigest) || recordedDigest != hex.EncodeToString(digest[:]) {
		return platformError("pdf.platform_lock_invalid", "platform lock record digest is invalid")
	}
	return nil
}

func validateModules(modules []toolchainModule) error {
	if len(modules) != 1 {
		return platformError("pdf.platform_lock_invalid", "platform lock must contain the external chromedp module")
	}
	seen := make(map[string]toolchainModule, len(modules))
	for _, module := range modules {
		if module.Path == "" || module.Version == "" || module.Sum == "" || module.GoModSum == "" {
			return platformError("pdf.platform_lock_invalid", "module identity is incomplete")
		}
		if _, duplicate := seen[module.Path]; duplicate {
			return platformError("pdf.platform_lock_invalid", "module identity is duplicated")
		}
		seen[module.Path] = module
	}
	chromedp, chromedpOK := seen["github.com/chromedp/chromedp"]
	if !chromedpOK || chromedp.Version != "v0.14.2" || chromedp.Sum != "h1:r3b/WtwM50RsBZHMUm9fsNhhzRStTHrKdr2zmwbZSzM=" || chromedp.GoModSum != "h1:rHzAv60xDE7VNy/MYtTUrYreSc0ujt2O1/C3bzctYBo=" {
		return platformError("pdf.platform_lock_invalid", "module identity is unsupported")
	}
	return nil
}

func validateLockedRunners(runners []toolchainRunner, lockDirectory string) error {
	if len(runners) != len(knownRunnerIDs) {
		return platformError("pdf.platform_lock_invalid", "platform lock runner set is incomplete")
	}
	seen := make(map[RunnerID]struct{}, len(runners))
	for _, runner := range runners {
		if _, known := knownRunnerIDs[runner.ID]; !known {
			return platformError("pdf.platform_lock_invalid", fmt.Sprintf("runner %q is unsupported", runner.ID))
		}
		if _, duplicate := seen[runner.ID]; duplicate {
			return platformError("pdf.platform_lock_invalid", fmt.Sprintf("runner %q is duplicated", runner.ID))
		}
		seen[runner.ID] = struct{}{}
		if !safeRelativePath(runner.Probe) || !safeRelativePath(runner.SDKEvidencePath) || !safeRelativePath(runner.RuntimeEvidencePath) || !validSHA256(runner.SourceDigest) || runner.VersionPolicy != "tested-version-reported" {
			return platformError("pdf.platform_lock_invalid", fmt.Sprintf("runner %q identity is incomplete", runner.ID))
		}
		repositoryRoot := repositoryRootForLockDirectory(lockDirectory)
		data, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(runner.Probe)))
		if err != nil {
			return platformError("pdf.platform_lock_invalid", fmt.Sprintf("runner %q probe source is unavailable", runner.ID))
		}
		digest := sha256.Sum256(data)
		if runner.SourceDigest != hex.EncodeToString(digest[:]) {
			return platformError("pdf.platform_lock_invalid", fmt.Sprintf("runner %q probe source digest does not match", runner.ID))
		}
	}
	return nil
}

func repositoryRootForLockDirectory(lockDirectory string) string {
	absolute, err := filepath.Abs(lockDirectory)
	if err != nil {
		return filepath.Dir(lockDirectory)
	}
	return filepath.Dir(absolute)
}

func findRunner(runners []toolchainRunner, runnerID RunnerID) (toolchainRunner, bool) {
	for _, runner := range runners {
		if runner.ID == runnerID {
			return runner, true
		}
	}
	return toolchainRunner{}, false
}

func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && value == strings.ToLower(value)
}

func decodeStrictJSON(data []byte, destination any) error {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("JSON contains a second value")
		}
		return err
	}
	return nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := consumeJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("JSON contains a second value")
		}
		return err
	}
	return nil
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, container := token.(json.Delim)
	if !container {
		return nil
	}
	switch delimiter {
	case '{':
		keys := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("JSON object key is not a string")
			}
			if _, duplicate := keys[key]; duplicate {
				return fmt.Errorf("JSON contains duplicate key %q", key)
			}
			keys[key] = struct{}{}
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim('}') {
			return fmt.Errorf("JSON object is not closed")
		}
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim(']') {
			return fmt.Errorf("JSON array is not closed")
		}
	default:
		return fmt.Errorf("JSON has unexpected delimiter %q", delimiter)
	}
	return nil
}

func platformError(code, message string) error {
	return fmt.Errorf("%s: %s", code, message)
}
