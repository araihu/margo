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
	for _, name := range []string{"check", "html", "pdf", "deck", "doctor", "version"} {
		if !strings.Contains(stdout.String(), name) {
			t.Fatalf("help missing %q", name)
		}
	}
}
