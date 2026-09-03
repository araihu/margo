package charts

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"regexp"
	"strings"
	"unicode"

	"github.com/a-h/templ"
	"github.com/araihu/goshtoso-charts/components/charttheme"
)

const (
	maxChartStyleClassBytes = 256
)

// chartStyleModel is the closed, renderer-neutral appearance contract exposed
// by each v1 chart schema. An omitted style follows Goshtoso theme tokens.
type chartStyleModel struct {
	Palette string   `yaml:"palette"`
	Class   string   `yaml:"class"`
	Colors  []string `yaml:"colors"`
}

// chartPaintModel permits a local class or explicit hex color for one series or
// slice. Class and Color are mutually exclusive at this level, matching the
// upstream static component contract.
type chartPaintModel struct {
	Class string `yaml:"class"`
	Color string `yaml:"color"`
}

func (paint chartPaintModel) normalized() chartPaintModel {
	paint.Class = strings.TrimSpace(paint.Class)
	paint.Color = strings.TrimSpace(paint.Color)
	return paint
}

var chartPaintElementPattern = regexp.MustCompile(`<(?:path|circle|rect|line|polyline|polygon|text)\b[^>]*>`)

// validateChartClass adds byte and control-character checks that JSON Schema
// cannot express portably and protects internal callers that construct models.
func validateChartClass(value, subject string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		if value != "" {
			return chartDiagnostic("chart.style_class_invalid", fmt.Sprintf("%s class cannot be blank", subject))
		}
		return nil
	}
	if len(trimmed) > maxChartStyleClassBytes {
		return chartDiagnostic("chart.style_class_invalid", fmt.Sprintf("%s class is too long", subject))
	}
	if strings.ContainsAny(trimmed, `"'<>;`) {
		return chartDiagnostic("chart.style_class_invalid", fmt.Sprintf("%s class is unsafe", subject))
	}
	for _, r := range trimmed {
		if unicode.IsControl(r) {
			return chartDiagnostic("chart.style_class_invalid", fmt.Sprintf("%s class contains a control character", subject))
		}
	}
	return nil
}

func (style chartStyleModel) chartTheme() charttheme.Style {
	palette := charttheme.PaletteAuto
	switch style.Palette {
	case "araihu":
		palette = charttheme.PaletteAraiHu
	case "bold":
		palette = charttheme.PaletteBold
	case "neutral":
		palette = charttheme.PaletteNeutral
	case "pastel":
		palette = charttheme.PalettePastel
	case "status":
		palette = charttheme.PaletteStatus
	}
	return charttheme.Style{
		Palette: palette,
		Class:   strings.TrimSpace(style.Class),
		Colors:  append([]string(nil), style.Colors...),
	}
}

func chartThemeForSeries(style chartStyleModel, paints []chartPaintModel) charttheme.Style {
	theme := style.chartTheme()
	colors := append([]string(nil), theme.Colors...)
	for index, paint := range paints {
		paint = paint.normalized()
		if strings.TrimSpace(paint.Color) == "" {
			continue
		}
		for len(colors) <= index {
			colors = append(colors, "")
		}
		colors[index] = paint.Color
	}
	theme.Colors = colors
	return theme
}

// decorateBarSeriesClasses adds local classes to the SVG elements carrying a
// bar series color. The upstream bar API exposes root style classes and colors
// but not per-series classes; the decoration preserves the same token contract
// as the line, pie, and scatter adapters.
func decorateBarSeriesClasses(chart templ.Component, style charttheme.Style, paints []chartPaintModel) templ.Component {
	hasClass := false
	for _, paint := range paints {
		if strings.TrimSpace(paint.Class) != "" {
			hasClass = true
			break
		}
	}
	if !hasClass {
		return chart
	}
	return templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		var rendered bytes.Buffer
		if err := chart.Render(ctx, &rendered); err != nil {
			return err
		}
		markup := rendered.String()
		for index, paint := range paints {
			class := strings.TrimSpace(paint.Class)
			if class == "" {
				continue
			}
			color := html.EscapeString(style.SeriesColor(index))
			markup = addClassToChartPaintElements(markup, color, class)
		}
		_, err := io.WriteString(out, markup)
		return err
	})
}

func addClassToChartPaintElements(markup, color, class string) string {
	class = html.EscapeString(strings.TrimSpace(class))
	if class == "" || color == "" {
		return markup
	}
	return chartPaintElementPattern.ReplaceAllStringFunc(markup, func(element string) string {
		if !strings.Contains(element, "fill:"+color) && !strings.Contains(element, "stroke:"+color) {
			return element
		}
		if strings.Contains(element, ` class="`) {
			return strings.Replace(element, ` class="`, ` class="`+class+` `, 1)
		}
		return strings.Replace(element, " ", ` class="`+class+`" `, 1)
	})
}
