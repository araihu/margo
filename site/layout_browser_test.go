package site

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

func TestLayoutBrowserDocsShellMobileInteraction(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	server := layoutBrowserServer(t)
	defer server.Close()
	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(browserPath)...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 60*time.Second)
	defer cancel()

	type state struct {
		HeaderHeight     float64 `json:"headerHeight"`
		MenuCount        int     `json:"menuCount"`
		MenuVisible      bool    `json:"menuVisible"`
		SidebarOpen      bool    `json:"sidebarOpen"`
		SidebarInert     bool    `json:"sidebarInert"`
		SidebarOffCanvas bool    `json:"sidebarOffCanvas"`
		BackdropVisible  bool    `json:"backdropVisible"`
		FocusReturned    bool    `json:"focusReturned"`
		TOCVisible       bool    `json:"tocVisible"`
		Overflow         bool    `json:"overflow"`
	}
	const stateScript = `(() => {
		const visible = (node) => !!node && getComputedStyle(node).display !== "none" && getComputedStyle(node).visibility !== "hidden" && node.getClientRects().length > 0;
		const button = document.querySelector('.component-doc-shell__menu-button');
		const sidebar = document.querySelector('#componentdocshell-sidebar');
		const rect = sidebar?.getBoundingClientRect() || { right: 0 };
		return {
			headerHeight: document.querySelector('.component-doc-shell__header-inner')?.getBoundingClientRect().height || 0,
			menuCount: document.querySelectorAll('.component-doc-shell__menu-button').length,
			menuVisible: visible(button),
			sidebarOpen: sidebar?.classList.contains('is-open') || false,
			sidebarInert: sidebar?.hasAttribute('inert') || false,
			sidebarOffCanvas: rect.right <= 1,
			backdropVisible: visible(document.querySelector('.component-doc-shell__backdrop')),
			focusReturned: document.activeElement === button,
			tocVisible: visible(document.querySelector('[data-componentdocshell-toc]')),
			overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1 || document.body.scrollWidth > document.documentElement.clientWidth + 1,
		};
	})()`
	var closed state
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(390, 844),
		chromedp.Navigate(server.URL+"/cli/"),
		chromedp.WaitVisible(`[data-margo-layout="docs"].component-doc-shell`, chromedp.ByQuery),
		chromedp.Evaluate(stateScript, &closed),
	); err != nil {
		t.Fatal(err)
	}
	if closed.HeaderHeight < 63 || closed.HeaderHeight > 65 || closed.MenuCount != 1 || !closed.MenuVisible || !closed.SidebarInert || !closed.SidebarOffCanvas || closed.SidebarOpen || closed.BackdropVisible || closed.TOCVisible || closed.Overflow {
		t.Fatalf("390px shell initial state = %+v", closed)
	}

	var opened state
	if err := chromedp.Run(ctx,
		chromedp.Click(`.component-doc-shell__menu-button`, chromedp.ByQuery),
		chromedp.Poll(`(() => {
			const sidebar = document.querySelector('#componentdocshell-sidebar');
			const backdrop = document.querySelector('.component-doc-shell__backdrop');
			return sidebar?.classList.contains('is-open') && !!backdrop?.getClientRects().length;
		})()`, nil, chromedp.WithPollingInterval(20*time.Millisecond)),
		chromedp.Evaluate(stateScript, &opened),
	); err != nil {
		t.Fatal(err)
	}
	if !opened.SidebarOpen || !opened.BackdropVisible || opened.SidebarInert || opened.Overflow {
		t.Fatalf("390px shell menu did not open drawer safely: %+v", opened)
	}

	var escaped state
	if err := chromedp.Run(ctx,
		chromedp.KeyEvent(kb.Escape),
		chromedp.Poll(`(() => {
			const sidebar = document.querySelector('#componentdocshell-sidebar');
			const backdrop = document.querySelector('.component-doc-shell__backdrop');
			const trigger = document.querySelector('.component-doc-shell__menu-button');
			return !sidebar?.classList.contains('is-open') && !backdrop?.getClientRects().length && document.activeElement === trigger;
		})()`, nil, chromedp.WithPollingInterval(20*time.Millisecond)),
		chromedp.Evaluate(stateScript, &escaped),
	); err != nil {
		t.Fatal(err)
	}
	if escaped.SidebarOpen || !escaped.SidebarInert || !escaped.SidebarOffCanvas || escaped.BackdropVisible || !escaped.FocusReturned || escaped.Overflow {
		t.Fatalf("390px shell Escape did not restore menu state/focus: %+v", escaped)
	}
}

func TestLayoutBrowserTourIsOutsideDocsShell(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	server := layoutBrowserServer(t)
	defer server.Close()
	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(browserPath)...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 45*time.Second)
	defer cancel()
	var state struct {
		Layout      string `json:"layout"`
		Heading     string `json:"heading"`
		Shell       bool   `json:"shell"`
		Sidebar     bool   `json:"sidebar"`
		TOC         bool   `json:"toc"`
		FamilyNav   bool   `json:"familyNav"`
		Pagination  bool   `json:"pagination"`
		PageActions bool   `json:"pageActions"`
		Overflow    bool   `json:"overflow"`
	}
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(390, 844),
		chromedp.Navigate(server.URL+"/"),
		chromedp.WaitVisible(`[data-margo-layout="landing"]`, chromedp.ByQuery),
		chromedp.Evaluate(`(() => ({
			layout: document.querySelector('[data-margo-layout]')?.dataset.margoLayout || '',
			heading: document.querySelector('article.margo-document h1')?.textContent.trim() || '',
			shell: !!document.querySelector('.component-doc-shell'),
			sidebar: !!document.querySelector('#componentdocshell-sidebar'),
			toc: !!document.querySelector('[data-componentdocshell-toc]'),
			familyNav: !!document.querySelector('.component-doc-shell__family-navigation'),
			pagination: !!document.querySelector('.margo-pagination'),
			pageActions: !!document.querySelector('.margo-page-actions'),
			overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1 || document.body.scrollWidth > document.documentElement.clientWidth + 1,
		}))()`, &state),
	); err != nil {
		t.Fatal(err)
	}
	if state.Layout != "landing" || state.Heading != "Tour" || state.Shell || state.Sidebar || state.TOC || state.FamilyNav || state.Pagination || state.PageActions || state.Overflow {
		t.Fatalf("Tour leaked docs shell chrome: %+v", state)
	}
}

func TestLayoutBrowserCrossFamilyNavigationRefreshesShell(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	server := layoutBrowserServer(t)
	defer server.Close()
	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(browserPath)...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 60*time.Second)
	defer cancel()

	var state struct {
		CurrentFamilies []string `json:"currentFamilies"`
		Scope           string   `json:"scope"`
		Heading         string   `json:"heading"`
	}
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(1498, 844),
		chromedp.Navigate(server.URL+"/module/"),
		chromedp.WaitVisible(`a.component-doc-shell__family-link[href="/cli/"]`, chromedp.ByQuery),
		chromedp.Click(`a.component-doc-shell__family-link[href="/cli/"]`, chromedp.ByQuery),
		chromedp.WaitVisible(`[data-sidebar-section="CLI"]`, chromedp.ByQuery),
		chromedp.Evaluate(`(() => ({
			currentFamilies: [...document.querySelectorAll('.component-doc-shell__family-link[aria-current="location"]')].map(link => link.textContent.trim()),
			scope: document.querySelector('.component-doc-shell__scope-family')?.textContent.trim() || '',
			heading: document.querySelector('#main-content h1')?.textContent.trim() || '',
		}))()`, &state),
	); err != nil {
		t.Fatal(err)
	}
	if len(state.CurrentFamilies) != 1 || state.CurrentFamilies[0] != "CLI" || state.Scope != "CLI" || state.Heading != "CLI" {
		t.Fatalf("cross-family navigation left stale shell state: %+v", state)
	}
}

type landingGeometryState struct {
	HeroWidth           float64   `json:"heroWidth"`
	CopyTop             float64   `json:"copyTop"`
	CopyRight           float64   `json:"copyRight"`
	CopyBottom          float64   `json:"copyBottom"`
	VisualTop           float64   `json:"visualTop"`
	CopyLeft            float64   `json:"copyLeft"`
	VisualLeft          float64   `json:"visualLeft"`
	ActionHeights       []float64 `json:"actionHeights"`
	ActionBottoms       []float64 `json:"actionBottoms"`
	ActionLefts         []float64 `json:"actionLefts"`
	ActionTops          []float64 `json:"actionTops"`
	ImageAspectRatio    float64   `json:"imageAspectRatio"`
	ImageCSSAspectRatio string    `json:"imageCSSAspectRatio"`
	ImageObjectFit      string    `json:"imageObjectFit"`
	SectionWidth        float64   `json:"sectionWidth"`
	SectionTextWidth    float64   `json:"sectionTextWidth"`
	SectionMediaWidth   float64   `json:"sectionMediaWidth"`
	ArticleCount        int       `json:"articleCount"`
	Overflow            bool      `json:"overflow"`
}

const landingGeometryScript = `(() => {
  const rect = (node) => {
    if (!node) return { left: 0, right: 0, top: 0, width: 0, height: 0 };
    const value = node.getBoundingClientRect();
    return { left: value.left, right: value.right, top: value.top, width: value.width, height: value.height };
  };
	  const hero = document.querySelector('.margo-landing-hero');
	  const copy = document.querySelector('.margo-landing-hero__copy');
	  const visual = document.querySelector('.margo-landing-hero__visual');
	  const actions = [...document.querySelectorAll('.margo-landing-hero__copy > ul a')];
	  const image = visual?.querySelector('img');
	  const section = document.querySelector('.margo-landing-section');
	  const sectionText = section?.querySelector(':scope > p:not(.margo-landing-media)');
	  const sectionMedia = section?.querySelector(':scope > .margo-landing-media');
	  const heroRect = rect(hero);
	  const copyRect = rect(copy);
	  const visualRect = rect(visual);
	  const imageRect = rect(image);
	  const imageStyle = image ? getComputedStyle(image) : {};
	  return {
	    heroWidth: heroRect.width,
	    copyTop: copyRect.top,
	    copyRight: copyRect.right,
	    copyBottom: copyRect.top + copyRect.height,
	    visualTop: visualRect.top,
	    copyLeft: copyRect.left,
	    visualLeft: visualRect.left,
	    actionHeights: actions.map(action => rect(action).height),
	    actionBottoms: actions.map(action => rect(action).top + rect(action).height),
	    actionLefts: actions.map(action => rect(action).left),
	    actionTops: actions.map(action => rect(action).top),
	    imageAspectRatio: imageRect.height ? imageRect.width / imageRect.height : 0,
	    imageCSSAspectRatio: imageStyle.aspectRatio || '',
	    imageObjectFit: imageStyle.objectFit || '',
	    sectionWidth: rect(section).width,
	    sectionTextWidth: rect(sectionText).width,
	    sectionMediaWidth: rect(sectionMedia).width,
	    articleCount: document.querySelectorAll('[data-margo-landing-article="true"] > article.margo-document').length,
	    overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1 || document.body.scrollWidth > document.documentElement.clientWidth + 1,
	  };
	})()`

func TestLandingLayoutVisualGeometry(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	server := layoutBrowserServer(t)
	defer server.Close()
	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(browserPath)...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 180*time.Second)
	defer cancel()
	for _, fixture := range []struct {
		name           string
		width          int64
		heroStacked    bool
		actionsStacked bool
	}{
		{name: "phone", width: 390, heroStacked: true, actionsStacked: true},
		{name: "narrow-edge", width: 719, heroStacked: true, actionsStacked: true},
		{name: "intermediate", width: 720, heroStacked: true, actionsStacked: true},
		{name: "compact-wide", width: 900, actionsStacked: true},
		{name: "wide", width: 1493},
		{name: "max-wide", width: 1775},
	} {
		width := fixture.width
		height := int64(900)
		if width < 720 {
			height = 844
		}
		var state landingGeometryState
		if err := chromedp.Run(ctx,
			chromedp.EmulateViewport(width, height),
			chromedp.Navigate(server.URL+"/"),
			chromedp.WaitVisible(`[data-margo-layout="landing"].margo-frame--main`, chromedp.ByQuery),
			chromedp.Evaluate(landingGeometryScript, &state),
		); err != nil {
			t.Fatalf("landing geometry at %dpx failed: %v", width, err)
		}
		if state.ArticleCount != 1 || state.Overflow {
			t.Fatalf("landing %s at %dpx composition = %+v, want one article and no overflow", fixture.name, width, state)
		}
		if (fixture.heroStacked && state.VisualTop < state.CopyBottom-1) || (!fixture.heroStacked && state.VisualLeft < state.CopyRight-1) {
			t.Fatalf("landing %s at %dpx hero placement = %+v", fixture.name, width, state)
		}
		if len(state.ActionHeights) != 2 || len(state.ActionBottoms) != 2 || len(state.ActionLefts) != 2 || len(state.ActionTops) != 2 {
			t.Fatalf("landing at %dpx action count = %+v", width, state)
		}
		if (fixture.actionsStacked && state.ActionTops[1] <= state.ActionTops[0]) || (!fixture.actionsStacked && state.ActionLefts[1] <= state.ActionLefts[0]) {
			t.Fatalf("landing %s at %dpx intrinsic action placement = %+v", fixture.name, width, state)
		}
		for index, actionHeight := range state.ActionHeights {
			if actionHeight < 44 || state.ActionBottoms[index] > float64(height) {
				t.Fatalf("landing at %dpx action %d misses first-viewport 44px target: %+v", width, index, state)
			}
		}
		if state.ImageAspectRatio < 1.28 || state.ImageAspectRatio > 1.38 || state.ImageCSSAspectRatio != "4 / 3" || state.ImageObjectFit != "cover" {
			t.Fatalf("landing at %dpx hero media geometry = %+v", width, state)
		}
		if !fixture.heroStacked && (state.HeroWidth < float64(width)*0.72 || state.SectionTextWidth > 760 || state.SectionMediaWidth <= state.SectionTextWidth) {
			t.Fatalf("landing at %dpx canvas/reading measure = %+v", width, state)
		}
	}
}

type searchSemanticsState struct {
	Role             string `json:"role"`
	Controls         string `json:"controls"`
	Expanded         string `json:"expanded"`
	ActiveDescendant string `json:"activeDescendant"`
	ActiveOptionID   string `json:"activeOptionID"`
	SelectedCount    int    `json:"selectedCount"`
	Status           string `json:"status"`
	ClearVisible     bool   `json:"clearVisible"`
	FocusReturned    bool   `json:"focusReturned"`
}

const searchSemanticsScript = `(() => {
  const input = document.querySelector('[data-search-modal][data-margo-search-a11y="true"] [role="combobox"]');
  const trigger = document.querySelector('[data-search-field][data-margo-search-a11y="true"] button');
  const listbox = input && document.getElementById(input.getAttribute('aria-controls'));
  const selected = listbox ? [...listbox.querySelectorAll('[role="option"][aria-selected="true"]')] : [];
  const clear = document.querySelector('[data-margo-search-clear]');
  return {
    role: input?.getAttribute('role') || '',
    controls: input?.getAttribute('aria-controls') || '',
    expanded: input?.getAttribute('aria-expanded') || '',
    activeDescendant: input?.getAttribute('aria-activedescendant') || '',
    activeOptionID: selected[0]?.id || '',
    selectedCount: selected.length,
    status: document.querySelector('[data-margo-search-status]')?.textContent || '',
    clearVisible: !!clear && getComputedStyle(clear).display !== 'none' && !clear.hidden,
    focusReturned: !!trigger && document.activeElement === trigger,
  };
})()`

func TestLayoutSearchSemanticsAndFocusReturn(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	server := layoutBrowserServer(t)
	defer server.Close()
	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(browserPath)...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 60*time.Second)
	defer cancel()
	var state searchSemanticsState
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(390, 844),
		chromedp.Navigate(server.URL+"/module/"),
		chromedp.WaitVisible(`[data-margo-layout="docs"]`, chromedp.ByQuery),
	); err != nil {
		t.Fatal(err)
	}
	if err := chromedp.Run(ctx,
		chromedp.Click(`[data-search-field][data-margo-search-a11y="true"] button`, chromedp.ByQuery),
		chromedp.WaitVisible(`[data-search-modal][data-margo-search-a11y="true"]`, chromedp.ByQuery),
		chromedp.SendKeys(`[data-search-modal][data-margo-search-a11y="true"] [role="combobox"]`, "Module", chromedp.ByQuery),
		chromedp.Poll(`(() => {
			const input = document.querySelector('[data-search-modal][data-margo-search-a11y="true"] [role="combobox"]');
			const listbox = input && document.getElementById(input.getAttribute('aria-controls'));
			return input?.getAttribute('aria-expanded') === 'true' && !!input.getAttribute('aria-activedescendant') && listbox?.querySelectorAll('[role="option"][aria-selected="true"]').length === 1;
		})()`, nil, chromedp.WithPollingInterval(20*time.Millisecond)),
		chromedp.Evaluate(searchSemanticsScript, &state),
	); err != nil {
		t.Fatal(err)
	}
	if state.Role != "combobox" || state.Controls != "margo-site-search-results" || state.Expanded != "true" || state.SelectedCount != 1 || state.ActiveDescendant != state.ActiveOptionID || state.Status == "" || !state.ClearVisible {
		t.Fatalf("search semantics after query = %+v", state)
	}
	if err := chromedp.Run(ctx,
		chromedp.KeyEvent(kb.ArrowDown),
		chromedp.Sleep(80*time.Millisecond),
		chromedp.Evaluate(searchSemanticsScript, &state),
	); err != nil {
		t.Fatal(err)
	}
	if state.SelectedCount != 1 || state.ActiveDescendant != state.ActiveOptionID {
		t.Fatalf("search active option desynchronized after ArrowDown = %+v", state)
	}
	if err := chromedp.Run(ctx,
		chromedp.Click(`[data-margo-search-clear]`, chromedp.ByQuery),
		chromedp.SendKeys(`[data-search-modal][data-margo-search-a11y="true"] [role="combobox"]`, "not-a-real-page", chromedp.ByQuery),
		chromedp.Poll(`document.querySelector('[data-margo-search-status]')?.textContent === 'No matching pages.'`, nil, chromedp.WithPollingInterval(20*time.Millisecond)),
		chromedp.Evaluate(searchSemanticsScript, &state),
	); err != nil {
		t.Fatal(err)
	}
	if state.Status != "No matching pages." || state.SelectedCount != 0 || state.ActiveDescendant != "" {
		t.Fatalf("search no-result state = %+v", state)
	}
	if err := chromedp.Run(ctx,
		chromedp.KeyEvent(kb.Escape),
		chromedp.Poll(`document.activeElement === document.querySelector('[data-search-field][data-margo-search-a11y="true"] button')`, nil, chromedp.WithPollingInterval(20*time.Millisecond)),
		chromedp.Evaluate(searchSemanticsScript, &state),
	); err != nil {
		t.Fatal(err)
	}
	if !state.FocusReturned {
		t.Fatalf("search Escape did not return focus to trigger = %+v", state)
	}
}

func layoutBrowserServer(t *testing.T) *httptest.Server {
	t.Helper()
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "---\nlayout:\n  kind: landing\n---\n# Tour\n\nPublish one Markdown source in the format your project needs.\n\nMargo turns ordinary Markdown into durable outputs.\n\n- [Publish with the CLI — standalone publishing workflow](cli/index.md)\n- [Embed the Go module — host-owned composition](module/index.md)\n\n![Margo mascot preparing a document](margo-mascot.png)\n\n## One source, several projections\n\nThe same source can serve several formats.\n\n![Several generated outputs](margo-mascot.png)\n")
	mascot, err := os.ReadFile(filepath.Join("..", "showcase", "content", "margo-mascot.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "margo-mascot.png"), mascot, 0o644); err != nil {
		t.Fatal(err)
	}
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "# Module\n\nModule documentation.\n\n## Module section\n\nDetails.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "guide.md"), "# Module guide\n\nGuide.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "index.md"), "---\nlayout:\n  values:\n    toc: true\n    sidebar: true\nmargo:\n  actions:\n    markdown: true\n---\n# CLI\n\nCLI documentation.\n\n## CLI section\n\nDetails.\n\n## CLI workflows\n\nMore details.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "guide.md"), "# CLI guide\n\nGuide.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "_layout.yaml"), "values:\n  family: module\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "_layout.yaml"), "values:\n  family: cli\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
offline: true
site:
  name: Margo
  description: Layout browser fixture.
  repository_url: https://github.com/araihu/margo
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo layout preview.
layout:
  kind: docs
  default:
    families: [module, cli]
  values:
    sidebar: true
    toc: true
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
	artifacts := make(map[string][]byte, len(result.Artifacts))
	for _, artifact := range result.Artifacts {
		artifacts["/"+strings.TrimPrefix(artifact.Path, "/")] = artifact.Content
	}
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		artifactPath := request.URL.Path
		if artifactPath == "" || artifactPath == "/" {
			artifactPath = "/index.html"
		} else if strings.HasSuffix(artifactPath, "/") {
			artifactPath += "index.html"
		}
		content, ok := artifacts[artifactPath]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", siteBrowserContentType(artifactPath))
		_, _ = writer.Write(content)
	}))
}
