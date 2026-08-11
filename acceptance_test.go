package margo

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestPreV1LongFormPortugueseAcceptanceFixture(t *testing.T) {
	source, err := os.ReadFile("testdata/markdown/pre-v1-long-form-pt.md")
	if err != nil {
		t.Fatal(err)
	}
	diagnostics, err := Check(context.Background(), Source{Name: "proposal.md", Content: source})
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("check diagnostics = %+v", diagnostics)
	}
	compiler := New()
	document, err := compiler.Compile(context.Background(), Source{Name: "proposal.md", Content: source})
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := compiler.Render(context.Background(), document, WithRenderTarget(TargetPDF))
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, rendered.Content())
	for _, forbidden := range []string{"markdownlint-disable", "markdownlint-enable"} {
		if strings.Contains(markup, forbidden) {
			t.Fatalf("authoring comment leaked: %q", forbidden)
		}
	}
	for _, required := range []string{"<table", `class="codeblock`, "margo-mermaid", "interface pública v1"} {
		if !strings.Contains(markup, required) {
			t.Fatalf("acceptance rendering missing %q", required)
		}
	}
}
