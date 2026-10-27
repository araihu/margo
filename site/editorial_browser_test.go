package site

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

func TestEditorialBrowserHeaderTOCAndTheme(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	for _, mode := range []AssetMode{AssetsLocal, AssetsInline} {
		t.Run(string(mode), func(t *testing.T) {
			result := editorialFixture(t, mode)
			root := t.TempDir()
			for _, artifact := range result.Artifacts {
				name := filepath.Join(root, filepath.FromSlash(artifact.Path))
				if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(name, artifact.Content, 0644); err != nil {
					t.Fatal(err)
				}
			}
			server := httptest.NewServer(http.FileServer(http.Dir(root)))
			defer server.Close()
			allocator, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(browserPath)...)
			defer cancelAllocator()
			browser, cancelBrowser := chromedp.NewContext(allocator)
			defer cancelBrowser()
			ctx, cancel := context.WithTimeout(browser, 60*time.Second)
			defer cancel()
			const geometry = `(() => {
 const h=document.querySelector('.component-doc-shell__header').getBoundingClientRect();
 const toc=document.querySelector('.component-doc-shell__toc-inner').getBoundingClientRect();
 return Math.abs(h.bottom-toc.top)<2 && document.documentElement.scrollWidth<=innerWidth && document.querySelectorAll('.component-doc-shell__header').length===1;
 })()`
			if err := chromedp.Run(ctx, chromedp.EmulateViewport(1440, 900), chromedp.Navigate(server.URL+"/"), chromedp.WaitVisible(`.component-doc-shell__toc-inner`, chromedp.ByQuery), chromedp.Poll(geometry, nil)); err != nil {
				t.Fatal(err)
			}
			// The component owns both the offset and scrolling, including hash navigation.
			if err := chromedp.Run(ctx, chromedp.Click(`[data-toc-link="second-section"]`, chromedp.ByQuery), chromedp.Poll(`location.hash==='#second-section' && document.querySelector('[data-toc-link="second-section"]').classList.contains('is-active')`, nil), chromedp.Poll(geometry, nil)); err != nil {
				t.Fatal(err)
			}
			var visible bool
			if err := chromedp.Run(ctx, chromedp.Evaluate(`document.querySelector('#second-section').getBoundingClientRect().top >= document.querySelector('.component-doc-shell__header').getBoundingClientRect().bottom`, &visible)); err != nil || !visible {
				t.Fatalf("anchor hidden below header: %v %v", visible, err)
			}
			for _, width := range []int64{1280, 1440} {
				if err := chromedp.Run(ctx, chromedp.EmulateViewport(width, 900), chromedp.Poll(geometry, nil)); err != nil {
					t.Fatal(err)
				}
			}
			// Use the real theme control, then inspect active logo and document-link color.
			if err := chromedp.Run(ctx, chromedp.Click(`button[x-on\:click="toggleDark()"]`, chromedp.ByQuery), chromedp.Poll(`document.documentElement.classList.contains('dark')`, nil), chromedp.Evaluate(`(() => {
 const logo=document.querySelector('.margo-theme-logo__dark');
 const link=document.querySelector('.margo-document a');
 const light=document.querySelector('.margo-theme-logo__light');
 const token=getComputedStyle(document.documentElement).getPropertyValue('--color-primary-dark').trim();
 const probe=document.createElement('span');probe.style.color=token;document.body.appendChild(probe);
 const matches=getComputedStyle(link).color===getComputedStyle(probe).color;probe.remove();
 return matches && getComputedStyle(logo).display!=='none' && getComputedStyle(light).display==='none';
 })()`, &visible)); err != nil || !visible {
				t.Fatalf("dark logo/link mismatch: %v %v", visible, err)
			}
			if err := chromedp.Run(ctx, chromedp.EmulateViewport(390, 844), chromedp.Poll(`document.documentElement.scrollWidth<=innerWidth && getComputedStyle(document.querySelector('[data-componentdocshell-toc]')).display==='none'`, nil), chromedp.Click(`#margo-header-menu button`, chromedp.ByQuery), chromedp.WaitVisible(`#margo-header-menu a`, chromedp.ByQuery), chromedp.KeyEvent(kb.Escape), chromedp.Click(`#margo-site-search button`, chromedp.ByQuery), chromedp.WaitVisible(`input[placeholder="Find a post"]`, chromedp.ByQuery)); err != nil {
				t.Fatal(err)
			}
		})
	}
}
