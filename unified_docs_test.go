package margo

import (
	"encoding/json"
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
		"margo check", "margo html", "margo site", "margo serve", "margo pdf", "margo deck", "margo doctor", "margo version", "margo help [command]",
		"margo serve [INPUT_DIR|CONFIG]", "development server is not for production",
		"127.0.0.1", "8080, 8000, 3000, 1313, and 4000", "operating system selects any available port",
		"last successful site", "live reload", "--open",
		"margo completion SHELL [--no-descriptions]", "SHELL` is `bash`, `zsh`, `fish`, or `powershell`",
		"`--no-descriptions` flag applies to all four generators.",
		"margo.Check",
		"margo.WithExtension(charts.Extension())",
		"--engine auto|chromium|native", "MARGO_CHROMIUM_PATH", "never downloads",
		"CGO_ENABLED=0", "--output -", "--force", "stderr", "stdout",
		"--relative-links strip|error|keep|resolve", "readable document margins of", "set all four margin flags to `0` for full bleed",
		"Its defaults are HTML to stdout, `--engine auto`, A4, portrait, and zero", "margins. PDF decks require",
		"historical submodule tags", "docs/decisions/0001-unified-module-and-cli.md",
		"docs/testing/pdf-engine-matrix.md",
		"semantic page layouts and documentation families",
		"layout:\n  kind: docs\n  default:",
		"`layout.default.families`",
		"`_layout.yaml`",
		"site defaults, directory patches from root to nearest",
		"maps merge recursively",
		"Changing `kind` creates a typed boundary",
		"Landing and article pages have no",
		"The `landing` layout is for",
		"The `docs`",
		"layout provides family-local navigation",
		"Tour at `/`,",
		"the CLI overview at `/cli/`, and one CLI command page",
		"under `/cli/COMMAND/`",
		"Static artifacts remain",
		"public links, canonicals, search, family navigation, sitemap, `llms.txt`",
		"base-path and locale prefixes",
		"Retired Tour feature routes",
		"return HTTP 404",
		"produce no artifacts",
		"Sites without `layout` retain existing top-level `frame` or `shell` behavior.",
		"Existing `componentdocshell` consumers remain supported",
	} {
		if !strings.Contains(readme, required) {
			t.Fatalf("README missing %q", required)
		}
	}
	for _, required := range []string{
		"`github.com/araihu/margo`", "`github.com/araihu/margo/assets`", "`github.com/araihu/margo/charts`",
		"`github.com/araihu/margo/deck`", "`github.com/araihu/margo/pdf`",
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
	for _, stale := range []string{"github.com/araihu/margo/embed", "trusted-embed"} {
		if strings.Contains(readme, stale) {
			t.Fatalf("README retains removed embed contract %q", stale)
		}
	}
}

func TestDocumentSchemaOmitsRemovedSiteLayoutPreference(t *testing.T) {
	data, err := os.ReadFile("schema/v1/document.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("document schema has no properties object")
	}
	margoProperty, ok := properties["margo"].(map[string]any)
	if !ok {
		t.Fatal("document schema has no margo property")
	}
	margoProperties, ok := margoProperty["properties"].(map[string]any)
	if !ok {
		t.Fatal("margo schema has no properties object")
	}
	if _, exists := margoProperties["site"]; exists {
		t.Fatal("document schema retains removed margo.site layout preference")
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
