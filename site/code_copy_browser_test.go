package site

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/chromedp/chromedp"
)

func TestBuildLocalSiteCodeCopyWorksWithoutAlpine(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	result, err := Build(context.Background(), Request{
		Sources:  []Source{{Path: "guide.md", Content: []byte("# Guide\n\n```sh\necho hello\n```\n")}},
		Compiler: margo.New(), Assets: AssetsLocal,
	})
	if err != nil {
		t.Fatal(err)
	}
	artifacts := make(map[string][]byte, len(result.Artifacts))
	for _, artifact := range result.Artifacts {
		artifacts["/"+strings.TrimPrefix(artifact.Path, "/")] = artifact.Content
	}
	if _, ok := artifacts["/assets/js/code-block.js"]; !ok {
		t.Fatal("local site did not publish the Goshtoso code-block runtime")
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		path := request.URL.Path
		if path == "/" {
			path = "/guide.html"
		}
		content, ok := artifacts[path]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", siteBrowserContentType(path))
		_, _ = writer.Write(content)
	}))
	defer server.Close()

	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(browserPath)...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 15*time.Second)
	defer cancel()

	var state struct {
		Copied           string `json:"copied"`
		Label            string `json:"label"`
		InitialAriaLabel string `json:"initialAriaLabel"`
		AriaLabel        string `json:"ariaLabel"`
		RuntimeSeen      bool   `json:"runtimeSeen"`
		AlpineAttrs      bool   `json:"alpineAttrs"`
	}
	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL+"/guide.html"),
		chromedp.WaitVisible(`[data-code-block-copy]`, chromedp.ByQuery),
		chromedp.Evaluate(`(() => {
			globalThis.margoInitialCodeCopyAriaLabel = document.querySelector('[data-code-block-copy]')?.getAttribute('aria-label') || '';
			const clipboard = navigator.clipboard || {};
			clipboard.writeText = (text) => { globalThis.margoCopiedCode = text; return Promise.resolve(); };
			try { Object.defineProperty(navigator, 'clipboard', { configurable: true, value: clipboard }); } catch (_) {}
			return typeof navigator.clipboard?.writeText === 'function';
		})()`, nil),
		chromedp.Click(`[data-code-block-copy]`, chromedp.ByQuery),
		chromedp.Poll(`document.querySelector('[data-code-block-copy-status]')?.textContent.trim() === 'Copied!'`, nil, chromedp.WithPollingInterval(20*time.Millisecond)),
		chromedp.Evaluate(`(() => ({
			copied: globalThis.margoCopiedCode || '',
			label: document.querySelector('[data-code-block-copy-status]')?.textContent.trim() || '',
			initialAriaLabel: globalThis.margoInitialCodeCopyAriaLabel || '',
			ariaLabel: document.querySelector('[data-code-block-copy]')?.getAttribute('aria-label') || '',
			runtimeSeen: [...document.scripts].some((script) => script.src.endsWith('/assets/js/code-block.js')),
			alpineAttrs: !!document.querySelector('[data-code-block][x-data]') || [...document.querySelectorAll('[data-code-block-copy]')].some((button) => button.hasAttribute('@click'))
		}))()`, &state),
	); err != nil {
		t.Fatal(err)
	}
	if state.Copied != "echo hello\n" || state.Label != "Copied!" || state.AriaLabel == "" || state.AriaLabel != state.InitialAriaLabel || !state.RuntimeSeen || state.AlpineAttrs {
		t.Fatalf("code-copy browser state = %+v", state)
	}
}

func TestBuildConfiguredSiteCodeCopyWorksWithoutAlpine(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "---\ntitle: Copy test\nlanguage: en\n---\n\n# Copy test\n\n```\necho hello\n```\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
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
	if _, ok := artifacts["/assets/js/code-block.js"]; !ok {
		t.Fatal("configured site did not publish the Goshtoso code-block runtime")
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		path := request.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		content, ok := artifacts[path]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", siteBrowserContentType(path))
		_, _ = writer.Write(content)
	}))
	defer server.Close()

	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(browserPath)...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 20*time.Second)
	defer cancel()

	var state struct {
		Copied           string `json:"copied"`
		Label            string `json:"label"`
		InitialAriaLabel string `json:"initialAriaLabel"`
		AriaLabel        string `json:"ariaLabel"`
		RuntimeSeen      bool   `json:"runtimeSeen"`
		AlpineAttrs      bool   `json:"alpineAttrs"`
	}
	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL+"/index.html"),
		chromedp.WaitVisible(`[data-code-block-copy]`, chromedp.ByQuery),
		chromedp.Evaluate(`(() => {
			globalThis.margoInitialCodeCopyAriaLabel = document.querySelector('[data-code-block-copy]')?.getAttribute('aria-label') || '';
			const clipboard = navigator.clipboard || {};
			clipboard.writeText = (text) => { globalThis.margoCopiedCode = text; return Promise.resolve(); };
			try { Object.defineProperty(navigator, 'clipboard', { configurable: true, value: clipboard }); } catch (_) {}
			return true;
		})()`, nil),
		chromedp.Click(`[data-code-block-copy]`, chromedp.ByQuery),
		chromedp.Poll(`document.querySelector('[data-code-block-copy-status]')?.textContent.trim() === 'Copied!'`, nil, chromedp.WithPollingInterval(20*time.Millisecond)),
		chromedp.Evaluate(`(() => ({
			copied: globalThis.margoCopiedCode || '',
			label: document.querySelector('[data-code-block-copy-status]')?.textContent.trim() || '',
			initialAriaLabel: globalThis.margoInitialCodeCopyAriaLabel || '',
			ariaLabel: document.querySelector('[data-code-block-copy]')?.getAttribute('aria-label') || '',
			runtimeSeen: [...document.scripts].some((script) => script.src.endsWith('/assets/js/code-block.js')),
			alpineAttrs: !!document.querySelector('[data-code-block][x-data]') || [...document.querySelectorAll('[data-code-block-copy]')].some((button) => button.hasAttribute('@click'))
		}))()`, &state),
	); err != nil {
		t.Fatal(err)
	}
	if state.Copied != "echo hello\n" || state.Label != "Copied!" || state.AriaLabel == "" || state.AriaLabel != state.InitialAriaLabel || !state.RuntimeSeen || state.AlpineAttrs {
		t.Fatalf("configured site code-copy browser state = %+v", state)
	}
}
