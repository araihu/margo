package main

import (
	"bytes"
	"errors"
	"testing"
)

func TestJSONDiagnosticWritesOneObjectToStderr(t *testing.T) {
	var output bytes.Buffer
	if err := writeDiagnostic(&output, diagnosticJSON, errors.New("pdf.engine_unavailable: no engine")); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "{\"code\":\"pdf.engine_unavailable\",\"message\":\"no engine\"}\n"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestTextDiagnosticUsesStableCodeAndMessage(t *testing.T) {
	var output bytes.Buffer
	if err := writeDiagnostic(&output, diagnosticText, errors.New("cli.input_invalid: bad input")); err != nil {
		t.Fatal(err)
	}
	if output.String() != "cli.input_invalid: bad input\n" {
		t.Fatalf("output = %q", output.String())
	}
}

func TestDiagnosticRejectsUnknownFormat(t *testing.T) {
	err := writeDiagnostic(&bytes.Buffer{}, diagnosticFormat("yaml"), errors.New("failed"))
	if cliDiagnosticCode(err) != "cli.diagnostics_invalid" {
		t.Fatalf("error = %v", err)
	}
}
