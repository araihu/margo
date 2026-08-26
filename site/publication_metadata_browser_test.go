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

func TestBuildLocalSitePublicationDatesRemainReadableWithoutOverflow(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	result, err := Build(context.Background(), Request{
		Sources:  []Source{{Path: "post.md", Content: []byte("---\ntitle: Release notes\nlanguage: en\npublishedAt: \"2026-08-25T12:00:00Z\"\nmodifiedAt: \"2026-08-26T12:00:00Z\"\n---\n# Release notes\n\nA publication date layout check.\n")}},
		Compiler: margo.New(), Assets: AssetsInline,
	})
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
			artifactPath = "/post.html"
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

	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(browserPath)...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 30*time.Second)
	defer cancel()

	type state struct {
		Overflow       bool    `json:"overflow"`
		DatesWidth     float64 `json:"datesWidth"`
		ViewportWidth  float64 `json:"viewportWidth"`
		PublishedLabel string  `json:"publishedLabel"`
		UpdatedLabel   string  `json:"updatedLabel"`
		Separator      bool    `json:"separator"`
	}
	readState := `(() => {
		const dates = document.querySelector('.margo-document__publication-dates');
		const published = document.querySelector('[data-margo-publication-label="published"]');
		const updated = document.querySelector('[data-margo-publication-label="modified"]');
		const separator = document.querySelector('[data-margo-publication-separator="true"]');
		return {
			overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1 || document.body.scrollWidth > document.documentElement.clientWidth + 1,
			datesWidth: dates?.getBoundingClientRect().width || 0,
			viewportWidth: window.innerWidth,
			publishedLabel: published?.textContent.trim() || '',
			updatedLabel: updated?.textContent.trim() || '',
			separator: !!separator && getComputedStyle(separator).display !== 'none' && separator.getClientRects().length > 0,
		};
	})()`
	for _, width := range []int64{390, 1280} {
		var got state
		if err := chromedp.Run(ctx,
			chromedp.EmulateViewport(width, 900),
			chromedp.Navigate(server.URL+"/post.html"),
			chromedp.WaitVisible(`[data-margo-publication-date="published"]`, chromedp.ByQuery),
			chromedp.Evaluate(readState, &got),
		); err != nil {
			t.Fatalf("%dpx publication metadata browser check: %v", width, err)
		}
		if got.Overflow || got.DatesWidth > got.ViewportWidth+1 || got.PublishedLabel != "Published" || got.UpdatedLabel != "Updated" || !got.Separator {
			t.Fatalf("%dpx publication metadata state = %+v", width, got)
		}
	}
}
