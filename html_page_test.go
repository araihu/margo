package margo

import (
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestRenderHTMLPageOwnsOnlyGenericDocumentShell(t *testing.T) {
	result, err := RenderHTML(mustRenderSource(t, "---\ntitle: Generic page\ndescription: Reusable HTML output.\n---\n\nBody.\n"))
	if err != nil {
		t.Fatal(err)
	}
	page, err := RenderHTMLPage(result, HTMLPageInput{
		Theme: ThemeModern, ColorMode: ColorModeLight, DependencyMode: HTMLDependenciesInline,
		Head: templ.Raw(`<meta name="robots" content="noindex">`),
	})
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, page)
	for _, want := range []string{
		`<!doctype html>`, `<title>Generic page</title>`,
		`<meta name="description" content="Reusable HTML output.">`,
		`<meta name="robots" content="noindex">`,
		`data-margo-html-fingerprint="` + result.Fingerprint().String() + `"`,
		`<article class="margo-document">`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("missing %q: %s", want, markup)
		}
	}
	for _, forbidden := range []string{`rel="canonical"`, `property="og:`, `name="twitter:`, `property="article:`} {
		if strings.Contains(markup, forbidden) {
			t.Fatalf("generic page contains publication policy %q: %s", forbidden, markup)
		}
	}
}

func TestRenderHTMLPageRejectsInvalidShellInput(t *testing.T) {
	result, err := RenderHTML(mustRenderSource(t, "# Page\n"))
	if err != nil {
		t.Fatal(err)
	}
	if component, err := RenderHTMLPage(result, HTMLPageInput{Theme: ThemeModern, ColorMode: ColorModeLight, DependencyMode: "remote"}); err == nil || component != nil {
		t.Fatalf("component=%v err=%v", component, err)
	}
}
