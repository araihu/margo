package charts

import (
	"math"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/araihu/goshtoso-charts/components/line"
	margo "github.com/araihu/margo"
)

type lineModel struct {
	SchemaVersion int               `yaml:"schemaVersion"`
	Type          string            `yaml:"type"`
	Title         string            `yaml:"title"`
	Categories    []string          `yaml:"categories"`
	Series        []lineSeriesModel `yaml:"series"`
}

type lineSeriesModel struct {
	Name   string    `yaml:"name"`
	Values []float64 `yaml:"values"`
}

func validateLineModel(model lineModel) error {
	if model.SchemaVersion != 1 || model.Type != "line" {
		return chartDiagnostic("chart.schema_invalid", "line model envelope is invalid")
	}
	if strings.TrimSpace(model.Title) == "" {
		return chartDiagnostic("chart.semantic_title_invalid", "line title is required")
	}
	if err := validateLineCategories(model.Categories); err != nil {
		return err
	}
	if len(model.Series) == 0 || len(model.Series) > 256 {
		return chartDiagnostic("chart.resource_series_invalid", "line chart requires 1 to 256 series")
	}
	seen := make(map[string]struct{}, len(model.Series))
	for _, series := range model.Series {
		if strings.TrimSpace(series.Name) == "" {
			return chartDiagnostic("chart.semantic_series_invalid", "line series name is required")
		}
		if _, exists := seen[series.Name]; exists {
			return chartDiagnostic("chart.semantic_series_duplicate", "line series names must be unique")
		}
		seen[series.Name] = struct{}{}
		if len(series.Values) != len(model.Categories) {
			return chartDiagnostic("chart.semantic_alignment_invalid", "line series values must align with categories")
		}
		for _, value := range series.Values {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return chartDiagnostic("chart.value_non_finite", "line values must be finite")
			}
		}
	}
	return nil
}

func validateLineCategories(categories []string) error {
	if len(categories) == 0 || len(categories) > 4096 {
		return chartDiagnostic("chart.resource_categories_invalid", "line chart requires 1 to 4096 categories")
	}
	seen := make(map[string]struct{}, len(categories))
	for _, category := range categories {
		if strings.TrimSpace(category) == "" {
			return chartDiagnostic("chart.semantic_category_invalid", "line category cannot be empty")
		}
		if _, exists := seen[category]; exists {
			return chartDiagnostic("chart.semantic_category_duplicate", "line categories must be unique")
		}
		seen[category] = struct{}{}
	}
	return nil
}

func renderLine(rc margo.RenderContext, model lineModel) (templ.Component, error) {
	return renderLineWithOptions(rc, model, defaultChartRenderOptions)
}

func renderLineWithOptions(rc margo.RenderContext, model lineModel, options chartRenderOptions) (templ.Component, error) {
	if err := validateLineModel(model); err != nil {
		return nil, err
	}
	series := make([]line.Series, len(model.Series))
	rows := make([]AccessibleRow, 0, len(model.Series)*len(model.Categories))
	for seriesIndex, source := range model.Series {
		series[seriesIndex] = line.Series{Name: source.Name, Values: append([]float64(nil), source.Values...)}
		for categoryIndex, category := range model.Categories {
			rows = append(rows, AccessibleRow{Series: source.Name, Category: category, Value: formatLineNumber(source.Values[categoryIndex])})
		}
	}
	controlOptions, exportOptions := chartControlConfig(options)
	component := line.Line(line.Config{
		Label:    model.Title,
		Title:    line.Title{Text: model.Title},
		Labels:   append([]string(nil), model.Categories...),
		Series:   series,
		Controls: controlOptions,
		Export:   exportOptions,
	})
	chartComponent := applyChartPrintPolicy(templ.Component(component), options)
	return WithAccessibleData(chartComponent, AccessibleData{Title: model.Title, Rows: rows}, AccessibleRenderPolicy{MaxOutputBytes: rc.EffectivePolicy.OutputBytes}), nil
}

func init() {
	registerFamilyHandler("line", func(rc margo.RenderContext, raw any, options chartRenderOptions) (templ.Component, error) {
		model, ok := raw.(lineModel)
		if !ok {
			return nil, chartDiagnostic("chart.model_invalid", "line handler received the wrong model")
		}
		return renderLineWithOptions(rc, model, options)
	})
}

func formatLineNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
