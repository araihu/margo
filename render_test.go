package margo

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func mustRenderSource(t *testing.T, source string, options ...RenderOption) *RenderResult {
	t.Helper()
	compiler := New(WithHostPolicy(Policy{RawHTML: RawHTMLDeny, OutputBytes: MaxOutputBytes}))
	document, err := compiler.Compile(context.Background(), Source{Name: "render.md", Content: []byte(source)})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	result, err := compiler.Render(context.Background(), document, options...)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	return result
}

func renderComponent(t *testing.T, component templ.Component) string {
	t.Helper()
	var buffer bytes.Buffer
	if err := component.Render(context.Background(), &buffer); err != nil {
		t.Fatalf("component.Render() error = %v", err)
	}
	return buffer.String()
}

func TestSemanticRender(t *testing.T) {
	result := mustRenderSource(t, "# Hello\n\nA [safe link](https://example.com).\n\n- one\n- two\n\n> quoted\n")
	markup := renderComponent(t, result.Content())
	for _, want := range []string{
		`<h1 id="hello">Hello</h1>`,
		`<p class="margo-document__lead">A <a href="https://example.com">safe link</a>.</p>`,
		`<ul><li>one</li><li>two</li></ul>`,
		`<blockquote><p>quoted</p></blockquote>`,
	} {
		if !bytes.Contains([]byte(markup), []byte(want)) {
			t.Fatalf("rendered markup missing %q:\n%s", want, markup)
		}
	}
}

func TestSemanticRenderPreservesLinksInsideMarkdownTableCells(t *testing.T) {
	markup := renderComponent(t, mustRenderSource(t, "# Table links\n\n| Resource | Destination |\n| --- | --- |\n| Guide | [Open guide](guide.md) |\n\n[Open guide outside table](guide.md).\n").Content())
	if !strings.Contains(markup, `<a href="guide.md">Open guide</a>`) {
		t.Fatalf("table cell link was not rendered as an anchor:\n%s", markup)
	}
	if !strings.Contains(markup, `<a href="guide.md">Open guide outside table</a>`) {
		t.Fatalf("outside-table link regression:\n%s", markup)
	}
}

func TestSemanticRenderRejectsMalformedLinksInsideMarkdownTableCells(t *testing.T) {
	compiler := New()
	document, err := compiler.Compile(context.Background(), Source{Name: "table.md", Content: []byte("# Table links\n\n| Resource | Destination |\n| --- | --- |\n| Unsafe | [Open](javascript:alert(1)) |\n")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := compiler.Render(context.Background(), document); err == nil || !strings.Contains(err.Error(), "render.link_invalid") {
		t.Fatalf("malformed table link error = %v, want render.link_invalid", err)
	}
}

func TestFirstParagraphAfterDocumentTitleIsTheLead(t *testing.T) {
	markup := renderComponent(t, mustRenderSource(t, "# Visual foundation\n\nPurpose and scope.\n\n## Evidence\n\nOrdinary body copy.\n").Content())
	if !strings.Contains(markup, `<h1 id="visual-foundation">Visual foundation</h1><p class="margo-document__lead">Purpose and scope.</p>`) {
		t.Fatalf("document purpose is not identified as the lead:\n%s", markup)
	}
	if strings.Contains(markup, `<p class="margo-document__lead">Ordinary body copy.</p>`) {
		t.Fatalf("ordinary body copy was promoted to lead:\n%s", markup)
	}
}

func TestSemanticRenderKeepsOneDocumentTitleAndOneLead(t *testing.T) {
	markup := renderComponent(t, mustRenderSource(t, "# Primary title\n\nDocument purpose.\n\nSecondary setext title\n======================\n\nSection body.\n").Content())
	if strings.Count(markup, "<h1 ") != 1 {
		t.Fatalf("document title count = %d, want 1:\n%s", strings.Count(markup, "<h1 "), markup)
	}
	if strings.Count(markup, `class="margo-document__lead"`) != 1 {
		t.Fatalf("document lead count = %d, want 1:\n%s", strings.Count(markup, `class="margo-document__lead"`), markup)
	}
	if !strings.Contains(markup, `<h2 id="secondary-setext-title">Secondary setext title</h2><p>Section body.</p>`) {
		t.Fatalf("later level-one heading was not demoted into the section hierarchy:\n%s", markup)
	}
}

func TestTitledMarkdownImageUsesSemanticFigureAndSharedCaptionRole(t *testing.T) {
	markup := renderComponent(t, mustRenderSource(t, `![Deployment topology](/topology.png "Services and trust boundaries")`).Content())
	for _, want := range []string{
		`<figure class="margo-figure">`,
		`<img src="/topology.png" alt="Deployment topology"`,
		`<figcaption class="margo-figure-caption">Services and trust boundaries</figcaption>`,
		`</figure>`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("titled image is missing semantic figure content %q:\n%s", want, markup)
		}
	}
}

func TestSemanticRenderUsesFrontmatterBodyForFenceLanguage(t *testing.T) {
	result := mustRenderSource(t, "---\ntitle: With code\n---\n\n~~~go\nfmt.Println(\"hello\")\n~~~\n")
	markup := renderComponent(t, result.Content())
	if !bytes.Contains([]byte(markup), []byte(`aria-label="Copy go code"`)) {
		t.Fatalf("code fence language was not preserved after frontmatter:\n%s", markup)
	}
}

func TestMermaidSourceStartsCollapsedButRemainsDisclosure(t *testing.T) {
	result := mustRenderSource(t, "```mermaid\nflowchart TD\n  A --> B\n```\n")
	markup := renderComponent(t, result.Content())
	want := `<details class="margo-mermaid__source"><summary>Mermaid source`
	if !bytes.Contains([]byte(markup), []byte(want)) {
		t.Fatalf("Mermaid source disclosure is not collapsed by default:\n%s", markup)
	}
	if bytes.Contains([]byte(markup), []byte(`<details open class="margo-mermaid__source">`)) {
		t.Fatalf("Mermaid source disclosure is expanded by default:\n%s", markup)
	}
}

func TestMermaidFiguresUseUniqueContextualAccessibleNames(t *testing.T) {
	markup := renderComponent(t, mustRenderSource(t, "## Request flow\n\n```mermaid\nflowchart LR\n  source[Markdown source] --> ready{Runtime ready?}\n  ready --> pdf[PDF artifact]\n```\n\n## Handoff\n\n```mermaid\nsequenceDiagram\n  participant Author\n  participant Margo\n  Author->>Margo: Compile source\n```\n").Content())
	for _, want := range []string{
		`id="margo-mermaid-caption-0"`,
		`aria-labelledby="margo-mermaid-caption-0"`,
		`aria-describedby="margo-mermaid-source-0"`,
		`<span id="margo-mermaid-source-0" class="margo-mermaid__accessible-source">Complete Mermaid source: flowchart LR`,
		`ready --&gt; pdf[PDF artifact]</span>`,
		`data-margo-print-layout="landscape"`,
		`<figcaption id="margo-mermaid-caption-0" class="margo-figure-caption">Flowchart connecting Markdown source, Runtime ready?, and PDF artifact.</figcaption>`,
		`id="margo-mermaid-caption-1"`,
		`aria-labelledby="margo-mermaid-caption-1"`,
		`aria-describedby="margo-mermaid-source-1"`,
		`<figcaption id="margo-mermaid-caption-1" class="margo-figure-caption">Sequence diagram showing interactions between Author and Margo.</figcaption>`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("Mermaid figure is missing %q:\n%s", want, markup)
		}
	}
	if strings.Contains(markup, `aria-label="Mermaid diagram"`) {
		t.Fatalf("Mermaid figures still share a generic accessible name:\n%s", markup)
	}
	if !strings.Contains(markup, `<span class="margo-mermaid__overflow-cue">Scroll diagram horizontally to inspect all labels.</span>`) {
		t.Fatalf("Mermaid figure has no narrow-layout overflow cue:\n%s", markup)
	}
}

func TestSemanticRenderMatchesGolden(t *testing.T) {
	got := renderComponent(t, mustRenderSource(t, "# Hello\n\nA [safe link](https://example.com).\n\n- one\n- two\n\n> quoted\n").Content())
	wantBytes, err := os.ReadFile("testdata/render/semantic.html")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSuffix(string(wantBytes), "\n")
	if got != want {
		t.Fatalf("semantic golden mismatch:\nwant %s\n got %s", wantBytes, got)
	}
}

func TestC0RootModuleTransferRemainsHistorical(t *testing.T) {
	const (
		expectedSchemaVersion   = "margo/root-module-transfer/v1"
		expectedModulePath      = "github.com/araihu/margo"
		expectedModuleVersion   = "v0.0.1"
		expectedGoModPostSHA256 = "0eb36e99f0c59989a8c8772899acafa7b30dd205c241801b2d1c52ad775617fe"
		expectedGoSumPostSHA256 = "1c7ae9b89ad246a943998c8e7a4a4f19bd59a53f84409e0d930ebe9b1670ddbb"
	)
	transferBytes, err := os.ReadFile("integration/root-module-transfer.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var transfer struct {
		SchemaVersion         string
		ModulePath            string
		ModuleVersion         string
		GoModPostSHA256       string
		GoSumPostSHA256       string
		C5ExpectedGoModSHA256 string
		C5ExpectedGoSumSHA256 string
	}
	if err := json.Unmarshal(transferBytes, &transfer); err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name string
		got  string
		want string
	}{
		{name: "schemaVersion", got: transfer.SchemaVersion, want: expectedSchemaVersion},
		{name: "modulePath", got: transfer.ModulePath, want: expectedModulePath},
		{name: "moduleVersion", got: transfer.ModuleVersion, want: expectedModuleVersion},
		{name: "goModPostSHA256", got: transfer.GoModPostSHA256, want: expectedGoModPostSHA256},
		{name: "c5ExpectedGoModSHA256", got: transfer.C5ExpectedGoModSHA256, want: expectedGoModPostSHA256},
		{name: "goSumPostSHA256", got: transfer.GoSumPostSHA256, want: expectedGoSumPostSHA256},
		{name: "c5ExpectedGoSumSHA256", got: transfer.C5ExpectedGoSumSHA256, want: expectedGoSumPostSHA256},
	}
	for _, check := range checks {
		if check.got == "" || check.got != check.want {
			t.Fatalf("historical transfer %s = %q, want %q", check.name, check.got, check.want)
		}
	}
}
