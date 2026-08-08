package charts

import (
	"strings"
	"testing"
)

func TestScatterPointModeRendersEveryPoint(t *testing.T) {
	body, err := readFixtureFile("testdata/scatter/scatter-valid.yaml")
	if err != nil {
		t.Fatal(err)
	}
	out := renderChartPayload(t, body, 1<<20)
	want := "Latency|p50|12\x00Latency|p95|18\x00Throughput|p50|30\x00Throughput|p95|42"
	if got := extractAccessibleRows(out); strings.Join(got, "\x00") != want {
		t.Fatalf("point rows = %#v", got)
	}
	if !strings.Contains(out, `role="img"`) || strings.Contains(out, "data-chart-runtime") {
		t.Fatalf("scatter output is not static accessible SVG")
	}
}

func TestScatterAlignedModeRendersEverySample(t *testing.T) {
	body, err := readFixtureFile("testdata/scatter/scatter-aligned.yaml")
	if err != nil {
		t.Fatal(err)
	}
	out := renderChartPayload(t, body, 1<<20)
	want := "Latency|Development|p50|12\x00Latency|Production|p95|18\x00Throughput|Development|p50|30\x00Throughput|Production|p95|42"
	if got := extractAccessibleRows(out); strings.Join(got, "\x00") != want {
		t.Fatalf("aligned rows = %#v", got)
	}
}

func TestScatterRejectsUnknownCategoryAndConflictingModes(t *testing.T) {
	for name, payload := range map[string]string{
		"unknown":   "schemaVersion: 1\ntype: scatter\ntitle: T\ncategories: [A]\nseries: [{name: S, points: [{category: B, value: 1}]}]\n",
		"both":      "schemaVersion: 1\ntype: scatter\ntitle: T\ncategories: [A]\nseries: [{name: S, points: [{category: A, value: 1}], values: [[1]]}]\n",
		"nonfinite": "schemaVersion: 1\ntype: scatter\ntitle: T\ncategories: [A]\nseries: [{name: S, points: [{category: A, value: .inf}]}]\n",
	} {
		t.Run(name, func(t *testing.T) {
			err := renderChartPayloadError(payload)
			if err == nil {
				t.Fatal("invalid scatter accepted")
			}
			if name == "unknown" && !strings.Contains(err.Error(), "chart.scatter.category_unknown") {
				t.Fatalf("unknown category error = %v", err)
			}
		})
	}
}
