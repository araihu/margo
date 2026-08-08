package charts

import (
	"bytes"
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
		return renderWithChartControlAlpineRoot(ctx, chart, out)
	})
}

// renderWithChartControlAlpineRoot gives action buttons a local Alpine scope.
// The upstream chart wrapper emits x-on handlers on the action buttons while
// its modal owns a separate x-data scope; without a root scope, Alpine skips
// those handlers in standalone documents that do not have an app-wide x-data.
func renderWithChartControlAlpineRoot(ctx context.Context, chart templ.Component, out io.Writer) error {
	writer := &chartControlAlpineWriter{out: out}
	if err := chart.Render(ctx, writer); err != nil {
		return err
	}
	return writer.flush()
}

const chartControlWrapperStart = `<div class="goshtoso-charts-control-wrapper" data-goshtoso-chart-wrapper`

type chartControlAlpineWriter struct {
	out         io.Writer
	pending     bytes.Buffer
	transformed bool
}

func (writer *chartControlAlpineWriter) Write(data []byte) (int, error) {
	if writer.transformed {
		return writer.out.Write(data)
	}
	if _, err := writer.pending.Write(data); err != nil {
		return 0, err
	}
	pending := writer.pending.Bytes()
	index := bytes.Index(pending, []byte(chartControlWrapperStart))
	if index < 0 {
		return len(data), nil
	}
	if _, err := writer.out.Write(pending[:index]); err != nil {
		return 0, err
	}
	if _, err := writer.out.Write(append([]byte(`<div x-data`), pending[index+len(`<div`):]...)); err != nil {
		return 0, err
	}
	writer.pending.Reset()
	writer.transformed = true
	return len(data), nil
}

func (writer *chartControlAlpineWriter) flush() error {
	if writer.transformed || writer.pending.Len() == 0 {
		return nil
	}
	_, err := writer.out.Write(writer.pending.Bytes())
	return err
}
