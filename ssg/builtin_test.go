package ssg

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestBuiltinFramesPublishStableSchemas(t *testing.T) {
	for _, name := range BuiltinFrameNames() {
		frame, err := BuiltinFrame(name)
		if err != nil {
			t.Fatal(err)
		}
		schema, err := frame.Schema(FrameContext{Profile: DocsProfile, Locale: "en", Direction: "ltr"})
		if err != nil {
			t.Fatalf("%s schema: %v", name, err)
		}
		if err := ValidateFrameSchema(schema, DocsProfile); name != "main" && name != "main-footer" && err != nil {
			t.Fatalf("%s docs schema: %v", name, err)
		}
		if _, err := SchemaHash(schema); err != nil {
			t.Fatalf("%s hash: %v", name, err)
		}
	}
}

func TestBuiltinFrameRendersDocumentAndPaginationInMain(t *testing.T) {
	frame, err := BuiltinFrame("top-left-main-footer")
	if err != nil {
		t.Fatal(err)
	}
	schema, err := frame.Schema(FrameContext{})
	if err != nil {
		t.Fatal(err)
	}
	hash, err := SchemaHash(schema)
	if err != nil {
		t.Fatal(err)
	}
	document, err := NewAreaBinding(hash, "guide.html", BindingSpec{Kind: "document", Area: "main-content"}, 0, templ.Raw(`<article class="margo-document"><h1>Guide</h1></article>`))
	if err != nil {
		t.Fatal(err)
	}
	pagination, err := NewAreaBinding(hash, "guide.html", BindingSpec{Kind: "pagination", Area: "main-content", Slot: "after-article"}, 0, templ.Raw(`<nav aria-label="Article navigation">next</nav>`))
	if err != nil {
		t.Fatal(err)
	}
	output, err := frame.Render(FrameInput{SchemaHash: hash, Bindings: map[string][]AreaBinding{"main-content": {document, pagination}}})
	if err != nil {
		t.Fatal(err)
	}
	var rendered strings.Builder
	if err := output.Fragment.Render(context.Background(), &rendered); err != nil {
		t.Fatal(err)
	}
	markup := rendered.String()
	if strings.Count(markup, `<main `) != 1 || !strings.Contains(markup, `id="margo-document"`) || !strings.Contains(markup, `Article navigation`) {
		t.Fatalf("fragment = %s", markup)
	}
	if strings.Index(markup, `>Guide<`) > strings.Index(markup, `Article navigation`) {
		t.Fatalf("pagination rendered before document: %s", markup)
	}
}

func TestValidateBindingsRejectsWrongKindAndDuplicateMounts(t *testing.T) {
	frame, err := BuiltinFrame("top-main")
	if err != nil {
		t.Fatal(err)
	}
	schema, err := frame.Schema(FrameContext{})
	if err != nil {
		t.Fatal(err)
	}
	hash, err := SchemaHash(schema)
	if err != nil {
		t.Fatal(err)
	}
	good, err := NewAreaBinding(hash, "index.html", BindingSpec{Kind: "document", Area: "main-content"}, 0, templ.Raw(`<article><h1>Home</h1></article>`))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateBindings(schema, map[string][]AreaBinding{"main-content": {good}, "top-nav": {good}}); err == nil || !strings.Contains(err.Error(), "top-nav") {
		t.Fatalf("wrong kind error = %v", err)
	}
}
