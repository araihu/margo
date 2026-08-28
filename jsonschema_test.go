package margo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONSchemaFenceRendersInlinePropertyTree(t *testing.T) {
	compiler := New(WithHostPolicy(Policy{RawHTML: RawHTMLDeny, OutputBytes: MaxOutputBytes}))
	source := "# Contract\n\n```jsonschema\n" + `{"title":"Example contract","description":"Fields sent by the worker.","type":"object","required":["id"],"properties":{"id":{"type":"string","description":"Stable identifier."},"count":{"type":"integer","minimum":0}}}` + "\n```\n"
	document, err := compiler.Compile(context.Background(), Source{Name: "contract.md", Content: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, result.Content())
	for _, want := range []string{
		`class="margo-jsonschema"`,
		`aria-label="Example contract"`,
		`class="margo-jsonschema__tree"`,
		`<code class="margo-jsonschema__tree-path" title="/count">count</code>`,
		`<code class="margo-jsonschema__tree-path" title="/id">id</code>`,
		`class="margo-jsonschema__tree-type">integer</span>`,
		`class="margo-jsonschema__tree-required" title="required" aria-label="required">*</span>`,
		`minimum=0`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("JSON Schema rendering missing %q:\n%s", want, markup)
		}
	}
	if strings.Contains(markup, "<table") {
		t.Fatalf("JSON Schema renderer still emits a table:\n%s", markup)
	}
	if strings.Contains(markup, ">optional<") {
		t.Fatalf("optional properties should not render a status label:\n%s", markup)
	}
}

func TestJSONSchemaFenceResolvesContainedPathReference(t *testing.T) {
	root := t.TempDir()
	schemaPath := filepath.Join(root, "contract.json")
	if err := os.WriteFile(schemaPath, []byte(`{"title":"File contract","type":"object","properties":{"ready":{"type":"boolean"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	source := Source{Name: "contract.md", BaseURL: root, Content: []byte("# Contract\n\n```jsonschema ref=contract.json\n```\n")}
	compiler := New()
	document, err := compiler.Compile(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, result.Content())
	if !strings.Contains(markup, `data-margo-jsonschema-reference="contract.json"`) || !strings.Contains(markup, `title="/ready"`) {
		t.Fatalf("path-referenced schema was not rendered:\n%s", markup)
	}
}

func TestJSONSchemaFenceAcceptsQuotedPathAndFragment(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "schema with spaces.json")
	if err := os.WriteFile(path, []byte(`{"$defs":{"payload":{"title":"Payload","type":"object","properties":{"token":{"type":"string"}}}},"type":"object"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	source := Source{Name: "contract.md", BaseURL: root, Content: []byte("---\nlanguage: en\n---\n\n```jsonschema ref='schema with spaces.json#/$defs/payload'\n```\n")}
	compiler := New()
	document, err := compiler.Compile(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, result.Content())
	if !strings.Contains(markup, `data-margo-jsonschema-reference="schema with spaces.json#/$defs/payload"`) || !strings.Contains(markup, `title="/token"`) {
		t.Fatalf("quoted fragment schema was not rendered:\n%s", markup)
	}
}

func TestJSONSchemaFenceStopsRecursiveReferences(t *testing.T) {
	source := Source{Name: "tree.md", Content: []byte("```jsonschema\n" + `{"title":"Tree","type":"object","properties":{"name":{"type":"string"},"children":{"type":"array","items":{"$ref":"#"}}}}` + "\n```\n")}
	compiler := New()
	document, err := compiler.Compile(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, result.Content())
	if strings.Count(markup, `class="margo-jsonschema"`) != 1 || !strings.Contains(markup, `title="/children/*/name"`) {
		t.Fatalf("recursive schema did not render a bounded tree:\n%s", markup)
	}
}

func TestJSONSchemaFenceResolvesEmbeddedOutputSchema(t *testing.T) {
	source := Source{Name: "doctor.md", Content: []byte("```jsonschema ref=margo://schema/v1/output/doctor-report.json\n```\n")}
	compiler := New()
	document, err := compiler.Compile(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, result.Content())
	if !strings.Contains(markup, `aria-label="Margo doctor report v1"`) || !strings.Contains(markup, `title="/candidates/*/name"`) {
		t.Fatalf("embedded output schema was not rendered:\n%s", markup)
	}
}

func TestJSONSchemaFenceResolvesEmbeddedConfigSchema(t *testing.T) {
	source := Source{Name: "policy.md", Content: []byte("```jsonschema ref=margo://schema/v1/policy.json\n```\n")}
	compiler := New()
	document, err := compiler.Compile(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, result.Content())
	if !strings.Contains(markup, `aria-label="Margo host policy v1"`) || !strings.Contains(markup, `title="/schemaVersion"`) {
		t.Fatalf("embedded config schema was not rendered:\n%s", markup)
	}
}

func TestCheckValidatesJSONSchemaFenceReference(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "bad.json")
	if err := os.WriteFile(path, []byte(`{"type": "wat"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	diagnostics, err := Check(context.Background(), Source{Name: "doc.md", BaseURL: root, Content: []byte("---\nlanguage: en\n---\n\n```jsonschema ref=bad.json\n```\n")}, WithCheckAssetReader(FilesystemCheckAssetReader{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "jsonschema.schema_invalid" {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestOutputSchemasAreValidJSONSchemaDocuments(t *testing.T) {
	for _, kind := range []SchemaKind{
		SchemaDiagnostic, SchemaDoctorReport, SchemaCheckReport, SchemaSiteReport,
		SchemaSiteManifest, SchemaRuntimeDescriptor, SchemaRuntimeReport,
		SchemaDeckLayoutEvidence, SchemaDeckPDFArtifactReport,
	} {
		data, err := OutputSchema(kind)
		if err != nil {
			t.Fatalf("OutputSchema(%s): %v", kind, err)
		}
		if !strings.HasSuffix(string(data), "\n") {
			t.Errorf("%s schema has no final newline", kind)
		}
		if _, err := parseAndValidateJSONSchema(data); err != nil {
			t.Errorf("%s schema does not compile: %v", kind, err)
		}
	}
}
