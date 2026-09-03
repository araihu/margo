package charts

import (
	"fmt"
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

func validateLineSemantics(model lineModel) error {
	if err := validateChartClass(model.Style.Class, "chart style"); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(model.Series))
	for _, series := range model.Series {
		if err := validateChartClass(series.Class, fmt.Sprintf("line series %q", series.Name)); err != nil {
			return err
		}
		if model.Renderer == "interactive" && strings.TrimSpace(series.Class) != "" {
			return chartDiagnostic("chart.renderer_style_unsupported", "interactive line series do not support class")
		}
		if _, exists := seen[series.Name]; exists {
			return chartDiagnostic("chart.semantic_series_duplicate", "line series names must be unique")
		}
		seen[series.Name] = struct{}{}
		if len(series.Values) != len(model.Categories) {
			return chartDiagnostic("chart.semantic_alignment_invalid", "line series values must align with categories")
		}
	}
	return nil
}

func renderLine(rc margo.RenderContext, model lineModel) (templ.Component, error) {
	return renderLineWithOptions(rc, model, defaultChartRenderOptions)
}

func renderLineWithOptions(rc margo.RenderContext, model lineModel, options chartRenderOptions) (templ.Component, error) {
	if err := validateLineSemantics(model); err != nil {
		return nil, err
	}
	if model.Renderer == "interactive" && !options.controlWrapper {
		return nil, interactiveRendererUnavailable(options)
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
		chartComponent := applyInteractiveLineLayout(templ.Component(component), model.Title)
		chartComponent = applyChartPrintPolicy(chartComponent, options)
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
