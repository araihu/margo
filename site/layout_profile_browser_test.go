package site

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

func TestLayoutProfileBrowser(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Fatal("layout profile browser acceptance requires an installed Chromium-family browser")
	}

	server := layoutProfileBrowserServer(t)
	defer server.Close()
	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(browserPath)...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 120*time.Second)
	defer cancel()

	var errorMu sync.Mutex
	var browserErrors []string
	chromedp.ListenTarget(browserContext, func(event interface{}) {
		var message string
		switch event := event.(type) {
		case *runtime.EventConsoleAPICalled:
			if event.Type != runtime.APITypeError {
				return
			}
			message = "console.error"
			if len(event.Args) > 0 && event.Args[0].Description != "" {
				message += ": " + event.Args[0].Description
			}
		case *runtime.EventExceptionThrown:
			message = "uncaught exception"
			if event.ExceptionDetails != nil && event.ExceptionDetails.Text != "" {
				message += ": " + event.ExceptionDetails.Text
			}
		case *network.EventLoadingFailed:
			if event.Canceled || event.ErrorText == "" {
				return
			}
			message = "network load failed: " + event.ErrorText
		default:
			return
		}
		errorMu.Lock()
		browserErrors = append(browserErrors, message)
		errorMu.Unlock()
	})

	type browserCell struct {
		route       string
		name        string
		layout      string
		family      string
		sidebar     bool
		tocArea     bool
		pagination  bool
		familyCount int
	}
	cells := []browserCell{
		{route: "/", name: "Tour", layout: "landing"},
		{route: "/module/", name: "Module", layout: "docs", family: "module", familyCount: 1, sidebar: true, tocArea: true, pagination: true},
		{route: "/cli/", name: "CLI", layout: "docs", family: "cli", familyCount: 1, sidebar: true, tocArea: true, pagination: true},
	}
	for _, colorMode := range []string{"light", "dark"} {
		for _, viewport := range []struct {
			name  string
			width int64
		}{
			{name: "mobile", width: 390},
			{name: "desktop", width: 1440},
		} {
			for _, cell := range cells {
				cell := cell
				route := cell.route
				t.Run(fmt.Sprintf("%s/%s/%s", colorMode, viewport.name, cell.name), func(t *testing.T) {
					resetBrowserErrors := func() {
						errorMu.Lock()
						browserErrors = nil
						errorMu.Unlock()
					}
					readBrowserErrors := func() []string {
						errorMu.Lock()
						defer errorMu.Unlock()
						return append([]string(nil), browserErrors...)
					}
					resetBrowserErrors()

					var state layoutProfileBrowserState
					actions := chromedp.Tasks{
						runtime.Enable(),
						network.Enable(),
						emulation.SetEmulatedMedia().WithFeatures([]*emulation.MediaFeature{{Name: "prefers-color-scheme", Value: colorMode}}),
						chromedp.EmulateViewport(viewport.width, 844),
						chromedp.Navigate(server.URL + route),
						chromedp.WaitVisible(`[data-margo-layout]`, chromedp.ByQuery),
						chromedp.Evaluate(layoutProfileBrowserStateScript, &state),
					}
					if err := chromedp.Run(ctx, actions...); err != nil {
						t.Fatalf("%s at %s/%d browser check failed: %v", route, colorMode, viewport.width, err)
					}
					if errors := readBrowserErrors(); len(errors) > 0 {
						t.Fatalf("%s at %s/%d emitted browser errors: %s", route, colorMode, viewport.width, strings.Join(errors, "; "))
					}
					if state.Path != route && !(route == "/" && state.Path == "/index.html") {
						t.Fatalf("route path = %q, want %q: %+v", state.Path, route, state)
					}
					if state.Layout != cell.layout || state.ActiveFamily != cell.family || state.ActiveFamilyCount != cell.familyCount {
						t.Fatalf("%s active layout/family = %+v, want layout=%q family=%q", cell.name, state, cell.layout, cell.family)
					}
					if state.ColorMode != colorMode {
						t.Fatalf("color mode = %q, want %q: %+v", state.ColorMode, colorMode, state)
					}
					if state.Sidebar != cell.sidebar || state.TOCArea != cell.tocArea || state.Pagination != cell.pagination || state.TOC != cell.tocArea || (cell.tocArea && state.TOCLinks == 0) {
						t.Fatalf("%s chrome presence = %+v, want sidebar=%t toc-area=%t pagination=%t and a usable docs TOC", cell.name, state, cell.sidebar, cell.tocArea, cell.pagination)
					}
					wantMobileTrigger := viewport.width == 390 && cell.layout == "docs"
					if state.MobileTriggerCount != boolToInt(wantMobileTrigger) || state.MobileTriggerVisible != wantMobileTrigger {
						t.Fatalf("mobile trigger state = %+v, want count/visibility for width %d", state, viewport.width)
					}
					if state.DocumentOverflow || state.FrameOverflow {
						t.Fatalf("horizontal overflow at %s/%d: %+v", route, viewport.width, state)
					}
				})
			}
		}
	}

	t.Run("normal navigation retains active family", func(t *testing.T) {
		errorMu.Lock()
		browserErrors = nil
		errorMu.Unlock()
		if err := chromedp.Run(ctx,
			runtime.Enable(),
			network.Enable(),
			emulation.SetEmulatedMedia().WithFeatures([]*emulation.MediaFeature{{Name: "prefers-color-scheme", Value: "light"}}),
			chromedp.EmulateViewport(390, 844),
			chromedp.Navigate(server.URL+"/module/"),
			chromedp.WaitVisible(`[data-margo-layout="docs"]`, chromedp.ByQuery),
			chromedp.Click(`[data-margo-family-link="cli"]`, chromedp.ByQuery),
			chromedp.Evaluate(waitForFamilyRouteScript("cli", "/cli/"), nil),
		); err != nil {
			t.Fatal(err)
		}
		errorMu.Lock()
		errors := append([]string(nil), browserErrors...)
		errorMu.Unlock()
		if len(errors) > 0 {
			t.Fatalf("normal navigation emitted browser errors: %s", strings.Join(errors, "; "))
		}
	})
}

type layoutProfileBrowserState struct {
	Path                 string  `json:"path"`
	Layout               string  `json:"layout"`
	ActiveFamily         string  `json:"activeFamily"`
	ActiveFamilyCount    int     `json:"activeFamilyCount"`
	ColorMode            string  `json:"colorMode"`
	Sidebar              bool    `json:"sidebar"`
	TOCArea              bool    `json:"tocArea"`
	TOC                  bool    `json:"toc"`
	TOCLinks             int     `json:"tocLinks"`
	Pagination           bool    `json:"pagination"`
	MobileTriggerCount   int     `json:"mobileTriggerCount"`
	MobileTriggerVisible bool    `json:"mobileTriggerVisible"`
	DocumentOverflow     bool    `json:"documentOverflow"`
	FrameOverflow        bool    `json:"frameOverflow"`
	DocumentClientWidth  float64 `json:"documentClientWidth"`
	DocumentScrollWidth  float64 `json:"documentScrollWidth"`
	FrameClientWidth     float64 `json:"frameClientWidth"`
	FrameScrollWidth     float64 `json:"frameScrollWidth"`
}

type layoutProfileBrowserFocusState struct {
	Found        bool `json:"found"`
	Focused      bool `json:"focused"`
	FocusVisible bool `json:"focusVisible"`
	Steps        int  `json:"steps"`
}

func tabUntilFocus(name, script string, state *layoutProfileBrowserFocusState) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		const maxTabPresses = 64
		for step := 1; step <= maxTabPresses; step++ {
			if err := chromedp.KeyEvent(kb.Tab).Do(ctx); err != nil {
				return fmt.Errorf("Tab traversal toward %s failed at press %d: %w", name, step, err)
			}
			if err := chromedp.Evaluate(script, state).Do(ctx); err != nil {
				return fmt.Errorf("reading focus state for %s failed at press %d: %w", name, step, err)
			}
			state.Steps = step
			if state.Found && state.Focused {
				return nil
			}
		}
		return fmt.Errorf("Tab traversal did not reach %s after %d presses: %+v", name, maxTabPresses, *state)
	})
}

const layoutProfileBrowserStateScript = `(() => {
  const visible = (element) => {
    if (!element) return false;
    const style = getComputedStyle(element);
    const rect = element.getBoundingClientRect();
    return style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
  };
  const familyLinks = [...document.querySelectorAll("[data-margo-family-link]")];
  const mobileTriggers = familyLinks.length === 0 ? [] : [...document.querySelectorAll("[data-margo-global-navigation] button")]
    .filter((button) => (button.getAttribute("aria-label") || "").toLowerCase().includes("mobile menu"));
	const frame = document.querySelector("[data-margo-layout]");
  return {
    path: window.location.pathname,
    layout: frame?.dataset.margoLayout || "",
    activeFamily: familyLinks.find((link) => link.getAttribute("aria-current") === "location")?.dataset.margoFamilyLink || "",
    activeFamilyCount: familyLinks.filter((link) => link.getAttribute("aria-current") === "location").length,
		colorMode: document.documentElement.dataset.colorMode || "",
		sidebar: !!document.querySelector('[data-margo-area="left-nav"] nav[aria-label="sidebar navigation"]'),
		tocArea: !!document.querySelector('[data-margo-area="right-nav"]'),
		toc: !!document.querySelector('[data-margo-area="right-nav"] [data-margo-toc="true"]'),
		tocLinks: document.querySelectorAll('[data-margo-area="right-nav"] [data-margo-toc-link]').length,
    pagination: !!document.querySelector('[data-margo-area="main-content"] .margo-pagination'),
    mobileTriggerCount: mobileTriggers.filter(visible).length,
    mobileTriggerVisible: mobileTriggers.some(visible),
		documentOverflow: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1 || document.body.scrollWidth > document.body.clientWidth + 1,
		frameOverflow: !!frame && frame.scrollWidth > frame.clientWidth + 1,
		documentClientWidth: document.documentElement.clientWidth,
		documentScrollWidth: document.documentElement.scrollWidth,
		frameClientWidth: frame?.clientWidth || 0,
		frameScrollWidth: frame?.scrollWidth || 0,
  };
})()`

const focusMobileScript = `(() => {
  const trigger = [...document.querySelectorAll("[data-margo-global-navigation] button")]
    .find((button) => {
      const label = (button.getAttribute("aria-label") || "").toLowerCase();
      const style = getComputedStyle(button);
      return label.includes("mobile menu") && style.display !== "none" && style.visibility !== "hidden" && button.getClientRects().length > 0;
    });
  if (!trigger) return { found: false, focused: false, focusVisible: false };
  const style = getComputedStyle(trigger);
  return {
    found: true,
    focused: document.activeElement === trigger,
    focusVisible: trigger.matches(":focus-visible") && style.outlineStyle !== "none" && parseFloat(style.outlineWidth) > 0,
  };
})()`

const focusDesktopScript = `(() => {
  const target = document.querySelector(".margo-skip-link");
  if (!target) return { found: false, focused: false, focusVisible: false };
  const style = getComputedStyle(target);
  return {
    found: true,
    focused: document.activeElement === target,
    focusVisible: target.matches(":focus-visible") && style.outlineStyle !== "none" && parseFloat(style.outlineWidth) > 0,
  };
})()`

func waitForFamilyRouteScript(family, route string) string {
	return fmt.Sprintf(`(async () => {
  const deadline = Date.now() + 10000;
  while (Date.now() < deadline) {
    const frame = document.querySelector("[data-margo-layout]");
    const active = document.querySelector('[data-margo-family-link="%s"][aria-current="location"]');
    const path = window.location.pathname;
    if (frame && active && (path === "%s" || (path === "/index.html" && "%s" === "/"))) return true;
    await new Promise((resolve) => setTimeout(resolve, 25));
  }
  throw new Error("family navigation did not settle for %s");
})()`, family, route, route, family)
}

type landingGeometryState struct {
	Display              string  `json:"display"`
	GridColumns          string  `json:"gridColumns"`
	GridAreas            string  `json:"gridAreas"`
	FrameLeft            float64 `json:"frameLeft"`
	FrameRight           float64 `json:"frameRight"`
	MainLeft             float64 `json:"mainLeft"`
	MainRight            float64 `json:"mainRight"`
	BrandWidth           float64 `json:"brandWidth"`
	BrandHeight          float64 `json:"brandHeight"`
	BrandLabelRight      float64 `json:"brandLabelRight"`
	SearchContainerWidth float64 `json:"searchContainerWidth"`
	SearchLeft           float64 `json:"searchLeft"`
	SearchRight          float64 `json:"searchRight"`
	SearchWidth          float64 `json:"searchWidth"`
	RepositoryLeft       float64 `json:"repositoryLeft"`
	RepositoryIcon       bool    `json:"repositoryIcon"`
	ImageWidth           float64 `json:"imageWidth"`
	ImageHeight          float64 `json:"imageHeight"`
	ImageAspectRatio     float64 `json:"imageAspectRatio"`
	ImageCSSAspectRatio  string  `json:"imageCSSAspectRatio"`
	ImageObjectFit       string  `json:"imageObjectFit"`
	CTAExists            bool    `json:"ctaExists"`
	CTAInViewport        bool    `json:"ctaInViewport"`
	CTATop               float64 `json:"ctaTop"`
	CTAHeight            float64 `json:"ctaHeight"`
	ShowcaseWrapper      bool    `json:"showcaseWrapper"`
}

const landingGeometryScript = `(() => {
  const rect = (node) => {
    if (!node) return { left: 0, right: 0, top: 0, width: 0, height: 0 };
    const value = node.getBoundingClientRect();
    return { left: value.left, right: value.right, top: value.top, width: value.width, height: value.height };
  };
	const frame = document.querySelector('[data-margo-layout="landing"].margo-frame--main');
  const main = document.querySelector('[data-margo-area="main-content"]');
  const brand = document.querySelector('[data-margo-global-navigation] .margo-site-brand');
  const brandLabel = brand && brand.lastElementChild;
  const search = document.querySelector('[data-search-field] button');
  const repository = document.querySelector('[data-margo-global-navigation] [data-margo-repository-link="true"]');
  const searchContainer = document.querySelector('[data-margo-global-navigation] .margo-site-search');
  const image = document.querySelector('[data-margo-layout="landing"] img[alt^="Margo mascot"]');
  const cta = document.querySelector('[data-margo-layout="landing"] .margo-document a');
  const frameRect = rect(frame);
  const mainRect = rect(main);
  const imageRect = rect(image);
  const ctaRect = rect(cta);
  const viewportHeight = Math.max(window.innerHeight || 0, document.documentElement.clientHeight || 0, 900);
  const style = frame ? getComputedStyle(frame) : {};
  const imageStyle = image ? getComputedStyle(image) : {};
  return {
    display: style.display || '',
    gridColumns: style.gridTemplateColumns || '',
    gridAreas: style.gridTemplateAreas || '',
    frameLeft: frameRect.left,
    frameRight: frameRect.right,
    mainLeft: mainRect.left,
    mainRight: mainRect.right,
    brandWidth: rect(brand).width,
    brandHeight: rect(brand).height,
    brandLabelRight: rect(brandLabel).right,
    searchContainerWidth: rect(searchContainer).width,
    searchLeft: rect(search).left,
    searchRight: rect(search).right,
    searchWidth: rect(search).width,
    repositoryLeft: rect(repository).left,
    repositoryIcon: !!repository && repository.getAttribute('aria-label') === 'Repository' && !!repository.querySelector('svg[aria-hidden="true"]'),
    imageWidth: imageRect.width,
    imageHeight: imageRect.height,
    imageAspectRatio: imageRect.height ? imageRect.width / imageRect.height : 0,
    imageCSSAspectRatio: imageStyle.aspectRatio || '',
    imageObjectFit: imageStyle.objectFit || '',
    ctaExists: !!cta,
    ctaInViewport: !!cta && ctaRect.height > 0 && ctaRect.top < viewportHeight && ctaRect.top + ctaRect.height > 0,
    ctaTop: ctaRect.top,
    ctaHeight: ctaRect.height,
		showcaseWrapper: !!document.querySelector('[data-margo-landing-article="true"] > article.margo-document'),
  };
})()`

func TestLandingProfileVisualGeometry(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	server := layoutProfileBrowserServer(t)
	defer server.Close()
	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(browserPath)...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 90*time.Second)
	defer cancel()
	for _, width := range []int64{390, 719, 720, 900, 1280, 1775} {
		var state landingGeometryState
		if err := chromedp.Run(ctx,
			chromedp.EmulateViewport(width, 900),
			chromedp.Navigate(server.URL+"/"),
			chromedp.WaitVisible(`[data-margo-layout="landing"].margo-frame--main`, chromedp.ByQuery),
			chromedp.Evaluate(landingGeometryScript, &state),
		); err != nil {
			t.Fatalf("landing geometry at %dpx failed: %v", width, err)
		}
		if state.Display != "block" {
			t.Fatalf("landing at %dpx did not use one-column block composition: %+v", width, state)
		}
		if state.MainLeft < state.FrameLeft-1 || state.MainRight > state.FrameRight+1 {
			t.Fatalf("landing at %dpx main content escapes frame: %+v", width, state)
		}
		if state.ImageWidth > 386 || state.ImageAspectRatio < 1.28 || state.ImageAspectRatio > 1.38 || state.ImageCSSAspectRatio != "4 / 3" || state.ImageObjectFit != "cover" {
			t.Fatalf("landing at %dpx hero geometry = %+v, want compact 4:3 cover", width, state)
		}
		if !state.CTAExists || !state.CTAInViewport || state.CTAHeight <= 0 || !state.ShowcaseWrapper {
			t.Fatalf("landing at %dpx first-viewport CTA/wrapper = %+v", width, state)
		}
	}
}

type docsGeometryState struct {
	Display         string  `json:"display"`
	GridColumns     string  `json:"gridColumns"`
	GridAreas       string  `json:"gridAreas"`
	LeftNavWidth    float64 `json:"leftNavWidth"`
	MainWidth       float64 `json:"mainWidth"`
	RightNavWidth   float64 `json:"rightNavWidth"`
	HeadingWidth    float64 `json:"headingWidth"`
	LeadWidth       float64 `json:"leadWidth"`
	HeadingFontSize float64 `json:"headingFontSize"`
	LeadFontSize    float64 `json:"leadFontSize"`
}

const docsGeometryScript = `(() => {
  const frame = document.querySelector('[data-margo-layout="docs"].margo-frame--top-left-main-right-footer');
  const rect = (node) => node ? node.getBoundingClientRect() : { width: 0 };
  const style = frame ? getComputedStyle(frame) : {};
  const heading = document.querySelector('[data-margo-area="main-content"] .margo-document h1');
  const lead = document.querySelector('[data-margo-area="main-content"] .margo-document__lead');
  const headingStyle = heading ? getComputedStyle(heading) : {};
  const leadStyle = lead ? getComputedStyle(lead) : {};
  return {
    display: style.display || '',
    gridColumns: style.gridTemplateColumns || '',
    gridAreas: style.gridTemplateAreas || '',
    leftNavWidth: rect(document.querySelector('[data-margo-area="left-nav"]')).width,
    mainWidth: rect(document.querySelector('[data-margo-area="main-content"]')).width,
    rightNavWidth: rect(document.querySelector('[data-margo-area="right-nav"]')).width,
    headingWidth: rect(heading).width,
    leadWidth: rect(lead).width,
    headingFontSize: parseFloat(headingStyle.fontSize || '0'),
    leadFontSize: parseFloat(leadStyle.fontSize || '0'),
  };
})()`

func TestDocsProfileGridTemplate(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	server := layoutProfileBrowserServer(t)
	defer server.Close()
	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(browserPath)...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 90*time.Second)
	defer cancel()
	for _, width := range []int64{719, 720, 799, 800, 879, 880, 884, 900, 1100, 1200, 1280, 1775} {
		var state docsGeometryState
		if err := chromedp.Run(ctx,
			chromedp.EmulateViewport(width, 900),
			chromedp.Navigate(server.URL+"/module/"),
			chromedp.WaitVisible(`[data-margo-layout="docs"].margo-frame--top-left-main-right-footer`, chromedp.ByQuery),
			chromedp.Evaluate(docsGeometryScript, &state),
		); err != nil {
			t.Fatalf("docs geometry at %dpx failed: %v", width, err)
		}
		if state.Display != "grid" {
			t.Fatalf("docs at %dpx display = %q, want grid: %+v", width, state.Display, state)
		}
		if width < 880 {
			if strings.Count(strings.TrimSpace(state.GridColumns), " ")+1 != 1 {
				t.Fatalf("docs at %dpx did not collapse to one grid track: %+v", width, state)
			}
			if !strings.Contains(state.GridAreas, `"top-nav"`) || !strings.Contains(state.GridAreas, `"left-nav"`) || !strings.Contains(state.GridAreas, `"main-content"`) || !strings.Contains(state.GridAreas, `"right-nav"`) {
				t.Fatalf("docs at %dpx missing stacked navigation/content areas: %+v", width, state)
			}
			if state.MainWidth < 320 {
				t.Fatalf("docs at %dpx stacked content is not usable: %+v", width, state)
			}
			continue
		}
		if strings.Count(strings.TrimSpace(state.GridColumns), " ")+1 != 3 || !strings.Contains(state.GridAreas, `"left-nav main-content right-nav"`) {
			t.Fatalf("docs at %dpx did not retain three-column grid: %+v", width, state)
		}
		if state.LeftNavWidth <= 0 || state.MainWidth < 320 || state.RightNavWidth <= 0 {
			t.Fatalf("docs at %dpx grid tracks lack a usable reading column: %+v", width, state)
		}
		if state.HeadingWidth < 180 || state.LeadWidth < 220 || state.HeadingFontSize > 64 || state.LeadFontSize > 32 {
			t.Fatalf("docs at %dpx typography or page-heading geometry is not readable: %+v", width, state)
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

func TestProfileSearchSemanticsAndFocusReturn(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	server := layoutProfileBrowserServer(t)
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
		chromedp.Sleep(120*time.Millisecond),
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
		chromedp.Sleep(80*time.Millisecond),
		chromedp.SendKeys(`[data-search-modal][data-margo-search-a11y="true"] [role="combobox"]`, "not-a-real-page", chromedp.ByQuery),
		chromedp.Sleep(120*time.Millisecond),
		chromedp.Evaluate(searchSemanticsScript, &state),
	); err != nil {
		t.Fatal(err)
	}
	if state.Status != "No matching pages." || state.SelectedCount != 0 || state.ActiveDescendant != "" {
		t.Fatalf("search no-result state = %+v", state)
	}
	if err := chromedp.Run(ctx,
		chromedp.KeyEvent(kb.Escape),
		chromedp.Sleep(400*time.Millisecond),
		chromedp.Evaluate(searchSemanticsScript, &state),
	); err != nil {
		t.Fatal(err)
	}
	if !state.FocusReturned {
		t.Fatalf("search Escape did not return focus to trigger = %+v", state)
	}
}

func layoutProfileBrowserServer(t *testing.T) *httptest.Server {
	t.Helper()
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), `---
layout:
  kind: landing
---
# Tour

Publish one Markdown source in the format your project needs.

**Choose a starting path:** [Start with the CLI](cli/index.md) or [embed the Go module](module/index.md).

![Margo mascot preparing a document](margo-mascot.png)
`)
	mascot, err := os.ReadFile(filepath.Join("..", "showcase", "content", "margo-mascot.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "margo-mascot.png"), mascot, 0o644); err != nil {
		t.Fatal(err)
	}
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), `# Module

Module documentation overview.

## Module section

The module family is navigable.
`)
	writeConfigFile(t, filepath.Join(root, "docs", "module", "guide.md"), "# Module guide\n\nA second module page for scoped pagination.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "index.md"), `# CLI

CLI documentation overview.

## CLI section

The CLI family is navigable.
`)
	writeConfigFile(t, filepath.Join(root, "docs", "cli", "guide.md"), "# CLI guide\n\nA second CLI page for scoped pagination.\n")
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

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
