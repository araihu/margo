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

func TestHTMLCommandOverridesTitleAndLanguage(t *testing.T) {
	var stdout bytes.Buffer
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader("# Original\n"), Stdout: &stdout,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
	})
	command.SetArgs([]string{"html", "-", "--title", "Published guide", "--lang", "pt-BR"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<title>Published guide</title>`, `<html lang="pt-BR"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("HTML override missing %q", want)
		}
	}
}

func TestHTMLCommandReportsInvalidLanguageAsActionableJSON(t *testing.T) {
	var stderr bytes.Buffer
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader("# Original\n"), Stderr: &stderr,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
	})
	command.SetArgs([]string{"html", "-", "--lang", "pt_BR", "--diagnostics", "json"})
	if err := command.ExecuteContext(context.Background()); cliDiagnosticCode(err) != "html.metadata_invalid" {
		t.Fatalf("error = %v", err)
	}
	for _, want := range []string{`"code":"html.metadata_invalid"`, `"pointer":"/language"`, `"hint":"Use a BCP 47 language tag`} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("diagnostic missing %q: %s", want, stderr.String())
		}
	}
}
