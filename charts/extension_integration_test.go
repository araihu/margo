package charts

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"

	chartassets "github.com/araihu/goshtoso-charts/assets"
	margo "github.com/araihu/margo"
)

func TestRootCompileDispatchesEveryV1ChartFamily(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{name: "bar", body: readChartFixtureForIntegration(t, "testdata/bar/bar-valid.yaml"), want: "Revenue|Development|12\x00Revenue|Production|18\x00Cost|Development|7\x00Cost|Production|9"},
		{name: "line", body: readChartFixtureForIntegration(t, "testdata/line/line-valid.yaml"), want: "Revenue|Development|12\x00Revenue|Production|18\x00Cost|Development|7\x00Cost|Production|9"},
		{name: "pie", body: readChartFixtureForIntegration(t, "testdata/pie/pie-valid.json"), want: "Desktop|40\x00Mobile|60"},
		{name: "doughnut", body: readChartFixtureForIntegration(t, "testdata/pie/doughnut-valid.json"), want: "Desktop|40\x00Mobile|60"},
		{name: "scatter", body: readChartFixtureForIntegration(t, "testdata/scatter/scatter-valid.yaml"), want: "Latency|p50|12\x00Latency|p95|18\x00Throughput|p50|30\x00Throughput|p95|42"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, observed, err := renderThroughRoot(t, tc.body, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(extractAccessibleRows(string(out)), "\x00") != tc.want {
				t.Fatalf("rows = %#v", extractAccessibleRows(string(out)))
			}
			if !strings.Contains(string(out), `data-margo-chart-print`) {
				t.Fatalf("%s output missing print policy", tc.name)
			}
			if observed != 1<<20 {
				t.Fatalf("observed policy = %d", observed)
			}
		})
	}
}

func TestRootDefaultEnablesChartControlWrapper(t *testing.T) {
	body := readChartFixtureForIntegration(t, "testdata/bar/bar-valid.yaml")
	out, observed, err := renderThroughRoot(t, body, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	markup := string(out)
	for _, marker := range []string{
		`data-goshtoso-chart-wrapper`,
		`<div x-data class="goshtoso-charts-control-wrapper"`,
		`data-goshtoso-chart-wrapper-mode="enabled"`,
		`data-goshtoso-chart-capability="static-svg"`,
		`data-goshtoso-chart-export-filename="revenue"`,
		chartassets.ControlRuntimeURL,
		`goshtoso-charts-palette-auto`,
		`var(--color-chart-series-1)`,
		`data-margo-chart-print`,
		`@media print`,
		`[data-goshtoso-chart-wrapper] [data-goshtoso-chart-actions-fieldset]`,
	} {
		if !strings.Contains(markup, marker) {
			t.Fatalf("default chart output missing %q", marker)
		}
	}
	if observed != 1<<20 {
		t.Fatalf("observed policy = %d", observed)
	}
}

func TestRootSupportsInteractiveBarRenderer(t *testing.T) {
	body := `schemaVersion: 1
type: bar
renderer: interactive
title: Revenue
categories: [Development, Production]
series:
  - name: Revenue
    values: [12, 18]
  - name: Cost
    values: [7, 9]`
	out, observed, err := renderThroughRoot(t, body, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	markup := string(out)
	for _, marker := range []string{
		`class="goshtoso-charts-interactive`,
		`data-goshtoso-chart-capability="interactive-raster"`,
		`data-goshtoso-charts-explicit-animation="false"`,
		`exportFromMenu($el, &#34;png&#34;)`,
		`data-margo-chart-data="v1"`,
		`data-margo-extension-script="charts"`,
	} {
		if !strings.Contains(markup, marker) {
			t.Fatalf("interactive chart output missing %q: %s", marker, markup)
		}
	}
	if observed != 1<<20 {
		t.Fatalf("observed policy = %d", observed)
	}
}

func TestRootSupportsInteractiveLineRenderer(t *testing.T) {
	body := `schemaVersion: 1
type: line
renderer: interactive
title: Revenue trend
categories: [Q1, Q2]
series:
  - name: Revenue
    values: [12, 18]
  - name: Cost
    values: [7, 9]`
	out, observed, err := renderThroughRoot(t, body, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	markup := string(out)
	for _, marker := range []string{
		`class="goshtoso-charts-interactive`,
		`data-goshtoso-chart-capability="interactive-raster"`,
		`data-goshtoso-charts-explicit-animation="false"`,
		`exportFromMenu($el, &#34;png&#34;)`,
		`Revenue|Q1|12`,
		`Cost|Q2|9`,
	} {
		if !strings.Contains(markup, marker) {
			t.Fatalf("interactive line output missing %q: %s", marker, markup)
		}
	}
	if observed != 1<<20 {
		t.Fatalf("observed policy = %d", observed)
	}
}

func TestRootSupportsRemainingInteractiveChartFamilies(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantRows string
		want     string
	}{
		{
			name: "pie",
			body: `schemaVersion: 1
type: pie
renderer: interactive
title: Device share
slices:
  - {name: Desktop, value: 40}
  - {name: Mobile, value: 60}`,
			wantRows: "Desktop|40\x00Mobile|60",
			want:     `"radius":["0%","75%"]`,
		},
		{
			name: "doughnut",
			body: `schemaVersion: 1
type: doughnut
renderer: interactive
title: Device share
slices:
  - {name: Desktop, value: 40}
  - {name: Mobile, value: 60}`,
			wantRows: "Desktop|40\x00Mobile|60",
			want:     `"radius":["40%","75%"]`,
		},
		{
			name: "scatter",
			body: `schemaVersion: 1
type: scatter
renderer: interactive
title: Latency
categories: [p50, p95]
series:
  - name: Latency
    points:
      - {category: p95, value: 18}
      - {category: p50, value: 12}
  - name: Throughput
    values: [[30], [42]]`,
			wantRows: "Latency|p50|12\x00Latency|p95|18\x00Throughput|p50|30\x00Throughput|p95|42",
			want:     `"data":[{"name":"p50","value":12},{"name":"p95","value":18}]`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, observed, err := renderThroughRoot(t, tc.body, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			markup := string(out)
			for _, marker := range []string{
				`class="goshtoso-charts-interactive`,
				`data-goshtoso-chart-capability="interactive-raster"`,
				`data-goshtoso-charts-explicit-animation="false"`,
				`exportFromMenu($el, &#34;png&#34;)`,
				`data-margo-chart-data="v1"`,
				tc.want,
			} {
				if !strings.Contains(markup, marker) {
					t.Fatalf("interactive %s output missing %q: %s", tc.name, marker, markup)
				}
			}
			if got := strings.Join(extractAccessibleRows(markup), "\x00"); got != tc.wantRows {
				t.Fatalf("interactive %s rows = %q, want %q", tc.name, got, tc.wantRows)
			}
			if observed != 1<<20 {
				t.Fatalf("observed policy = %d", observed)
			}
		})
	}
}

func TestRemainingInteractiveChartsRejectUnknownRenderer(t *testing.T) {
	cases := map[string]string{
		"pie":      "schemaVersion: 1\ntype: pie\nrenderer: holographic\ntitle: Share\nslices: [{name: A, value: 1}]",
		"doughnut": "schemaVersion: 1\ntype: doughnut\nrenderer: holographic\ntitle: Share\nslices: [{name: A, value: 1}]",
		"scatter":  "schemaVersion: 1\ntype: scatter\nrenderer: holographic\ntitle: Latency\ncategories: [p50]\nseries: [{name: S, points: [{category: p50, value: 1}]}]",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := renderThroughRoot(t, body, 1<<20)
			if err == nil || !strings.Contains(err.Error(), "chart.renderer_invalid") {
				t.Fatalf("error = %v, want chart.renderer_invalid", err)
			}
		})
	}
}

func TestRemainingInteractiveChartsRejectElementClasses(t *testing.T) {
	cases := map[string]string{
		"pie":     "schemaVersion: 1\ntype: pie\nrenderer: interactive\ntitle: Share\nslices: [{name: A, class: slice-a, value: 1}]",
		"scatter": "schemaVersion: 1\ntype: scatter\nrenderer: interactive\ntitle: Latency\ncategories: [p50]\nseries: [{name: S, class: series-s, points: [{category: p50, value: 1}]}]",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := renderThroughRoot(t, body, 1<<20)
			if err == nil || !strings.Contains(err.Error(), "chart.renderer_style_unsupported") {
				t.Fatalf("error = %v, want chart.renderer_style_unsupported", err)
			}
		})
	}
}

func TestInteractiveScatterRequiresExactlyOneSamplePerCategory(t *testing.T) {
	cases := map[string]string{
		"multiple aligned values": `schemaVersion: 1
type: scatter
renderer: interactive
title: Latency
categories: [Development, Production]
series:
  - name: Latency
    values: [[12, 14], [18]]
    samples: [[p50, p75], [p95]]`,
		"duplicate point category": `schemaVersion: 1
type: scatter
renderer: interactive
title: Latency
categories: [p50]
series:
  - name: Latency
    points:
      - {category: p50, value: 12}
      - {category: p50, value: 14}`,
		"missing point category": `schemaVersion: 1
type: scatter
renderer: interactive
title: Latency
categories: [p50, p95]
series:
  - name: Latency
    points: [{category: p50, value: 12}]`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := renderThroughRoot(t, body, 1<<20)
			if err == nil || !strings.Contains(err.Error(), "chart.renderer_data_unsupported") {
				t.Fatalf("error = %v, want chart.renderer_data_unsupported", err)
			}
		})
	}
}

func TestRemainingInteractiveRenderersRequireControlWrapper(t *testing.T) {
	cases := map[string]string{
		"pie":     "schemaVersion: 1\ntype: pie\nrenderer: interactive\ntitle: Share\nslices: [{name: A, value: 1}]",
		"scatter": "schemaVersion: 1\ntype: scatter\nrenderer: interactive\ntitle: Latency\ncategories: [p50]\nseries: [{name: S, points: [{category: p50, value: 1}]}]",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := renderThroughRootWithExtension(t, body, 1<<20, Extension(WithControlWrapper(false)))
			if err == nil || !strings.Contains(err.Error(), "chart.renderer_controls_required") {
				t.Fatalf("error = %v, want chart.renderer_controls_required", err)
			}
		})
	}
}

func TestCartesianChartsRejectUnknownRenderer(t *testing.T) {
	for _, chartType := range []string{"bar", "line"} {
		t.Run(chartType, func(t *testing.T) {
			body := "schemaVersion: 1\ntype: " + chartType + "\nrenderer: holographic\ntitle: Revenue\ncategories: [Q1]\nseries: [{name: Revenue, values: [12]}]"
			_, _, err := renderThroughRoot(t, body, 1<<20)
			if err == nil || !strings.Contains(err.Error(), "chart.renderer_invalid") {
				t.Fatalf("error = %v, want chart.renderer_invalid", err)
			}
		})
	}
}

func TestInteractiveCartesianChartsRejectSeriesClasses(t *testing.T) {
	for _, chartType := range []string{"bar", "line"} {
		t.Run(chartType, func(t *testing.T) {
			body := "schemaVersion: 1\ntype: " + chartType + "\nrenderer: interactive\ntitle: Revenue\ncategories: [Q1]\nseries: [{name: Revenue, class: revenue-series, values: [12]}]"
			_, _, err := renderThroughRoot(t, body, 1<<20)
			if err == nil || !strings.Contains(err.Error(), "chart.renderer_style_unsupported") {
				t.Fatalf("error = %v, want chart.renderer_style_unsupported", err)
			}
		})
	}
}

func TestInteractiveRendererRequiresControlWrapperInPOC(t *testing.T) {
	body := `schemaVersion: 1
type: bar
renderer: interactive
title: Revenue
categories: [Q1]
series: [{name: Revenue, values: [12]}]`
	_, _, err := renderThroughRootWithExtension(t, body, 1<<20, Extension(WithControlWrapper(false)))
	if err == nil || !strings.Contains(err.Error(), "chart.renderer_controls_required") {
		t.Fatalf("error = %v, want chart.renderer_controls_required", err)
	}
}

func TestRootChartSchemaSupportsThemeClassAndHexOverrides(t *testing.T) {
	barBody := `schemaVersion: 1
type: bar
title: Revenue
style:
  palette: pastel
  class: margo-custom-chart
  colors: ["#112233", "#445566"]
categories: [Development, Production]
series:
  - name: Revenue
    values: [12, 18]
  - name: Cost
    values: [7, 9]`
	out, _, err := renderThroughRoot(t, barBody, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	markup := string(out)
	for _, marker := range []string{
		`goshtoso-charts-palette-pastel`,
		`margo-custom-chart`,
		`fill:#112233`,
		`fill:#445566`,
	} {
		if !strings.Contains(markup, marker) {
			t.Fatalf("custom bar output missing %q", marker)
		}
	}

	lineBody := `schemaVersion: 1
type: line
title: Trend
style:
  class: margo-line-chart
categories: [Q1, Q2]
series:
  - name: Revenue
    color: "#123456"
    values: [10, 14]
  - name: Cost
    class: margo-cost-series
    values: [7, 9]`
	out, _, err = renderThroughRoot(t, lineBody, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	markup = string(out)
	for _, marker := range []string{`margo-line-chart`, `fill:#123456`, `margo-cost-series`} {
		if !strings.Contains(markup, marker) {
			t.Fatalf("custom line output missing %q", marker)
		}
	}

	pieBody := `schemaVersion: 1
type: pie
title: Mix
style:
  class: margo-pie-chart
slices:
  - name: Desktop
    class: desktop-slice
    value: 40
  - name: Mobile
    color: "#0f766e"
    value: 60`
	out, _, err = renderThroughRoot(t, pieBody, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	markup = string(out)
	for _, marker := range []string{`margo-pie-chart`, `desktop-slice`, `fill:#0f766e`} {
		if !strings.Contains(markup, marker) {
			t.Fatalf("custom pie output missing %q", marker)
		}
	}

	scatterBody := `schemaVersion: 1
type: scatter
title: Latency
style:
  class: margo-scatter-chart
categories: [p50, p95]
series:
  - name: Latency
    class: latency-series
    points:
      - category: p50
        value: 12
      - category: p95
        value: 18
  - name: Throughput
    color: "#7c3aed"
    points:
      - category: p50
        value: 30
      - category: p95
        value: 42`
	out, _, err = renderThroughRoot(t, scatterBody, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	markup = string(out)
	for _, marker := range []string{`margo-scatter-chart`, `latency-series`, `fill:#7c3aed`} {
		if !strings.Contains(markup, marker) {
			t.Fatalf("custom scatter output missing %q", marker)
		}
	}
}

func TestRootCanDisableChartControlWrapper(t *testing.T) {
	body := readChartFixtureForIntegration(t, "testdata/bar/bar-valid.yaml")
	out, observed, err := renderThroughRootWithExtension(t, body, 1<<20, Extension(WithControlWrapper(false)))
	if err != nil {
		t.Fatal(err)
	}
	markup := string(out)
	for _, marker := range []string{
		`data-goshtoso-chart-wrapper-mode`,
		chartassets.ControlRuntimeURL,
		`data-goshtoso-chart-export-filename`,
	} {
		if strings.Contains(markup, marker) {
			t.Fatalf("disabled chart output contains %q", marker)
		}
	}
	if !strings.Contains(markup, `data-margo-chart-print-data="disabled"`) {
		t.Fatal("disabled chart wrapper lost default print-data policy")
	}
	if !strings.Contains(markup, `data-margo-chart-data="v1"`) {
		t.Fatal("disabled chart output lost accessible data table")
	}
	if observed != 1<<20 {
		t.Fatalf("observed policy = %d", observed)
	}
}

func TestChartControlWrapperConfigurationIsIdentityBound(t *testing.T) {
	enabled := Extension().Identity.ConfigurationHash
	disabled := Extension(WithControlWrapper(false)).Identity.ConfigurationHash
	if enabled == "" || disabled == "" || enabled == disabled {
		t.Fatalf("configuration hashes = enabled %q disabled %q", enabled, disabled)
	}
}

func TestPrintableAccessibleDataIsExplicitAndIdentityBound(t *testing.T) {
	defaultRegistration := Extension()
	enabledRegistration := Extension(WithPrintableAccessibleData(true))
	if defaultRegistration.Identity.ConfigurationHash == enabledRegistration.Identity.ConfigurationHash {
		t.Fatal("printable accessible data did not change extension identity")
	}
	body := readChartFixtureForIntegration(t, "testdata/bar/bar-valid.yaml")
	out, _, err := renderThroughRootWithExtension(t, body, 1<<20, enabledRegistration)
	if err != nil {
		t.Fatal(err)
	}
	markup := string(out)
	for _, want := range []string{
		`data-margo-chart-print-data="enabled"`,
		`.margo-chart-data {`,
		`display: block !important`,
		`border-collapse: collapse`,
		`[data-field="canonical"]`,
		`[data-goshtoso-chart-content] > details`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("printable accessible data output missing %q: %s", want, markup)
		}
	}
}

func TestRootPolicyOverflowWritesNoCallerBytes(t *testing.T) {
	body := readChartFixtureForIntegration(t, "testdata/bar/bar-valid.yaml")
	out, observed, err := renderThroughRoot(t, body, 1)
	if err == nil || !strings.Contains(err.Error(), "chart.output_limit") {
		t.Fatalf("overflow error = %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("overflow returned %d bytes", len(out))
	}
	if observed != 1 {
		t.Fatalf("observed policy = %d", observed)
	}
}

func TestRootCompilerMismatchFailsBeforeChartSession(t *testing.T) {
	body := readChartFixtureForIntegration(t, "testdata/bar/bar-valid.yaml")
	compiler := margo.New(margo.WithHostPolicy(margo.Policy{OutputBytes: 1 << 20}), margo.WithExtension(Extension()))
	doc, err := compiler.Compile(context.Background(), margo.Source{Name: "bar.md", Content: []byte("```goshtosochart\n" + body + "\n```")})
	if err != nil {
		t.Fatal(err)
	}
	observed := int64(0)
	restore := setAccessiblePolicyObserverForTest(func(policy AccessibleRenderPolicy) { observed = policy.MaxOutputBytes })
	defer restore()
	mismatched := margo.New(margo.WithHostPolicy(margo.Policy{OutputBytes: 1}), margo.WithExtension(Extension()))
	_, err = mismatched.Render(context.Background(), doc)
	if err == nil || !strings.Contains(err.Error(), "compiler.document_config_mismatch") {
		t.Fatalf("mismatch error = %v", err)
	}
	if observed != 0 {
		t.Fatalf("chart session observed policy before mismatch: %d", observed)
	}
}

func TestChartSessionIsSafeForConcurrentRenders(t *testing.T) {
	body := readChartFixtureForIntegration(t, "testdata/line/line-valid.yaml")
	session, err := extensionFactory(margo.RenderContext{EffectivePolicy: margo.EffectivePolicy{OutputBytes: 1 << 20}})
	if err != nil {
		t.Fatal(err)
	}
	const renders = 8
	var wg sync.WaitGroup
	for i := 0; i < renders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var out bytes.Buffer
			if err := session.Render(context.Background(), margo.ExtensionNode{Fence: "goshtosochart", Payload: []byte(body)}, &out); err != nil {
				t.Errorf("concurrent render: %v", err)
			}
		}()
	}
	wg.Wait()
}

func renderThroughRoot(t *testing.T, body string, limit int64) ([]byte, int64, error) {
	return renderThroughRootWithExtension(t, body, limit, Extension())
}

func renderThroughRootWithExtension(t *testing.T, body string, limit int64, extension margo.ExtensionRegistration) ([]byte, int64, error) {
	t.Helper()
	var observed int64
	restore := setAccessiblePolicyObserverForTest(func(policy AccessibleRenderPolicy) { observed = policy.MaxOutputBytes })
	defer restore()
	compiler := margo.New(margo.WithHostPolicy(margo.Policy{OutputBytes: limit}), margo.WithExtension(extension))
	doc, err := compiler.Compile(context.Background(), margo.Source{Name: "chart.md", Content: []byte("```goshtosochart\n" + body + "\n```")})
	if err != nil {
		return nil, observed, err
	}
	result, err := compiler.Render(context.Background(), doc)
	if err != nil {
		return nil, observed, err
	}
	var out bytes.Buffer
	err = result.Content().Render(context.Background(), &out)
	return out.Bytes(), observed, err
}

func readChartFixtureForIntegration(t *testing.T, name string) string {
	t.Helper()
	body, err := readFixtureFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
