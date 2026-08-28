package margo

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestLibraryCSSNeverTargetsHostRoot(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(css, []byte("@layer reset, goshtoso, document, brand, overrides")) {
		t.Fatalf("document stylesheet is missing the locked layer order")
	}
	if regexp.MustCompile(`(^|[,{])\s*(html|body|:root)\b`).Match(css) {
		t.Fatalf("document stylesheet targets a host root")
	}
	if !bytes.Contains(css, []byte(".margo-document")) {
		t.Fatalf("document stylesheet is not scoped to .margo-document")
	}
	withoutStandaloneFurniture := bytes.ReplaceAll(css, []byte(".goshtoso-document__"), []byte(".standalone-furniture__"))
	if bytes.Contains(withoutStandaloneFurniture, []byte(".goshtoso-document")) {
		t.Fatal("editorial rules still depend on the standalone goshtoso-document wrapper")
	}
}

func TestHTMLAssetHandlerOwnsOnlyMargoMount(t *testing.T) {
	handler := HTMLAssetHandler()
	for _, test := range []struct {
		path        string
		contentType string
	}{
		{path: "/margo-assets/document.css", contentType: "text/css"},
		{path: "/margo-assets/table-sort.js", contentType: "application/javascript"},
		{path: "/margo-assets/code-copy.js", contentType: "application/javascript"},
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		if recorder.Code != http.StatusOK || !strings.HasPrefix(recorder.Header().Get("Content-Type"), test.contentType) || recorder.Body.Len() == 0 {
			t.Fatalf("GET %s = status %d, type %q, bytes %d", test.path, recorder.Code, recorder.Header().Get("Content-Type"), recorder.Body.Len())
		}
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assets/document.css", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("editorial handler accepted Goshtoso mount: %d", recorder.Code)
	}

	legacyRecorder := httptest.NewRecorder()
	AssetHandler().ServeHTTP(legacyRecorder, httptest.NewRequest(http.MethodGet, "/assets/table-sort.js", nil))
	if legacyRecorder.Code != http.StatusNotFound {
		t.Fatalf("legacy handler exposed editorial runtime: %d", legacyRecorder.Code)
	}
}

func TestValidateAssetPathAllowsOrdinaryXAndZeroButRejectsNUL(t *testing.T) {
	if err := validateAssetPath("charts-inline-x0.js"); err != nil {
		t.Fatalf("ordinary asset path rejected: %v", err)
	}
	if err := validateAssetPath("charts\x00inline.js"); err == nil {
		t.Fatal("asset path containing NUL was accepted")
	}
}

func TestDocumentCSSRestoresSemanticMarkdownRhythmAfterGoshtosoPreflight(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{
		[]byte("font-size: var(--text-4xl)"),
		[]byte("max-width: 75ch"),
		[]byte("list-style-type: disc"),
		[]byte("color: var(--color-primary)"),
		[]byte("border-inline-start: 1px solid var(--color-outline)"),
		[]byte(".margo-document [x-cloak]"),
		[]byte(".margo-document:is(.dark *)"),
	} {
		if !bytes.Contains(css, want) {
			t.Fatalf("document stylesheet missing semantic rhythm rule %q", want)
		}
	}
}

func TestDocumentCSSSeparatesPublicationDatesAtNarrowWidths(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{
		[]byte(".margo-document .margo-document__publication-dates"),
		[]byte("flex-wrap: wrap"),
		[]byte(".margo-document .margo-document__publication-label"),
		[]byte("overflow-wrap: anywhere"),
	} {
		if !bytes.Contains(css, want) {
			t.Fatalf("document stylesheet missing publication-date rule %q", want)
		}
	}
}

func TestDocumentCSSSpacesConsecutiveGoshtosoCodeBlocks(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	rule := regexp.MustCompile(`(?s)\.margo-document \[data-code-block\],\s*\.margo-document div:has\(> \.codeblock\) \{([^}]*)\}`).FindSubmatch(css)
	if len(rule) != 2 {
		t.Fatal("document stylesheet is missing the scoped Goshtoso code-block rhythm rule")
	}
	for _, want := range [][]byte{
		[]byte("margin-block-start: calc(var(--spacing) * 4)"),
		[]byte("margin-block-end: calc(var(--spacing) * 4)"),
		[]byte("break-inside: avoid"),
	} {
		if !bytes.Contains(rule[1], want) {
			t.Fatalf("document stylesheet missing code-block rhythm rule %q", want)
		}
	}
}

func TestDocumentCSSSpacesGoshtosoTableFromSurroundingProse(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	rule := regexp.MustCompile(`(?s)\.margo-document \[data-table-client-sort="true"\] \{([^}]*)\}`).FindSubmatch(css)
	if len(rule) != 2 {
		t.Fatal("document stylesheet is missing the scoped Goshtoso table rhythm rule")
	}
	for _, want := range [][]byte{
		[]byte("margin-block-start: calc(var(--spacing) * 6)"),
		[]byte("margin-block-end: calc(var(--spacing) * 4)"),
	} {
		if !bytes.Contains(rule[1], want) {
			t.Fatalf("document stylesheet missing table rhythm rule %q", want)
		}
	}
}

func TestDocumentPrintCSSWrapsLongCodeAndUsesLandscapeForWideDiagrams(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{
		[]byte(".margo-document pre {"),
		[]byte("white-space: pre-wrap"),
		[]byte("overflow-wrap: anywhere"),
		[]byte("max-height: 70vh !important"),
		[]byte("aspect-ratio: auto !important"),
		[]byte(`[data-margo-image-overflow="allow"] .margo-document img`),
		[]byte(`.margo-document .margo-mermaid[data-margo-print-layout="landscape"]`),
		[]byte("page: margo-diagram-landscape"),
		[]byte("max-inline-size: none"),
		[]byte(`:has(+ .margo-mermaid[data-margo-print-layout="landscape"])`),
		[]byte(`:has(+ :where(h2, h3, h4, h5, h6) + .margo-mermaid[data-margo-print-layout="landscape"])`),
	} {
		if !bytes.Contains(css, want) {
			t.Fatalf("print stylesheet missing containment rule %q", want)
		}
	}
}

func TestDocumentPrintCSSKeepsPortraitMermaidFigureWithItsCaption(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{
		[]byte(".margo-document .margo-mermaid:not([data-margo-print-layout=\"landscape\"])"),
		[]byte("break-inside: avoid-page"),
		[]byte("max-block-size: 155mm"),
		[]byte("block-size: auto"),
	} {
		if !bytes.Contains(css, want) {
			t.Fatalf("print stylesheet missing portrait Mermaid fitting rule %q", want)
		}
	}
}

func TestDocumentCSSGivesInlineCodeAVisibleThemedBoundary(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	rule := regexp.MustCompile(`(?s)\.margo-document :not\(pre\) > code \{([^}]*)\}`).FindSubmatch(css)
	if len(rule) != 2 {
		t.Fatal("document stylesheet is missing the scoped inline-code rule")
	}
	for _, want := range [][]byte{
		[]byte("display: inline-block"),
		[]byte("box-sizing: border-box"),
		[]byte("max-inline-size: 100%"),
		[]byte("overflow: hidden"),
		[]byte("text-overflow: ellipsis"),
		[]byte("white-space: nowrap"),
		[]byte("vertical-align: bottom"),
		[]byte("line-height: 1.2"),
		[]byte("margin-block: calc(var(--spacing) / 4)"),
		[]byte("margin-inline: calc(var(--spacing) / 4)"),
		[]byte("color: var(--color-on-surface-strong)"),
		[]byte("background: color-mix(in oklch, var(--color-surface-alt) 50%, var(--color-outline) 50%)"),
		[]byte("border: 1px solid var(--color-outline)"),
		[]byte("padding-inline: calc(var(--spacing) / 2)"),
		[]byte("padding-block: calc(var(--spacing) / 4)"),
	} {
		if !bytes.Contains(rule[1], want) {
			t.Fatalf("document stylesheet missing inline-code contrast rule %q", want)
		}
	}
	if !bytes.Contains(css, []byte(".margo-document:is(.dark *) :not(pre) > code {")) ||
		!bytes.Contains(css, []byte("background: color-mix(in oklch, var(--color-surface-dark-alt) 50%, var(--color-outline-dark) 50%)")) ||
		!bytes.Contains(css, []byte("border-color: var(--color-outline-dark)")) {
		t.Fatal("document stylesheet is missing the dark inline-code boundary token")
	}
}

func TestDocumentCSSStylesJSONSchemaAsIndentedTree(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{
		[]byte(".margo-jsonschema__tree-list"),
		[]byte("border-inline-start: 1px solid var(--color-outline)"),
		[]byte(".margo-jsonschema__tree-row"),
		[]byte(".margo-jsonschema__tree-path"),
		[]byte("text-overflow: ellipsis"),
		[]byte("background: transparent"),
		[]byte("font-style: italic"),
		[]byte("color: var(--color-danger)"),
	} {
		if !bytes.Contains(css, want) {
			t.Fatalf("document stylesheet missing JSON Schema tree rule %q", want)
		}
	}
}

func TestDocumentCSSGivesLiveDeckLinkAVisibleSurface(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{
		[]byte(`blockquote:has(a[href*="deck-workspace/slides.html"])`),
		[]byte("border-inline-start: calc(var(--spacing) * 1) solid var(--color-primary)"),
		[]byte("background: color-mix(in oklch, var(--color-surface-alt) 72%, var(--color-outline) 28%)"),
		[]byte("border-radius: 999px"),
		[]byte("text-decoration: none"),
	} {
		if !bytes.Contains(css, want) {
			t.Fatalf("document stylesheet missing live deck surface rule %q", want)
		}
	}
}

func TestDocumentCSSKeepsExpandedMermaidSourceReadableWhenPrinted(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{
		[]byte(".margo-document .margo-mermaid__source {\n    break-inside: auto;"),
		[]byte(".margo-document .margo-mermaid__source pre"),
		[]byte("font-size: var(--text-sm)"),
		[]byte("white-space: pre-wrap"),
		[]byte("overflow-wrap: anywhere"),
	} {
		if !bytes.Contains(css, want) {
			t.Fatalf("document stylesheet missing printed Mermaid source rule %q", want)
		}
	}
}

func TestDocumentCSSConstrainsTrustedEmbedsToDocumentWidth(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{
		[]byte(".margo-document .margo-embed"),
		[]byte("max-inline-size: 100%"),
		[]byte("overflow: hidden"),
		[]byte("overflow-wrap: anywhere"),
	} {
		if !bytes.Contains(css, want) {
			t.Fatalf("document stylesheet missing embed rule %q", want)
		}
	}
}

func TestDocumentStylesKeepNarrowMermaidLabelsReadableWithLocalOverflow(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{
		[]byte(".margo-document .margo-mermaid__canvas"),
		[]byte("overflow-x: auto"),
		[]byte(".margo-document .margo-mermaid__canvas > svg"),
		[]byte("min-inline-size: 40rem"),
		[]byte(".margo-document .margo-mermaid__overflow-cue"),
		[]byte("@media (max-width: 42rem)"),
	} {
		if !bytes.Contains(css, want) {
			t.Fatalf("document CSS has no narrow Mermaid behavior %q", want)
		}
	}
}

func TestDocumentStylesThemeMermaidTreeViewLabels(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{
		[]byte(".margo-document .margo-mermaid .treeView-node-label"),
		[]byte("color: var(--color-on-surface-strong)"),
		[]byte("fill: var(--color-on-surface-strong) !important"),
		[]byte(".margo-document .margo-mermaid .treeView-node-icon"),
		[]byte("color: var(--color-primary) !important"),
		[]byte("stroke: var(--color-outline-strong) !important"),
		[]byte(".margo-document:is(.dark *) .margo-mermaid .treeView-node-label"),
		[]byte(".dark .margo-document .margo-mermaid .treeView-node-label"),
		[]byte("[data-color-mode=\"dark\"] .margo-document .margo-mermaid .treeView-node-label"),
		[]byte("color: var(--color-on-surface-dark-strong)"),
		[]byte("fill: var(--color-on-surface-dark-strong) !important"),
		[]byte(".dark .margo-document .margo-mermaid .treeView-node-icon"),
		[]byte("color: var(--color-primary-dark) !important"),
		[]byte("stroke: var(--color-outline-dark-strong) !important"),
	} {
		if !bytes.Contains(css, want) {
			t.Fatalf("document CSS missing themed TreeView rule %q", want)
		}
	}
}

func TestDocumentStylesKeepDeepHeadingLevelsVisuallyDistinct(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	h5 := regexp.MustCompile(`(?s)\.margo-document h5 \{([^}]*)\}`).FindSubmatch(css)
	h6 := regexp.MustCompile(`(?s)\.margo-document h6 \{([^}]*)\}`).FindSubmatch(css)
	if len(h5) != 2 || len(h6) != 2 {
		t.Fatal("H5 and H6 do not have independent visual roles")
	}
	if bytes.Equal(bytes.TrimSpace(h5[1]), bytes.TrimSpace(h6[1])) {
		t.Fatal("H5 and H6 visual roles are identical")
	}
	if !bytes.Contains(h6[1], []byte("text-transform: uppercase")) {
		t.Fatal("H6 lacks the label treatment that distinguishes it from body text")
	}
}

func TestAssetOverrideRejectsInvalidPathWithoutFallback(t *testing.T) {
	result := mustRenderSource(t, "# override")
	_, err := RenderStandalone(result, WithAssetOverride("document.css", AssetRef{Path: "../outside.css"}))
	if err == nil {
		t.Fatal("invalid asset override unexpectedly succeeded")
	}
	if got := err.Error(); got == "" {
		t.Fatal("invalid asset override returned an empty diagnostic")
	}
}
