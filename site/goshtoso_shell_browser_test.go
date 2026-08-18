package site

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/chromedp/chromedp"
)

func TestGoshtosoShellModeToggleSurvivesHTMXNavigation(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}

	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "showcase", "index.md"), "# Home\n\nA shell page for the mode-toggle browser check.\n")
	writeConfigFile(t, filepath.Join(root, "showcase", "guide.md"), "# Guide\n\nA second page loaded through HTMX.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: showcase
assets: local
site:
  name: Margo
  description: A browser fixture.
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo browser fixture.
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
	artifacts := make(map[string][]byte, len(result.Artifacts))
	for _, artifact := range result.Artifacts {
		artifacts["/"+strings.TrimPrefix(artifact.Path, "/")] = artifact.Content
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		artifactPath := request.URL.Path
		if artifactPath == "/" {
			artifactPath = "/index.html"
		}
		content, ok := artifacts[artifactPath]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", siteBrowserContentType(artifactPath))
		_, _ = writer.Write(content)
	}))
	defer server.Close()

	allocatorOptions := siteTestChromiumAllocatorOptions(browserPath)
	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), allocatorOptions...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 25*time.Second)
	defer cancel()

	type modeState struct {
		ButtonVisible bool   `json:"buttonVisible"`
		IconVisible   bool   `json:"iconVisible"`
		Dark          bool   `json:"dark"`
		Label         string `json:"label"`
	}
	readModeState := `(() => {
		const button = document.querySelector('#componentdocshell-dark-mode');
		const icons = button ? [...button.querySelectorAll('span')] : [];
		return {
			buttonVisible: !!button && getComputedStyle(button).display !== 'none' && button.getBoundingClientRect().width > 0,
			iconVisible: icons.some((icon) => getComputedStyle(icon).display !== 'none' && icon.getBoundingClientRect().width > 0),
			dark: document.documentElement.classList.contains('dark'),
			label: button ? button.getAttribute('aria-label') || '' : '',
		};
	})()`
	var initial, toggled, navigated modeState
	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL+"/"),
		chromedp.WaitVisible(`#componentdocshell-dark-mode`, chromedp.ByQuery),
		chromedp.Evaluate(readModeState, &initial),
		chromedp.Click(`#componentdocshell-dark-mode`, chromedp.ByQuery),
		chromedp.Evaluate(`(async () => {
			const before = document.documentElement.classList.contains('dark');
			const deadline = Date.now() + 5000;
			while (Date.now() < deadline) {
				if (document.documentElement.classList.contains('dark') !== before) return true;
				await new Promise((resolve) => setTimeout(resolve, 25));
			}
			throw new Error('mode toggle did not update document state');
		})()`, nil),
		chromedp.Evaluate(readModeState, &toggled),
		chromedp.Click(`a[href="/guide.html"]`, chromedp.ByQuery),
		chromedp.Evaluate(`(async () => {
			const deadline = Date.now() + 5000;
			while (Date.now() < deadline) {
				const heading = document.querySelector('#main-content h1');
				if (heading && heading.textContent.trim() === 'Guide') return true;
				await new Promise((resolve) => setTimeout(resolve, 50));
			}
			throw new Error('HTMX navigation did not render Guide');
		})()`, nil),
		chromedp.Evaluate(readModeState, &navigated),
	); err != nil {
		t.Fatal(err)
	}
	if !initial.ButtonVisible || !initial.IconVisible {
		t.Fatalf("mode toggle is not visible on initial load: %+v", initial)
	}
	if !toggled.ButtonVisible || !toggled.IconVisible || toggled.Dark == initial.Dark || toggled.Label == initial.Label {
		t.Fatalf("mode toggle did not change state: initial=%+v toggled=%+v", initial, toggled)
	}
	if !navigated.ButtonVisible || !navigated.IconVisible || navigated.Dark != toggled.Dark || navigated.Label != toggled.Label {
		t.Fatalf("mode toggle did not survive HTMX navigation: toggled=%+v navigated=%+v", toggled, navigated)
	}
}

func TestGoshtosoShellSearchSupportsShortcutAndFullViewportNavigation(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}

	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "showcase", "index.md"), "# Home\n\nA shell page for the search browser check.\n")
	writeConfigFile(t, filepath.Join(root, "showcase", "guide.md"), "# Guide\n\nA second page loaded through the search dialog.\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: showcase
assets: local
site:
  name: Margo
  description: A browser fixture.
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo browser fixture.
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
	artifacts := make(map[string][]byte, len(result.Artifacts))
	for _, artifact := range result.Artifacts {
		artifacts["/"+strings.TrimPrefix(artifact.Path, "/")] = artifact.Content
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		artifactPath := request.URL.Path
		if artifactPath == "/" {
			artifactPath = "/index.html"
		}
		content, ok := artifacts[artifactPath]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", siteBrowserContentType(artifactPath))
		_, _ = writer.Write(content)
	}))
	defer server.Close()

	allocatorOptions := siteTestChromiumAllocatorOptions(browserPath)
	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), allocatorOptions...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 25*time.Second)
	defer cancel()
	var modalState struct {
		Width          float64 `json:"width"`
		Height         float64 `json:"height"`
		ViewportWidth  float64 `json:"viewportWidth"`
		ViewportHeight float64 `json:"viewportHeight"`
		InsideSidebar  bool    `json:"insideSidebar"`
	}

	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL+"/"),
		chromedp.WaitVisible(`[data-search-field] button`, chromedp.ByQuery),
		chromedp.Evaluate(`window.dispatchEvent(new KeyboardEvent("keydown", {key: "k", ctrlKey: true, bubbles: true}))`, nil),
		chromedp.WaitVisible(`#margo-doc-search-dialog`, chromedp.ByQuery),
		chromedp.Evaluate(`(() => {
			const modal = document.querySelector('#margo-doc-search-dialog');
			const rect = modal?.getBoundingClientRect();
			return {
				width: rect?.width || 0,
				height: rect?.height || 0,
				viewportWidth: window.innerWidth,
				viewportHeight: window.innerHeight,
				insideSidebar: !!modal?.closest('#componentdocshell-sidebar-content'),
			};
		})()`, &modalState),
		chromedp.SendKeys(`#margo-doc-search-input`, "guide", chromedp.ByQuery),
		chromedp.Evaluate(`(async () => {
			const deadline = Date.now() + 5000;
			while (Date.now() < deadline) {
				const result = document.querySelector('#margo-search-feature-guide');
				if (result && !result.hidden && result.getClientRects().length > 0) return true;
				await new Promise((resolve) => setTimeout(resolve, 25));
			}
			throw new Error('search result did not become visible');
		})()`, nil),
		chromedp.Click(`#margo-search-feature-guide`, chromedp.ByQuery),
		chromedp.Evaluate(`(async () => {
			const deadline = Date.now() + 5000;
			while (Date.now() < deadline) {
				const heading = document.querySelector('#main-content h1');
				if (heading && heading.textContent.trim() === 'Guide') return true;
				await new Promise((resolve) => setTimeout(resolve, 50));
			}
			throw new Error('search navigation did not render Guide');
		})()`, nil),
		chromedp.WaitVisible(`[data-search-field] button`, chromedp.ByQuery),
	); err != nil {
		t.Fatal(err)
	}
	if modalState.InsideSidebar || modalState.Width < modalState.ViewportWidth-2 || modalState.Height < modalState.ViewportHeight-2 {
		t.Fatalf("search modal is not full viewport: %+v", modalState)
	}

	var state struct {
		Path       string `json:"path"`
		URL        string `json:"url"`
		Heading    string `json:"heading"`
		Navigation string `json:"navigation"`
		Shortcut   string `json:"shortcut"`
	}
	if err := chromedp.Run(ctx, chromedp.Evaluate(`(() => ({
		path: window.location.pathname,
		url: window.location.href,
		heading: document.querySelector('#main-content h1')?.textContent.trim() || '',
		navigation: performance.getEntriesByType('navigation')[0]?.type || '',
		shortcut: document.querySelector('[data-search-field] kbd')?.textContent.trim() || '',
	}))()`, &state)); err != nil {
		t.Fatal(err)
	}
	if state.Path != "/guide.html" || state.Navigation != "navigate" || state.Shortcut != "⌘ K" {
		t.Fatalf("search state = %+v, want guide navigation with visible shortcut", state)
	}
}

func siteTestChromiumAllocatorOptions(browserPath string) []chromedp.ExecAllocatorOption {
	options := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	options = append(options, chromedp.ExecPath(browserPath))
	if runtime.GOOS == "linux" {
		options = append(options, chromedp.NoSandbox, chromedp.DisableGPU)
	}
	return options
}

func installedSiteTestChromium() string {
	candidates := []string{
		"/opt/homebrew/bin/chromium",
		"/usr/local/bin/chromium",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable"} {
		if candidate, err := exec.LookPath(name); err == nil {
			return candidate
		}
	}
	return ""
}

func siteBrowserContentType(path string) string {
	switch filepath.Ext(path) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".json":
		return "application/json; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
