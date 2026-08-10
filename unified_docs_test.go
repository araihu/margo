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
		"Margo turns Markdown into standalone HTML, PDF documents, and presentation decks.",
		"go install github.com/araihu/margo/cmd/margo@vX.Y.Z",
		"margo html", "margo pdf", "margo deck", "margo doctor", "margo version",
		"margo.WithExtension(charts.Extension())",
		"--engine auto|chromium|native", "MARGO_CHROMIUM_PATH", "never downloads",
		"CGO_ENABLED=0", "musl", "--output -", "--force", "stderr", "stdout",
		"historical submodule tags", "docs/decisions/0001-unified-module-and-cli.md",
		"docs/testing/pdf-engine-matrix.md",
	} {
		if !strings.Contains(readme, required) {
			t.Fatalf("README missing %q", required)
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
