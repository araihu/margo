package platform

import (
	"flag"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var (
	platformToolchainLockArgument = flag.String("lock", "platform-toolchain.lock", "platform toolchain lock path relative to the pdf module")
	runnerContractsArgument       = flag.String("contracts", "platform/runner-contracts.json", "runner contracts path relative to the pdf module")
)

func TestRunnerContractRepositoryRecordsEveryRunner(t *testing.T) {
	t.Parallel()

	contracts, err := LoadRunnerContracts(moduleArgumentPath(*runnerContractsArgument))
	if err != nil {
		t.Fatalf("LoadRunnerContracts(repository) error = %v", err)
	}
	wantCommands := map[RunnerID][]string{
		RunnerWindowsWebView2: {"go", "test", "./platform", "-run", "^TestProbeWindowsWebView2$", "-count=1"},
		RunnerDarwinWKWebView: {"go", "test", "./platform", "-run", "^TestProbeDarwinWKWebView$", "-count=1"},
		RunnerLinuxWebKitGTK:  {"go", "test", "./platform", "-run", "^TestProbeLinuxWebKitGTK$", "-count=1"},
		RunnerChromiumCDP:     {"go", "test", "./platform", "-run", "^TestProbeChromiumCDP$", "-count=1"},
	}
	for runnerID, wantCommand := range wantCommands {
		contract, ok := contracts.Runner(runnerID)
		if !ok {
			t.Fatalf("runner %q is absent", runnerID)
		}
		if !reflect.DeepEqual(contract.Command, wantCommand) {
			t.Fatalf("runner %q command = %v, want %v", runnerID, contract.Command, wantCommand)
		}
		if !containsPath(contract.OwnedSourcePaths, "platform/bootstrap.go") || !containsPath(contract.OwnedProbePaths, "platform/engine_probe_test.go") {
			t.Fatalf("runner %q ownership = source %v probe %v", runnerID, contract.OwnedSourcePaths, contract.OwnedProbePaths)
		}
	}
}

func moduleArgumentPath(argument string) string {
	if filepath.IsAbs(argument) {
		return argument
	}
	return filepath.Join("..", filepath.FromSlash(argument))
}

func TestRunnerContractAcceptsLockedGoTestProbe(t *testing.T) {
	t.Parallel()

	contractsPath := filepath.Join(t.TempDir(), "runner-contracts.json")
	writePlatformTestFile(t, contractsPath, validRunnerContractsJSON(`[
    "go", "test", "./platform", "-run", "^TestProbeWindowsWebView2$", "-count=1"
  ]`))

	contracts, err := LoadRunnerContracts(contractsPath)
	if err != nil {
		t.Fatalf("LoadRunnerContracts() error = %v", err)
	}
	contract, ok := contracts.Runner(RunnerWindowsWebView2)
	if !ok {
		t.Fatal("Runner() did not return the locked runner")
	}
	if contract.ExpectedExitCode != 0 {
		t.Fatalf("ExpectedExitCode = %d, want 0", contract.ExpectedExitCode)
	}
	contract.Command[0] = "curl"
	again, ok := contracts.Runner(RunnerWindowsWebView2)
	if !ok || again.Command[0] != "go" {
		t.Fatalf("Runner() exposed mutable command state: %+v", again.Command)
	}
}

func TestRunnerContractRejectsIncompleteRunnerSet(t *testing.T) {
	t.Parallel()

	contractsPath := filepath.Join(t.TempDir(), "runner-contracts.json")
	writePlatformTestFile(t, contractsPath, singleRunnerContractsJSON(`[
    "go", "test", "./platform", "-run", "^TestProbeWindowsWebView2$", "-count=1"
  ]`))

	_, err := LoadRunnerContracts(contractsPath)
	requirePlatformErrorCode(t, err, "pdf.platform_contract_invalid")
}

func TestRunnerContractRejectsProbeBoundToDifferentRunner(t *testing.T) {
	t.Parallel()

	contractsPath := filepath.Join(t.TempDir(), "runner-contracts.json")
	writePlatformTestFile(t, contractsPath, validRunnerContractsJSON(`[
    "go", "test", "./platform", "-run", "^TestProbeDarwinWKWebView$", "-count=1"
  ]`))

	_, err := LoadRunnerContracts(contractsPath)
	requirePlatformErrorCode(t, err, "pdf.platform_contract_invalid")
}

func TestRunnerContractRejectsDownloadCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
	}{
		{name: "curl", command: `["curl", "https://example.invalid/browser"]`},
		{name: "go-get", command: `["go", "get", "example.invalid/browser"]`},
		{name: "go-download", command: `["go", "mod", "download"]`},
		{name: "npm-install", command: `["npm", "install"]`},
		{name: "regex-expansion", command: `["go", "test", "./platform", "-run", "^TestProbe.*$", "-count=1"]`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			contractsPath := filepath.Join(t.TempDir(), "runner-contracts.json")
			writePlatformTestFile(t, contractsPath, validRunnerContractsJSON(test.command))
			_, err := LoadRunnerContracts(contractsPath)
			requirePlatformErrorCode(t, err, "pdf.platform_download_forbidden")
		})
	}
}

func TestRunnerContractRejectsUnownedPath(t *testing.T) {
	t.Parallel()

	contractsPath := filepath.Join(t.TempDir(), "runner-contracts.json")
	contents := validRunnerContractsJSON(`["go", "test", "./platform", "-run", "^TestProbeWindowsWebView2$", "-count=1"]`)
	contents = strings.Replace(contents, `"ownedSourcePaths": ["platform/bootstrap.go"]`, `"ownedSourcePaths": ["../go.mod"]`, 1)
	writePlatformTestFile(t, contractsPath, contents)

	_, err := LoadRunnerContracts(contractsPath)
	requirePlatformErrorCode(t, err, "pdf.platform_contract_invalid")
}

func TestRunnerContractRejectsWindowsVolumePath(t *testing.T) {
	t.Parallel()

	contractsPath := filepath.Join(t.TempDir(), "runner-contracts.json")
	contents := validRunnerContractsJSON(`["go", "test", "./platform", "-run", "^TestProbeWindowsWebView2$", "-count=1"]`)
	contents = strings.Replace(contents, `"ownedSourcePaths": ["platform/bootstrap.go"]`, `"ownedSourcePaths": ["C:/go.mod"]`, 1)
	writePlatformTestFile(t, contractsPath, contents)

	_, err := LoadRunnerContracts(contractsPath)
	requirePlatformErrorCode(t, err, "pdf.platform_contract_invalid")
}

func TestRunnerContractRejectsDuplicateJSONKey(t *testing.T) {
	t.Parallel()

	contractsPath := filepath.Join(t.TempDir(), "runner-contracts.json")
	contents := validRunnerContractsJSON(`["go", "test", "./platform", "-run", "^TestProbeWindowsWebView2$", "-count=1"]`)
	contents = strings.Replace(contents, `"schemaVersion": "margo/pdf-runner-contracts/v1",`, `"schemaVersion": "margo/pdf-runner-contracts/v1",
  "schemaVersion": "margo/pdf-runner-contracts/v1",`, 1)
	writePlatformTestFile(t, contractsPath, contents)

	_, err := LoadRunnerContracts(contractsPath)
	requirePlatformErrorCode(t, err, "pdf.platform_contract_invalid")
}

func validRunnerContractsJSON(command string) string {
	return `{
  "schemaVersion": "margo/pdf-runner-contracts/v1",
  "runners": {
    "windows-webview2/v1": {
      "command": ` + command + `,
      "expectedExitCode": 0,
      "ownedSourcePaths": ["platform/bootstrap.go"],
      "ownedProbePaths": ["platform/engine_probe_test.go"]
    },
    "darwin-wkwebview/v1": {
      "command": ["go", "test", "./platform", "-run", "^TestProbeDarwinWKWebView$", "-count=1"],
      "expectedExitCode": 0,
      "ownedSourcePaths": ["platform/bootstrap.go"],
      "ownedProbePaths": ["platform/engine_probe_test.go"]
    },
    "linux-webkitgtk/v1": {
      "command": ["go", "test", "./platform", "-run", "^TestProbeLinuxWebKitGTK$", "-count=1"],
      "expectedExitCode": 0,
      "ownedSourcePaths": ["platform/bootstrap.go"],
      "ownedProbePaths": ["platform/engine_probe_test.go"]
    },
    "chromium-cdp/v1": {
      "command": ["go", "test", "./platform", "-run", "^TestProbeChromiumCDP$", "-count=1"],
      "expectedExitCode": 0,
      "ownedSourcePaths": ["platform/bootstrap.go"],
      "ownedProbePaths": ["platform/engine_probe_test.go"]
    }
  }
}`
}

func singleRunnerContractsJSON(command string) string {
	return `{
  "schemaVersion": "margo/pdf-runner-contracts/v1",
  "runners": {
    "windows-webview2/v1": {
      "command": ` + command + `,
      "expectedExitCode": 0,
      "ownedSourcePaths": ["platform/bootstrap.go"],
      "ownedProbePaths": ["platform/engine_probe_test.go"]
    }
  }
}`
}
