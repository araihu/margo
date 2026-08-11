package margo

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestEmbeddedSchemasCompileAndValidateExamples(t *testing.T) {
	for _, kind := range []SchemaKind{SchemaPolicy, SchemaDocument} {
		data, err := Schema(kind)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasSuffix(data, []byte("\n")) {
			t.Fatalf("%s schema must have a final newline", kind)
		}
		if _, err := compiledSchema(kind); err != nil {
			t.Fatalf("compile %s schema: %v", kind, err)
		}
		var document struct {
			Examples []json.RawMessage `json:"examples"`
		}
		if err := json.Unmarshal(data, &document); err != nil {
			t.Fatal(err)
		}
		for index, example := range document.Examples {
			if _, err := validateJSONSchema(kind, example); err != nil {
				t.Errorf("%s example %d: %v", kind, index, err)
			}
		}
	}
}

func TestSchemaReturnsDetachedBytes(t *testing.T) {
	first, err := Schema(SchemaPolicy)
	if err != nil {
		t.Fatal(err)
	}
	first[0] = 'x'
	second, err := Schema(SchemaPolicy)
	if err != nil {
		t.Fatal(err)
	}
	if second[0] != '{' {
		t.Fatal("embedded schema bytes were mutated")
	}
}

func TestPolicyCodeDefaultsMatchCanonicalSchema(t *testing.T) {
	data, err := Schema(SchemaPolicy)
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties map[string]struct {
			Default any `json:"default"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	defaults := DefaultPolicy()
	if schema.Properties["rawHTML"].Default != string(defaults.RawHTML) ||
		int64(schema.Properties["inputBytes"].Default.(float64)) != defaults.InputBytes ||
		int64(schema.Properties["outputBytes"].Default.(float64)) != defaults.OutputBytes {
		t.Fatalf("schema defaults do not match code: schema=%+v code=%+v", schema.Properties, defaults)
	}
}
