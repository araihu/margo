package site

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/charts"
	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/chromedp/chromedp"
)

// Build the published showcase source and configuration, including its real
// landing shell, theme, Markdown, Mermaid, and chart. Only PDF downloads use
// the existing test engine; the HTML and its dependencies follow production.
func TestShowcaseHomepageComposition(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	configPath := filepath.Join("..", "showcase.yaml")
	for _, mode := range []AssetMode{AssetsLocal, AssetsInline} {
		t.Run(string(mode), func(t *testing.T) {
			config, err := LoadConfig(configPath)
			if err != nil {
				t.Fatal(err)
			}
			config.Assets = string(mode)
			result, err := buildConfigured(context.Background(), ConfigRequest{
				ConfigPath: configPath,
				Compiler:   margo.New(margo.WithUnsafeHTML(), margo.WithExtension(charts.Extension(charts.WithExternalizedControlRuntime(true)))),
				PDFEngine:  siteTestPDFEngine{},
			}, config)
			if err != nil {
				t.Fatal(err)
			}
			artifacts := make(map[string][]byte)
			for _, artifact := range result.Artifacts {
				artifacts["/"+artifact.Path] = artifact.Content
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path
				if strings.HasSuffix(path, "/") {
					path += "index.html"
				}
				data, ok := artifacts[path]
				if !ok {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", siteBrowserContentType(path))
				_, _ = w.Write(data)
			}))
			defer server.Close()
			allocator, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(browserPath)...)
			defer cancelAllocator()
			browser, cancelBrowser := chromedp.NewContext(allocator)
			defer cancelBrowser()
			ctx, cancel := context.WithTimeout(browser, 90*time.Second)
			defer cancel()
			for _, width := range []int64{390, 984, 1440} {
				for _, dark := range []bool{false, true} {
					t.Run(fmt.Sprintf("%d/dark=%t", width, dark), func(t *testing.T) {
						var failures []string
						err := chromedp.Run(ctx, chromedp.EmulateViewport(width, 1109), chromedp.Navigate(server.URL+"/"),
							chromedp.Poll(`window.Alpine && document.querySelector('.margo-mermaid__viewer') && document.querySelector('[data-margo-runtime-status="succeeded"] svg')`, nil),
							chromedp.Evaluate(fmt.Sprintf(`document.documentElement.classList.toggle('dark', %t)`, dark), nil),
							chromedp.Evaluate(homepageCompositionChecks, &failures))
						if err != nil {
							t.Fatal(err)
						}
						if len(failures) > 0 {
							t.Fatalf("homepage composition: %v", failures)
						}
						// Long commands must remain complete and wrap within the real paragraph.
						var wraps bool
						if err := chromedp.Run(ctx, chromedp.Evaluate(`(() => { const code = [...document.querySelectorAll('p > code.margo-inline-code')].find(n=>n.textContent.includes('margo serve')); code.textContent = 'margo serve ' + 'long-path-segment/'.repeat(80) + ' --open'; return code.getClientRects().length > 1 && code.textContent.endsWith(' --open') && document.documentElement.scrollWidth <= innerWidth + 1; })()`, &wraps)); err != nil {
							t.Fatal(err)
						}
						if !wraps {
							t.Fatal("inline command was clipped or overflowed the homepage")
						}
					})
				}
			}
			// A host or a later HTMX requirement can load base CSS after document
			// CSS. The declared layer contract must protect content in that order too.
			var failures []string
			err = chromedp.Run(ctx, chromedp.Navigate(server.URL+"/"), chromedp.Poll(`document.querySelector('.margo-mermaid__viewer')`, nil),
				chromedp.Evaluate(`(() => { window.__margoTestLateBase = true; const link = [...document.querySelectorAll('link[rel="stylesheet"]')].find(n=>new URL(n.href).pathname === '/assets/styles.css'); document.head.appendChild(link); })()`, nil),
				chromedp.Poll(`Array.from(document.styleSheets).some(s=>s.href && new URL(s.href).pathname==='/assets/styles.css')`, nil),
				chromedp.Evaluate(homepageCompositionChecks, &failures))
			if err != nil {
				t.Fatal(err)
			}
			if len(failures) > 0 {
				t.Fatalf("late base stylesheet: %v", failures)
			}
		})
	}
}

const homepageCompositionChecks = `(() => {
 const failures = [];
 const check = (ok, message) => {if (!ok) failures.push(message)};
 const diagram = document.querySelector('figure.margo-mermaid');
 const code = [...document.querySelectorAll('p > code.margo-inline-code')].find(n=>n.textContent.includes('margo serve'));
 const figureStyle = getComputedStyle(diagram), codeStyle = getComputedStyle(code);
 check(parseFloat(figureStyle.paddingLeft) === 20 && parseFloat(figureStyle.paddingRight) === 20, 'diagram lost 20px padding');
 check(parseFloat(figureStyle.borderLeftWidth) === 1, 'diagram lost its border');
 check(parseFloat(codeStyle.paddingLeft) > 0 && parseFloat(codeStyle.borderLeftWidth) === 1, 'inline code lost padding/border');
 check(codeStyle.display === 'inline' && codeStyle.verticalAlign === 'baseline', 'inline code is not on the text baseline');
 check(codeStyle.lineHeight === getComputedStyle(code.parentElement).lineHeight, 'inline code changed the paragraph line-height');
 check(codeStyle.whiteSpace !== 'nowrap' && codeStyle.textOverflow !== 'ellipsis', 'inline code truncates commands');
 const table = document.querySelector('[data-margo-chart-data] table');
 check([...table.querySelectorAll('thead th')].map(n=>n.textContent).join('|') === 'Category|Value', 'chart has empty dimension columns');
 check([...table.querySelectorAll('tbody tr')].map(tr=>[...tr.querySelectorAll('td:not([hidden])')].map(n=>n.textContent).join('|')).join(',') === 'HTML|40,Site|30,PDF|20,Deck|10', 'chart data changed');
 check(document.documentElement.scrollWidth <= innerWidth+1, 'homepage overflows viewport');
 check([...document.querySelectorAll('link[rel="stylesheet"]')].filter(n=>new URL(n.href).pathname === '/assets/styles.css').length === 1, 'base stylesheet duplicated');
 if (!window.__margoTestLateBase) {
   const base = [...document.querySelectorAll('link[rel="stylesheet"]')].find(n=>new URL(n.href).pathname === '/assets/styles.css');
   const content = document.querySelector('[data-margo-requirement="margo.document.styles"]');
   check(!!(base.compareDocumentPosition(content) & Node.DOCUMENT_POSITION_FOLLOWING), 'shell deduplication broke the stylesheet dependency order');
 }
 return failures;
})()`
