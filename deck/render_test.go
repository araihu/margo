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

func TestRenderEmitsLogicalStageGeometryAndLocalizedLanguage(t *testing.T) {
	result, err := Render(context.Background(), margo.New(), RenderInput{
		Name:     "deck.md",
		Markdown: []byte("---\nlang: pt-BR\nsize: 4:3\n---\n# Um\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	html := string(result.HTML())
	for _, fragment := range []string{
		`<html lang="pt-BR"`,
		`data-margo-font-bundle-digest="`,
		`<section id="slide-0001" class="margo-deck__slide" role="region" aria-label="Slide 1 de 1"`,
		`data-margo-slide="0" aria-current="page" lang="pt-BR"`,
		`class="margo-deck-stage"`,
		`data-margo-width="960"`,
		`data-margo-height="720"`,
		`class="margo-deck-controls"`,
		`aria-label="Controles de slides"`,
		`>Anterior</button>`,
		`>Próximo</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("HTML missing %q", fragment)
		}
	}
	if got := result.Geometry(); got.Width != 960 || got.Height != 720 {
		t.Fatalf("geometry = %#v", got)
	}
}

func TestRenderProjectsStructuralLayoutDOMInSourceOrder(t *testing.T) {
	result := mustRenderDeck(t, "<!-- _class: columns -->\n<!-- layout: columns -->\n<!-- slot: left -->\n# Left\n<!-- slot: right -->\n# Right\n<!-- /layout -->\n")
	html := string(result.HTML())
	for _, fragment := range []string{
		`margo-deck__slide--columns`,
		`margo-layout margo-layout--columns`,
		`margo-layout__slot margo-layout__slot--left`,
		`margo-layout__slot margo-layout__slot--right`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("HTML missing %q", fragment)
		}
	}
	if strings.Index(html, ">Left</h1>") > strings.Index(html, ">Right</h1>") {
		t.Fatal("layout DOM reordered source slots")
	}
}

func TestRenderEmitsBackgroundAccessibilitySemantics(t *testing.T) {
	result := mustRenderDeck(t, "<!-- color: accent-strong -->\n<!-- backgroundImage: images/hero.png -->\n<!-- backgroundPosition: top-right -->\n<!-- backgroundRepeat: no-repeat -->\n<!-- backgroundSize: contain -->\n<!-- backgroundAlt: Hero image -->\n# One\n")
	html := string(result.HTML())
	for _, fragment := range []string{`data-margo-color="accent-strong"`, `role="img"`, `aria-label="Hero image"`, `data-margo-background="images/hero.png"`, `data-margo-background-position="top-right"`, `data-margo-background-repeat="no-repeat"`, `data-margo-background-size="contain"`} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("HTML missing %q", fragment)
		}
	}
}

func TestRenderComposesRuntimeDescriptorsAcrossSlides(t *testing.T) {
	result := mustRenderDeck(t, "```mermaid\ngraph TD; A-->B\n```\n---\n```mermaid\ngraph TD; C-->D\n```\n")
	descriptor, err := result.RuntimeDescriptor("ri-00000042")
	if err != nil {
		t.Fatal(err)
	}
	if len(descriptor.Tasks) != 4 {
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

func TestRenderRejectsConflictingGeometryLayers(t *testing.T) {
	inputGeometry := DeckGeometry{Preset: "4:3", Width: 960, Height: 720, Unit: DeckUnitPX}
	optionGeometry := DeckGeometry{Preset: "16:9", Width: 1280, Height: 720, Unit: DeckUnitPX}
	_, err := Render(context.Background(), margo.New(), RenderInput{Name: "conflict.md", Markdown: []byte("# One\n"), Geometry: inputGeometry}, WithGeometry(optionGeometry))
	if err == nil || !strings.Contains(err.Error(), "deck.geometry_conflict") {
		t.Fatalf("error = %v", err)
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

func TestRenderEmbedsNavigationPrintAndDependencies(t *testing.T) {
	result := mustRenderDeck(t, "# One\n---\n# Two\n")
	html := string(result.HTML())
	for _, value := range []string{
		"<!doctype html>",
		`data-margo-requirement="margo.document.styles"`,
		"data-margo-deck-previous",
		"data-margo-deck-next",
		"ArrowLeft",
		"ArrowRight",
		"Home",
		"End",
		`closest("button, a, [role='button']`,
		"@media print",
		"break-after: page",
		"window.print",
		"margoPrepareDeckPrint",
		"beforeprint",
		`script-src 'self' 'unsafe-inline'`,
	} {
		if !strings.Contains(html, value) {
			t.Fatalf("HTML missing %q", value)
		}
	}
}

func TestResultRequirementsAreMergedAndDefensive(t *testing.T) {
	result := mustRenderDeck(t, "| A | B |\n|---|---|\n| 1 | 2 |\n---\n# Two\n")
	first := result.Requirements().List()
	if len(first) < 3 {
		t.Fatalf("requirements = %#v", first)
	}
	first[0].ID = "changed"
	if result.Requirements().List()[0].ID == "changed" {
		t.Fatal("requirements alias result storage")
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
