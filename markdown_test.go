package margo

import (
	"context"
	"testing"

	"github.com/yuin/goldmark/ast"
	extensionAst "github.com/yuin/goldmark/extension/ast"
)

func TestMarkdownProfile(t *testing.T) {
	doc, err := New().Compile(context.Background(), Source{Name: "profile.md", Content: []byte("# Heading\n\n| a | b |\n| - | - |\n| 1 | 2 |\n\n~~strike~~ https://example.com\n\n- [ ] task\n\n[^1]: note")})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	parsed, ok := doc.parsed.(normalizedMarkdown)
	if !ok || parsed.root == nil {
		t.Fatal("normalized Goldmark AST missing")
	}
	var sawTable, sawStrike, sawLink, sawTask bool
	_ = ast.Walk(parsed.root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node.Kind() {
		case extensionAst.KindTable:
			sawTable = true
		case extensionAst.KindStrikethrough:
			sawStrike = true
		case ast.KindLink, ast.KindAutoLink:
			sawLink = true
		case extensionAst.KindTaskCheckBox:
			sawTask = true
		}
		return ast.WalkContinue, nil
	})
	if !sawTable || !sawStrike || !sawLink || !sawTask {
		t.Fatalf("GFM nodes missing table=%v strike=%v link=%v task=%v", sawTable, sawStrike, sawLink, sawTask)
	}
}

func TestRawHTMLRemainsASTNode(t *testing.T) {
	doc, err := New().Compile(context.Background(), Source{Name: "raw.md", Content: []byte("<span>raw</span>")})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	parsed := doc.parsed.(normalizedMarkdown)
	found := false
	_ = ast.Walk(parsed.root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if node.Kind() == ast.KindRawHTML {
			found = true
		}
		return ast.WalkContinue, nil
	})
	if !found {
		t.Fatal("raw HTML was not preserved for policy validation")
	}
}
