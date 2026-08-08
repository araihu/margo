package charts

import (
	"embed"
	"fmt"
)

//go:embed schema/v1/*.json
var schemaFiles embed.FS

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
