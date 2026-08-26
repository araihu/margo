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
	if got := withCodeHTML.Requirements().List(); len(got) == 0 || got[len(got)-1].ID != "margo.code-copy" {
		t.Fatalf("code requirements = %#v, want margo.code-copy last", got)
	}

	withoutCode := mustRenderSource(t, "# No code\n\nPlain text.\n")
	withoutCodeHTML, err := RenderHTML(withoutCode)
	if err != nil {
		t.Fatal(err)
	}
	for _, requirement := range withoutCodeHTML.Requirements().List() {
		if requirement.ID == "margo.code-copy" {
			t.Fatalf("plain document unexpectedly projected code-copy requirement: %#v", withoutCodeHTML.Requirements().List())
		}
	}
	disabled := mustRenderSource(t, "# Disabled\n\n```sh:copy_disabled\necho hello\n```\n")
	disabledHTML, err := RenderHTML(disabled)
	if err != nil {
		t.Fatal(err)
	}
	for _, requirement := range disabledHTML.Requirements().List() {
		if requirement.ID == "margo.code-copy" {
			t.Fatalf("copy-disabled document unexpectedly projected code-copy requirement: %#v", disabledHTML.Requirements().List())
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
