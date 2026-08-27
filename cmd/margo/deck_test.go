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

func TestDeckAndCheckRejectRecognizedLegacyDollarDirectives(t *testing.T) {
	for _, commandArgs := range [][]string{
		{"deck", "-", "--diagnostics", "json"},
		{"check", "-", "--target", "deck", "--diagnostics", "json"},
	} {
		t.Run(commandArgs[0], func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			command := NewRootCommand(Dependencies{
				Stdin:  strings.NewReader("<!-- $paginate: true -->\n# One\n"),
				Stdout: &stdout, Stderr: &stderr, Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
			})
			command.SetArgs(commandArgs)
			if err := command.ExecuteContext(context.Background()); cliDiagnosticCode(err) != "deck.directive_invalid" {
				t.Fatalf("error = %v, stdout=%s stderr=%s", err, stdout.String(), stderr.String())
			}
			var diagnostic struct {
				Code string `json:"code"`
				Hint string `json:"hint"`
			}
			if err := json.Unmarshal(stderr.Bytes(), &diagnostic); err != nil {
				t.Fatalf("diagnostic JSON = %q: %v", stderr.String(), err)
			}
			if diagnostic.Code != "deck.directive_invalid" {
				t.Fatalf("diagnostic code = %q", diagnostic.Code)
			}
			if diagnostic.Hint != "Use the unprefixed `paginate` directive." {
				t.Fatalf("diagnostic hint = %q", diagnostic.Hint)
			}
			if stdout.Len() != 0 {
				t.Fatalf("failed %s published %d stdout bytes", commandArgs[0], stdout.Len())
			}
		})
	}
}

func TestDeckRejectsInteractiveChartWithTargetDiagnostic(t *testing.T) {
	input := "---\nlanguage: en\n---\n\n# Revenue\n\n```goshtosochart\nschemaVersion: 1\ntype: line\nrenderer: interactive\ntitle: Weekly revenue\ncategories: [Mon, Tue]\nseries:\n  - name: Revenue\n    values: [12, 18]\n```\n"
	var stdout, stderr bytes.Buffer
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader(input), Stdout: &stdout, Stderr: &stderr,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
	})
	command.SetArgs([]string{"deck", "-", "--diagnostics", "json"})
	if err := command.ExecuteContext(context.Background()); cliDiagnosticCode(err) != "chart.renderer_target_unsupported" {
		t.Fatalf("error = %v, stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), `"code":"chart.renderer_target_unsupported"`) ||
		!strings.Contains(stderr.String(), `"pointer":"/renderer"`) ||
		!strings.Contains(stderr.String(), "set renderer: static") {
		t.Fatalf("target-specific chart diagnostic missing: %s", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q; failed deck must not publish bytes", stdout.String())
	}
}

func TestDeckConfidentialityBadgeFlagAddsHostChrome(t *testing.T) {
	var stdout bytes.Buffer
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader("<!-- paginate: true -->\n# One\n"), Stdout: &stdout, Stderr: io.Discard,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
	})
	command.SetArgs([]string{"deck", "-", "--confidentiality-badge", "Confidencial"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `margo-deck__confidentiality-badge`) {
		t.Fatal("deck confidentiality badge missing")
	}
}

func TestDeckPaginationIconFlagsAddHostChrome(t *testing.T) {
	var stdout bytes.Buffer
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader("<!-- paginate: true -->\n# One\n"), Stdout: &stdout, Stderr: io.Discard,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
	})
	command.SetArgs([]string{
		"deck", "-", "--pagination-icon", "hi-16-solid-clock",
		"--pagination-icon-placement", "before", "--pagination-icon-decorative",
	})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	markup := stdout.String()
	if !strings.Contains(markup, `href="#hi-16-solid-clock"`) || !strings.Contains(markup, `<symbol id="hi-16-solid-clock"`) {
		t.Fatalf("deck pagination icon missing: %s", markup)
	}
}

func TestDeckPrintChartDataFlagProjectsAccessibleTable(t *testing.T) {
	markdown := "# Chart\n\n```goshtosochart\nschemaVersion: 1\ntype: bar\ntitle: Revenue\ncategories: [Q1]\nseries:\n  - name: Actual\n    values: [12]\n```\n"
	var stdout bytes.Buffer
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader(markdown), Stdout: &stdout, Stderr: io.Discard,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
	})
	command.SetArgs([]string{"deck", "-", "--print-chart-data"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `data-margo-chart-print-data="enabled"`) {
		t.Fatal("deck did not enable printable chart data")
	}
}

func TestDeckPDFPrintChartDataFitsSidebarRows(t *testing.T) {
	browser := installedCLITestChromium()
	if browser == "" {
		t.Skip("installed Chromium unavailable")
	}
	markdown := `---
title: Chart
language: en
size: 16:9
---

<!-- _class: sidebar -->
<!-- layout: sidebar -->
<!-- slot: main -->
## Revenue

` + "```goshtosochart\n" + `schemaVersion: 1
type: bar
title: Revenue by motion
categories: [W1, W2, W3, W4, W5, W6]
series:
  - name: New logos
    values: [4, 7, 9, 12, 15, 18]
  - name: Expansion
    values: [4, 6, 8, 10, 13, 16]
` + "```\n" + `<!-- slot: rail -->
### Readout

The exact values remain available for review and print.
<!-- /layout -->
`
	output := filepath.Join(t.TempDir(), "chart.pdf")
	var stderr bytes.Buffer
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader(markdown), Stderr: &stderr,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
		NextExecutionID: func() margo.ExecutionID { return "cli-deck-chart-overflow" },
	})
	command.SetArgs([]string{
		"deck", "-", "--format", "pdf", "--output", output,
		"--slide-size", "16:9", "--print-chart-data",
		"--engine", "chromium", "--engine-path", browser,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := command.ExecuteContext(ctx)
	if err != nil {
		t.Fatalf("sidebar chart-data PDF failed: %v, stderr = %s", err, stderr.String())
	}
	data, readErr := os.ReadFile(output)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) || bytes.Count(data, []byte("/Type /Page")) < 2 {
		t.Fatalf("sidebar chart-data PDF bytes = %d page objects = %d", len(data), bytes.Count(data, []byte("/Type /Page")))
	}
}

func TestDeckHTMLOmitsBrowserChartControls(t *testing.T) {
	markdown := "# Chart\n\n```goshtosochart\nschemaVersion: 1\ntype: bar\ntitle: Revenue\ncategories: [Q1]\nseries:\n  - name: Actual\n    values: [12]\n```\n"
	var stdout bytes.Buffer
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader(markdown), Stdout: &stdout, Stderr: io.Discard,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
	})
	command.SetArgs([]string{"deck", "-"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	markup := stdout.String()
	for _, forbidden := range []string{
		`data-goshtoso-chart-wrapper-mode`,
		`chart-expand-export-copy-action`,
		`exportFromMenu`,
	} {
		if strings.Contains(markup, forbidden) {
			t.Fatalf("deck chart controls leaked %q", forbidden)
		}
	}
	for _, required := range []string{`data-margo-chart-data="v1"`, `<svg`, `<table`} {
		if !strings.Contains(markup, required) {
			t.Fatalf("deck chart missing %q", required)
		}
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
	browser := installedCLITestChromium()
	if browser == "" {
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
	if err := command.ExecuteContext(context.Background()); cliDiagnosticCode(err) != "cli.deck_validator_unavailable" {
		t.Fatalf("error = %v", err)
	}
}

func TestDeckPDFRequiresChromiumForSlideGeometry(t *testing.T) {
	engine := &capturingEngine{name: "native"}
	probe := engines.Probe{Native: func(context.Context) (pdf.Engine, engines.Candidate) {
		return engine, engines.Candidate{Name: "native", Version: "test", Compiled: true, Available: true}
	}}
	output := filepath.Join(t.TempDir(), "deck.pdf")
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader("# One\n"), EngineProbe: probe, Stderr: io.Discard,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
		NextExecutionID: func() margo.ExecutionID { return "cli-deck-geometry" },
	})
	command.SetArgs([]string{"deck", "-", "--format", "pdf", "--output", output, "--engine", "native", "--slide-size", "4:3"})
	if err := command.ExecuteContext(context.Background()); cliDiagnosticCode(err) != "cli.deck_validator_unavailable" {
		t.Fatalf("error = %v", err)
	}
}

func TestDeckRejectsImplicitCustomGeometry(t *testing.T) {
	for _, args := range [][]string{
		{"--slide-width", "1280", "--slide-height", "720"},
		{"--slide-size", "custom", "--slide-width", "1280", "--slide-height", "720"},
	} {
		command := NewRootCommand(Dependencies{
			Stdin: strings.NewReader("# One\n"), Stderr: io.Discard,
			Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
		})
		command.SetArgs(append([]string{"deck", "-"}, args...))
		if err := command.ExecuteContext(context.Background()); cliDiagnosticCode(err) != "cli.deck_geometry_invalid" {
			t.Fatalf("args %v error = %v", args, err)
		}
	}
}

func TestDeckRejectsConflictingFrontmatterAndLegacyPageGeometry(t *testing.T) {
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader("---\nsize: 4:3\n---\n# One\n"), Stderr: io.Discard,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
		NextExecutionID: func() margo.ExecutionID { return "cli-deck-conflict" },
	})
	command.SetArgs([]string{"deck", "-", "--format", "pdf", "--output", filepath.Join(t.TempDir(), "deck.pdf"), "--engine", "native", "--page-size", "A4"})
	if err := command.ExecuteContext(context.Background()); cliDiagnosticCode(err) != "cli.deck_geometry_conflict" {
		t.Fatalf("error = %v", err)
	}
}
