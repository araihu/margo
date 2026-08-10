package deck

import (
	"bytes"
	"errors"
	"testing"

	"github.com/araihu/margo"
)

func TestParseSplitsOnlyExactSeparatorsOutsideFences(t *testing.T) {
	source := []byte("---\ntitle: Demo\ndescription: Two slides\n---\n# One\n\n```text\n---\n```\n---\n# Two\n")
	doc, err := Parse("demo.md", source)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := doc.Metadata().Title, "Demo"; got != want {
		t.Fatalf("title = %q want %q", got, want)
	}
	if got, want := doc.Metadata().Description, "Two slides"; got != want {
		t.Fatalf("description = %q want %q", got, want)
	}
	slides := doc.Slides()
	if len(slides) != 2 {
		t.Fatalf("slides = %d", len(slides))
	}
	if !bytes.Contains(slides[0].Markdown(), []byte("```text\n---\n```")) {
		t.Fatal("fenced separator was removed")
	}
	if slides[0].Ordinal() != 1 || slides[1].Ordinal() != 2 {
		t.Fatal("unstable ordinals")
	}
	if slides[0].ID() != "slide-0001" || slides[1].ID() != "slide-0002" {
		t.Fatalf("slide IDs = %q, %q", slides[0].ID(), slides[1].ID())
	}
}

func TestParseSupportsTildeFencesAndCRLF(t *testing.T) {
	doc, err := Parse("windows.md", []byte("# One\r\n~~~~yaml\r\n---\r\n~~~~\r\n---\r\n# Two\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(doc.Slides()); got != 2 {
		t.Fatalf("slides = %d", got)
	}
}

func TestParseRejectsEmptySlides(t *testing.T) {
	for _, source := range []string{"\n---\n# Two\n", "# One\n---\n  \n", "# One\n---\n---\n# Three\n"} {
		_, err := Parse("empty.md", []byte(source))
		if got := deckDiagnosticCode(err); got != "deck.slide_empty" {
			t.Fatalf("source %q code = %q", source, got)
		}
	}
}

func TestParseRejectsInvalidFrontmatterAndUnclosedFence(t *testing.T) {
	_, err := Parse("metadata.md", []byte("---\ntitle: [\n---\n# One\n"))
	if got := deckDiagnosticCode(err); got != "deck.frontmatter_invalid" {
		t.Fatalf("frontmatter code = %q", got)
	}
	_, err = Parse("fence.md", []byte("# One\n```text\ncontent\n"))
	if got := deckDiagnosticCode(err); got != "deck.fence_unclosed" {
		t.Fatalf("fence code = %q", got)
	}
}

func TestDocumentAndSlidesReturnDefensiveCopies(t *testing.T) {
	doc, err := Parse("copy.md", []byte("# One\n---\n# Two\n"))
	if err != nil {
		t.Fatal(err)
	}
	first := doc.Slides()
	first[0].markdown[0] = 'X'
	first = append(first, Slide{})
	second := doc.Slides()
	if got := string(second[0].Markdown()); got != "# One\n" {
		t.Fatalf("markdown = %q", got)
	}
	if len(second) != 2 {
		t.Fatalf("slides = %d", len(second))
	}
}

func deckDiagnosticCode(err error) string {
	if err == nil {
		return ""
	}
	var diagnostic *margo.DiagnosticError
	if errors.As(err, &diagnostic) && len(diagnostic.Diagnostics) > 0 {
		return diagnostic.Diagnostics[0].Code
	}
	return err.Error()
}
