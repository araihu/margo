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
