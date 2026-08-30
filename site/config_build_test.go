package site

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/charts"
	"golang.org/x/net/html"
)

func TestInjectComponentDocShellPageDependenciesAfterShellHead(t *testing.T) {
	for _, closingHead := range []string{"</head>", "</HEAD>"} {
		t.Run(closingHead, func(t *testing.T) {
			document := []byte(`<!doctype html><html><head><script src="shell.js"></script>` + closingHead + `<body></body></html>`)
			got, err := injectComponentDocShellPageDependencies(document, []byte(`<script defer src="page.js"></script>`), "guide.md")
			if err != nil {
				t.Fatal(err)
			}
			want := `<!doctype html><html><head><script src="shell.js"></script><script defer src="page.js"></script>` + closingHead + `<body></body></html>`
			if string(got) != want {
				t.Fatalf("document = %s, want %s", got, want)
			}
		})
	}
}

func TestInjectComponentDocShellPageDependenciesRequiresHead(t *testing.T) {
	_, err := injectComponentDocShellPageDependencies([]byte(`<html><body></body></html>`), []byte(`<script src="page.js"></script>`), "guide.md")
	if err == nil {
		t.Fatal("expected missing-head diagnostic")
	}
	var diagnosticError *margo.DiagnosticError
	if !errors.As(err, &diagnosticError) || len(diagnosticError.Diagnostics) != 1 {
		t.Fatalf("error = %v, want one diagnostic", err)
	}
	diagnostic := diagnosticError.Diagnostics[0]
	if diagnostic.Code != "site.html_invalid" || diagnostic.Source != "guide.md" {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
}

func TestBuildConfigRendersRouteOwnedSocialMetadataAndFrameBindings(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Home\n\nWelcome to the Margo docs.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "guide.md"), "---\ntitle: Guide\ndescription: A guide-specific description.\nlanguage: en\nauthors: [Ana Silva]\npublishedAt: \"2026-08-25T12:00:00Z\"\nmodifiedAt: \"2026-08-26T12:00:00Z\"\ntags: [operations]\n---\n# Guide\n\nA guide-specific description.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
output: dist
assets: local
offline: true
base_path: /docs
site:
  name: Margo
  description: Margo documentation
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo documentation preview
frame:
  builtin: top-left-main-footer
  values:
    areas:
      top-nav:
        sticky:
          enabled: true
          edge: top
          offset: 1rem
locales:
  default: en
  supported: [en]
navigation:
  mode: file-tree
bindings:
  navigation:
    area: left-nav
  breadcrumbs:
    area: top-nav
  pagination:
    area: main-content
    slot: after-article
  footer:
    area: footer
theme:
  builtin: true
  name: modern
  color_mode: light
`)

	first, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("configured build is not byte-deterministic")
	}
	if first.Site.Layout != "frame:top-left-main-footer" || first.Site.BasePath != "/docs" || len(first.Site.Routes) != 2 {
		t.Fatalf("site manifest = %+v", first.Site)
	}
	index := string(configArtifact(t, first, "index.html"))
	guide := string(configArtifact(t, first, "guide.html"))
	for _, page := range []string{index, guide} {
		for _, required := range []string{
			`<link rel="canonical" href="https://margo.example/docs/`,
			`property="og:url"`, `property="og:image" content="https://margo.example/docs/assets/social.jpg"`, `property="og:image:type" content="image/jpeg"`,
			`property="og:image:width" content="1280"`, `property="og:image:height" content="640"`,
			`name="twitter:card" content="summary_large_image"`,
			`name="twitter:image:alt" content="Margo documentation preview"`,
			`id="margo-document"`, `class="margo-skip-link"`, `aria-current="page"`,
		} {
			if !strings.Contains(page, required) {
				t.Fatalf("page missing %q: %s", required, page)
			}
		}
		if strings.Count(page, `<title>`) != 1 || strings.Count(page, `property="og:url"`) != 1 || strings.Count(page, `name="twitter:card"`) != 1 {
			t.Fatalf("duplicate route metadata: %s", page)
		}
	}
	if !strings.Contains(index, `href="guide.html"`) || !strings.Contains(index, `Article navigation`) {
		t.Fatalf("frame bindings missing: %s", index)
	}
	if strings.Contains(index, `margo-breadcrumbs`) {
		t.Fatalf("home page unexpectedly renders breadcrumbs: %s", index)
	}
	for _, required := range []string{
		`<nav class="text-sm font-medium text-on-surface dark:text-on-surface-dark margo-breadcrumbs" aria-label="Breadcrumbs">`,
		`<ol class="flex flex-wrap items-center gap-1">`,
		`href="index.html">Home</a>`,
		`aria-current="page">Guide</li>`,
		`<svg xmlns="http://www.w3.org/2000/svg"`,
	} {
		if !strings.Contains(guide, required) {
			t.Fatalf("Goshtoso breadcrumb markup missing %q: %s", required, guide)
		}
	}
	if strings.Contains(guide, `aria-current="false"`) {
		t.Fatalf("breadcrumb still emits a non-semantic current marker: %s", guide)
	}
	if !strings.Contains(index, `data-margo-sticky="true" data-margo-sticky-edge="block-start" data-margo-sticky-offset="1rem"`) {
		t.Fatalf("frame values missing from rendered area: %s", index)
	}
	if !strings.Contains(guide, `https://margo.example/docs/guide.html`) || strings.Contains(index, `https://margo.example/docs/guide.html`) {
		t.Fatalf("route metadata not distinct: index=%s guide=%s", index, guide)
	}
	for _, required := range []string{
		`<address aria-label="Authors"><span rel="author">Ana Silva</span></address>`,
		`class="margo-document__publication-dates" role="group" aria-label="Publication dates"`,
		`data-margo-publication-label="published">Published</span>`,
		`data-margo-publication-label="modified">Updated</span>`,
		`data-margo-publication-separator="true"`,
		`<time datetime="2026-08-25T12:00:00Z" data-margo-publication-date="published">2026-08-25T12:00:00Z</time>`,
		`<time datetime="2026-08-26T12:00:00Z" data-margo-publication-date="modified">2026-08-26T12:00:00Z</time>`,
		`<li data-margo-publication-tag="operations">operations</li>`,
		`<meta property="article:published_time" content="2026-08-25T12:00:00Z"`,
		`<meta property="article:modified_time" content="2026-08-26T12:00:00Z"`,
		`<meta property="article:author" content="Ana Silva"`,
		`<meta property="article:tag" content="operations"`,
	} {
		if !strings.Contains(guide, required) {
			t.Fatalf("configured publication projection missing %q: %s", required, guide)
		}
	}
	if len(first.Site.Routes) != 2 || !reflect.DeepEqual(first.Site.Routes[1].Authors, []string{"Ana Silva"}) {
		t.Fatalf("configured route publication metadata = %+v", first.Site.Routes)
	}
	if got := configArtifact(t, first, "assets/social.jpg"); len(got) < 1000 {
		t.Fatalf("social asset unexpectedly small: %d", len(got))
	}
	sitemap := string(configArtifact(t, first, SitemapPath))
	for _, required := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`,
		`<loc>https://margo.example/docs/</loc>`,
		`<loc>https://margo.example/docs/guide.html</loc>`,
	} {
		if !strings.Contains(sitemap, required) {
			t.Fatalf("sitemap missing %q: %s", required, sitemap)
		}
	}
	if strings.Count(sitemap, "<url>") != 2 {
		t.Fatalf("sitemap url count = %d: %s", strings.Count(sitemap, "<url>"), sitemap)
	}
	llms := string(configArtifact(t, first, LLMSPath))
	for _, required := range []string{
		"# Margo", "> Margo documentation", "## Documentation",
		"[Home](<https://margo.example/docs/>)",
		"[Guide](<https://margo.example/docs/guide.html>)",
	} {
		if !strings.Contains(llms, required) {
			t.Fatalf("llms.txt missing %q: %s", required, llms)
		}
	}
}

func TestBuildConfigPublishesNonIndexHomeAtCanonicalRoot(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "reports", "quarterly.md"), `---
title: Q3 Operations Briefing
description: Board-ready quarterly operations briefing.
language: en
margo:
  actions:
    markdown: true
---
# Q3 Operations Briefing

![Briefing mark](../assets/briefing.svg)

[Read the guide](../guide.md)
`)
	writeConfigFile(t, filepath.Join(root, "docs", "guide.md"), "# Guide\n\n[Back to briefing](reports/quarterly.md)\n")
	copyMargoAsset(t, filepath.Join(root, "docs", "assets", "briefing.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
output: dist
assets: local
offline: true
base_path: /briefing
site:
  name: Northstar Holdings
  description: Board-ready quarterly operations briefing.
  base_url: https://briefing.northstar.example
  home: reports/quarterly.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Northstar quarterly operations briefing
locales:
  default: en
  supported: [en]
navigation:
  mode: file-tree
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	if !artifactExists(result, "index.html") {
		t.Fatalf("canonical home artifact missing: %+v", result.Artifacts)
	}
	if artifactExists(result, "reports/quarterly.html") {
		t.Fatalf("source-derived home HTML artifact was published: %+v", result.Artifacts)
	}
	if !artifactExists(result, "reports/quarterly.md") {
		t.Fatalf("home Markdown action artifact missing: %+v", result.Artifacts)
	}
	if len(result.Pages) != 2 || result.Pages[0].Source != "reports/quarterly.md" || result.Pages[0].Output != "index.html" {
		t.Fatalf("home page route = %+v, want source reports/quarterly.md at index.html", result.Pages)
	}
	if result.Pages[0].Canonical != "https://briefing.northstar.example/briefing/" {
		t.Fatalf("home canonical = %q", result.Pages[0].Canonical)
	}
	index := string(configArtifact(t, result, "index.html"))
	for _, want := range []string{
		"Q3 Operations Briefing",
		`src="assets/briefing.svg"`,
		`href="guide.html"`,
		`data-margo-markdown-url="reports/quarterly.md"`,
	} {
		if !strings.Contains(index, want) {
			t.Fatalf("canonical home missing %q: %s", want, index)
		}
	}
	guide := string(configArtifact(t, result, "guide.html"))
	if !strings.Contains(guide, `href="index.html"`) {
		t.Fatalf("guide home link does not target canonical index: %s", guide)
	}
	sitemap := string(configArtifact(t, result, SitemapPath))
	if !strings.Contains(sitemap, "<loc>https://briefing.northstar.example/briefing/</loc>") {
		t.Fatalf("sitemap does not advertise canonical home: %s", sitemap)
	}
	llms := string(configArtifact(t, result, LLMSPath))
	if !strings.Contains(llms, "[Q3 Operations Briefing](<https://briefing.northstar.example/briefing/>)") {
		t.Fatalf("llms.txt does not advertise canonical home: %s", llms)
	}
}

func TestBuildConfigPublishesExistingLinkedAssets(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Home\n\n[Download](assets/format-study.pdf?download=1#page=2)\n")
	writeConfigFile(t, filepath.Join(root, "docs", "nested", "guide.md"), "# Guide\n\n[Download](../assets/format-study.pdf?download=1#page=2)\n")
	assetContent := []byte("format study")
	assetPath := filepath.Join(root, "docs", "assets", "format-study.pdf")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath, assetContent, 0o644); err != nil {
		t.Fatal(err)
	}
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
site:
  name: Margo
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo documentation preview
locales:
  default: en
  supported: [en]
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{
		"index.html":        `href="assets/format-study.pdf?download=1#page=2"`,
		"nested/guide.html": `href="../assets/format-study.pdf?download=1#page=2"`,
	} {
		page := string(configArtifact(t, result, name))
		if !strings.Contains(page, want) {
			t.Fatalf("%s missing %q:\n%s", name, want, page)
		}
	}
	if got := configArtifact(t, result, "assets/format-study.pdf"); !bytes.Equal(got, assetContent) {
		t.Fatalf("linked asset = %q, want %q", got, assetContent)
	}
}

func TestBuildConfigPublishesLocalAVIFImage(t *testing.T) {
	root := t.TempDir()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	avif, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "examples", "blog", "site", "assets", "atelier-hero.avif"))
	if err != nil {
		t.Fatal(err)
	}
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Home\n\n![Atelier hero](assets/atelier-hero.avif)\n")
	writeConfigFile(t, filepath.Join(root, "docs", "assets", "atelier-hero.avif"), string(avif))
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
site:
  name: Margo
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo documentation preview
locales:
  default: en
  supported: [en]
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	page := string(configArtifact(t, result, "index.html"))
	if !strings.Contains(page, `src="assets/atelier-hero.avif"`) {
		t.Fatalf("configured page did not publish AVIF reference: %s", page)
	}
	if got := configArtifact(t, result, "assets/atelier-hero.avif"); !bytes.Equal(got, avif) {
		t.Fatalf("published AVIF differs from source: got %d bytes, want %d", len(got), len(avif))
	}
}

func TestBuildConfigRejectsSourceAssetShadowingConfiguredCache(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Home\n\n[Stylesheet](assets/site.css)\n")
	writeConfigFile(t, filepath.Join(root, "docs", "assets", "site.css"), "source stylesheet")
	writeConfigFile(t, filepath.Join(root, "assets", "site.css"), "configured stylesheet")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
site:
  name: Margo
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo documentation preview
custom_css:
  - css_url: assets/site.css
locales:
  default: en
  supported: [en]
`)
	_, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	requireSiteCode(t, err, "site.artifact_collision")
}

func TestBuildConfigLocalizesVisibleFrameControls(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Início\n\nPágina inicial.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "guide.md"), "# Guia\n\nPágina do guia.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
output: dist
site:
  name: Margo
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Prévia da documentação
frame:
  builtin: top-main-footer
locales:
  default: pt-BR
  supported: [pt-BR]
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	index := string(configArtifact(t, result, "index.html"))
	guide := string(configArtifact(t, result, "guide.html"))
	for route, page := range map[string]string{"index": index, "guide": guide} {
		if !strings.Contains(page, `>Ir para o conteúdo</a>`) {
			t.Fatalf("%s skip link is not localized: %s", route, page)
		}
		for _, english := range []string{"Skip to content", "Previous:", "Next:"} {
			if strings.Contains(page, english) {
				t.Fatalf("%s contains English frame control %q: %s", route, english, page)
			}
		}
	}
	if !strings.Contains(index, `>Próximo: Guia</a>`) {
		t.Fatalf("next-page label is not localized: %s", index)
	}
	if !strings.Contains(guide, `>Anterior: Início</a>`) {
		t.Fatalf("previous-page label is not localized: %s", guide)
	}
}

func TestLocalizedLabelFallsBackToEnglish(t *testing.T) {
	for key, want := range map[string]string{
		"skip_content": "Skip to content",
		"previous":     "Previous",
		"next":         "Next",
	} {
		if got := localizedLabel("es", key); got != want {
			t.Errorf("localizedLabel(es, %q) = %q, want %q", key, got, want)
		}
	}
}

func TestApplyLandingShellSemanticsLocalizesSkipLink(t *testing.T) {
	document := []byte(`<!doctype html><html><body><a class="landing-shell__skip" href="#main-content">Skip to main content</a><main id="main-content">Content</main><footer><div class="landing-shell__footer-inner">Old footer</div></footer></body></html>`)
	got, err := applyLandingShellSemantics(document, Page{Locale: "pt-BR", Source: "index.md"}, []byte(`<p>Footer</p>`))
	if err != nil {
		t.Fatal(err)
	}
	page := string(got)
	for _, required := range []string{`href="#margo-document"`, `>Ir para o conteúdo</a>`} {
		if !strings.Contains(page, required) {
			t.Fatalf("localized landing skip link missing %q: %s", required, page)
		}
	}
	if strings.Contains(page, "Skip to main content") {
		t.Fatalf("landing skip link retained English text: %s", page)
	}
	if !strings.Contains(page, `<p>Footer</p>`) || strings.Contains(page, "Old footer") {
		t.Fatalf("landing footer was not replaced: %s", page)
	}
}

func TestBuildConfiguredShowcasePublicationContract(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	root := filepath.Join(filepath.Dir(filename), "..")
	result, err := BuildConfig(context.Background(), ConfigRequest{
		ConfigPath: filepath.Join(root, "showcase.yaml"),
		Compiler: margo.New(margo.WithUnsafeHTML(), margo.WithExtension(charts.Extension(
			charts.WithExternalizedControlRuntime(true),
		))),
		PDFEngine: siteTestPDFEngine{},
	})
	if err != nil {
		t.Fatal(err)
	}
	commandNames := []string{"check", "completion", "deck", "doctor", "html", "pdf", "schema", "serve", "site", "version"}
	schemaNames := []string{"check-report", "deck-layout-evidence", "deck-pdf-artifact-report", "diagnostic", "doctor-report", "document", "goshtoso-chart", "policy", "runtime-descriptor", "runtime-report", "site", "site-manifest", "site-report"}
	fencedTypeNames := []string{"code", "goshtoso-chart", "jsonschema", "mermaid"}

	htmlRoutes := make([]string, 0)
	artifacts := make(map[string][]byte, len(result.Artifacts))
	for _, artifact := range result.Artifacts {
		artifacts[artifact.Path] = artifact.Content
		if strings.HasSuffix(artifact.Path, ".html") {
			htmlRoutes = append(htmlRoutes, artifact.Path)
		}
	}
	sort.Strings(htmlRoutes)
	wantHTMLRoutes := []string{
		"cli/index.html", "examples/cli-workspace/guide.html", "examples/deck-workspace/slides.html",
		"cli/deck/chart-slides.html", "cli/deck/chrome-pagination.html", "cli/deck/compositions-r1.html",
		"cli/deck/structural-layouts.html", "cli/deck/theme-goshtoso.html", "cli/deck/theme-minimal.html",
		"index.html", "module/index.html",
	}
	for _, command := range commandNames {
		wantHTMLRoutes = append(wantHTMLRoutes, "cli/"+command+"/index.html")
	}
	for _, schema := range schemaNames {
		wantHTMLRoutes = append(wantHTMLRoutes, "schemas/"+schema+"/index.html")
	}
	wantHTMLRoutes = append(wantHTMLRoutes, "schemas/index.html")
	wantHTMLRoutes = append(wantHTMLRoutes, "fenced-types/index.html")
	for _, fencedType := range fencedTypeNames {
		wantHTMLRoutes = append(wantHTMLRoutes, "fenced-types/"+fencedType+"/index.html")
	}
	sort.Strings(wantHTMLRoutes)
	if got, want := htmlRoutes, wantHTMLRoutes; !reflect.DeepEqual(got, want) {
		t.Fatalf("HTML routes = %v, want exactly %v", got, want)
	}
	for _, retired := range []string{"charts", "cli", "decks", "determinism", "html", "markdown", "mermaid", "module", "pdf", "policy", "site"} {
		if _, exists := artifacts[retired+".html"]; exists {
			t.Fatalf("retired route artifact %q exists", retired+".html")
		}
	}
	if got, want := len(result.Site.Routes), 3+len(commandNames)+len(schemaNames)+1+len(fencedTypeNames)+1; got != want {
		t.Fatalf("configured routes = %d, want %d: %+v", got, want, result.Site.Routes)
	}
	wantRoutes := map[string]struct {
		output string
		family string
		layout string
	}{
		"index.md":              {output: "index.html", family: "", layout: "landing"},
		"module/index.md":       {output: "module/index.html", family: "module", layout: "docs"},
		"cli/index.md":          {output: "cli/index.html", family: "cli", layout: "docs"},
		"schemas/index.md":      {output: "schemas/index.html", family: "schemas", layout: "docs"},
		"fenced-types/index.md": {output: "fenced-types/index.html", family: "fenced-types", layout: "docs"},
	}
	for _, command := range commandNames {
		wantRoutes["cli/"+command+"/index.md"] = struct {
			output string
			family string
			layout string
		}{output: "cli/" + command + "/index.html", family: "cli", layout: "docs"}
	}
	for _, schema := range schemaNames {
		wantRoutes["schemas/"+schema+"/index.md"] = struct {
			output string
			family string
			layout string
		}{output: "schemas/" + schema + "/index.html", family: "schemas", layout: "docs"}
	}
	for _, fencedType := range fencedTypeNames {
		wantRoutes["fenced-types/"+fencedType+"/index.md"] = struct {
			output string
			family string
			layout string
		}{output: "fenced-types/" + fencedType + "/index.html", family: "fenced-types", layout: "docs"}
	}
	for _, route := range result.Site.Routes {
		want, ok := wantRoutes[route.Source]
		if !ok || route.Output != want.output || route.Family != want.family || route.Layout != want.layout {
			t.Fatalf("route = %+v, want one of %+v", route, wantRoutes)
		}
		if route.Source == "index.md" {
			if route.Actions != nil {
				t.Fatalf("Tour unexpectedly has page actions: %+v", route.Actions)
			}
		} else if route.Actions == nil || !route.Actions.Markdown || !route.Actions.PDF {
			t.Fatalf("technical route %q actions = %+v, want Markdown and PDF", route.Source, route.Actions)
		}
	}
	if got, want := result.Site.Layout, "layout:docs"; got != want {
		t.Fatalf("layout identity = %q, want %q", got, want)
	}
	if result.Site.LayoutSchemaHash == "" || result.Site.LayoutSchemaHash == "legacy" {
		t.Fatalf("layout schema identity = %q", result.Site.LayoutSchemaHash)
	}
	for artifact := range artifacts {
		if artifact == "_layout.yaml" || strings.HasSuffix(artifact, "/_layout.yaml") {
			t.Fatalf("reserved layout patch emitted as artifact %q", artifact)
		}
	}

	landing := string(artifacts["index.html"])
	module := string(artifacts["module/index.html"])
	cli := string(artifacts["cli/index.html"])
	schemas := string(artifacts["schemas/index.html"])
	fencedTypes := string(artifacts["fenced-types/index.html"])
	publicationPages := map[string]struct {
		content string
		route   string
	}{
		"Tour":         {content: landing, route: "https://margo.araihu.com/"},
		"Module":       {content: module, route: "https://margo.araihu.com/module/"},
		"CLI":          {content: cli, route: "https://margo.araihu.com/cli/"},
		"Schemas":      {content: schemas, route: "https://margo.araihu.com/schemas/"},
		"Fenced types": {content: fencedTypes, route: "https://margo.araihu.com/fenced-types/"},
	}
	for _, command := range commandNames {
		publicationPages[command] = struct {
			content string
			route   string
		}{
			content: string(artifacts["cli/"+command+"/index.html"]),
			route:   "https://margo.araihu.com/cli/" + command + "/",
		}
	}
	for _, schema := range schemaNames {
		publicationPages[schema] = struct {
			content string
			route   string
		}{
			content: string(artifacts["schemas/"+schema+"/index.html"]),
			route:   "https://margo.araihu.com/schemas/" + schema + "/",
		}
	}
	for _, fencedType := range fencedTypeNames {
		publicationPages[fencedType] = struct {
			content string
			route   string
		}{content: string(artifacts["fenced-types/"+fencedType+"/index.html"]), route: "https://margo.araihu.com/fenced-types/" + fencedType + "/"}
	}
	for name, page := range publicationPages {
		route := page.route
		if !strings.Contains(page.content, `<link rel="canonical" href="`+route+`"`) {
			t.Fatalf("%s canonical does not expose public route %q: %s", name, route, page.content)
		}
		if strings.Count(page.content, "<h1") != 1 {
			t.Fatalf("%s h1 count = %d", name, strings.Count(page.content, "<h1"))
		}
	}
	for _, required := range []string{`href="/module/"`, `href="/cli/"`, `href="/schemas/"`, `One source, several projections`} {
		if !strings.Contains(landing, required) {
			t.Fatalf("Tour missing %q", required)
		}
	}
	for _, removed := range []string{
		"Publish with the CLI — check, preview, and build from a standalone workflow",
		"Embed the Go module — keep composition and delivery inside your application",
		`class="landing-shell__footer-brand"`,
	} {
		if strings.Contains(landing, removed) {
			t.Fatalf("Tour retained removed hero/footer content %q", removed)
		}
	}
	for _, required := range []string{
		`<p class="margo-shell-footer">Margo · <strong>Buildt</strong> by `,
		`href="https://araihu.com/"`, `<strong>Arai Hû</strong>`,
		`href="https://goshtoso.araihu.com/"`, `>Goshtoso</a>`,
		`href="/llms.txt"><strong>llms.txt</strong>`,
		`href="/sitemap.xml"><strong>sitemap.xml</strong>`,
	} {
		if !strings.Contains(landing, required) {
			t.Fatalf("Tour missing Goshtoso link/footer contract %q", required)
		}
	}
	if strings.Count(landing, `class="font-medium text-primary`) < 2 {
		t.Fatalf("Tour final navigation did not render two Goshtoso text links")
	}
	for _, required := range []string{"good fit", "not a fit"} {
		if !strings.Contains(strings.ToLower(landing), required) {
			t.Fatalf("Tour missing %q", required)
		}
	}
	for _, required := range []string{"mermaid", "goshtoso-chart", "Margo mascot"} {
		if !strings.Contains(strings.ToLower(landing), strings.ToLower(required)) {
			t.Fatalf("Tour missing retained example marker %q", required)
		}
	}
	for _, required := range []string{
		`class="goshtoso-charts-interactive`,
		`data-goshtoso-chart-capability="interactive-raster"`,
		`data-margo-requirement="goshtoso-charts.runtime"`,
		`Illustrative output mix`,
		`HTML`,
		`Deck`,
	} {
		if !strings.Contains(landing, required) {
			t.Fatalf("Tour interactive chart missing %q", required)
		}
	}
	for _, forbidden := range []string{"margo-breadcrumbs", "margo-pagination", "margo-page-actions", `id="left-nav"`, `id="right-nav"`, "data-toc-heading"} {
		if strings.Contains(landing, forbidden) {
			t.Fatalf("Tour contains forbidden landing markup %q", forbidden)
		}
	}
	for _, required := range []string{"Compiler and render lifecycle", "Public package map", "Select the projection", "Dependencies and upstream boundaries"} {
		if !strings.Contains(module, required) {
			t.Fatalf("Module outline missing %q", required)
		}
	}
	for _, required := range []string{"Command map", "Configuration and policy layering", "Operational gotchas", "check", "completion"} {
		if !strings.Contains(cli, required) {
			t.Fatalf("CLI outline missing %q", required)
		}
	}
	for _, required := range []string{"Configuration schemas", "Output and runtime schemas", "property tree", "margo schema"} {
		if !strings.Contains(schemas, required) {
			t.Fatalf("Schemas outline missing %q", required)
		}
	}
	for _, required := range []string{"Built-in and optional fences", "Mermaid", "Goshtoso charts", "JSON Schema", "Code blocks"} {
		if !strings.Contains(fencedTypes, required) {
			t.Fatalf("Fenced types outline missing %q", required)
		}
	}
	for _, required := range []string{`Generated HTML preview`, `href="../examples/cli-workspace/guide.html"`, `src="../examples/cli-workspace/guide.png"`} {
		if !strings.Contains(cli, required) {
			t.Fatalf("CLI visual example missing %q", required)
		}
	}
	siteGuide := string(artifacts["cli/site/index.html"])
	for _, required := range []string{"margo schema site", "margo.theme.tokens/v1", "css_digest", "custom Marpit theme"} {
		if !strings.Contains(siteGuide, required) {
			t.Fatalf("site guide missing %q", required)
		}
	}
	deckGuide := string(artifacts["cli/deck/index.html"])
	for _, required := range []string{"Live HTML deck", `href="../../examples/deck-workspace/slides.html"`, `src="chart-slides.html"`, `src="structural-layouts.html"`, `src="compositions-r1.html"`, `src="chrome-pagination.html"`, `src="theme-goshtoso.html"`, `src="theme-minimal.html"`} {
		if !strings.Contains(deckGuide, required) {
			t.Fatalf("deck guide visual example missing %q", required)
		}
	}
	visualArtifacts := []string{
		"margo-mascot.png",
		"cli/page-actions.png",
		"cli/deck/chart-slides.html",
		"cli/deck/structural-layouts.html",
		"cli/deck/compositions-r1.html",
		"cli/deck/chrome-pagination.html",
		"cli/deck/theme-goshtoso.html",
		"cli/deck/theme-minimal.html",
		"examples/cli-workspace/guide.png",
		"examples/cli-workspace/guide.html",
		"examples/deck-workspace/slides.html",
	}
	for _, visualPath := range visualArtifacts {
		sourcePath := filepath.Join(root, "showcase", "content", filepath.FromSlash(visualPath))
		want, readErr := os.ReadFile(sourcePath)
		if readErr != nil {
			t.Fatalf("visual source %q could not be read: %v", visualPath, readErr)
		}
		got, exists := artifacts[visualPath]
		if !exists {
			t.Fatalf("visual asset %q was not staged", visualPath)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("visual asset %q was changed while staging", visualPath)
		}
		for _, route := range result.Site.Routes {
			if route.Output == visualPath {
				t.Fatalf("visual asset %q was advertised as a page route", visualPath)
			}
		}
	}
	schemaGuide := string(artifacts["cli/schema/index.html"])
	for _, required := range []string{"yaml.schemas", "json.schemas", "top-level", "$schema", "x-margo-*"} {
		if !strings.Contains(schemaGuide, required) {
			t.Fatalf("schema guide missing %q", required)
		}
	}
	for _, command := range commandNames {
		if !strings.Contains(cli, `href="/cli/`+command+`/"`) {
			t.Fatalf("CLI overview missing command link %q", command)
		}
	}
	checkPage := string(artifacts["cli/check/index.html"])
	for _, command := range commandNames {
		if !strings.Contains(checkPage, `href="/cli/`+command+`/"`) {
			t.Fatalf("CLI sidebar missing command %q", command)
		}
	}
	sitemap := string(artifacts[SitemapPath])
	if got, want := strings.Count(sitemap, "<url>"), 3+len(commandNames)+len(schemaNames)+1+len(fencedTypeNames)+1; got != want {
		t.Fatalf("sitemap URL count = %d, want %d: %s", got, want, sitemap)
	}
	for _, visualPath := range visualArtifacts {
		if strings.Contains(sitemap, visualPath) {
			t.Fatalf("sitemap unexpectedly advertises visual artifact %q: %s", visualPath, sitemap)
		}
	}
	publicRoutes := []string{"https://margo.araihu.com/", "https://margo.araihu.com/module/", "https://margo.araihu.com/cli/", "https://margo.araihu.com/schemas/", "https://margo.araihu.com/fenced-types/"}
	for _, command := range commandNames {
		publicRoutes = append(publicRoutes, "https://margo.araihu.com/cli/"+command+"/")
	}
	for _, schema := range schemaNames {
		publicRoutes = append(publicRoutes, "https://margo.araihu.com/schemas/"+schema+"/")
	}
	for _, fencedType := range fencedTypeNames {
		publicRoutes = append(publicRoutes, "https://margo.araihu.com/fenced-types/"+fencedType+"/")
	}
	for _, route := range publicRoutes {
		if !strings.Contains(sitemap, "<loc>"+route+"</loc>") {
			t.Fatalf("sitemap missing %q", route)
		}
	}
	llms := string(artifacts[LLMSPath])
	wantTitles := []string{"[Margo]", "[Go module]", "[CLI]", "[Schemas]", "[Fenced types]"}
	for _, command := range commandNames {
		wantTitles = append(wantTitles, "["+command+"]")
	}
	for _, title := range wantTitles {
		if !strings.Contains(llms, title) {
			t.Fatalf("llms.txt missing %q: %s", title, llms)
		}
	}
	for _, route := range publicRoutes {
		if !strings.Contains(llms, route) {
			t.Fatalf("llms.txt missing public route %q: %s", route, llms)
		}
	}
	for _, retired := range []string{"charts.html", "decks.html", "determinism.html", "html.html", "markdown.html", "mermaid.html", "pdf.html", "policy.html", "site.html"} {
		if strings.Contains(sitemap, retired) || strings.Contains(llms, retired) {
			t.Fatalf("discovery artifact contains retired route %q", retired)
		}
	}
}

func TestBuildConfigResolvesFamilyLayoutAndPresentationPerPage(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "---\nlayout:\n  kind: landing\n---\n# Landing\n\nChoose a path.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "# Module\n\nModule documentation.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "index.md"), "# CLI\n\nCLI documentation.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "_layout.yaml"), "values:\n  family: module\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "_layout.yaml"), "values:\n  family: cli\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
site:
  name: Margo
  description: Margo documentation
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo preview
layout:
  kind: docs
  default:
    families: [module, cli]
  values:
    family: default
navigation:
  mode: file-tree
locales:
  default: en
  supported: [en]
theme:
  builtin: true
  name: modern
  color_mode: light
`)

	first, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("configured layout build is not deterministic")
	}
	if len(first.Site.Routes) != 3 {
		t.Fatalf("routes = %+v", first.Site.Routes)
	}
	want := map[string]Page{
		"index.md":        {Family: "", Layout: "landing"},
		"module/index.md": {Family: "module", Layout: "docs"},
		"cli/index.md":    {Family: "cli", Layout: "docs"},
	}
	for _, page := range first.Site.Routes {
		expected, ok := want[page.Source]
		if !ok || page.Family != expected.Family || page.Layout != expected.Layout {
			t.Fatalf("route %q identity = family %q layout %q, want %+v", page.Source, page.Family, page.Layout, expected)
		}
	}
	if first.Site.Layout != "layout:docs" {
		t.Fatalf("layout identity = %q", first.Site.Layout)
	}
	if first.Site.LayoutSchemaHash == "" || first.Site.LayoutSchemaHash == "legacy" {
		t.Fatalf("layout schema identity = %q", first.Site.LayoutSchemaHash)
	}
	landing := string(configArtifact(t, first, "index.html"))
	docs := string(configArtifact(t, first, "module/index.html"))
	if !strings.Contains(landing, `data-margo-frame="main"`) {
		t.Fatalf("landing frame missing: %s", landing)
	}
	for _, required := range []string{`class="component-doc-shell`, `id="componentdocshell-sidebar"`, `data-componentdocshell-toc`} {
		if !strings.Contains(docs, required) {
			t.Fatalf("docs component shell marker %q missing: %s", required, docs)
		}
	}
	if strings.Contains(docs, `data-margo-frame="top-left-main-right-footer"`) {
		t.Fatalf("typed docs retained the legacy frame marker: %s", docs)
	}
}

func TestBuildConfigRendersSharedFamilyNavigationAndScopedPagination(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "---\nlayout:\n  kind: landing\n---\n# Tour\n\nChoose a path.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "# Module\n\nModule documentation.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "index.md"), "# CLI\n\nCLI documentation.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "_layout.yaml"), "values:\n  family: module\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "_layout.yaml"), "values:\n  family: cli\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "icon.svg"), "icon.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
base_path: /docs
site:
  name: Margo
  description: Margo documentation
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/icon.svg
  social_image:
    path: assets/social.jpg
    alt: Margo preview
layout:
  kind: docs
  default:
    families: [module, cli]
  values:
    family: default
navigation:
  mode: file-tree
locales:
  default: en
  supported: [en]
theme:
  builtin: true
  name: modern
  color_mode: light
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	pages := map[string]string{
		"index.md":        string(configArtifact(t, result, "index.html")),
		"module/index.md": string(configArtifact(t, result, "module/index.html")),
		"cli/index.md":    string(configArtifact(t, result, "cli/index.html")),
	}
	for source, page := range map[string]string{"module/index.md": pages["module/index.md"], "cli/index.md": pages["cli/index.md"]} {
		for _, required := range []string{`class="component-doc-shell__managed-logo"`, `src="../assets/logo.svg"`, `<link rel="icon" href="../assets/icon.svg"`} {
			if !strings.Contains(page, required) {
				t.Fatalf("%s missing configured brand asset %q: %s", source, required, page)
			}
		}
		if !strings.Contains(page, `data-margo-layout="`) {
			t.Fatalf("%s missing semantic layout hook: %s", source, page)
		}
		if strings.Count(page, `aria-current="location"`) != 2 {
			t.Fatalf("%s has wrong active family count: %s", source, page)
		}
		if strings.Count(page, `data-search-field=""`) != 1 {
			t.Fatalf("%s renders duplicate global search fields: %s", source, page)
		}
		globalStart := strings.Index(page, `class="component-doc-shell__family-links"`)
		if globalStart < 0 {
			t.Fatalf("%s missing public component shell family navigation: %s", source, page)
		}
		globalEnd := strings.Index(page[globalStart:], `</nav>`)
		if globalEnd < 0 {
			t.Fatalf("%s global navigation is not closed: %s", source, page)
		}
		global := page[globalStart : globalStart+globalEnd]
		last := -1
		for _, label := range []string{"Module", "CLI"} {
			index := strings.Index(global, ">"+label+"<")
			if index < 0 || index <= last {
				t.Fatalf("%s global family order missing %s: %s", source, label, global)
			}
			last = index
		}
		for _, route := range []string{"/docs/", "/docs/module/", "/docs/cli/"} {
			if !strings.Contains(page, `data-search-href="`+route+`"`) {
				t.Fatalf("%s global search missing %s: %s", source, route, page)
			}
		}
		if strings.Contains(global, `hx-get=`) || strings.Contains(global, `hx-target=`) {
			t.Fatalf("%s typed family navigation bypasses full shell refresh: %s", source, global)
		}
	}
	landing := pages["index.md"]
	if strings.Contains(landing, `id="left-nav"`) || strings.Contains(landing, `aria-label="sidebar navigation"`) {
		t.Fatalf("landing unexpectedly renders local navigation: %s", landing)
	}
	styles := string(configArtifact(t, result, configuredDocsStylePath))
	for _, required := range []string{`.margo-showcase-article`, `.margo-pagination`, `.margo-page-actions`} {
		if !strings.Contains(styles, required) {
			t.Fatalf("docs stylesheet missing Margo-owned article contract %s", required)
		}
	}
	for _, forbidden := range []string{"margo-frame", "component-doc-shell", "componentdocshell", "grid-template-areas"} {
		if strings.Contains(styles, forbidden) {
			t.Fatalf("docs stylesheet retains shell-owned selector %q", forbidden)
		}
	}
	for _, asset := range []string{
		"margo-assets/goshtoso/shell.css",
		"margo-assets/goshtoso/shell.js",
		"assets/styles.css",
		"assets/js/goshtoso.min.js",
		"assets/js/runtime/alpinejs/3.14.9/alpine.min.js",
	} {
		if len(configArtifact(t, result, asset)) == 0 {
			t.Fatalf("docs navigation asset %q missing", asset)
		}
	}
	if artifactExists(result, "margo-assets/goshtoso/margo-scroll-spy.js") {
		t.Fatal("typed docs staged Margo's legacy TOC runtime")
	}
	for source, family := range map[string]string{"module/index.md": "Module", "cli/index.md": "CLI"} {
		page := pages[source]
		leftStart := strings.Index(page, `data-sidebar-section="`+family+`"`)
		leftEnd := strings.Index(page[leftStart:], `</nav>`)
		if leftStart < 0 || leftEnd < 0 {
			t.Fatalf("%s missing active-family public sidebar: %s", source, page)
		}
		left := page[leftStart : leftStart+leftEnd]
		if !strings.Contains(left, `>`+family+`<`) {
			t.Fatalf("%s sidebar missing active-family overview: %s", source, left)
		}
		for _, other := range []string{"Tour", "Module", "CLI"} {
			if other != family && strings.Contains(left, `>`+other+`<`) {
				t.Fatalf("%s sidebar leaked %s: %s", source, other, left)
			}
		}
		if strings.Contains(page, `class="margo-pagination"`) {
			t.Fatalf("%s renders pagination for one-page family: %s", source, page)
		}
	}
}

func TestBuildConfigRendersLayoutSemanticChromePresenceAndAbsence(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "---\nlayout:\n  kind: landing\n---\n# Tour\n\nChoose a documentation family.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "# Module\n\nModule overview.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "guide.md"), "# Module guide\n\nModule detail.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "index.md"), "# CLI\n\nCLI overview.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "guide.md"), "# CLI guide\n\nCLI detail.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "_layout.yaml"), "values:\n  family: module\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "_layout.yaml"), "values:\n  family: cli\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
site:
  name: Margo
  description: Layout semantic fixture.
  repository_url: https://github.com/araihu/margo
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo preview.
layout:
  kind: docs
  default:
    families: [module, cli]
  values:
    family: default
navigation:
  mode: file-tree
locales:
  default: en
  supported: [en]
theme:
  builtin: true
  name: modern
  color_mode: system
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	pages := map[string]string{
		"Tour":   string(configArtifact(t, result, "index.html")),
		"Module": string(configArtifact(t, result, "module/index.html")),
		"CLI":    string(configArtifact(t, result, "cli/index.html")),
	}
	for name, page := range map[string]string{"Module": pages["Module"], "CLI": pages["CLI"]} {
		if strings.Count(page, `aria-current="location"`) != 2 || !strings.Contains(page, `id="componentdocshell-family-navigation"`) {
			t.Fatalf("%s public family navigation state is missing: %s", name, page)
		}
		if !strings.Contains(page, `component-doc-shell__repository`) || !strings.Contains(page, `aria-label="Source repository"`) || !strings.Contains(page, `<svg`) {
			t.Fatalf("%s repository action is not an accessible icon link: %s", name, page)
		}
		if strings.Contains(page, `data-margo-repository-link="true"`) {
			t.Fatalf("%s repository action still uses the removed Margo hook: %s", name, page)
		}
		if !strings.Contains(page, `class="component-doc-shell`) || !strings.Contains(page, `data-componentdocshell-toc`) {
			t.Fatalf("%s docs output is not rendered by componentdocshell: %s", name, page)
		}
	}
	tour := pages["Tour"]
	for _, forbidden := range []string{`data-margo-layout="docs"`, `id="left-nav"`, `id="right-nav"`, `aria-label="sidebar navigation"`, `class="margo-pagination"`, `data-toc-heading`} {
		if strings.Contains(tour, forbidden) {
			t.Fatalf("Tour unexpectedly contains %q: %s", forbidden, tour)
		}
	}
	for name, family := range map[string]string{"Module": "Module", "CLI": "CLI"} {
		page := pages[name]
		if strings.Contains(page, `margo-breadcrumbs`) || strings.Contains(page, `aria-label="Breadcrumbs"`) {
			t.Fatalf("%s renders a breadcrumb in docs chrome: %s", name, page)
		}
		for _, required := range []string{
			`data-margo-layout="docs"`,
			`class="component-doc-shell`,
			`id="componentdocshell-sidebar"`,
			`aria-label="sidebar navigation"`,
			`data-sidebar-section="` + family + `"`,
			`aria-current="page"`,
			`class="margo-pagination"`,
			`id="componentdocshell-toc"`,
			`data-componentdocshell-toc`,
		} {
			if !strings.Contains(page, required) {
				t.Fatalf("%s missing semantic docs output %q: %s", name, required, page)
			}
		}
		if strings.Contains(page, `data-margo-toc-drawer`) || strings.Contains(page, `id="right-nav"`) {
			t.Fatalf("%s retains the removed custom TOC drawer: %s", name, page)
		}
	}
	styles := string(configArtifact(t, result, configuredDocsStylePath))
	for _, required := range []string{`.margo-showcase-article`, `.margo-pagination`, `.margo-page-actions`} {
		if !strings.Contains(styles, required) {
			t.Fatalf("docs stylesheet missing article/action constraint %q: %s", required, styles)
		}
	}
	for _, forbidden := range []string{"margo-frame", "data-margo-toc-drawer", "component-doc-shell", "grid-template-areas"} {
		if strings.Contains(styles, forbidden) {
			t.Fatalf("docs stylesheet retains shell-owned selector %q: %s", forbidden, styles)
		}
	}
	searchInteractions := string(configArtifact(t, result, searchInteractionsScriptPath))
	for _, forbidden := range []string{"data-margo-toc", "initTOCDrawer", "componentdocshell-sidebar"} {
		if strings.Contains(searchInteractions, forbidden) {
			t.Fatalf("search enhancer retains shell navigation behavior %q: %s", forbidden, searchInteractions)
		}
	}
}

func TestBuildConfigRendersLocaleScopedFamilySearch(t *testing.T) {
	root := t.TempDir()
	for _, localePrefix := range []string{"", "pt-BR"} {
		prefix := filepath.Join(root, "docs", localePrefix)
		writeConfigFile(t, filepath.Join(prefix, "index.md"), "---\nlayout:\n  kind: landing\n---\n# Tour\n\nTour documentation. [Module](module/index.md).\n")
		writeConfigFile(t, filepath.Join(prefix, "module", "index.md"), "---\nlayout:\n  values:\n    family: module\n---\n# Module\n\nModule documentation.\n")
		writeConfigFile(t, filepath.Join(prefix, "cli", "index.md"), "---\nlayout:\n  values:\n    family: cli\n---\n# CLI\n\nCLI documentation.\n")
	}
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
base_path: /docs
site:
  name: Margo
  description: Margo documentation
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo preview
layout:
  kind: docs
  default:
    families: [module, cli]
  values:
    family: default
navigation:
  mode: file-tree
locales:
  default: en
  supported: [en, pt-BR]
theme:
  builtin: true
  name: modern
  color_mode: light
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		name        string
		artifact    string
		landing     string
		localRoutes []string
		otherRoutes []string
	}{
		{
			name:        "English",
			artifact:    "module/index.html",
			landing:     "index.html",
			localRoutes: []string{"/docs/", "/docs/module/", "/docs/cli/"},
			otherRoutes: []string{"/docs/pt-br/", "/docs/pt-br/module/", "/docs/pt-br/cli/"},
		},
		{
			name:        "Portuguese",
			artifact:    "pt-br/module/index.html",
			landing:     "pt-br/index.html",
			localRoutes: []string{"/docs/pt-br/", "/docs/pt-br/module/", "/docs/pt-br/cli/"},
			otherRoutes: []string{"/docs/", "/docs/module/", "/docs/cli/"},
		},
	} {
		page := string(configArtifact(t, result, fixture.artifact))
		canonical := `rel="canonical" href="https://margo.example` + fixture.localRoutes[1] + `"`
		if !strings.Contains(page, canonical) {
			t.Fatalf("%s canonical missing public route %s: %s", fixture.name, canonical, page)
		}
		for _, route := range fixture.localRoutes {
			if !strings.Contains(page, `data-search-href="`+route+`"`) {
				t.Fatalf("%s search missing same-locale route %s: %s", fixture.name, route, page)
			}
		}
		for _, route := range fixture.otherRoutes {
			if strings.Contains(page, `data-search-href="`+route+`"`) {
				t.Fatalf("%s search leaked other-locale route %s: %s", fixture.name, route, page)
			}
		}
		for _, family := range []struct {
			id   string
			href string
		}{
			{id: "module", href: fixture.localRoutes[1]},
			{id: "cli", href: fixture.localRoutes[2]},
		} {
			link := `class="component-doc-shell__family-link" href="` + family.href + `"`
			if !strings.Contains(page, link) {
				t.Fatalf("%s global family navigation missing locale-owned overview %s: %s", fixture.name, link, page)
			}
		}
		landing := string(configArtifact(t, result, fixture.landing))
		if !strings.Contains(landing, `href="`+fixture.localRoutes[1]+`"`) {
			t.Fatalf("%s rewritten Markdown link missing public route %s: %s", fixture.name, fixture.localRoutes[1], landing)
		}
	}
	sitemap := string(configArtifact(t, result, SitemapPath))
	for _, route := range []string{"https://margo.example/docs/module/", "https://margo.example/docs/pt-br/module/"} {
		if !strings.Contains(sitemap, `<loc>`+route+`</loc>`) {
			t.Fatalf("localized sitemap missing public route %s: %s", route, sitemap)
		}
	}
}

func TestBuildConfigFrontmatterKindOverrideWinsOverSiteDefault(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Landing\n\nChoose a path.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "---\nlayout:\n  kind: landing\n---\n# Module\n\nModule documentation.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
site:
  name: Margo
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo preview
layout:
  kind: docs
theme:
  builtin: true
  name: modern
  color_mode: light
`)
	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	for _, page := range result.Site.Routes {
		if page.Source == "module/index.md" && (page.Family != "" || page.Layout != "landing") {
			t.Fatalf("page override identity = %+v", page)
		}
	}
	if page := string(configArtifact(t, result, "module/index.html")); !strings.Contains(page, `data-margo-frame="main"`) {
		t.Fatalf("page override did not select landing frame: %s", page)
	}
}

func TestBuildConfigLayoutPreflightInvalidSelectedKindDoesNotEmitHTML(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Landing\n\nChoose a path.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "---\nlayout:\n  kind: missing\n---\n# Module\n\nModule documentation.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
site:
  name: Margo
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo preview
layout:
  kind: docs
theme:
  builtin: true
  name: modern
  color_mode: light
`)
	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err == nil {
		t.Fatal("expected invalid page layout diagnostic")
	}
	if len(result.Artifacts) != 0 {
		t.Fatalf("invalid layout kind emitted artifacts: %+v", result.Artifacts)
	}
	var diagnosticError *margo.DiagnosticError
	if !errors.As(err, &diagnosticError) || len(diagnosticError.Diagnostics) != 1 || diagnosticError.Diagnostics[0].Code != "site.layout_unknown" {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildConfigTypedLayoutIdentityMatchesRoutes(t *testing.T) {
	config := Config{
		Version: 1, Source: "docs", Output: "dist", Assets: string(AssetsLocal),
		Site:    SiteConfig{Name: "Margo", BaseURL: "https://margo.example", Home: "index.md"},
		Locales: LocaleConfig{Default: "en", Supported: []string{"en"}},
		Theme:   ThemeSelection{Name: "modern", ColorMode: "light"},
		Layout: &LayoutConfig{Kind: LayoutDocs, Default: map[string]any{
			"families": []any{"default", "module"},
		}, Values: map[string]any{"family": "default"}},
	}
	siteCascade, err := resolveSiteLayout(*config.Layout, "")
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{
		request: Request{Compiler: margo.New()}, config: &config, sourceDir: t.TempDir(),
		layoutPatches: []LayoutPatch{{Source: "module/_layout.yaml", Values: map[string]any{"family": "module"}}},
		configured:    map[string]configuredPage{},
	}
	sources := []Source{
		{Path: "index.md", Content: []byte("---\nlayout:\n  kind: landing\n---\n# Landing\n\nChoose a path.\n")},
		{Path: "module/index.md", Content: []byte("# Module\n\nModule documentation.\n")},
	}
	if err := b.preflightConfigured(context.Background(), sources); err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		family string
		layout string
	}{
		"index.md":        {family: "", layout: "landing"},
		"module/index.md": {family: "module", layout: "docs"},
	}
	for source, expected := range want {
		prepared, ok := b.configured[source]
		if !ok {
			t.Fatalf("configured page %q missing", source)
		}
		if prepared.page.Family != expected.family || prepared.page.Layout != expected.layout {
			t.Fatalf("route %q = family %q layout %q, want family %q layout %q", source, prepared.page.Family, prepared.page.Layout, expected.family, expected.layout)
		}
	}
	if got := siteCascade.resolved().Kind; got != LayoutDocs {
		t.Fatalf("site layout kind = %q, want docs", got)
	}
}

func TestBuildConfigRendersGoshtosoComponentDocShell(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "showcase", "index.md"), "# Showcase\n\nA public feature tour.\n\n## A section\n\nA section for the shell TOC.\n")
	writeConfigFile(t, filepath.Join(root, "showcase", "markdown.md"), "# Markdown\n\nThe compiler path.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "icon.svg"), "icon.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: showcase
site:
  name: Margo
  description: A public feature tour.
  version: v0.0.5
  repository_url: https://github.com/araihu/margo
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/icon.svg
  social_image:
    path: assets/social.jpg
    alt: Margo showcase preview.
shell:
  builtin: componentdocshell
locales:
  default: en
  supported: [en]
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	page := string(configArtifact(t, result, "index.html"))
	if result.Site.Layout != "shell:componentdocshell" || len(result.Site.Routes) != 2 {
		t.Fatalf("shell manifest = %+v", result.Site)
	}
	for _, required := range []string{
		`component-doc-shell__header`, `component-doc-shell__sidebar`, `component-doc-shell__toc`,
		`margo-shell-search`, `href="/markdown.html"`, `margo-assets/goshtoso/shell.js`,
		`assets/js/goshtoso.min.js`, `component-doc-shell__brand-badge`, `v0.0.5`,
		`component-doc-shell__repository`, `https://github.com/araihu/margo`,
		`hx-get="/markdown.html"`, `hx-select="#main-content"`, `hx-target="#main-content"`,
		`hx-swap="outerHTML transition:true swap:160ms settle:240ms"`, `hx-push-url="true"`,
		`hx-swap-oob="outerHTML:#componentdocshell-sidebar-content"`,
		`data-margo-navigation="true"`,
		`data-search-id="margo-doc-search"`, `data-search-global-shortcut="true"`, `<kbd`, `⌘ K`,
		`data-search-title="Markdown"`, `data-search-href="/markdown.html"`,
		`data-toc-heading`, `src="margo-assets/goshtoso/margo-scroll-spy.js"`,
		`Margo · <strong>Buildt</strong> by `,
		`href="https://araihu.com/"`, `<strong>Arai Hû</strong>`,
		`href="https://goshtoso.araihu.com/"`, `>Goshtoso</a>`,
		`href="/llms.txt"><strong>llms.txt</strong>`, `href="/sitemap.xml"><strong>sitemap.xml</strong>`,
		`<title>Showcase</title>`, `<meta name="description" content="A public feature tour."`,
		`<link rel="canonical" href="https://margo.example/"`, `property="og:image"`,
		`name="twitter:card" content="summary_large_image"`,
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("shell page missing %q: %s", required, page)
		}
	}
	document, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	searchField := configElementByID(document, "margo-doc-search")
	if searchField == nil || !configNodeHasAncestorClass(searchField, "component-doc-shell__controls") {
		t.Fatalf("search field is not in the shell header controls: %s", page)
	}
	if configNodeHasAncestorClass(searchField, "component-doc-shell__sidebar") {
		t.Fatalf("search field remains in the shell sidebar: %s", page)
	}
	if strings.Contains(page, `margo-shell-topnav`) {
		t.Fatalf("shell still renders the removed primary-link navigation: %s", page)
	}
	searchModalIndex := strings.Index(page, `id="margo-doc-search-dialog"`)
	mainEndIndex := strings.Index(page, `</main>`)
	if searchModalIndex == -1 || mainEndIndex == -1 || searchModalIndex < mainEndIndex {
		t.Fatalf("search modal is not mounted outside the shell content: modal=%d main-end=%d", searchModalIndex, mainEndIndex)
	}
	resources := componentDocShellHeadResources(t, page)
	shellScript := -1
	firstPageDependency := -1
	for index, resource := range resources {
		if shellScript == -1 && strings.Contains(resource.url, "margo-assets/goshtoso/shell.js") {
			shellScript = index
		}
		if firstPageDependency == -1 && resource.requirement != "" {
			firstPageDependency = index
		}
	}
	if shellScript == -1 || firstPageDependency == -1 || shellScript > firstPageDependency {
		t.Fatalf("page dependencies run before the component shell: shell=%d dependency=%d resources=%+v", shellScript, firstPageDependency, resources)
	}
	for _, required := range []string{`class="component-doc-shell__managed-logo"`, `src="assets/logo.svg"`, `<link rel="icon" href="assets/icon.svg"`} {
		if !strings.Contains(page, required) {
			t.Fatalf("shell page missing configured brand asset %q: %s", required, page)
		}
	}
	footerStart := strings.Index(page, `<p class="margo-shell-footer">`)
	if footerStart == -1 {
		t.Fatalf("shell footer missing: %s", page)
	}
	footerEnd := strings.Index(page[footerStart:], `</p>`)
	if footerEnd == -1 {
		t.Fatalf("shell footer is not closed: %s", page)
	}
	footer := page[footerStart : footerStart+footerEnd]
	if strings.Count(footer, `<a `) != 4 || strings.Count(footer, ` text-primary `) != 4 {
		t.Fatalf("shell footer links are not Goshtoso Link components: %s", footer)
	}
	if strings.Count(footer, `target="_blank"`) != 2 || strings.Count(footer, `rel="noopener noreferrer"`) != 2 {
		t.Fatalf("shell footer external-link contract missing: %s", footer)
	}
	for _, asset := range []string{
		"margo-assets/goshtoso/shell.css", "margo-assets/goshtoso/shell.js", "margo-assets/goshtoso/margo-scroll-spy.js", "assets/styles.css",
	} {
		if len(configArtifact(t, result, asset)) == 0 {
			t.Fatalf("shell asset %q missing", asset)
		}
	}
	scrollSpy := string(configArtifact(t, result, "margo-assets/goshtoso/margo-scroll-spy.js"))
	for _, required := range []string{
		"IntersectionObserver", "history.replaceState", "htmx:afterSwap", "htmx:afterSettle", "htmx:historyRestore",
		"window.addEventListener(\"componentdocshell:navigated\"", "observerGeneration", "visibleHeadings", "entry.time", "explicitLock", "explicitRestore", "headingIsVisible", "automaticHold",
		"data-margo-toc-active", "aria-current", "scrollend", "renderMermaidAfterSwap", "margoRunMermaid", "mermaid.min.js",
	} {
		if !strings.Contains(scrollSpy, required) {
			t.Fatalf("scroll-spy asset missing %q: %s", required, scrollSpy)
		}
	}
	if strings.Contains(scrollSpy, `document.addEventListener("componentdocshell:navigated"`) {
		t.Fatal("scroll-spy listens for shell navigation on document instead of window")
	}
	if strings.Contains(scrollSpy, `rootMargin: "0px 0px -65% 0px"`) || strings.Contains(scrollSpy, "visible.sort") {
		t.Fatal("scroll-spy still uses the old topmost-heading algorithm")
	}
	afterSwap := strings.Index(scrollSpy, `document.addEventListener("htmx:afterSwap"`)
	afterSettle := strings.Index(scrollSpy, `document.addEventListener("htmx:afterSettle"`)
	if afterSwap == -1 || afterSettle == -1 || afterSettle < afterSwap {
		t.Fatalf("scroll-spy Mermaid lifecycle ordering is invalid: afterSwap=%d afterSettle=%d", afterSwap, afterSettle)
	}
	if strings.Contains(scrollSpy[afterSwap:afterSettle], "queueMermaidRender()") {
		t.Fatal("scroll-spy queues Mermaid before HTMX settles the swapped content")
	}
	styles := string(configArtifact(t, result, "margo-assets/site.css"))
	for _, required := range []string{
		".margo-shell-search {",
		".component-doc-shell__brand-mark { display: none !important; }",
		".margo-showcase-article .margo-pagination ul { justify-content: space-between; column-gap: 2rem; row-gap: 0.75rem; }",
		".margo-showcase-article .margo-pagination a { color: var(--margo-accent);",
		":where(.margo-frame button, .margo-frame a) { min-block-size: 2.75rem; }",
		":where(.margo-frame a, .margo-document a) { color: var(--margo-accent); }",
		"view-transition-name: margo-main-content",
		"prefers-reduced-motion: reduce",
	} {
		if !strings.Contains(styles, required) {
			t.Fatalf("shell ordering CSS missing %q: %s", required, styles)
		}
	}
	if strings.Contains(styles, ".margo-shell-search { order:") {
		t.Fatalf("search uses CSS order instead of matching header DOM order: %s", styles)
	}
	for _, reordered := range []string{"#componentdocshell-dark-mode { order:", ".component-doc-shell__repository { order:"} {
		if strings.Contains(styles, reordered) {
			t.Fatalf("header utility uses CSS order instead of matching DOM order %q: %s", reordered, styles)
		}
	}
	if strings.Contains(styles, ".margo-shell-topnav") {
		t.Fatalf("site stylesheet still contains removed top navigation rules: %s", styles)
	}
	for _, forbidden := range []string{"\nbutton, a {", "\nbutton {", "\n:focus-visible {", "margo-landing", `data-margo-layout="landing"`, `[alt^=`} {
		if strings.Contains(styles, forbidden) {
			t.Fatalf("shell CSS leaks an unscoped control rule %q: %s", forbidden, styles)
		}
	}
}

func TestTypedDocsLayoutUsesComponentDocShellPublicFrame(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "---\nlayout:\n  kind: landing\n---\n# Tour\n\nWelcome.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "---\nlayout:\n  values:\n    family: module\n---\n# Module\n\n## Usage\n\nDocs page.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
site:
  name: Margo
  description: Typed docs shell fixture.
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo preview
layout:
  kind: docs
  default:
    families: [module]
locales:
  default: en
  supported: [en]
theme:
  builtin: true
  name: modern
  color_mode: light
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	docs := string(configArtifact(t, result, "module/index.html"))
	for _, required := range []string{
		`class="component-doc-shell`, `id="componentdocshell-sidebar"`,
		`data-componentdocshell-toc`, `data-margo-layout="docs"`,
		`margo-assets/goshtoso/shell.css`, `margo-assets/goshtoso/shell.js`,
	} {
		if !strings.Contains(docs, required) {
			t.Fatalf("typed docs output missing %q: %s", required, docs)
		}
	}
	for _, forbidden := range []string{`class="margo-frame`, `data-margo-toc-drawer`, `data-margo-global-navigation`} {
		if strings.Contains(docs, forbidden) {
			t.Fatalf("typed docs output retained custom shell marker %q: %s", forbidden, docs)
		}
	}
}

func TestTypedDocsSidebarFalseUsesSemanticComponentShellBridge(t *testing.T) {
	document := []byte(`<!doctype html><html lang="pt_BR"><body><button class="component-doc-shell__menu-button"></button><div id="componentdocshell-sidebar"></div><div class="component-doc-shell__backdrop"></div><main id="page-scroll"><div id="main-content">content</div></main></body></html>`)
	got, err := applyTypedComponentDocShellSemantics(document, Page{Locale: "pt-BR"}, false)
	if err != nil {
		t.Fatal(err)
	}
	page := string(got)
	for _, required := range []string{`lang="pt-BR"`, `dir="ltr"`, `data-margo-layout="docs"`, `data-margo-sidebar="false"`, `id="main-content"`} {
		if !strings.Contains(page, required) {
			t.Fatalf("semantic shell bridge missing %q: %s", required, page)
		}
	}
	for _, forbidden := range []string{"component-doc-shell__menu-button", "componentdocshell-sidebar", "component-doc-shell__backdrop"} {
		if strings.Contains(page, forbidden) {
			t.Fatalf("semantic shell bridge retained %q: %s", forbidden, page)
		}
	}
}

func configElementByID(root *html.Node, id string) *html.Node {
	if root.Type == html.ElementNode && attributeValue(root, "id") == id {
		return root
	}
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if found := configElementByID(child, id); found != nil {
			return found
		}
	}
	return nil
}

func configNodeHasAncestorClass(node *html.Node, className string) bool {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if hasClass(parent, className) {
			return true
		}
	}
	return false
}

func TestBuildConfigComponentDocShellUsesConsumerLocaleAndRoutes(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Início\n\nDocumentação do fornecedor.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "vendor.md"), "# Fornecedor\n\nPágina do fornecedor.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
site:
  name: Vendor Docs
  description: Documentação do fornecedor.
  base_url: https://vendor.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Prévia da documentação.
shell:
  builtin: componentdocshell
locales:
  default: pt-BR
  supported: [pt-BR]
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	page := string(configArtifact(t, result, "index.html"))
	for _, required := range []string{
		`<html lang="pt-BR"`,
		`dir="ltr"`,
		`<title>Início</title>`,
		`>Ir para o conteúdo</a>`,
		`aria-label="Abrir navegação"`,
		`sidebarOpen ? &#39;Fechar navegação&#39; : &#39;Abrir navegação&#39;`,
		`aria-label="Vendor Docs início"`,
		`aria-label="Navegação lateral"`,
		`>ativa</span>`,
		`aria-label="Usar modo escuro"`,
		`dark ? &#39;Usar modo claro&#39; : &#39;Usar modo escuro&#39;`,
		`aria-label="Nesta página"`,
		`>Nesta página</p>`,
		`>Buscar páginas</span>`,
		`>Buscar páginas</label>`,
		`placeholder="Buscar páginas"`,
		`aria-label="Resultados da busca"`,
		`href="/vendor.html"`,
		`>Fornecedor</span>`,
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("consumer shell missing %q: %s", required, page)
		}
	}
	for _, forbidden := range []string{"Markdown to durable outputs", "Margo features", ">Features<", "Search features"} {
		if strings.Contains(page, forbidden) {
			t.Fatalf("consumer shell leaked showcase content %q: %s", forbidden, page)
		}
	}
}

func TestBuildConfigRendersInlineShellDependenciesWithBasePath(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "showcase", "index.md"), "# Home\n\nThe home page owns a Mermaid runtime dependency.\n\n```mermaid\nflowchart LR\n  source[Markdown] --> output[HTML]\n```\n")
	writeConfigFile(t, filepath.Join(root, "showcase", "guide.md"), "# Guide\n\nThe guide has no page-specific runtime dependency.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: showcase
assets: inline
base_path: /docs
site:
  name: Margo
  description: An inline shell fixture.
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo inline shell fixture.
shell:
  builtin: componentdocshell
locales:
  default: en
  supported: [en]
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	if result.Site.BasePath != "/docs" {
		t.Fatalf("base path = %q, want /docs", result.Site.BasePath)
	}
	index := string(configArtifact(t, result, "index.html"))
	guide := string(configArtifact(t, result, "guide.html"))
	for _, required := range []string{
		`hx-get="/docs/guide.html"`,
		`data-margo-requirement="margo.mermaid.runtime"`,
		`data-margo-requirement="margo.mermaid.execute"`,
	} {
		if !strings.Contains(index, required) {
			t.Fatalf("inline shell page missing %q: %s", required, index)
		}
	}
	if strings.Contains(guide, `data-margo-requirement="margo.mermaid.runtime"`) {
		t.Fatalf("page-specific Mermaid dependency leaked into guide: %s", guide)
	}
	resources := componentDocShellHeadResources(t, index)
	shellScript := -1
	mermaidRequirement := -1
	for resourceIndex, resource := range resources {
		if shellScript == -1 && strings.Contains(resource.url, "margo-assets/goshtoso/shell.js") {
			shellScript = resourceIndex
		}
		if mermaidRequirement == -1 && resource.requirement == "margo.mermaid.runtime" {
			mermaidRequirement = resourceIndex
		}
	}
	if shellScript == -1 || mermaidRequirement == -1 || shellScript > mermaidRequirement {
		t.Fatalf("inline dependency order is invalid: shell=%d mermaid=%d resources=%+v", shellScript, mermaidRequirement, resources)
	}
	if artifactExists(result, "margo-assets/mermaid/11.16.1/mermaid.min.js") || artifactExists(result, "margo-assets/runtime/mermaid-run.js") {
		t.Fatal("inline shell unexpectedly publishes Mermaid runtime artifacts")
	}
}

type shellHeadResource struct {
	url         string
	requirement string
}

func componentDocShellHeadResources(t *testing.T, document string) []shellHeadResource {
	t.Helper()
	root, err := html.Parse(strings.NewReader(document))
	if err != nil {
		t.Fatalf("parse generated document: %v", err)
	}
	var head *html.Node
	var findHead func(*html.Node)
	findHead = func(node *html.Node) {
		if head != nil {
			return
		}
		if node.Type == html.ElementNode && node.Data == "head" {
			head = node
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			findHead(child)
		}
	}
	findHead(root)
	if head == nil {
		t.Fatal("generated document has no head")
	}

	resources := make([]shellHeadResource, 0)
	var collect func(*html.Node)
	collect = func(node *html.Node) {
		if node.Type == html.ElementNode && (node.Data == "script" || node.Data == "link") {
			resourceURL := attributeValue(node, "src")
			if resourceURL == "" {
				resourceURL = attributeValue(node, "href")
			}
			resources = append(resources, shellHeadResource{
				url:         resourceURL,
				requirement: attributeValue(node, "data-margo-requirement"),
			})
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			collect(child)
		}
	}
	collect(head)
	return resources
}

func TestBuildConfigUsesSelectedCustomThemeInGoshtosoComponentDocShell(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "showcase", "index.md"), "# Showcase\n\nA themed feature tour.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	themeCSS := []byte(`[data-theme="margo"] { --color-surface: #fcfaf4; }`)
	writeConfigFile(t, filepath.Join(root, "themes", "margo.css"), string(themeCSS))
	digest := sha256.Sum256(themeCSS)
	writeConfigFile(t, filepath.Join(root, "themes", "margo.tokens.yaml"), `schema: margo.theme.tokens/v1
css_digest: sha256-`+hex.EncodeToString(digest[:])+`
fonts: []
minimum_text_size: 1rem
touch_target: {min_css_px: 44}
typography:
  display: {}
  headline: {}
  title: {}
  body: {}
  label: {}
  caption: {alias_of: label}
layout: {}
spacing: {}
breakpoints: []
grid: {}
drawer: {}
colors: {light: {}, dark: {}}
states: {light: {}, dark: {}}
feedback: {light: {}, dark: {}}
contrast_pairs: {}
`)
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: showcase
site:
  name: Margo
  description: A themed feature tour.
  version: v0.0.5
  repository_url: https://github.com/araihu/margo
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo themed showcase preview.
shell:
  builtin: componentdocshell
themes:
  - name: margo
    css_url: themes/margo.css
    token_catalog: themes/margo.tokens.yaml
theme:
  builtin: false
  name: margo
  color_mode: dark
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	page := string(configArtifact(t, result, "index.html"))
	for _, required := range []string{
		`"theme":"margo"`,
		`"colorScheme":"dark"`,
		`<link rel="stylesheet" href="themes/margo.css"/>`,
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("custom shell theme missing %q: %s", required, page)
		}
	}
	if strings.Contains(page, `araihu.css`) {
		t.Fatalf("custom shell theme still loads the Arai Hû stylesheet: %s", page)
	}
	if len(configArtifact(t, result, "themes/margo.css")) == 0 {
		t.Fatal("custom shell theme stylesheet was not staged")
	}
}

func TestLoadConfigRejectsUnknownKeys(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "site.yaml"), "version: 1\nunknown: true\n")
	if _, err := LoadConfig(filepath.Join(root, "site.yaml")); err == nil || !strings.Contains(err.Error(), "site.config_invalid") {
		t.Fatalf("unknown key error = %v", err)
	}
}

func TestBuildConfigResolvesRootBasePathAndLocaleRoutes(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Home\n\nEnglish home.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "guide.md"), "# Guide\n\nEnglish guide.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "pt-BR", "index.md"), "# Início\n\nPágina inicial.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "pt-BR", "guide.md"), "# Guia\n\nPágina do guia.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
site:
  name: Margo
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo documentation preview
locales:
  default: en
  supported: [en, pt-BR]
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	index := string(configArtifact(t, result, "index.html"))
	ptGuide := string(configArtifact(t, result, "pt-br/guide.html"))
	if !strings.Contains(index, `property="og:image" content="https://margo.example/assets/social.jpg"`) || strings.Contains(index, `https://margo.example//assets/`) {
		t.Fatalf("root social URL is not canonical: %s", index)
	}
	if !strings.Contains(index, `href="pt-br/index.html"`) {
		t.Fatalf("root locale switch is not relative: %s", index)
	}
	if !strings.Contains(ptGuide, `href="https://margo.example/pt-br/guide.html"`) || !strings.Contains(ptGuide, `href="../guide.html"`) || !strings.Contains(ptGuide, `href="index.html">Início`) {
		t.Fatalf("locale route metadata is not coherent: %s", ptGuide)
	}
	if strings.Contains(index, `>Início</a>`) || strings.Contains(index, `>Guia</a>`) {
		t.Fatalf("English navigation leaked Portuguese routes: %s", index)
	}
}

func TestBuildConfigStagesSelectedThemeAndBootstrap(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Home\n\nTheme fixture.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	themeCSS := []byte(`:root { --margo-accent: rebeccapurple; }`)
	writeConfigFile(t, filepath.Join(root, "themes", "custom.css"), string(themeCSS))
	digest := sha256.Sum256(themeCSS)
	writeConfigFile(t, filepath.Join(root, "themes", "custom.tokens.yaml"), `schema: margo.theme.tokens/v1
css_digest: sha256-`+hex.EncodeToString(digest[:])+`
fonts: []
minimum_text_size: 1rem
touch_target: {min_css_px: 44}
typography:
  display: {}
  headline: {}
  title: {}
  body: {}
  label: {}
  caption: {alias_of: label}
layout: {}
spacing: {}
breakpoints: []
grid: {}
drawer: {}
colors: {light: {}, dark: {}}
states: {light: {}, dark: {}}
feedback: {light: {}, dark: {}}
contrast_pairs: {}
`)
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
site:
  name: Margo
  base_url: https://margo.example
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo documentation preview
themes:
  - name: custom_theme
    css_url: themes/custom.css
    token_catalog: themes/custom.tokens.yaml
theme:
  name: custom_theme
  allow_switch_theme: true
  color_mode: dark
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	page := string(configArtifact(t, result, "index.html"))
	if !strings.Contains(page, `data-theme="custom_theme"`) || !strings.Contains(page, `data-color-mode="dark"`) || !strings.Contains(page, `data-margo-theme-css="custom_theme"`) || !strings.Contains(page, `data-margo-theme-bootstrap`) {
		t.Fatalf("theme bootstrap or stylesheet missing: %s", page)
	}
	if len(configArtifact(t, result, "themes/custom.tokens.yaml")) == 0 || result.Site.DocumentStyleDigest == "" {
		t.Fatalf("theme provenance missing: %+v", result.Site)
	}
}

func TestBuildConfigLayoutCascadeAppliesSiteRootNearestAndFrontmatter(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Root\n\nRoot page.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "# Module\n\nNested page.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "guide.md"), `---
layout:
  values:
    family: markdown
    sidebar: true
---
# Guide

Markdown-owned page.
`)
	writeConfigFile(t, filepath.Join(root, "docs", "_layout.yaml"), `values:
  family: root
  sidebar: false
`)
	writeConfigFile(t, filepath.Join(root, "docs", "module", "_layout.yaml"), `values:
  family: nested
  toc: false
`)
	writeTypedLayoutBuildConfig(t, root, `layout:
  kind: docs
  default:
    families: [default, root, nested, markdown]
    sidebar: true
    toc: true
  values:
    family: default
`)

	preflight := preflightTypedLayoutBuild(t, filepath.Join(root, "site.yaml"))
	want := map[string]struct {
		family  string
		sidebar bool
		toc     bool
		sources []string
	}{
		"index.md":        {family: "root", sidebar: false, toc: true, sources: []string{"_layout.yaml"}},
		"module/index.md": {family: "nested", sidebar: false, toc: false, sources: []string{"_layout.yaml", "module/_layout.yaml"}},
		"module/guide.md": {family: "markdown", sidebar: true, toc: false, sources: []string{"_layout.yaml", "module/_layout.yaml", "module/guide.md"}},
	}
	for source, expected := range want {
		prepared := preflight.configured[source]
		if prepared.layout.Kind != LayoutDocs || prepared.layout.Family != expected.family || prepared.page.Layout != "docs" || prepared.page.Family != expected.family {
			t.Fatalf("%s layout = %+v page = %+v", source, prepared.layout, prepared.page)
		}
		if prepared.layout.Values["sidebar"] != expected.sidebar || prepared.layout.Values["toc"] != expected.toc {
			t.Fatalf("%s values = %#v, want sidebar=%t toc=%t", source, prepared.layout.Values, expected.sidebar, expected.toc)
		}
		if !reflect.DeepEqual(prepared.layoutSources, expected.sources) {
			t.Fatalf("%s sources = %q, want %q", source, prepared.layoutSources, expected.sources)
		}
		if prepared.layout.Identity == "" {
			t.Fatalf("%s layout identity is empty", source)
		}
	}
	if preflight.siteManifest.LayoutSchemaHash == "" {
		t.Fatal("preflight layout schema hash is empty")
	}

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	if result.Site.LayoutSchemaHash != preflight.siteManifest.LayoutSchemaHash {
		t.Fatalf("build hash = %q, preflight hash = %q", result.Site.LayoutSchemaHash, preflight.siteManifest.LayoutSchemaHash)
	}
	for _, page := range result.Site.Routes {
		expected := want[page.Source]
		if page.Layout != "docs" || page.Family != expected.family {
			t.Fatalf("route = %+v, want docs family %q", page, expected.family)
		}
	}
}

func TestBuildConfigLayoutKindBoundaryRestoresDocsValues(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Landing\n\nLanding page.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), `---
layout:
  values:
    sidebar: true
---
# Module

Docs page.
`)
	writeConfigFile(t, filepath.Join(root, "docs", "_layout.yaml"), "kind: landing\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "_layout.yaml"), "kind: docs\n")
	writeTypedLayoutBuildConfig(t, root, `layout:
  kind: docs
  default:
    families: [default, module]
    sidebar: false
    toc: false
  values:
    family: module
`)

	b := preflightTypedLayoutBuild(t, filepath.Join(root, "site.yaml"))
	landing := b.configured["index.md"]
	if landing.layout.Kind != LayoutLanding || landing.page.Layout != "landing" || landing.page.Family != "" || landing.layout.Family != "" {
		t.Fatalf("landing = layout %+v page %+v", landing.layout, landing.page)
	}
	if !reflect.DeepEqual(landing.layout.Values, map[string]any{"shell": false, "content": map[string]any{"layout": "article"}}) {
		t.Fatalf("landing values leaked docs state: %#v", landing.layout.Values)
	}
	docs := b.configured["module/index.md"]
	if docs.layout.Kind != LayoutDocs || docs.page.Layout != "docs" || docs.page.Family != "module" {
		t.Fatalf("restored docs = layout %+v page %+v", docs.layout, docs.page)
	}
	assertLayoutValues(t, docs.layout.Values, map[string]any{
		"families": []any{"default", "module"},
		"family":   "module",
		"sidebar":  true,
		"toc":      false,
		"content":  map[string]any{"layout": "article"},
	})
}

func TestBuildConfigFamilyIndexUsesDocsPagesAndConfiguredOrder(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), `---
layout:
  kind: landing
---
# Tour

Landing page.
`)
	writeConfigFile(t, filepath.Join(root, "docs", "module", "a.md"), "# Module A\n\nFirst route.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "z", "index.md"), "# Module Overview\n\nIndex route.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "a.md"), "# CLI Overview\n\nCLI route.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "_layout.yaml"), "values:\n  family: module\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "_layout.yaml"), "values:\n  family: cli\n")
	writeTypedLayoutBuildConfig(t, root, `layout:
  kind: docs
  default:
    families: [module, cli]
`)

	b := preflightTypedLayoutBuild(t, filepath.Join(root, "site.yaml"))
	if got, want := len(b.docsFamilies), 2; got != want {
		t.Fatalf("docs family count = %d, want %d: %+v", got, want, b.docsFamilies)
	}
	want := []struct {
		id       string
		locale   string
		overview string
		label    string
	}{
		{id: "module", locale: "en", overview: "module/z/index.md", label: "Module Overview"},
		{id: "cli", locale: "en", overview: "cli/a.md", label: "CLI Overview"},
	}
	for index, expected := range want {
		family := b.docsFamilies[index]
		if family.ID != expected.id || family.Locale != expected.locale || family.Overview.Source != expected.overview || family.Overview.Title != expected.label {
			t.Fatalf("docs family %d = %+v, want %+v", index, family, expected)
		}
	}
}

func TestBuildConfigFamilyEmptyDiagnosticUsesDeclarationIndex(t *testing.T) {
	for _, test := range []struct {
		name     string
		families string
		pointer  string
	}{
		{name: "implicit default", families: "[module, cli]", pointer: "/layout/default/families/1"},
		{name: "explicit default", families: "[default, module, cli]", pointer: "/layout/default/families/2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeConfigFile(t, filepath.Join(root, "docs", "index.md"), `---
layout:
  kind: landing
---
# Tour

Landing page.
`)
			writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "# Module\n\nDocs page.\n")
			writeConfigFile(t, filepath.Join(root, "docs", "module", "_layout.yaml"), "values:\n  family: module\n")
			writeTypedLayoutBuildConfig(t, root, `layout:
  kind: docs
  default:
    families: `+test.families+`
`)
			configPath := filepath.Join(root, "site.yaml")

			result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: configPath})
			if len(result.Artifacts) != 0 || len(result.Pages) != 0 || len(result.Site.Routes) != 0 {
				t.Fatalf("empty family returned partial result: %+v", result)
			}
			requirePresentationDiagnostic(t, err, "site.family_empty", configPath, test.pointer)
		})
	}
}

func TestBuildConfigFamilyAllowsUnusedImplicitDefault(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), `---
layout:
  kind: landing
---
# Tour

Landing page.
`)
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "# Module\n\nDocs page.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "_layout.yaml"), "values:\n  family: module\n")
	writeTypedLayoutBuildConfig(t, root, `layout:
  kind: docs
  default:
    families: [module]
`)

	b := preflightTypedLayoutBuild(t, filepath.Join(root, "site.yaml"))
	if got, want := len(b.docsFamilies), 1; got != want || b.docsFamilies[0].ID != "module" {
		t.Fatalf("docs families = %+v, want only module", b.docsFamilies)
	}
}

func TestBuildConfigFrontmatterPreflightReturnsNoArtifacts(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), `---
layout:
  values:
    sidebar: enabled
---
# Home

Invalid final patch.
`)
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
site:
  name: Margo
  base_url: https://margo.example
  home: index.md
  logo: assets/missing-logo.svg
  icon: assets/missing-icon.svg
  social_image:
    path: assets/missing-social.jpg
    alt: Margo documentation preview
layout:
  kind: docs
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if len(result.Artifacts) != 0 || len(result.Pages) != 0 || len(result.Site.Routes) != 0 {
		t.Fatalf("invalid Markdown patch returned partial result: %+v", result)
	}
	requirePresentationDiagnostic(t, err, "site.layout_value_invalid", "index.md", "/layout/values/sidebar")
}

func TestBuildConfigPreflightValidatesOrphanDirectoryPatchesDeterministically(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Home\n\nPublic page.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "z", "_layout.yaml"), "values:\n  zeta: true\n")
	writeConfigFile(t, filepath.Join(root, "docs", "a", "_layout.yaml"), "values:\n  alpha: true\n")
	writeTypedLayoutBuildConfig(t, root, "layout:\n  kind: docs\n")

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err == nil {
		t.Fatal("expected orphan layout patch diagnostic")
	}
	if len(result.Artifacts) != 0 || len(result.Pages) != 0 || len(result.Site.Routes) != 0 {
		t.Fatal("invalid orphan patch returned partial result")
	}
	requirePresentationDiagnostic(t, err, "site.layout_value_unknown", "a/_layout.yaml", "/values/alpha")
}

func TestBuildConfigPreflightValidatesOrphanPatchAgainstAncestorKind(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Home\n\nPublic page.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "orphan", "_layout.yaml"), "kind: landing\n")
	writeConfigFile(t, filepath.Join(root, "docs", "orphan", "nested", "_layout.yaml"), "values:\n  sidebar: false\n")
	writeTypedLayoutBuildConfig(t, root, "layout:\n  kind: docs\n")

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err == nil {
		t.Fatal("expected ancestor-kind layout patch diagnostic")
	}
	if len(result.Artifacts) != 0 || len(result.Pages) != 0 || len(result.Site.Routes) != 0 {
		t.Fatal("invalid orphan patch returned partial result")
	}
	requirePresentationDiagnostic(t, err, "site.layout_value_unknown", "orphan/nested/_layout.yaml", "/values/sidebar")
}

func TestBuildConfigLayoutCascadeIdentityIsDeterministic(t *testing.T) {
	firstLayout := ResolvedLayout{Kind: LayoutDocs, Values: map[string]any{
		"sidebar": true,
		"content": map[string]any{"layout": "article"},
	}}
	secondLayout := ResolvedLayout{Kind: LayoutDocs, Values: map[string]any{
		"content": map[string]any{"layout": "article"},
		"sidebar": true,
	}}
	firstPageIdentity, err := configuredPageLayoutIdentity("guide.md", firstLayout, []string{"_layout.yaml", "guide.md"})
	if err != nil {
		t.Fatal(err)
	}
	secondPageIdentity, err := configuredPageLayoutIdentity("guide.md", secondLayout, []string{"_layout.yaml", "guide.md"})
	if err != nil {
		t.Fatal(err)
	}
	if firstPageIdentity == "" || firstPageIdentity != secondPageIdentity {
		t.Fatalf("map order changed page identity: %q != %q", firstPageIdentity, secondPageIdentity)
	}
	reversedSourcesIdentity, err := configuredPageLayoutIdentity("guide.md", secondLayout, []string{"guide.md", "_layout.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if reversedSourcesIdentity == firstPageIdentity {
		t.Fatal("ordered patch sources did not affect page identity")
	}

	firstLayout.Identity = firstPageIdentity
	secondLayout.Identity = secondPageIdentity
	firstConfigured := map[string]configuredPage{
		"z.md": {layout: firstLayout},
		"a.md": {layout: secondLayout},
	}
	secondConfigured := map[string]configuredPage{
		"a.md": {layout: secondLayout},
		"z.md": {layout: firstLayout},
	}
	firstSiteIdentity, err := configuredSiteLayoutIdentity(firstLayout, nil, firstConfigured)
	if err != nil {
		t.Fatal(err)
	}
	secondSiteIdentity, err := configuredSiteLayoutIdentity(secondLayout, nil, secondConfigured)
	if err != nil {
		t.Fatal(err)
	}
	if firstSiteIdentity == "" || firstSiteIdentity != secondSiteIdentity {
		t.Fatalf("map order changed site identity: %q != %q", firstSiteIdentity, secondSiteIdentity)
	}
}

func TestBuildConfigLayoutCascadeIdentityIncludesDirectoryPatchValues(t *testing.T) {
	build := func(root string, sidebar bool) string {
		writeConfigFile(t, filepath.Join(root, "docs", "index.md"), `---
layout:
  values:
    sidebar: true
---
# Home

Same final layout.
`)
		writeConfigFile(t, filepath.Join(root, "docs", "_layout.yaml"), fmt.Sprintf("values:\n  sidebar: %t\n", sidebar))
		writeTypedLayoutBuildConfig(t, root, "layout:\n  kind: docs\n")
		return preflightTypedLayoutBuild(t, filepath.Join(root, "site.yaml")).siteManifest.LayoutSchemaHash
	}

	first := build(t.TempDir(), false)
	second := build(t.TempDir(), true)
	if first == second {
		t.Fatalf("directory patch values did not affect site identity: %q", first)
	}
}

func TestBuildConfigRendersLayoutKindsWithOwnedChrome(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), `---
layout:
  kind: landing
---
# Landing article

Landing Markdown survives. [Read module docs](module/index.md).
`)
	writeConfigFile(t, filepath.Join(root, "docs", "article.md"), `---
layout:
  kind: article
---
# Standalone article

Article Markdown survives.
`)
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), `---
margo:
  actions:
    markdown: true
---
# Module overview

Module Markdown survives.

## Module section

Module details.
`)
	writeConfigFile(t, filepath.Join(root, "docs", "module", "guide.md"), "# Module guide\n\nSecond module page.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "index.md"), "# CLI overview\n\nCLI family page.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "_layout.yaml"), "values:\n  family: module\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "_layout.yaml"), "values:\n  family: cli\n")
	writeTypedLayoutBuildConfig(t, root, `layout:
  kind: docs
  default:
    families: [module, cli]
    sidebar: true
    toc: true
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	pages := map[string]string{
		"landing": string(configArtifact(t, result, "index.html")),
		"article": string(configArtifact(t, result, "article.html")),
		"docs":    string(configArtifact(t, result, "module/index.html")),
	}
	for route, content := range map[string]string{
		"landing": "Landing Markdown survives.",
		"article": "Article Markdown survives.",
		"docs":    "Module Markdown survives.",
	} {
		page := pages[route]
		if !strings.Contains(page, `class="margo-document`) || !strings.Contains(page, content) {
			t.Fatalf("Markdown article missing from %s: %s", route, page)
		}
	}
	for _, route := range []string{"landing", "article"} {
		page := pages[route]
		for _, forbidden := range []string{
			`data-margo-family-navigation`, `id="left-nav"`, `id="right-nav"`,
			"margo-breadcrumbs", "margo-pagination", "margo-page-actions",
			"data-toc-heading", "component-doc-shell",
		} {
			if strings.Contains(page, forbidden) {
				t.Fatalf("%s leaked into %s", forbidden, route)
			}
		}
		for _, forbidden := range []string{`data-margo-global-navigation`, `data-search-field`} {
			if strings.Contains(page, forbidden) {
				t.Fatalf("docs navigation %s leaked into %s", forbidden, route)
			}
		}
		if !strings.Contains(page, `data-margo-frame="main"`) {
			t.Fatalf("%s did not use builtin main frame: %s", route, page)
		}
	}
	docs := pages["docs"]
	for _, required := range []string{
		`class="component-doc-shell`,
		`id="componentdocshell-family-navigation"`,
		`class="component-doc-shell__family-link"`,
		`id="componentdocshell-sidebar"`,
		`data-sidebar-section="Module overview"`,
		`id="componentdocshell-toc"`,
		`data-componentdocshell-toc`,
		`class="margo-page-actions"`,
		`class="margo-pagination"`,
		`rel="next"`,
		`>Next: Module guide</a>`,
	} {
		if !strings.Contains(docs, required) {
			t.Fatalf("docs chrome missing %q: %s", required, docs)
		}
	}
	for _, forbidden := range []string{`data-margo-frame="top-left-main-right-footer"`, `data-margo-global-navigation="true"`, `data-margo-family-navigation="true"`, `id="left-nav"`, `id="right-nav"`, `data-margo-toc-drawer`} {
		if strings.Contains(docs, forbidden) {
			t.Fatalf("docs output retained custom shell marker %q", forbidden)
		}
	}
	if strings.Contains(docs, `>Next: CLI overview</a>`) {
		t.Fatalf("docs pagination crossed family boundary: %s", docs)
	}
	if !strings.Contains(pages["landing"], `href="/module/"`) {
		t.Fatalf("typed Markdown link did not use the public docs route: %s", pages["landing"])
	}
	if !strings.Contains(docs, `rel="canonical" href="https://margo.example/module/"`) {
		t.Fatalf("typed docs canonical does not match its public route: %s", docs)
	}
}

func TestBuildConfigLandingGroupsMarkdownIntoSemanticComposition(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), `---
layout:
  kind: landing
---
# Landing

Publish one Markdown source in the format your project needs.

- [Publish with the CLI — standalone publishing workflow](cli/index.md)
- [Embed the Go module — host-owned composition](module/index.md)

![A document becoming several outputs](hero.png)

## Proof

One source, several projections.

![Several generated outputs](proof.png)

## Trust boundaries

The host keeps authority.
`)
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "index.md"), "# CLI\n\nCLI guide.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "# Module\n\nModule guide.\n")
	copyMargoAsset(t, filepath.Join(root, "docs", "hero.png"), "margo-mascot.png")
	copyMargoAsset(t, filepath.Join(root, "docs", "proof.png"), "margo-mascot.png")
	writeTypedLayoutBuildConfig(t, root, "layout:\n  kind: landing\n")

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	page := string(configArtifact(t, result, "index.html"))
	for _, required := range []string{
		`<header class="margo-landing-hero">`,
		`class="margo-landing-hero__copy"`,
		`class="margo-landing-hero__visual"`,
		`<section class="margo-landing-section" aria-labelledby="proof">`,
		`<section class="margo-landing-section" aria-labelledby="trust-boundaries">`,
		`class="margo-landing-media"`,
		`href="/cli/"`,
		`href="/module/"`,
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("landing composition missing %q: %s", required, page)
		}
	}
	if strings.Count(page, `<article class="margo-document`) != 1 || strings.Count(page, "<h1") != 1 {
		t.Fatalf("landing must preserve one article and one h1: %s", page)
	}
	for _, forbidden := range []string{"component-doc-shell", "margo-breadcrumbs", "margo-pagination", "margo-page-actions", `id="left-nav"`, `id="right-nav"`} {
		if strings.Contains(page, forbidden) {
			t.Fatalf("landing leaked docs chrome %q", forbidden)
		}
	}
}

func TestBuildConfigLandingShellUsesPublicAppShell(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), `---
layout:
  kind: landing
  values:
    shell: true
    navigation: [cli/index.md]
    navigation_label: Docs
---
# Landing

Publish one Markdown source in the format your project needs.

![A document becoming several outputs](hero.png)

## Proof

One source, several projections.
`)
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "# Go module\n\nModule guide.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "index.md"), "# CLI workflows\n\nCLI guide.\n")
	copyMargoAsset(t, filepath.Join(root, "docs", "hero.png"), "margo-mascot.png")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
offline: true
site:
  name: Margo
  description: Landing shell fixture.
  version: v0.0.5
  repository_url: https://github.com/araihu/margo
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo landing preview
layout:
  kind: article
locales:
  default: en
  supported: [en]
theme:
  builtin: true
  name: modern
  allow_switch_theme: true
  color_mode: system
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	page := string(configArtifact(t, result, "index.html"))
	for _, required := range []string{
		`class="landing-shell`,
		`data-margo-layout="landing"`,
		`class="landing-shell__header`,
		`class="landing-shell__brand-badge"`,
		`>v0.0.5</span>`,
		`href="/cli/"`,
		`>Docs</a>`,
		`id="landingshell-dark-mode"`,
		`aria-label="Source repository"`,
		`class="landing-shell__footer"`,
		`class="landing-shell__container landing-shell__hero-slot"`,
		`<header class="margo-landing-hero"`,
		`<section class="margo-landing-section" aria-labelledby="proof">`,
		`og:image:type`,
		`twitter:image:alt`,
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("landing shell missing %q: %s", required, page)
		}
	}
	for _, unique := range []string{"<title>", `rel="canonical"`, `property="og:url"`, `name="twitter:card"`} {
		if got := strings.Count(page, unique); got != 1 {
			t.Fatalf("landing shell %q count = %d, want one: %s", unique, got, page)
		}
	}
	for _, artifact := range []string{"landingshell/assets/shell.css", "landingshell/assets/shell.js", "assets/styles.css", "assets/js/goshtoso.min.js"} {
		if !artifactExists(result, artifact) {
			t.Fatalf("landing shell asset %q missing", artifact)
		}
	}
	for _, forbidden := range []string{"component-doc-shell", "componentdocshell", "margo-pagination", "margo-page-actions"} {
		if strings.Contains(page, forbidden) {
			t.Fatalf("landing shell leaked docs chrome %q: %s", forbidden, page)
		}
	}
}

func TestBuildConfigLandingShellRewritesConfiguredBasePathImage(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), `---
layout:
  kind: landing
  values:
    shell: true
    navigation: [guide.md]
---
# Landing

Publish one Markdown source.
`)
	writeConfigFile(t, filepath.Join(root, "docs", "guide.md"), "# Guide\n\nA guide.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
base_path: /portal
site:
  name: Margo Portal
  description: A configured landing shell.
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo portal preview
layout:
  kind: article
locales:
  default: en
  supported: [en]
theme:
  builtin: true
  name: modern
  color_mode: light
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	page := string(configArtifact(t, result, "index.html"))
	if !strings.Contains(page, `<img src="assets/logo.svg"`) {
		t.Fatalf("configured branding image was not rewritten to its local artifact: %s", page)
	}
	if !artifactExists(result, "assets/logo.svg") {
		t.Fatal("configured branding image was not published")
	}
}

func TestBuildConfigLandingShellRejectsUnknownNavigationTarget(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), `---
layout:
  kind: landing
  values:
    shell: true
    navigation: [missing.md]
---
# Landing

Choose a path.
`)
	writeTypedLayoutBuildConfig(t, root, "layout:\n  kind: article\n")

	_, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	requirePresentationDiagnostic(t, err, "site.landing_navigation_target_invalid", "index.md", "/layout/values/navigation/0")
}

func TestBuildConfigLandingSupportsTextOnlyMarkdownWithoutSections(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "---\nlayout:\n  kind: landing\n---\n# Text only\n\nA useful landing needs no image or section.\n")
	writeTypedLayoutBuildConfig(t, root, "layout:\n  kind: landing\n")

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	page := string(configArtifact(t, result, "index.html"))
	if strings.Count(page, `<article class="margo-document`) != 1 || !strings.Contains(page, `<header class="margo-landing-hero"><div class="margo-landing-hero__copy">`) || !strings.Contains(page, "A useful landing needs no image or section.") {
		t.Fatalf("text-only landing fallback malformed: %s", page)
	}
	if strings.Contains(page, "margo-landing-hero__visual") || strings.Contains(page, "margo-landing-section") {
		t.Fatalf("text-only landing emitted empty visual or section: %s", page)
	}
}

func TestBuildConfigLandingFragmentRejectsMalformedRoot(t *testing.T) {
	for _, fragment := range []string{
		`<div class="margo-document"><h1>Wrong element</h1></div>`,
		`<article class="margo-document"><h1>One</h1></article><article class="margo-document"><h1>Two</h1></article>`,
	} {
		if _, err := transformLandingArticle([]byte(fragment)); err == nil {
			t.Fatalf("malformed landing fragment accepted: %s", fragment)
		}
	}
}

func TestBuildConfigStagesLayoutKindAssetsAndDependencies(t *testing.T) {
	build := func(t *testing.T, assets string) Result {
		t.Helper()
		root := t.TempDir()
		writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "---\nlayout:\n  kind: landing\n---\n# Landing\n\nLanding page.\n")
		writeConfigFile(t, filepath.Join(root, "docs", "article.md"), "---\nlayout:\n  kind: article\n---\n# Article\n\nArticle page.\n")
		writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "# Module\n\n## Module section\n\nDocs page.\n")
		writeConfigFile(t, filepath.Join(root, "docs", "cli", "index.md"), "# CLI\n\nCLI page.\n")
		writeConfigFile(t, filepath.Join(root, "docs", "module", "_layout.yaml"), "values:\n  family: module\n")
		writeConfigFile(t, filepath.Join(root, "docs", "cli", "_layout.yaml"), "values:\n  family: cli\n")
		copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
		copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
		writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: `+assets+`
site:
  name: Margo
  description: Typed dependency fixture.
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo preview
layout:
  kind: docs
  default:
    families: [module, cli]
locales:
  default: en
  supported: [en]
theme:
  builtin: true
  name: modern
  color_mode: light
`)
		result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
		if err != nil {
			t.Fatal(err)
		}
		return result
	}

	localResult := build(t, "local")
	inlineResult := build(t, "inline")
	if localResult.Site.DocumentStyleDigest == "" || localResult.Site.DocumentStyleDigest != inlineResult.Site.DocumentStyleDigest {
		t.Fatalf("asset mode changed typed style identity: local=%q inline=%q", localResult.Site.DocumentStyleDigest, inlineResult.Site.DocumentStyleDigest)
	}

	t.Run("local", func(t *testing.T) {
		result := localResult
		landing := string(configArtifact(t, result, "index.html"))
		article := string(configArtifact(t, result, "article.html"))
		docs := string(configArtifact(t, result, "module/index.html"))
		for _, required := range []string{configuredTypedSiteStylePath, configuredLandingStylePath} {
			if !strings.Contains(landing, required) {
				t.Fatalf("landing dependency missing %q: %s", required, landing)
			}
		}
		for _, forbidden := range []string{configuredDocsStylePath, pageActionsScriptPath, searchInteractionsScriptPath, "assets/js/goshtoso.min.js"} {
			if strings.Contains(landing, forbidden) || strings.Contains(article, forbidden) {
				t.Fatalf("docs dependency %q leaked into landing/article", forbidden)
			}
		}
		if !strings.Contains(article, configuredTypedSiteStylePath) || strings.Contains(article, configuredLandingStylePath) {
			t.Fatalf("article dependencies are not isolated: %s", article)
		}
		for _, required := range []string{configuredDocsStylePath, pageActionsScriptPath, searchInteractionsScriptPath, "margo-assets/goshtoso/shell.css", "margo-assets/goshtoso/shell.js", "assets/styles.css", "assets/js/goshtoso.min.js"} {
			if !strings.Contains(docs, required) {
				t.Fatalf("docs dependency missing %q: %s", required, docs)
			}
		}
		for _, forbidden := range []string{configuredTypedSiteStylePath} {
			if strings.Contains(docs, forbidden) {
				t.Fatalf("legacy typed docs dependency %q leaked into component shell: %s", forbidden, docs)
			}
		}
		for _, artifact := range []string{configuredTypedSiteStylePath, configuredLandingStylePath, configuredDocsStylePath, pageActionsScriptPath, searchInteractionsScriptPath, "margo-assets/goshtoso/shell.css", "margo-assets/goshtoso/shell.js", "assets/styles.css", "assets/js/goshtoso.min.js"} {
			if !artifactExists(result, artifact) {
				t.Fatalf("owned local artifact %q missing", artifact)
			}
		}
		if styles := string(configArtifact(t, result, configuredTypedSiteStylePath)); strings.Contains(styles, "data-margo-layout") || strings.Contains(styles, "component-doc-shell") || strings.Contains(styles, "margo-page-actions") {
			t.Fatalf("shared stylesheet contains kind-owned selectors: %s", styles)
		}
		if styles := string(configArtifact(t, result, configuredLandingStylePath)); !strings.Contains(styles, `.margo-landing-hero`) || strings.Contains(styles, `[alt^=`) || strings.Contains(styles, `@media (min-width: 56.25rem)`) || strings.Contains(styles, `data-margo-layout="docs"`) || strings.Contains(styles, "component-doc-shell") {
			t.Fatalf("landing stylesheet ownership is not isolated: %s", styles)
		} else if !strings.Contains(styles, `.margo-landing-section > .margo-landing-media > img, .margo-landing-section > .margo-landing-media > svg`) || strings.Contains(styles, `.margo-landing-section > .margo-landing-media img`) || strings.Contains(styles, `.margo-landing-section > .margo-landing-media svg`) {
			t.Fatalf("landing media styles must not resize nested component SVGs: %s", styles)
		} else if !strings.Contains(styles, `[data-margo-layout="landing"] .goshtoso-charts-expand-panel`) || !strings.Contains(styles, `max-inline-size: min(100%, 36rem)`) || !strings.Contains(styles, `block-size: min(calc(100dvh - 2rem), 36rem)`) || !strings.Contains(styles, `max-block-size: calc(100dvh - 2rem)`) {
			t.Fatalf("landing chart expansion must stay compact and viewport-safe: %s", styles)
		}
		if styles := string(configArtifact(t, result, configuredDocsStylePath)); !strings.Contains(styles, `.margo-showcase-article`) || !strings.Contains(styles, `.margo-pagination`) || strings.Contains(styles, "component-doc-shell") || strings.Contains(styles, "margo-frame") {
			t.Fatalf("docs stylesheet ownership is not isolated: %s", styles)
		}
	})

	t.Run("inline", func(t *testing.T) {
		result := inlineResult
		landing := string(configArtifact(t, result, "index.html"))
		article := string(configArtifact(t, result, "article.html"))
		docs := string(configArtifact(t, result, "module/index.html"))
		for route, page := range map[string]string{"landing": landing, "article": article} {
			for _, forbidden := range []string{`data-margo-layout-style="docs"`, `data-margo-layout-dependency="page-actions"`, `data-margo-layout-dependency="search-interactions"`, `data-margo-layout-dependency="site-navigation"`, `data-margo-layout-dependency="goshtoso-navigation"`} {
				if strings.Contains(page, forbidden) {
					t.Fatalf("inline docs dependency %q leaked into %s", forbidden, route)
				}
			}
		}
		if !strings.Contains(landing, `data-margo-layout-style="landing"`) || strings.Contains(article, `data-margo-layout-style="landing"`) {
			t.Fatalf("inline landing style ownership is not isolated")
		}
		for _, required := range []string{`data-margo-layout-style="docs"`, `data-margo-layout-dependency="page-actions"`, `data-margo-layout-dependency="search-interactions"`, `margo-assets/goshtoso/shell.css`, `margo-assets/goshtoso/shell.js`} {
			if !strings.Contains(docs, required) {
				t.Fatalf("inline docs dependency missing %q: %s", required, docs)
			}
		}
		for _, external := range []string{configuredDocsStylePath, pageActionsScriptPath, searchInteractionsScriptPath} {
			if strings.Contains(docs, `src="/`+external) || strings.Contains(docs, `href="/`+external) {
				t.Fatalf("inline docs retained external dependency %q: %s", external, docs)
			}
		}
		for _, artifact := range []string{configuredTypedSiteStylePath, configuredLandingStylePath, configuredDocsStylePath, pageActionsScriptPath, searchInteractionsScriptPath} {
			if artifactExists(result, artifact) {
				t.Fatalf("inline build published layout artifact %q", artifact)
			}
		}
		for _, artifact := range []string{"margo-assets/goshtoso/shell.css", "margo-assets/goshtoso/shell.js", "assets/styles.css", "assets/js/goshtoso.min.js"} {
			if !artifactExists(result, artifact) {
				t.Fatalf("inline build did not stage public shell asset %q", artifact)
			}
		}
		if artifactExists(result, "margo-assets/goshtoso/margo-scroll-spy.js") {
			t.Fatal("inline typed docs staged Margo's legacy TOC runtime")
		}
	})
}

func TestBuildConfigNonDocsLayoutsHonorGoshtosoRequirementsAndPreserveTableSort(t *testing.T) {
	for _, kind := range []LayoutKind{LayoutLanding, LayoutArticle} {
		for _, assets := range []string{string(AssetsLocal), string(AssetsInline)} {
			t.Run(string(kind)+"/"+assets, func(t *testing.T) {
				root := t.TempDir()
				writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Example\n\n| Name | Value |\n| --- | --- |\n| Margo | Site |\n")
				copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
				copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
				writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: `+assets+`
site:
  name: Margo
  description: Non-docs dependency fixture.
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo preview
layout:
  kind: `+string(kind)+`
`)

				result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
				if err != nil {
					t.Fatal(err)
				}
				page := string(configArtifact(t, result, "index.html"))
				if !strings.Contains(page, `data-margo-requirement="goshtoso.styles"`) {
					t.Fatalf("%s %s page dropped its declared Goshtoso stylesheet requirement: %s", kind, assets, page)
				}
				if assets == string(AssetsLocal) {
					if !strings.Contains(page, "assets/styles.css") || !artifactExists(result, "assets/styles.css") {
						t.Fatalf("%s local build did not publish its required Goshtoso stylesheet", kind)
					}
				} else if !strings.Contains(page, "tailwindcss v4.3.3") {
					t.Fatalf("%s inline build did not embed its required Goshtoso stylesheet", kind)
				}
				if !strings.Contains(page, `data-margo-requirement="margo.table-sort"`) {
					t.Fatalf("%s %s page lost semantic table-sort dependency: %s", kind, assets, page)
				}
				if assets == string(AssetsLocal) && !artifactExists(result, "margo-assets/table-sort.js") {
					t.Fatalf("%s local build did not stage table-sort runtime", kind)
				}
			})
		}
	}
}

func preflightTypedLayoutBuild(t *testing.T, configPath string) *builder {
	t.Helper()
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Dir(configPath)
	sourceDir := filepath.Join(configDir, filepath.FromSlash(config.Source))
	inputs, err := discoverConfiguredInputs(context.Background(), sourceDir, config.Navigation.Exclude)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{
		request:       requestToSiteRequest(ConfigRequest{Compiler: margo.New()}, sourceDir, inputs.Sources, AssetMode(config.Assets)),
		config:        &config,
		configSource:  configPath,
		configDir:     configDir,
		sourceDir:     sourceDir,
		layoutPatches: inputs.Patches,
		configured:    make(map[string]configuredPage),
		sources:       make(map[string]Source, len(inputs.Sources)),
		outputs:       make(map[string]string, len(inputs.Sources)),
		siteManifest: SiteManifest{
			ConfigVersion: 1,
			Layout:        "layout:" + string(config.Layout.Kind),
			BaseURL:       strings.TrimSuffix(config.Site.BaseURL, "/"),
			BasePath:      normalizedBasePath(config.BasePath),
		},
	}
	ordered, err := b.indexSources()
	if err != nil {
		t.Fatal(err)
	}
	if err := b.preflightConfigured(context.Background(), ordered); err != nil {
		t.Fatal(err)
	}
	return b
}

func writeTypedLayoutBuildConfig(t *testing.T, root, layout string) {
	t.Helper()
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
site:
  name: Margo
  description: Margo documentation
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo documentation preview
locales:
  default: en
  supported: [en]
theme:
  builtin: true
  name: modern
  color_mode: light
`+layout)
}

func configArtifact(t *testing.T, result Result, name string) []byte {
	t.Helper()
	for _, artifact := range result.Artifacts {
		if artifact.Path == name {
			return artifact.Content
		}
	}
	t.Fatalf("artifact %q missing", name)
	return nil
}

func writeConfigFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyMargoAsset(t *testing.T, target, relative string) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "assets", relative))
	if err != nil {
		t.Fatal(err)
	}
	writeConfigFile(t, target, string(data))
}
