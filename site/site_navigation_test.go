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
		t.Fatalf("profile-only stylesheet count = %d, want one: %s", strings.Count(got, "styles.css"), got)
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

func TestProfilePageHeadEmitsGoshtosoStylesheetOnceWithoutDocumentStyle(t *testing.T) {
	builder := &builder{
		profileMode: true,
		config:      &Config{Site: SiteConfig{Name: "Margo", BaseURL: "https://margo.example"}},
	}
	page := Page{Title: "Home", Description: "Margo docs", Canonical: "https://margo.example/"}
	head, err := builder.renderPageHead(page, "", nil, margo.HTMLRequirements{})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(head, "styles.css"); got != 1 {
		t.Fatalf("profile-only page head emits Goshtoso stylesheet %d times, want one: %s", got, head)
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
