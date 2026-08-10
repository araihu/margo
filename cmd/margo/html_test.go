package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestHTMLCommandRendersChartsToStdout(t *testing.T) {
	input := "# Report\n\n```goshtosochart\n" +
		"schemaVersion: 1\ntype: bar\ntitle: Revenue\ncategories: [Q1]\n" +
		"series:\n  - name: Actual\n    values: [12]\n```\n"
	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Stdin: strings.NewReader(input), Stdout: &stdout, Stderr: &stderr,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
	})
	cmd.SetArgs([]string{"html", "-"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"<!doctype html>", "<h1", ">Report</h1>", "<svg", "<table"} {
		if !strings.Contains(strings.ToLower(stdout.String()), strings.ToLower(fragment)) {
			t.Fatalf("missing %q", fragment)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestHTMLCommandRefusesExistingOutput(t *testing.T) {
	output := t.TempDir() + "/page.html"
	deps := Dependencies{Stdin: strings.NewReader("# Page\n"), Build: testBuildInfo(), WorkingDirectory: t.TempDir()}
	cmd := NewRootCommand(deps)
	cmd.SetArgs([]string{"html", "-", "--output", output})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	cmd = NewRootCommand(deps)
	cmd.SetArgs([]string{"html", "-", "--output", output})
	if err := cmd.ExecuteContext(context.Background()); cliDiagnosticCode(err) != "margo.atomic.destination_exists" {
		t.Fatalf("error = %v", err)
	}
}
