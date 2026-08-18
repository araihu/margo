//go:build editorial_e2e

package e2e_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	blogsite "github.com/araihu/margo/examples/blog/site"
	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

type decodedBlogImage struct {
	Path   string  `json:"path"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func TestGeneratedBlogPagesDecodePopularImageFormats(t *testing.T) {
	output := t.TempDir()
	if err := blogsite.Generate(output); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.FileServer(http.Dir(output)))
	defer server.Close()

	browserPath := requireInstalledChromium(t)
	t.Logf("tested browser: %s on %s/%s", chromiumVersion(t, browserPath), runtime.GOOS, runtime.GOARCH)
	allocatorOptions := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath(browserPath))
	allocatorContext, allocatorCancel := browserlaunch.NewExecAllocator(context.Background(), allocatorOptions...)
	defer allocatorCancel()

	ctx, cancel, evidence := newJourneyContext(t, allocatorContext)
	defer cancel()
	var selectedHero string
	var decoded []decodedBlogImage
	if err := chromedp.Run(ctx,
		network.Enable(), cdpruntime.Enable(),
		chromedp.Navigate(server.URL+"/field-notes.html"),
		chromedp.WaitVisible(`article.margo-document`, chromedp.ByQuery),
		chromedp.WaitVisible(`.blog-hero img`, chromedp.ByQuery),
		chromedp.Evaluate(`document.querySelector('.blog-hero img').currentSrc`, &selectedHero),
		chromedp.Evaluate(`Promise.all([
			'atelier-hero.avif', 'atelier-hero.webp', 'atelier-hero.jpg',
			'format-study.png', 'format-study.gif'
		].map(path => new Promise((resolve, reject) => {
			const image = new Image();
			image.onload = () => resolve({path, width: image.naturalWidth, height: image.naturalHeight});
			image.onerror = () => reject(new Error('decode failed: ' + path));
			image.src = '/assets/' + path + '?decode=' + path;
		})))`, &decoded, func(params *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams {
			return params.WithAwaitPromise(true)
		}),
	); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 5 {
		t.Fatalf("decoded images = %#v", decoded)
	}
	for _, image := range decoded {
		if image.Width <= 0 || image.Height <= 0 {
			t.Fatalf("image did not decode: %#v", image)
		}
	}
	if !strings.HasSuffix(selectedHero, ".avif") && !strings.HasSuffix(selectedHero, ".webp") && !strings.HasSuffix(selectedHero, ".jpg") {
		t.Fatalf("unexpected picture source %q", selectedHero)
	}
	assertCleanEvidence(t, server.URL, evidence)
}

func TestGeneratedBlogPagesHideScreenFooterWhenPrinting(t *testing.T) {
	output := t.TempDir()
	if err := blogsite.Generate(output); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.FileServer(http.Dir(output)))
	defer server.Close()

	browserPath := requireInstalledChromium(t)
	allocatorOptions := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath(browserPath))
	allocatorContext, allocatorCancel := browserlaunch.NewExecAllocator(context.Background(), allocatorOptions...)
	defer allocatorCancel()
	ctx, cancel := chromedp.NewContext(allocatorContext)
	defer cancel()

	var display string
	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL+"/index.html"),
		chromedp.WaitVisible(`.margo-page-footer`, chromedp.ByQuery),
		emulation.SetEmulatedMedia().WithMedia("print"),
		chromedp.Evaluate(`getComputedStyle(document.querySelector('.margo-page-footer')).display`, &display),
	); err != nil {
		t.Fatal(err)
	}
	if display != "none" {
		t.Fatalf("print footer display = %q, want none to avoid a footer-only page", display)
	}
}
