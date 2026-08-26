package site

import (
	"context"
	"net/http"
	"net/http/httptest"
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
	if _, ok := artifacts["/margo-assets/code-copy.js"]; !ok {
		t.Fatal("local site did not publish the Margo code-copy runtime")
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
		Copied      string `json:"copied"`
		Label       string `json:"label"`
		AriaLabel   string `json:"ariaLabel"`
		RuntimeSeen bool   `json:"runtimeSeen"`
		AlpineAttrs bool   `json:"alpineAttrs"`
	}
	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL+"/guide.html"),
		chromedp.WaitVisible(`[data-margo-code-copy-button]`, chromedp.ByQuery),
		chromedp.Evaluate(`(() => {
			const clipboard = navigator.clipboard || {};
			clipboard.writeText = (text) => { globalThis.margoCopiedCode = text; return Promise.resolve(); };
			try { Object.defineProperty(navigator, 'clipboard', { configurable: true, value: clipboard }); } catch (_) {}
			return typeof navigator.clipboard?.writeText === 'function';
		})()`, nil),
		chromedp.Click(`[data-margo-code-copy-button]`, chromedp.ByQuery),
		chromedp.Poll(`document.querySelector('[data-margo-code-copy-label]')?.textContent.trim() === 'Copied!'`, nil, chromedp.WithPollingInterval(20*time.Millisecond)),
		chromedp.Evaluate(`(() => ({
			copied: globalThis.margoCopiedCode || '',
			label: document.querySelector('[data-margo-code-copy-label]')?.textContent.trim() || '',
			ariaLabel: document.querySelector('[data-margo-code-copy-button]')?.getAttribute('aria-label') || '',
			runtimeSeen: [...document.scripts].some((script) => script.src.endsWith('/margo-assets/code-copy.js')),
			alpineAttrs: !!document.querySelector('[data-margo-code-copy][x-data]') || [...document.querySelectorAll('[data-margo-code-copy-button]')].some((button) => button.hasAttribute('@click'))
		}))()`, &state),
	); err != nil {
		t.Fatal(err)
	}
	if state.Copied != "echo hello\n" || state.Label != "Copied!" || state.AriaLabel != "Code copied" || !state.RuntimeSeen || state.AlpineAttrs {
		t.Fatalf("code-copy browser state = %+v", state)
	}
}
