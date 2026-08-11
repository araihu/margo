package margo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestPrintPreparationUsesStaticStructureAndScalesTallMermaid(t *testing.T) {
	browserPath := installedPrintTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}
	document := `<!doctype html><html><head><style>
html, body { margin: 0; }
.margo-document { margin: 0; }
.spacer { block-size: 900px; }
.margo-mermaid { padding: 10px; border: 1px solid #ccc; }
.margo-mermaid__canvas { block-size: 1600px; inline-size: 500px; }
.margo-mermaid__canvas svg { block-size: 1600px; inline-size: 500px; }
</style></head><body><div class="goshtoso-document"><article class="margo-document">
<p><strong id="emphasis-target" class="source-strong">Strong <a id="nested-link" href="https://example.com/evidence">evidence</a></strong> and <em>important context</em>.</p>
<div data-table-client-sort="true"><table><thead><tr><th><button class="margo-table-sort-button">Column</button></th></tr></thead></table></div>
<p><input type="checkbox" disabled checked aria-label="Completed task"> Complete</p>
<div class="spacer"></div>
<h2 id="tall-heading">Tall diagram</h2>
<figure id="tall-mermaid" class="margo-mermaid"><div class="margo-mermaid__canvas"><svg viewBox="0 0 500 1600"></svg></div><details class="margo-mermaid__source"><summary>Source</summary><pre>graph TD</pre></details></figure>
</article></div>` + standalonePrintPaginationScript + `</body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = writer.Write([]byte(document))
	}))
	defer server.Close()

	allocatorOptions := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	allocatorOptions = append(allocatorOptions, chromedp.ExecPath(browserPath))
	allocatorContext, cancelAllocator := chromedp.NewExecAllocator(context.Background(), allocatorOptions...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 20*time.Second)
	defer cancel()

	var prepared struct {
		Buttons        int     `json:"buttons"`
		Inputs         int     `json:"inputs"`
		Strong         int     `json:"strong"`
		Emphasis       int     `json:"emphasis"`
		StaticLabels   int     `json:"staticLabels"`
		NestedLinks    int     `json:"nestedLinks"`
		PreservedAttrs int     `json:"preservedAttrs"`
		Scale          float64 `json:"scale"`
		Oversized      string  `json:"oversized"`
		HeadingBreak   string  `json:"headingBreak"`
		FigureHeight   float64 `json:"figureHeight"`
		ViewportHeight float64 `json:"viewportHeight"`
	}
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(794, 1123),
		chromedp.Navigate(server.URL),
		chromedp.Evaluate(`(() => {
			window.margoPreparePrintTOC();
			const figure = document.querySelector('#tall-mermaid');
			return {
				buttons: document.querySelectorAll('button').length,
				inputs: document.querySelectorAll('input').length,
				strong: document.querySelectorAll('strong, b').length,
				emphasis: document.querySelectorAll('em, i').length,
				staticLabels: document.querySelectorAll('[data-margo-print-static]').length,
				nestedLinks: document.querySelectorAll('#nested-link[href="https://example.com/evidence"]').length,
				preservedAttrs: document.querySelectorAll('span#emphasis-target.source-strong.margo-print-strong').length,
				scale: Number.parseFloat(figure.dataset.margoPrintScale || '1'),
				oversized: figure.dataset.margoPrintOversized || '',
				headingBreak: document.querySelector('#tall-heading').dataset.margoPrintBreakBefore || '',
				figureHeight: figure.getBoundingClientRect().height,
				viewportHeight: window.innerHeight,
			};
		})()`, &prepared),
	); err != nil {
		t.Fatal(err)
	}
	if prepared.Buttons != 0 || prepared.Inputs != 0 || prepared.Strong != 0 || prepared.Emphasis != 0 {
		t.Fatalf("interactive/invalid print structure remains: %+v", prepared)
	}
	if prepared.StaticLabels != 4 {
		t.Fatalf("static print labels = %d, want 4", prepared.StaticLabels)
	}
	if prepared.NestedLinks != 1 || prepared.PreservedAttrs != 1 {
		t.Fatalf("nested semantic content was lost during print projection: %+v", prepared)
	}
	if prepared.Scale <= 0 || prepared.Scale >= 1 || prepared.Oversized != "true" {
		t.Fatalf("tall Mermaid was not scaled and marked: %+v", prepared)
	}
	if prepared.FigureHeight > prepared.ViewportHeight+1 {
		t.Fatalf("scaled Mermaid height %.2f exceeds page height %.2f", prepared.FigureHeight, prepared.ViewportHeight)
	}
	if prepared.HeadingBreak != "page" {
		t.Fatalf("tall Mermaid heading break = %q, want page", prepared.HeadingBreak)
	}

	var restored struct {
		Buttons        int    `json:"buttons"`
		Inputs         int    `json:"inputs"`
		Strong         int    `json:"strong"`
		Emphasis       int    `json:"emphasis"`
		StaticLabels   int    `json:"staticLabels"`
		NestedLinks    int    `json:"nestedLinks"`
		PreservedAttrs int    `json:"preservedAttrs"`
		Scale          string `json:"scale"`
		CanvasZoom     string `json:"canvasZoom"`
		HeadingBreak   string `json:"headingBreak"`
	}
	if err := chromedp.Run(ctx, chromedp.Evaluate(`(() => {
		window.margoRestorePrintState();
		const figure = document.querySelector('#tall-mermaid');
		return {
			buttons: document.querySelectorAll('button').length,
			inputs: document.querySelectorAll('input').length,
			strong: document.querySelectorAll('strong, b').length,
			emphasis: document.querySelectorAll('em, i').length,
			staticLabels: document.querySelectorAll('[data-margo-print-static]').length,
			nestedLinks: document.querySelectorAll('strong #nested-link[href="https://example.com/evidence"]').length,
			preservedAttrs: document.querySelectorAll('strong#emphasis-target.source-strong').length,
			scale: figure.dataset.margoPrintScale || '',
			canvasZoom: figure.querySelector('.margo-mermaid__canvas').style.zoom,
			headingBreak: document.querySelector('#tall-heading').dataset.margoPrintBreakBefore || '',
		};
	})()`, &restored)); err != nil {
		t.Fatal(err)
	}
	if restored.Buttons != 1 || restored.Inputs != 1 || restored.Strong != 1 || restored.Emphasis != 1 || restored.StaticLabels != 0 || restored.NestedLinks != 1 || restored.PreservedAttrs != 1 || restored.Scale != "" || restored.CanvasZoom != "" || restored.HeadingBreak != "" {
		t.Fatalf("screen structure was not restored exactly: %+v", restored)
	}
}

func installedPrintTestChromium() string {
	candidates := []string{}
	if runtime.GOOS == "darwin" {
		candidates = append(candidates, "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome")
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
