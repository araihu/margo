package site

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	for name, page := range map[string]string{"Tour": landing, "Module": module, "CLI": cli} {
		if strings.Count(page, "<h1") != 1 {
			t.Fatalf("%s h1 count = %d", name, strings.Count(page, "<h1"))
		}
	}
	for _, required := range []string{`href="module/index.html"`, `href="cli/index.html"`, `One source, several projections`} {
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
	for _, route := range []string{"https://margo.araihu.com/", "https://margo.araihu.com/module/index.html", "https://margo.araihu.com/cli/index.html"} {
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
		for _, route := range []string{"index.html", "module/index.html", "cli/index.html"} {
			if !strings.Contains(page, `data-search-href="/docs/`+route+`"`) && !(route == "index.html" && strings.Contains(page, `data-search-href="/docs/"`)) {
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
		`@media (min-width: 720px) and (max-width: 1099px)`,
		`grid-template-columns: minmax(10rem, 14rem) minmax(0, 1fr) minmax(10rem, 14rem);`,
		`[data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--left-nav`,
		`[data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--main-content`,
		`[data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--right-nav`,
		`@media (max-width: 719px)`,
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

func TestBuildConfigRendersLocaleScopedFamilySearch(t *testing.T) {
	root := t.TempDir()
	for _, localePrefix := range []string{"", "pt-BR"} {
		prefix := filepath.Join(root, "docs", localePrefix)
		writeConfigFile(t, filepath.Join(prefix, "index.md"), "# Tour\n\nTour documentation.\n")
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
		localRoutes []string
		otherRoutes []string
	}{
		{
			name:        "English",
			artifact:    "module/index.html",
			localRoutes: []string{"/docs/", "/docs/module/index.html", "/docs/cli/index.html"},
			otherRoutes: []string{"/docs/pt-br/index.html", "/docs/pt-br/module/index.html", "/docs/pt-br/cli/index.html"},
		},
		{
			name:        "Portuguese",
			artifact:    "pt-br/module/index.html",
			localRoutes: []string{"/docs/pt-br/index.html", "/docs/pt-br/module/index.html", "/docs/pt-br/cli/index.html"},
			otherRoutes: []string{"/docs/", "/docs/module/index.html", "/docs/cli/index.html"},
		},
	} {
		page := string(configArtifact(t, result, fixture.artifact))
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
	}
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
