package charts

import (
	"strings"
	"testing"
)

func TestWrapChartLabelUsesConservativeWordBoundaries(t *testing.T) {
	const source = "FY2026 consolidated revenue and operating performance by global operating segment and customer channel"
	wrapped := wrapChartLabel(source, interactiveBarTitleLineRunes)
	if wrapped == source {
		t.Fatal("long chart title was not wrapped")
	}
	if strings.ReplaceAll(wrapped, "\n", " ") != source {
		t.Fatalf("wrapped title changed source: %q", wrapped)
	}
	for _, line := range strings.Split(wrapped, "\n") {
		if len([]rune(line)) > interactiveBarTitleLineRunes {
			t.Fatalf("wrapped line has %d runes: %q", len([]rune(line)), line)
		}
	}
	if got := wrapChartLabel("short title", interactiveBarTitleLineRunes); got != "short title" {
		t.Fatalf("short title changed: %q", got)
	}
}

func TestRewriteInteractiveBarLayoutScriptKeepsOriginalSourceValues(t *testing.T) {
	const markup = `<script>let goecharts_ABC123 = echarts.init(document.getElementById('ABC123')); let option_ABC123 = {"title":{"text":"Original"},"legend":{}}; goecharts_ABC123.setOption(option_ABC123);</script>`
	out := rewriteInteractiveBarLayoutScript(markup, "A deliberately long chart title that needs wrapping")
	for _, marker := range []string{
		`option_ABC123.title = Object.assign`,
		`textStyle: Object.assign`,
		`overflow: "breakAll"`,
		`option_ABC123.grid = Object.assign`,
		`containLabel: true`,
		`const margoResponsiveBarLayout_ABC123`,
		`window.addEventListener("resize", margoResponsiveBarLayout_ABC123)`,
		`goecharts_ABC123.setOption(option_ABC123);`,
	} {
		if !strings.Contains(out, marker) {
			t.Fatalf("layout script missing %q: %s", marker, out)
		}
	}
	if !strings.Contains(out, `"title":{"text":"Original"}`) {
		t.Fatal("original option source was not preserved")
	}
}

func TestInteractiveBarStressLayoutPreservesAccessibleRows(t *testing.T) {
	body := `schemaVersion: 1
type: bar
renderer: interactive
title: FY2026 consolidated revenue and operating performance by global operating segment and customer channel
categories: [Advisory Services, Platforms and Software, Managed Services and Support]
series:
  - name: Revenue from recurring enterprise subscriptions and professional services (USD millions)
    values: [318, 412, 288]
  - name: Adjusted operating income after delivery, platform, and corporate investment (USD millions)
    values: [126, 168, 142]
  - name: Net cash generated from operating activities after working capital movements (USD millions)
    values: [91, 122, 103]`
	out, _, err := renderThroughRoot(t, body, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	markup := string(out)
	for _, marker := range []string{
		`overflow: "breakAll"`,
		`top: narrow ? 105 : 85`,
		`grid: { top: narrow ? 245 : 180`,
		`data-margo-chart-data="v1"`,
	} {
		if !strings.Contains(markup, marker) {
			t.Fatalf("stress output missing %q: %s", marker, markup)
		}
	}
	want := strings.Join([]string{
		"Revenue from recurring enterprise subscriptions and professional services (USD millions)|Advisory Services|318",
		"Revenue from recurring enterprise subscriptions and professional services (USD millions)|Platforms and Software|412",
		"Revenue from recurring enterprise subscriptions and professional services (USD millions)|Managed Services and Support|288",
		"Adjusted operating income after delivery, platform, and corporate investment (USD millions)|Advisory Services|126",
		"Adjusted operating income after delivery, platform, and corporate investment (USD millions)|Platforms and Software|168",
		"Adjusted operating income after delivery, platform, and corporate investment (USD millions)|Managed Services and Support|142",
		"Net cash generated from operating activities after working capital movements (USD millions)|Advisory Services|91",
		"Net cash generated from operating activities after working capital movements (USD millions)|Platforms and Software|122",
		"Net cash generated from operating activities after working capital movements (USD millions)|Managed Services and Support|103",
	}, "\x00")
	if got := strings.Join(extractAccessibleRows(markup), "\x00"); got != want {
		t.Fatalf("accessible rows changed: %q", got)
	}
	if strings.Contains(markup, "aria-label=\"FY2026 consolidated revenue and\n") {
		t.Fatal("visual title wrapping leaked into the source aria-label")
	}
}
