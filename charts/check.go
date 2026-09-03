package charts

import (
	"context"

	margo "github.com/araihu/margo"
)

// checkChart performs the same payload and semantic validation used by the
// renderer, then applies the chart projection contract for the requested
// target. The CLI deck projection is deliberately static: its navigation and
// PDF validator own the browser lifecycle, so chart controls are not part of
// that artifact contract.
func checkChart(ctx context.Context, node margo.ExtensionNode, options chartRenderOptions) error {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	envelope, err := decodeEnvelope(node.Payload)
	if err != nil {
		return err
	}
	if err := validateChartSemantics(envelope.Model); err != nil {
		return err
	}
	renderer := chartRenderer(envelope.Model)
	if renderer != "interactive" {
		return nil
	}
	if node.Target == margo.TargetDeck || options.deckProjection {
		return chartDiagnosticWithContract(
			"chart.renderer_target_unsupported",
			"interactive charts are not supported by the deck target; deck HTML and PDF use static chart projections",
			"/renderer",
			"Omit renderer or set renderer: static for margo deck; use margo html, margo site, or margo pdf for interactive charts.",
		)
	}
	if !options.controlWrapper {
		return chartDiagnosticWithContract(
			"chart.renderer_controls_required",
			"interactive renderer requires the chart control wrapper",
			"/renderer",
			"Enable the default chart control wrapper or use renderer: static.",
		)
	}
	return nil
}

func validateChartSemantics(model any) error {
	switch typed := model.(type) {
	case barModel:
		return validateBarSemantics(typed)
	case lineModel:
		return validateLineSemantics(typed)
	case pieModel:
		return validatePieSemantics(typed)
	case scatterModel:
		return validateScatterSemantics(typed)
	default:
		return chartDiagnostic("chart.model_invalid", "chart checker received an unsupported model")
	}
}

func chartRenderer(model any) string {
	switch typed := model.(type) {
	case barModel:
		return typed.Renderer
	case lineModel:
		return typed.Renderer
	case pieModel:
		return typed.Renderer
	case scatterModel:
		return typed.Renderer
	default:
		return ""
	}
}

func chartDiagnosticWithContract(code, message, pointer, hint string) error {
	return &margo.DiagnosticError{Diagnostics: []margo.Diagnostic{{
		Code: code, Severity: margo.SeverityError, Pointer: pointer, Message: message, Hint: hint,
	}}}
}

func interactiveRendererUnavailable(options chartRenderOptions) error {
	if options.deckProjection {
		return chartDiagnosticWithContract(
			"chart.renderer_target_unsupported",
			"interactive charts are not supported by the deck target; deck HTML and PDF use static chart projections",
			"/renderer",
			"Omit renderer or set renderer: static for margo deck; use margo html, margo site, or margo pdf for interactive charts.",
		)
	}
	return chartDiagnostic("chart.renderer_controls_required", "interactive renderer requires the chart control wrapper")
}
