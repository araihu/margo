package margo

import (
	"bytes"
	"context"
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
