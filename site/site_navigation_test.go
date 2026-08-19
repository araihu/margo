package site

import (
	"context"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

func TestWithoutGoshtosoStylesheetDeduplicatesByRequirementIdentity(t *testing.T) {
	markup := []byte(`<link rel="stylesheet" href="assets/styles.css"/><script src="assets/goshtoso.js"></script>`)
	requirements := margo.HTMLRequirements{}
	if got := string(withoutGoshtosoStylesheet(markup, requirements)); strings.Count(got, "styles.css") != 1 {
		t.Fatalf("layout-only stylesheet count = %d, want one: %s", strings.Count(got, "styles.css"), got)
	}

	compiler := margo.New()
	document, err := compiler.Compile(context.Background(), margo.Source{Name: "index.md", Content: []byte("# Home\n")})
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	htmlResult, err := margo.RenderHTML(rendered)
	if err != nil {
		t.Fatal(err)
	}
	requirements = htmlResult.Requirements()
	if got := string(withoutGoshtosoStylesheet(markup, requirements)); strings.Contains(got, "styles.css") {
		t.Fatalf("document-owned stylesheet was not removed from Goshtoso fragment: %s", got)
	}
}

func TestDocsPageHeadEmitsGoshtosoStylesheetOnceWithoutDocumentStyle(t *testing.T) {
	builder := &builder{
		config: &Config{Layout: &LayoutConfig{Kind: LayoutDocs}, Site: SiteConfig{Name: "Margo", BaseURL: "https://margo.example"}},
	}
	page := Page{Title: "Home", Description: "Margo docs", Canonical: "https://margo.example/"}
	head, err := builder.renderPageHeadForLayout(page, ResolvedLayout{Kind: LayoutDocs, dependencies: layoutDependencies{goshtosoNavigation: true}}, "", nil, margo.HTMLRequirements{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(head, "styles.css"); got != 1 {
		t.Fatalf("docs page head emits Goshtoso stylesheet %d times, want one: %s", got, head)
	}
}

func TestGithubRepositoryActionRendersAccessibleIconLink(t *testing.T) {
	markup, err := renderComponentBytes(githubRepositoryAction("https://github.com/araihu/margo"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(markup)
	for _, want := range []string{
		`data-margo-repository-link="true"`,
		`href="https://github.com/araihu/margo"`,
		`target="_blank"`,
		`rel="noopener noreferrer"`,
		`aria-label="Repository"`,
		`title="Repository"`,
		`<svg`,
		`aria-hidden="true"`,
		`class="sr-only">Repository</span>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("repository action missing %q: %s", want, html)
		}
	}
	if strings.Contains(html, `>Repository</a>`) {
		t.Fatalf("repository action exposes visible text link: %s", html)
	}
}

func TestSiteNavigationFamilyPagesAreDocsOnlyAndOverviewFirst(t *testing.T) {
	b := typedFamilyNavigationBuilder()
	b.configPages = []Page{
		{Source: "landing.md", Output: "landing.html", Locale: "en", Family: "module", Layout: "landing", Title: "Landing"},
		{Source: "module/a.md", Output: "module/a.html", Locale: "en", Family: "module", Layout: "docs", Title: "Module A"},
		{Source: "module/z/index.md", Output: "module/z/index.html", Locale: "en", Family: "module", Layout: "docs", Title: "Module Overview"},
		{Source: "pt-BR/module/index.md", Output: "pt-br/module/index.html", Locale: "pt-BR", Family: "module", Layout: "docs", Title: "Módulo"},
		{Source: "cli/index.md", Output: "cli/index.html", Locale: "en", Family: "cli", Layout: "docs", Title: "CLI"},
	}
	b.docsFamilies = []docsFamily{{
		ID:       "module",
		Locale:   "en",
		Overview: b.configPages[2],
	}}

	pages := b.familyPages(b.configPages[1])
	if got, want := len(pages), 2; got != want {
		t.Fatalf("family page count = %d, want %d: %+v", got, want, pages)
	}
	if pages[0].Source != "module/z/index.md" || pages[1].Source != "module/a.md" {
		t.Fatalf("family pages = %+v, want overview then route order", pages)
	}
	pagination := b.paginationFragment(b.configPages[1])
	if !strings.Contains(pagination, `rel="prev"`) || !strings.Contains(pagination, `Module Overview`) || strings.Contains(pagination, `Landing`) || strings.Contains(pagination, `CLI`) {
		t.Fatalf("pagination is not docs-family scoped: %s", pagination)
	}

	b.configPages = []Page{
		{Source: "landing.md", Output: "landing.html", Locale: "en", Family: "module", Layout: "landing", Title: "Landing"},
		{Source: "module/index.md", Output: "module/index.html", Locale: "en", Family: "module", Layout: "docs", Title: "Module"},
	}
	b.docsFamilies = []docsFamily{{ID: "module", Locale: "en", Overview: b.configPages[1]}}
	if pagination := b.paginationFragment(b.configPages[1]); pagination != "" {
		t.Fatalf("single-page docs family pagination = %s, want suppressed", pagination)
	}
}

func TestSiteNavigationFamilyNavbarUsesConfiguredOrder(t *testing.T) {
	b := typedFamilyNavigationBuilder()
	module := Page{Source: "module/index.md", Output: "module/index.html", Locale: "en", Family: "module", Layout: "docs", Title: "Module Overview"}
	cli := Page{Source: "cli/index.md", Output: "cli/index.html", Locale: "en", Family: "cli", Layout: "docs", Title: "CLI Overview"}
	b.configPages = []Page{cli, module}
	b.docsFamilies = []docsFamily{
		{ID: "module", Locale: "en", Overview: module},
		{ID: "cli", Locale: "en", Overview: cli},
	}

	markup, err := b.siteNavigationFragment(module)
	if err != nil {
		t.Fatal(err)
	}
	moduleIndex := strings.Index(markup, `data-margo-family-link="module"`)
	cliIndex := strings.Index(markup, `data-margo-family-link="cli"`)
	if moduleIndex < 0 || cliIndex < 0 || moduleIndex >= cliIndex {
		t.Fatalf("family links do not follow configured order: %s", markup)
	}
	if !strings.Contains(markup, "Module Overview") || !strings.Contains(markup, "CLI Overview") {
		t.Fatalf("overview titles do not label family links: %s", markup)
	}
	if got := strings.Count(markup, `aria-current="location"`); got != 1 {
		t.Fatalf("current family marker count = %d, want 1: %s", got, markup)
	}
}

func TestSiteNavigationExposesActiveFamilyPagesToMobileNavbar(t *testing.T) {
	b := typedFamilyNavigationBuilder()
	overview := Page{Source: "module/index.md", Output: "module/index.html", Locale: "en", Family: "module", Layout: "docs", Title: "Module"}
	guide := Page{Source: "module/guide.md", Output: "module/guide.html", Locale: "en", Family: "module", Layout: "docs", Title: "Guide"}
	b.configPages = []Page{guide, overview}
	b.docsFamilies = []docsFamily{{ID: "module", Locale: "en", Overview: overview}}

	markup, err := b.siteNavigationFragment(guide)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`data-margo-mobile-menu-trigger="true"`,
		`data-margo-mobile-menu="true"`,
		`data-margo-navbar-desktop-actions="true"`,
		`data-margo-family-page-link="module/index.md"`,
		`data-margo-family-page-link="module/guide.md"`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("navbar missing mobile family navigation hook %q: %s", want, markup)
		}
	}
	if got := strings.Count(markup, `data-margo-family-page-link="module/guide.md"`); got != 2 {
		t.Fatalf("active page link rendered %d times, want desktop/mobile projections: %s", got, markup)
	}
	if got := strings.Count(markup, `aria-current="page"`); got != 2 {
		t.Fatalf("active page state rendered %d times, want desktop/mobile projections: %s", got, markup)
	}
}

func TestTOCFragmentUsesOneNativeResponsiveDrawer(t *testing.T) {
	b := typedFamilyNavigationBuilder()
	markup := b.tocFragment([]byte(`<article><h1 id="overview">Overview</h1><h2 id="install">Install</h2></article>`), "en")

	for _, want := range []string{
		`<details class="margo-toc-drawer" data-margo-toc-drawer="true">`,
		`<summary data-margo-toc-summary="true">On this page</summary>`,
		`<nav class="margo-toc" aria-label="On this page" data-margo-toc="true"><p class="margo-toc-title" data-margo-toc-title="true">On this page</p>`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("TOC missing responsive drawer semantic %q: %s", want, markup)
		}
	}
	if got := strings.Count(markup, `data-margo-toc-link=`); got != 2 {
		t.Fatalf("TOC link count = %d, want one projection of each heading: %s", got, markup)
	}
	if got := strings.Count(markup, `data-margo-toc-title="true"`); got != 1 {
		t.Fatalf("desktop rail title count = %d, want one: %s", got, markup)
	}
}

func TestSiteNavigationFamilyNavbarSuppressesSingleEffectiveFamily(t *testing.T) {
	b := typedFamilyNavigationBuilder()
	module := Page{Source: "module/index.md", Output: "module/index.html", Locale: "en", Family: "module", Layout: "docs", Title: "Module Overview"}
	b.configPages = []Page{module}
	b.docsFamilies = []docsFamily{{ID: "module", Locale: "en", Overview: module}}

	markup, err := b.siteNavigationFragment(module)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(markup, `data-margo-family-navigation`) || strings.Contains(markup, `data-margo-family-link`) {
		t.Fatalf("single-family secondary navbar was rendered: %s", markup)
	}
}

func typedFamilyNavigationBuilder() *builder {
	return &builder{
		config: &Config{
			Layout:  &LayoutConfig{Kind: LayoutDocs},
			Site:    SiteConfig{Name: "Margo", BaseURL: "https://margo.example", Home: "index.md", Logo: "assets/logo.svg"},
			Locales: LocaleConfig{Default: "en", Supported: []string{"en", "pt-BR"}},
		},
	}
}
