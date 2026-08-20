package margo

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestRenderHTMLProducesDefensiveProjection(t *testing.T) {
	result := mustRenderSource(t, "---\ntitle: Host title\n---\n\nBody with **meaning**.\n")
	editorial, err := RenderHTML(result)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, editorial.Fragment())
	for _, forbidden := range []string{"<!doctype", "<html", "<head", "<body", "<script", "<style", "data-theme="} {
		if strings.Contains(strings.ToLower(markup), forbidden) {
			t.Fatalf("fragment owns shell %q: %s", forbidden, markup)
		}
	}
	if strings.Count(markup, `<article class="margo-document">`) != 1 {
		t.Fatalf("article contract: %s", markup)
	}
	if editorial.PlainText() != "Body with meaning." {
		t.Fatalf("plain text = %q", editorial.PlainText())
	}
	if editorial.Metadata().Title != "Host title" {
		t.Fatalf("title = %q", editorial.Metadata().Title)
	}
}

func TestRenderHTMLPlainTextExcludesComponentChrome(t *testing.T) {
	editorial, err := RenderHTML(mustRenderSource(t, "# Example\n\n![Architecture diagram](diagram.png)\n\n```go\nfmt.Println(\"hello\")\n```\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := editorial.PlainText(); got != "Example Architecture diagram fmt.Println(\"hello\")" {
		t.Fatalf("plain text = %q", got)
	}
}

func TestRenderHTMLTitlePolicy(t *testing.T) {
	fromBody, err := RenderHTML(mustRenderSource(t, "# Body title\n"))
	if err != nil {
		t.Fatal(err)
	}
	if fromBody.Metadata().Title != "Body title" {
		t.Fatalf("body title fallback = %q", fromBody.Metadata().Title)
	}

	withoutHeading := mustRenderSource(t, "---\ntitle: Metadata title\n---\n\nBody\n")
	plain, err := RenderHTML(withoutHeading)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(renderComponent(t, plain.Fragment()), "<h1") {
		t.Fatal("default fragment invented a heading")
	}
	withHeader, err := RenderHTML(withoutHeading, WithHTMLHeader())
	if err != nil {
		t.Fatal(err)
	}
	if got := renderComponent(t, withHeader.Fragment()); strings.Count(got, "<h1") != 1 || !strings.Contains(got, ">Metadata title</h1>") {
		t.Fatalf("explicit header = %s", got)
	}

	conflict, err := RenderHTML(mustRenderSource(t, "---\ntitle: Metadata title\n---\n\n# Body title\n"), WithHTMLHeader())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(renderComponent(t, conflict.Fragment()), "<h1") != 1 {
		t.Fatal("body heading was duplicated")
	}
	diagnostics := conflict.Diagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Code != "html.title_conflict" || diagnostics[0].Severity != SeverityInfo {
		t.Fatalf("conflict diagnostics = %#v", diagnostics)
	}
}

func TestRenderHTMLFallsBackToSourceFilename(t *testing.T) {
	compiler := New()
	document, err := compiler.Compile(context.Background(), Source{Name: "guides/operations-runbook.md", Content: []byte("Body without a heading.\n")})
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	result, err := RenderHTML(rendered)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Metadata().Title; got != "operations-runbook" {
		t.Fatalf("filename title fallback = %q", got)
	}
}

func TestRenderHTMLRejectsNilAndDocumentShell(t *testing.T) {
	if _, err := RenderHTML(nil); diagnosticCode(err) != "html.result_required" {
		t.Fatalf("nil diagnostic = %v", err)
	}
	result := &RenderResult{content: templ.Raw(`<html><body>wrong</body></html>`)}
	if _, err := RenderHTML(result); diagnosticCode(err) != "html.metadata_invalid" {
		t.Fatalf("shell diagnostic = %v", err)
	}
}

func TestRenderHTMLAllowsOnlyProvenanceMarkedExtensionStyles(t *testing.T) {
	allowed := &RenderResult{content: templ.Raw(`<article class="margo-document"><style data-margo-extension-style="charts">.chart{display:block}</style><p>Chart</p></article>`)}
	editorial, err := RenderHTML(allowed)
	if err != nil {
		t.Fatal(err)
	}
	if editorial.PlainText() != "Chart" {
		t.Fatalf("plain text = %q", editorial.PlainText())
	}
	for _, markup := range []string{
		`<article class="margo-document"><style>.host{display:none}</style></article>`,
		`<article class="margo-document"><style data-margo-extension-style="other">.host{display:none}</style></article>`,
	} {
		if _, err := RenderHTML(&RenderResult{content: templ.Raw(markup)}); diagnosticCode(err) != "html.metadata_invalid" {
			t.Fatalf("unowned style accepted: %s: %v", markup, err)
		}
	}
}

func TestRenderHTMLRelocatesProvenanceMarkedChartScripts(t *testing.T) {
	requirements, err := mergeHTMLRequirements([]HTMLRequirement{{
		ID: "goshtoso-charts.runtime", Kind: HTMLScript,
		Inline: AssetRef{Path: "charts-runtime.js", MediaType: "application/javascript", Content: []byte("window.echarts = {};")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := RenderHTML(&RenderResult{
		content:          templ.Raw(`<article class="margo-document"><h1>Chart</h1><figure><script data-margo-extension-script="charts">window.chartReady = true;</script></figure></article>`),
		htmlRequirements: requirements,
	})
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, result.Fragment())
	if !strings.Contains(markup, `data-margo-chart-script-slot="0"`) || strings.Contains(markup, `data-margo-extension-script="charts"`) {
		t.Fatalf("chart script was not relocated: %s", markup)
	}
	if got := result.Requirements().List(); len(got) != 2 || got[1].ID != "margo.charts.inline.0" || got[1].LocalURL == "" {
		t.Fatalf("requirements = %+v", got)
	}
}

func TestHTMLResultIsDefensiveAndFingerprintSensitive(t *testing.T) {
	firstRendered := mustRenderSource(t, "---\ntitle: One\nauthors: [A]\ntags: [HTML]\n---\n\nBody\n")
	first, err := RenderHTML(firstRendered)
	if err != nil {
		t.Fatal(err)
	}
	metadata := first.Metadata()
	metadata.Authors[0] = "mutated"
	metadata.Tags[0] = "mutated"
	diagnostics := first.Diagnostics()
	diagnostics = append(diagnostics, Diagnostic{Code: "mutated"})
	if first.Metadata().Authors[0] != "A" || first.Metadata().Tags[0] != "HTML" || len(first.Diagnostics()) != 0 {
		t.Fatal("editorial result aliases caller")
	}

	equivalent, err := RenderHTML(mustRenderSource(t, "---\ntitle: One\nauthors: [A]\ntags: [HTML]\n---\n\nBody\n"))
	if err != nil {
		t.Fatal(err)
	}
	changedMetadata, err := RenderHTML(mustRenderSource(t, "---\ntitle: Two\nauthors: [A]\ntags: [HTML]\n---\n\nBody\n"))
	if err != nil {
		t.Fatal(err)
	}
	withHeader, err := RenderHTML(mustRenderSource(t, "---\ntitle: One\nauthors: [A]\ntags: [HTML]\n---\n\nBody\n"), WithHTMLHeader())
	if err != nil {
		t.Fatal(err)
	}
	requirements, err := mergeHTMLRequirements([]HTMLRequirement{{ID: "demo.styles", Kind: HTMLStylesheet, LocalURL: "/demo/styles.css"}})
	if err != nil {
		t.Fatal(err)
	}
	resultWithRequirement := *firstRendered
	resultWithRequirement.htmlRequirements = requirements
	withRequirement, err := RenderHTML(&resultWithRequirement)
	if err != nil {
		t.Fatal(err)
	}
	projectedRequirements := withRequirement.Requirements().List()
	projectedRequirements[0].ID = "mutated"
	if withRequirement.Requirements().List()[0].ID != "demo.styles" {
		t.Fatal("editorial requirements alias caller")
	}
	if first.Fingerprint() != equivalent.Fingerprint() {
		t.Fatal("equivalent editorial inputs changed identity")
	}
	if first.Fingerprint() == changedMetadata.Fingerprint() || first.Fingerprint() == withHeader.Fingerprint() || first.Fingerprint() == withRequirement.Fingerprint() {
		t.Fatal("metadata, requirements, or option did not change editorial identity")
	}
}
