package pdf

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	margo "github.com/araihu/margo"
)

// ExportReport carries PDF bytes and renderer-neutral provenance. Native PDF
// byte digests are exact transport evidence but are not promised stable across
// engines or hosts.
type ExportReport struct {
	PDF                 []byte                    `json:"pdf"`
	DocumentFingerprint margo.DocumentFingerprint `json:"documentFingerprint"`
	ArtifactFingerprint margo.ArtifactFingerprint `json:"artifactFingerprint"`
	ArtifactDigest      margo.ArtifactDigest      `json:"artifactDigest"`
	Engine              EngineInfo                `json:"engine"`
	Runtime             margo.RuntimeReport       `json:"runtime"`
	Page                PageConfig                `json:"page"`
	CompilerVersion     string                    `json:"compilerVersion"`
	Theme               margo.ThemeName           `json:"theme"`
	Assets              map[string]string         `json:"assets"`
	Warnings            []string                  `json:"warnings"`
}

// Clone returns a report whose mutable values do not alias the receiver.
func (report ExportReport) Clone() ExportReport {
	report.PDF = append([]byte(nil), report.PDF...)
	report.Runtime = cloneRuntimeReport(report.Runtime)
	if report.Assets != nil {
		assets := report.Assets
		report.Assets = make(map[string]string, len(assets))
		for name, digest := range assets {
			report.Assets[name] = digest
		}
	}
	report.Warnings = append([]string(nil), report.Warnings...)
	return report
}

// Validate checks the complete public provenance shape without selecting or
// invoking an engine.
func (report ExportReport) Validate() error {
	if !bytes.HasPrefix(report.PDF, []byte("%PDF-")) {
		return exportReportError("pdf.export_report_invalid", "artifact is not a PDF")
	}
	if report.DocumentFingerprint == (margo.DocumentFingerprint{}) || report.ArtifactFingerprint == (margo.ArtifactFingerprint{}) {
		return exportReportError("pdf.export_report_invalid", "document and artifact fingerprints are required")
	}
	if report.ArtifactDigest != margo.ArtifactDigestOf(report.PDF) {
		return exportReportError("pdf.artifact_digest_mismatch", "artifact digest does not match PDF bytes")
	}
	if err := report.Engine.Validate(); err != nil {
		return err
	}
	if _, err := margo.CanonicalRuntimeProjection(report.Runtime); err != nil {
		return exportReportError("pdf.export_report_invalid", err.Error())
	}
	if report.Runtime.DocumentFingerprint != report.DocumentFingerprint {
		return exportReportError("pdf.export_report_invalid", "runtime and document fingerprints differ")
	}
	if err := report.Page.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(report.CompilerVersion) == "" {
		return exportReportError("pdf.export_report_invalid", "compiler version is required")
	}
	switch report.Theme {
	case margo.ThemeModern, margo.ThemeGoshtoso, margo.ThemeMinimal:
	default:
		return exportReportError("pdf.export_report_invalid", "theme is unsupported")
	}
	if report.Assets == nil || report.Warnings == nil {
		return exportReportError("pdf.export_report_invalid", "assets and warnings must be explicit")
	}
	for name, digest := range report.Assets {
		if strings.TrimSpace(name) == "" || !validReportSHA256(digest) {
			return exportReportError("pdf.export_report_invalid", "asset identity is invalid")
		}
	}
	for _, warning := range report.Warnings {
		if strings.TrimSpace(warning) == "" {
			return exportReportError("pdf.export_report_invalid", "warning identity is invalid")
		}
	}
	return nil
}

func validReportSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32 && value == strings.ToLower(value)
}

func exportReportError(code, message string) error {
	return fmt.Errorf("%s: %s", code, message)
}
