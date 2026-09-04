package margo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/chromedp/chromedp"
)

func TestStandaloneCodeCopyReportsUnavailableClipboardWithoutAlpine(t *testing.T) {
	browserPath := installedPrintTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	result := mustRenderSource(t, "# Standalone\n\n```\necho hello\n```\n")
	component, err := RenderStandalone(result)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, component)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = writer.Write([]byte(markup))
	}))
	defer server.Close()

	allocatorOptions := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	allocatorOptions = append(allocatorOptions, chromedp.ExecPath(browserPath))
	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), allocatorOptions...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 20*time.Second)
	defer cancel()

	var state struct {
		Label            string `json:"label"`
		InitialAriaLabel string `json:"initialAriaLabel"`
		AriaLabel        string `json:"ariaLabel"`
		RuntimeSeen      bool   `json:"runtimeSeen"`
		AlpineAttrs      bool   `json:"alpineAttrs"`
	}
	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL),
		chromedp.WaitVisible(`[data-code-block-copy]`, chromedp.ByQuery),
		chromedp.Evaluate(`(() => {
			globalThis.margoInitialCodeCopyAriaLabel = document.querySelector('[data-code-block-copy]')?.getAttribute('aria-label') || '';
			Object.defineProperty(navigator, 'clipboard', { configurable: true, value: undefined });
			return true;
		})()`, nil),
		chromedp.Click(`[data-code-block-copy]`, chromedp.ByQuery),
		chromedp.Poll(`document.querySelector('[data-code-block-copy-status]')?.textContent.trim() === 'Unable to copy'`, nil, chromedp.WithPollingInterval(20*time.Millisecond)),
		chromedp.Evaluate(`(() => ({
			label: document.querySelector('[data-code-block-copy-status]')?.textContent.trim() || '',
			initialAriaLabel: globalThis.margoInitialCodeCopyAriaLabel || '',
			ariaLabel: document.querySelector('[data-code-block-copy]')?.getAttribute('aria-label') || '',
			runtimeSeen: [...document.querySelectorAll('script[data-margo-requirement="goshtoso.runtime.code-block"]')].some((script) => !script.src && script.textContent.includes('data-code-block-copy')),
			alpineAttrs: !!document.querySelector('[data-code-block][x-data]') || [...document.querySelectorAll('[data-code-block-copy]')].some((button) => button.hasAttribute('@click'))
		}))()`, &state),
	); err != nil {
		t.Fatal(err)
	}
	if state.Label != "Unable to copy" || state.AriaLabel == "" || state.AriaLabel != state.InitialAriaLabel || !state.RuntimeSeen || state.AlpineAttrs {
		t.Fatalf("standalone unavailable code-copy state = %+v", state)
	}
}
