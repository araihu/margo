package deck

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/araihu/margo"
)

func TestCompositionR1FixtureCoversEveryCatalogEntry(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "compositions-r1.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Parse("compositions-r1.md", source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Render(context.Background(), margo.New(), RenderInput{Name: "compositions-r1.md", Markdown: source}); err != nil {
		t.Fatal(err)
	}
	counts := make(map[CompositionName]int)
	for _, slide := range doc.Slides() {
		if spec := slide.Composition(); spec.Name != "" {
			counts[spec.Name]++
			if isStructuralClass(spec.LayoutClass) {
				layout := slide.Layout()
				if layout == nil || layout.Class != spec.LayoutClass {
					t.Fatalf("composition %q layout = %#v want family %q", spec.Name, layout, spec.LayoutClass)
				}
			}
		}
	}
	for name := range compositionCatalog {
		if counts[name] == 0 {
			t.Fatalf("fixture is missing composition %q", name)
		}
	}
}
