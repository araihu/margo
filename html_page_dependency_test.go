package margo

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func testThemeStylesheet() AssetRef {
	content := []byte(`[data-theme="araihu"]{--color-surface:#fff;--color-on-surface:#111}`)
	digest := sha256.Sum256(content)
	return AssetRef{Path: "themes/araihu.css", MediaType: "text/css", SHA256: hex.EncodeToString(digest[:]), Content: content}
}

func TestRenderHTMLPageResolvesInlineDependenciesInOrder(t *testing.T) {
	result, err := RenderHTML(mustRenderSource(t, "---\ntitle: Dependencies\n---\n\n| A | B |\n|---|---|\n| 1 | 2 |\n"))
	if err != nil {
		t.Fatal(err)
	}
	page, err := RenderHTMLPage(result, HTMLPageInput{
		Theme: ThemeName("araihu"), ColorMode: ColorModeDark,
		DependencyMode: HTMLDependenciesInline, ThemeStylesheet: testThemeStylesheet(),
	})
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, page)
	order := []string{"goshtoso.styles", "margo.document.styles", "margo.table-sort", "margo.theme.araihu"}
	last := -1
	for _, id := range order {
		index := strings.Index(markup, `data-margo-requirement="`+id+`"`)
		if index <= last {
			t.Fatalf("dependency order %v: %s", order, markup)
		}
		last = index
	}
	for _, want := range []string{`[data-theme="araihu"]`, `const collator = new Intl.Collator`} {
		if !strings.Contains(markup, want) {
			t.Fatalf("missing inline dependency %q: %s", want, markup)
		}
	}
}

func TestRenderHTMLPageSupportsLocalDependencies(t *testing.T) {
	result, err := RenderHTML(mustRenderSource(t, "# Local page\n"))
	if err != nil {
		t.Fatal(err)
	}
	page, err := RenderHTMLPage(result, HTMLPageInput{
		Theme: ThemeModern, ColorMode: ColorModeLight, DependencyMode: HTMLDependenciesLocal,
	})
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, page)
	if !strings.Contains(markup, `<link rel="stylesheet"`) || strings.Contains(markup, `rel="canonical"`) {
		t.Fatalf("local generic page = %s", markup)
	}
}
