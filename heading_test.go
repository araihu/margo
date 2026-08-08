package margo

import (
	"context"
	"testing"
)

func TestHeadingID(t *testing.T) {
	doc, err := New().Compile(context.Background(), Source{Name: "heading.md", Content: []byte("# Hello World\n\n## Hello World")})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	ids := headingIDsForTest(doc)
	if len(ids) != 2 || ids[0] != "hello-world" || ids[1] != "hello-world-1" {
		t.Fatalf("heading ids = %#v", ids)
	}
}
