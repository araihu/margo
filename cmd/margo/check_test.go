package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	margo "github.com/araihu/margo"
)

func TestCheckCommandReportsAllFindingsAsJSON(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "guide.md")
	if err := os.WriteFile(input, []byte("<span>raw</span>\n\n![](missing.png)\n\n[Guide](other.md)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	command := NewRootCommand(Dependencies{Stdout: &stdout, Stderr: &stderr, WorkingDirectory: root, Build: testBuildInfo()})
	command.SetArgs([]string{"check", input, "--diagnostics", "json"})
	err := command.ExecuteContext(context.Background())
	if cliDiagnosticCode(err) != "check.failed" {
		t.Fatalf("error = %v, stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	var report struct {
		Diagnostics []margo.Diagnostic `json:"diagnostics"`
		Errors      int                `json:"errors"`
		Warnings    int                `json:"warnings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, stdout.String())
	}
	if len(report.Diagnostics) != 5 || report.Errors != 2 || report.Warnings != 3 {
		t.Fatalf("report = %+v", report)
	}
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Source != input || diagnostic.Line == 0 || diagnostic.Pointer == "" || diagnostic.Hint == "" {
			t.Fatalf("incomplete diagnostic = %+v", diagnostic)
		}
	}
}

func TestCheckCommandSucceedsForCompatibleInput(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "clean.md")
	if err := os.WriteFile(input, []byte("---\nlanguage: en\n---\n\n# Clean\n\n[External](https://example.com)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	command := NewRootCommand(Dependencies{Stdout: &stdout, Stderr: &stderr, WorkingDirectory: root, Build: testBuildInfo()})
	command.SetArgs([]string{"check", input, "--diagnostics", "json"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "{\"diagnostics\":[],\"errors\":0,\"warnings\":0}\n" {
		t.Fatalf("stdout = %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
