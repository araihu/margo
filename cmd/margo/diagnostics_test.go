package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
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

func TestDiagnosticPreservesActionableMargoFields(t *testing.T) {
	failure := &margo.DiagnosticError{Diagnostics: []margo.Diagnostic{{
		Code: "source.metadata_invalid", Severity: margo.SeverityError,
		Source: "guide.md", Line: 2, Column: 3, Pointer: "/language",
		Message: "language is invalid", Hint: "Use a BCP 47 language tag.",
	}}}
	var output bytes.Buffer
	if err := writeDiagnostic(&output, diagnosticJSON, failure); err != nil {
		t.Fatal(err)
	}
	var diagnostic margo.Diagnostic
	if err := json.Unmarshal(output.Bytes(), &diagnostic); err != nil {
		t.Fatal(err)
	}
	if diagnostic.Source != "guide.md" || diagnostic.Line != 2 || diagnostic.Column != 3 || diagnostic.Pointer != "/language" || diagnostic.Hint == "" || diagnostic.Severity != margo.SeverityError {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}

	output.Reset()
	if err := writeDiagnostic(&output, diagnosticText, failure); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"guide.md:2:3", "error source.metadata_invalid", "[/language]", "hint: Use a BCP 47 language tag."} {
		if !strings.Contains(output.String(), fragment) {
			t.Fatalf("text diagnostic %q missing %q", output.String(), fragment)
		}
	}
}

func TestArgumentAndFlagFailuresHonorJSONDiagnostics(t *testing.T) {
	for _, args := range [][]string{
		{"html", "--diagnostics", "json"},
		{"html", "--diagnostics", "json", "--unknown"},
	} {
		var stdout, stderr bytes.Buffer
		command := NewRootCommand(Dependencies{Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr, Build: testBuildInfo()})
		command.SetArgs(args)
		if err := command.ExecuteContext(context.Background()); err == nil {
			t.Fatalf("margo %v unexpectedly succeeded", args)
		}
		var projection map[string]any
		if err := json.Unmarshal(stderr.Bytes(), &projection); err != nil {
			t.Fatalf("margo %v stderr is not JSON: %q (%v)", args, stderr.String(), err)
		}
		if code, _ := projection["code"].(string); code != "cli.arguments_invalid" && code != "cli.flag_invalid" {
			t.Fatalf("margo %v code = %q", args, code)
		}
		if stdout.Len() != 0 {
			t.Fatalf("margo %v stdout = %q", args, stdout.String())
		}
	}
}
