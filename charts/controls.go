package charts

import (
	"bytes"
	"context"
	"io"

	"github.com/a-h/templ"
	chartassets "github.com/araihu/goshtoso-charts/assets"
	"github.com/araihu/goshtoso-charts/components/chartcontrol"
)

const chartPrintControlStyle = `<style data-margo-extension-style="charts" data-margo-chart-print>@media print {
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
	_, err := writer.out.Write(data)
	return err
}
