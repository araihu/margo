package charts

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

func TestBarFenceRendersAccessibleStaticSVG(t *testing.T) {
	const outputLimit = int64(1 << 20)
	out := renderBarFixture(t, "bar-valid.yaml", outputLimit)
	if !strings.Contains(out, `role="img"`) {
		t.Fatalf("bar output has no img role")
	}
	if !strings.Contains(out, `aria-label="Revenue"`) {
		t.Fatalf("bar output has no title aria-label")
	}
	want := []string{"Revenue|Development|12", "Revenue|Production|18", "Cost|Development|7", "Cost|Production|9"}
	if got := extractAccessibleRows(out); len(got) != len(want) || strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("accessible rows = %#v, want %#v", got, want)
	}
	if strings.Contains(out, "data-chart-runtime") || int64(len(out)) > outputLimit {
		t.Fatalf("unexpected runtime marker or output size: %d", len(out))
	}
}

func TestBarFenceRendersBothOrientationsWithCompleteAccessibleRows(t *testing.T) {
	want := []string{"Revenue|Development|12", "Revenue|Production|18", "Cost|Development|7", "Cost|Production|9"}
	for _, orientation := range []string{"vertical", "horizontal"} {
		payload := strings.Replace(readBarFixture(t), "orientation: vertical", "orientation: "+orientation, 1)
		out := renderChartPayload(t, payload, 1<<20)
		if got := extractAccessibleRows(out); len(got) != len(want) || strings.Join(got, "\x00") != strings.Join(want, "\x00") {
			t.Fatalf("%s rows = %#v, want %#v", orientation, got, want)
		}
		if strings.Contains(out, "data-chart-runtime") {
			t.Fatalf("%s output contains runtime marker", orientation)
		}
	}
}

func TestBarJSONPayloadUsesTheSameAdapter(t *testing.T) {
	payload := `{"schemaVersion":1,"type":"bar","title":"Revenue","categories":["Development","Production"],"series":[{"name":"Revenue","values":[12,18]},{"name":"Cost","values":[7,9]}]}`
	out := renderChartPayload(t, payload, 1<<20)
	want := []string{"Revenue|Development|12", "Revenue|Production|18", "Cost|Development|7", "Cost|Production|9"}
	if got := extractAccessibleRows(out); len(got) != len(want) || strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("JSON rows = %#v, want %#v", got, want)
	}
}

func TestBarSemanticValidationRejectsInvalidModels(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		code    string
	}{
		{name: "alignment", payload: "schemaVersion: 1\ntype: bar\ntitle: T\ncategories: [A, B]\nseries: [{name: S, values: [1]}]\n", code: "chart.semantic_alignment_invalid"},
		{name: "duplicate category", payload: "schemaVersion: 1\ntype: bar\ntitle: T\ncategories: [A, A]\nseries: [{name: S, values: [1, 2]}]\n", code: "chart.semantic_category_duplicate"},
		{name: "duplicate series", payload: "schemaVersion: 1\ntype: bar\ntitle: T\ncategories: [A]\nseries: [{name: S, values: [1]}, {name: S, values: [2]}]\n", code: "chart.semantic_series_duplicate"},
		{name: "nonfinite", payload: "schemaVersion: 1\ntype: bar\ntitle: T\ncategories: [A]\nseries: [{name: S, values: [.nan]}]\n", code: "chart.semantic_value_invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := renderChartPayloadError(tc.payload); err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("error = %v, want %s", err, tc.code)
			}
		})
	}
}

func TestBarPropagatesEffectiveOutputPolicy(t *testing.T) {
	payload := readBarFixture(t)
	if err := renderChartPayloadErrorWithLimit(payload, 1); err == nil || !strings.Contains(err.Error(), "chart.output_limit") {
		t.Fatalf("policy overflow = %v", err)
	}
}

func renderBarFixture(t *testing.T, name string, limit int64) string {
	t.Helper()
	return renderChartPayload(t, mustReadBarFixture(t, name), limit)
}

func renderChartPayload(t *testing.T, payload string, limit int64) string {
	t.Helper()
	envelope, err := decodeEnvelope([]byte(payload))
	if err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	session, err := extensionFactory(margo.RenderContext{EffectivePolicy: margo.EffectivePolicy{OutputBytes: limit}})
	if err != nil {
		t.Fatalf("extension factory: %v", err)
	}
	var output bytes.Buffer
	if err := session.Render(context.Background(), margo.ExtensionNode{Fence: "goshtosochart", Payload: []byte(payload)}, &output); err != nil {
		t.Fatalf("render %s: %v", envelope.Type, err)
	}
	return output.String()
}

func renderChartPayloadError(payload string) error {
	return renderChartPayloadErrorWithLimit(payload, 1<<20)
}

func renderChartPayloadErrorWithLimit(payload string, limit int64) error {
	session, err := extensionFactory(margo.RenderContext{EffectivePolicy: margo.EffectivePolicy{OutputBytes: limit}})
	if err != nil {
		return err
	}
	var output bytes.Buffer
	return session.Render(context.Background(), margo.ExtensionNode{Fence: "goshtosochart", Payload: []byte(payload)}, &output)
}

func readBarFixture(t *testing.T) string {
	return mustReadBarFixture(t, "bar-valid.yaml")
}

func mustReadBarFixture(t *testing.T, name string) string {
	t.Helper()
	body, err := readFixtureFile("testdata/bar/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func readFixtureFile(name string) (string, error) {
	body, err := os.ReadFile(name)
	return string(body), err
}
