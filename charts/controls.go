package charts

import "github.com/araihu/goshtoso-charts/components/chartcontrol"

// chartControlConfig maps the extension-level choice to the upstream shared
// wrapper. Omitted mode is deliberately paired with disabled exports so the
// static path contains no browser lifecycle or export affordance.
func chartControlConfig(options chartRenderOptions) (chartcontrol.Options, *chartcontrol.ExportOptions) {
	if options.controlWrapper {
		return chartcontrol.Options{Mode: chartcontrol.WrapperModeEnabled}, nil
	}
	return chartcontrol.Options{Mode: chartcontrol.WrapperModeOmitted}, &chartcontrol.ExportOptions{Disabled: true}
}
