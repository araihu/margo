package charts

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

func TestOptimisticBenchmarkIncludesChartProjections(t *testing.T) {
	base, err := os.ReadFile(filepath.Join("..", "testdata", "markdown", "margo-full-feature-set.md"))
	if err != nil {
		t.Fatal(err)
	}
	appendix, err := os.ReadFile(filepath.Join("testdata", "markdown", "optimistic-charts.md"))
	if err != nil {
		t.Fatal(err)
	}
	source := append(append(append([]byte(nil), base...), '\n', '\n'), appendix...)

	compiler := margo.New(
		margo.WithHostPolicy(margo.Policy{RawHTML: margo.RawHTMLSanitized, OutputBytes: margo.MaxOutputBytes}),
		margo.WithExtension(Extension(WithControlWrapper(false))),
	)
	document, err := compiler.Compile(context.Background(), margo.Source{
		Name:    "margo-full-feature-set-with-charts.md",
		Content: source,
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	result, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	var output bytes.Buffer
	if err := result.Content().Render(context.Background(), &output); err != nil {
		t.Fatalf("render content: %v", err)
	}
	markup := output.String()
	if got := strings.Count(markup, `data-margo-chart-data="v1"`); got != 4 {
		t.Fatalf("accessible chart tables = %d, want 4", got)
	}
	for _, marker := range []string{
		`goshtoso-charts-bar`,
		`goshtoso-charts-line`,
		`goshtoso-charts-pie`,
		`goshtoso-charts-scatter`,
		`fill:#dc2626`,
		`fill:#2563eb`,
		`benchmark-revenue-series`,
		`benchmark-cost-series`,
		`benchmark-desktop-slice`,
		`benchmark-api-series`,
	} {
		if !strings.Contains(markup, marker) {
			t.Errorf("chart benchmark output missing %q", marker)
		}
	}
	if strings.Contains(markup, `data-chart-runtime`) {
		t.Fatal("chart benchmark output contains a browser runtime marker")
	}
	if strings.Contains(markup, `data-goshtoso-chart-wrapper-mode="enabled"`) {
		t.Fatal("chart benchmark output contains an unbundled control runtime")
	}
}
