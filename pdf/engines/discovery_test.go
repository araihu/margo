package engines

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/araihu/margo/pdf"
)

func TestDiscoverUsesDeterministicAutoOrder(t *testing.T) {
	directory := t.TempDir()
	explicit := executableFile(t, directory, "explicit-chrome")
	environment := executableFile(t, directory, "environment-chrome")
	pathBrowser := executableFile(t, directory, "path-chromium")
	native := &fakeEngine{name: "native", version: "native-1"}
	probe := testProbe()
	probe.LookupEnv = func(name string) (string, bool) {
		if name == "MARGO_CHROMIUM_PATH" {
			return environment, true
		}
		return "", false
	}
	probe.LookPath = func(name string) (string, error) {
		if name == "chromium" {
			return pathBrowser, nil
		}
		return "", os.ErrNotExist
	}
	probe.Native = func(context.Context) (pdf.Engine, Candidate) {
		return native, Candidate{Name: "native", Source: SourceNative, Compiled: true, Available: true, Version: "native-1"}
	}
	discovery, err := Discover(context.Background(), Request{Mode: ModeAuto, EnginePath: explicit}, probe)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := discovery.Sources(), []Source{SourceFlag, SourceEnvironment, SourcePath, SourceNative}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sources = %v want %v", got, want)
	}
	_, candidate, err := discovery.Select()
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Source != SourceFlag || candidate.Path != explicit {
		t.Fatalf("selected = %+v", candidate)
	}
}

func TestDiscoverRejectsInvalidExplicitPathImmediately(t *testing.T) {
	probe := testProbe()
	nativeCalls := 0
	probe.Native = func(context.Context) (pdf.Engine, Candidate) {
		nativeCalls++
		return &fakeEngine{name: "native", version: "1"}, Candidate{Name: "native", Source: SourceNative, Compiled: true, Available: true}
	}
	_, err := Discover(context.Background(), Request{Mode: ModeAuto, EnginePath: filepath.Join(t.TempDir(), "missing")}, probe)
	if diagnosticCode(err) != "pdf.engine_path_invalid" {
		t.Fatalf("error = %v", err)
	}
	if nativeCalls != 0 {
		t.Fatal("native discovery ran after invalid explicit path")
	}
}

func TestSelectedEngineRecordsCandidateProvenanceWithoutFallback(t *testing.T) {
	failed := &fakeEngine{name: "chromium", version: "Chrome 140", exportErr: errors.New("render failed")}
	native := &fakeEngine{name: "native", version: "native-1"}
	discovery := Discovery{
		candidates: []Candidate{
			{Name: "chromium", Version: "Chrome 140", Path: "/browser", Source: SourcePath, Compiled: true, Available: true},
			{Name: "native", Version: "native-1", Source: SourceNative, Compiled: true, Available: true},
		},
		engines: []pdf.Engine{failed, native},
	}
	selected, _, err := discovery.Select()
	if err != nil {
		t.Fatal(err)
	}
	_, err = selected.Export(context.Background(), pdf.Request{})
	if err == nil || failed.exports != 1 || native.exports != 0 {
		t.Fatalf("error = %v chromium exports = %d native exports = %d", err, failed.exports, native.exports)
	}
}

func TestSelectedEngineDecoratesSuccessfulResult(t *testing.T) {
	engine := &fakeEngine{name: "chromium", version: "Chrome 140", result: pdf.Result{PDF: []byte("%PDF-test")}}
	discovery := Discovery{
		candidates: []Candidate{{Name: "chromium", Version: "Chrome 140", Path: "/browser", Source: SourceKnownLocation, Compiled: true, Available: true}},
		engines:    []pdf.Engine{engine},
	}
	selected, _, err := discovery.Select()
	if err != nil {
		t.Fatal(err)
	}
	result, err := selected.Export(context.Background(), pdf.Request{})
	if err != nil {
		t.Fatal(err)
	}
	want := pdf.EngineInfo{Name: "chromium", Version: "Chrome 140", Path: "/browser", Source: string(SourceKnownLocation)}
	if result.Engine != want {
		t.Fatalf("engine info = %+v want %+v", result.Engine, want)
	}
}

func TestDefaultProbeConstructsInstalledChromium(t *testing.T) {
	path := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	if _, err := os.Stat(path); err != nil {
		t.Skip("known installed Chromium unavailable")
	}
	discovery, err := Discover(context.Background(), Request{Mode: ModeChromium, EnginePath: path}, Probe{})
	if err != nil {
		t.Fatal(err)
	}
	engine, candidate, err := discovery.Select()
	if err != nil {
		t.Fatal(err)
	}
	if engine == nil || !candidate.Available || candidate.Version == "" {
		t.Fatalf("candidate = %+v engine = %T", candidate, engine)
	}
}

func testProbe() Probe {
	return Probe{
		LookupEnv: func(string) (string, bool) { return "", false },
		LookPath:  func(string) (string, error) { return "", os.ErrNotExist },
		Stat:      os.Stat,
		ChromiumVersion: func(context.Context, string) (string, error) {
			return "Chrome 140", nil
		},
		ChromiumEngine: func(string) (pdf.Engine, error) {
			return &fakeEngine{name: "chromium", version: "Chrome 140"}, nil
		},
		Native: func(context.Context) (pdf.Engine, Candidate) {
			return nil, Candidate{Name: "native", Source: SourceNative, Code: "pdf.native.compiled_out", Reason: "not compiled"}
		},
		GOOS: "linux",
	}
}

func executableFile(t *testing.T, directory, name string) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("browser"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

type fakeEngine struct {
	name      string
	version   string
	result    pdf.Result
	exportErr error
	exports   int
}

func (engine *fakeEngine) Name() string                            { return engine.name }
func (engine *fakeEngine) Version(context.Context) (string, error) { return engine.version, nil }
func (engine *fakeEngine) Export(context.Context, pdf.Request) (pdf.Result, error) {
	engine.exports++
	return engine.result.Clone(), engine.exportErr
}

func diagnosticCode(err error) string {
	if err == nil {
		return ""
	}
	var diagnostic interface{ Code() string }
	if errors.As(err, &diagnostic) {
		return diagnostic.Code()
	}
	message := err.Error()
	for index, character := range message {
		if character == ':' {
			return message[:index]
		}
	}
	return message
}
