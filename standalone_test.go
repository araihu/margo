package margo

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/a-h/templ"
	goshtosoassets "github.com/araihu/goshtoso/assets"
)

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

func TestStandaloneCodeCopyRuntimeIsInline(t *testing.T) {
	result := mustRenderSource(t, "# Standalone\n\n```sh\necho hello\n```\n")
	component, err := RenderStandalone(result)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, component)
	for _, want := range []string{
		`data-margo-requirement="margo.code-copy"`,
		`data-margo-code-copy-button`,
		`data-margo-code-copy-label`,
		`aria-live="polite"`,
		`navigator.clipboard`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("standalone code-copy output missing %q: %s", want, markup)
		}
	}
	if strings.Contains(markup, `src="/margo-assets/code-copy.js"`) {
		t.Fatalf("standalone code-copy runtime was externalized: %s", markup)
	}
}

func TestStandalonePreservesDerivedMetadataAndMainLandmark(t *testing.T) {
	result := mustRenderSource(t, "---\nlanguage: pt-BR\ndescription: Resumo para distribuição.\n---\n\n# Relatório operacional\n\nConteúdo.\n")
	component, err := RenderStandalone(result)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, component)
	for _, want := range []string{
		`<html lang="pt-BR"`,
		"<title>Relatório operacional</title>",
		`<meta name="description" content="Resumo para distribuição."`,
		`<a class="margo-skip-link" href="#margo-document-content"`,
		`<main id="margo-document-content"`,
	} {
		if !strings.Contains(markup, want) {
			t.Errorf("standalone semantic shell missing %q: %s", want, markup)
		}
	}
	if strings.Count(markup, "<main") != 1 {
		t.Fatalf("standalone main landmark count = %d", strings.Count(markup, "<main"))
	}
}

func TestStandaloneAllowsExplicitPageLanguage(t *testing.T) {
	result := mustRenderSource(t, "# Guia\n")
	component, err := RenderStandalone(result, WithPageLanguage("pt-BR"))
	if err != nil {
		t.Fatal(err)
	}
	if markup := renderComponent(t, component); !strings.Contains(markup, `<html lang="pt-BR"`) {
		t.Fatalf("standalone language override missing: %s", markup)
	}
	if _, err := RenderStandalone(result, WithPageLanguage("pt_BR")); diagnosticCode(err) != "html.metadata_invalid" {
		t.Fatalf("invalid language error = %v", err)
	}
}

func TestPageLanguageOptionIsSafeForConcurrentReuse(t *testing.T) {
	option := WithPageLanguage(" pt-BR ")
	const workers = 64
	var wait sync.WaitGroup
	errors := make(chan error, workers)
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			config := standaloneConfig{}
			if err := option(&config); err != nil {
				errors <- err
				return
			}
			if config.lang != "pt-BR" {
				errors <- fmt.Errorf("language = %q", config.lang)
			}
		}()
	}
	wait.Wait()
	close(errors)
	for err := range errors {
		t.Fatal(err)
	}
}

func TestStandaloneMermaidEmbedsOfflineBrowserRuntime(t *testing.T) {
	if _, err := mermaidBrowserCapabilities(); err != nil {
		if diagnostic, ok := err.(*DiagnosticError); ok {
			t.Fatalf("browser capabilities: %+v", diagnostic.Diagnostics)
		}
		t.Fatal(err)
	}
	compiler := New()
	document, err := compiler.Compile(context.Background(), Source{Name: "diagram.md", Content: []byte("```mermaid\ngraph TD; A-->B\n```\n")})
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	component, err := RenderStandalone(rendered)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, component)
	for _, marker := range []string{`data-margo-requirement="margo.mermaid.runtime"`, `data-margo-requirement="margo.mermaid.execute"`, "materializeTreeViewIcons", "treeView: {showIcons: true}", "theme: \"base\"", "themeVariables", "MutationObserver", "margoMermaidThemeObserver", "margoRunMermaid", "margoRuntimeReady"} {
		if !strings.Contains(markup, marker) {
			t.Fatalf("standalone Mermaid HTML missing %q", marker)
		}
	}
}

func TestStandaloneRelocatesProvenanceMarkedChartScripts(t *testing.T) {
	requirements, err := mergeHTMLRequirements([]HTMLRequirement{{
		ID: "goshtoso-charts.runtime", Kind: HTMLScript,
		Inline: AssetRef{Path: "charts-runtime.js", MediaType: "application/javascript", Content: []byte("window.echarts = {};")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	result := &RenderResult{
		content:          templ.Raw(`<article class="margo-document"><h1>Chart</h1><figure class="goshtoso-charts-interactive"><script data-margo-extension-script="charts">window.chartReady = true;</script></figure></article>`),
		htmlRequirements: requirements,
	}
	component, err := RenderStandalone(result)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, component)
	for _, want := range []string{
		`data-margo-chart-script-slot="0"`,
		`data-margo-requirement="margo.charts.inline.0"`,
		`window.chartReady = true;`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("standalone chart runtime missing %q: %s", want, markup)
		}
	}
	if strings.Contains(markup, `data-margo-extension-script="charts"`) {
		t.Fatalf("provenance script remained in article: %s", markup)
	}
}

func TestChartScriptBootstrapEscapesJavaScriptSource(t *testing.T) {
	source := "window.value = \"double\"; // single ' quote \\\\ newline\n"
	markup := string(chartScriptBootstrap(3, source))
	if !strings.Contains(markup, `data-margo-chart-script-slot="3"`) {
		t.Fatalf("slot ordinal missing: %s", markup)
	}
	if want := `script.textContent = ` + strconv.Quote(source) + `;`; !strings.Contains(markup, want) {
		t.Fatalf("escaped chart source missing %q: %s", want, markup)
	}
}

func TestStandaloneUsesEditorialFragmentExactlyOnce(t *testing.T) {
	result := mustRenderSource(t, "# Shared\n\n| Name | Count |\n|---|---:|\n| Item 2 | 2 |\n")
	editorial, err := RenderHTML(result)
	if err != nil {
		t.Fatal(err)
	}
	fragment := renderComponent(t, editorial.Fragment())
	standalone, err := RenderStandalone(result, WithStandaloneTheme(ThemeMinimal))
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, standalone)
	if strings.Count(markup, fragment) != 1 {
		t.Fatalf("fragment count != 1: %s", markup)
	}
	for _, want := range []string{
		`data-margo-html-fingerprint="` + editorial.Fingerprint().String() + `"`,
		`data-margo-requirement="goshtoso.styles"`,
		`data-margo-requirement="margo.document.styles"`,
		`data-margo-requirement="margo.table-sort"`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("shared standalone path missing %q: %s", want, markup)
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
		"--margo-page-background: var(--color-surface-dark);",
		"--margo-print-page-background: var(--margo-page-background);",
		"--margo-print-chrome-background: var(--margo-print-page-background);",
		"--margo-print-chrome-foreground: var(--color-on-surface-dark);",
		"--margo-print-chrome-outline: var(--color-outline-dark);",
		"background: var(--margo-print-page-background);",
		"color: var(--margo-print-chrome-foreground);",
		"border-block-end-color: var(--margo-print-chrome-outline);",
		"border-block-start-color: var(--margo-print-chrome-outline);",
		"break-inside: avoid;",
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
		".margo-document:is(.dark *) .margo-mermaid__source",
		"background: var(--color-surface-dark-alt);",
		"color: var(--color-on-surface-dark);",
		"border-color: var(--color-outline-dark);",
		".margo-document:is(.dark *) details",
		".margo-document:is(.dark *) :where(summary, dt)",
	} {
		if !strings.Contains(string(documentCSS.Content), want) {
			t.Errorf("dark Mermaid source stylesheet missing %q", want)
		}
	}
	if _, err := RenderStandalone(result, WithStandaloneColorMode(ColorMode("sepia"))); err == nil {
		t.Fatal("unsupported standalone color mode unexpectedly accepted")
	}
}

func TestPrintWatermarkUsesReservedFlowSpaceInsteadOfOverlay(t *testing.T) {
	asset, err := EmbeddedAsset("standalone.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(asset.Content)
	printIndex := strings.Index(css, "@media print")
	if printIndex < 0 {
		t.Fatal("standalone stylesheet has no print projection")
	}
	printCSS := css[printIndex:]
	start := strings.Index(printCSS, ".goshtoso-document__watermark {")
	if start < 0 {
		t.Fatal("print projection has no watermark rule")
	}
	end := strings.Index(printCSS[start:], "}")
	if end < 0 {
		t.Fatal("print watermark rule is malformed")
	}
	rule := printCSS[start : start+end]
	for _, want := range []string{"display: block", "position: static", "margin-block-start:"} {
		if !strings.Contains(rule, want) {
			t.Fatalf("print watermark does not reserve flow space with %q: %s", want, rule)
		}
	}
	if strings.Contains(rule, "position: fixed") {
		t.Fatalf("print watermark can overlap paginated content: %s", rule)
	}
}

func TestStandaloneLeavesPageGeometryToThePDFEngine(t *testing.T) {
	asset, err := EmbeddedAsset("standalone.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(asset.Content)
	for _, forbidden := range []string{"size: A4", "margin: 24mm 22mm 26mm"} {
		if strings.Contains(css, forbidden) {
			t.Errorf("standalone stylesheet owns page geometry %q", forbidden)
		}
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
	goshtosoStart := strings.Index(html, `data-margo-stylesheet="goshtoso"`)
	documentStart := strings.Index(html, `data-margo-stylesheet="document"`)
	if goshtosoStart < 0 || documentStart < 0 || goshtosoStart >= documentStart {
		t.Fatalf("stylesheet order invalid: goshtoso=%d document=%d", goshtosoStart, documentStart)
	}
	openingTagEnd := strings.Index(html[goshtosoStart:], `>`)
	if openingTagEnd < 0 {
		t.Fatal("Goshtoso stylesheet has no opening tag end")
	}
	cssStart := goshtosoStart + openingTagEnd + 1
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
	for _, want := range []string{
		".margo-mermaid__source",
		"details.open = true",
		"window.margoRestorePrintState",
		"details.open = originalDetailsState.get(details)",
	} {
		if !strings.Contains(standalonePrintPreparationScript, want) {
			t.Errorf("print disclosure state contract missing %q", want)
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
		"border: 0;",
		"border-radius: 0;",
		"background: var(--margo-page-background);",
		"break-after: page;",
		"break-inside: auto;",
		"overflow: visible;",
	} {
		if !strings.Contains(tocCSS, want) {
			t.Errorf("table-of-contents shell missing %q", want)
		}
	}
	linkStart := strings.Index(css, ".goshtoso-document__toc a {")
	if linkStart < 0 {
		t.Fatal("standalone stylesheet has no table-of-contents link block")
	}
	linkEnd := strings.Index(css[linkStart:], "\n  }")
	if linkEnd < 0 {
		t.Fatal("table-of-contents link block has no stable end")
	}
	linkCSS := css[linkStart : linkStart+linkEnd]
	for _, want := range []string{"color: var(--color-primary);", "overflow-wrap: anywhere;"} {
		if !strings.Contains(linkCSS, want) {
			t.Errorf("table-of-contents links missing %q", want)
		}
	}
	if !strings.Contains(css, "html.dark .goshtoso-document__toc a {") ||
		!strings.Contains(css, "color: var(--color-primary-dark);") {
		t.Fatal("dark table-of-contents links must use the dark primary color")
	}
	printStart := strings.Index(css, "@media print {")
	if printStart < 0 {
		t.Fatal("standalone stylesheet has no print media block")
	}
	printCSS := css[printStart:]
	printTOCStart := strings.Index(printCSS, ".goshtoso-document__toc-disclosure > ol {")
	if printTOCStart < 0 {
		t.Fatal("print stylesheet has no table-of-contents list block")
	}
	printTOCEnd := strings.Index(printCSS[printTOCStart:], "\n  }")
	if printTOCEnd < 0 {
		t.Fatal("print table-of-contents list block has no stable end")
	}
	printTOCCSS := printCSS[printTOCStart : printTOCStart+printTOCEnd]
	for _, want := range []string{
		"column-gap: 8mm;",
	} {
		if !strings.Contains(printTOCCSS, want) {
			t.Errorf("adaptive print table-of-contents layout missing %q", want)
		}
	}
	for _, want := range []string{
		"columns: 1;",
		"column-fill: auto;",
		"list-style: none;",
		"columns: auto 12rem;",
		"column-fill: balance;",
		"page-break-after: page;",
		"break-after: page;",
		"page-break-inside: avoid;",
		"break-inside: avoid-page;",
		`.goshtoso-document__toc[data-margo-toc-columns="2"] .goshtoso-document__toc-disclosure > ol`,
		`.goshtoso-document__toc details`,
		"background: transparent;",
	} {
		if !strings.Contains(printCSS, want) {
			t.Errorf("adaptive print table-of-contents flow missing %q", want)
		}
	}
}

func TestStandalonePrintBlocksUseNativeFragmentation(t *testing.T) {
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
		".margo-document :where(h1, h2, h3, h4, h5, h6)",
		"page-break-after: avoid;",
		"break-after: avoid-page;",
		"widows: 3;",
		"orphans: 3;",
		"break-inside: auto;",
		"display: table-header-group;",
	} {
		if !strings.Contains(printCSS, want) {
			t.Errorf("print block-fragmentation contract missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"window.innerHeight",
		"getBoundingClientRect",
		"data-margo-print-break-before",
		"data-margo-print-oversized",
	} {
		if strings.Contains(standalonePrintPreparationScript, forbidden) || strings.Contains(printCSS, forbidden) {
			t.Errorf("predictive pagination contract still contains %q", forbidden)
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
	result := mustRenderSource(t, "# Benchmark\n\nPurpose before navigation.\n\n## First section\n\n### Detail\n\n## Second section\n")
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
			Stamps:    []string{"v0.0.1", "review required"},
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, component)
	for _, want := range []string{
		`<nav id="margo-table-of-contents" class="goshtoso-document__toc" aria-label="Table of contents">`,
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
		`<span class="goshtoso-document__stamp">review required</span>`,
		`class="goshtoso-document__watermark"`,
		`class="goshtoso-document__footer"`,
		`data-margo-stylesheet="shell"`,
		`<a class="goshtoso-document__back-to-contents" href="#margo-table-of-contents">Back to contents</a>`,
	} {
		if !strings.Contains(markup, want) {
			t.Errorf("standalone furniture missing %q", want)
		}
	}
	titleIndex := strings.Index(markup, `<h1 id="benchmark">Benchmark</h1>`)
	leadIndex := strings.Index(markup, `<p class="margo-document__lead">Purpose before navigation.</p>`)
	tocIndex := strings.Index(markup, `class="goshtoso-document__toc"`)
	sectionIndex := strings.Index(markup, `<h2 id="first-section">First section</h2>`)
	if titleIndex < 0 || leadIndex < 0 || tocIndex < 0 || sectionIndex < 0 {
		t.Fatalf("standalone entry markers missing: title=%d lead=%d toc=%d section=%d", titleIndex, leadIndex, tocIndex, sectionIndex)
	}
	if !(titleIndex < leadIndex && leadIndex < tocIndex && tocIndex < sectionIndex) {
		t.Fatalf("standalone entry order = title:%d lead:%d toc:%d section:%d; want title, lead, contents, section", titleIndex, leadIndex, tocIndex, sectionIndex)
	}
	if backIndex := strings.Index(markup, `class="goshtoso-document__back-to-contents"`); backIndex <= sectionIndex {
		t.Fatalf("return-to-contents path does not follow document sections: back=%d section=%d", backIndex, sectionIndex)
	}
}

func TestStandaloneTOCStagesTopLevelSectionsBeforeDeeperHeadings(t *testing.T) {
	result := mustRenderSource(t, "# Benchmark\n\nPurpose.\n\n## First section\n\n### First detail\n\n#### First depth\n\n## Second section\n")
	component, err := RenderStandalone(result, WithTableOfContents())
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, component)
	for _, want := range []string{
		`<details class="goshtoso-document__toc-disclosure" open="">`,
		`<li data-level="2"><a href="#first-section">First section</a><details class="goshtoso-document__toc-section-disclosure" open="">`,
		`<summary>More in First section</summary>`,
		`<li data-level="3"><a href="#first-detail">First detail</a></li>`,
		`<li data-level="4"><a href="#first-depth">First depth</a></li>`,
		`<li data-level="2"><a href="#second-section">Second section</a></li>`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("staged contents missing %q:\n%s", want, markup)
		}
	}
}
