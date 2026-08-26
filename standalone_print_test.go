package margo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func TestPrintPreparationChangesOnlyStaticContentStructure(t *testing.T) {
	browserPath := installedPrintTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	document := `<!doctype html><html><body><div class="goshtoso-document"><article class="margo-document">
<p><strong id="strong">Strong <a id="nested" href="https://example.com">link</a></strong> and <em>emphasis</em>.</p>
<div data-table-client-sort="true"><table><thead><tr><th class="margo-table-sort-button" tabindex="0">Column</th></tr></thead><tbody><tr><td>Value</td></tr></tbody></table></div>
<p><input type="checkbox" disabled checked> Complete</p>
<figure id="diagram" class="margo-mermaid"><div class="margo-mermaid__canvas"></div><details class="margo-mermaid__source"><summary>Source</summary></details></figure>
<div data-goshtoso-chart-wrapper data-goshtoso-chart-capability="interactive-raster" data-goshtoso-chart-export-pixel-ratio="1">
<div data-goshtoso-chart-content><figure class="goshtoso-charts-interactive" aria-label="Interactive revenue"><canvas></canvas></figure></div>
</div>
</article></div>
<script>document.addEventListener("goshtoso-charts:export-request", (event) => {
  event.detail.dataURL = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAFgQIAJyt0rQAAAABJRU5ErkJggg==";
});</script>` + standalonePrintPreparationScript + `</body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = writer.Write([]byte(document))
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

	var prepared struct {
		Buttons           int    `json:"buttons"`
		Inputs            int    `json:"inputs"`
		StaticLabels      int    `json:"staticLabels"`
		TableHeaders      int    `json:"tableHeaders"`
		NestedLinks       int    `json:"nestedLinks"`
		DetailsOpen       bool   `json:"detailsOpen"`
		PaginationMarkers int    `json:"paginationMarkers"`
		Scale             string `json:"scale"`
		PrintChartImages  int    `json:"printChartImages"`
		InteractiveCharts int    `json:"interactiveCharts"`
		ChartAlt          string `json:"chartAlt"`
	}
	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL),
		chromedp.Evaluate(`(async () => {
			await Promise.resolve(window.margoPreparePrint());
			const figure = document.querySelector('#diagram');
			const chartImage = document.querySelector('[data-margo-chart-print-image]');
			return {
				buttons: document.querySelectorAll('button').length,
				inputs: document.querySelectorAll('input').length,
				staticLabels: document.querySelectorAll('[data-margo-print-static]').length,
				tableHeaders: document.querySelectorAll('th.margo-table-sort-button').length,
				nestedLinks: document.querySelectorAll('#nested[href="https://example.com"]').length,
				detailsOpen: document.querySelector('.margo-mermaid__source').open,
				paginationMarkers: document.querySelectorAll('[data-margo-print-break-before], [data-margo-print-oversized], [data-margo-print-heading-group]').length,
				scale: figure.dataset.margoPrintScale || '',
				printChartImages: document.querySelectorAll('[data-margo-chart-print-image]').length,
				interactiveCharts: document.querySelectorAll('.goshtoso-charts-interactive').length,
				chartAlt: chartImage?.getAttribute('alt') || '',
			};
		})()`, &prepared, func(parameters *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams {
			return parameters.WithAwaitPromise(true)
		}),
	); err != nil {
		t.Fatal(err)
	}
	if prepared.Buttons != 0 || prepared.Inputs != 0 || prepared.StaticLabels != 3 || prepared.TableHeaders != 1 || prepared.NestedLinks != 1 || !prepared.DetailsOpen {
		t.Fatalf("static print preparation = %+v", prepared)
	}
	if prepared.PaginationMarkers != 0 || prepared.Scale != "" {
		t.Fatalf("print preparation predicted pagination: %+v", prepared)
	}
	if prepared.PrintChartImages != 1 || prepared.InteractiveCharts != 0 || prepared.ChartAlt != "Interactive revenue" {
		t.Fatalf("interactive print chart = %+v", prepared)
	}

	var restored struct {
		Buttons           int  `json:"buttons"`
		Inputs            int  `json:"inputs"`
		StaticLabels      int  `json:"staticLabels"`
		TableHeaders      int  `json:"tableHeaders"`
		NestedLinks       int  `json:"nestedLinks"`
		DetailsOpen       bool `json:"detailsOpen"`
		PrintChartImages  int  `json:"printChartImages"`
		InteractiveCharts int  `json:"interactiveCharts"`
	}
	if err := chromedp.Run(ctx, chromedp.Evaluate(`(() => {
		window.margoRestorePrintState();
		return {
			buttons: document.querySelectorAll('button').length,
			inputs: document.querySelectorAll('input').length,
			staticLabels: document.querySelectorAll('[data-margo-print-static]').length,
			tableHeaders: document.querySelectorAll('th.margo-table-sort-button').length,
			nestedLinks: document.querySelectorAll('strong #nested[href="https://example.com"]').length,
			detailsOpen: document.querySelector('.margo-mermaid__source').open,
			printChartImages: document.querySelectorAll('[data-margo-chart-print-image]').length,
			interactiveCharts: document.querySelectorAll('.goshtoso-charts-interactive').length,
		};
	})()`, &restored)); err != nil {
		t.Fatal(err)
	}
	if restored.Buttons != 0 || restored.Inputs != 1 || restored.StaticLabels != 0 || restored.TableHeaders != 1 || restored.NestedLinks != 1 || restored.DetailsOpen {
		t.Fatalf("screen structure was not restored exactly: %+v", restored)
	}
	if restored.PrintChartImages != 0 || restored.InteractiveCharts != 1 {
		t.Fatalf("interactive chart was not restored exactly: %+v", restored)
	}
}

func TestBrandedFooterDoesNotCreateTrailingFooterOnlyPage(t *testing.T) {
	browserPath := installedPrintTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	if _, err := exec.LookPath("pdfinfo"); err != nil {
		t.Skip("Poppler pdfinfo unavailable")
	}
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("Poppler pdftotext unavailable")
	}

	var source strings.Builder
	source.WriteString("# Footer pagination\n\n")
	for index := 1; index <= 137; index++ {
		fmt.Fprintf(&source, "Paragraph %03d: This is deliberately long content to consume the page flow and test generated footer placement under tight pagination.\n\n", index)
	}
	result := mustRenderSource(t, source.String())
	logo, err := EmbeddedAsset("logo.svg")
	if err != nil {
		t.Fatal(err)
	}
	component, err := RenderStandalone(result, WithPDFBrand("Margo", logo))
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, component)
	markup = strings.Replace(markup, "</head>", "<style>@page { size: A4 portrait; margin: 24mm 22mm 26mm 22mm; }</style></head>", 1)
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
	ctx, cancel := context.WithTimeout(browserContext, 30*time.Second)
	defer cancel()

	var pdfBytes []byte
	if err := chromedp.Run(ctx,
		chromedp.Navigate(server.URL),
		chromedp.WaitVisible(`article.margo-document h1`, chromedp.ByQuery),
		chromedp.Evaluate(`(async () => { if (typeof window.margoPreparePrint === "function") await window.margoPreparePrint(); return true; })()`, nil, func(parameters *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams {
			return parameters.WithAwaitPromise(true)
		}),
		emulation.SetEmulatedMedia().WithMedia("print"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBytes, _, err = page.PrintToPDF().WithPrintBackground(true).WithPreferCSSPageSize(true).WithGenerateTaggedPDF(true).WithGenerateDocumentOutline(true).Do(ctx)
			return err
		}),
	); err != nil {
		t.Fatal(err)
	}

	pdfPath := filepath.Join(t.TempDir(), "footer.pdf")
	if err := os.WriteFile(pdfPath, pdfBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	pageOutput, err := exec.Command("pdfinfo", pdfPath).Output()
	if err != nil {
		t.Fatalf("pdfinfo: %v", err)
	}
	pageCount := 0
	for _, line := range strings.Split(string(pageOutput), "\n") {
		if strings.HasPrefix(line, "Pages:") {
			pageCount, err = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Pages:")))
			if err != nil {
				t.Fatalf("page count: %v", err)
			}
		}
	}
	if pageCount < 2 {
		t.Fatalf("page count = %d; stress fixture should span multiple pages", pageCount)
	}
	textOutput, err := exec.Command("pdftotext", "-f", strconv.Itoa(pageCount), "-l", strconv.Itoa(pageCount), "-layout", pdfPath, "-").Output()
	if err != nil {
		t.Fatalf("pdftotext final page: %v", err)
	}
	finalText := string(textOutput)
	if !strings.Contains(finalText, "Paragraph 137:") || !strings.Contains(finalText, "Generated by Margo") {
		t.Fatalf("final page lost content or footer: %q", finalText)
	}
}

func installedPrintTestChromium() string {
	candidates := []string{}
	if runtime.GOOS == "darwin" {
		candidates = append(candidates,
			"/opt/homebrew/bin/chromium",
			"/usr/local/bin/chromium",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		)
	}
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser"} {
		if path, err := exec.LookPath(name); err == nil {
			candidates = append(candidates, path)
		}
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}
