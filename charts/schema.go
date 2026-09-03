package charts

import (
	"bytes"
	"embed"
	"fmt"
	"math"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

//go:embed schema/v1/*.json
var schemaFiles embed.FS

var loadChartSchemas = sync.OnceValues(func() (map[string]*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	for _, chartType := range []string{"bar", "line", "pie", "scatter"} {
		body, err := schemaFiles.ReadFile("schema/v1/" + chartType + ".json")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", chartType, err)
		}
		document, err := jsonschema.UnmarshalJSON(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", chartType, err)
		}
		if err := compiler.AddResource(chartSchemaURL(chartType), document); err != nil {
			return nil, fmt.Errorf("%s: %w", chartType, err)
		}
	}
	result := make(map[string]*jsonschema.Schema, 4)
	for _, chartType := range []string{"bar", "line", "pie", "scatter"} {
		compiled, err := compiler.Compile(chartSchemaURL(chartType))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", chartType, err)
		}
		result[chartType] = compiled
	}
	return result, nil
})

// Schema returns the closed v1 JSON schema for one supported chart family.
func Schema(chartType string) ([]byte, error) {
	if chartType == "doughnut" {
		chartType = "pie"
	}
	if chartType != "bar" && chartType != "line" && chartType != "pie" && chartType != "scatter" {
		return nil, chartDiagnostic("chart.type_unsupported", fmt.Sprintf("chart type %q is unsupported", chartType))
	}
	body, err := schemaFiles.ReadFile("schema/v1/" + chartType + ".json")
	if err != nil {
		return nil, chartDiagnostic("chart.schema_unavailable", err.Error())
	}
	return append([]byte(nil), body...), nil
}

func validateChartSchema(chartType string, node *yaml.Node) error {
	if chartType == "doughnut" {
		chartType = "pie"
	}
	if chartType != "bar" && chartType != "line" && chartType != "pie" && chartType != "scatter" {
		return chartDiagnostic("chart.type_unsupported", fmt.Sprintf("chart type %q is unsupported", chartType))
	}
	schemas, err := loadChartSchemas()
	if err != nil {
		return chartDiagnostic("chart.schema_unavailable", err.Error())
	}
	var value any
	if err := node.Decode(&value); err != nil {
		return chartDiagnosticAt("chart.schema_invalid", err.Error(), node.Line, node.Column)
	}
	// YAML accepts .nan and .inf, which are outside JSON's data model and unsafe
	// to pass to the JSON Schema validator.
	if containsNonFiniteNumber(value) {
		code := "chart.value_non_finite"
		if chartType == "bar" {
			code = "chart.semantic_value_invalid"
		}
		return chartDiagnosticAt(code, chartType+" values must be finite", node.Line, node.Column)
	}
	if err := schemas[chartType].Validate(value); err != nil {
		return chartSchemaValidationDiagnostic(err, node.Line, node.Column)
	}
	return nil
}

func containsNonFiniteNumber(value any) bool {
	switch value := value.(type) {
	case float64:
		return math.IsNaN(value) || math.IsInf(value, 0)
	case []any:
		for _, item := range value {
			if containsNonFiniteNumber(item) {
				return true
			}
		}
	case map[string]any:
		for _, item := range value {
			if containsNonFiniteNumber(item) {
				return true
			}
		}
	}
	return false
}

func chartSchemaURL(chartType string) string {
	return "https://araihu.github.io/margo/charts/schema/v1/" + chartType + ".json"
}

func chartSchemaValidationDiagnostic(err error, line, column int) error {
	code := "chart.schema_invalid"
	pointer := ""
	if validation, ok := err.(*jsonschema.ValidationError); ok {
		output := validation.BasicOutput()
		pointer = output.InstanceLocation
		if len(output.Errors) > 0 {
			pointer = output.Errors[0].InstanceLocation
		}
		if pointer == "/renderer" {
			code = "chart.renderer_invalid"
		}
	}
	return chartDiagnosticAtPointer(code, err.Error(), pointer, line, column)
}
