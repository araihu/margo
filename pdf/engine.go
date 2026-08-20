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

// RelativeLinkPolicy controls how document-relative anchors are projected into
// a PDF. The zero value is the safe strip policy because a renderer's temporary
// document origin must never leak into a distributed artifact by default.
type RelativeLinkPolicy string

const (
	RelativeLinksStrip   RelativeLinkPolicy = "strip"
	RelativeLinksError   RelativeLinkPolicy = "error"
	RelativeLinksKeep    RelativeLinkPolicy = "keep"
	RelativeLinksResolve RelativeLinkPolicy = "resolve"
)

// Request is the renderer-neutral input shared by every PDF engine.
type Request struct {
	HTML          []byte                  `json:"html"`
	Runtime       margo.RuntimeDescriptor `json:"runtime"`
	ExecutionID   margo.ExecutionID       `json:"executionID"`
	Page          PageConfig              `json:"page"`
	RelativeLinks RelativeLinkPolicy      `json:"relativeLinks,omitempty"`
	BaseURL       string                  `json:"baseURL,omitempty"`
}

// Clone returns a request whose mutable slices do not alias the receiver.
func (request Request) Clone() Request {
	request.HTML = append([]byte(nil), request.HTML...)
	request.Runtime = cloneRuntimeDescriptor(request.Runtime)
	if request.Page.Custom != nil {
		custom := *request.Page.Custom
		request.Page.Custom = &custom
	}
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
	if descriptor.ValidationRequest != nil {
		request := *descriptor.ValidationRequest
		descriptor.ValidationRequest = &request
	}
	if descriptor.Tasks != nil {
		descriptor.Tasks = append([]margo.RuntimeTask{}, descriptor.Tasks...)
	}
	for index := range descriptor.Tasks {
		if descriptor.Tasks[index].DependsOn != nil {
			descriptor.Tasks[index].DependsOn = append([]string{}, descriptor.Tasks[index].DependsOn...)
		}
	}
	return descriptor
}

func cloneRuntimeReport(report margo.RuntimeReport) margo.RuntimeReport {
	if report.ValidationIdentity != nil {
		identity := *report.ValidationIdentity
		report.ValidationIdentity = &identity
	}
	if report.Tasks != nil {
		report.Tasks = append([]margo.RuntimeTaskReport{}, report.Tasks...)
	}
	if report.FontChecks != nil {
		report.FontChecks = append([]margo.FontCheck{}, report.FontChecks...)
	}
	if report.BlockedRequests != nil {
		report.BlockedRequests = append([]margo.BlockedRequest{}, report.BlockedRequests...)
	}
	if report.Diagnostic != nil {
		diagnostic := *report.Diagnostic
		report.Diagnostic = &diagnostic
	}
	return report
}
