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

func TestStandaloneCodeCopyUsesFallbackWithoutClipboardOrAlpine(t *testing.T) {
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
		Copied      string `json:"copied"`
		Label       string `json:"label"`
		AriaLabel   string `json:"ariaLabel"`
		RuntimeSeen bool   `json:"runtimeSeen"`
		AlpineAttrs bool   `json:"alpineAttrs"`
	}
	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL),
		chromedp.WaitVisible(`[data-margo-code-copy-button]`, chromedp.ByQuery),
		chromedp.Evaluate(`(() => {
			Object.defineProperty(navigator, 'clipboard', { configurable: true, value: undefined });
			document.execCommand = (command) => {
				if (command !== 'copy') return false;
				globalThis.margoFallbackCode = document.querySelector('textarea')?.value || '';
				return true;
			};
			return true;
		})()`, nil),
		chromedp.Click(`[data-margo-code-copy-button]`, chromedp.ByQuery),
		chromedp.Poll(`document.querySelector('[data-margo-code-copy-label]')?.textContent.trim() === 'Copied!'`, nil, chromedp.WithPollingInterval(20*time.Millisecond)),
		chromedp.Evaluate(`(() => ({
			copied: globalThis.margoFallbackCode || '',
			label: document.querySelector('[data-margo-code-copy-label]')?.textContent.trim() || '',
			ariaLabel: document.querySelector('[data-margo-code-copy-button]')?.getAttribute('aria-label') || '',
			runtimeSeen: [...document.querySelectorAll('script[data-margo-requirement="margo.code-copy"]')].some((script) => !script.src && script.textContent.includes('fallbackCopy')),
			alpineAttrs: !!document.querySelector('[data-margo-code-copy][x-data]') || [...document.querySelectorAll('[data-margo-code-copy-button]')].some((button) => button.hasAttribute('@click'))
		}))()`, &state),
	); err != nil {
		t.Fatal(err)
	}
	if state.Copied != "echo hello\n" || state.Label != "Copied!" || state.AriaLabel != "Code copied" || !state.RuntimeSeen || state.AlpineAttrs {
		t.Fatalf("standalone fallback code-copy state = %+v", state)
	}
}
