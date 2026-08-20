package deck

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/araihu/margo"
	"github.com/araihu/margo/charts"
)

func TestRenderNamespacesFeatureIDsAcrossSlidesAndSlots(t *testing.T) {
	source := "| A | B |\n|---|---|\n| 1 | 2 |\n\n```mermaid\ngraph TD; A-->B\n```\n---\n| A | B |\n|---|---|\n| 3 | 4 |\n\n```mermaid\ngraph TD; C-->D\n```\n"
	result, err := Render(context.Background(), margo.New(), RenderInput{Name: "ids.md", Markdown: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	markup := string(result.HTML())
	ids := regexp.MustCompile(`\sid="([^"]+)"`).FindAllStringSubmatch(markup, -1)
	seen := make(map[string]struct{}, len(ids))
	for _, match := range ids {
		if _, exists := seen[match[1]]; exists {
			t.Fatalf("duplicate rendered id %q in %s", match[1], markup)
		}
		seen[match[1]] = struct{}{}
	}
}

func TestRenderKeepsMermaidAndOptionalChartFeatureParity(t *testing.T) {
	source := "# Features\n\n```mermaid\ngraph TD; A-->B\n```\n\n```goshtosochart\nschemaVersion: 1\ntype: bar\ntitle: Revenue\ncategories: [Q1, Q2]\nseries:\n  - name: Actual\n    values: [12, 18]\n```\n"
	result, err := Render(context.Background(), margo.New(margo.WithExtension(charts.Extension(charts.WithExternalizedControlRuntime(true)))), RenderInput{Name: "features.md", Markdown: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	markup := string(result.HTML())
	for _, fragment := range []string{`margo-mermaid`, `data-goshtoso-chart-wrapper`, `<svg`, `<table`} {
		if !strings.Contains(markup, fragment) {
			t.Fatalf("feature markup missing %q", fragment)
		}
	}
}

func TestRenderPreparesInteractiveChartScriptsForDeckShell(t *testing.T) {
	source := "# Interactive\n\n```goshtosochart\nschemaVersion: 1\ntype: line\nrenderer: interactive\ntitle: Coverage\ncategories: [Q1, Q2]\nseries:\n  - name: Actual\n    values: [12, 18]\n```\n"
	compiler := margo.New(margo.WithExtension(charts.Extension(charts.WithExternalizedControlRuntime(true))))
	result, err := Render(context.Background(), compiler, RenderInput{Name: "interactive.md", Markdown: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	markup := string(result.HTML())
	for _, fragment := range []string{
		`data-goshtoso-chart-capability="interactive-raster"`,
		`data-margo-requirement="margo.charts.inline.0"`,
		`data-margo-requirement="goshtoso-charts.controls"`,
	} {
		if !strings.Contains(markup, fragment) {
			t.Fatalf("interactive deck output missing %q", fragment)
		}
	}
	if strings.Contains(markup, `data-margo-extension-script="charts"`) {
		t.Fatal("interactive extension script leaked into the deck fragment")
	}
}

func TestParseRejectsCrossSlideAndCrossSlotReferences(t *testing.T) {
	cases := []struct {
		name   string
		source string
		code   string
	}{
		{
			name:   "cross slide reference",
			source: "# One\n[target]: /one\n---\n# Two\n[link][target]\n",
			code:   "deck.cross_slide_reference",
		},
		{
			name:   "cross slide footnote",
			source: "# One\n[^note]: note\n---\n# Two\n[^note]\n",
			code:   "deck.cross_slide_reference",
		},
		{
			name:   "cross slot reference",
			source: "<!-- _class: columns -->\n<!-- layout: columns -->\n<!-- slot: left -->\n[target]: /one\n<!-- slot: right -->\n[link][target]\n<!-- /layout -->\n",
			code:   "deck.cross_slot_reference",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.name+".md", []byte(tc.source))
			if got := deckDiagnosticCode(err); got != tc.code {
				t.Fatalf("code = %q want %q (err=%v)", got, tc.code, err)
			}
		})
	}
}
