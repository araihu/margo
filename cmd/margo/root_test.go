package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRootHelpListsCompleteSurface(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
		Stderr: &stderr,
		Build:  testBuildInfo(),
	})
	cmd.SetArgs([]string{"--help"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"html", "pdf", "deck", "doctor", "version"} {
		if !strings.Contains(stdout.String(), name) {
			t.Fatalf("help missing %q", name)
		}
	}
}

func TestRootPlaceholderReportsUnimplementedCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(Dependencies{
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
		Stderr: &stderr,
		Build:  testBuildInfo(),
	})
	cmd.SetArgs([]string{"html"})
	err := cmd.ExecuteContext(context.Background())
	if err == nil || err.Error() != "margo.command_not_implemented: html" {
		t.Fatalf("error = %v", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q stderr = %q", stdout.String(), stderr.String())
	}
}
