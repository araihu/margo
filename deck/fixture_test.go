package deck

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/araihu/margo"
	"github.com/araihu/margo/charts"
)

func TestCompatibilityFixtureConsumesRepresentativeProfile(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "compatibility.md"))
	if err != nil {
		t.Fatal(err)
	}
	compiler := margo.New(margo.WithExtension(charts.Extension(charts.WithExternalizedControlRuntime(true))))
	result, err := Render(context.Background(), compiler, RenderInput{Name: "compatibility.md", Markdown: source})
	if err != nil {
		t.Fatal(err)
	}
	if result.SlideCount() != 3 {
		t.Fatalf("slides = %d", result.SlideCount())
	}
	markup := string(result.HTML())
	for _, fragment := range []string{
		`lang="pt-BR"`,
		`margo-layout--columns`,
		`margo-layout--timeline`,
		`margo-mermaid`,
		`data-goshtoso-chart-wrapper`,
	} {
		if !strings.Contains(markup, fragment) {
			t.Fatalf("fixture output missing %q", fragment)
		}
	}

	geometry := DeckGeometry{Preset: "4:3", Width: 960, Height: 720}
	result, err = Render(context.Background(), compiler, RenderInput{Name: "compatibility.md", Markdown: source, Geometry: geometry})
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Geometry(); got != geometry {
		t.Fatalf("geometry = %#v", got)
	}
	descriptor, err := result.RuntimeDescriptor("ri-00000042")
	if err != nil {
		t.Fatal(err)
	}
	if len(descriptor.Tasks) < 2 || descriptor.Tasks[len(descriptor.Tasks)-2].Kind != "deck-layout-screen" || descriptor.Tasks[len(descriptor.Tasks)-1].Kind != "deck-layout-print-dom" {
		t.Fatalf("layout tasks = %#v", descriptor.Tasks)
	}
}
