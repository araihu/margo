package site

import (
	"context"
	"testing"
	"time"

	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/chromedp/chromedp"
)

// TestLayoutDocsFrameResponsiveComputedStyles protects the public
// componentdocshell breakpoints. The docs renderer owns article content; the
// Goshtoso shell owns the frame, responsive sidebar, and TOC rail.
func TestLayoutDocsFrameResponsiveComputedStyles(t *testing.T) {
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
		HeaderHeight  float64 `json:"headerHeight"`
		HeaderRows    string  `json:"headerRows"`
		SidebarWidth  float64 `json:"sidebarWidth"`
		SidebarFixed  bool    `json:"sidebarFixed"`
		MenuVisible   bool    `json:"menuVisible"`
		TOCVisible    bool    `json:"tocVisible"`
		TOCWidth      float64 `json:"tocWidth"`
		TOCEnabled    string  `json:"tocEnabled"`
		TOCHidden     bool    `json:"tocHidden"`
		ViewportWidth float64 `json:"viewportWidth"`
		TOCMedia      bool    `json:"tocMedia"`
		FamilyCenter  bool    `json:"familyCenter"`
		Overflow      bool    `json:"overflow"`
	}
	const script = `(() => {
		const visible = (node) => !!node && getComputedStyle(node).display !== "none" && node.getClientRects().length > 0;
		const rect = (node) => node?.getBoundingClientRect() || { width: 0, left: 0, right: 0 };
		const header = document.querySelector('.component-doc-shell__header-inner');
		const sidebar = document.querySelector('#componentdocshell-sidebar');
		const toc = document.querySelector('[data-componentdocshell-toc]');
		const links = Array.from(document.querySelectorAll('.component-doc-shell__family-links')).find(visible);
		const headerRect = rect(header);
		return {
			headerHeight: headerRect.height,
			headerRows: header ? getComputedStyle(header).gridTemplateRows : '',
			sidebarWidth: rect(sidebar).width,
			sidebarFixed: sidebar ? getComputedStyle(sidebar).position === 'fixed' : false,
			menuVisible: visible(document.querySelector('.component-doc-shell__menu-button')),
			tocVisible: visible(toc),
			tocWidth: rect(toc).width,
			tocEnabled: toc?.dataset.enabled || '',
			tocHidden: toc?.hasAttribute('hidden') || false,
			viewportWidth: innerWidth,
			tocMedia: matchMedia('(min-width: 1280px)').matches,
			familyCenter: visible(links) && getComputedStyle(links).justifyContent.includes('center'),
			overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1 || document.body.scrollWidth > document.documentElement.clientWidth + 1,
		};
	})()`

	for _, check := range []struct {
		width            int64
		name             string
		headerHeight     float64
		wantMenu         bool
		wantSidebarFixed bool
		wantTOC          bool
		wantFamilyCenter bool
	}{
		{390, "mobile", 64, true, true, false, false},
		{720, "tablet", 108, true, true, false, false},
		{1023, "sidebar-drawer-boundary", 108, true, true, false, false},
		{1024, "sidebar-rail-boundary", 108, false, false, false, false},
		{1279, "toc-hidden-boundary", 108, false, false, false, false},
		{1280, "toc-visible-boundary", 108, false, false, true, false},
		{1439, "two-row-header-boundary", 108, false, false, true, false},
		{1440, "single-row-header-boundary", 64, false, false, true, true},
		{1498, "wide", 64, false, false, true, true},
	} {
		var got state
		if err := chromedp.Run(ctx,
			chromedp.EmulateViewport(check.width, 844),
			chromedp.Navigate(server.URL+"/cli/"),
			chromedp.WaitVisible(`[data-margo-layout="docs"].component-doc-shell`, chromedp.ByQuery),
			chromedp.Evaluate(script, &got),
		); err != nil {
			t.Fatalf("%s viewport browser check failed: %v", check.name, err)
		}
		if got.HeaderHeight < check.headerHeight-1 || got.HeaderHeight > check.headerHeight+1 || got.MenuVisible != check.wantMenu || got.SidebarFixed != check.wantSidebarFixed || got.TOCVisible != check.wantTOC || (check.wantFamilyCenter && !got.FamilyCenter) || got.Overflow {
			t.Fatalf("%s componentdocshell geometry = %+v", check.name, got)
		}
		if check.width >= 1024 && (got.SidebarWidth < 287 || got.SidebarWidth > 289) {
			t.Fatalf("%s sidebar is not the persistent 18rem shell rail: %+v", check.name, got)
		}
		if check.wantTOC && (got.TOCWidth < 239 || got.TOCWidth > 241) {
			t.Fatalf("%s TOC is not the 15rem shell rail: %+v", check.name, got)
		}
	}
}
