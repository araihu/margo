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
	"github.com/araihu/margo/pdf"
	"github.com/araihu/margo/pdf/engines"
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

func TestDeckCommandUsesStaticEmbedProjectionFromTrustedPolicy(t *testing.T) {
	root := t.TempDir()
	policyPath := filepath.Join(root, "policy.json")
	policy := `{"schemaVersion":"margo-policy/v1","iframe":{"allowedOrigins":["https://video.example.com"],"projections":{"html":"interactive","pdf":"static-link","site":"interactive","deck":"static-link"}}}`
	if err := os.WriteFile(policyPath, []byte(policy), 0o600); err != nil {
		t.Fatal(err)
	}
	markdown := "# Architecture\n\n<iframe src=\"https://video.example.com/watch/123\" title=\"Architecture overview\"></iframe>\n"
	var stdout bytes.Buffer
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader(markdown), Stdout: &stdout, Stderr: io.Discard,
		Build: testBuildInfo(), WorkingDirectory: root,
	})
	command.SetArgs([]string{"deck", "-", "--policy", policyPath})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "<iframe") {
		t.Fatalf("deck contains interactive iframe")
	}
	for _, want := range []string{`class="margo-embed__link"`, `href="https://video.example.com/watch/123"`} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("deck missing %q", want)
		}
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
	// The fixture has two slides. Chromium emits one /Pages tree object plus
	// one /Page object per printed slide.
	if !bytes.HasPrefix(data, []byte("%PDF-")) || len(data) < 1000 || bytes.Count(data, []byte("/Type /Page")) < 3 {
		t.Fatalf("PDF bytes = %d page objects = %d", len(data), bytes.Count(data, []byte("/Type /Page")))
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

func TestDeckPDFUsesDocumentPagePreferenceUnlessCLIOverridesIt(t *testing.T) {
	engine := &capturingEngine{name: "native"}
	probe := engines.Probe{Native: func(context.Context) (pdf.Engine, engines.Candidate) {
		return engine, engines.Candidate{Name: "native", Version: "test", Compiled: true, Available: true}
	}}
	output := filepath.Join(t.TempDir(), "deck.pdf")
	markdown := "---\ntitle: Deck\nmargo:\n  page:\n    size: Letter\n    orientation: landscape\n---\n# One\n"
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader(markdown), EngineProbe: probe, Stderr: io.Discard,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
		NextExecutionID: func() margo.ExecutionID { return "cli-deck-page" },
	})
	command.SetArgs([]string{"deck", "-", "--format", "pdf", "--output", output, "--engine", "native"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if engine.request.Page.Size != pdf.PageLetter || engine.request.Page.Orientation != pdf.Landscape {
		t.Fatalf("page config = %+v", engine.request.Page)
	}
}
