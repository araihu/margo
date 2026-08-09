package margo

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

func loadPublicationAuthority(t *testing.T) AuthorityRecord {
	t.Helper()
	data, err := os.ReadFile("testdata/authority/record.json")
	if err != nil {
		t.Fatal(err)
	}
	record, err := VerifyAuthorityRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func mustEditorialFixture(t *testing.T) *EditorialResult {
	t.Helper()
	source, err := os.ReadFile("testdata/markdown/editorial-article.md")
	if err != nil {
		t.Fatal(err)
	}
	editorial, err := RenderEditorial(mustRenderSource(t, string(source)))
	if err != nil {
		t.Fatal(err)
	}
	return editorial
}

func testThemeStylesheet() AssetRef {
	content := []byte(`[data-theme="araihu"]{--color-surface:#fff;--color-on-surface:#111}`)
	digest := sha256.Sum256(content)
	return AssetRef{Path: "themes/araihu.css", MediaType: "text/css", SHA256: hex.EncodeToString(digest[:]), Content: content}
}

func publicPublicationInput(authority AuthorityRecord, mode HTMLDependencyMode) PublicationInput {
	return PublicationInput{
		Mode: PublicationPublic, Kind: PublicationArticle,
		Authority: authority, RoutePath: authority.Routes.Representative,
		SiteName: "Arai Hû", Locale: "pt_BR",
		Image: SocialImage{
			URL:      string(authority.CanonicalOrigin) + authority.Routes.Preview,
			MIMEType: authority.Asset.MIMEType, Width: authority.Asset.Width, Height: authority.Asset.Height,
			Alt: "Editorial preview fixture.",
		},
		Theme: ThemeName("araihu"), ColorMode: ColorModeDark,
		DependencyMode: mode, ThemeStylesheet: testThemeStylesheet(),
	}
}

func TestRenderPublicationComposesArticleInInitialHTML(t *testing.T) {
	authority := loadPublicationAuthority(t)
	page, err := RenderPublication(mustEditorialFixture(t), publicPublicationInput(authority, HTMLDependenciesInline))
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, page)
	for _, want := range []string{
		`<!doctype html>`, `lang="en"`, `data-theme="araihu"`, `class="dark"`,
		`property="og:type" content="article"`, `rel="canonical" href="https://margo.invalid/guide"`,
		`<address`, `<time datetime="2026-08-09T12:00:00-03:00"`, `<article class="margo-document">`,
		`data-margo-requirement="goshtoso.styles"`, `data-margo-requirement="margo.document.styles"`,
		`data-margo-requirement="margo.table-sort"`, `data-margo-requirement="margo.theme.araihu"`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("missing %q: %s", want, markup)
		}
	}
	for _, field := range []string{`name="description"`, `rel="canonical"`, `property="og:url"`, `property="article:published_time"`} {
		if strings.Count(markup, field) != 1 {
			t.Fatalf("metadata %q count = %d: %s", field, strings.Count(markup, field), markup)
		}
	}
	order := []string{"goshtoso.styles", "margo.document.styles", "margo.table-sort", "margo.theme.araihu"}
	last := -1
	for _, id := range order {
		index := strings.Index(markup, `data-margo-requirement="`+id+`"`)
		if index <= last {
			t.Fatalf("dependency order %v: %s", order, markup)
		}
		last = index
	}
}

func TestRenderPublicationSupportsLocalDocumentAndPrivateOutput(t *testing.T) {
	authority := loadPublicationAuthority(t)
	input := publicPublicationInput(authority, HTMLDependenciesLocal)
	input.Kind = PublicationDocument
	page, err := RenderPublication(mustEditorialFixture(t), input)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, page)
	if !strings.Contains(markup, `property="og:type" content="website"`) || !strings.Contains(markup, `<link rel="stylesheet"`) || !strings.Contains(markup, `<script defer`) {
		t.Fatalf("local document dependencies = %s", markup)
	}
	if strings.Contains(markup, `property="article:`) {
		t.Fatalf("document emitted article metadata: %s", markup)
	}

	privateInput := input
	privateInput.Mode = PublicationPrivate
	privateInput.Authority = AuthorityRecord{}
	privateInput.RoutePath = ""
	privateInput.SiteName = ""
	privateInput.Locale = ""
	privateInput.Image = SocialImage{}
	privatePage, err := RenderPublication(mustEditorialFixture(t), privateInput)
	if err != nil {
		t.Fatal(err)
	}
	privateMarkup := renderComponent(t, privatePage)
	for _, forbidden := range []string{`rel="canonical"`, `property="og:`, `name="twitter:`, "https://margo.invalid"} {
		if strings.Contains(privateMarkup, forbidden) {
			t.Fatalf("private output contains %q: %s", forbidden, privateMarkup)
		}
	}
}

func TestPublicationInputRejectsInvalidValuesBeforeRendering(t *testing.T) {
	authority := loadPublicationAuthority(t)
	valid := publicPublicationInput(authority, HTMLDependenciesInline)
	tests := []struct {
		name      string
		editorial *EditorialResult
		mutate    func(*PublicationInput)
	}{
		{name: "nil editorial", editorial: nil},
		{name: "unlisted route", editorial: mustEditorialFixture(t), mutate: func(input *PublicationInput) { input.RoutePath = "/other" }},
		{name: "unsafe theme", editorial: mustEditorialFixture(t), mutate: func(input *PublicationInput) { input.Theme = `araihu\" onclick=\"alert(1)` }},
		{name: "invalid theme stylesheet", editorial: mustEditorialFixture(t), mutate: func(input *PublicationInput) { input.ThemeStylesheet.SHA256 = strings.Repeat("0", 64) }},
		{name: "invalid dependency mode", editorial: mustEditorialFixture(t), mutate: func(input *PublicationInput) { input.DependencyMode = "remote" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			if test.mutate != nil {
				test.mutate(&input)
			}
			component, err := RenderPublication(test.editorial, input)
			if err == nil || component != nil {
				t.Fatalf("component=%v err=%v", component, err)
			}
		})
	}
}
