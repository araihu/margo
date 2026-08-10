package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/araihu/margo/pdf"
	"github.com/araihu/margo/pdf/engines"
)

func doctorSources(data []byte) ([]string, error) {
	var report doctorReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	sources := make([]string, len(report.Candidates))
	for index, candidate := range report.Candidates {
		sources[index] = string(candidate.Source)
	}
	return sources, nil
}

func TestDoctorJSONUsesDiscoveryOrder(t *testing.T) {
	directory := t.TempDir()
	environmentPath := testBrowserFile(t, directory, "environment")
	pathBrowser := testBrowserFile(t, directory, "path")
	probe := engines.Probe{
		LookupEnv: func(name string) (string, bool) {
			if name == "MARGO_CHROMIUM_PATH" {
				return environmentPath, true
			}
			return "", false
		},
		LookPath: func(string) (string, error) { return pathBrowser, nil },
		Stat:     os.Stat,
		ChromiumVersion: func(context.Context, string) (string, error) {
			return "Chromium test", nil
		},
		ChromiumEngine: func(string) (pdf.Engine, error) { return doctorEngine{}, nil },
		Native: func(context.Context) (pdf.Engine, engines.Candidate) {
			return nil, engines.Candidate{Name: "native", Code: "pdf.native.compiled_out", Reason: "test"}
		},
		KnownPaths: []string{}, GOOS: "linux",
	}
	var stdout bytes.Buffer
	command := NewRootCommand(Dependencies{Stdout: &stdout, Build: testBuildInfo(), EngineProbe: probe})
	command.SetArgs([]string{"doctor", "--diagnostics", "json"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := doctorSources(stdout.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"environment", "path", "native"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sources = %v want %v", got, want)
	}
}

func testBrowserFile(t *testing.T, directory, name string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

type doctorEngine struct{}

func (doctorEngine) Name() string                            { return "chromium" }
func (doctorEngine) Version(context.Context) (string, error) { return "Chromium test", nil }
func (doctorEngine) Export(context.Context, pdf.Request) (pdf.Result, error) {
	return pdf.Result{}, nil
}
