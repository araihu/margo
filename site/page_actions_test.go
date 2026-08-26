package site

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/charts"
	"github.com/araihu/margo/pdf"
)

type siteTestPDFEngine struct {
	captured *pdf.Request
}

var _ pdf.Engine = siteTestPDFEngine{}

func (siteTestPDFEngine) Name() string { return "site-test" }

func (siteTestPDFEngine) Version(context.Context) (string, error) { return "1.0.0", nil }

func (engine siteTestPDFEngine) Export(_ context.Context, request pdf.Request) (pdf.Result, error) {
	if engine.captured != nil {
		*engine.captured = request.Clone()
	}
	tasks := make([]margo.RuntimeTaskReport, len(request.Runtime.Tasks))
	for index, task := range request.Runtime.Tasks {
		tasks[index] = margo.RuntimeTaskReport{
			ID:           task.ID,
			Kind:         task.Kind,
			InputSHA256:  task.InputSHA256,
			OutputSHA256: strings.Repeat("a", 64),
			OutputBytes:  1,
			Status:       margo.RuntimeTaskSucceeded,
		}
	}
	return pdf.Result{
		PDF: []byte("%PDF-site-test"),
		Runtime: margo.RuntimeReport{
			Protocol:            margo.RuntimeProtocolV1,
			DocumentFingerprint: request.Runtime.DocumentFingerprint,
			RenderInstanceID:    request.Runtime.RenderInstanceID,
			ExecutionID:         request.ExecutionID,
			Status:              margo.RuntimeReady,
			Tasks:               tasks,
			FontChecks:          []margo.FontCheck{},
			BlockedRequests:     []margo.BlockedRequest{},
			Layout:              margo.LayoutMetrics{ScrollWidth: 1280, ScrollHeight: 720},
		},
		Engine: pdf.EngineInfo{Name: "site-test", Version: "1.0.0"},
	}, nil
}

func TestBuildConfigPublishesDeclaredMarkdownAndPDFActions(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, root+"/docs/index.md", `---
title: PDF guide
description: A downloadable guide.
margo:
  actions:
    pdf: true
---
# PDF guide

This page has both source and PDF outputs.
`)
	writeConfigFile(t, root+"/docs/client.md", `---
title: Client print
description: Print from the current browser theme.
margo:
  page:
    imageOverflow: allow
  actions:
    pdf: client
---
# Client print

This page uses browser printing instead of a pre-rendered PDF.
`)
	copyMargoAsset(t, root+"/assets/logo.svg", "logo.svg")
	copyMargoAsset(t, root+"/assets/social.jpg", "social/margo-social-v2.jpg")
	writeConfigFile(t, root+"/site.yaml", `version: 1
source: docs
output: dist
assets: local
site:
  name: Margo
  description: A downloadable guide.
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo guide preview
locales:
  default: en
  supported: [en]
`)

	var pdfRequest pdf.Request
	result, err := BuildConfig(context.Background(), ConfigRequest{
		ConfigPath: root + "/site.yaml",
		PDFEngine:  siteTestPDFEngine{captured: &pdfRequest},
	})
	if err != nil {
		t.Fatal(err)
	}
	page := string(configArtifact(t, result, "index.html"))
	for _, required := range []string{
		`class="margo-page-actions"`,
		`data-split-button`,
		`data-popover-panel`,
		`data-margo-copy-page`,
		`href="index.md"`,
		`href="/margo-assets/icons/page-actions.svg#heroicons-copy-16-solid-clipboard"`,
		`href="/margo-assets/icons/page-actions.svg#heroicons-document-text-16-solid-document-text"`,
		`View this page as plain text`,
		`target="_blank"`,
		`rel="noopener noreferrer"`,
		`id="margo-page-actions-index-view-markdown"`,
		`href="/margo-assets/icons/page-actions.svg#heroicons-arrow-top-right-16-solid-arrow-top-right-on-square"`,
		`href="index.pdf"`,
		`href="/margo-assets/icons/page-actions.svg#heroicons-arrow-down-tray-16-solid-arrow-down-tray"`,
		`download`,
		`Download PDF`,
		`class="margo-page-heading__anchor"`,
		`href="#pdf-guide"`,
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("configured action markup missing %q: %s", required, page)
		}
	}
	iconSprite := string(configArtifact(t, result, "margo-assets/icons/page-actions.svg"))
	for _, required := range []string{
		`<symbol id="heroicons-copy-16-solid-clipboard"`,
		`<symbol id="heroicons-document-text-16-solid-document-text"`,
		`<symbol id="heroicons-arrow-top-right-16-solid-arrow-top-right-on-square"`,
		`<symbol id="heroicons-arrow-down-tray-16-solid-arrow-down-tray"`,
		`<symbol id="heroicons-printer-16-solid-printer"`,
		`fill="currentColor"`,
	} {
		if !strings.Contains(iconSprite, required) {
			t.Fatalf("iconpack sprite missing %q: %s", required, iconSprite)
		}
	}
	if !artifactExists(result, "margo-assets/icons/page-actions.svg") {
		t.Fatalf("iconpack sprite artifact missing: %+v", result.Artifacts)
	}
	leadIndex := strings.Index(page, `class="margo-document__lead"`)
	actionsIndex := strings.Index(page, `class="margo-page-actions"`)
	if leadIndex < 0 || actionsIndex < 0 || actionsIndex < leadIndex {
		t.Fatalf("actions do not follow the lead in the heading flow: %s", page)
	}
	styles := string(configArtifact(t, result, "margo-assets/site.css"))
	for _, required := range []string{
		"grid-template-columns: minmax(0, 1fr) auto",
		"@media (max-width: 42rem)",
		".margo-page-actions [data-split-button] > a {",
		"display: inline-flex;",
		"align-items: center;",
		"justify-content: center;",
		".dark .margo-page-actions [data-split-button] > a",
		".margo-page-actions [data-split-button]",
		"left: 0 !important; right: auto !important",
		".margo-page-heading__anchor {",
		"clip: rect(0 0 0 0)",
		".margo-page-heading__anchor:focus-visible",
	} {
		if !strings.Contains(styles, required) {
			t.Fatalf("responsive action CSS missing %q", required)
		}
	}
	if got := string(configArtifact(t, result, "index.md")); !strings.Contains(got, "both source and PDF") {
		t.Fatalf("configured Markdown = %q", got)
	}
	if got := string(configArtifact(t, result, "index.pdf")); !strings.HasPrefix(got, "%PDF-") {
		t.Fatalf("configured PDF = %q", got)
	}
	for _, required := range []string{
		`class="goshtoso-document__header"`,
		`class="goshtoso-document__logo"`,
		`class="margo-pdf-brand-name">Margo</span>`,
		`class="goshtoso-document__footer"`,
		`class="margo-pdf-page-title">PDF guide</span>`,
		`class="margo-pdf-generated">Generated by Margo</span>`,
	} {
		if !strings.Contains(string(pdfRequest.HTML), required) {
			t.Fatalf("configured PDF HTML missing %q: %s", required, pdfRequest.HTML)
		}
	}
	clientPage := string(configArtifact(t, result, "client.html"))
	if !strings.Contains(clientPage, `data-margo-image-overflow="allow"`) {
		t.Fatalf("client page is missing image overflow policy: %s", clientPage)
	}
	for _, required := range []string{`data-split-button`, `data-popover-panel`, `data-margo-print-page`, `id="margo-page-actions-client-print"`, `Print / Save PDF`, `Open the browser print dialog`, `href="client.md"`, `href="/margo-assets/icons/page-actions.svg#heroicons-printer-16-solid-printer"`} {
		if !strings.Contains(clientPage, required) {
			t.Fatalf("client action markup missing %q: %s", required, clientPage)
		}
	}
	if strings.Contains(clientPage, `download`) {
		t.Fatalf("client action unexpectedly contains a download attribute: %s", clientPage)
	}
	if strings.Contains(clientPage, `href="client.pdf"`) || artifactExists(result, "client.pdf") {
		t.Fatalf("client mode unexpectedly generated a PDF: %s", clientPage)
	}
	if len(result.Site.Routes) != 2 || result.Site.Routes[0].Actions == nil || !result.Site.Routes[0].Actions.Markdown || !result.Site.Routes[0].Actions.PDF || result.Site.Routes[1].Actions == nil || !result.Site.Routes[1].Actions.UsesClientPDF() {
		t.Fatalf("route actions = %+v", result.Site.Routes)
	}
}

func TestBuildConfigLayoutKindsPublishDeclaredMarkdownAndPDFWithoutLeakingActions(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), `---
layout:
  kind: landing
margo:
  page:
    imageOverflow: allow
  actions:
    pdf: true
---
# Landing export

Landing source remains publishable.
`)
	writeConfigFile(t, filepath.Join(root, "docs", "article.md"), `---
layout:
  kind: article
margo:
  actions:
    pdf: true
---
# Article export

Article source remains publishable.
`)
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), `---
margo:
  actions:
    markdown: true
---
# Module docs

Docs keep their toolbar.
`)
	writeConfigFile(t, filepath.Join(root, "docs", "module", "_layout.yaml"), "values:\n  family: module\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
site:
  name: Margo
  description: Typed page-action fixture.
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
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{
		ConfigPath: filepath.Join(root, "site.yaml"),
		PDFEngine:  siteTestPDFEngine{},
	})
	if err != nil {
		t.Fatal(err)
	}
	for route, output := range map[string]string{"landing": "index.html", "article": "article.html"} {
		page := string(configArtifact(t, result, output))
		for _, forbidden := range []string{"margo-page-actions", pageActionsScriptPath, pageActionsIconSpritePath} {
			if strings.Contains(page, forbidden) {
				t.Fatalf("%s action UI leaked into %s: %s", forbidden, route, page)
			}
		}
		markdown := pageMarkdownOutput(output)
		pdfOutput := pagePDFOutput(output)
		if !artifactExists(result, markdown) || !artifactExists(result, pdfOutput) {
			t.Fatalf("%s declared artifacts missing: markdown=%t pdf=%t", route, artifactExists(result, markdown), artifactExists(result, pdfOutput))
		}
		if route == "landing" && !strings.Contains(page, `data-margo-image-overflow="allow"`) {
			t.Fatalf("landing lost semantic image-overflow policy: %s", page)
		}
	}
	docs := string(configArtifact(t, result, "module/index.html"))
	if !strings.Contains(docs, `class="margo-page-actions"`) || !strings.Contains(docs, pageActionsScriptPath) {
		t.Fatalf("docs page lost its action toolbar or dependency: %s", docs)
	}
	if !artifactExists(result, "module/index.md") {
		t.Fatal("docs Markdown artifact missing")
	}
}

func TestBuildConfigEmbedsLocalImagesForPreRenderedPDF(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, root+"/docs/index.md", `---
title: Illustrated guide
description: A guide with a local image.
margo:
  page:
    imageOverflow: allow
  actions:
    pdf: true
---
# Illustrated guide

![Margo mascot](mascot.png)
`)
	copyMargoAsset(t, root+"/docs/mascot.png", "margo-mascot.png")
	copyMargoAsset(t, root+"/assets/logo.svg", "logo.svg")
	copyMargoAsset(t, root+"/assets/social.jpg", "social/margo-social-v2.jpg")
	writeConfigFile(t, root+"/site.yaml", `version: 1
source: docs
output: dist
assets: local
site:
  name: Margo
  description: An illustrated guide.
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo guide preview
locales:
  default: en
  supported: [en]
`)

	var pdfRequest pdf.Request
	result, err := BuildConfig(context.Background(), ConfigRequest{
		ConfigPath: root + "/site.yaml",
		PDFEngine:  siteTestPDFEngine{captured: &pdfRequest},
	})
	if err != nil {
		t.Fatal(err)
	}
	pdfHTML := string(pdfRequest.HTML)
	if !strings.Contains(pdfHTML, "data:image/png;base64,") {
		t.Fatalf("pre-rendered PDF did not embed the local image")
	}
	if strings.Contains(pdfHTML, `src="mascot.png"`) {
		t.Fatalf("pre-rendered PDF retained a relative image URL")
	}
	if pdfRequest.Page.ImageOverflow != pdf.ImageOverflowAllow {
		t.Fatalf("PDF image overflow policy = %q", pdfRequest.Page.ImageOverflow)
	}
	if !artifactExists(result, "mascot.png") {
		t.Fatalf("site did not retain the local image asset")
	}
}

func TestBuildConfigPreRenderedPDFChartDataOptIn(t *testing.T) {
	tests := []struct {
		name    string
		action  string
		visible bool
	}{
		{name: "default hidden", action: "pdf: true", visible: false},
		{name: "explicitly visible", action: "pdf:\n      printChartData: true", visible: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeConfigFile(t, filepath.Join(root, "docs", "index.md"), `---
title: Chart report
description: A report with exact chart data.
margo:
  actions:
    `+test.action+`
---
# Chart report

~~~goshtosochart
schemaVersion: 1
type: bar
title: Capital allocation
categories: [Grid, Water]
series:
  - name: Plan
    values: [18, 12]
~~~
`)
			copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
			copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
			writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
output: dist
assets: local
site:
  name: Margo Reports
  description: Reports with charts.
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo reports preview
locales:
  default: en
  supported: [en]
`)

			var pdfRequest pdf.Request
			_, err := BuildConfig(context.Background(), ConfigRequest{
				ConfigPath: filepath.Join(root, "site.yaml"),
				Compiler:   margo.New(margo.WithExtension(charts.Extension(charts.WithExternalizedControlRuntime(true)))),
				PDFEngine:  siteTestPDFEngine{captured: &pdfRequest},
			})
			if err != nil {
				t.Fatal(err)
			}
			pdfHTML := string(pdfRequest.HTML)
			if !strings.Contains(pdfHTML, `data-margo-chart-data="v1"`) {
				t.Fatalf("pre-rendered PDF omitted the exact chart table: %s", pdfHTML)
			}
			enabledIndex := strings.LastIndex(pdfHTML, `data-margo-chart-print-data="enabled"`)
			disabledIndex := strings.Index(pdfHTML, `data-margo-chart-print-data="disabled"`)
			if disabledIndex < 0 {
				t.Fatalf("chart print policy marker missing: %s", pdfHTML)
			}
			if test.visible {
				if enabledIndex < 0 || enabledIndex < strings.Index(pdfHTML, `data-margo-chart-data="v1"`) {
					t.Fatalf("opt-in chart print stylesheet missing or precedes the table: %s", pdfHTML)
				}
			} else if enabledIndex >= 0 {
				t.Fatalf("default PDF unexpectedly enabled chart data: %s", pdfHTML)
			}
		})
	}
}

func artifactExists(result Result, name string) bool {
	for _, artifact := range result.Artifacts {
		if artifact.Path == name {
			return true
		}
	}
	return false
}

func TestPageActionIDsAreUniqueAcrossHTMXPageSwaps(t *testing.T) {
	first := pageActionIDsFor(Page{Output: "guide.html"})
	second := pageActionIDsFor(Page{Output: "reference/guide.html"})
	if first.root == second.root {
		t.Fatalf("page action roots collide: %q", first.root)
	}
	firstPanel := first.root + "-menu-panel"
	secondPanel := second.root + "-menu-panel"
	if firstPanel == secondPanel {
		t.Fatalf("page action panels collide: %q", firstPanel)
	}
	if firstPanel != "margo-page-actions-guide-menu-panel" {
		t.Fatalf("unexpected first panel ID: %q", firstPanel)
	}
	if secondPanel != "margo-page-actions-reference-guide-menu-panel" {
		t.Fatalf("unexpected second panel ID: %q", secondPanel)
	}
}
