package margo

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	goshtosoassets "github.com/araihu/goshtoso/assets"
)

func TestHeadOwnerSelectionIsFrozenBeforeSocialTask(t *testing.T) {
	selection := FrozenHeadOwnerSelection()
	if selection.SchemaVersion != "margo/head-owner-selection/v1" {
		t.Fatalf("schemaVersion = %q", selection.SchemaVersion)
	}
	if selection.Owner != "goshtoso" && selection.Owner != "margo" {
		t.Fatalf("unexpected owner %q", selection.Owner)
	}
	if selection.Primitive != "head.Metadata" && selection.Primitive != "socialMetadataTags" {
		t.Fatalf("unexpected primitive %q", selection.Primitive)
	}
	if selection.APISourcePath == "" || len(selection.APISourceSHA256) != 64 {
		t.Fatalf("incomplete API evidence: %#v", selection)
	}
	if err := selection.Validate(); err != nil {
		t.Fatalf("frozen selection invalid: %v", err)
	}
}

func TestHeadOwnerSelectionRejectsUnknownAndTrailingFields(t *testing.T) {
	valid := `{"schemaVersion":"margo/head-owner-selection/v1","owner":"margo","primitive":"socialMetadataTags","goshtosoCommit":"module:v0.1.2","goshtosoTree":"module-cache:v0.1.2","apiSourcePath":"components/head/component.go","apiSourceSHA256":"833562eafa47d917587c21e300d28c45006b855a569266b96041123ca870b3fb"}`
	if _, err := ParseHeadOwnerSelection([]byte(valid + ` {"extra":true}`)); err == nil {
		t.Fatal("trailing JSON unexpectedly accepted")
	}
	if _, err := ParseHeadOwnerSelection([]byte(`{"schemaVersion":"margo/head-owner-selection/v1","owner":"margo","primitive":"socialMetadataTags","goshtosoCommit":"module:v0.1.2","goshtosoTree":"module-cache:v0.1.2","apiSourcePath":"components/head/component.go","apiSourceSHA256":"833562eafa47d917587c21e300d28c45006b855a569266b96041123ca870b3fb","extra":true}`)); err == nil {
		t.Fatal("unknown selection field unexpectedly accepted")
	}
}

func TestStandaloneIsOfflineDeterministicAndScoped(t *testing.T) {
	result := mustRenderSource(t, "# Standalone\n\ncontent")
	component, err := RenderStandalone(result, WithPageTitle("Standalone"))
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if err := component.Render(context.Background(), &got); err != nil {
		t.Fatal(err)
	}
	html := got.String()
	for _, want := range []string{
		"<!doctype html>",
		`data-theme="modern"`,
		`data-color-mode="light"`,
		`data-margo-render-instance="ri-00000000"`,
		`class="goshtoso-document"`,
		`data-margo-stylesheet="goshtoso"`,
		`data-margo-stylesheet="document"`,
		"Standalone",
		"--document-font-body",
	} {
		if !bytes.Contains(got.Bytes(), []byte(want)) {
			t.Fatalf("standalone HTML missing %q:\n%s", want, html)
		}
	}
	for _, forbidden := range []string{`<link `, `<script src=`, `url(http://`, `url(https://`} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("offline standalone unexpectedly contains %q:\n%s", forbidden, html)
		}
	}
}

func TestStandaloneDarkColorModeIsExplicitAndPrintSafe(t *testing.T) {
	result := mustRenderSource(t, "# Dark PDF\n\ncontent")
	component, err := RenderStandalone(result, WithStandaloneColorMode(ColorModeDark))
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, component)
	for _, want := range []string{`class="dark"`, `data-color-mode="dark"`} {
		if !strings.Contains(markup, want) {
			t.Errorf("dark standalone missing %q", want)
		}
	}
	asset, err := EmbeddedAsset("standalone.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"html.dark body",
		"background: var(--color-surface-dark);",
		"print-color-adjust: exact;",
	} {
		if !strings.Contains(string(asset.Content), want) {
			t.Errorf("dark print stylesheet missing %q", want)
		}
	}
	documentCSS, err := EmbeddedAsset("document.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		".goshtoso-document:is(.dark *) .margo-mermaid__source",
		"background: var(--color-surface-dark-alt);",
		"color: var(--color-on-surface-dark);",
		"border-color: var(--color-outline-dark);",
		".goshtoso-document:is(.dark *) details",
		".goshtoso-document:is(.dark *) :where(summary, dt)",
	} {
		if !strings.Contains(string(documentCSS.Content), want) {
			t.Errorf("dark Mermaid source stylesheet missing %q", want)
		}
	}
	if _, err := RenderStandalone(result, WithStandaloneColorMode(ColorMode("sepia"))); err == nil {
		t.Fatal("unsupported standalone color mode unexpectedly accepted")
	}
}

func TestStandaloneEmbedsExactGoshtosoCSSBeforeDocumentAdjustments(t *testing.T) {
	result := mustRenderSource(t, "# Styled")
	component, err := RenderStandalone(result)
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if err := component.Render(context.Background(), &got); err != nil {
		t.Fatal(err)
	}

	want, err := goshtosoassets.StylesCSS()
	if err != nil {
		t.Fatal(err)
	}
	html := got.String()
	goshtosoStart := strings.Index(html, `<style data-margo-stylesheet="goshtoso">`)
	documentStart := strings.Index(html, `<style data-margo-stylesheet="document">`)
	if goshtosoStart < 0 || documentStart < 0 || goshtosoStart >= documentStart {
		t.Fatalf("stylesheet order invalid: goshtoso=%d document=%d", goshtosoStart, documentStart)
	}
	prefix := `<style data-margo-stylesheet="goshtoso">`
	cssStart := goshtosoStart + len(prefix)
	cssEnd := strings.Index(html[cssStart:], `</style>`)
	if cssEnd < 0 {
		t.Fatal("Goshtoso stylesheet has no closing style tag")
	}
	if gotCSS := html[cssStart : cssStart+cssEnd]; gotCSS != string(want) {
		t.Fatalf("embedded Goshtoso CSS differs: got %d bytes, want %d", len(gotCSS), len(want))
	}
}

func TestStandalonePrintBackdropUsesPageCenter(t *testing.T) {
	asset, err := EmbeddedAsset("standalone.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(asset.Content)
	printStart := strings.Index(css, "@media print {")
	if printStart < 0 {
		t.Fatal("standalone stylesheet has no print media block")
	}
	printCSS := css[printStart:]
	for _, want := range []string{
		".goshtoso-document__backdrop {",
		"inset-block-start: 50%;",
		"inset-inline-start: 50%;",
		"inset-inline-end: auto;",
		"transform: translate(-50%, -50%);",
		"transform-origin: center;",
	} {
		if !strings.Contains(printCSS, want) {
			t.Errorf("print backdrop centering missing %q", want)
		}
	}
}

func TestStandaloneTOCPrintLayoutIsAdaptiveAndFragmentable(t *testing.T) {
	asset, err := EmbeddedAsset("standalone.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(asset.Content)
	tocStart := strings.Index(css, ".goshtoso-document__toc {")
	if tocStart < 0 {
		t.Fatal("standalone stylesheet has no table-of-contents block")
	}
	tocEnd := strings.Index(css[tocStart:], "\n  }\n\n  .goshtoso-document__toc-title")
	if tocEnd < 0 {
		t.Fatal("table-of-contents block has no stable end")
	}
	tocCSS := css[tocStart : tocStart+tocEnd]
	for _, want := range []string{
		"background: transparent;",
		"break-after: page;",
		"break-inside: auto;",
		"overflow: visible;",
	} {
		if !strings.Contains(tocCSS, want) {
			t.Errorf("table-of-contents shell missing %q", want)
		}
	}
	linkStart := strings.Index(css, ".goshtoso-document__toc a {")
	if linkStart < 0 || !strings.Contains(css[linkStart:], "overflow-wrap: anywhere;") {
		t.Fatal("table-of-contents links must wrap long labels")
	}
	printStart := strings.Index(css, "@media print {")
	if printStart < 0 {
		t.Fatal("standalone stylesheet has no print media block")
	}
	printCSS := css[printStart:]
	printTOCStart := strings.Index(printCSS, ".goshtoso-document__toc ol {")
	if printTOCStart < 0 {
		t.Fatal("print stylesheet has no table-of-contents list block")
	}
	printTOCEnd := strings.Index(printCSS[printTOCStart:], "\n  }")
	if printTOCEnd < 0 {
		t.Fatal("print table-of-contents list block has no stable end")
	}
	printTOCCSS := printCSS[printTOCStart : printTOCStart+printTOCEnd]
	for _, want := range []string{
		"columns: auto 12rem;",
		"column-fill: balance;",
		"column-gap: 8mm;",
	} {
		if !strings.Contains(printTOCCSS, want) {
			t.Errorf("adaptive print table-of-contents layout missing %q", want)
		}
	}
}

func TestStandalonePrintBlocksAvoidInternalFragmentation(t *testing.T) {
	asset, err := EmbeddedAsset("document.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(asset.Content)
	printStart := strings.Index(css, "@media print {")
	if printStart < 0 {
		t.Fatal("document stylesheet has no print media block")
	}
	printCSS := css[printStart:]
	for _, want := range []string{
		".goshtoso-document > .margo-document :where(h1, h2, h3, h4, h5, h6)",
		"page-break-after: avoid;",
		"break-after: avoid-page;",
		".goshtoso-document > .margo-document :where(ul, ol, blockquote, dl, details, figure, table, img, pre)",
		"page-break-inside: avoid;",
		".goshtoso-document > .margo-document [data-table-client-sort=\"true\"]",
		".goshtoso-document > .margo-document [data-code-block]",
		".goshtoso-document > .margo-document div:has(> .codeblock)",
		".goshtoso-document > .margo-document .margo-mermaid",
		"break-inside: avoid-page;",
	} {
		if !strings.Contains(printCSS, want) {
			t.Errorf("print block-fragmentation contract missing %q", want)
		}
	}
}

func TestStandaloneThemeOverrideChangesDocumentAttribute(t *testing.T) {
	result := mustRenderSource(t, "# Minimal")
	component, err := RenderStandalone(result, WithStandaloneTheme(ThemeMinimal))
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if err := component.Render(context.Background(), &got); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.String(), `data-theme="minimal"`) {
		t.Fatalf("minimal theme attribute missing: %s", got.String())
	}
}

func TestStandaloneComposesTOCAndCompleteBrandFurniture(t *testing.T) {
	result := mustRenderSource(t, "# Benchmark\n\n## First section\n\n### Detail\n\n## Second section\n")
	logo, err := EmbeddedAsset("logo.svg")
	if err != nil {
		t.Fatal(err)
	}
	component, err := RenderStandalone(
		result,
		WithTableOfContents(),
		WithBrand(Brand{
			Header:    templ.Raw(`<span>Optimistic benchmark</span>`),
			Footer:    templ.Raw(`<span>Generated by Margo</span>`),
			Logo:      logo,
			LogoAlt:   "Margo",
			Backdrop:  logo,
			Watermark: "OPTIMISTIC",
			Stamps:    []string{"v0.0.1", "human review"},
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, component)
	for _, want := range []string{
		`<nav class="goshtoso-document__toc" aria-label="Table of contents">`,
		`href="#first-section"`,
		`href="#detail"`,
		`href="#second-section"`,
		`class="goshtoso-document__header"`,
		`class="goshtoso-document__logo"`,
		`src="data:image/svg+xml;base64,`,
		`alt="Margo"`,
		`class="goshtoso-document__backdrop"`,
		`aria-hidden="true"`,
		`class="goshtoso-document__stamps"`,
		`<span class="goshtoso-document__stamp">v0.0.1</span>`,
		`<span class="goshtoso-document__stamp">human review</span>`,
		`class="goshtoso-document__watermark"`,
		`class="goshtoso-document__footer"`,
		`data-margo-stylesheet="shell"`,
	} {
		if !strings.Contains(markup, want) {
			t.Errorf("standalone furniture missing %q", want)
		}
	}
	if strings.Index(markup, `class="goshtoso-document__toc"`) > strings.Index(markup, `<article class="margo-document">`) {
		t.Error("table of contents must precede article content")
	}
}
