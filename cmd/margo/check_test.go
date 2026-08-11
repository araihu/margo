package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestCheckCommandAppliesTrustedPolicyAndReportsIdentity(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "policy.json")
	policy := `{
  "schemaVersion":"margo-policy/v1",
  "rawHTML":"sanitized",
  "trustedEmbeds":{
    "allowedKinds":["iframe"],
    "allowedOrigins":["https://video.example.com"],
    "projections":{"html":"interactive","pdf":"static-link","site":"deny","deck":"deny"}
  }
}`
	if err := os.WriteFile(policyPath, []byte(policy), 0o600); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(root, "trusted.md")
	markdown := "---\nlanguage: en\ngoshtoso:\n  security:\n    rawHTML: sanitized\n---\n\n<span>trusted text</span>\n\n```trusted-embed\nkind: iframe\nurl: https://video.example.com/watch/123\ntitle: Architecture overview\n```\n"
	if err := os.WriteFile(input, []byte(markdown), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	command := NewRootCommand(Dependencies{Stdout: &stdout, Stderr: &stderr, WorkingDirectory: root, Build: testBuildInfo()})
	command.SetArgs([]string{"check", input, "--policy", policyPath, "--diagnostics", "json"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Diagnostics []margo.Diagnostic `json:"diagnostics"`
		Policy      string             `json:"policy"`
		Errors      int                `json:"errors"`
		Warnings    int                `json:"warnings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 0 || report.Errors != 0 || report.Warnings != 0 {
		t.Fatalf("report = %+v", report)
	}
	if report.Policy != "sha256:49a3bcdab2cd761114678d38d6fecec223d6b0375a97b3c8f15d90321989d810" {
		t.Fatalf("policy identity = %q", report.Policy)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestWriteCheckReportIncludesPolicyIdentityInText(t *testing.T) {
	var output bytes.Buffer
	const digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := writeCheckReport(&output, diagnosticText, checkReport{Policy: digest}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "policy "+digest) {
		t.Fatalf("text report = %q", output.String())
	}
}
