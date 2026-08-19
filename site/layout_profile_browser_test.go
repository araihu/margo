package site

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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
		route      string
		name       string
		layout     string
		family     string
		sidebar    bool
		tocArea    bool
		pagination bool
	}
	cells := []browserCell{
		{route: "/", name: "Tour", layout: "landing", family: "tour"},
		{route: "/module/", name: "Module", layout: "docs", family: "module", sidebar: true, tocArea: true, pagination: true},
		{route: "/cli/", name: "CLI", layout: "docs", family: "cli", sidebar: true, tocArea: true, pagination: true},
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
					var focusState layoutProfileBrowserFocusState
					actions := chromedp.Tasks{
						runtime.Enable(),
						network.Enable(),
						emulation.SetEmulatedMedia().WithFeatures([]*emulation.MediaFeature{{Name: "prefers-color-scheme", Value: colorMode}}),
						chromedp.EmulateViewport(viewport.width, 844),
						chromedp.Navigate(server.URL + route),
						chromedp.WaitVisible(`[data-margo-layout]`, chromedp.ByQuery),
						chromedp.Evaluate(layoutProfileBrowserStateScript, &state),
					}
					focusName, focusScript := "skip link", focusDesktopScript
					if viewport.width == 390 {
						focusName, focusScript = "mobile menu trigger", focusMobileScript
					}
					actions = append(actions,
						tabUntilFocus(focusName, focusScript, &focusState),
						chromedp.Evaluate(layoutProfileBrowserStateScript, &state),
					)
					if err := chromedp.Run(ctx, actions...); err != nil {
						t.Fatalf("%s at %s/%d browser check failed: %v", route, colorMode, viewport.width, err)
					}
					if errors := readBrowserErrors(); len(errors) > 0 {
						t.Fatalf("%s at %s/%d emitted browser errors: %s", route, colorMode, viewport.width, strings.Join(errors, "; "))
					}
					if state.Path != route && !(route == "/" && state.Path == "/index.html") {
						t.Fatalf("route path = %q, want %q: %+v", state.Path, route, state)
					}
					if state.Layout != cell.layout || state.ActiveFamily != cell.family || state.ActiveFamilyCount != 1 {
						t.Fatalf("%s active layout/family = %+v, want layout=%q family=%q", cell.name, state, cell.layout, cell.family)
					}
					if state.ColorMode != colorMode {
						t.Fatalf("color mode = %q, want %q: %+v", state.ColorMode, colorMode, state)
					}
					if state.Sidebar != cell.sidebar || state.TOCArea != cell.tocArea || state.Pagination != cell.pagination || state.TOC {
						t.Fatalf("%s chrome presence = %+v, want sidebar=%t toc-area=%t pagination=%t and no TOC payload", cell.name, state, cell.sidebar, cell.tocArea, cell.pagination)
					}
					if state.MobileTriggerCount != boolToInt(viewport.width == 390) || state.MobileTriggerVisible != (viewport.width == 390) {
						t.Fatalf("mobile trigger state = %+v focus=%+v, want count/visibility for width %d", state, focusState, viewport.width)
					}
					if !focusState.Found || !focusState.Focused || !focusState.FocusVisible {
						t.Fatalf("%s lacks keyboard focus visibility after real Tab traversal: focus=%+v state=%+v", focusName, focusState, state)
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
			chromedp.Navigate(server.URL+"/"),
			chromedp.WaitVisible(`[data-margo-layout="landing"]`, chromedp.ByQuery),
			chromedp.Click(`[data-margo-family-link="module"]`, chromedp.ByQuery),
			chromedp.Evaluate(waitForFamilyRouteScript("module", "/module/index.html"), nil),
			chromedp.Click(`[data-margo-family-link="cli"]`, chromedp.ByQuery),
			chromedp.Evaluate(waitForFamilyRouteScript("cli", "/cli/index.html"), nil),
			chromedp.Click(`[data-margo-family-link="tour"]`, chromedp.ByQuery),
			chromedp.Evaluate(waitForFamilyRouteScript("tour", "/"), nil),
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
		toc: !!document.querySelector('[data-margo-area="right-nav"] nav'),
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

func layoutProfileBrowserServer(t *testing.T) *httptest.Server {
	t.Helper()
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), `# Tour

Choose a documentation family: [Module](module/index.md) or [CLI](cli/index.md).
`)
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
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
offline: true
site:
  name: Margo
  description: Layout profile browser fixture.
  repository_url: https://github.com/araihu/margo
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo layout profile preview.
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
