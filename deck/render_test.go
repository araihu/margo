package deck

import (
	"context"
	"strings"
	"testing"

	"github.com/araihu/margo"
)

func TestRenderProducesAccessibleSections(t *testing.T) {
	compiler := margo.New()
	result, err := Render(context.Background(), compiler, RenderInput{
		Name:     "deck.md",
		Markdown: []byte("# One\n---\n# Two\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	html := string(result.HTML())
	for _, fragment := range []string{
		`<article class="margo-deck"`,
		`id="slide-0001"`,
		`role="region"`,
		`aria-label="Slide 1 of 2"`,
		`aria-label="Slide 2 of 2"`,
		`>One</h1>`,
		`>Two</h1>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("HTML missing %q", fragment)
		}
	}
	if result.SlideCount() != 2 {
		t.Fatalf("count = %d", result.SlideCount())
	}
}

func TestRenderEmitsLogicalStageGeometryAndLocalizedLanguage(t *testing.T) {
	result, err := Render(context.Background(), margo.New(), RenderInput{
		Name:     "deck.md",
		Markdown: []byte("---\nlang: pt-BR\nsize: 4:3\n---\n<!-- paginate: true -->\n# Um\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	html := string(result.HTML())
	for _, fragment := range []string{
		`<html lang="pt-BR"`,
		`data-margo-font-bundle-digest="`,
		`<section id="slide-0001" class="margo-deck__slide" role="region" aria-label="Slide 1 de 1"`,
		`data-margo-slide="0" aria-current="page" lang="pt-BR"`,
		`class="margo-deck-stage"`,
		`data-margo-width="960"`,
		`data-margo-height="720"`,
		`class="margo-deck-controls"`,
		`aria-label="Controles de slides"`,
		`>Anterior</button>`,
		`>Próximo</button>`,
		`class="margo-deck__pagination" aria-hidden="true">1</span>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("HTML missing %q", fragment)
		}
	}
	if got := result.Geometry(); got.Width != 960 || got.Height != 720 {
		t.Fatalf("geometry = %#v", got)
	}
}

func TestRenderFrontmatterDirectivesEnableHostChrome(t *testing.T) {
	result, err := Render(context.Background(), margo.New(), RenderInput{
		Name: "frontmatter-chrome.md",
		Markdown: []byte(`---
title: Chrome probe
paginate: true
header: Atlas review
footer: Internal review
---
# First slide

Visible content.

---

# Second slide

More visible content.
`),
	}, WithConfidentialityBadge("Internal"), WithPaginationIcon(PaginationIconConfig{
		Symbol:     "hi-16-solid-clock",
		Placement:  PaginationIconBefore,
		Decorative: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	markup := string(result.HTML())
	if strings.Count(markup, `data-margo-paginate="true"`) != 2 {
		t.Fatalf("frontmatter pagination markers = %d, want 2: %s", strings.Count(markup, `data-margo-paginate="true"`), markup)
	}
	if strings.Count(markup, `class="margo-deck__header">Atlas review</header>`) != 2 || strings.Count(markup, `class="margo-deck__footer">Internal review</footer>`) != 2 {
		t.Fatalf("frontmatter header/footer chrome missing: %s", markup)
	}
	badge := strings.Index(markup, `margo-deck__confidentiality-badge`)
	icon := strings.Index(markup, `href="#hi-16-solid-clock"`)
	ordinal := strings.Index(markup, `class="margo-deck__pagination" aria-hidden="true">1</span>`)
	if badge < 0 || icon < 0 || ordinal < 0 || badge > icon || icon > ordinal {
		t.Fatalf("frontmatter host chrome is not emitted in contract order: %s", markup)
	}
}

func TestRenderChromeProjectsBoundedInlineMarkdown(t *testing.T) {
	result, err := Render(context.Background(), margo.New(), RenderInput{
		Name: "inline-chrome.md",
		Markdown: []byte("<!-- paginate: true -->\n" +
			"<!-- header: \"**Bold header** and [deck docs](https://example.com/docs)\" -->\n" +
			"<!-- footer: \"*Italic footer* with `code` & text\" -->\n" +
			"# Inline chrome\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	markup := string(result.HTML())
	for _, fragment := range []string{
		`<header class="margo-deck__header"><strong>Bold header</strong> and <a href="https://example.com/docs">deck docs</a></header>`,
		`<footer class="margo-deck__footer"><em>Italic footer</em> with <code>code</code> &amp; text</footer>`,
	} {
		if !strings.Contains(markup, fragment) {
			t.Fatalf("inline chrome missing %q: %s", fragment, markup)
		}
	}
	for _, literal := range []string{"**Bold header**", "*Italic footer*", "`code`"} {
		if strings.Contains(markup, literal) {
			t.Fatalf("inline Markdown marker leaked into deck HTML: %q", literal)
		}
	}
}

func TestRenderRejectsUnsafeInlineChrome(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		wantErr string
	}{
		{name: "raw HTML", header: `<script>alert(1)</script>`, wantErr: "policy.raw_html.denied"},
		{name: "unsafe link", header: `[unsafe](javascript:alert(1))`, wantErr: "render.link_invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Render(context.Background(), margo.New(), RenderInput{
				Name:     "unsafe-inline-chrome.md",
				Markdown: []byte("<!-- header: " + quoteDirectiveValue(test.header) + " -->\n# Unsafe chrome\n"),
			})
			if result != nil || err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("result = %#v, err = %v; want %q", result, err, test.wantErr)
			}
		})
	}
}

func quoteDirectiveValue(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func TestRenderDarkDeckActivatesGoshtosoDarkSelectors(t *testing.T) {
	result, err := Render(context.Background(), margo.New(), RenderInput{
		Name:     "dark-table.md",
		Markdown: []byte("---\ntheme: goshtoso\ncolorMode: dark\n---\n# Readable table\n\n| Surface | Status |\n| --- | --- |\n| HTML | pass |\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	markup := string(result.HTML())
	for _, fragment := range []string{
		`<html lang="en" class="dark" data-theme="goshtoso" data-color-mode="dark"`,
		`text-on-surface dark:text-on-surface-dark`,
		`dark:text-on-surface-dark-strong`,
	} {
		if !strings.Contains(markup, fragment) {
			t.Fatalf("dark deck HTML missing %q", fragment)
		}
	}
}

func TestRenderPlacesHostConfidentialityBadgeBeforeOrdinal(t *testing.T) {
	result, err := Render(context.Background(), margo.New(), RenderInput{
		Name:     "confidential.md",
		Markdown: []byte("<!-- paginate: true -->\n<!-- footer: Gerado pelo Margo -->\n# One\n"),
	}, WithConfidentialityBadge("Confidencial"))
	if err != nil {
		t.Fatal(err)
	}
	markup := string(result.HTML())
	badge := strings.Index(markup, `margo-deck__confidentiality-badge`)
	ordinal := strings.Index(markup, `class="margo-deck__pagination" aria-hidden="true">1</span>`)
	if badge < 0 || ordinal < 0 || badge > ordinal {
		t.Fatalf("confidentiality badge and ordinal are not in contract order: %s", markup)
	}
	if !strings.Contains(markup, `class="margo-deck__bottom-chrome"`) || !strings.Contains(markup, `class="margo-deck__pagination-cluster"`) || !strings.Contains(markup[badge:ordinal], `>Confidencial</span>`) {
		t.Fatalf("confidentiality cluster missing accessible Goshtoso badge: %s", markup)
	}
}

func TestRenderPlacesDecorativePaginationIconBeforeOrdinal(t *testing.T) {
	result, err := Render(context.Background(), margo.New(), RenderInput{
		Name:     "icon.md",
		Markdown: []byte("<!-- paginate: true -->\n# One\n"),
	}, WithPaginationIcon(PaginationIconConfig{
		Symbol:     "hi-16-solid-clock",
		Placement:  PaginationIconBefore,
		Decorative: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	markup := string(result.HTML())
	icon := strings.Index(markup, `href="#hi-16-solid-clock"`)
	ordinal := strings.Index(markup, `class="margo-deck__pagination" aria-hidden="true">1</span>`)
	if icon < 0 || ordinal < 0 || icon > ordinal {
		t.Fatalf("pagination icon and ordinal are not in contract order: %s", markup)
	}
	for _, fragment := range []string{
		`class="size-4 margo-deck__pagination-icon"`,
		`aria-hidden="true"`,
		`<symbol id="hi-16-solid-clock"`,
		`class="margo-deck__pagination-cluster"`,
	} {
		if !strings.Contains(markup, fragment) {
			t.Fatalf("pagination icon markup missing %q: %s", fragment, markup)
		}
	}
}

func TestRenderPlacesLabeledPaginationIconAfterOrdinal(t *testing.T) {
	result, err := Render(context.Background(), margo.New(), RenderInput{
		Name:     "icon.md",
		Markdown: []byte("<!-- paginate: true -->\n# One\n"),
	}, WithPaginationIcon(PaginationIconConfig{
		Symbol:    "hi-16-solid-clock",
		Placement: PaginationIconAfter,
		Label:     "Horário",
	}))
	if err != nil {
		t.Fatal(err)
	}
	markup := string(result.HTML())
	ordinal := strings.Index(markup, `class="margo-deck__pagination" aria-hidden="true">1</span>`)
	icon := strings.Index(markup, `href="#hi-16-solid-clock"`)
	if ordinal < 0 || icon < 0 || ordinal > icon {
		t.Fatalf("pagination icon and ordinal are not in contract order: %s", markup)
	}
	for _, fragment := range []string{`role="img"`, `aria-label="Horário"`} {
		if !strings.Contains(markup, fragment) {
			t.Fatalf("labeled pagination icon missing %q: %s", fragment, markup)
		}
	}
}

func TestRenderRejectsUnknownPaginationIcon(t *testing.T) {
	_, err := Render(context.Background(), margo.New(), RenderInput{
		Name:     "icon.md",
		Markdown: []byte("<!-- paginate: true -->\n# One\n"),
	}, WithPaginationIcon(PaginationIconConfig{
		Symbol:     "unknown-icon",
		Placement:  PaginationIconBefore,
		Decorative: true,
	}))
	if got := deckDiagnosticCode(err); got != "deck.pagination_icon_invalid" {
		t.Fatalf("error code = %q, err = %v", got, err)
	}
}

func TestRenderProjectsStructuralLayoutDOMInSourceOrder(t *testing.T) {
	result := mustRenderDeck(t, "<!-- _class: columns -->\n<!-- layout: columns -->\n<!-- slot: left -->\n# Left\n<!-- slot: right -->\n# Right\n<!-- /layout -->\n")
	html := string(result.HTML())
	for _, fragment := range []string{
		`margo-deck__slide--columns`,
		`margo-layout margo-layout--columns`,
		`margo-layout__slot margo-layout__slot--left`,
		`margo-layout__slot margo-layout__slot--right`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("HTML missing %q", fragment)
		}
	}
	if strings.Index(html, ">Left</h1>") > strings.Index(html, ">Right</h1>") {
		t.Fatal("layout DOM reordered source slots")
	}
}

func TestRenderCompositionDataAttributesAndSemanticSlots(t *testing.T) {
	result := mustRenderDeck(t, "<!-- composition: media-split -->\n<!-- slot: media -->\n# Media\n<!-- slot: content -->\n# Content\n")
	html := string(result.HTML())
	for _, fragment := range []string{
		`data-margo-composition-catalog="r1"`,
		`data-margo-composition="media-split"`,
		`data-margo-composition-variant="split"`,
		`data-margo-composition-family="columns"`,
		`data-margo-slot="media"`,
		`data-margo-slot-role="media"`,
		`data-margo-slot="content"`,
		`data-margo-slot-role="content"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("HTML missing %q", fragment)
		}
	}
	if strings.Index(html, `data-margo-slot="media"`) > strings.Index(html, `data-margo-slot="content"`) {
		t.Fatal("composition slots are not emitted in source order")
	}
}

func TestRenderUncomposedSlideHasNoCompositionMetadata(t *testing.T) {
	html := string(mustRenderDeck(t, "# One\n").HTML())
	start := strings.Index(html, "<article")
	end := strings.Index(html, "</article>")
	if start < 0 || end < start {
		t.Fatal("article not found")
	}
	article := html[start:end]
	if strings.Contains(article, `data-margo-composition="`) || strings.Contains(article, `data-margo-composition-catalog="`) || strings.Contains(article, `data-margo-slot-role="`) {
		t.Fatalf("uncomposed deck gained composition metadata: %s", article)
	}
}

func TestRenderCompositionLabelsFollowSlideLanguage(t *testing.T) {
	result := mustRenderDeck(t, "---\nlang: pt-BR\ncomposition: compare-grid\n---\n<!-- slot: item-1 -->\n# Um\n<!-- slot: item-2 -->\n# Dois\n")
	html := string(result.HTML())
	if !strings.Contains(html, `role="group" aria-label="Comparação"`) {
		t.Fatalf("localized composition label missing: %s", html)
	}
}

func TestDeckCSSDeclaresR1GridAndVariants(t *testing.T) {
	for _, fragment := range []string{
		".margo-layout--grid",
		"data-margo-composition-variant",
		"media-stage",
		"compare-grid",
		"image-grid",
	} {
		if !strings.Contains(deckCSS, fragment) {
			t.Fatalf("deck CSS missing %q", fragment)
		}
	}
}

func TestRenderEmitsBackgroundAccessibilitySemantics(t *testing.T) {
	result := mustRenderDeck(t, "<!-- color: accent-strong -->\n<!-- backgroundImage: images/hero.png -->\n<!-- backgroundPosition: top-right -->\n<!-- backgroundRepeat: no-repeat -->\n<!-- backgroundSize: contain -->\n<!-- backgroundAlt: Hero image -->\n# One\n")
	html := string(result.HTML())
	for _, fragment := range []string{`data-margo-color="accent-strong"`, `role="img"`, `aria-label="Hero image"`, `data-margo-background="images/hero.png"`, `data-margo-background-position="top-right"`, `data-margo-background-repeat="no-repeat"`, `data-margo-background-size="contain"`} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("HTML missing %q", fragment)
		}
	}
}

func TestRenderComposesRuntimeDescriptorsAcrossSlides(t *testing.T) {
	result := mustRenderDeck(t, "```mermaid\ngraph TD; A-->B\n```\n---\n```mermaid\ngraph TD; C-->D\n```\n")
	descriptor, err := result.RuntimeDescriptor("ri-00000042")
	if err != nil {
		t.Fatal(err)
	}
	if len(descriptor.Tasks) != 4 {
		t.Fatalf("tasks = %#v", descriptor.Tasks)
	}
	if descriptor.Tasks[0].ID == descriptor.Tasks[1].ID {
		t.Fatal("runtime task IDs collide")
	}
	if descriptor.DocumentFingerprint != result.DocumentFingerprint() {
		t.Fatal("runtime and deck fingerprints differ")
	}
	if err := margo.ValidateRuntimeDescriptor(descriptor); err != nil {
		t.Fatal(err)
	}
}

func TestRenderFailsAtomicallyForInvalidLaterSlide(t *testing.T) {
	result, err := Render(context.Background(), margo.New(), RenderInput{
		Name:     "invalid.md",
		Markdown: []byte("# Valid\n---\n```mermaid\n%%{init: {}}%%\ngraph TD; A-->B\n```\n"),
	})
	if err == nil || result != nil {
		t.Fatalf("result = %#v error = %v", result, err)
	}
}

func TestRenderRejectsNilCompiler(t *testing.T) {
	result, err := Render(context.Background(), nil, RenderInput{Name: "deck.md", Markdown: []byte("# One\n")})
	if got := deckDiagnosticCode(err); got != "deck.compiler_required" || result != nil {
		t.Fatalf("result = %#v code = %q", result, got)
	}
}

func TestRenderRejectsConflictingGeometryLayers(t *testing.T) {
	inputGeometry := DeckGeometry{Preset: "4:3", Width: 960, Height: 720, Unit: DeckUnitPX}
	optionGeometry := DeckGeometry{Preset: "16:9", Width: 1280, Height: 720, Unit: DeckUnitPX}
	_, err := Render(context.Background(), margo.New(), RenderInput{Name: "conflict.md", Markdown: []byte("# One\n"), Geometry: inputGeometry}, WithGeometry(optionGeometry))
	if err == nil || !strings.Contains(err.Error(), "deck.geometry_conflict") {
		t.Fatalf("error = %v", err)
	}
}

func TestResultHTMLIsDefensive(t *testing.T) {
	result := mustRenderDeck(t, "# One\n")
	first := result.HTML()
	first[0] = 'X'
	if second := result.HTML(); len(second) == 0 || second[0] == 'X' {
		t.Fatal("HTML aliases result storage")
	}
}

func TestRenderEmbedsNavigationPrintAndDependencies(t *testing.T) {
	result := mustRenderDeck(t, "# One\n---\n# Two\n")
	html := string(result.HTML())
	for _, value := range []string{
		"<!doctype html>",
		`data-margo-requirement="margo.document.styles"`,
		"data-margo-deck-previous",
		"data-margo-deck-next",
		"ArrowLeft",
		"ArrowRight",
		"Home",
		"End",
		`closest("button, a, [role='button']`,
		"@media print",
		"break-after: page",
		"window.print",
		"margoPrepareDeckPrint",
		"beforeprint",
		`script-src 'self' 'unsafe-inline'`,
	} {
		if !strings.Contains(html, value) {
			t.Fatalf("HTML missing %q", value)
		}
	}
}

func TestResultRequirementsAreMergedAndDefensive(t *testing.T) {
	result := mustRenderDeck(t, "| A | B |\n|---|---|\n| 1 | 2 |\n---\n# Two\n")
	first := result.Requirements().List()
	if len(first) < 3 {
		t.Fatalf("requirements = %#v", first)
	}
	first[0].ID = "changed"
	if result.Requirements().List()[0].ID == "changed" {
		t.Fatal("requirements alias result storage")
	}
}

func mustRenderDeck(t *testing.T, source string) *Result {
	t.Helper()
	result, err := Render(context.Background(), margo.New(), RenderInput{Name: "deck.md", Markdown: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
