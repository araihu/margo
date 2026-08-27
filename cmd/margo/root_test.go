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
	for _, name := range []string{"check", "html", "site", "serve", "pdf", "deck", "doctor", "version", "schema", "help", "completion"} {
		if !strings.Contains(stdout.String(), name) {
			t.Fatalf("help missing %q", name)
		}
	}
}

func TestCommandHelpIncludesCopyableWorkflowExamples(t *testing.T) {
	cases := map[string]string{
		"check":  "--target site",
		"html":   "--output build/guide.html",
		"site":   "--output-dir ./build/site",
		"serve":  "--host 127.0.0.1 --port 8080",
		"pdf":    "--base-url https://docs.example.com/guide/",
		"deck":   "--slide-size 16:9",
		"doctor": "--diagnostics json",
		"schema": "margo schema site",
	}
	for commandName, expected := range cases {
		t.Run(commandName, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cmd := NewRootCommand(Dependencies{
				Stdin:  strings.NewReader(""),
				Stdout: &stdout,
				Stderr: &stderr,
				Build:  testBuildInfo(),
			})
			cmd.SetArgs([]string{commandName, "--help"})
			if err := cmd.ExecuteContext(context.Background()); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stdout.String(), expected) {
				t.Fatalf("help missing copyable example %q: %s", expected, stdout.String())
			}
		})
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
