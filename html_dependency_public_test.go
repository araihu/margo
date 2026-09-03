package margo

import (
	"context"
	"strings"
	"testing"
)

func TestMergeAndRenderHTMLDependenciesDeduplicatesInOrder(t *testing.T) {
	rendered := mustRenderSource(t, "| A | B |\n|---|---|\n| 1 | 2 |\n")
	htmlResult, err := RenderHTML(rendered)
	if err != nil {
		t.Fatal(err)
	}
	merged, err := MergeHTMLRequirements(htmlResult.Requirements(), htmlResult.Requirements())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(merged.List()), len(htmlResult.Requirements().List()); got != want {
		t.Fatalf("requirements = %d want %d", got, want)
	}
	component, err := RenderHTMLDependencies(merged, HTMLDependenciesInline)
	if err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	if err := component.Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	markup := output.String()
	styles := strings.Index(markup, `data-margo-requirement="margo.document.styles"`)
	tableSort := strings.Index(markup, `data-margo-requirement="margo.table-sort"`)
	if styles < 0 || tableSort < 0 || styles >= tableSort {
		t.Fatalf("dependency order is invalid: %s", markup)
	}
}

func TestCodeCopyRequirementIsProjectedOnlyForCopyableCode(t *testing.T) {
	withCode := mustRenderSource(t, "# Code\n\n```sh\necho hello\n```\n")
	withCodeHTML, err := RenderHTML(withCode)
	if err != nil {
		t.Fatal(err)
	}
	if got := withCodeHTML.Requirements().List(); len(got) == 0 || got[len(got)-1].ID != "goshtoso.runtime.code-block" {
		t.Fatalf("code requirements = %#v, want goshtoso.runtime.code-block last", got)
	}
	unlabeledCode := mustRenderSource(t, "# Unlabeled code\n\n```\necho hello\n```\n")
	unlabeledCodeHTML, err := RenderHTML(unlabeledCode)
	if err != nil {
		t.Fatal(err)
	}
	if got := unlabeledCodeHTML.Requirements().List(); len(got) == 0 || got[len(got)-1].ID != "goshtoso.runtime.code-block" {
		t.Fatalf("unlabeled code requirements = %#v, want goshtoso.runtime.code-block last", got)
	}

	withoutCode := mustRenderSource(t, "# No code\n\nPlain text.\n")
	withoutCodeHTML, err := RenderHTML(withoutCode)
	if err != nil {
		t.Fatal(err)
	}
	for _, requirement := range withoutCodeHTML.Requirements().List() {
		if requirement.ID == "goshtoso.runtime.code-block" {
			t.Fatalf("plain document unexpectedly projected code-copy requirement: %#v", withoutCodeHTML.Requirements().List())
		}
	}
	disabled := mustRenderSource(t, "# Disabled\n\n```sh:copy_disabled\necho hello\n```\n")
	disabledHTML, err := RenderHTML(disabled)
	if err != nil {
		t.Fatal(err)
	}
	for _, requirement := range disabledHTML.Requirements().List() {
		if requirement.ID == "goshtoso.runtime.code-block" {
			t.Fatalf("copy-disabled document unexpectedly projected code-copy requirement: %#v", disabledHTML.Requirements().List())
		}
	}
	mermaid := mustRenderSource(t, "# Mermaid\n\n```mermaid\nflowchart TD\n  A --> B\n```\n")
	mermaidHTML, err := RenderHTML(mermaid)
	if err != nil {
		t.Fatal(err)
	}
	for _, requirement := range mermaidHTML.Requirements().List() {
		if requirement.ID == "goshtoso.runtime.code-block" {
			t.Fatalf("Mermaid document unexpectedly projected code-copy requirement: %#v", mermaidHTML.Requirements().List())
		}
	}
	extensionCompiler := New(WithExtension(ExtensionRegistration{
		Identity: ExtensionIdentity{Name: "demo-copy-test", Version: "v1"},
		Fences:   []string{"demo-copy-test"},
		Factory: func(RenderContext) (ExtensionSession, error) {
			return &testExtensionSession{}, nil
		},
	}))
	extensionDocument, err := extensionCompiler.Compile(context.Background(), Source{
		Name: "extension.md", Content: []byte("# Extension\n\n```demo-copy-test\npayload\n```\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	extensionResult, err := extensionCompiler.Render(context.Background(), extensionDocument)
	if err != nil {
		t.Fatal(err)
	}
	extensionHTML, err := RenderHTML(extensionResult)
	if err != nil {
		t.Fatal(err)
	}
	for _, requirement := range extensionHTML.Requirements().List() {
		if requirement.ID == "goshtoso.runtime.code-block" {
			t.Fatalf("extension document unexpectedly projected code-copy requirement: %#v", extensionHTML.Requirements().List())
		}
	}
}

func TestRenderHTMLDependenciesRejectsUnknownMode(t *testing.T) {
	rendered := mustRenderSource(t, "# Page\n")
	htmlResult, err := RenderHTML(rendered)
	if err != nil {
		t.Fatal(err)
	}
	component, err := RenderHTMLDependencies(htmlResult.Requirements(), "remote")
	if component != nil || diagnosticCode(err) != "html.dependency_mode_invalid" {
		t.Fatalf("component = %#v error = %v", component, err)
	}
}
