package margo

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed schema/v1/output/*.json
var outputSchemaFiles embed.FS

var outputSchemaNames = map[SchemaKind]string{
	SchemaDiagnostic:            "diagnostic.json",
	SchemaDoctorReport:          "doctor-report.json",
	SchemaCheckReport:           "check-report.json",
	SchemaSiteReport:            "site-report.json",
	SchemaSiteManifest:          "site-manifest.json",
	SchemaRuntimeDescriptor:     "runtime-descriptor.json",
	SchemaRuntimeReport:         "runtime-report.json",
	SchemaDeckLayoutEvidence:    "deck-layout-evidence.json",
	SchemaDeckPDFArtifactReport: "deck-pdf-artifact-report.json",
}

// OutputSchema returns the exact JSON Schema shipped for a versioned Margo
// output envelope. These schemas are also available to the jsonschema fence
// through margo://schema/v1/output/<name> references.
func OutputSchema(kind SchemaKind) ([]byte, error) {
	name, ok := outputSchemaNames[kind]
	if !ok {
		return nil, fmt.Errorf("schema.kind_invalid: unsupported output schema %q", kind)
	}
	data, err := outputSchemaFiles.ReadFile("schema/v1/output/" + name)
	if err != nil {
		return nil, fmt.Errorf("schema.unavailable: %s: %w", kind, err)
	}
	return append([]byte(nil), data...), nil
}

func embeddedJSONSchemaReference(reference string) ([]byte, error) {
	const prefix = "margo://schema/v1/"
	if !strings.HasPrefix(reference, prefix) {
		return nil, fmt.Errorf("unknown embedded schema reference %q", reference)
	}
	name := strings.TrimPrefix(reference, prefix)
	if index := strings.IndexByte(name, '#'); index >= 0 {
		name = name[:index]
	}
	if name == "policy.json" {
		return append([]byte(nil), policySchemaBytes...), nil
	}
	if name == "document.json" {
		return append([]byte(nil), documentSchemaBytes...), nil
	}
	if name == "site.json" {
		return append([]byte(nil), siteSchemaBytes...), nil
	}
	const outputPrefix = "output/"
	if !strings.HasPrefix(name, outputPrefix) {
		return nil, fmt.Errorf("embedded schema reference %q is not supported", reference)
	}
	name = strings.TrimPrefix(name, outputPrefix)
	if strings.ContainsAny(name, "/\\") || name == "" {
		return nil, fmt.Errorf("embedded schema reference %q is not a supported output schema", reference)
	}
	for kind, candidate := range outputSchemaNames {
		if candidate == name {
			return OutputSchema(kind)
		}
	}
	return nil, fmt.Errorf("unknown embedded schema reference %q", reference)
}
