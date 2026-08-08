package charts

import (
	"context"
	"io"

	"github.com/a-h/templ"
	"github.com/araihu/goshtoso-charts/components/chartcontrol"
)

const chartPrintControlStyle = `<style data-margo-chart-print>@media print {
  [data-goshtoso-chart-wrapper] [data-goshtoso-chart-actions-fieldset],
  [data-goshtoso-chart-wrapper] [data-goshtoso-chart-expand],
  [data-goshtoso-chart-wrapper] [data-goshtoso-chart-export-status] {
    display: none !important;
  }
}</style>`

// chartControlConfig maps the extension-level choice to the upstream shared
// wrapper. Omitted mode is deliberately paired with disabled exports so the
// static path contains no browser lifecycle or export affordance.
func chartControlConfig(options chartRenderOptions) (chartcontrol.Options, *chartcontrol.ExportOptions) {
	if options.controlWrapper {
		return chartcontrol.Options{Mode: chartcontrol.WrapperModeEnabled}, nil
	}
	return chartcontrol.Options{Mode: chartcontrol.WrapperModeOmitted}, &chartcontrol.ExportOptions{Disabled: true}
}

// applyChartPrintPolicy keeps the interactive wrapper available on screen but
// removes its actions from browser print/PDF output. The style is emitted only
// for the enabled wrapper; static charts remain free of control-specific CSS.
func applyChartPrintPolicy(chart templ.Component, options chartRenderOptions) templ.Component {
	if !options.controlWrapper {
		return chart
	}
	return templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		if _, err := io.WriteString(out, chartPrintControlStyle); err != nil {
			return err
		}
		return chart.Render(ctx, out)
	})
}
