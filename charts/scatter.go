package charts

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	sharedchart "github.com/araihu/goshtoso-charts/components/chart"
	interactivescatter "github.com/araihu/goshtoso-charts/components/interactive/scatter"
	staticscatter "github.com/araihu/goshtoso-charts/components/scatter"
	margo "github.com/araihu/margo"
)

const (
	maxScatterSamples = 65536
)

type scatterModel struct {
	SchemaVersion int                  `yaml:"schemaVersion"`
	Type          string               `yaml:"type"`
	Renderer      string               `yaml:"renderer"`
	Title         string               `yaml:"title"`
	Style         chartStyleModel      `yaml:"style"`
	Categories    []string             `yaml:"categories"`
	Series        []scatterSeriesModel `yaml:"series"`
}

type scatterSeriesModel struct {
	Name            string              `yaml:"name"`
	Points          []scatterPointModel `yaml:"points"`
	Values          [][]float64         `yaml:"values"`
	Samples         [][]string          `yaml:"samples"`
	chartPaintModel `yaml:",inline"`
}

type scatterPointModel struct {
	Category string  `yaml:"category"`
	Value    float64 `yaml:"value"`
}

func validateScatterSemantics(model scatterModel) error {
	if err := validateChartClass(model.Style.Class, "chart style"); err != nil {
		return err
	}
	categorySet := make(map[string]struct{}, len(model.Categories))
	for _, category := range model.Categories {
		categorySet[category] = struct{}{}
	}
	seenSeries := make(map[string]struct{}, len(model.Series))
	totalSamples := 0
	for _, series := range model.Series {
		if err := validateChartClass(series.Class, fmt.Sprintf("scatter series %q", series.Name)); err != nil {
			return err
		}
		if model.Renderer == "interactive" && strings.TrimSpace(series.Class) != "" {
			return chartDiagnostic("chart.renderer_style_unsupported", "interactive scatter series do not support class")
		}
		if _, exists := seenSeries[series.Name]; exists {
			return chartDiagnostic("chart.semantic_series_duplicate", "scatter series names must be unique")
		}
		seenSeries[series.Name] = struct{}{}
		if series.Points != nil {
			for _, point := range series.Points {
				if _, exists := categorySet[point.Category]; !exists {
					return chartDiagnostic("chart.scatter.category_unknown", "scatter point references an unknown category")
				}
			}
			totalSamples += len(series.Points)
			if totalSamples > maxScatterSamples {
				return chartDiagnostic("chart.resource_points_invalid", "scatter point count exceeds 65536")
			}
			continue
		}
		if len(series.Values) != len(model.Categories) {
			return chartDiagnostic("chart.semantic_alignment_invalid", "scatter aligned values must match categories")
		}
		if series.Samples != nil && len(series.Samples) != len(series.Values) {
			return chartDiagnostic("chart.semantic_alignment_invalid", "scatter samples must match categories")
		}
		seriesSamples := 0
		for categoryIndex, values := range series.Values {
			if series.Samples != nil && len(series.Samples[categoryIndex]) != len(values) {
				return chartDiagnostic("chart.semantic_alignment_invalid", "scatter samples must align with values")
			}
			seriesSamples += len(values)
			if totalSamples+seriesSamples > maxScatterSamples {
				return chartDiagnostic("chart.resource_points_invalid", "scatter sample count exceeds 65536")
			}
		}
		totalSamples += seriesSamples
	}
	if model.Renderer == "interactive" {
		for _, series := range model.Series {
			if series.Points != nil {
				counts := make(map[string]int, len(model.Categories))
				for _, point := range series.Points {
					counts[point.Category]++
				}
				for _, category := range model.Categories {
					if counts[category] != 1 {
						return chartDiagnostic("chart.renderer_data_unsupported", "interactive scatter requires exactly one point per category")
					}
				}
				continue
			}
			for _, values := range series.Values {
				if len(values) != 1 {
					return chartDiagnostic("chart.renderer_data_unsupported", "interactive scatter requires exactly one value per category")
				}
			}
		}
	}
	return nil
}

func renderScatter(rc margo.RenderContext, model scatterModel) (templ.Component, error) {
	return renderScatterWithOptions(rc, model, defaultChartRenderOptions)
}

func renderScatterWithOptions(rc margo.RenderContext, model scatterModel, options chartRenderOptions) (templ.Component, error) {
	if err := validateScatterSemantics(model); err != nil {
		return nil, err
	}
	if model.Renderer == "interactive" && !options.controlWrapper {
		return nil, interactiveRendererUnavailable(options)
	}
	series := make([]staticscatter.Series, len(model.Series))
	paints := make([]chartPaintModel, len(model.Series))
	rows := make([]AccessibleRow, 0)
	for seriesIndex, source := range model.Series {
		paint := source.chartPaintModel.normalized()
		paints[seriesIndex] = paint
		series[seriesIndex] = staticscatter.Series{Name: source.Name, Color: paint.Color, Class: paint.Class}
		if source.Points != nil {
			series[seriesIndex].Points = make([]staticscatter.Point, len(source.Points))
			for pointIndex, point := range source.Points {
				series[seriesIndex].Points[pointIndex] = staticscatter.Point{Category: point.Category, Value: point.Value}
				if model.Renderer != "interactive" {
					rows = append(rows, AccessibleRow{Series: source.Name, Category: point.Category, Value: formatScatterNumber(point.Value)})
				}
			}
			if model.Renderer == "interactive" {
				valuesByCategory := make(map[string]float64, len(source.Points))
				for _, point := range source.Points {
					valuesByCategory[point.Category] = point.Value
				}
				for _, category := range model.Categories {
					rows = append(rows, AccessibleRow{Series: source.Name, Category: category, Value: formatScatterNumber(valuesByCategory[category])})
				}
			}
			continue
		}
		series[seriesIndex].Values = make([][]float64, len(source.Values))
		for categoryIndex, values := range source.Values {
			series[seriesIndex].Values[categoryIndex] = append([]float64(nil), values...)
			for sampleIndex, value := range values {
				sample := ""
				if source.Samples != nil {
					sample = source.Samples[categoryIndex][sampleIndex]
				} else if len(values) > 1 {
					sample = strconv.Itoa(sampleIndex)
				}
				rows = append(rows, AccessibleRow{Series: source.Name, Category: model.Categories[categoryIndex], Sample: sample, Value: formatScatterNumber(value)})
			}
		}
	}
	controlOptions, exportOptions := chartControlConfig(options)
	style := chartThemeForSeries(model.Style, paints)
	if model.Renderer == "interactive" {
		interactiveSeries := make([]interactivescatter.Series, len(model.Series))
		for seriesIndex, source := range model.Series {
			data := make([]interactivescatter.Data, len(model.Categories))
			if source.Points != nil {
				valuesByCategory := make(map[string]float64, len(source.Points))
				for _, point := range source.Points {
					valuesByCategory[point.Category] = point.Value
				}
				for categoryIndex, category := range model.Categories {
					data[categoryIndex] = interactivescatter.Data{Name: category, Value: valuesByCategory[category]}
				}
			} else {
				for categoryIndex, category := range model.Categories {
					data[categoryIndex] = interactivescatter.Data{Name: category, Value: source.Values[categoryIndex][0]}
				}
			}
			interactiveSeries[seriesIndex] = interactivescatter.Series{Name: source.Name, Data: data}
		}
		component := interactivescatter.Scatter(interactivescatter.Config{
			Label: model.Title, Caption: Caption(model.Title), XAxis: append([]string(nil), model.Categories...),
			XAxisType: interactivescatter.AxisCategory, Series: interactiveSeries, Style: style,
			Options: sharedchart.ChartOptions{
				Title: &sharedchart.TitleOptions{Text: model.Title}, Animation: sharedchart.Bool(false), Controls: controlOptions, Export: exportOptions,
			},
		})
		chartComponent := applyChartPrintPolicy(templ.Component(component), options)
		return WithAccessibleData(chartComponent, AccessibleData{Title: model.Title, Rows: rows}, AccessibleRenderPolicy{MaxOutputBytes: rc.EffectivePolicy.OutputBytes}), nil
	}
	component := staticscatter.Scatter(staticscatter.Config{
		Label:      model.Title,
		Caption:    Caption(model.Title),
		Categories: append([]string(nil), model.Categories...),
		Series:     series,
		Options:    staticscatter.Options{TopNLabels: staticscatter.TopNLabels{Count: 0}},
		Controls:   controlOptions,
		Export:     exportOptions,
		Style:      style,
	})
	chartComponent := applyChartPrintPolicy(templ.Component(component), options)
	return WithAccessibleData(chartComponent, AccessibleData{Title: model.Title, Rows: rows}, AccessibleRenderPolicy{MaxOutputBytes: rc.EffectivePolicy.OutputBytes}), nil
}

func init() {
	registerFamilyHandler("scatter", func(rc margo.RenderContext, raw any, options chartRenderOptions) (templ.Component, error) {
		model, ok := raw.(scatterModel)
		if !ok {
			return nil, chartDiagnostic("chart.model_invalid", "scatter handler received the wrong model")
		}
		return renderScatterWithOptions(rc, model, options)
	})
}

func formatScatterNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
