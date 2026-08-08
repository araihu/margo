package charts

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/araihu/goshtoso-charts/components/charttheme"
)

func TestChartStyleDefaultsToThemeTokensAndMapsOverrides(t *testing.T) {
	defaultStyle := chartStyleModel{}
	if got := defaultStyle.chartTheme().SeriesColor(0); got != "var(--color-chart-series-1)" {
		t.Fatalf("default series color = %q", got)
	}
	style := chartStyleModel{Palette: "pastel", Class: "custom-chart", Colors: []string{"#123456"}}
	theme := style.chartTheme()
	if theme.Palette != charttheme.PalettePastel {
		t.Fatalf("palette = %q", theme.Palette)
	}
	if got := theme.RootClasses("figure"); got != "figure goshtoso-charts-palette goshtoso-charts-palette-pastel custom-chart" {
		t.Fatalf("root classes = %q", got)
	}
	if got := theme.SeriesColor(0); got != "#123456" {
		t.Fatalf("explicit series color = %q", got)
	}
	if got := theme.SeriesColor(1); got != "var(--color-chart-series-2)" {
		t.Fatalf("token fallback = %q", got)
	}
	seriesTheme := chartThemeForSeries(chartStyleModel{Colors: []string{"#abcdef"}}, []chartPaintModel{{Color: "#fedcba"}})
	if got := seriesTheme.SeriesColor(0); got != "#fedcba" {
		t.Fatalf("series hex override = %q", got)
	}
}

func TestChartStyleValidationRejectsUnsafeOrUnsupportedOverrides(t *testing.T) {
	cases := []struct {
		name  string
		style chartStyleModel
		want  string
	}{
		{name: "palette", style: chartStyleModel{Palette: "future"}, want: "chart.style_palette_invalid"},
		{name: "class", style: chartStyleModel{Class: `custom;bad`}, want: "chart.style_class_invalid"},
		{name: "blank class", style: chartStyleModel{Class: "   "}, want: "chart.style_class_invalid"},
		{name: "color", style: chartStyleModel{Colors: []string{"red"}}, want: "chart.style_color_invalid"},
		{name: "count", style: chartStyleModel{Colors: make([]string, maxChartStyleColors+1)}, want: "chart.style_colors_invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateChartStyle(tc.style); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %s", err, tc.want)
			}
		})
	}
	if err := validateChartPaint(chartPaintModel{Class: "custom", Color: "#123456"}, "series"); err == nil || !strings.Contains(err.Error(), "chart.style_conflict") {
		t.Fatalf("paint conflict error = %v", err)
	}
}

func TestDecorateBarSeriesClassesTargetsChartPaintElements(t *testing.T) {
	component := templ.ComponentFunc(func(_ context.Context, out io.Writer) error {
		_, err := out.Write([]byte(`<figure><svg><path style="fill:var(--color-chart-series-1)"/></svg></figure>`))
		return err
	})
	decorated := decorateBarSeriesClasses(component, charttheme.Style{}, []chartPaintModel{{Class: "revenue-series"}})
	var output bytes.Buffer
	if err := decorated.Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `class="revenue-series"`) {
		t.Fatalf("decorated markup = %s", output.String())
	}
}
