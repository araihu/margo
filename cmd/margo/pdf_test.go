package main

import (
	"bytes"
	"context"
	"errors"
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

func TestPDFRequiresExplicitOutput(t *testing.T) {
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader("# Page\n"), Stdout: io.Discard, Stderr: io.Discard,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
	})
	command.SetArgs([]string{"pdf", "-"})
	if err := command.ExecuteContext(context.Background()); cliDiagnosticCode(err) != "cli.output_required" {
		t.Fatalf("error = %v", err)
	}
}

func TestPDFLinkFlagsUseSafeDefaultsAndExplicitResolution(t *testing.T) {
	tests := []struct {
		name           string
		flags          pdfLinkFlags
		policyExplicit bool
		wantPolicy     pdf.RelativeLinkPolicy
		wantBase       string
		wantCode       string
	}{
		{name: "default strips", flags: pdfLinkFlags{Policy: "strip"}, wantPolicy: pdf.RelativeLinksStrip},
		{
			name:       "base URL implies resolution",
			flags:      pdfLinkFlags{Policy: "strip", BaseURL: "https://docs.example.com/manual/"},
			wantPolicy: pdf.RelativeLinksResolve, wantBase: "https://docs.example.com/manual/",
		},
		{
			name:  "explicit resolution",
			flags: pdfLinkFlags{Policy: "resolve", BaseURL: "https://docs.example.com/manual/"}, policyExplicit: true,
			wantPolicy: pdf.RelativeLinksResolve, wantBase: "https://docs.example.com/manual/",
		},
		{
			name:  "resolution requires base",
			flags: pdfLinkFlags{Policy: "resolve"}, policyExplicit: true,
			wantCode: "cli.relative_link_base_required",
		},
		{
			name:  "explicit non-resolution rejects unused base",
			flags: pdfLinkFlags{Policy: "strip", BaseURL: "https://docs.example.com/manual/"}, policyExplicit: true,
			wantCode: "cli.relative_link_options_invalid",
		},
		{
			name:  "unknown policy",
			flags: pdfLinkFlags{Policy: "surprise"}, policyExplicit: true,
			wantCode: "cli.relative_link_policy_invalid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.flags.config(test.policyExplicit)
			if test.wantCode != "" {
				if cliDiagnosticCode(err) != test.wantCode {
					t.Fatalf("error = %v, want %s", err, test.wantCode)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Policy != test.wantPolicy || got.BaseURL != test.wantBase {
				t.Fatalf("link config = %+v, want policy=%q base=%q", got, test.wantPolicy, test.wantBase)
			}
		})
	}
}

func TestPDFCommandExportsWithInstalledChromium(t *testing.T) {
	path := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	if _, err := os.Stat(path); err != nil {
		t.Skip("installed Chromium unavailable")
	}
	output := filepath.Join(t.TempDir(), "page.pdf")
	var stderr bytes.Buffer
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader("# PDF command E2E\n"), Stderr: &stderr,
		Build: testBuildInfo(), WorkingDirectory: t.TempDir(),
		NextExecutionID: func() margo.ExecutionID { return "cli-pdf-e2e" },
	})
	command.SetArgs([]string{"pdf", "-", "--output", output, "--engine", "chromium", "--engine-path", path})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := command.ExecuteContext(ctx); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) || len(data) < 1000 || stderr.Len() != 0 {
		t.Fatalf("PDF bytes = %d stderr = %q", len(data), stderr.String())
	}
}

func TestPDFRenderFailureDoesNotFallbackToNative(t *testing.T) {
	directory := t.TempDir()
	path := testBrowserFile(t, directory, "chromium")
	chromium := &countingEngine{name: "chromium", exportErr: errors.New("pdf.render_failed: boom")}
	native := &countingEngine{name: "native"}
	probe := engines.Probe{
		LookupEnv: func(name string) (string, bool) {
			if name == "MARGO_CHROMIUM_PATH" {
				return path, true
			}
			return "", false
		},
		LookPath: func(string) (string, error) { return "", os.ErrNotExist }, Stat: os.Stat,
		ChromiumVersion: func(context.Context, string) (string, error) { return "Chromium test", nil },
		ChromiumEngine:  func(string) (pdf.Engine, error) { return chromium, nil },
		Native: func(context.Context) (pdf.Engine, engines.Candidate) {
			return native, engines.Candidate{Name: "native", Version: "native-test", Compiled: true, Available: true}
		},
		KnownPaths: []string{}, GOOS: "linux",
	}
	output := filepath.Join(directory, "missing.pdf")
	command := NewRootCommand(Dependencies{
		Stdin: strings.NewReader("# Failure\n"), EngineProbe: probe, Stderr: io.Discard,
		Build: testBuildInfo(), WorkingDirectory: directory,
		NextExecutionID: func() margo.ExecutionID { return "cli-no-fallback" },
	})
	command.SetArgs([]string{"pdf", "-", "--output", output})
	if err := command.ExecuteContext(context.Background()); cliDiagnosticCode(err) != "pdf.render_failed" {
		t.Fatalf("error = %v", err)
	}
	if chromium.exports != 1 || native.exports != 0 {
		t.Fatalf("chromium exports = %d native exports = %d", chromium.exports, native.exports)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("output stat error = %v", err)
	}
}

type countingEngine struct {
	name      string
	exportErr error
	exports   int
}

func (engine *countingEngine) Name() string { return engine.name }
func (engine *countingEngine) Version(context.Context) (string, error) {
	return engine.name + "-test", nil
}
func (engine *countingEngine) Export(context.Context, pdf.Request) (pdf.Result, error) {
	engine.exports++
	return pdf.Result{}, engine.exportErr
}
