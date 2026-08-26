package charts

import (
	"bytes"
	"context"
	"io"

	"github.com/a-h/templ"
	chartassets "github.com/araihu/goshtoso-charts/assets"
	"github.com/araihu/goshtoso-charts/components/chartcontrol"
)

const chartScreenDataStyle = `.goshtoso-charts-bar > figcaption,
  .goshtoso-charts-line > figcaption,
  .goshtoso-charts-interactive > figcaption {
    color: var(--color-on-surface);
    font-size: var(--text-sm);
    font-weight: var(--font-weight-medium);
    font-style: italic;
    line-height: var(--text-sm--line-height);
    margin: 0.5rem auto 0;
    max-width: 80%;
    text-align: center;
  }
  .dark .goshtoso-charts-bar > figcaption,
  .dark .goshtoso-charts-line > figcaption,
  .dark .goshtoso-charts-interactive > figcaption {
    color: var(--color-on-surface-dark);
  }
  [data-margo-chart-with-data="v1"] > details,
  [data-margo-chart-with-data="v1"] [data-goshtoso-chart-content] > details {
    display: none !important;
  }
  .margo-chart-data {
    width: 100%;
    max-width: 100%;
    margin-block: 1rem;
    overflow-x: auto;
  }
  .margo-chart-data table {
    width: 100%;
    border: 1px solid var(--color-outline);
    border-collapse: collapse;
    font-size: 0.875rem;
    text-align: left;
  }
  .margo-chart-data caption {
    padding-block: 0.5rem;
    color: var(--color-on-surface-strong);
    font-weight: 600;
    text-align: left;
  }
  .margo-chart-data thead {
    background: var(--color-surface-alt);
  }
  .margo-chart-data th,
  .margo-chart-data td {
    border: 1px solid var(--color-outline);
    padding: 0.5rem 0.625rem;
    text-align: left;
  }
  .margo-chart-data [data-field="canonical"] {
    display: none !important;
  }
  .dark .margo-chart-data table,
  .dark .margo-chart-data th,
  .dark .margo-chart-data td {
    border-color: var(--color-outline-dark);
  }
  .dark .margo-chart-data caption {
    color: var(--color-on-surface-dark-strong);
  }
  .dark .margo-chart-data thead {
    background: var(--color-surface-dark-alt);
  }
  `

const chartPrintHiddenDataStyle = `<style data-margo-extension-style="charts" data-margo-chart-print data-margo-chart-print-data="disabled">` + chartScreenDataStyle + `@media print {
  [data-goshtoso-chart-wrapper] [data-goshtoso-chart-actions-fieldset],
  [data-goshtoso-chart-wrapper] [data-goshtoso-chart-expand],
  [data-goshtoso-chart-wrapper] [data-goshtoso-chart-export-status],
  [data-goshtoso-chart-content] > details,
  .margo-chart-data {
    display: none !important;
}
}</style>`

const chartPrintVisibleDataStyle = `<style data-margo-extension-style="charts" data-margo-chart-print data-margo-chart-print-data="enabled">` + chartScreenDataStyle + `@media print {
  [data-goshtoso-chart-wrapper] [data-goshtoso-chart-actions-fieldset],
  [data-goshtoso-chart-wrapper] [data-goshtoso-chart-expand],
  [data-goshtoso-chart-wrapper] [data-goshtoso-chart-export-status],
  [data-goshtoso-chart-content] > details {
    display: none !important;
  }
  .margo-chart-data {
    display: block !important;
    margin-block: 1rem;
    break-inside: avoid;
  }
  .margo-chart-data table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.875rem;
  }
  .margo-chart-data caption {
    margin-bottom: 0.5rem;
    font-weight: 600;
    text-align: left;
  }
  .margo-chart-data th,
  .margo-chart-data td {
    border: 1px solid var(--color-outline);
    padding: 0.35rem 0.5rem;
    text-align: left;
  }
  .margo-chart-data [data-field="canonical"] {
    display: none !important;
  }
}</style>`

func chartPrintStyle(options chartRenderOptions) string {
	if options.printAccessibleData {
		return chartPrintVisibleDataStyle
	}
	return chartPrintHiddenDataStyle
}

// PrintableAccessibleDataStyle returns the print stylesheet used when exact
// chart data tables are enabled. Hosts that cannot choose the charts
// extension's frozen registration options (for example configured site
// publication) can append this stylesheet to their PDF-only HTML projection.
func PrintableAccessibleDataStyle() string {
	return chartPrintVisibleDataStyle
}

// chartControlConfig maps the extension-level choice to the upstream shared
// wrapper. Omitted mode is deliberately paired with disabled exports so the
// static path contains no browser lifecycle or export affordance.
func chartControlConfig(options chartRenderOptions) (chartcontrol.Options, *chartcontrol.ExportOptions) {
	if options.controlWrapper {
		return chartcontrol.Options{Mode: chartcontrol.WrapperModeEnabled}, nil
	}
	return chartcontrol.Options{Mode: chartcontrol.WrapperModeOmitted}, &chartcontrol.ExportOptions{Disabled: true}
}

// applyChartPrintPolicy keeps chart accessibility data in HTML while removing
// it and interactive actions from browser print/PDF output by default.
func applyChartPrintPolicy(chart templ.Component, options chartRenderOptions) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if _, err := io.WriteString(out, chartPrintStyle(options)); err != nil {
			return err
		}
		if !options.controlWrapper {
			// Static-only hosts still need extension-owned style provenance;
			// otherwise Margo's fragment validator rejects the chart's bundled
			// stylesheet before it can reach the deck/page renderer.
			writer := &chartControlAlpineWriter{out: out, externalizedControlRuntime: false}
			if err := chart.Render(ctx, writer); err != nil {
				return err
			}
			return writer.flush()
		}
		return renderWithChartControlAlpineRootOptions(ctx, chart, out, options.externalizedControlRuntime)
	})
}

// renderWithChartControlAlpineRoot gives action buttons a local Alpine scope.
// The upstream chart wrapper emits x-on handlers on the action buttons while
// its modal owns a separate x-data scope; without a root scope, Alpine skips
// those handlers in standalone documents that do not have an app-wide x-data.
func renderWithChartControlAlpineRoot(ctx context.Context, chart templ.Component, out io.Writer) error {
	return renderWithChartControlAlpineRootOptions(ctx, chart, out, false)
}

func renderWithChartControlAlpineRootOptions(ctx context.Context, chart templ.Component, out io.Writer, externalizedControlRuntime bool) error {
	writer := &chartControlAlpineWriter{out: out, externalizedControlRuntime: externalizedControlRuntime}
	if err := chart.Render(ctx, writer); err != nil {
		return err
	}
	return writer.flush()
}

const chartControlWrapperStart = `<div class="goshtoso-charts-control-wrapper" data-goshtoso-chart-wrapper`
const chartExtensionStyleAttribute = `data-margo-extension-style="charts"`
const chartExtensionScriptAttribute = `data-margo-extension-script="charts"`

type chartControlAlpineWriter struct {
	out                        io.Writer
	pending                    bytes.Buffer
	externalizedControlRuntime bool
}

func (writer *chartControlAlpineWriter) Write(data []byte) (int, error) {
	return writer.pending.Write(data)
}

func (writer *chartControlAlpineWriter) flush() error {
	if writer.pending.Len() == 0 {
		return nil
	}
	data := append([]byte(nil), writer.pending.Bytes()...)
	if index := bytes.Index(data, []byte(chartControlWrapperStart)); index >= 0 {
		transformed := make([]byte, 0, len(data)+len(` x-data`))
		transformed = append(transformed, data[:index]...)
		transformed = append(transformed, []byte(`<div x-data`)...)
		transformed = append(transformed, data[index+len(`<div`):]...)
		data = transformed
	}
	if writer.externalizedControlRuntime {
		exactLoader := []byte(`<script src="` + chartassets.ControlRuntimeURL + `" defer></script>`)
		data = bytes.ReplaceAll(data, exactLoader, nil)
	}
	data = bytes.ReplaceAll(data, []byte(`<style>`), []byte(`<style `+chartExtensionStyleAttribute+`>`))
	data = bytes.ReplaceAll(data, []byte(`<style data-margo-chart-print>`), []byte(`<style `+chartExtensionStyleAttribute+` data-margo-chart-print>`))
	data = bytes.ReplaceAll(data, []byte(`<script `), []byte(`<script `+chartExtensionScriptAttribute+` `))
	data = bytes.ReplaceAll(data, []byte(`<script>`), []byte(`<script `+chartExtensionScriptAttribute+`>`))
	_, err := writer.out.Write(data)
	return err
}
