package margo

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func tableMarkdown() string {
	return "| Name | Value |\n| --- | ---: |\n| alpha | 1 |\n| beta | 2 |\n"
}

func TestMarkdownTableUsesClientOnlyGoshtosoTable(t *testing.T) {
	markup := renderComponent(t, mustRenderSource(t, tableMarkdown()).Content())
	if !bytes.Contains([]byte(markup), []byte(`data-table-client-sort`)) {
		t.Fatalf("table markup does not identify client sorting:\n%s", markup)
	}
	if bytes.Contains([]byte(markup), []byte(`hx-get`)) {
		t.Fatalf("table unexpectedly enabled server interaction:\n%s", markup)
	}
}

func TestMarkdownTableDeclaresProgressiveSort(t *testing.T) {
	result := mustRenderSource(t, "| Name | Count |\n|---|---:|\n| Item 10 | 10 |\n| Item 2 | 2 |\n")
	editorial, err := RenderEditorial(result)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, editorial.Fragment())
	if !strings.Contains(markup, `data-margo-table-sort="natural"`) {
		t.Fatalf("missing sort marker: %s", markup)
	}
	if strings.Contains(markup, `<button`) {
		t.Fatalf("server emitted JS-only table control: %s", markup)
	}
	requireRequirementIDs(t, editorial.Requirements(), "goshtoso.styles", "margo.document.styles", "margo.table-sort")
}

func TestEditorialWithoutTableDeclaresOnlyStyles(t *testing.T) {
	editorial, err := RenderEditorial(mustRenderSource(t, "Body\n"))
	if err != nil {
		t.Fatal(err)
	}
	requireRequirementIDs(t, editorial.Requirements(), "goshtoso.styles", "margo.document.styles")
}

func TestMarkdownTableRejectsServerSortMode(t *testing.T) {
	compiler := New()
	document, err := compiler.Compile(context.Background(), Source{Name: "table.md", Content: []byte(tableMarkdown())})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := compiler.Render(context.Background(), document, WithTableSort(TableSortMode("server"))); err == nil {
		t.Fatal("server table sorting unexpectedly accepted")
	}
}

func TestMarkdownTableFlattensInlineCodeWithoutMarkdownDelimiters(t *testing.T) {
	markup := renderComponent(t, mustRenderSource(t, "| Vector | Diagnostic |\n| --- | --- |\n| `script` | `mermaid.svg_element_forbidden` |\n").Content())
	if !bytes.Contains([]byte(markup), []byte(`>script<`)) || !bytes.Contains([]byte(markup), []byte(`>mermaid.svg_element_forbidden<`)) {
		t.Fatalf("table did not preserve inline-code text:\n%s", markup)
	}
	if bytes.Contains([]byte(markup), []byte("`script")) || bytes.Contains([]byte(markup), []byte("`mermaid.svg_element_forbidden")) {
		t.Fatalf("table leaked Markdown code delimiters:\n%s", markup)
	}
}
