package margo

import (
	"context"
	"errors"
	"testing"
)

func TestEditorialFrontmatterNormalizesMetadata(t *testing.T) {
	document, err := New().Compile(context.Background(), Source{Name: "post.md", Content: []byte(`---
title: Durable HTML
description: One semantic source.
language: pt-BR
slug: durable-html
authors: ["Arai Hû"]
publishedAt: "2026-08-09T12:00:00-03:00"
modifiedAt: "2026-08-09T15:00:00Z"
tags: [Go, HTML]
---
Body
`)})
	if err != nil {
		t.Fatal(err)
	}
	metadata := document.Metadata()
	if metadata.Language != "pt-BR" || metadata.Slug != "durable-html" {
		t.Fatalf("metadata = %#v", metadata)
	}
	if metadata.PublishedAt != "2026-08-09T12:00:00-03:00" || metadata.ModifiedAt != "2026-08-09T15:00:00Z" {
		t.Fatalf("dates = (%q, %q)", metadata.PublishedAt, metadata.ModifiedAt)
	}
	metadata.Authors[0] = "mutated"
	metadata.Tags[0] = "mutated"
	again := document.Metadata()
	if again.Authors[0] != "Arai Hû" || again.Tags[0] != "Go" {
		t.Fatal("metadata aliases caller")
	}
}

func TestEditorialFrontmatterRejectsInvalidMetadata(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   string
		pointer string
	}{
		{name: "title type", field: "title", value: "[not, text]", pointer: "/title"},
		{name: "authors type", field: "authors", value: "Arai Hû", pointer: "/authors"},
		{name: "authors member", field: "authors", value: "[Arai Hû, 42]", pointer: "/authors/1"},
		{name: "tags type", field: "tags", value: "HTML", pointer: "/tags"},
		{name: "published date", field: "publishedAt", value: `"09/08/2026"`, pointer: "/publishedAt"},
		{name: "modified date", field: "modifiedAt", value: `"tomorrow"`, pointer: "/modifiedAt"},
		{name: "language", field: "language", value: `"pt_BR"`, pointer: "/language"},
		{name: "slug", field: "slug", value: `"Durable HTML"`, pointer: "/slug"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := []byte("---\n" + test.field + ": " + test.value + "\n---\nBody\n")
			_, err := New().Compile(context.Background(), Source{Name: "invalid.md", Content: source})
			if got := diagnosticCode(err); got != "source.metadata_invalid" {
				t.Fatalf("diagnostic code = %q, err = %v", got, err)
			}
			var diagnosticErr *DiagnosticError
			if !errors.As(err, &diagnosticErr) || len(diagnosticErr.Diagnostics) != 1 || diagnosticErr.Diagnostics[0].Pointer != test.pointer {
				t.Fatalf("diagnostic pointer = %#v, want %q", diagnosticErr, test.pointer)
			}
		})
	}
}
