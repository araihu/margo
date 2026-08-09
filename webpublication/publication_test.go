package webpublication

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/araihu/margo"
)

func TestRenderAddsPublicWebPolicyAroundGenericHTML(t *testing.T) {
	result := compileHTML(t, "---\ntitle: Public article\ndescription: A public example.\nauthors: [Ana]\npublishedAt: \"2026-08-09T12:00:00-03:00\"\ntags: [Go]\n---\n\nBody.\n")
	authority := loadAuthority(t)
	page, err := Render(result, Input{
		Kind: KindArticle, Authority: authority, RoutePath: authority.Routes.Representative,
		SiteName: "Arai Hû", Locale: "pt_BR",
		Image: SocialImage{
			URL:      string(authority.CanonicalOrigin) + authority.Routes.Preview,
			MIMEType: authority.Asset.MIMEType, Width: authority.Asset.Width, Height: authority.Asset.Height,
			Alt: "Article preview.",
		},
		Page: margo.HTMLPageInput{
			Theme: margo.ThemeModern, ColorMode: margo.ColorModeLight,
			DependencyMode: margo.HTMLDependenciesInline,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, page)
	for _, want := range []string{
		`rel="canonical" href="https://margo.invalid/guide"`,
		`property="og:type" content="article"`,
		`property="article:author" content="Ana"`,
		`class="margo-webpublication-byline"`,
		`data-margo-html-fingerprint="` + result.Fingerprint().String() + `"`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("missing %q: %s", want, markup)
		}
	}
	if strings.Count(markup, `<title>`) != 1 || strings.Count(markup, `name="description"`) != 1 {
		t.Fatalf("base metadata ownership changed: %s", markup)
	}
}

func TestRenderRejectsUnlistedPublicRoute(t *testing.T) {
	result := compileHTML(t, "---\ntitle: Public article\ndescription: A public example.\n---\n\nBody.\n")
	authority := loadAuthority(t)
	component, err := Render(result, Input{
		Kind: KindDocument, Authority: authority, RoutePath: "/unlisted",
		SiteName: "Arai Hû",
		Image: SocialImage{
			URL:      string(authority.CanonicalOrigin) + authority.Routes.Preview,
			MIMEType: authority.Asset.MIMEType, Width: authority.Asset.Width, Height: authority.Asset.Height,
			Alt: "Document preview.",
		},
		Page: margo.HTMLPageInput{Theme: margo.ThemeModern, ColorMode: margo.ColorModeLight, DependencyMode: margo.HTMLDependenciesInline},
	})
	if err == nil || component != nil {
		t.Fatalf("component=%v err=%v", component, err)
	}
}

func compileHTML(t *testing.T, markdown string) *margo.HTMLResult {
	t.Helper()
	compiler := margo.New()
	document, err := compiler.Compile(context.Background(), margo.Source{Name: "test.md", Content: []byte(markdown)})
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	result, err := margo.RenderHTML(rendered)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func loadAuthority(t *testing.T) AuthorityRecord {
	t.Helper()
	data, err := os.ReadFile("../testdata/authority/record.json")
	if err != nil {
		t.Fatal(err)
	}
	record, err := VerifyAuthorityRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func renderComponent(t *testing.T, component templ.Component) string {
	t.Helper()
	var output strings.Builder
	if err := component.Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	return output.String()
}
