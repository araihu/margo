package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/pdf"
	"github.com/araihu/margo/pdf/engines"
)

func TestSingleFileCommandsReportActionableExistingOutputDiagnostics(t *testing.T) {
	root := t.TempDir()
	engine := &capturingEngine{name: "native"}
	probe := engines.Probe{Native: func(context.Context) (pdf.Engine, engines.Candidate) {
		return engine, engines.Candidate{Name: "native", Version: "test", Compiled: true, Available: true}
	}}

	tests := []struct {
		name         string
		path         string
		diagnostics  diagnosticFormat
		makeArgs     func(string) []string
		needsEngines bool
	}{
		{
			name:        "html text",
			path:        filepath.Join(root, "page.html"),
			diagnostics: diagnosticText,
			makeArgs: func(output string) []string {
				return []string{"html", "-", "--output", output}
			},
		},
		{
			name:        "deck html json",
			path:        filepath.Join(root, "deck.html"),
			diagnostics: diagnosticJSON,
			makeArgs: func(output string) []string {
				return []string{"deck", "-", "--format", "html", "--output", output}
			},
		},
		{
			name:         "pdf json",
			path:         filepath.Join(root, "page.pdf"),
			diagnostics:  diagnosticJSON,
			needsEngines: true,
			makeArgs: func(output string) []string {
				return []string{"pdf", "-", "--output", output, "--engine", "native"}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer
			deps := Dependencies{
				Stdin: strings.NewReader("# Output\n"), Stderr: &stderr,
				Build: testBuildInfo(), WorkingDirectory: root,
				NextExecutionID: func() margo.ExecutionID { return "destination-exists-test" },
			}
			if test.needsEngines {
				deps.EngineProbe = probe
			}

			first := NewRootCommand(deps)
			first.SetArgs(test.makeArgs(test.path))
			if err := first.ExecuteContext(context.Background()); err != nil {
				t.Fatalf("first command: %v", err)
			}

			stderr.Reset()
			secondDeps := deps
			secondDeps.Stdin = strings.NewReader("# Output\n")
			second := NewRootCommand(secondDeps)
			args := append(test.makeArgs(test.path), "--diagnostics", string(test.diagnostics))
			second.SetArgs(args)
			err := second.ExecuteContext(context.Background())
			if cliDiagnosticCode(err) != "margo.atomic.destination_exists" {
				t.Fatalf("second command error = %v", err)
			}
			if !strings.Contains(stderr.String(), test.path) || !strings.Contains(stderr.String(), "--force") || !strings.Contains(stderr.String(), "new destination") {
				t.Fatalf("diagnostic = %q", stderr.String())
			}
			if test.diagnostics == diagnosticJSON {
				var projection diagnosticProjection
				if err := json.Unmarshal(stderr.Bytes(), &projection); err != nil {
					t.Fatalf("JSON diagnostic = %q: %v", stderr.String(), err)
				}
				if projection.Code != "margo.atomic.destination_exists" || !strings.Contains(projection.Message, test.path) {
					t.Fatalf("JSON projection = %+v", projection)
				}
			}
		})
	}
}
