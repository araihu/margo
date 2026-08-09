//go:build editorial_e2e

package e2e_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/a-h/templ"
	chartassets "github.com/araihu/goshtoso-charts/assets"
	goshtosoassets "github.com/araihu/goshtoso/assets"
	margo "github.com/araihu/margo"
	charts "github.com/araihu/margo/charts"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

const themeCSS = `[data-theme="araihu"]{--color-surface:#f4f0ff;--color-on-surface:#241442;--color-primary:#6d28d9}`

type editorialFixture struct {
	server          *httptest.Server
	editorial       *margo.EditorialResult
	authority       margo.AuthorityRecord
	localPaths      []string
	publicationHTML string
}

func TestGeneratedEditorialHTMLJourneys(t *testing.T) {
	fixture := newEditorialFixture(t)
	defer fixture.server.Close()

	browserPath := requireInstalledChromium(t)
	t.Logf("tested browser: %s on %s/%s", chromiumVersion(t, browserPath), runtime.GOOS, runtime.GOARCH)
	assertInitialPublicationMetadata(t, fixture)

	allocatorOptions := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath(browserPath))
	allocatorContext, allocatorCancel := chromedp.NewExecAllocator(context.Background(), allocatorOptions...)
	defer allocatorCancel()

	runManjaFragmentJourney(t, allocatorContext, fixture)
	runPublicationJourney(t, allocatorContext, fixture, "/guide")
	runInlinePublicationJourney(t, allocatorContext, fixture, "/guide-inline")
	runJavaScriptDisabledJourney(t, browserPath, fixture)
}

func newEditorialFixture(t *testing.T) editorialFixture {
	t.Helper()
	chartMarkdown, err := os.ReadFile(filepath.Join("..", "testdata", "markdown", "editorial-charts.md"))
	if err != nil {
		t.Fatal(err)
	}
	source := `---
title: Durable HTML
description: One semantic source for documentation and publishing.
language: en
slug: durable-html
authors: [Arai Hû]
publishedAt: "2026-08-09T12:00:00-03:00"
modifiedAt: "2026-08-09T15:00:00Z"
tags: [Go, HTML]
---

# Durable HTML

Margo HTML remains readable in every projection.

| Name | Count |
|---|---:|
| Item 10 | 10 |
| Item 2 | 2 |

` + string(chartMarkdown)
	compiler := margo.New(
		margo.WithHostPolicy(margo.Policy{RawHTML: margo.RawHTMLSanitized, OutputBytes: margo.MaxOutputBytes}),
		margo.WithExtension(charts.Extension(charts.WithExternalizedControlRuntime(true))),
	)
	document, err := compiler.Compile(context.Background(), margo.Source{Name: "editorial-e2e.md", Content: []byte(source)})
	if err != nil {
		t.Fatalf("compile fixture: %v", err)
	}
	rendered, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatalf("render fixture: %v", err)
	}
	editorial, err := margo.RenderEditorial(rendered)
	if err != nil {
		t.Fatalf("editorial fixture: %v", err)
	}

	authorityBytes, err := os.ReadFile(filepath.Join("..", "..", "testdata", "authority", "record.json"))
	if err != nil {
		t.Fatal(err)
	}
	authority, err := margo.VerifyAuthorityRecord(authorityBytes)
	if err != nil {
		t.Fatal(err)
	}
	theme := materializedThemeAsset()
	publicInput := margo.PublicationInput{
		Mode: margo.PublicationPublic, Kind: margo.PublicationArticle,
		Authority: authority, RoutePath: authority.Routes.Representative,
		SiteName: "Arai Hû", Locale: "en_US",
		Image: margo.SocialImage{
			URL:      string(authority.CanonicalOrigin) + authority.Routes.Preview,
			MIMEType: authority.Asset.MIMEType, Width: authority.Asset.Width, Height: authority.Asset.Height,
			Alt: "Editorial preview fixture.",
		},
		Theme: margo.ThemeName("araihu"), ColorMode: margo.ColorModeLight,
		DependencyMode: margo.HTMLDependenciesLocal, ThemeStylesheet: theme,
	}
	publicPage, err := margo.RenderPublication(editorial, publicInput)
	if err != nil {
		t.Fatalf("public publication: %v", err)
	}
	publicHTML := renderComponent(t, publicPage)

	inlineInput := publicInput
	inlineInput.Mode = margo.PublicationPrivate
	inlineInput.Authority = margo.AuthorityRecord{}
	inlineInput.RoutePath = ""
	inlineInput.SiteName = ""
	inlineInput.Locale = ""
	inlineInput.Image = margo.SocialImage{}
	inlineInput.DependencyMode = margo.HTMLDependenciesInline
	inlinePage, err := margo.RenderPublication(editorial, inlineInput)
	if err != nil {
		t.Fatalf("inline publication: %v", err)
	}
	inlineHTML := renderComponent(t, inlinePage)
	manjaHTML := renderManjaHost(t, editorial)

	mux := http.NewServeMux()
	mux.Handle("/assets/", goshtosoassets.Handler())
	mux.Handle("/margo-assets/", margo.EditorialAssetHandler())
	mux.Handle(chartassets.Prefix, chartassets.Handler())
	mux.HandleFunc("/theme-araihu.css", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = io.WriteString(writer, themeCSS)
	})
	mux.HandleFunc("/favicon.ico", func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("/manja", htmlHandler(manjaHTML))
	mux.HandleFunc("/guide", htmlHandler(publicHTML))
	mux.HandleFunc("/guide-inline", htmlHandler(inlineHTML))
	server := httptest.NewServer(mux)

	localPaths := make([]string, 0, len(editorial.Requirements().List())+1)
	for _, requirement := range editorial.Requirements().List() {
		localPaths = append(localPaths, requirement.LocalURL)
	}
	localPaths = append(localPaths, "/theme-araihu.css")
	return editorialFixture{
		server: server, editorial: editorial, authority: authority,
		localPaths: localPaths, publicationHTML: publicHTML,
	}
}

func materializedThemeAsset() margo.AssetRef {
	content := []byte(themeCSS)
	digest := sha256.Sum256(content)
	return margo.AssetRef{
		Path: "theme-araihu.css", MediaType: "text/css",
		SHA256: hex.EncodeToString(digest[:]), Content: content,
	}
}

func htmlHandler(markup string) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(writer, markup)
	}
}

func renderManjaHost(t *testing.T, editorial *margo.EditorialResult) string {
	t.Helper()
	var head strings.Builder
	for _, requirement := range editorial.Requirements().List() {
		id := html.EscapeString(requirement.ID)
		url := html.EscapeString(requirement.LocalURL)
		if requirement.Kind == margo.HTMLStylesheet {
			fmt.Fprintf(&head, `<link rel="stylesheet" href="%s" data-margo-requirement="%s">`, url, id)
		} else {
			fmt.Fprintf(&head, `<script defer src="%s" data-margo-requirement="%s"></script>`, url, id)
		}
	}
	fragment := renderComponent(t, editorial.Fragment())
	return `<!doctype html><html lang="en" data-theme="modern"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">` + head.String() + `</head><body><main class="manja-shell"><div class="manja-markdown">` + fragment + `</div></main></body></html>`
}

func renderComponent(t *testing.T, component templ.Component) string {
	t.Helper()
	var output bytes.Buffer
	if err := component.Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	return output.String()
}

func assertInitialPublicationMetadata(t *testing.T, fixture editorialFixture) {
	t.Helper()
	response, err := http.Get(fixture.server.URL + "/guide")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	markup := string(data)
	canonical := string(fixture.authority.CanonicalOrigin) + fixture.authority.Routes.Representative
	if err := margo.RequireOneCompleteSocialSet(markup, margo.SocialMetadata{CanonicalURL: canonical}); err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		`property="og:type" content="article"`,
		`property="article:published_time"`,
		`property="article:modified_time"`,
		`property="article:author"`,
		`property="article:tag"`,
		`name="twitter:card"`,
	} {
		if !strings.Contains(markup, marker) {
			t.Fatalf("initial HTML missing %q", marker)
		}
	}
}

type browserEvidence struct {
	mu         sync.Mutex
	requests   []string
	failures   []string
	exceptions []string
}

func (evidence *browserEvidence) listen(ctx context.Context) {
	chromedp.ListenTarget(ctx, func(event any) {
		evidence.mu.Lock()
		defer evidence.mu.Unlock()
		switch typed := event.(type) {
		case *network.EventRequestWillBeSent:
			evidence.requests = append(evidence.requests, typed.Request.URL)
		case *network.EventLoadingFailed:
			evidence.failures = append(evidence.failures, typed.ErrorText)
		case *cdpruntime.EventExceptionThrown:
			evidence.exceptions = append(evidence.exceptions, typed.ExceptionDetails.Text)
		case *cdpruntime.EventConsoleAPICalled:
			if typed.Type == cdpruntime.APITypeError || typed.Type == cdpruntime.APITypeAssert {
				evidence.exceptions = append(evidence.exceptions, string(typed.Type))
			}
		}
	})
}

func (evidence *browserEvidence) snapshot() (requests, failures, exceptions []string) {
	evidence.mu.Lock()
	defer evidence.mu.Unlock()
	return append([]string(nil), evidence.requests...), append([]string(nil), evidence.failures...), append([]string(nil), evidence.exceptions...)
}

func newJourneyContext(t *testing.T, allocatorContext context.Context) (context.Context, context.CancelFunc, *browserEvidence) {
	t.Helper()
	ctx, cancel := chromedp.NewContext(allocatorContext)
	ctx, timeoutCancel := context.WithTimeout(ctx, 30*time.Second)
	evidence := &browserEvidence{}
	evidence.listen(ctx)
	return ctx, func() { timeoutCancel(); cancel() }, evidence
}

func runManjaFragmentJourney(t *testing.T, allocatorContext context.Context, fixture editorialFixture) {
	t.Helper()
	ctx, cancel, evidence := newJourneyContext(t, allocatorContext)
	defer cancel()
	var before, after, identity string
	err := chromedp.Run(ctx,
		network.Enable(), cdpruntime.Enable(),
		chromedp.Navigate(fixture.server.URL+"/manja"),
		chromedp.WaitVisible(`.manja-markdown > article.margo-document`, chromedp.ByQuery),
		chromedp.Evaluate(`(() => { const article = document.querySelector('article.margo-document'); article.dataset.e2eIdentity = 'same-node'; return getComputedStyle(article).backgroundColor + '|' + getComputedStyle(article).color; })()`, &before),
		chromedp.Evaluate(`(() => { document.documentElement.dataset.theme = 'dracula'; document.documentElement.classList.add('dark'); const article = document.querySelector('article.margo-document'); return getComputedStyle(article).backgroundColor + '|' + getComputedStyle(article).color; })()`, &after),
		chromedp.AttributeValue(`article.margo-document`, "data-e2e-identity", &identity, nil, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatal(err)
	}
	if before == after || identity != "same-node" {
		t.Fatalf("host theme inheritance before=%q after=%q identity=%q", before, after, identity)
	}
	assertCleanEvidence(t, fixture.server.URL, evidence)
}

func runPublicationJourney(t *testing.T, allocatorContext context.Context, fixture editorialFixture, route string) {
	t.Helper()
	ctx, cancel, evidence := newJourneyContext(t, allocatorContext)
	defer cancel()
	if err := chromedp.Run(ctx,
		network.Enable(), cdpruntime.Enable(),
		chromedp.Navigate(fixture.server.URL+route),
		chromedp.WaitVisible(`article.margo-document`, chromedp.ByQuery),
		chromedp.WaitVisible(`.margo-table-sort-button`, chromedp.ByQuery),
	); err != nil {
		t.Fatal(err)
	}
	assertSortCycle(t, ctx)
	assertChartControls(t, ctx)
	assertSemanticChartCounts(t, ctx)
	assertNoDuplicateIDs(t, ctx)
	assertCleanEvidence(t, fixture.server.URL, evidence)
	requests, _, _ := evidence.snapshot()
	counts := requestPathCounts(requests)
	for _, path := range fixture.localPaths {
		if counts[path] != 1 {
			t.Fatalf("local requirement %s requests = %d; all=%v", path, counts[path], counts)
		}
	}
}

func runInlinePublicationJourney(t *testing.T, allocatorContext context.Context, fixture editorialFixture, route string) {
	t.Helper()
	ctx, cancel, evidence := newJourneyContext(t, allocatorContext)
	defer cancel()
	if err := chromedp.Run(ctx,
		network.Enable(), cdpruntime.Enable(),
		chromedp.Navigate(fixture.server.URL+route),
		chromedp.WaitVisible(`article.margo-document`, chromedp.ByQuery),
		chromedp.WaitVisible(`.margo-table-sort-button`, chromedp.ByQuery),
	); err != nil {
		t.Fatal(err)
	}
	assertSemanticChartCounts(t, ctx)
	assertNoDuplicateIDs(t, ctx)
	assertCleanEvidence(t, fixture.server.URL, evidence)
	requests, _, _ := evidence.snapshot()
	for _, requestURL := range requests {
		parsed, err := url.Parse(requestURL)
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(parsed.Path, "/assets/") || strings.HasPrefix(parsed.Path, "/margo-assets/") || strings.HasPrefix(parsed.Path, chartassets.Prefix) {
			t.Fatalf("inline publication requested runtime asset %s", requestURL)
		}
	}
}

func assertSortCycle(t *testing.T, ctx context.Context) {
	t.Helper()
	readState := func() string {
		var state string
		if err := chromedp.Run(ctx, chromedp.Evaluate(`(() => { const th = document.querySelector('[data-margo-table-sort] th'); const rows = [...document.querySelectorAll('[data-margo-table-sort] tbody tr')].map(row => row.cells[0].textContent.trim()).join(','); return (th.getAttribute('aria-sort') || 'source') + '|' + rows; })()`, &state)); err != nil {
			t.Fatal(err)
		}
		return state
	}
	want := []string{"ascending|Item 2,Item 10", "descending|Item 10,Item 2", "source|Item 10,Item 2"}
	for index, expected := range want {
		if err := chromedp.Run(ctx, chromedp.Click(`[data-margo-table-sort] th:first-child button`, chromedp.ByQuery)); err != nil {
			t.Fatal(err)
		}
		if got := readState(); got != expected {
			t.Fatalf("sort click %d = %q, want %q", index+1, got, expected)
		}
	}
}

func assertChartControls(t *testing.T, ctx context.Context) {
	t.Helper()
	var opened, closed bool
	if err := chromedp.Run(ctx,
		chromedp.Evaluate(`(() => { const button = [...document.querySelectorAll('[data-goshtoso-chart-expand] button')].find(node => node.textContent.trim() === 'Expand'); if (!button) return false; button.click(); return true; })()`, &opened),
		chromedp.Poll(`(() => { const dialog = document.querySelector('[data-goshtoso-chart-expand] [role="dialog"]'); return dialog && getComputedStyle(dialog).display !== 'none'; })()`, nil),
		chromedp.Evaluate(`(() => { const button = document.querySelector('[data-goshtoso-chart-expand] [role="dialog"] button[aria-label="close modal"]'); if (!button) return false; button.click(); return true; })()`, &closed),
		chromedp.Poll(`(() => { const dialog = document.querySelector('[data-goshtoso-chart-expand] [role="dialog"]'); return dialog && getComputedStyle(dialog).display === 'none'; })()`, nil),
	); err != nil {
		t.Fatal(err)
	}
	if !opened || !closed {
		t.Fatalf("chart modal opened=%v closed=%v", opened, closed)
	}
}

func assertSemanticChartCounts(t *testing.T, ctx context.Context) {
	t.Helper()
	var counts string
	if err := chromedp.Run(ctx, chromedp.Evaluate(`document.querySelectorAll('[data-goshtoso-chart-content] > figure[role="img"] svg').length + '|' + document.querySelectorAll('[data-margo-chart-data="v1"]').length`, &counts)); err != nil {
		t.Fatal(err)
	}
	if counts != "4|4" {
		t.Fatalf("chart projections = %s, want 4|4", counts)
	}
}

func assertNoDuplicateIDs(t *testing.T, ctx context.Context) {
	t.Helper()
	var duplicates []string
	if err := chromedp.Run(ctx, chromedp.Evaluate(`(() => { const seen = new Set(); const duplicates = new Set(); document.querySelectorAll('[id]').forEach(node => seen.has(node.id) ? duplicates.add(node.id) : seen.add(node.id)); return [...duplicates].sort(); })()`, &duplicates)); err != nil {
		t.Fatal(err)
	}
	if len(duplicates) != 0 {
		t.Fatalf("duplicate DOM IDs: %v", duplicates)
	}
}

func runJavaScriptDisabledJourney(t *testing.T, browserPath string, fixture editorialFixture) {
	t.Helper()
	options := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath(browserPath))
	allocatorContext, allocatorCancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer allocatorCancel()
	ctx, cancel := chromedp.NewContext(allocatorContext)
	defer cancel()
	ctx, timeoutCancel := context.WithTimeout(ctx, 30*time.Second)
	defer timeoutCancel()
	var result string
	if err := chromedp.Run(ctx,
		emulation.SetScriptExecutionDisabled(true),
		chromedp.Navigate(fixture.server.URL+"/guide"),
		chromedp.WaitVisible(`article.margo-document`, chromedp.ByQuery),
		chromedp.Evaluate(`(() => { const article = document.querySelector('article.margo-document'); const dialog = document.querySelector('[role="dialog"]'); return [article.textContent.includes('Margo HTML remains readable'), document.querySelectorAll('[data-margo-table-sort] tbody tr').length, document.querySelectorAll('[data-goshtoso-chart-content] > figure[role="img"] svg').length, document.querySelectorAll('[data-margo-chart-data="v1"]').length, document.querySelectorAll('.margo-table-sort-button').length, dialog ? getComputedStyle(dialog).display : 'missing'].join('|'); })()`, &result),
	); err != nil {
		t.Fatal(err)
	}
	if result != "true|2|4|4|0|none" {
		t.Fatalf("JavaScript-disabled fallback = %q", result)
	}
}

func assertCleanEvidence(t *testing.T, serverOrigin string, evidence *browserEvidence) {
	t.Helper()
	requests, failures, exceptions := evidence.snapshot()
	if len(failures) != 0 || len(exceptions) != 0 {
		t.Fatalf("browser failures=%v exceptions=%v", failures, exceptions)
	}
	for _, requestURL := range requests {
		if strings.HasPrefix(requestURL, serverOrigin) || strings.HasPrefix(requestURL, "data:") || strings.HasPrefix(requestURL, "blob:") {
			continue
		}
		t.Fatalf("external browser request: %s", requestURL)
	}
}

func requestPathCounts(requests []string) map[string]int {
	counts := make(map[string]int)
	for _, requestURL := range requests {
		parsed, err := url.Parse(requestURL)
		if err == nil {
			counts[parsed.Path]++
		}
	}
	return counts
}

func requireInstalledChromium(t *testing.T) string {
	t.Helper()
	if configured := os.Getenv("MARGO_CHROMIUM"); configured != "" {
		if info, err := os.Stat(configured); err == nil && !info.IsDir() {
			return configured
		}
		t.Fatalf("MARGO_CHROMIUM is not an executable file: %s", configured)
	}
	candidates := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate
		}
	}
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser", "brave-browser", "microsoft-edge"} {
		if candidate, err := exec.LookPath(name); err == nil {
			return candidate
		}
	}
	t.Skip("no installed Chromium-family browser; set MARGO_CHROMIUM")
	return ""
}

func chromiumVersion(t *testing.T, path string) string {
	t.Helper()
	output, err := exec.Command(path, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("browser version: %v: %s", err, output)
	}
	return strings.TrimSpace(string(output))
}
