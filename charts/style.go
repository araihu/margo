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
	maxChartStyleColors     = 12
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

func validateChartStyle(style chartStyleModel) error {
	switch style.Palette {
	case "", "auto", "araihu", "bold", "neutral", "pastel", "status":
	default:
		return chartDiagnostic("chart.style_palette_invalid", fmt.Sprintf("unsupported chart palette %q", style.Palette))
	}
	if err := validateChartClass(style.Class, "chart style"); err != nil {
		return err
	}
	if len(style.Colors) > maxChartStyleColors {
		return chartDiagnostic("chart.style_colors_invalid", fmt.Sprintf("chart style supports at most %d colors", maxChartStyleColors))
	}
	for index, color := range style.Colors {
		if color == "" {
			continue
		}
		if !isHexColor(color) {
			return chartDiagnostic("chart.style_color_invalid", fmt.Sprintf("chart style color %d must be a hex color", index))
		}
	}
	return nil
}

func validateChartPaint(paint chartPaintModel, subject string) error {
	paint = paint.normalized()
	if strings.TrimSpace(paint.Class) != "" && strings.TrimSpace(paint.Color) != "" {
		return chartDiagnostic("chart.style_conflict", fmt.Sprintf("%s cannot set both class and color", subject))
	}
	if err := validateChartClass(paint.Class, subject); err != nil {
		return err
	}
	if strings.TrimSpace(paint.Color) != "" && !isHexColor(paint.Color) {
		return chartDiagnostic("chart.style_color_invalid", fmt.Sprintf("%s color must be a hex color", subject))
	}
	return nil
}

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

func isHexColor(value string) bool {
	if value == "" || value[0] != '#' {
		return false
	}
	switch len(value) {
	case 4, 5, 7, 9: // #RGB, #RGBA, #RRGGBB, #RRGGBBAA
	default:
		return false
	}
	for _, r := range value[1:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
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
