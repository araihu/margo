package charts

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	chartassets "github.com/araihu/goshtoso-charts/assets"
	"github.com/araihu/goshtoso-charts/components/chartcontrol"
)

func TestChartControlConfigDefaultsToEnabledWrapper(t *testing.T) {
	controls, export := chartControlConfig(defaultChartRenderOptions)
	if controls.Mode != chartcontrol.WrapperModeEnabled {
		t.Fatalf("wrapper mode = %q, want enabled", controls.Mode)
	}
	if export != nil {
		t.Fatalf("default export config = %#v, want capability-derived exports", export)
	}
}

func TestChartControlConfigCanOmitWrapperAndExports(t *testing.T) {
	controls, export := chartControlConfig(chartRenderOptions{controlWrapper: false})
	if controls.Mode != chartcontrol.WrapperModeOmitted {
		t.Fatalf("wrapper mode = %q, want omitted", controls.Mode)
	}
	if export == nil || !export.Disabled {
		t.Fatalf("static export config = %#v, want disabled", export)
	}
}

func TestInteractiveChartWrapperOwnsAlpineScope(t *testing.T) {
	chart := templ.ComponentFunc(func(_ context.Context, out io.Writer) error {
		_, err := io.WriteString(out, `<style>chart</style><div class="goshtoso-charts-control-wrapper" data-goshtoso-chart-wrapper><button x-on:click="run()">Expand</button></div>`)
		return err
	})
	var out bytes.Buffer
	if err := renderWithChartControlAlpineRoot(context.Background(), chart, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `<div x-data class="goshtoso-charts-control-wrapper" data-goshtoso-chart-wrapper>`) {
		t.Fatalf("interactive chart wrapper missing local Alpine scope: %s", out.String())
	}
}

func TestStaticChartDoesNotGainInteractiveAlpineScope(t *testing.T) {
	chart := templ.ComponentFunc(func(_ context.Context, out io.Writer) error {
		_, err := io.WriteString(out, `<div class="chart">static</div>`)
		return err
	})
	var out bytes.Buffer
	if err := applyChartPrintPolicy(chart, chartRenderOptions{controlWrapper: false}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	markup := out.String()
	if strings.Contains(markup, `x-data`) {
		t.Fatalf("static chart unexpectedly contains Alpine scope: %s", markup)
	}
	for _, want := range []string{
		`data-margo-chart-print`,
		`.margo-chart-data`,
		`[data-goshtoso-chart-content] > details`,
		`display: none !important`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("static chart default print policy missing %q: %s", want, markup)
		}
	}
}

func TestChartPrintPolicyStylesGoshtosoFigcaptionAsCaption(t *testing.T) {
	for _, options := range []chartRenderOptions{
		{printAccessibleData: false},
		{printAccessibleData: true},
	} {
		style := chartPrintStyle(options)
		for _, want := range []string{
			`.goshtoso-charts-bar > figcaption`,
			`.goshtoso-charts-line > figcaption`,
			`.goshtoso-charts-interactive > figcaption`,
			`color: var(--color-on-surface)`,
			`font-size: var(--text-sm)`,
			`line-height: var(--text-sm--line-height)`,
			`margin: 0.5rem auto 0`,
			`max-width: 80%`,
			`text-align: center`,
		} {
			if !strings.Contains(style, want) {
				t.Fatalf("print style missing caption treatment %q: %s", want, style)
			}
		}
	}
}

func TestChartHTMLPolicyStylesGoshtosoFigcaptionAsCaption(t *testing.T) {
	for _, options := range []chartRenderOptions{
		{printAccessibleData: false},
		{printAccessibleData: true},
	} {
		style := chartPrintStyle(options)
		screenStyle := strings.SplitN(style, `@media print`, 2)[0]
		for _, want := range []string{
			`.goshtoso-charts-bar > figcaption`,
			`.goshtoso-charts-line > figcaption`,
			`.goshtoso-charts-interactive > figcaption`,
			`color: var(--color-on-surface)`,
			`font-size: var(--text-sm)`,
			`line-height: var(--text-sm--line-height)`,
			`margin: 0.5rem auto 0`,
			`max-width: 80%`,
			`text-align: center`,
		} {
			if !strings.Contains(screenStyle, want) {
				t.Fatalf("screen style missing caption treatment %q: %s", want, screenStyle)
			}
		}
	}
}

func TestChartScreenPolicyShowsOnlyStyledMargoTable(t *testing.T) {
	for _, options := range []chartRenderOptions{
		{printAccessibleData: false},
		{printAccessibleData: true},
	} {
		style := chartPrintStyle(options)
		for _, want := range []string{
			`[data-margo-chart-with-data="v1"] > details`,
			`[data-margo-chart-with-data="v1"] [data-goshtoso-chart-content] > details`,
			`display: none !important`,
			`.margo-chart-data table`,
			`border: 1px solid var(--color-outline)`,
			`background: var(--color-surface-alt)`,
			`.margo-chart-data caption`,
			`color: var(--color-on-surface-strong)`,
			`width: 100%`,
		} {
			if !strings.Contains(style, want) {
				t.Fatalf("screen chart policy missing %q: %s", want, style)
			}
		}
		if strings.Contains(style, `#525252`) || strings.Contains(style, `#a3a3a3`) {
			t.Fatalf("chart style bypasses semantic Goshtoso color tokens: %s", style)
		}
	}
}

func TestChartControlLoaderIsSuppressedOnlyWhenExternalized(t *testing.T) {
	exactLoader := `<script src="` + chartassets.ControlRuntimeURL + `" defer></script>`
	changedLoader := `<script defer src="` + chartassets.ControlRuntimeURL + `"></script>`
	markedChangedLoader := `<script ` + chartExtensionScriptAttribute + ` defer src="` + chartassets.ControlRuntimeURL + `"></script>`
	chart := templ.ComponentFunc(func(_ context.Context, out io.Writer) error {
		_, err := io.WriteString(out, `<div class="goshtoso-charts-control-wrapper" data-goshtoso-chart-wrapper><svg aria-label="Chart"></svg><table><tr><td>Data</td></tr></table>`+exactLoader+changedLoader+`</div>`)
		return err
	})
	var output bytes.Buffer
	options := chartRenderOptions{controlWrapper: true, externalizedControlRuntime: true}
	if err := applyChartPrintPolicy(chart, options).Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	markup := output.String()
	if strings.Contains(markup, exactLoader) {
		t.Fatalf("exact loader remained: %s", markup)
	}
	for _, want := range []string{markedChangedLoader, `<svg aria-label="Chart">`, `<table>`, `<div x-data class="goshtoso-charts-control-wrapper"`} {
		if !strings.Contains(markup, want) {
			t.Fatalf("externalized chart missing %q: %s", want, markup)
		}
	}
	if !strings.Contains(markup, `<style data-margo-extension-style="charts" data-margo-chart-print data-margo-chart-print-data="disabled">`) {
		t.Fatalf("chart print style is not provenance-marked: %s", markup)
	}
}

func TestChartControlLoaderIsExternalizedForEveryFamily(t *testing.T) {
	for _, fixture := range []string{
		"testdata/bar/bar-valid.yaml",
		"testdata/line/line-valid.yaml",
		"testdata/pie/pie-valid.json",
		"testdata/scatter/scatter-valid.yaml",
	} {
		t.Run(fixture, func(t *testing.T) {
			body := readChartFixtureForIntegration(t, fixture)
			out, _, err := renderThroughRootWithExtension(t, body, 1<<20, Extension(WithExternalizedControlRuntime(true)))
			if err != nil {
				t.Fatal(err)
			}
			markup := string(out)
			if strings.Contains(markup, `src="`+chartassets.ControlRuntimeURL+`"`) {
				t.Fatalf("externalized chart retained loader: %s", markup)
			}
			for _, want := range []string{`data-goshtoso-chart-wrapper`, `data-margo-chart-data="v1"`, `<svg`} {
				if !strings.Contains(markup, want) {
					t.Fatalf("externalized chart missing %q: %s", want, markup)
				}
			}
			if strings.Contains(markup, `<style>`) || strings.Count(markup, `data-margo-extension-style="charts"`) != strings.Count(markup, `</style>`) {
				t.Fatalf("chart styles are not provenance-marked: %s", markup)
			}
		})
	}
}
