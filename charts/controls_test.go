package charts

import (
	"testing"

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
