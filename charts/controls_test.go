package charts

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
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
	if strings.Contains(out.String(), `x-data`) || strings.Contains(out.String(), `data-margo-chart-print`) {
		t.Fatalf("static chart unexpectedly contains interactive policy: %s", out.String())
	}
}
