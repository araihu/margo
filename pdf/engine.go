package pdf

import (
	"context"
	"fmt"
	"strings"

	margo "github.com/araihu/margo"
)

// Engine exports one immutable HTML/runtime request through one explicitly
// selected renderer. Implementations never select or fall back to another
// engine.
type Engine interface {
	Name() string
	Version(context.Context) (string, error)
	Export(context.Context, Request) (Result, error)
}

// Request is the renderer-neutral input shared by every PDF engine.
type Request struct {
	HTML        []byte                  `json:"html"`
	Runtime     margo.RuntimeDescriptor `json:"runtime"`
	ExecutionID margo.ExecutionID       `json:"executionID"`
	Page        PageConfig              `json:"page"`
}

// Clone returns a request whose mutable slices do not alias the receiver.
func (request Request) Clone() Request {
	request.HTML = append([]byte(nil), request.HTML...)
	request.Runtime = cloneRuntimeDescriptor(request.Runtime)
	return request
}

// Result is the renderer-neutral output returned by an Engine.
type Result struct {
	PDF     []byte              `json:"pdf"`
	Runtime margo.RuntimeReport `json:"runtime"`
	Engine  EngineInfo          `json:"engine"`
}

// Clone returns a result whose mutable slices do not alias the receiver.
func (result Result) Clone() Result {
	result.PDF = append([]byte(nil), result.PDF...)
	result.Runtime = cloneRuntimeReport(result.Runtime)
	return result
}

// EngineInfo records the selected engine identity without exposing an engine
// implementation value.
type EngineInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path,omitempty"`
	Source  string `json:"source,omitempty"`
}

// Validate rejects incomplete engine provenance.
func (info EngineInfo) Validate() error {
	if strings.TrimSpace(info.Name) == "" || strings.TrimSpace(info.Version) == "" {
		return fmt.Errorf("pdf.engine_identity_invalid: engine name and version are required")
	}
	if info.Source != "" {
		switch info.Source {
		case "flag", "environment", "path", "known-location", "native":
		default:
			return fmt.Errorf("pdf.engine_identity_invalid: engine selection source is invalid")
		}
		if info.Source == "native" && info.Path != "" {
			return fmt.Errorf("pdf.engine_identity_invalid: native engine must not report an executable path")
		}
		if info.Source != "native" && strings.TrimSpace(info.Path) == "" {
			return fmt.Errorf("pdf.engine_identity_invalid: Chromium engine path is required")
		}
	}
	return nil
}

func cloneRuntimeDescriptor(descriptor margo.RuntimeDescriptor) margo.RuntimeDescriptor {
	descriptor.Tasks = append([]margo.RuntimeTask(nil), descriptor.Tasks...)
	for index := range descriptor.Tasks {
		descriptor.Tasks[index].DependsOn = append([]string(nil), descriptor.Tasks[index].DependsOn...)
	}
	return descriptor
}

func cloneRuntimeReport(report margo.RuntimeReport) margo.RuntimeReport {
	report.Tasks = append([]margo.RuntimeTaskReport(nil), report.Tasks...)
	report.FontChecks = append([]margo.FontCheck(nil), report.FontChecks...)
	report.BlockedRequests = append([]margo.BlockedRequest(nil), report.BlockedRequests...)
	if report.Diagnostic != nil {
		diagnostic := *report.Diagnostic
		report.Diagnostic = &diagnostic
	}
	return report
}
