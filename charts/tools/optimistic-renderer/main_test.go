package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

func TestGenerateHTMLAppendsAndRendersChartsAtomically(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "feature.md")
	chartsSourcePath := filepath.Join(root, "charts.md")
	outputPath := filepath.Join(root, "out", "feature.html")
	if err := os.Mkdir(filepath.Dir(outputPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte("# Feature\n\nA bounded human review artifact.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chartsSourcePath, []byte("```goshtosochart\nschemaVersion: 1\ntype: bar\ntitle: Revenue\ncategories: [A]\nseries:\n  - name: Revenue\n    color: \"#123456\"\n    values: [1]\n```\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := generateHTML(context.Background(), generatorConfig{
		SourcePath:       sourcePath,
		ChartsSourcePath: chartsSourcePath,
		OutputPath:       outputPath,
		Title:            "Feature review",
		Description:      "A deterministic feature review.",
		ColorMode:        margo.ColorModeDark,
	})
	if err != nil {
		t.Fatalf("generateHTML() error = %v", err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`<title>Feature review</title>`,
		`data-theme="modern"`,
		`data-color-mode="dark"`,
		`<h1 id="feature">Feature</h1>`,
		`goshtoso-charts-bar`,
		`fill:#123456`,
		`data-margo-chart-data="v1"`,
		`data-goshtoso-chart-wrapper-mode="enabled"`,
		`data-margo-goshtoso-runtime="first-party"`,
		`data-margo-goshtoso-runtime="alpine-focus"`,
		`data-margo-goshtoso-runtime="alpine"`,
		`data-margo-chart-controls-inline="v5"`,
	} {
		if !strings.Contains(string(data), required) {
			t.Errorf("generated HTML missing %q", required)
		}
	}
	markup := string(data)
	if strings.Contains(markup, `/charts/assets/js/controls/5/controls.js`) {
		t.Fatal("generated HTML retains an external chart-control runtime")
	}
	for _, external := range []string{
		`/assets/js/runtime/alpinejs/3.14.9/alpine.min.js`,
		`/assets/js/runtime/alpinejs-focus/3.14.9/alpine-focus.min.js`,
		`/assets/js/goshtoso.min.js`,
	} {
		if strings.Contains(markup, external) {
			t.Fatalf("generated HTML retains external Goshtoso runtime %q", external)
		}
	}
	for _, role := range []string{"first-party", "alpine-focus", "alpine"} {
		if strings.Count(markup, `data-margo-goshtoso-runtime="`+role+`"`) != 1 {
			t.Fatalf("inline Goshtoso runtime %q count = %d, want 1", role, strings.Count(markup, `data-margo-goshtoso-runtime="`+role+`"`))
		}
	}
	if strings.Count(markup, `data-margo-chart-controls-inline="v5"`) != 1 {
		t.Fatalf("inline chart-control runtime count = %d, want 1", strings.Count(markup, `data-margo-chart-controls-inline="v5"`))
	}
	if strings.Index(markup, `data-margo-chart-controls-inline="v5"`) > strings.Index(markup, `</body>`) {
		t.Fatal("inline chart-control runtime was emitted after </body>")
	}
	entries, err := os.ReadDir(filepath.Dir(outputPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".margo-render-charts-") {
			t.Errorf("atomic temporary file leaked: %s", entry.Name())
		}
	}
}

func TestGenerateHTMLRejectsInvalidColorModeWithoutOutput(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "feature.md")
	outputPath := filepath.Join(root, "feature.html")
	if err := os.WriteFile(sourcePath, []byte("# Feature\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := generateHTML(context.Background(), generatorConfig{
		SourcePath: sourcePath,
		OutputPath: outputPath,
		Title:      "Feature review",
		ColorMode:  margo.ColorMode("sepia"),
	})
	if err == nil {
		t.Fatal("generateHTML() error = nil, want invalid color mode error")
	}
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf("output exists after rejected render: stat error = %v", statErr)
	}
}
