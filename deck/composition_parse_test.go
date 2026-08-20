package deck

import (
	"strings"
	"testing"
)

func TestParseCompositionFrontmatterDefault(t *testing.T) {
	source := []byte("---\ncomposition: hero\n---\n# Opening\n")
	doc, err := Parse("composition-frontmatter.md", source)
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.Slides()[0].Directives().Composition; got != "hero" {
		t.Fatalf("composition = %q want hero", got)
	}
}

func TestParseCompositionBodyInheritanceAndSpotClear(t *testing.T) {
	source := []byte("# One\n---\n<!-- composition: hero -->\n# Two\n---\n<!-- _composition: none -->\n# Three\n---\n# Four\n")
	doc, err := Parse("composition-inheritance.md", source)
	if err != nil {
		t.Fatal(err)
	}
	slides := doc.Slides()
	if len(slides) != 4 {
		t.Fatalf("slides = %d want 4", len(slides))
	}
	if got := slides[0].Directives().Composition; got != "" {
		t.Fatalf("slide 1 composition = %q want empty", got)
	}
	if got := slides[1].Directives().Composition; got != "hero" {
		t.Fatalf("slide 2 composition = %q want hero", got)
	}
	if got := slides[2].Directives().Composition; got != "" {
		t.Fatalf("slide 3 composition = %q want empty", got)
	}
	if got := slides[3].Directives().Composition; got != "hero" {
		t.Fatalf("slide 4 composition = %q want inherited hero", got)
	}
}

func TestParseCompositionRejectsMalformedValues(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{name: "unknown", source: "---\ncomposition: unknown\n---\n# One\n"},
		{name: "empty", source: "---\ncomposition:\n---\n# One\n"},
		{name: "sequence", source: "---\ncomposition: [hero]\n---\n# One\n"},
		{name: "mapping", source: "---\ncomposition:\n  name: hero\n---\n# One\n"},
		{name: "mixed case", source: "<!-- composition: Hero -->\n# One\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.name+".md", []byte(tc.source))
			if got, want := deckDiagnosticCode(err), "deck.composition_invalid"; got != want {
				t.Fatalf("diagnostic = %q want %q (err=%v)", got, want, err)
			}
		})
	}
}

func TestParseCompositionOutsideFenceOnly(t *testing.T) {
	source := []byte("<!-- composition: hero -->\n```markdown\n<!-- composition: image-grid -->\n```\n# Opening\n")
	doc, err := Parse("composition-fence.md", source)
	if err != nil {
		t.Fatal(err)
	}
	slide := doc.Slides()[0]
	if got := slide.Directives().Composition; got != "hero" {
		t.Fatalf("composition = %q want hero", got)
	}
	if !strings.Contains(string(slide.Markdown()), "composition: image-grid") {
		t.Fatalf("fenced composition was removed from Markdown: %q", slide.Markdown())
	}
}
