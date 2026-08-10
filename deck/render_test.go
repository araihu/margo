package deck

import (
	"context"
	"strings"
	"testing"

	"github.com/araihu/margo"
)

func TestRenderProducesAccessibleSections(t *testing.T) {
	compiler := margo.New()
	result, err := Render(context.Background(), compiler, RenderInput{
		Name:     "deck.md",
		Markdown: []byte("# One\n---\n# Two\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	html := string(result.HTML())
	for _, fragment := range []string{
		`<article class="margo-deck"`,
		`id="slide-0001"`,
		`role="region"`,
		`aria-label="Slide 1 of 2"`,
		`aria-label="Slide 2 of 2"`,
		`>One</h1>`,
		`>Two</h1>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("HTML missing %q", fragment)
		}
	}
	if result.SlideCount() != 2 {
		t.Fatalf("count = %d", result.SlideCount())
	}
}

func TestRenderComposesRuntimeDescriptorsAcrossSlides(t *testing.T) {
	result := mustRenderDeck(t, "```mermaid\ngraph TD; A-->B\n```\n---\n```mermaid\ngraph TD; C-->D\n```\n")
	descriptor, err := result.RuntimeDescriptor("ri-00000042")
	if err != nil {
		t.Fatal(err)
	}
	if len(descriptor.Tasks) != 2 {
		t.Fatalf("tasks = %#v", descriptor.Tasks)
	}
	if descriptor.Tasks[0].ID == descriptor.Tasks[1].ID {
		t.Fatal("runtime task IDs collide")
	}
	if descriptor.DocumentFingerprint != result.DocumentFingerprint() {
		t.Fatal("runtime and deck fingerprints differ")
	}
	if err := margo.ValidateRuntimeDescriptor(descriptor); err != nil {
		t.Fatal(err)
	}
}

func TestRenderFailsAtomicallyForInvalidLaterSlide(t *testing.T) {
	result, err := Render(context.Background(), margo.New(), RenderInput{
		Name:     "invalid.md",
		Markdown: []byte("# Valid\n---\n```mermaid\n%%{init: {}}%%\ngraph TD; A-->B\n```\n"),
	})
	if err == nil || result != nil {
		t.Fatalf("result = %#v error = %v", result, err)
	}
}

func TestRenderRejectsNilCompiler(t *testing.T) {
	result, err := Render(context.Background(), nil, RenderInput{Name: "deck.md", Markdown: []byte("# One\n")})
	if got := deckDiagnosticCode(err); got != "deck.compiler_required" || result != nil {
		t.Fatalf("result = %#v code = %q", result, got)
	}
}

func TestResultHTMLIsDefensive(t *testing.T) {
	result := mustRenderDeck(t, "# One\n")
	first := result.HTML()
	first[0] = 'X'
	if second := result.HTML(); len(second) == 0 || second[0] == 'X' {
		t.Fatal("HTML aliases result storage")
	}
}

func mustRenderDeck(t *testing.T, source string) *Result {
	t.Helper()
	result, err := Render(context.Background(), margo.New(), RenderInput{Name: "deck.md", Markdown: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
