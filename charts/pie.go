package charts

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/araihu/goshtoso-charts/components/pie"
	margo "github.com/araihu/margo"
)

type pieModel struct {
	SchemaVersion int             `yaml:"schemaVersion"`
	Type          string          `yaml:"type"`
	Title         string          `yaml:"title"`
	Style         chartStyleModel `yaml:"style"`
	Slices        []pieSliceModel `yaml:"slices"`
}

type pieSliceModel struct {
	Name            string  `yaml:"name"`
	Value           float64 `yaml:"value"`
	chartPaintModel `yaml:",inline"`
}

func validatePieModel(model pieModel) error {
	if err := validateChartStyle(model.Style); err != nil {
		return err
	}
	if model.SchemaVersion != 1 || (model.Type != "pie" && model.Type != "doughnut") {
		return chartDiagnostic("chart.schema_invalid", "pie model envelope is invalid")
	}
	if strings.TrimSpace(model.Title) == "" {
		return chartDiagnostic("chart.semantic_title_invalid", "pie title is required")
	}
	if len(model.Slices) == 0 || len(model.Slices) > 256 {
		return chartDiagnostic("chart.resource_slices_invalid", "pie chart requires 1 to 256 slices")
	}
	seen := make(map[string]struct{}, len(model.Slices))
	for _, slice := range model.Slices {
		if err := validateChartPaint(slice.chartPaintModel, fmt.Sprintf("pie slice %q", slice.Name)); err != nil {
			return err
		}
		if strings.TrimSpace(slice.Name) == "" {
			return chartDiagnostic("chart.semantic_slice_invalid", "pie slice name is required")
		}
		if _, exists := seen[slice.Name]; exists {
			return chartDiagnostic("chart.semantic_slice_duplicate", "pie slice names must be unique")
		}
		seen[slice.Name] = struct{}{}
		if math.IsNaN(slice.Value) || math.IsInf(slice.Value, 0) {
			return chartDiagnostic("chart.value_non_finite", "pie slice values must be finite")
		}
		if slice.Value < 0 {
			return chartDiagnostic("chart.semantic_value_negative", "pie slice values must be non-negative")
		}
	}
	return nil
}

func renderPie(rc margo.RenderContext, model pieModel) (templ.Component, error) {
	return renderPieWithOptions(rc, model, defaultChartRenderOptions)
}

func renderPieWithOptions(rc margo.RenderContext, model pieModel, options chartRenderOptions) (templ.Component, error) {
	if err := validatePieModel(model); err != nil {
		return nil, err
	}
	variant := pie.VariantPie
	if model.Type == "doughnut" {
		variant = pie.VariantDoughnut
	}
	slices := make([]pie.Slice, len(model.Slices))
	paints := make([]chartPaintModel, len(model.Slices))
	rows := make([]AccessibleRow, 0, len(model.Slices))
	for index, source := range model.Slices {
		paint := source.chartPaintModel.normalized()
		paints[index] = paint
		slices[index] = pie.Slice{Name: source.Name, Value: source.Value, Color: paint.Color, Class: paint.Class}
		rows = append(rows, AccessibleRow{Category: source.Name, Value: strconv.FormatFloat(source.Value, 'f', -1, 64)})
	}
	controlOptions, exportOptions := chartControlConfig(options)
	component := pie.Pie(pie.Config{
		Label:    model.Title,
		Slices:   slices,
		Variant:  variant,
		Controls: controlOptions,
		Export:   exportOptions,
		Style:    chartThemeForSeries(model.Style, paints),
	})
	chartComponent := applyChartPrintPolicy(templ.Component(component), options)
	return WithAccessibleData(chartComponent, AccessibleData{Title: model.Title, Rows: rows}, AccessibleRenderPolicy{MaxOutputBytes: rc.EffectivePolicy.OutputBytes}), nil
}

func init() {
	registerFamilyHandler("pie", func(rc margo.RenderContext, raw any, options chartRenderOptions) (templ.Component, error) {
		model, ok := raw.(pieModel)
		if !ok {
			return nil, chartDiagnostic("chart.model_invalid", "pie handler received the wrong model")
		}
		return renderPieWithOptions(rc, model, options)
	})
	registerFamilyHandler("doughnut", func(rc margo.RenderContext, raw any, options chartRenderOptions) (templ.Component, error) {
		model, ok := raw.(pieModel)
		if !ok {
			return nil, chartDiagnostic("chart.model_invalid", "doughnut handler received the wrong model")
		}
		model.Type = "doughnut"
		return renderPieWithOptions(rc, model, options)
	})
}
