package site

import (
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
	writeConfigFile(t, filepath.Join(root, "docs", "guide.md"), "# Guide\n\nA guide-specific description.\n")
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

func TestBuildConfiguredShowcasePublicationContract(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	root := filepath.Join(filepath.Dir(filename), "..")
	result, err := BuildConfig(context.Background(), ConfigRequest{
		ConfigPath: filepath.Join(root, "showcase.yaml"),
		Compiler: margo.New(margo.WithExtension(charts.Extension(
			charts.WithExternalizedControlRuntime(true),
		))),
	})
	if err != nil {
		t.Fatal(err)
	}

	htmlRoutes := make([]string, 0)
	artifacts := make(map[string][]byte, len(result.Artifacts))
	for _, artifact := range result.Artifacts {
		artifacts[artifact.Path] = artifact.Content
		if strings.HasSuffix(artifact.Path, ".html") {
			htmlRoutes = append(htmlRoutes, artifact.Path)
		}
	}
	sort.Strings(htmlRoutes)
	if got, want := htmlRoutes, []string{"cli/index.html", "index.html", "module/index.html"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("HTML routes = %v, want exactly %v", got, want)
	}
	for _, retired := range []string{"charts", "cli", "decks", "determinism", "html", "markdown", "mermaid", "module", "pdf", "policy", "site"} {
		if _, exists := artifacts[retired+".html"]; exists {
			t.Fatalf("retired route artifact %q exists", retired+".html")
		}
	}
	if got, want := len(result.Site.Routes), 3; got != want {
		t.Fatalf("configured routes = %d, want %d: %+v", got, want, result.Site.Routes)
	}
	wantRoutes := map[string]struct {
		output string
		family string
		layout string
	}{
		"index.md":        {output: "index.html", family: "tour", layout: "landing"},
		"module/index.md": {output: "module/index.html", family: "module", layout: "docs"},
		"cli/index.md":    {output: "cli/index.html", family: "cli", layout: "docs"},
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
	if got, want := result.Site.Layout, "profiles:docs=top-left-main-right-footer,landing=top-main-footer"; got != want {
		t.Fatalf("layout identity = %q, want %q", got, want)
	}
	if result.Site.LayoutSchemaHash == "" || result.Site.LayoutSchemaHash == "legacy" {
		t.Fatalf("layout schema identity = %q", result.Site.LayoutSchemaHash)
	}

	landing := string(artifacts["index.html"])
	module := string(artifacts["module/index.html"])
	cli := string(artifacts["cli/index.html"])
	for name, page := range map[string]string{
		"Tour":   landing,
		"Module": module,
		"CLI":    cli,
	} {
		route := map[string]string{"Tour": "https://margo.araihu.com/", "Module": "https://margo.araihu.com/module/", "CLI": "https://margo.araihu.com/cli/"}[name]
		if !strings.Contains(page, `<link rel="canonical" href="`+route+`"`) {
			t.Fatalf("%s canonical does not expose public route %q: %s", name, route, page)
		}
	}
	for name, page := range map[string]string{"Tour": landing, "Module": module, "CLI": cli} {
		if strings.Count(page, "<h1") != 1 {
			t.Fatalf("%s h1 count = %d", name, strings.Count(page, "<h1"))
		}
	}
	for _, required := range []string{`href="/module/"`, `href="/cli/"`, `One source, several projections`} {
		if !strings.Contains(landing, required) {
			t.Fatalf("Tour missing %q", required)
		}
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
	for _, forbidden := range []string{"margo-breadcrumbs", "margo-pagination", "margo-page-actions", `id="left-nav"`, `id="right-nav"`, "data-toc-heading"} {
		if strings.Contains(landing, forbidden) {
			t.Fatalf("Tour contains forbidden landing markup %q", forbidden)
		}
	}
	for _, required := range []string{"Compiler lifecycle", "Public package map", "Installation and versioning", "Testing"} {
		if !strings.Contains(module, required) {
			t.Fatalf("Module outline missing %q", required)
		}
	}
	for _, required := range []string{"Command map", "Configuration and policy layering", "Operational gotchas", "check", "completion"} {
		if !strings.Contains(cli, required) {
			t.Fatalf("CLI outline missing %q", required)
		}
	}
	sitemap := string(artifacts[SitemapPath])
	if strings.Count(sitemap, "<url>") != 3 {
		t.Fatalf("sitemap URL count = %d, want 3: %s", strings.Count(sitemap, "<url>"), sitemap)
	}
	for _, route := range []string{"https://margo.araihu.com/", "https://margo.araihu.com/module/", "https://margo.araihu.com/cli/"} {
		if !strings.Contains(sitemap, "<loc>"+route+"</loc>") {
			t.Fatalf("sitemap missing %q", route)
		}
	}
	llms := string(artifacts[LLMSPath])
	for _, title := range []string{"[Margo]", "[Go module]", "[CLI workflows]"} {
		if !strings.Contains(llms, title) {
			t.Fatalf("llms.txt missing %q: %s", title, llms)
		}
	}
	for _, route := range []string{"https://margo.araihu.com/", "https://margo.araihu.com/module/", "https://margo.araihu.com/cli/"} {
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
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Landing\n\nChoose a path.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "# Module\n\nModule documentation.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "index.md"), "# CLI\n\nCLI documentation.\n")
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
layouts:
  default: docs
  profiles:
    landing:
      frame:
        builtin: top-main-footer
    docs:
      frame:
        builtin: top-left-main-right-footer
navigation:
  mode: file-tree
  families:
    - id: tour
      label: Tour
      source: .
      overview: index.md
      layout: landing
    - id: module
      label: Module
      source: module
      overview: module/index.md
      layout: docs
    - id: cli
      label: CLI
      source: cli
      overview: cli/index.md
      layout: docs
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
		t.Fatal("configured profile build is not deterministic")
	}
	if len(first.Site.Routes) != 3 {
		t.Fatalf("routes = %+v", first.Site.Routes)
	}
	want := map[string]Page{
		"index.md":        {Family: "tour", Layout: "landing"},
		"module/index.md": {Family: "module", Layout: "docs"},
		"cli/index.md":    {Family: "cli", Layout: "docs"},
	}
	for _, page := range first.Site.Routes {
		expected, ok := want[page.Source]
		if !ok || page.Family != expected.Family || page.Layout != expected.Layout {
			t.Fatalf("route %q identity = family %q layout %q, want %+v", page.Source, page.Family, page.Layout, expected)
		}
	}
	if first.Site.Layout == "" || !strings.Contains(first.Site.Layout, "landing") || !strings.Contains(first.Site.Layout, "docs") {
		t.Fatalf("profile layout identity = %q", first.Site.Layout)
	}
	if first.Site.LayoutSchemaHash == "" || first.Site.LayoutSchemaHash == "legacy" {
		t.Fatalf("profile schema identity = %q", first.Site.LayoutSchemaHash)
	}
	landing := string(configArtifact(t, first, "index.html"))
	docs := string(configArtifact(t, first, "module/index.html"))
	if !strings.Contains(landing, `data-margo-frame="top-main-footer"`) {
		t.Fatalf("landing frame missing: %s", landing)
	}
	if !strings.Contains(docs, `data-margo-frame="top-left-main-right-footer"`) {
		t.Fatalf("docs frame missing: %s", docs)
	}
}

func TestBuildConfigRendersSharedFamilyNavigationAndScopedPagination(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Tour\n\nChoose a path.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "# Module\n\nModule documentation.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "index.md"), "# CLI\n\nCLI documentation.\n")
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
layouts:
  default: docs
  profiles:
    landing:
      frame:
        builtin: top-main-footer
    docs:
      frame:
        builtin: top-left-main-right-footer
navigation:
  mode: file-tree
  families:
    - id: tour
      label: Tour
      source: .
      overview: index.md
      layout: landing
    - id: module
      label: Module
      source: module
      overview: module/index.md
      layout: docs
    - id: cli
      label: CLI
      source: cli
      overview: cli/index.md
      layout: docs
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
	for source, page := range pages {
		if !strings.Contains(page, `data-margo-layout="`) {
			t.Fatalf("%s missing semantic layout hook: %s", source, page)
		}
		if strings.Count(page, `aria-current="location"`) != 1 {
			t.Fatalf("%s has wrong active family count: %s", source, page)
		}
		if strings.Count(page, `data-search-field=""`) != 1 {
			t.Fatalf("%s renders duplicate global search fields: %s", source, page)
		}
		globalStart := strings.Index(page, `data-margo-family-navigation="true"`)
		if globalStart < 0 {
			t.Fatalf("%s missing global navigation: %s", source, page)
		}
		globalEnd := strings.Index(page[globalStart:], `</nav>`)
		if globalEnd < 0 {
			t.Fatalf("%s global navigation is not closed: %s", source, page)
		}
		global := page[globalStart : globalStart+globalEnd]
		last := -1
		for _, label := range []string{"Tour", "Module", "CLI"} {
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
	}
	landing := pages["index.md"]
	if strings.Contains(landing, `id="left-nav"`) || strings.Contains(landing, `aria-label="sidebar navigation"`) {
		t.Fatalf("landing unexpectedly renders local navigation: %s", landing)
	}
	styles := string(configArtifact(t, result, "margo-assets/site.css"))
	for _, required := range []string{`[data-margo-layout="landing"]`, `[data-margo-layout="docs"]`} {
		if !strings.Contains(styles, required) {
			t.Fatalf("profile stylesheet missing %s", required)
		}
	}
	for _, required := range []string{
		`[data-margo-layout="docs"].margo-frame--top-left-main-right-footer`,
		`grid-template-columns: minmax(12rem, 16rem) minmax(0, var(--margo-reading-measure)) minmax(12rem, 16rem);`,
		`grid-template-areas: "top-nav top-nav top-nav" "left-nav main-content right-nav" "footer footer footer";`,
		`@media (min-width: 880px) and (max-width: 1199px)`,
		`grid-template-columns: minmax(9rem, 11rem) minmax(0, 1fr) minmax(9rem, 11rem);`,
		`grid-template-areas: "top-nav top-nav top-nav" "left-nav main-content right-nav" "footer footer footer";`,
		`[data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--left-nav`,
		`[data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--main-content`,
		`[data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--right-nav`,
		`@media (max-width: 879px)`,
		`grid-template-areas: "top-nav" "left-nav" "main-content" "right-nav" "footer";`,
		`overflow-x: clip`,
	} {
		if !strings.Contains(styles, required) {
			t.Fatalf("profile stylesheet missing responsive docs rail contract %s", required)
		}
	}
	if strings.Contains(styles, `[data-margo-layout="docs"] .margo-frame--top-left-main-right-footer`) {
		t.Fatalf("profile stylesheet scopes the docs frame as a descendant instead of the frame element")
	}
	if strings.Contains(styles, "component-doc-shell") || strings.Contains(styles, "componentdocshell") {
		t.Fatalf("profile stylesheet leaks App Shell-private selectors")
	}
	for _, asset := range []string{
		"assets/styles.css",
		"assets/js/goshtoso.min.js",
		"assets/js/runtime/alpinejs/3.14.9/alpine.min.js",
	} {
		if len(configArtifact(t, result, asset)) == 0 {
			t.Fatalf("profile navigation asset %q missing", asset)
		}
	}
	for source, family := range map[string]string{"module/index.md": "Module", "cli/index.md": "CLI"} {
		page := pages[source]
		if got := strings.Count(page, "assets/styles.css"); got != 1 {
			t.Fatalf("%s emits Goshtoso stylesheet %d times, want exactly once: %s", source, got, page)
		}
		leftStart := strings.Index(page, `id="left-nav"`)
		leftEnd := strings.Index(page[leftStart:], `</div>`)
		if leftStart < 0 || leftEnd < 0 {
			t.Fatalf("%s missing family sidebar: %s", source, page)
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

func TestBuildConfigRendersProfileSemanticChromePresenceAndAbsence(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Tour\n\nChoose a documentation family.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "# Module\n\nModule overview.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "guide.md"), "# Module guide\n\nModule detail.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "index.md"), "# CLI\n\nCLI overview.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "guide.md"), "# CLI guide\n\nCLI detail.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
site:
  name: Margo
  description: Profile semantic fixture.
  repository_url: https://github.com/araihu/margo
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo preview.
layouts:
  default: docs
  profiles:
    landing:
      frame:
        builtin: top-main-footer
    docs:
      frame:
        builtin: top-left-main-right-footer
navigation:
  mode: file-tree
  families:
    - id: tour
      label: Tour
      source: .
      overview: index.md
      layout: landing
    - id: module
      label: Module
      source: module
      overview: module/index.md
      layout: docs
    - id: cli
      label: CLI
      source: cli
      overview: cli/index.md
      layout: docs
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
	for name, page := range pages {
		if strings.Count(page, `aria-current="location"`) != 1 || !strings.Contains(page, `data-margo-family-navigation="true"`) {
			t.Fatalf("%s global family state is not semantic: %s", name, page)
		}
		if !strings.Contains(page, `data-margo-repository-link="true"`) || !strings.Contains(page, `aria-label="Repository"`) || !strings.Contains(page, `<svg`) {
			t.Fatalf("%s repository action is not an accessible icon link: %s", name, page)
		}
		if strings.Contains(page, `>Repository</a>`) {
			t.Fatalf("%s repository action still exposes the old text link: %s", name, page)
		}
		if strings.Contains(page, "component-doc-shell") || strings.Contains(page, "componentdocshell") {
			t.Fatalf("%s profile output leaks App Shell-private markup: %s", name, page)
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
			t.Fatalf("%s renders a breadcrumb in the profile docs chrome: %s", name, page)
		}
		for _, required := range []string{
			`data-margo-layout="docs"`,
			`id="left-nav"`,
			`aria-label="sidebar navigation"`,
			`data-sidebar-section="` + family + `"`,
			`aria-current="page"`,
			`class="margo-pagination"`,
			`id="right-nav"`,
		} {
			if !strings.Contains(page, required) {
				t.Fatalf("%s missing semantic profile output %q: %s", name, required, page)
			}
		}
		if !strings.Contains(page, `data-margo-toc="true"`) || !strings.Contains(page, `data-margo-toc-link=`) {
			t.Fatalf("%s is missing a usable Margo-owned TOC payload: %s", name, page)
		}
		if strings.Contains(page, `data-toc-heading`) || strings.Contains(page, `component-doc-shell`) {
			t.Fatalf("%s TOC leaks private App Shell semantics: %s", name, page)
		}
	}
	styles := string(configArtifact(t, result, "margo-assets/site.css"))
	for _, required := range []string{
		`[data-margo-layout="landing"] .margo-area--top-nav > *`,
		`[data-margo-layout="docs"] .margo-area--top-nav > *`,
		`@media (width < 30rem)`,
		`[data-margo-layout="landing"] .margo-site-search`,
		`[data-margo-layout="docs"] .margo-site-search`,
		`[data-margo-layout="landing"] .margo-site-repository`,
		`[data-margo-layout="docs"] .margo-site-repository`,
	} {
		if !strings.Contains(styles, required) {
			t.Fatalf("profile stylesheet missing mobile chrome constraint %q: %s", required, styles)
		}
	}
}

func TestBuildConfigRendersLocaleScopedFamilySearch(t *testing.T) {
	root := t.TempDir()
	for _, localePrefix := range []string{"", "pt-BR"} {
		prefix := filepath.Join(root, "docs", localePrefix)
		writeConfigFile(t, filepath.Join(prefix, "index.md"), "# Tour\n\nTour documentation. [Module](module/index.md).\n")
		writeConfigFile(t, filepath.Join(prefix, "module", "index.md"), "# Module\n\nModule documentation.\n")
		writeConfigFile(t, filepath.Join(prefix, "cli", "index.md"), "# CLI\n\nCLI documentation.\n")
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
layouts:
  default: docs
  profiles:
    landing:
      frame:
        builtin: top-main-footer
    docs:
      frame:
        builtin: top-left-main-right-footer
navigation:
  mode: file-tree
  families:
    - id: tour
      label: Tour
      source: .
      overview: index.md
      layout: landing
    - id: module
      label: Module
      source: module
      overview: module/index.md
      layout: docs
    - id: cli
      label: CLI
      source: cli
      overview: cli/index.md
      layout: docs
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
			{id: "tour", href: fixture.localRoutes[0]},
			{id: "module", href: fixture.localRoutes[1]},
			{id: "cli", href: fixture.localRoutes[2]},
		} {
			link := `data-margo-family-link="` + family.id + `" href="` + family.href + `"`
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

func TestBuildConfigRejectsMissingConfiguredFamilyOverview(t *testing.T) {
	root := t.TempDir()
	writeProfileOverviewFixture(t, root, map[string]string{
		"index.md":        "# Tour\n\nTour documentation.\n",
		"module/guide.md": "# Module guide\n\nModule detail.\n",
	}, []string{"en"})

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if len(result.Artifacts) != 0 || len(result.Pages) != 0 || len(result.Site.Routes) != 0 {
		t.Fatalf("failed build returned partial result: %+v", result)
	}
	diagnostic := presentationDiagnostic(t, err)
	if diagnostic.Code != "site.family_overview_missing" || diagnostic.Pointer != "/navigation/families/1/overview" || diagnostic.Source != "module/index.md" {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
}

func TestBuildConfigRejectsLocaleIncompleteFamilyOverview(t *testing.T) {
	root := t.TempDir()
	writeProfileOverviewFixture(t, root, map[string]string{
		"index.md":              "# Tour\n\nTour documentation.\n",
		"module/index.md":       "# Module\n\nModule documentation.\n",
		"pt-BR/index.md":        "# Tour\n\nDocumentação.\n",
		"pt-BR/module/guide.md": "# Guia do módulo\n\nDetalhes.\n",
	}, []string{"en", "pt-BR"})

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if len(result.Artifacts) != 0 || len(result.Pages) != 0 || len(result.Site.Routes) != 0 {
		t.Fatalf("failed build returned partial result: %+v", result)
	}
	diagnostic := presentationDiagnostic(t, err)
	if diagnostic.Code != "site.family_overview_locale_incomplete" || diagnostic.Pointer != "/navigation/families/1/overview" || diagnostic.Source != "module/index.md" {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
	if !strings.Contains(diagnostic.Message, "pt-BR") {
		t.Fatalf("diagnostic message = %q", diagnostic.Message)
	}
}

func writeProfileOverviewFixture(t *testing.T, root string, pages map[string]string, supported []string) {
	t.Helper()
	for source, content := range pages {
		writeConfigFile(t, filepath.Join(root, "docs", filepath.FromSlash(source)), content)
	}
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
    alt: Margo preview
layouts:
  default: docs
  profiles:
    landing:
      frame:
        builtin: top-main-footer
    docs:
      frame:
        builtin: top-left-main-right-footer
navigation:
  mode: file-tree
  families:
    - id: tour
      label: Tour
      source: .
      overview: index.md
      layout: landing
    - id: module
      label: Module
      source: module
      overview: module/index.md
      layout: docs
locales:
  default: en
  supported: [`+strings.Join(supported, ", ")+`]
theme:
  builtin: true
  name: modern
  color_mode: light
`)
}

func TestBuildConfigPageLayoutOverrideWinsOverFamily(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Landing\n\nChoose a path.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "---\nmargo:\n  site:\n    layout: landing\n---\n# Module\n\nModule documentation.\n")
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
layouts:
  default: docs
  profiles:
    landing:
      frame:
        builtin: top-main-footer
    docs:
      frame:
        builtin: top-left-main-right-footer
navigation:
  families:
    - id: tour
      label: Tour
      source: .
      overview: index.md
      layout: landing
    - id: module
      label: Module
      source: module
      overview: module/index.md
      layout: docs
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
		if page.Source == "module/index.md" && (page.Family != "module" || page.Layout != "landing") {
			t.Fatalf("page override identity = %+v", page)
		}
	}
	if page := string(configArtifact(t, result, "module/index.html")); !strings.Contains(page, `data-margo-frame="top-main-footer"`) {
		t.Fatalf("page override did not select landing frame: %s", page)
	}
}

func TestBuildConfigLayoutPreflightInvalidSelectedProfileDoesNotEmitHTML(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Landing\n\nChoose a path.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "---\nmargo:\n  site:\n    layout: missing\n---\n# Module\n\nModule documentation.\n")
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
layouts:
  default: docs
  profiles:
    docs:
      frame:
        builtin: top-left-main-right-footer
navigation:
  families:
    - id: tour
      label: Tour
      source: .
      overview: index.md
      layout: docs
    - id: module
      label: Module
      source: module
      overview: module/index.md
      layout: docs
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
		t.Fatalf("invalid profile emitted artifacts: %+v", result.Artifacts)
	}
	var diagnosticError *margo.DiagnosticError
	if !errors.As(err, &diagnosticError) || len(diagnosticError.Diagnostics) != 1 || diagnosticError.Diagnostics[0].Code != "site.layout_unknown" {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildConfigPresentationIdentityMatchesRoutes(t *testing.T) {
	config := Config{
		Version: 1, Source: "docs", Output: "dist", Assets: string(AssetsLocal),
		Site:    SiteConfig{Name: "Margo", BaseURL: "https://margo.example", Home: "index.md"},
		Locales: LocaleConfig{Default: "en", Supported: []string{"en"}},
		Theme:   ThemeSelection{Name: "modern", ColorMode: "light"},
		Layouts: LayoutProfiles{
			Default: "docs",
			Profiles: map[string]LayoutProfile{
				"landing": {Frame: LayoutSelection{Builtin: "top-main-footer"}},
				"docs":    {Frame: LayoutSelection{Builtin: "top-left-main-right-footer"}},
			},
		},
		Navigation: NavigationConfig{Families: []FamilyConfig{
			{ID: "tour", Label: "Tour", Source: ".", Overview: "index.md", Layout: "landing"},
			{ID: "module", Label: "Module", Source: "module", Overview: "module/index.md", Layout: "docs"},
		}},
	}
	presentations, err := prepareFramePresentations(config)
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{
		request: Request{Compiler: margo.New()}, config: &config, sourceDir: t.TempDir(),
		profileMode: true, presentations: presentations, configured: map[string]configuredPage{},
	}
	sources := []Source{
		{Path: "index.md", Content: []byte("# Landing\n\nChoose a path.\n")},
		{Path: "module/index.md", Content: []byte("# Module\n\nModule documentation.\n")},
	}
	if err := b.preflightConfigured(context.Background(), sources); err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		family string
		layout string
	}{
		"index.md":        {family: "tour", layout: "landing"},
		"module/index.md": {family: "module", layout: "docs"},
	}
	for source, expected := range want {
		prepared, ok := b.configured[source]
		if !ok {
			t.Fatalf("configured page %q missing", source)
		}
		if prepared.presentation.FamilyID != expected.family || prepared.presentation.LayoutName != expected.layout {
			t.Fatalf("presentation %q = family %q layout %q, want family %q layout %q", source, prepared.presentation.FamilyID, prepared.presentation.LayoutName, expected.family, expected.layout)
		}
		if prepared.page.Family != prepared.presentation.FamilyID || prepared.page.Layout != prepared.presentation.LayoutName {
			t.Fatalf("route %q = family %q layout %q, presentation = family %q layout %q", source, prepared.page.Family, prepared.page.Layout, prepared.presentation.FamilyID, prepared.presentation.LayoutName)
		}
	}
}

func TestBuildConfigRendersGoshtosoComponentDocShell(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "showcase", "index.md"), "# Showcase\n\nA public feature tour.\n\n## A section\n\nA section for the shell TOC.\n")
	writeConfigFile(t, filepath.Join(root, "showcase", "markdown.md"), "# Markdown\n\nThe compiler path.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
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
  icon: assets/logo.svg
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
		`href="https://goshtoso.araihu.com/"`, `Built with Goshtoso`,
		`href="https://araihu.com/"`, `Arai Hû`,
		`href="/llms.txt">llms.txt`, `href="/sitemap.xml">sitemap.xml`,
		`<title>Showcase — Markdown to durable outputs</title>`, `<meta name="description" content="A public feature tour."`,
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
	if strings.Contains(page, `component-doc-shell__brand-logo`) {
		t.Fatalf("shell page unexpectedly includes a brand image: %s", page)
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
	for _, forbidden := range []string{"\nbutton, a {", "\nbutton {", "\n:focus-visible {"} {
		if strings.Contains(styles, forbidden) {
			t.Fatalf("shell CSS leaks an unscoped control rule %q: %s", forbidden, styles)
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
	if !reflect.DeepEqual(landing.layout.Values, map[string]any{"content": map[string]any{"layout": "article"}}) {
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
