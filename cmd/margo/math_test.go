package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestMathFlagsAndCLIParity(t *testing.T) {
	for _, name := range []string{"check", "html", "pdf", "site", "serve", "deck"} {
		command := NewRootCommand(Dependencies{Build: testBuildInfo()})
		child, _, err := command.Find([]string{name})
		if err != nil {
			t.Fatal(err)
		}
		flag := child.Flags().Lookup("math")
		if flag == nil || flag.DefValue != "false" {
			t.Fatalf("%s: missing disabled math flag", name)
		}
	}
	for _, enabled := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		command := NewRootCommand(Dependencies{Stdin: strings.NewReader("# Math\n\n$x$\n\n```math\ny\n```\n"), Stdout: &stdout, Stderr: &stderr, Build: testBuildInfo(), WorkingDirectory: t.TempDir()})
		args := []string{"html", "-"}
		if enabled {
			args = append(args, "--math")
		}
		command.SetArgs(args)
		if err := command.ExecuteContext(context.Background()); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(stdout.String(), `data-margo-requirement="margo.math.engine"`) != enabled {
			t.Fatalf("math=%v: wrong dependencies", enabled)
		}
		if strings.Contains(stdout.String(), `class="margo-math margo-math--inline"`) != enabled {
			t.Fatalf("math=%v: wrong authoring interpretation", enabled)
		}
	}
}

func TestMathServeSettingPersists(t *testing.T) {
	var request serveRequest
	command := NewRootCommand(Dependencies{Build: testBuildInfo(), Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, ServeSite: func(_ context.Context, r serveRequest) error { request = r; return nil }})
	command.SetArgs([]string{"serve", "--math"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !request.Math {
		t.Fatal("serve request lost math configuration")
	}
}

func TestMathCheckWarnsWithoutBrowser(t *testing.T) {
	var stdout, stderr bytes.Buffer
	command := NewRootCommand(Dependencies{Stdin: strings.NewReader("---\nlanguage: en\n---\n\n# Math\n\n$\\unknowncommand$\n"), Stdout: &stdout, Stderr: &stderr, Build: testBuildInfo(), WorkingDirectory: t.TempDir()})
	command.SetArgs([]string{"check", "-", "--math", "--diagnostics", "json"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "check.math_runtime_validation_required") {
		t.Fatal(stdout.String())
	}
}
