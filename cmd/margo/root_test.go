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
	for _, name := range []string{"check", "html", "site", "pdf", "deck", "doctor", "version", "help", "completion"} {
		if !strings.Contains(stdout.String(), name) {
			t.Fatalf("help missing %q", name)
		}
	}
}

func TestCompletionShellHelpExposesNoDescriptions(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cmd := NewRootCommand(Dependencies{
				Stdin:  strings.NewReader(""),
				Stdout: &stdout,
				Stderr: &stderr,
				Build:  testBuildInfo(),
			})
			cmd.SetArgs([]string{"completion", shell, "--help"})
			if err := cmd.ExecuteContext(context.Background()); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stdout.String(), "--no-descriptions") {
				t.Fatalf("completion %s help omits --no-descriptions: %s", shell, stdout.String())
			}
		})
	}
}
