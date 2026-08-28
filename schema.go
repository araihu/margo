package margo

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// SchemaKind identifies one public, version-matched configuration or output
// schema. Output schemas describe the stable JSON envelopes emitted by the
// CLI and runtime integrations.
type SchemaKind string

const (
	SchemaPolicy   SchemaKind = "policy"
	SchemaDocument SchemaKind = "document"
	SchemaSite     SchemaKind = "site"

	SchemaDiagnostic            SchemaKind = "diagnostic"
	SchemaDoctorReport          SchemaKind = "doctor-report"
	SchemaCheckReport           SchemaKind = "check-report"
	SchemaSiteReport            SchemaKind = "site-report"
	SchemaSiteManifest          SchemaKind = "site-manifest"
	SchemaRuntimeDescriptor     SchemaKind = "runtime-descriptor"
	SchemaRuntimeReport         SchemaKind = "runtime-report"
	SchemaDeckLayoutEvidence    SchemaKind = "deck-layout-evidence"
	SchemaDeckPDFArtifactReport SchemaKind = "deck-pdf-artifact-report"
)

const (
	policySchemaID   = "https://araihu.github.io/margo/schema/v1/policy.json"
	documentSchemaID = "https://araihu.github.io/margo/schema/v1/document.json"
	siteSchemaID     = "https://araihu.github.io/margo/schema/v1/site.json"
)

var (
	//go:embed schema/v1/policy.json
	policySchemaBytes []byte

	//go:embed schema/v1/document.json
	documentSchemaBytes []byte

	//go:embed schema/v1/site.json
	siteSchemaBytes []byte

	compiledSchemasOnce sync.Once
	compiledSchemas     map[SchemaKind]*jsonschema.Schema
	compiledSchemasErr  error
)

// Schema returns detached exact bytes shipped with this Margo version.
func Schema(kind SchemaKind) ([]byte, error) {
	var source []byte
	switch kind {
	case SchemaPolicy:
		source = policySchemaBytes
	case SchemaDocument:
		source = documentSchemaBytes
	case SchemaSite:
		source = siteSchemaBytes
	default:
		if output, err := OutputSchema(kind); err == nil {
			return output, nil
		}
		return nil, fmt.Errorf("schema.kind_invalid: unsupported schema %q", kind)
	}
	return append([]byte(nil), source...), nil
}

func validateJSONSchema(kind SchemaKind, input []byte) (any, error) {
	if err := rejectDuplicateJSONKeys(input); err != nil {
		return nil, err
	}
	value, err := jsonschema.UnmarshalJSON(bytes.NewReader(input))
	if err != nil {
		return nil, err
	}
	schema, err := compiledSchema(kind)
	if err != nil {
		return nil, err
	}
	if err := schema.Validate(value); err != nil {
		return nil, err
	}
	return value, nil
}

func compiledSchema(kind SchemaKind) (*jsonschema.Schema, error) {
	compiledSchemasOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		compiledSchemas = make(map[SchemaKind]*jsonschema.Schema, 3)
		for _, item := range []struct {
			kind SchemaKind
			id   string
			data []byte
		}{
			{kind: SchemaPolicy, id: policySchemaID, data: policySchemaBytes},
			{kind: SchemaDocument, id: documentSchemaID, data: documentSchemaBytes},
			{kind: SchemaSite, id: siteSchemaID, data: siteSchemaBytes},
		} {
			document, err := jsonschema.UnmarshalJSON(bytes.NewReader(item.data))
			if err != nil {
				compiledSchemasErr = fmt.Errorf("schema.compile_failed: %s: %w", item.kind, err)
				return
			}
			if err := compiler.AddResource(item.id, document); err != nil {
				compiledSchemasErr = fmt.Errorf("schema.compile_failed: %s: %w", item.kind, err)
				return
			}
		}
		for _, item := range []struct {
			kind SchemaKind
			id   string
		}{{SchemaPolicy, policySchemaID}, {SchemaDocument, documentSchemaID}, {SchemaSite, siteSchemaID}} {
			compiled, err := compiler.Compile(item.id)
			if err != nil {
				compiledSchemasErr = fmt.Errorf("schema.compile_failed: %s: %w", item.kind, err)
				return
			}
			compiledSchemas[item.kind] = compiled
		}
	})
	if compiledSchemasErr != nil {
		return nil, compiledSchemasErr
	}
	result := compiledSchemas[kind]
	if result == nil {
		return nil, fmt.Errorf("schema.kind_invalid: unsupported schema %q", kind)
	}
	return result, nil
}

func rejectDuplicateJSONKeys(input []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	if err := consumeUniqueJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func consumeUniqueJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("JSON object key must be a string")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate JSON property %q", key)
			}
			seen[key] = struct{}{}
			if err := consumeUniqueJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return fmt.Errorf("JSON object is not closed")
		}
	case '[':
		for decoder.More() {
			if err := consumeUniqueJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return fmt.Errorf("JSON array is not closed")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}
