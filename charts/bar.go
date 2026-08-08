package charts

import (
	"math"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/araihu/goshtoso-charts/components/bar"
	"github.com/araihu/goshtoso-charts/components/chartcontrol"
	margo "github.com/araihu/margo"
)

type barModel struct {
	SchemaVersion int              `yaml:"schemaVersion"`
	Type          string           `yaml:"type"`
	Title         string           `yaml:"title"`
	Categories    []string         `yaml:"categories"`
	Orientation   string           `yaml:"orientation"`
	Series        []barSeriesModel `yaml:"series"`
}

type barSeriesModel struct {
	Name   string    `yaml:"name"`
	Values []float64 `yaml:"values"`
}

func validateBarModel(model barModel) error {
	if model.SchemaVersion != 1 || model.Type != "bar" {
		return chartDiagnostic("chart.schema_invalid", "bar model envelope is invalid")
	}
	if strings.TrimSpace(model.Title) == "" {
		return chartDiagnostic("chart.semantic_title_invalid", "bar title is required")
	}
	if len(model.Categories) == 0 || len(model.Categories) > 4096 {
		return chartDiagnostic("chart.resource_categories_invalid", "bar chart requires 1 to 4096 categories")
	}
	seenCategories := make(map[string]struct{}, len(model.Categories))
	for index, category := range model.Categories {
		if strings.TrimSpace(category) == "" {
			return chartDiagnostic("chart.semantic_category_invalid", "bar category cannot be empty")
		}
		if _, exists := seenCategories[category]; exists {
			return chartDiagnostic("chart.semantic_category_duplicate", "bar categories must be unique")
		}
		seenCategories[category] = struct{}{}
		if index >= 4096 {
			return chartDiagnostic("chart.resource_categories_invalid", "bar chart has too many categories")
		}
	}
	if model.Orientation == "" {
		model.Orientation = "vertical"
	}
	if model.Orientation != "vertical" && model.Orientation != "horizontal" {
		return chartDiagnostic("chart.semantic_orientation_invalid", "bar orientation must be vertical or horizontal")
	}
	if len(model.Series) == 0 || len(model.Series) > 256 {
		return chartDiagnostic("chart.resource_series_invalid", "bar chart requires 1 to 256 series")
	}
	seenSeries := make(map[string]struct{}, len(model.Series))
	for _, series := range model.Series {
		if strings.TrimSpace(series.Name) == "" {
			return chartDiagnostic("chart.semantic_series_invalid", "bar series name is required")
		}
		if _, exists := seenSeries[series.Name]; exists {
			return chartDiagnostic("chart.semantic_series_duplicate", "bar series names must be unique")
		}
		seenSeries[series.Name] = struct{}{}
		if len(series.Values) != len(model.Categories) {
			return chartDiagnostic("chart.semantic_alignment_invalid", "bar series values must align with categories")
		}
		for _, value := range series.Values {
			if math.IsNaN(value) || math.IsInf(value, 0) {
				return chartDiagnostic("chart.semantic_value_invalid", "bar values must be finite")
			}
		}
	}
	return nil
}

func renderBar(rc margo.RenderContext, model barModel) (templ.Component, error) {
	if err := validateBarModel(model); err != nil {
		return nil, err
	}
	orientation := bar.OrientationVertical
	if model.Orientation == "horizontal" {
		orientation = bar.OrientationHorizontal
	}
	series := make([]bar.Series, len(model.Series))
	rows := make([]AccessibleRow, 0, len(model.Series)*len(model.Categories))
	for seriesIndex, source := range model.Series {
		series[seriesIndex] = bar.Series{Name: source.Name, Values: append([]float64(nil), source.Values...)}
		for categoryIndex, category := range model.Categories {
			rows = append(rows, AccessibleRow{
				Series: source.Name, Category: category, Value: formatNumber(source.Values[categoryIndex]),
			})
		}
	}
	component := bar.Bar(bar.Config{
		Label:       model.Title,
		Caption:     Caption(model.Title),
		Title:       model.Title,
		Labels:      append([]string(nil), model.Categories...),
		Series:      series,
		Orientation: orientation,
		Controls:    chartcontrol.Options{Mode: chartcontrol.WrapperModeOmitted},
		Export:      &chartcontrol.ExportOptions{Disabled: true},
	})
	return WithAccessibleData(component, AccessibleData{Title: model.Title, Rows: rows}, AccessibleRenderPolicy{MaxOutputBytes: rc.EffectivePolicy.OutputBytes}), nil
}

func init() {
	registerFamilyHandler("bar", func(rc margo.RenderContext, raw any) (templ.Component, error) {
		model, ok := raw.(barModel)
		if !ok {
			return nil, chartDiagnostic("chart.model_invalid", "bar handler received the wrong model")
		}
		return renderBar(rc, model)
	})
}

func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
