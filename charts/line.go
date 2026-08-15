package charts

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	sharedchart "github.com/araihu/goshtoso-charts/components/chart"
	interactiveline "github.com/araihu/goshtoso-charts/components/interactive/line"
	"github.com/araihu/goshtoso-charts/components/line"
	margo "github.com/araihu/margo"
)

type lineModel struct {
	SchemaVersion int               `yaml:"schemaVersion"`
	Type          string            `yaml:"type"`
	Renderer      string            `yaml:"renderer"`
	Title         string            `yaml:"title"`
	Style         chartStyleModel   `yaml:"style"`
	Categories    []string          `yaml:"categories"`
	Series        []lineSeriesModel `yaml:"series"`
}

type lineSeriesModel struct {
	Name            string    `yaml:"name"`
	Values          []float64 `yaml:"values"`
	chartPaintModel `yaml:",inline"`
}

func validateLineModel(model lineModel) error {
	if err := validateChartStyle(model.Style); err != nil {
		return err
	}
	if model.SchemaVersion != 1 || model.Type != "line" {
		return chartDiagnostic("chart.schema_invalid", "line model envelope is invalid")
	}
	if model.Renderer != "" && model.Renderer != "static" && model.Renderer != "interactive" {
		return chartDiagnostic("chart.renderer_invalid", "line renderer must be static or interactive")
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
		if err := validateChartPaint(series.chartPaintModel, fmt.Sprintf("line series %q", series.Name)); err != nil {
			return err
		}
		if model.Renderer == "interactive" && strings.TrimSpace(series.Class) != "" {
			return chartDiagnostic("chart.renderer_style_unsupported", "interactive line series do not support class")
		}
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
	if model.Renderer == "interactive" && !options.controlWrapper {
		return nil, chartDiagnostic("chart.renderer_controls_required", "interactive renderer requires the chart control wrapper in this proof of concept")
	}
	series := make([]line.Series, len(model.Series))
	paints := make([]chartPaintModel, len(model.Series))
	rows := make([]AccessibleRow, 0, len(model.Series)*len(model.Categories))
	for seriesIndex, source := range model.Series {
		paint := source.chartPaintModel.normalized()
		paints[seriesIndex] = paint
		series[seriesIndex] = line.Series{Name: source.Name, Values: append([]float64(nil), source.Values...), Color: paint.Color, Class: paint.Class}
		for categoryIndex, category := range model.Categories {
			rows = append(rows, AccessibleRow{Series: source.Name, Category: category, Value: formatLineNumber(source.Values[categoryIndex])})
		}
	}
	controlOptions, exportOptions := chartControlConfig(options)
	style := chartThemeForSeries(model.Style, paints)
	if model.Renderer == "interactive" {
		interactiveSeries := make([]interactiveline.Series, len(model.Series))
		for seriesIndex, source := range model.Series {
			data := make([]interactiveline.Data, len(source.Values))
			for valueIndex, value := range source.Values {
				data[valueIndex] = interactiveline.Data{Name: model.Categories[valueIndex], Value: value}
			}
			interactiveSeries[seriesIndex] = interactiveline.Series{Name: source.Name, Data: data}
		}
		component := interactiveline.Line(interactiveline.Config{
			Label: model.Title, Caption: Caption(model.Title), XAxis: append([]string(nil), model.Categories...),
			Series: interactiveSeries, Style: style,
			Options: sharedchart.ChartOptions{
				Title: &sharedchart.TitleOptions{Text: model.Title}, Animation: sharedchart.Bool(false), Controls: controlOptions, Export: exportOptions,
			},
		})
		chartComponent := applyChartPrintPolicy(templ.Component(component), options)
		return WithAccessibleData(chartComponent, AccessibleData{Title: model.Title, Rows: rows}, AccessibleRenderPolicy{MaxOutputBytes: rc.EffectivePolicy.OutputBytes}), nil
	}
	component := line.Line(line.Config{
		Label:    model.Title,
		Title:    line.Title{Text: model.Title},
		Labels:   append([]string(nil), model.Categories...),
		Series:   series,
		Controls: controlOptions,
		Export:   exportOptions,
		Style:    style,
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
