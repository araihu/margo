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
	interactiveChartTitleLineRunes = 28
	// Keep the bar-specific name as an internal compatibility alias for the
	// existing layout tests and callers in this package.
	interactiveBarTitleLineRunes = interactiveChartTitleLineRunes

	interactiveChartLayoutScriptMarker = "let goecharts_"
	interactiveBarLayoutScriptMarker   = interactiveChartLayoutScriptMarker
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
	return applyInteractiveCartesianLayout(chart, title, "Bar")
}

// applyInteractiveLineLayout keeps the title and legend in separate rows for
// interactive line charts. The line renderer shares ECharts' Cartesian title
// and legend behavior with bars, but is otherwise left untouched.
func applyInteractiveLineLayout(chart templ.Component, title string) templ.Component {
	return applyInteractiveCartesianLayout(chart, title, "Line")
}

func applyInteractiveCartesianLayout(chart templ.Component, title, family string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, out io.Writer) error {
		var rendered bytes.Buffer
		if err := chart.Render(ctx, &rendered); err != nil {
			return err
		}
		markup := rewriteInteractiveCartesianLayoutScript(rendered.String(), title, family)
		_, err := io.WriteString(out, markup)
		return err
	})
}

func rewriteInteractiveBarLayoutScript(markup, title string) string {
	return rewriteInteractiveCartesianLayoutScript(markup, title, "Bar")
}

func rewriteInteractiveLineLayoutScript(markup, title string) string {
	return rewriteInteractiveCartesianLayoutScript(markup, title, "Line")
}

func rewriteInteractiveCartesianLayoutScript(markup, title, family string) string {
	initStart := strings.Index(markup, interactiveChartLayoutScriptMarker)
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

	quotedTitle, err := json.Marshal(wrapChartLabel(title, interactiveChartTitleLineRunes))
	if err != nil {
		return markup
	}
	layoutFunctionName := "margoResponsive" + family + "Layout_" + strings.TrimPrefix(chartName, "goecharts_")
	layoutBeforeSetOption := fmt.Sprintf(`
    %s.title = Object.assign({}, %s.title || {}, { text: %s });
	    %s.legend = Object.assign({}, %s.legend || {}, {
      top: 85,
      textStyle: Object.assign({}, (%s.legend && %s.legend.textStyle) || {}, { width: 250, overflow: "breakAll" })
    });
	    %s.grid = Object.assign({}, %s.grid || {}, { top: 180, bottom: 70, containLabel: true });
    const %s = () => {
      const narrow = %s.getWidth() <= 480;
	      const titleText = String((%s.title && %s.title.text) || "");
	      const titleLines = Math.max(1, titleText.split("\n").length);
	      const baseLegendTop = narrow ? 105 : 85;
	      const legendTop = Math.max(baseLegendTop, baseLegendTop + Math.max(0, titleLines - 2) * 22);
	      const baseGridTop = narrow ? 245 : 180;
	      const gridTop = Math.max(baseGridTop, legendTop + (narrow ? 140 : 95));
	      %s.setOption({
        legend: { top: legendTop, textStyle: { width: narrow ? 210 : 250, overflow: "breakAll" } },
        grid: { top: gridTop, bottom: 70, containLabel: true }
      }, { notMerge: false, lazyUpdate: false, silent: true });
    };
`, optionName, optionName, quotedTitle, optionName, optionName, optionName, optionName, optionName, optionName, layoutFunctionName, chartName, optionName, optionName, chartName)
	layoutAfterSetOption := fmt.Sprintf(`
    %s();
    window.addEventListener("resize", %s);
    if (window.ResizeObserver) {
      const %sObserver = new ResizeObserver(%s);
      %sObserver.observe(%s.getDom());
    }
`, layoutFunctionName, layoutFunctionName, layoutFunctionName, layoutFunctionName, layoutFunctionName, chartName)

	setCallEnd := setIndex + len(setMarker)
	return markup[:setIndex] + layoutBeforeSetOption + markup[setIndex:setCallEnd] + layoutAfterSetOption + markup[setCallEnd:]
}
