package ssg

import (
	"strings"
	"testing"
)

func TestRenderResourcesIsDeterministicAndSeparatedByPlacement(t *testing.T) {
	resources := []ResourceRequirement{
		{Placement: "body-end", Kind: "module", Inline: `console.log("ready")`, Attributes: map[string]string{"data-owner": "margo"}},
		{Placement: "head", Kind: "stylesheet", URL: "/assets/site.css", Integrity: "sha256-abc", Attributes: map[string]string{"media": "screen", "data-owner": "margo"}},
		{Placement: "head", Kind: "preload", URL: "https://cdn.example.test/font.woff2", Attributes: map[string]string{"as": "font", "crossorigin": "anonymous"}},
	}
	first, err := RenderResources(resources)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderResources(resources)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("resource markup is not deterministic: %#v != %#v", first, second)
	}
	if !strings.Contains(first.Head, `<link rel="stylesheet" href="/assets/site.css" integrity="sha256-abc" data-owner="margo" media="screen">`) {
		t.Fatalf("head = %s", first.Head)
	}
	if !strings.Contains(first.Head, `<link rel="preload" href="https://cdn.example.test/font.woff2" as="font" crossorigin="anonymous">`) {
		t.Fatalf("head = %s", first.Head)
	}
	if first.BodyEnd != `<script type="module" data-owner="margo">console.log("ready")</script>` {
		t.Fatalf("body-end = %s", first.BodyEnd)
	}
}

func TestValidateResourcesRejectsUnsafeOrIncompatibleRequirements(t *testing.T) {
	tests := []ResourceRequirement{
		{Placement: "head", Kind: "stylesheet", Inline: "body"},
		{Placement: "body-end", Kind: "preload", URL: "/font.woff2"},
		{Placement: "head", Kind: "script", URL: "javascript:alert(1)"},
		{Placement: "head", Kind: "script", Inline: "</script>"},
		{Placement: "head", Kind: "script", URL: "/app.js", Attributes: map[string]string{"src": "/other.js"}},
	}
	for index, resource := range tests {
		if err := ValidateResources([]ResourceRequirement{resource}); err == nil {
			t.Fatalf("case %d unexpectedly succeeded", index)
		}
	}
}

func TestValidateResourcesRejectsDuplicateAndShellContract(t *testing.T) {
	resource := ResourceRequirement{Placement: "head", Kind: "script", URL: "/app.js"}
	if err := ValidateResources([]ResourceRequirement{resource, resource}); err == nil {
		t.Fatal("duplicate resource unexpectedly succeeded")
	}
	if err := ValidateShellSchema(ShellSchema{Contract: ShellContract, Resources: []ResourceRequirement{resource}}); err != nil {
		t.Fatalf("valid shell schema rejected: %v", err)
	}
	if err := ValidateShellSchema(ShellSchema{Contract: FrameContract}); err == nil {
		t.Fatal("wrong shell contract unexpectedly succeeded")
	}
}
