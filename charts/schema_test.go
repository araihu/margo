package charts

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

func TestSchemaReturnsExactBarV1(t *testing.T) {
	body, err := Schema("bar")
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatalf("schema is not JSON: %v", err)
	}
	if document["additionalProperties"] != false {
		t.Fatalf("schema is not closed: %#v", document)
	}
}

func TestSchemaRejectsUnknownFamily(t *testing.T) {
	if _, err := Schema("radar"); err == nil || !strings.Contains(err.Error(), "chart.type_unsupported") {
		t.Fatalf("unsupported schema error = %v", err)
	}
}

func TestSchemaUsesPieSchemaForDoughnut(t *testing.T) {
	pieSchema, err := Schema("pie")
	if err != nil {
		t.Fatal(err)
	}
	doughnutSchema, err := Schema("doughnut")
	if err != nil {
		t.Fatal(err)
	}
	if string(pieSchema) != string(doughnutSchema) {
		t.Fatal("pie and doughnut schemas diverged")
	}
}

func TestSchemasExposeThemeClassAndHexOverrides(t *testing.T) {
	for _, chartType := range []string{"bar", "line", "pie", "scatter"} {
		t.Run(chartType, func(t *testing.T) {
			body, err := Schema(chartType)
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err := json.Unmarshal(body, &document); err != nil {
				t.Fatal(err)
			}
			properties, ok := document["properties"].(map[string]any)
			if !ok {
				t.Fatal("schema properties missing")
			}
			style, ok := properties["style"].(map[string]any)
			if !ok {
				t.Fatal("style schema missing")
			}
			styleProperties, ok := style["properties"].(map[string]any)
			if !ok {
				t.Fatal("style properties missing")
			}
			for _, name := range []string{"palette", "class", "colors"} {
				if _, ok := styleProperties[name]; !ok {
					t.Fatalf("style property %q missing", name)
				}
			}

			itemsKey := "series"
			if chartType == "pie" {
				itemsKey = "slices"
			}
			items, ok := properties[itemsKey].(map[string]any)
			if !ok {
				t.Fatalf("%s property missing", itemsKey)
			}
			item, ok := items["items"].(map[string]any)
			if !ok {
				t.Fatalf("%s items missing", itemsKey)
			}
			itemProperties, ok := item["properties"].(map[string]any)
			if !ok {
				t.Fatalf("%s item properties missing", itemsKey)
			}
			for _, name := range []string{"class", "color"} {
				if _, ok := itemProperties[name]; !ok {
					t.Fatalf("%s property %q missing", itemsKey, name)
				}
			}
		})
	}
}

func TestEveryChartSchemaExposesRendererChoice(t *testing.T) {
	for _, chartType := range []string{"bar", "line", "pie", "doughnut", "scatter"} {
		t.Run(chartType, func(t *testing.T) {
			body, err := Schema(chartType)
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err := json.Unmarshal(body, &document); err != nil {
				t.Fatal(err)
			}
			properties := document["properties"].(map[string]any)
			renderer, ok := properties["renderer"].(map[string]any)
			if !ok {
				t.Fatal("renderer schema missing")
			}
			description, ok := renderer["description"].(string)
			if !ok || !strings.Contains(description, "by default") || !strings.Contains(description, "deck") || !strings.Contains(description, "static") {
				t.Fatalf("renderer target contract = %#v", renderer["description"])
			}
			enum, ok := renderer["enum"].([]any)
			if !ok || len(enum) != 2 || enum[0] != "static" || enum[1] != "interactive" {
				t.Fatalf("renderer enum = %#v", renderer["enum"])
			}
		})
	}
}

func TestDecodeEnvelopeValidatesEveryChartSchema(t *testing.T) {
	for chartType, payload := range map[string]string{
		"bar":      "schemaVersion: 1\ntype: bar\nstyle: null\ntitle: T\ncategories: [A]\nseries: [{name: S, values: [1]}]\n",
		"line":     "schemaVersion: 1\ntype: line\nstyle: null\ntitle: T\ncategories: [A]\nseries: [{name: S, values: [1]}]\n",
		"pie":      "schemaVersion: 1\ntype: pie\nstyle: null\ntitle: T\nslices: [{name: A, value: 1}]\n",
		"doughnut": "schemaVersion: 1\ntype: doughnut\nstyle: null\ntitle: T\nslices: [{name: A, value: 1}]\n",
		"scatter":  "schemaVersion: 1\ntype: scatter\nstyle: null\ntitle: T\ncategories: [A]\nseries: [{name: S, points: [{category: A, value: 1}]}]\n",
	} {
		t.Run(chartType, func(t *testing.T) {
			_, err := decodeEnvelope([]byte(payload))
			if err == nil || !strings.Contains(err.Error(), "chart.schema_invalid") {
				t.Fatalf("error = %v, want chart.schema_invalid", err)
			}
			var diagnostic *margo.DiagnosticError
			if !errors.As(err, &diagnostic) || len(diagnostic.Diagnostics) != 1 || diagnostic.Diagnostics[0].Pointer != "/style" {
				t.Fatalf("diagnostic = %#v, want /style pointer", diagnostic)
			}
		})
	}
}

func TestSchemasOwnDeclarativeChartValidation(t *testing.T) {
	for name, payload := range map[string]string{
		"bar categories are unique":      "schemaVersion: 1\ntype: bar\ntitle: T\ncategories: [A, A]\nseries: [{name: S, values: [1, 2]}]\n",
		"line names contain text":        "schemaVersion: 1\ntype: line\ntitle: T\ncategories: [A]\nseries: [{name: ' ', values: [1]}]\n",
		"pie paint is exclusive":         "schemaVersion: 1\ntype: pie\ntitle: T\nslices: [{name: A, value: 1, class: custom, color: '#123'}]\n",
		"doughnut paint is exclusive":    "schemaVersion: 1\ntype: doughnut\ntitle: T\nslices: [{name: A, value: 1, class: custom, color: '#123'}]\n",
		"scatter data mode is exclusive": "schemaVersion: 1\ntype: scatter\ntitle: T\ncategories: [A]\nseries: [{name: S, points: [{category: A, value: 1}], values: [[1]]}]\n",
		"scatter samples require values": "schemaVersion: 1\ntype: scatter\ntitle: T\ncategories: [A]\nseries: [{name: S, points: [{category: A, value: 1}], samples: [[one]]}]\n",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := decodeEnvelope([]byte(payload))
			if err == nil || !strings.Contains(err.Error(), "chart.schema_invalid") {
				t.Fatalf("error = %v, want chart.schema_invalid", err)
			}
		})
	}
}
