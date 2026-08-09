package pdf

import (
	"reflect"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

func TestExportReportAcceptsCompleteProvenance(t *testing.T) {
	t.Parallel()

	report := validExportReport()
	if err := report.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestExportReportRejectsDigestMismatch(t *testing.T) {
	t.Parallel()

	report := validExportReport()
	report.ArtifactDigest = margo.ArtifactDigest{9}
	if err := report.Validate(); err == nil || !strings.HasPrefix(err.Error(), "pdf.artifact_digest_mismatch") {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestExportReportClonesMutableProvenance(t *testing.T) {
	t.Parallel()

	report := validExportReport()
	cloned := report.Clone()
	report.PDF[0] = 'X'
	report.Runtime.Tasks[0].OutputSHA256 = "changed"
	report.Assets["document.css"] = "changed"
	report.Warnings[0] = "changed"
	if string(cloned.PDF) != "%PDF-contract" {
		t.Fatalf("cloned PDF = %q", cloned.PDF)
	}
	if got := cloned.Runtime.Tasks[0].OutputSHA256; got != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("cloned runtime digest = %q", got)
	}
	if got := cloned.Assets["document.css"]; got != "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" {
		t.Fatalf("cloned asset digest = %q", got)
	}
	if got := cloned.Warnings[0]; got != "pdf.native_bytes_may_vary" {
		t.Fatalf("cloned warning = %q", got)
	}
}

func TestExportReportPublicValuesExcludeEngineImplementations(t *testing.T) {
	t.Parallel()

	engineType := reflect.TypeOf((*Engine)(nil)).Elem()
	for _, value := range []any{Request{}, Result{}, EngineInfo{}, ExportReport{}} {
		valueType := reflect.TypeOf(value)
		for index := 0; index < valueType.NumField(); index++ {
			fieldType := valueType.Field(index).Type
			if fieldType == engineType || fieldType.Implements(engineType) || reflect.PointerTo(fieldType).Implements(engineType) {
				t.Fatalf("%s.%s exposes engine implementation type %s", valueType, valueType.Field(index).Name, fieldType)
			}
		}
	}
}

func validExportReport() ExportReport {
	request := validContractRequest()
	pdfBytes := []byte("%PDF-contract")
	return ExportReport{
		PDF:                 pdfBytes,
		DocumentFingerprint: request.Runtime.DocumentFingerprint,
		ArtifactFingerprint: margo.ArtifactFingerprint{2},
		ArtifactDigest:      margo.ArtifactDigestOf(pdfBytes),
		Engine:              EngineInfo{Name: "contract", Version: "1.0.0"},
		Runtime:             validContractRuntimeReport(request.Runtime, request.ExecutionID),
		Page:                request.Page,
		CompilerVersion:     "margo/v0.0.1",
		Theme:               margo.ThemeModern,
		Assets: map[string]string{
			"document.css": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		},
		Warnings: []string{"pdf.native_bytes_may_vary"},
	}
}
