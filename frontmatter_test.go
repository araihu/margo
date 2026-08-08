package margo

import (
	"context"
	"strings"
	"testing"
)

func TestFrontmatterRejectsUnknownGoshtosoField(t *testing.T) {
	_, err := New().Compile(context.Background(), Source{Name: "x.md", Content: []byte("---\ngoshtoso:\n  mystery: true\n---\n# X")})
	if got := diagnosticCode(err); got != "frontmatter.goshtoso.unknown_field" {
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
	parsed, ok := doc.parsed.(normalizedMarkdown)
	if !ok || parsed.frontmatter.values["owner"] != "docs" {
		t.Fatalf("generic metadata was not retained: %#v", parsed)
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
