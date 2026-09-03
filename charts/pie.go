package charts

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	sharedchart "github.com/araihu/goshtoso-charts/components/chart"
	interactivepie "github.com/araihu/goshtoso-charts/components/interactive/pie"
	staticpie "github.com/araihu/goshtoso-charts/components/pie"
	margo "github.com/araihu/margo"
)

type pieModel struct {
	SchemaVersion int             `yaml:"schemaVersion"`
	Type          string          `yaml:"type"`
	Renderer      string          `yaml:"renderer"`
	Title         string          `yaml:"title"`
	Style         chartStyleModel `yaml:"style"`
	Slices        []pieSliceModel `yaml:"slices"`
}

type pieSliceModel struct {
	Name            string  `yaml:"name"`
	Value           float64 `yaml:"value"`
	chartPaintModel `yaml:",inline"`
}

func validatePieSemantics(model pieModel) error {
	if err := validateChartClass(model.Style.Class, "chart style"); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(model.Slices))
	for _, slice := range model.Slices {
		if err := validateChartClass(slice.Class, fmt.Sprintf("pie slice %q", slice.Name)); err != nil {
			return err
		}
		if model.Renderer == "interactive" && strings.TrimSpace(slice.Class) != "" {
			return chartDiagnostic("chart.renderer_style_unsupported", "interactive pie slices do not support class")
		}
		if _, exists := seen[slice.Name]; exists {
			return chartDiagnostic("chart.semantic_slice_duplicate", "pie slice names must be unique")
		}
		seen[slice.Name] = struct{}{}
	}
	return nil
}

func renderPie(rc margo.RenderContext, model pieModel) (templ.Component, error) {
	return renderPieWithOptions(rc, model, defaultChartRenderOptions)
}

func renderPieWithOptions(rc margo.RenderContext, model pieModel, options chartRenderOptions) (templ.Component, error) {
	if err := validatePieSemantics(model); err != nil {
		return nil, err
	}
	if model.Renderer == "interactive" && !options.controlWrapper {
		return nil, interactiveRendererUnavailable(options)
	}
	variant := staticpie.VariantPie
	if model.Type == "doughnut" {
		variant = staticpie.VariantDoughnut
	}
	slices := make([]staticpie.Slice, len(model.Slices))
	paints := make([]chartPaintModel, len(model.Slices))
	rows := make([]AccessibleRow, 0, len(model.Slices))
	for index, source := range model.Slices {
		paint := source.chartPaintModel.normalized()
		paints[index] = paint
		slices[index] = staticpie.Slice{Name: source.Name, Value: source.Value, Color: paint.Color, Class: paint.Class}
		rows = append(rows, AccessibleRow{Category: source.Name, Value: strconv.FormatFloat(source.Value, 'f', -1, 64)})
	}
	controlOptions, exportOptions := chartControlConfig(options)
	style := chartThemeForSeries(model.Style, paints)
	if model.Renderer == "interactive" {
		data := make([]interactivepie.Data, len(model.Slices))
		for index, source := range model.Slices {
			data[index] = interactivepie.Data{Name: source.Name, Value: source.Value}
		}
		innerRadius := 0.0
		if model.Type == "doughnut" {
			innerRadius = 40
		}
		component := interactivepie.Pie(interactivepie.Config{
			Label: model.Title, Caption: Caption(model.Title), Style: style,
			Series: []interactivepie.Series{{Name: model.Title, InnerRadius: innerRadius, Data: data}},
			Options: sharedchart.ChartOptions{
				Title: &sharedchart.TitleOptions{Text: model.Title}, Animation: sharedchart.Bool(false), Controls: controlOptions, Export: exportOptions,
			},
		})
		chartComponent := applyChartPrintPolicy(templ.Component(component), options)
		return WithAccessibleData(chartComponent, AccessibleData{Title: model.Title, Rows: rows}, AccessibleRenderPolicy{MaxOutputBytes: rc.EffectivePolicy.OutputBytes}), nil
	}
	component := staticpie.Pie(staticpie.Config{
		Label:    model.Title,
		Slices:   slices,
		Variant:  variant,
		Controls: controlOptions,
		Export:   exportOptions,
		Style:    style,
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
