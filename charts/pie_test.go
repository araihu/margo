package charts

import (
	"strings"
	"testing"
)

func TestPieAndDoughnutRenderAccessibleStaticSVG(t *testing.T) {
	for _, name := range []string{"pie-valid.json", "doughnut-valid.json"} {
		t.Run(name, func(t *testing.T) {
			body, err := readFixtureFile("testdata/pie/" + name)
			if err != nil {
				t.Fatal(err)
			}
			out := renderChartPayload(t, body, 1<<20)
			if !strings.Contains(out, `role="img"`) || !strings.Contains(out, `aria-label="Share"`) {
				t.Fatalf("pie output is missing accessible name")
			}
			if got := extractAccessibleRows(out); strings.Join(got, "\x00") != "Desktop|40\x00Mobile|60" {
				t.Fatalf("rows = %#v", got)
			}
			if strings.Contains(out, "data-chart-runtime") {
				t.Fatalf("pie output contains runtime marker")
			}
			if name == "doughnut-valid.json" && !strings.Contains(out, "goshtoso-charts-pie") {
				t.Fatalf("doughnut output is not the static pie renderer")
			}
		})
	}
}

func TestPieRejectsInvalidSliceValuesAndNames(t *testing.T) {
	for name, payload := range map[string]string{
		"negative":  "schemaVersion: 1\ntype: pie\ntitle: T\nslices: [{name: A, value: -1}]\n",
		"nonfinite": "schemaVersion: 1\ntype: pie\ntitle: T\nslices: [{name: A, value: .nan}]\n",
		"duplicate": "schemaVersion: 1\ntype: pie\ntitle: T\nslices: [{name: A, value: 1}, {name: A, value: 2}]\n",
	} {
		t.Run(name, func(t *testing.T) {
			if err := renderChartPayloadError(payload); err == nil {
				t.Fatal("invalid pie accepted")
			}
		})
	}
}

func TestPieZeroDataKeepsRendererNoDataSummary(t *testing.T) {
	payload := "schemaVersion: 1\ntype: pie\ntitle: Share\nslices: [{name: Desktop, value: 0}, {name: Mobile, value: 0}]\n"
	out := renderChartPayload(t, payload, 1<<20)
	if !strings.Contains(out, "No data in this period.") {
		t.Fatalf("zero-data summary missing")
	}
	if got := extractAccessibleRows(out); strings.Join(got, "\x00") != "Desktop|0\x00Mobile|0" {
		t.Fatalf("zero-data rows = %#v", got)
	}
}
