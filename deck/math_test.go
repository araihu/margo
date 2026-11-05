package deck

import (
	"context"
	"strings"
	"testing"

	"github.com/araihu/margo"
)

func TestMathDeckProtectsSourceAndRuntime(t *testing.T) {
	source := []byte("# First\n\n$$\n---\n<!-- theme: minimal -->\n# Not a slide\n$$\n\n---\n\n# Second\n\n```math\n\\frac{1}{2}\n```\n")
	doc, err := ParseWithOptions("slides.md", source, WithMath(true))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Slides()) != 2 || doc.Directives().Theme == margo.ThemeMinimal {
		t.Fatalf("parsed: %d %+v", len(doc.Slides()), doc.Directives())
	}
	result, err := Render(context.Background(), margo.New(margo.WithMath(true)), RenderInput{Name: "slides.md", Markdown: source})
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := result.RuntimeDescriptor("ri-00000001")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, task := range descriptor.Tasks {
		if task.Kind == "math" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("math tasks: %+v", descriptor.Tasks)
	}
	if strings.Count(string(result.html), `<div class="margo-math margo-math--display"`) != 2 {
		t.Fatal("missing math markers")
	}
}

func TestMathDeckSourcePositionsAndLiteralChrome(t *testing.T) {
	source := []byte("---\nheader: '$header$'\n---\n\n# First\n\n$x$\n\n---\n\n# Second\n\n$y$\n")
	r, err := Render(context.Background(), margo.New(margo.WithMath(true)), RenderInput{Name: "original.md", Markdown: source})
	if err != nil {
		t.Fatal(err)
	}
	markup := string(r.HTML())
	for _, want := range []string{`data-margo-source="original.md" data-margo-line="7"`, `data-margo-source="original.md" data-margo-line="13"`, `>$header$</header>`} {
		if !strings.Contains(markup, want) {
			start := strings.Index(markup, "<header")
			t.Fatalf("missing %s; header=%s", want, markup[start:min(len(markup), start+180)])
		}
	}
	descriptor, err := r.RuntimeDescriptor("ri-00000001")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, task := range descriptor.Tasks {
		if task.Kind == "math" {
			count++
		}
	}
	if count != 2 {
		t.Fatal("directive chrome created math tasks")
	}
}

func TestMathDeckRemapsReorderedAndRepeatedEquations(t *testing.T) {
	source := []byte("# First\n\nSee[^f].\n\n[^f]: $x$\n\n$x$\n\n---\n\n<!-- _class: columns -->\n<!-- layout: columns -->\n<!-- slot: left -->\n\n  $x$\n<!-- slot: right -->\n$x$\n<!-- /layout -->\n")
	r, err := Render(context.Background(), margo.New(margo.WithMath(true)), RenderInput{Name: "original.md", Markdown: source})
	if err != nil {
		t.Fatal(err)
	}
	for _, position := range []string{`data-margo-line="5" data-margo-column="7"`, `data-margo-line="7" data-margo-column="1"`, `data-margo-line="15" data-margo-column="3"`, `data-margo-line="17" data-margo-column="1"`} {
		if !strings.Contains(string(r.HTML()), position) {
			t.Fatalf("missing original position %s", position)
		}
	}
}

func TestMathDeckSourceBudgetIncludesAllSlots(t *testing.T) {
	source := []byte("<!-- _class: columns -->\n<!-- layout: columns -->\n<!-- slot: left -->\n" + strings.Repeat("$x$ ", 600) + "\n<!-- slot: right -->\n" + strings.Repeat("$y$ ", 600) + "\n<!-- /layout -->\n")
	_, err := ParseWithOptions("limits.md", source, WithMath(true))
	if err == nil || !strings.Contains(err.Error(), "math.document_limit_exceeded") {
		t.Fatalf("slot budget: %v", err)
	}
}

func TestMathDeckProtectsBlocksImmediatelyAfterSlotMarkers(t *testing.T) {
	source := []byte("<!-- _class: columns -->\n<!-- layout: columns -->\n<!-- slot: left -->\n$$\nx\n---\n<!-- slot: right -->\n$$\n<!-- slot: right -->\n$y$\n<!-- /layout -->\n")
	doc, err := ParseWithOptions("slots.md", source, WithMath(true))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Slides()) != 1 || len(doc.Slides()[0].Layout().Slots) != 2 {
		t.Fatal("math changed slot boundaries")
	}
	if !strings.Contains(string(doc.Slides()[0].Layout().Slots[0].Markdown), "<!-- slot: right -->") {
		t.Fatal("a slot marker inside math was consumed")
	}
	r, err := Render(context.Background(), margo.New(margo.WithMath(true)), RenderInput{Name: "slots.md", Markdown: source})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(r.HTML()), `class="margo-math margo-math--`) != 2 {
		t.Fatal("lost slot equations")
	}
}
