package engines

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/araihu/margo/pdf"
)

type Mode string

const (
	ModeAuto     Mode = "auto"
	ModeChromium Mode = "chromium"
	ModeNative   Mode = "native"
)

type Source string

const (
	SourceFlag          Source = "flag"
	SourceEnvironment   Source = "environment"
	SourcePath          Source = "path"
	SourceKnownLocation Source = "known-location"
	SourceNative        Source = "native"
)

type Request struct {
	Mode       Mode
	EnginePath string
}

type Candidate struct {
	Name      string `json:"name"`
	Source    Source `json:"source"`
	Compiled  bool   `json:"compiled"`
	Path      string `json:"path,omitempty"`
	Version   string `json:"version,omitempty"`
	Available bool   `json:"available"`
	Code      string `json:"code,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type Probe struct {
	LookupEnv       func(string) (string, bool)
	LookPath        func(string) (string, error)
	Stat            func(string) (fs.FileInfo, error)
	ChromiumVersion func(context.Context, string) (string, error)
	ChromiumEngine  func(string) (pdf.Engine, error)
	Native          func(context.Context) (pdf.Engine, Candidate)
	KnownPaths      []string
	GOOS            string
}

type Discovery struct {
	candidates []Candidate
	engines    []pdf.Engine
}

func (discovery Discovery) Candidates() []Candidate {
	return append([]Candidate(nil), discovery.candidates...)
}

func (discovery Discovery) Sources() []Source {
	sources := make([]Source, len(discovery.candidates))
	for index, candidate := range discovery.candidates {
		sources[index] = candidate.Source
	}
	return sources
}

func Discover(ctx context.Context, request Request, probe Probe) (Discovery, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Discovery{}, err
	}
	if request.Mode == "" {
		request.Mode = ModeAuto
	}
	if request.Mode != ModeAuto && request.Mode != ModeChromium && request.Mode != ModeNative {
		return Discovery{}, engineError("pdf.engine_mode_invalid", "engine mode must be auto, chromium, or native", nil)
	}
	if request.Mode == ModeNative && strings.TrimSpace(request.EnginePath) != "" {
		return Discovery{}, engineError("pdf.engine_path_invalid", "engine path cannot select a native engine", nil)
	}
	probe = normalizeProbe(probe)
	discovery := Discovery{}
	seenPaths := make(map[string]struct{})
	appendChromium := func(path string, source Source, strict bool) error {
		candidate, engine, err := probeChromium(ctx, path, source, probe)
		if err != nil && strict {
			return err
		}
		if candidate.Path != "" {
			if _, exists := seenPaths[candidate.Path]; exists {
				return nil
			}
			seenPaths[candidate.Path] = struct{}{}
		}
		discovery.candidates = append(discovery.candidates, candidate)
		discovery.engines = append(discovery.engines, engine)
		return nil
	}

	if request.Mode != ModeNative {
		if path := strings.TrimSpace(request.EnginePath); path != "" {
			if err := appendChromium(path, SourceFlag, true); err != nil {
				return Discovery{}, err
			}
		}
		if path, ok := probe.LookupEnv("MARGO_CHROMIUM_PATH"); ok && strings.TrimSpace(path) != "" {
			if err := appendChromium(path, SourceEnvironment, false); err != nil {
				return Discovery{}, err
			}
		}
		for _, name := range chromiumExecutableNames(probe.GOOS) {
			path, err := probe.LookPath(name)
			if err != nil || strings.TrimSpace(path) == "" {
				continue
			}
			if err := appendChromium(path, SourcePath, false); err != nil {
				return Discovery{}, err
			}
		}
		for _, path := range probe.KnownPaths {
			if err := appendChromium(path, SourceKnownLocation, false); err != nil {
				return Discovery{}, err
			}
		}
	}

	if request.Mode != ModeChromium {
		engine, candidate := probe.Native(ctx)
		if candidate.Name == "" {
			candidate.Name = "native"
		}
		candidate.Source = SourceNative
		discovery.candidates = append(discovery.candidates, candidate)
		discovery.engines = append(discovery.engines, engine)
	}
	return discovery, nil
}

func normalizeProbe(probe Probe) Probe {
	if probe.LookupEnv == nil {
		probe.LookupEnv = os.LookupEnv
	}
	if probe.LookPath == nil {
		probe.LookPath = func(string) (string, error) { return "", os.ErrNotExist }
	}
	if probe.Stat == nil {
		probe.Stat = os.Stat
	}
	if probe.ChromiumVersion == nil {
		probe.ChromiumVersion = func(context.Context, string) (string, error) { return "", fmt.Errorf("version probe unavailable") }
	}
	if probe.ChromiumEngine == nil {
		probe.ChromiumEngine = func(string) (pdf.Engine, error) { return nil, fmt.Errorf("Chromium engine unavailable") }
	}
	if probe.Native == nil {
		probe.Native = func(context.Context) (pdf.Engine, Candidate) {
			return nil, Candidate{Name: "native", Source: SourceNative, Code: "pdf.native.compiled_out", Reason: "native engine is not compiled"}
		}
	}
	if probe.GOOS == "" {
		probe.GOOS = runtime.GOOS
	}
	probe.KnownPaths = append([]string(nil), probe.KnownPaths...)
	return probe
}

func probeChromium(ctx context.Context, path string, source Source, probe Probe) (Candidate, pdf.Engine, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Candidate{}, nil, engineError("pdf.engine_path_invalid", err.Error(), nil)
	}
	candidate := Candidate{Name: "chromium", Source: source, Path: absolute, Compiled: true}
	info, err := probe.Stat(absolute)
	if err != nil || !info.Mode().IsRegular() || (probe.GOOS != "windows" && info.Mode().Perm()&0o111 == 0) {
		candidate.Code = "pdf.engine_path_invalid"
		candidate.Reason = "path is not an executable regular file"
		return candidate, nil, engineError(candidate.Code, candidate.Reason, []Candidate{candidate})
	}
	version, err := probe.ChromiumVersion(ctx, absolute)
	if err != nil || strings.TrimSpace(version) == "" {
		candidate.Code = "pdf.chromium_probe_failed"
		candidate.Reason = "Chromium version probe failed"
		return candidate, nil, engineError(candidate.Code, candidate.Reason, []Candidate{candidate})
	}
	engine, err := probe.ChromiumEngine(absolute)
	if err != nil || engine == nil {
		candidate.Code = "pdf.chromium_unavailable"
		candidate.Reason = "Chromium engine could not be constructed"
		return candidate, nil, engineError(candidate.Code, candidate.Reason, []Candidate{candidate})
	}
	candidate.Version = strings.TrimSpace(version)
	candidate.Available = true
	return candidate, engine, nil
}

func chromiumExecutableNames(goos string) []string {
	switch goos {
	case "darwin":
		return []string{"chromium", "google-chrome", "Google Chrome"}
	case "windows":
		return []string{"chrome.exe", "msedge.exe", "chromium.exe"}
	default:
		return []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable", "microsoft-edge"}
	}
}
