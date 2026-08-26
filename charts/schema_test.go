package charts

import (
	"encoding/json"
	"strings"
	"testing"
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
			if !ok || !strings.Contains(description, "deck") || !strings.Contains(description, "static") {
				t.Fatalf("renderer target contract = %#v", renderer["description"])
			}
			enum, ok := renderer["enum"].([]any)
			if !ok || len(enum) != 2 || enum[0] != "static" || enum[1] != "interactive" {
				t.Fatalf("renderer enum = %#v", renderer["enum"])
			}
		})
	}
}
