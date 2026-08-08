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
	} {
		if !strings.Contains(string(data), required) {
			t.Errorf("generated HTML missing %q", required)
		}
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
