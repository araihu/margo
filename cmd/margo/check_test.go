package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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

func TestCheckDeckRunsDeckParser(t *testing.T) {
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader("<!-- class: unsupported -->\n# One\n"), Stdout: io.Discard, Stderr: io.Discard,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
	})
	command.SetArgs([]string{"check", "-", "--target", "deck"})
	if err := command.ExecuteContext(context.Background()); cliDiagnosticCode(err) != "deck.class_unsupported" {
		t.Fatalf("error = %v", err)
	}
}

func TestCheckCommandAppliesTrustedPolicyAndReportsIdentity(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "policy.json")
	policy := `{
  "schemaVersion":"margo-policy/v1",
  "rawHTML":"sanitized",
  "iframe":{
    "allowedOrigins":["https://video.example.com"],
    "projections":{"html":"interactive","pdf":"static-link","site":"deny","deck":"deny"}
  }
}`
	if err := os.WriteFile(policyPath, []byte(policy), 0o600); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(root, "trusted.md")
	markdown := "---\nlanguage: en\n---\n\n<span>trusted text</span>\n\n<iframe src=\"https://video.example.com/watch/123\" title=\"Architecture overview\"></iframe>\n"
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
	if report.Policy != "sha256:b8924e7eb3dfaedfc66bfad18b1b75f96b445c0026f85ac44243a064857f63e7" {
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

func TestCheckCommandRejectsRawHTMLOutsideSanitizedAllowlist(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "policy.json")
	input := filepath.Join(root, "unsafe.md")
	if err := os.WriteFile(policyPath, []byte(`{"schemaVersion":"margo-policy/v1","rawHTML":"sanitized"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	markdown := "---\nlanguage: en\n---\n\n<script>alert(1)</script>\n"
	if err := os.WriteFile(input, []byte(markdown), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	command := NewRootCommand(Dependencies{Stdout: &stdout, Stderr: &bytes.Buffer{}, Build: testBuildInfo()})
	command.SetArgs([]string{"check", input, "--policy", policyPath, "--diagnostics", "json"})
	if err := command.ExecuteContext(context.Background()); err == nil {
		t.Fatal("unsafe raw HTML passed policy-aware check")
	}
	var report checkReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("report: %v: %s", err, stdout.String())
	}
	if report.Errors != 1 || len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != "policy.html.invalid" {
		t.Fatalf("report = %+v", report)
	}
}

func TestCheckCommandRejectsUnsafeRawHTMLBlockUnderPolicy(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "policy.json")
	input := filepath.Join(root, "undeclared.md")
	if err := os.WriteFile(policyPath, []byte(`{"schemaVersion":"margo-policy/v1","rawHTML":"sanitized"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, []byte("---\nlanguage: en\n---\n\n<script>alert(1)</script>\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	command := NewRootCommand(Dependencies{Stdout: &stdout, Stderr: &bytes.Buffer{}, Build: testBuildInfo()})
	command.SetArgs([]string{"check", input, "--policy", policyPath, "--diagnostics", "json"})
	if err := command.ExecuteContext(context.Background()); err == nil {
		t.Fatal("undeclared HTML block passed policy-aware check")
	}
	var report checkReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("report: %v: %s", err, stdout.String())
	}
	if report.Errors != 1 || len(report.Diagnostics) != 1 || report.Diagnostics[0].Code != "policy.html.invalid" {
		t.Fatalf("report = %+v", report)
	}
}
