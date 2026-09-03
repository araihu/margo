package charts

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/araihu/goshtoso-charts/components/bar"
	sharedchart "github.com/araihu/goshtoso-charts/components/chart"
	interactivebar "github.com/araihu/goshtoso-charts/components/interactive/bar"
	margo "github.com/araihu/margo"
)

type barModel struct {
	SchemaVersion int              `yaml:"schemaVersion"`
	Type          string           `yaml:"type"`
	Renderer      string           `yaml:"renderer"`
	Title         string           `yaml:"title"`
	Style         chartStyleModel  `yaml:"style"`
	Categories    []string         `yaml:"categories"`
	Orientation   string           `yaml:"orientation"`
	Series        []barSeriesModel `yaml:"series"`
}

type barSeriesModel struct {
	Name            string    `yaml:"name"`
	Values          []float64 `yaml:"values"`
	chartPaintModel `yaml:",inline"`
}

func validateBarModel(model barModel) error {
	if err := validateChartStyle(model.Style); err != nil {
		return err
	}
	if model.SchemaVersion != 1 || model.Type != "bar" {
		return chartDiagnostic("chart.schema_invalid", "bar model envelope is invalid")
	}
	if model.Renderer != "" && model.Renderer != "static" && model.Renderer != "interactive" {
		return chartDiagnostic("chart.renderer_invalid", "bar renderer must be static or interactive")
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
		if err := validateChartPaint(series.chartPaintModel, fmt.Sprintf("bar series %q", series.Name)); err != nil {
			return err
		}
		if model.Renderer == "interactive" && strings.TrimSpace(series.Class) != "" {
			return chartDiagnostic("chart.renderer_style_unsupported", "interactive bar series do not support class")
		}
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

func renderBarWithOptions(rc margo.RenderContext, model barModel, options chartRenderOptions) (templ.Component, error) {
	if err := validateBarModel(model); err != nil {
		return nil, err
	}
	if model.Renderer == "interactive" && !options.controlWrapper {
		return nil, interactiveRendererUnavailable(options)
	}
	orientation := bar.OrientationVertical
	if model.Orientation == "horizontal" {
		orientation = bar.OrientationHorizontal
	}
	series := make([]bar.Series, len(model.Series))
	paints := make([]chartPaintModel, len(model.Series))
	rows := make([]AccessibleRow, 0, len(model.Series)*len(model.Categories))
	for seriesIndex, source := range model.Series {
		paints[seriesIndex] = source.chartPaintModel.normalized()
		series[seriesIndex] = bar.Series{Name: source.Name, Values: append([]float64(nil), source.Values...)}
		for categoryIndex, category := range model.Categories {
			rows = append(rows, AccessibleRow{
				Series: source.Name, Category: category, Value: formatNumber(source.Values[categoryIndex]),
			})
		}
	}
	controlOptions, exportOptions := chartControlConfig(options)
	style := chartThemeForSeries(model.Style, paints)
	if model.Renderer == "interactive" {
		interactiveSeries := make([]interactivebar.Series, len(model.Series))
		for seriesIndex, source := range model.Series {
			data := make([]interactivebar.Data, len(source.Values))
			for valueIndex, value := range source.Values {
				data[valueIndex] = interactivebar.Data{Name: model.Categories[valueIndex], Value: value}
			}
			interactiveSeries[seriesIndex] = interactivebar.Series{Name: source.Name, Data: data}
		}
		interactiveOrientation := interactivebar.OrientationVertical
		if model.Orientation == "horizontal" {
			interactiveOrientation = interactivebar.OrientationHorizontal
		}
		component := interactivebar.Bar(interactivebar.Config{
			Label: model.Title, Caption: Caption(model.Title), XAxis: append([]string(nil), model.Categories...),
			Series: interactiveSeries, Orientation: interactiveOrientation, Style: style,
			Options: sharedchart.ChartOptions{
				Title: &sharedchart.TitleOptions{Text: model.Title}, Animation: sharedchart.Bool(false), Controls: controlOptions, Export: exportOptions,
			},
		})
		chartComponent := applyInteractiveBarLayout(templ.Component(component), model.Title)
		chartComponent = applyChartPrintPolicy(chartComponent, options)
		return WithAccessibleData(chartComponent, AccessibleData{Title: model.Title, Rows: rows}, AccessibleRenderPolicy{MaxOutputBytes: rc.EffectivePolicy.OutputBytes}), nil
	}
	labels := model.Categories
	if options.deckProjection && model.Orientation != "horizontal" {
		labels = compactStaticBarLabels(labels)
	}
	component := bar.Bar(bar.Config{
		Label:       model.Title,
		Caption:     Caption(model.Title),
		Title:       model.Title,
		Labels:      append([]string(nil), labels...),
		Series:      series,
		Orientation: orientation,
		Controls:    controlOptions,
		Export:      exportOptions,
		Style:       style,
	})
	chartComponent := decorateBarSeriesClasses(templ.Component(component), style, paints)
	chartComponent = applyChartPrintPolicy(chartComponent, options)
	return WithAccessibleData(chartComponent, AccessibleData{Title: model.Title, Rows: rows}, AccessibleRenderPolicy{MaxOutputBytes: rc.EffectivePolicy.OutputBytes}), nil
}

func init() {
	registerFamilyHandler("bar", func(rc margo.RenderContext, raw any, options chartRenderOptions) (templ.Component, error) {
		model, ok := raw.(barModel)
		if !ok {
			return nil, chartDiagnostic("chart.model_invalid", "bar handler received the wrong model")
		}
		return renderBarWithOptions(rc, model, options)
	})
}

func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
