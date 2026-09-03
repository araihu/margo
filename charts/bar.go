package charts

import (
	"fmt"
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

func validateBarSemantics(model barModel) error {
	if err := validateChartClass(model.Style.Class, "chart style"); err != nil {
		return err
	}
	seenSeries := make(map[string]struct{}, len(model.Series))
	for _, series := range model.Series {
		if err := validateChartClass(series.Class, fmt.Sprintf("bar series %q", series.Name)); err != nil {
			return err
		}
		if model.Renderer == "interactive" && strings.TrimSpace(series.Class) != "" {
			return chartDiagnostic("chart.renderer_style_unsupported", "interactive bar series do not support class")
		}
		if _, exists := seenSeries[series.Name]; exists {
			return chartDiagnostic("chart.semantic_series_duplicate", "bar series names must be unique")
		}
		seenSeries[series.Name] = struct{}{}
		if len(series.Values) != len(model.Categories) {
			return chartDiagnostic("chart.semantic_alignment_invalid", "bar series values must align with categories")
		}
	}
	return nil
}

func renderBarWithOptions(rc margo.RenderContext, model barModel, options chartRenderOptions) (templ.Component, error) {
	if err := validateBarSemantics(model); err != nil {
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
