package charts

import (
	"strings"
	"testing"
)

func TestLineFenceRendersAccessibleStaticSVG(t *testing.T) {
	out := renderChartPayload(t, readLineFixture(t), 1<<20)
	if !strings.Contains(out, `role="img"`) || !strings.Contains(out, `aria-label="Revenue"`) {
		t.Fatalf("line output is missing accessible name")
	}
	want := []string{"Revenue|Development|12", "Revenue|Production|18", "Cost|Development|7", "Cost|Production|9"}
	if got := extractAccessibleRows(out); strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("line rows = %#v, want %#v", got, want)
	}
	if strings.Contains(out, "data-chart-runtime") {
		t.Fatalf("line output contains runtime marker")
	}
}

func TestLineJSONPayloadUsesTheSameAdapter(t *testing.T) {
	payload := `{"schemaVersion":1,"type":"line","title":"Revenue","categories":["Development","Production"],"series":[{"name":"Revenue","values":[12,18]},{"name":"Cost","values":[7,9]}]}`
	out := renderChartPayload(t, payload, 1<<20)
	if got := extractAccessibleRows(out); strings.Join(got, "\x00") != "Revenue|Development|12\x00Revenue|Production|18\x00Cost|Development|7\x00Cost|Production|9" {
		t.Fatalf("line JSON rows = %#v", got)
	}
}

func TestLineRejectsAlignmentAndNonFiniteValues(t *testing.T) {
	for name, payload := range map[string]string{
		"alignment": "schemaVersion: 1\ntype: line\ntitle: T\ncategories: [A, B]\nseries: [{name: S, values: [1]}]\n",
		"nonfinite": "schemaVersion: 1\ntype: line\ntitle: T\ncategories: [A]\nseries: [{name: S, values: [.inf]}]\n",
		"duplicate": "schemaVersion: 1\ntype: line\ntitle: T\ncategories: [A, A]\nseries: [{name: S, values: [1, 2]}]\n",
	} {
		t.Run(name, func(t *testing.T) {
			err := renderChartPayloadError(payload)
			if err == nil {
				t.Fatal("invalid line accepted")
			}
		})
	}
}

func readLineFixture(t *testing.T) string {
	t.Helper()
	body, err := readFixtureFile("testdata/line/line-valid.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return body
}
