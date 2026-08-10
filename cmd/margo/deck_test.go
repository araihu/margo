package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	margo "github.com/araihu/margo"
)

func TestDeckDefaultsToHTMLStdout(t *testing.T) {
	var stdout bytes.Buffer
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader("# One\n---\n# Two\n"), Stdout: &stdout, Stderr: io.Discard,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
	})
	command.SetArgs([]string{"deck", "-"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `class="margo-deck"`) || !strings.Contains(stdout.String(), "Slide 2 of 2") {
		t.Fatal("deck HTML missing")
	}
}

func TestDeckPDFExecutesMermaidAndKeepsStaticChart(t *testing.T) {
	browser := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	if _, err := os.Stat(browser); err != nil {
		t.Skip("installed Chromium unavailable")
	}
	input := "# Runtime\n\n```mermaid\ngraph TD; A-->B\n```\n---\n# Chart\n\n```goshtosochart\n" +
		"schemaVersion: 1\ntype: bar\ntitle: Revenue\ncategories: [Q1]\n" +
		"series:\n  - name: Actual\n    values: [12]\n```\n"
	output := filepath.Join(t.TempDir(), "deck.pdf")
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader(input), Stderr: io.Discard,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
		NextExecutionID: func() margo.ExecutionID { return "cli-deck-e2e" },
	})
	command.SetArgs([]string{"deck", "-", "--format", "pdf", "--output", output, "--engine", "chromium", "--engine-path", browser})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := command.ExecuteContext(ctx); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) || len(data) < 1000 {
		t.Fatalf("PDF bytes = %d", len(data))
	}
}

func TestDeckPDFRequiresOutput(t *testing.T) {
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader("# One\n"), Stdout: io.Discard, Stderr: io.Discard,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
	})
	command.SetArgs([]string{"deck", "-", "--format", "pdf"})
	if err := command.ExecuteContext(context.Background()); cliDiagnosticCode(err) != "cli.output_required" {
		t.Fatalf("error = %v", err)
	}
}
