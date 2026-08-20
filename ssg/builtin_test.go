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

func TestBuiltinTopMainRendersSiteNavigation(t *testing.T) {
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
	siteNav, err := NewAreaBinding(hash, "index.html", BindingSpec{
		Kind: "site_navigation", Area: "top-nav",
	}, 0, templ.Raw(`<nav aria-label="main navigation">site</nav>`))
	if err != nil {
		t.Fatal(err)
	}
	document, err := NewAreaBinding(hash, "index.html", BindingSpec{Kind: "document", Area: "main-content"}, 0, templ.Raw(`<article>home</article>`))
	if err != nil {
		t.Fatal(err)
	}
	output, err := frame.Render(FrameInput{SchemaHash: hash, Bindings: map[string][]AreaBinding{
		"top-nav":      {siteNav},
		"main-content": {document},
	}})
	if err != nil {
		t.Fatal(err)
	}
	var rendered strings.Builder
	if err := output.Fragment.Render(context.Background(), &rendered); err != nil {
		t.Fatal(err)
	}
	markup := rendered.String()
	if !strings.Contains(markup, `<nav aria-label="main navigation">site</nav>`) {
		t.Fatalf("fragment = %s", markup)
	}
	if strings.Index(markup, `main navigation`) > strings.Index(markup, `>home<`) {
		t.Fatalf("site navigation rendered after document: %s", markup)
	}
}

func TestBuiltinDocsFrameRendersSiteAndLocalNavigation(t *testing.T) {
	frame, err := BuiltinFrame("top-left-main-right-footer")
	if err != nil {
		t.Fatal(err)
	}
	schema, err := frame.Schema(FrameContext{Profile: DocsProfile})
	if err != nil {
		t.Fatal(err)
	}
	hash, err := SchemaHash(schema)
	if err != nil {
		t.Fatal(err)
	}
	siteNav, err := NewAreaBinding(hash, "guide.html", BindingSpec{
		Kind: "site_navigation", Area: "top-nav",
	}, 0, templ.Raw(`<nav aria-label="site navigation">site</nav>`))
	if err != nil {
		t.Fatal(err)
	}
	localNav, err := NewAreaBinding(hash, "guide.html", BindingSpec{
		Kind: "navigation", Area: "left-nav",
	}, 0, templ.Raw(`<nav aria-label="section navigation">section</nav>`))
	if err != nil {
		t.Fatal(err)
	}
	document, err := NewAreaBinding(hash, "guide.html", BindingSpec{Kind: "document", Area: "main-content"}, 0, templ.Raw(`<article>guide</article>`))
	if err != nil {
		t.Fatal(err)
	}
	output, err := frame.Render(FrameInput{SchemaHash: hash, Bindings: map[string][]AreaBinding{
		"top-nav":      {siteNav},
		"left-nav":     {localNav},
		"main-content": {document},
	}})
	if err != nil {
		t.Fatal(err)
	}
	var rendered strings.Builder
	if err := output.Fragment.Render(context.Background(), &rendered); err != nil {
		t.Fatal(err)
	}
	markup := rendered.String()
	if !strings.Contains(markup, `site navigation`) || !strings.Contains(markup, `section navigation`) {
		t.Fatalf("fragment = %s", markup)
	}
	if strings.Index(markup, `site navigation`) > strings.Index(markup, `section navigation`) {
		t.Fatalf("site navigation rendered after local navigation: %s", markup)
	}
}

func TestValidateBindingsRejectsSiteNavigationInLeftNav(t *testing.T) {
	frame, err := BuiltinFrame("top-left-main-right-footer")
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
	siteNav, err := NewAreaBinding(hash, "guide.html", BindingSpec{
		Kind: "site_navigation", Area: "left-nav",
	}, 0, templ.Raw(`<nav>site</nav>`))
	if err != nil {
		t.Fatal(err)
	}
	document, err := NewAreaBinding(hash, "guide.html", BindingSpec{Kind: "document", Area: "main-content"}, 0, templ.Raw(`<article>guide</article>`))
	if err != nil {
		t.Fatal(err)
	}
	err = ValidateBindings(schema, map[string][]AreaBinding{
		"main-content": {document},
		"left-nav":     {siteNav},
	})
	if err == nil || !strings.Contains(err.Error(), `left-nav`) || !strings.Contains(err.Error(), `site_navigation`) {
		t.Fatalf("error = %v", err)
	}
}
