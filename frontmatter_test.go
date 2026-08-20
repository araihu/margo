package margo

import (
	"context"
	"strings"
	"testing"
)

func TestFrontmatterRejectsLegacyGoshtosoNamespace(t *testing.T) {
	_, err := New().Compile(context.Background(), Source{Name: "x.md", Content: []byte("---\ngoshtoso:\n  mystery: true\n---\n# X")})
	if got := diagnosticCode(err); got != "frontmatter.goshtoso_removed" {
		t.Fatalf("diagnostic code = %q, err = %v", got, err)
	}
}

func TestFrontmatterPreservesGenericMetadata(t *testing.T) {
	doc, err := New().Compile(context.Background(), Source{Name: "x.md", Content: []byte("---\ntitle: Quarterly review\nowner: docs\n---\n# X")})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if got := doc.Metadata().Title; got != "Quarterly review" {
		t.Fatalf("title = %q", got)
	}
	metadata := doc.Metadata()
	if metadata.Additional["owner"] != "docs" {
		t.Fatalf("public additional metadata = %#v", metadata.Additional)
	}
	metadata.Additional["owner"] = "mutated"
	if doc.Metadata().Additional["owner"] != "docs" {
		t.Fatal("additional metadata was not defensively copied")
	}
	parsed, ok := doc.parsed.(normalizedMarkdown)
	if !ok || parsed.frontmatter.values["owner"] != "docs" {
		t.Fatalf("generic metadata was not retained: %#v", parsed)
	}
}

func TestFrontmatterAcceptsClosedMargoPagePreferences(t *testing.T) {
	doc, err := New().Compile(context.Background(), Source{Name: "page.md", Content: []byte("---\nmargo:\n  page:\n    size: Letter\n    orientation: landscape\n    imageOverflow: allow\n    margins:\n      top: 12.5\n      right: 0\n      left: 18\n---\n# Page")})
	if err != nil {
		t.Fatal(err)
	}
	page := doc.Metadata().Margo.Page
	if page == nil || page.Size != "Letter" || page.Orientation != "landscape" || page.ImageOverflow != "allow" {
		t.Fatalf("page preference = %#v", page)
	}
	if page.Margins == nil || page.Margins.Top == nil || *page.Margins.Top != 12.5 ||
		page.Margins.Right == nil || *page.Margins.Right != 0 || page.Margins.Bottom != nil ||
		page.Margins.Left == nil || *page.Margins.Left != 18 {
		t.Fatalf("margin preference = %#v", page.Margins)
	}
	*page.Margins.Left = 999
	if got := *doc.Metadata().Margo.Page.Margins.Left; got != 18 {
		t.Fatalf("margin preference was not defensively copied: %v", got)
	}
}

func TestFrontmatterRejectsUnknownMargoFieldAtSourcePosition(t *testing.T) {
	_, err := New().Compile(context.Background(), Source{Name: "typo.md", Content: []byte("---\nmargo:\n  mystery: true\n---\n# Typo")})
	diagnostic := unwrapDiagnostic(err)
	if diagnostic == nil || diagnostic.Diagnostics[0].Code != "frontmatter.schema_invalid" || diagnostic.Diagnostics[0].Line != 3 || diagnostic.Diagnostics[0].Pointer != "/margo" {
		t.Fatalf("diagnostic = %#v, error = %v", diagnostic, err)
	}
}

func TestFrontmatterRejectsRemovedMargoSiteField(t *testing.T) {
	_, err := New().Compile(context.Background(), Source{Name: "typo.md", Content: []byte("---\nmargo:\n  site:\n    mystery: true\n---\n# Typo")})
	diagnostic := unwrapDiagnostic(err)
	if diagnostic == nil || diagnostic.Diagnostics[0].Code != "frontmatter.schema_invalid" || diagnostic.Diagnostics[0].Pointer != "/margo" {
		t.Fatalf("diagnostic = %#v, error = %v", diagnostic, err)
	}
}

func TestFrontmatterRejectsInvalidPageMarginPreferences(t *testing.T) {
	tests := []struct {
		name    string
		content string
		pointer string
	}{
		{
			name:    "negative side",
			content: "---\nmargo:\n  page:\n    margins:\n      top: -1\n---\n# Page",
			pointer: "/margo/page/margins/top",
		},
		{
			name:    "unknown side",
			content: "---\nmargo:\n  page:\n    margins:\n      horizontal: 10\n---\n# Page",
			pointer: "/margo/page/margins",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New().Compile(context.Background(), Source{Name: "page.md", Content: []byte(test.content)})
			diagnostic := unwrapDiagnostic(err)
			if diagnostic == nil || diagnostic.Diagnostics[0].Code != "frontmatter.schema_invalid" || diagnostic.Diagnostics[0].Pointer != test.pointer {
				t.Fatalf("diagnostic = %#v, error = %v", diagnostic, err)
			}
		})
	}
}

func TestFrontmatterRejectsDuplicateYAMLKeys(t *testing.T) {
	_, err := New().Compile(context.Background(), Source{Name: "duplicate.md", Content: []byte("---\ntitle: One\ntitle: Two\n---\n# Duplicate")})
	if got := diagnosticCode(err); got != "frontmatter.limits" {
		t.Fatalf("diagnostic = %q, error = %v", got, err)
	}
}

func TestFrontmatterRequiresClosingDelimiter(t *testing.T) {
	_, err := New().Compile(context.Background(), Source{Name: "x.md", Content: []byte("---\ntitle: x\n# X")})
	if got := diagnosticCode(err); got != "frontmatter.unclosed" {
		t.Fatalf("diagnostic code = %q, err = %v", got, err)
	}
}

func TestYAMLLimits(t *testing.T) {
	content := "---\n"
	for i := 0; i < 34; i++ {
		content += strings.Repeat("  ", i) + "a:\n"
	}
	content += strings.Repeat("  ", 34) + "true\n---\n# X"
	_, err := New().Compile(context.Background(), Source{Name: "limits.md", Content: []byte(content)})
	if got := diagnosticCode(err); got != "frontmatter.limits" {
		t.Fatalf("diagnostic code = %q, err = %v", got, err)
	}
}
