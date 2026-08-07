package margo

import (
	"bytes"
	"os"
	"regexp"
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
	if !bytes.Contains(css, []byte(".goshtoso-document")) {
		t.Fatalf("document stylesheet is not scoped to .goshtoso-document")
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
		[]byte(".goshtoso-document [x-cloak]"),
		[]byte(".goshtoso-document:is(.dark *)"),
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
	rule := regexp.MustCompile(`(?s)\.goshtoso-document \[data-code-block\],\s*\.goshtoso-document div:has\(> \.codeblock\) \{([^}]*)\}`).FindSubmatch(css)
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

func TestDocumentCSSKeepsExpandedMermaidSourceReadableWhenPrinted(t *testing.T) {
	css, err := os.ReadFile("assets/document.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{
		[]byte(".goshtoso-document .margo-mermaid__source {\n    break-inside: avoid;"),
		[]byte(".goshtoso-document .margo-mermaid__source pre"),
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
