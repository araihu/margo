package charts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/a-h/templ"
)

const (
	// Chart titles are painted into a canvas, so CSS cannot make a long title
	// wrap. This line length is deliberately conservative for both the default
	// desktop chart width and the 390px narrow contract.
	interactiveBarTitleLineRunes = 28

	interactiveBarLayoutScriptMarker = "let goecharts_"
)

// wrapChartLabel inserts word-boundary breaks for text painted by ECharts.
// It preserves the source value in the figure aria-label, caption, and Margo's
// exact data table; the returned value is only used by the visual canvas.
func wrapChartLabel(value string, limit int) string {
	if limit < 1 || utf8.RuneCountInString(value) <= limit {
		return value
	}

	var result strings.Builder
	lineRunes := 0
	for _, word := range strings.Fields(value) {
		wordRunes := utf8.RuneCountInString(word)
		if lineRunes > 0 && lineRunes+1+wordRunes <= limit {
			result.WriteByte(' ')
			lineRunes++
		} else if lineRunes > 0 {
			result.WriteByte('\n')
			lineRunes = 0
		}

		for _, character := range word {
			if lineRunes == limit {
				result.WriteByte('\n')
				lineRunes = 0
			}
			result.WriteRune(character)
			lineRunes++
		}
	}
	return result.String()
}

// applyInteractiveBarLayout adds the small amount of renderer-specific layout
// that the upstream renderer-neutral API intentionally does not expose. The
// option mutation happens before ECharts paints and again on resize, which
// keeps PNG print exports and live HTML on the same layout contract.
func applyInteractiveBarLayout(chart templ.Component, title string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		var rendered bytes.Buffer
		if err := chart.Render(ctx, &rendered); err != nil {
			return err
		}
		markup := rewriteInteractiveBarLayoutScript(rendered.String(), title)
		_, err := io.WriteString(out, markup)
		return err
	})
}

func rewriteInteractiveBarLayoutScript(markup, title string) string {
	initStart := strings.Index(markup, interactiveBarLayoutScriptMarker)
	if initStart < 0 {
		return markup
	}
	initEnd := strings.Index(markup[initStart:], ";")
	if initEnd < 0 {
		return markup
	}
	initEnd += initStart
	chartDeclaration := markup[initStart:initEnd]
	chartNameEnd := strings.Index(chartDeclaration, " = echarts.init")
	if chartNameEnd <= len("let ") {
		return markup
	}
	chartName := strings.TrimSpace(chartDeclaration[len("let "):chartNameEnd])
	if chartName == "" {
		return markup
	}
	layoutFunctionName := "margoResponsiveBarLayout_" + strings.TrimPrefix(chartName, "goecharts_")

	optionStart := strings.Index(markup[initEnd+1:], "let option_")
	if optionStart < 0 {
		return markup
	}
	optionStart += initEnd + 1
	optionEnd := strings.Index(markup[optionStart:], " = ")
	if optionEnd < 0 {
		return markup
	}
	optionEnd += optionStart
	optionName := strings.TrimSpace(markup[optionStart+len("let ") : optionEnd])
	if optionName == "" {
		return markup
	}

	setMarker := chartName + ".setOption(" + optionName + ");"
	setIndex := strings.Index(markup[optionEnd:], setMarker)
	if setIndex < 0 {
		return markup
	}
	setIndex += optionEnd

	quotedTitle, err := json.Marshal(wrapChartLabel(title, interactiveBarTitleLineRunes))
	if err != nil {
		return markup
	}
	layout := fmt.Sprintf(`
    %s.title = Object.assign({}, %s.title || {}, { text: %s });
	    %s.legend = Object.assign({}, %s.legend || {}, {
      top: 85,
      textStyle: Object.assign({}, (%s.legend && %s.legend.textStyle) || {}, { width: 250, overflow: "breakAll" })
    });
    %s.grid = Object.assign({}, %s.grid || {}, { top: 180, bottom: 70, containLabel: true });
    %s.setOption(%s);
    const %s = () => {
      const narrow = %s.getWidth() <= 480;
      %s.setOption({
        legend: { top: narrow ? 105 : 85, textStyle: { width: narrow ? 210 : 250, overflow: "breakAll" } },
        grid: { top: narrow ? 245 : 180, bottom: 70, containLabel: true }
      }, { notMerge: false, lazyUpdate: false, silent: true });
    };
    %s();
    window.addEventListener("resize", %s);
`, optionName, optionName, quotedTitle, optionName, optionName, optionName, optionName, optionName, optionName, chartName, optionName, layoutFunctionName, chartName, chartName, layoutFunctionName, layoutFunctionName)

	return markup[:setIndex] + layout + markup[setIndex:]
}
