package main

import (
	"bytes"
	"testing"

	margo "github.com/araihu/margo"
)

func TestSchemaCommandEmitsExactEmbeddedBytes(t *testing.T) {
	for _, kind := range []margo.SchemaKind{margo.SchemaPolicy, margo.SchemaDocument, margo.SchemaSite} {
		var output bytes.Buffer
		command := NewRootCommand(Dependencies{Stdout: &output})
		command.SetArgs([]string{"schema", string(kind)})
		if err := command.Execute(); err != nil {
			t.Fatalf("schema %s: %v", kind, err)
		}
		want, err := margo.Schema(kind)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(output.Bytes(), want) {
			t.Fatalf("schema %s output differs from embedded bytes", kind)
		}
	}
}
