package deck

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompatibilityCorpusFixtures(t *testing.T) {
	tests := []struct {
		name       string
		wantSlides int
	}{
		{name: "heading-divider-scalar.md", wantSlides: 3},
		{name: "heading-divider-exact.md", wantSlides: 3},
		{name: "slide-rulers.md", wantSlides: 2},
		{name: "background-last-wins.md", wantSlides: 2},
		{name: "size-extension.md", wantSlides: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join("testdata", "corpus", test.name))
			if err != nil {
				t.Fatal(err)
			}
			document, err := Parse(test.name, source)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(document.Slides()); got != test.wantSlides {
				t.Fatalf("slides = %d want %d", got, test.wantSlides)
			}
		})
	}
	invalid, err := os.ReadFile(filepath.Join("testdata", "corpus", "directive-comment-boundary.md"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("directive-comment-boundary.md", invalid); deckDiagnosticCode(err) != "deck.directive_unsupported" {
		t.Fatalf("boundary fixture error = %v", err)
	}
}
