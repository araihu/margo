package margo

import (
	"os"
	"strings"
	"testing"
)

func TestREADMEExplainsUnifiedCLIAndReleaseContract(t *testing.T) {
	data, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	readme := string(data)
	for _, required := range []string{
		"Margo turns Markdown into standalone HTML, linked static sites, PDF documents,",
		"go install github.com/araihu/margo/cmd/margo@vX.Y.Z",
		"Starting with `v0.0.3`", "GitHub Release", "checksums.txt", "margo_VERSION_OS_ARCH",
		"margo check", "margo html", "margo site", "margo pdf", "margo deck", "margo doctor", "margo version", "margo help [command]",
		"margo completion SHELL [--no-descriptions]", "SHELL` is `bash`, `zsh`, `fish`, or `powershell`",
		"`--no-descriptions` flag applies to all four generators.",
		"margo.Check",
		"margo.WithExtension(charts.Extension())",
		"--engine auto|chromium|native", "MARGO_CHROMIUM_PATH", "never downloads",
		"CGO_ENABLED=0", "--output -", "--force", "stderr", "stdout",
		"--relative-links strip|error|keep|resolve", "Defaults are `--engine auto`, A4, portrait, and zero margins.",
		"Its defaults are HTML to stdout, `--engine auto`, A4, portrait, and zero", "margins. PDF decks require",
		"historical submodule tags", "docs/decisions/0001-unified-module-and-cli.md",
		"docs/testing/pdf-engine-matrix.md",
	} {
		if !strings.Contains(readme, required) {
			t.Fatalf("README missing %q", required)
		}
	}
	for _, required := range []string{
		"`github.com/araihu/margo`", "`github.com/araihu/margo/assets`", "`github.com/araihu/margo/charts`",
		"`github.com/araihu/margo/deck`", "`github.com/araihu/margo/embed`", "`github.com/araihu/margo/pdf`",
		"`github.com/araihu/margo/pdf/chromium`", "`github.com/araihu/margo/pdf/engines`",
		"`github.com/araihu/margo/pdf/native`", "`github.com/araihu/margo/pdf/platform`",
		"`github.com/araihu/margo/site`", "`cmd/margo` is the CLI program, not a library API.",
		"`internal/...` packages are", "unsupported implementation details.", "test and developer tools", "not release",
	} {
		if !strings.Contains(readme, required) {
			t.Fatalf("README missing package or classification %q", required)
		}
	}
	for _, stale := range []string{"go get github.com/araihu/margo/pdf@", "Released separately", "Optional repository module", "Each Go module is tested independently"} {
		if strings.Contains(readme, stale) {
			t.Fatalf("README retains stale release claim %q", stale)
		}
	}
}

func TestPDFEngineMatrixSeparatesEvidenceFromVersionPolicy(t *testing.T) {
	data, err := os.ReadFile("docs/testing/pdf-engine-matrix.md")
	if err != nil {
		t.Fatal(err)
	}
	matrix := string(data)
	for _, required := range []string{
		"Google Chrome 151.0.7922.77", "Chromium 142.0.7400.0", "macOS 26.5.2", "Go 1.26.5",
		"tested evidence", "not a minimum", "compiled out", "WKWebView", "WebView2", "WebKitGTK", "musl",
	} {
		if !strings.Contains(matrix, required) {
			t.Fatalf("engine matrix missing %q", required)
		}
	}
}

func TestUnifiedDecisionSeparatesCurrentAndTargetNativeCapabilities(t *testing.T) {
	data, err := os.ReadFile("docs/decisions/0001-unified-module-and-cli.md")
	if err != nil {
		t.Fatal(err)
	}
	decision := string(data)
	for _, required := range []string{
		"Current release capability", "Target native capabilities", "compiled out",
		"one root Go module", "one CLI, `margo`", "historical nested module requirements",
	} {
		if !strings.Contains(decision, required) {
			t.Fatalf("unified decision missing %q", required)
		}
	}
	if strings.Contains(decision, "Official release capabilities are") {
		t.Fatal("unified decision still claims unimplemented native engines as official release capabilities")
	}
}
