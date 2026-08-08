package charts

import (
	"math"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/araihu/goshtoso-charts/components/scatter"
	margo "github.com/araihu/margo"
)

const (
	maxScatterCategories = 4096
	maxScatterSeries     = 256
	maxScatterSamples    = 65536
)

type scatterModel struct {
	SchemaVersion int                  `yaml:"schemaVersion"`
	Type          string               `yaml:"type"`
	Title         string               `yaml:"title"`
	Categories    []string             `yaml:"categories"`
	Series        []scatterSeriesModel `yaml:"series"`
}

type scatterSeriesModel struct {
	Name    string              `yaml:"name"`
	Points  []scatterPointModel `yaml:"points"`
	Values  [][]float64         `yaml:"values"`
	Samples [][]string          `yaml:"samples"`
}

type scatterPointModel struct {
	Category string  `yaml:"category"`
	Value    float64 `yaml:"value"`
}

func validateScatterModel(model scatterModel) error {
	if model.SchemaVersion != 1 || model.Type != "scatter" {
		return chartDiagnostic("chart.schema_invalid", "scatter model envelope is invalid")
	}
	if strings.TrimSpace(model.Title) == "" {
		return chartDiagnostic("chart.semantic_title_invalid", "scatter title is required")
	}
	if len(model.Categories) == 0 || len(model.Categories) > maxScatterCategories {
		return chartDiagnostic("chart.resource_categories_invalid", "scatter chart requires 1 to 4096 categories")
	}
	categorySet := make(map[string]struct{}, len(model.Categories))
	for _, category := range model.Categories {
		if strings.TrimSpace(category) == "" {
			return chartDiagnostic("chart.semantic_category_invalid", "scatter category cannot be empty")
		}
		if _, exists := categorySet[category]; exists {
			return chartDiagnostic("chart.semantic_category_duplicate", "scatter categories must be unique")
		}
		categorySet[category] = struct{}{}
	}
	if len(model.Series) == 0 || len(model.Series) > maxScatterSeries {
		return chartDiagnostic("chart.resource_series_invalid", "scatter chart requires 1 to 256 series")
	}
	seenSeries := make(map[string]struct{}, len(model.Series))
	totalSamples := 0
	for _, series := range model.Series {
		if strings.TrimSpace(series.Name) == "" {
			return chartDiagnostic("chart.semantic_series_invalid", "scatter series name is required")
		}
		if _, exists := seenSeries[series.Name]; exists {
			return chartDiagnostic("chart.semantic_series_duplicate", "scatter series names must be unique")
		}
		seenSeries[series.Name] = struct{}{}
		hasPoints := series.Points != nil
		hasValues := series.Values != nil
		if hasPoints == hasValues {
			return chartDiagnostic("chart.scatter_data_mode_invalid", "scatter series requires exactly one of points or values")
		}
		if hasPoints {
			if series.Samples != nil {
				return chartDiagnostic("chart.scatter_samples_invalid", "scatter point series cannot declare samples")
			}
			if len(series.Points) == 0 {
				return chartDiagnostic("chart.resource_points_invalid", "scatter point series requires at least one point")
			}
			for pointIndex, point := range series.Points {
				if _, exists := categorySet[point.Category]; !exists {
					return chartDiagnostic("chart.scatter.category_unknown", "scatter point references an unknown category")
				}
				if math.IsNaN(point.Value) || math.IsInf(point.Value, 0) {
					return chartDiagnostic("chart.value_non_finite", "scatter point values must be finite")
				}
				if pointIndex >= maxScatterSamples {
					return chartDiagnostic("chart.resource_points_invalid", "scatter point count exceeds 65536")
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
			seenSamples := make(map[string]struct{}, len(values))
			for sampleIndex, value := range values {
				if math.IsNaN(value) || math.IsInf(value, 0) {
					return chartDiagnostic("chart.value_non_finite", "scatter aligned values must be finite")
				}
				if series.Samples != nil {
					sample := series.Samples[categoryIndex][sampleIndex]
					if strings.TrimSpace(sample) == "" {
						return chartDiagnostic("chart.semantic_sample_invalid", "scatter sample name is required")
					}
					if _, exists := seenSamples[sample]; exists {
						return chartDiagnostic("chart.semantic_sample_duplicate", "scatter sample names must be unique within a category")
					}
					seenSamples[sample] = struct{}{}
				}
				seriesSamples++
				if totalSamples+seriesSamples > maxScatterSamples {
					return chartDiagnostic("chart.resource_points_invalid", "scatter sample count exceeds 65536")
				}
			}
		}
		if seriesSamples == 0 {
			return chartDiagnostic("chart.resource_points_invalid", "scatter aligned series requires at least one value")
		}
		totalSamples += seriesSamples
	}
	return nil
}

func renderScatter(rc margo.RenderContext, model scatterModel) (templ.Component, error) {
	return renderScatterWithOptions(rc, model, defaultChartRenderOptions)
}

func renderScatterWithOptions(rc margo.RenderContext, model scatterModel, options chartRenderOptions) (templ.Component, error) {
	if err := validateScatterModel(model); err != nil {
		return nil, err
	}
	series := make([]scatter.Series, len(model.Series))
	rows := make([]AccessibleRow, 0)
	for seriesIndex, source := range model.Series {
		series[seriesIndex] = scatter.Series{Name: source.Name}
		if source.Points != nil {
			series[seriesIndex].Points = make([]scatter.Point, len(source.Points))
			for pointIndex, point := range source.Points {
				series[seriesIndex].Points[pointIndex] = scatter.Point{Category: point.Category, Value: point.Value}
				rows = append(rows, AccessibleRow{Series: source.Name, Category: point.Category, Value: formatScatterNumber(point.Value)})
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
	component := scatter.Scatter(scatter.Config{
		Label:      model.Title,
		Caption:    Caption(model.Title),
		Categories: append([]string(nil), model.Categories...),
		Series:     series,
		Options:    scatter.Options{TopNLabels: scatter.TopNLabels{Count: 0}},
		Controls:   controlOptions,
		Export:     exportOptions,
	})
	return WithAccessibleData(component, AccessibleData{Title: model.Title, Rows: rows}, AccessibleRenderPolicy{MaxOutputBytes: rc.EffectivePolicy.OutputBytes}), nil
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
