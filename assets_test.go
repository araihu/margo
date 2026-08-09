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

func TestDocumentCSSSpacesGoshtosoTableFromFollowingProse(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	rule := regexp.MustCompile(`(?s)\.margo-document \[data-table-client-sort="true"\] \{([^}]*)\}`).FindSubmatch(css)
	if len(rule) != 2 {
		t.Fatal("document stylesheet is missing the scoped Goshtoso table rhythm rule")
	}
	if want := []byte("margin-block-end: calc(var(--spacing) * 4)"); !bytes.Contains(rule[1], want) {
		t.Fatalf("document stylesheet missing table rhythm rule %q", want)
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
		[]byte("color: var(--color-on-surface-strong)"),
		[]byte("background: color-mix(in oklch, var(--color-surface-alt) 50%, var(--color-outline) 50%)"),
		[]byte("border: 1px solid var(--color-outline)"),
		[]byte("padding-block: calc(var(--spacing) / 2)"),
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

func TestDocumentCSSKeepsExpandedMermaidSourceReadableWhenPrinted(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{
		[]byte(".margo-document .margo-mermaid__source {\n    break-inside: avoid;"),
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
