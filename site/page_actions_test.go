package site

import (
	"context"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
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

func TestBuildInlineSiteEmbedsPageActionIconSprite(t *testing.T) {
	result, err := Build(context.Background(), Request{
		Sources: []Source{{Path: "guide.md", Content: []byte(`---
title: Guide
margo:
  actions:
    markdown: true
---
# Guide

The source stays available beside the rendered page.
`)}},
		Compiler: margo.New(), Assets: AssetsInline,
	})
	if err != nil {
		t.Fatal(err)
	}
	page := artifactContent(t, result, "guide.html")
	for _, required := range []string{
		`href="#heroicons-copy-16-solid-clipboard"`,
		`href="#heroicons-document-text-16-solid-document-text"`,
		`<svg xmlns="http://www.w3.org/2000/svg" hidden="" aria-hidden="true">`,
		`<symbol id="heroicons-copy-16-solid-clipboard"`,
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("inline page action markup missing %q: %s", required, page)
		}
	}
	if strings.Contains(page, pageActionsIconSpritePath) || artifactExists(result, pageActionsIconSpritePath) {
		t.Fatalf("inline page unexpectedly publishes an external icon sprite: %s", page)
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
