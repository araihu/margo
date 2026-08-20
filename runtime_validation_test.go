package margo

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRuntimeValidationRequestRejectsInvalidProfileValues(t *testing.T) {
	base := validRuntimeValidationRequest()
	tests := []struct {
		name string
		edit func(*RuntimeValidationRequest)
	}{
		{name: "viewport width", edit: func(value *RuntimeValidationRequest) { value.ViewportWidth = 0 }},
		{name: "viewport height", edit: func(value *RuntimeValidationRequest) { value.ViewportHeight = 0 }},
		{name: "device scale", edit: func(value *RuntimeValidationRequest) { value.DeviceScaleFactor = 0 }},
		{name: "zoom", edit: func(value *RuntimeValidationRequest) { value.Zoom = 0 }},
		{name: "browser profile", edit: func(value *RuntimeValidationRequest) { value.BrowserProfile = "" }},
		{name: "font digest", edit: func(value *RuntimeValidationRequest) { value.ExpectedFontBundleDigest = "A" + strings.Repeat("0", 63) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := *base
			test.edit(&candidate)
			if err := candidate.Validate(); err == nil {
				t.Fatal("Validate() unexpectedly succeeded")
			}
		})
	}
}

func TestRuntimeV2DescriptorAndReportBindObservedIdentity(t *testing.T) {
	descriptor := validRuntimeV2Descriptor()
	if err := ValidateRuntimeDescriptor(descriptor); err != nil {
		t.Fatalf("ValidateRuntimeDescriptor() error = %v", err)
	}
	report := validRuntimeV2Report(descriptor, "exec-v2")
	if err := ValidateRuntimeReport(descriptor, "exec-v2", report); err != nil {
		t.Fatalf("ValidateRuntimeReport() error = %v", err)
	}

	fontMismatch := cloneRuntimeReport(report)
	fontMismatch.ValidationIdentity.FontBundleDigest = strings.Repeat("f", 64)
	requireRuntimeDiagnostic(t, ValidateRuntimeReport(descriptor, "exec-v2", fontMismatch), "deck.font_bundle_mismatch")

	profileMismatch := cloneRuntimeReport(report)
	profileMismatch.ValidationIdentity.BrowserProfile = "other-browser"
	requireRuntimeDiagnostic(t, ValidateRuntimeReport(descriptor, "exec-v2", profileMismatch), "deck.validator_profile_mismatch")

	failed := cloneRuntimeV2Report(report)
	failed.Status = RuntimeFailed
	failed.Tasks[0].Status = RuntimeTaskFailed
	failed.Tasks[0].ErrorCode = "deck.overflow"
	failed.Tasks[0].OutputSHA256 = ""
	failed.Tasks[0].OutputBytes = 0
	failed.Diagnostic = &Diagnostic{Code: "deck.overflow", Severity: SeverityError}
	failed.ValidationIdentity = nil
	if err := ValidateRuntimeReport(descriptor, "exec-v2", failed); err != nil {
		t.Fatalf("failed report without identity rejected: %v", err)
	}
}

func TestRuntimeV2JSONIsStrictAndDefensivelyCloned(t *testing.T) {
	descriptor := validRuntimeV2Descriptor()
	encoded, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := ParseRuntimeDescriptor(encoded)
	if err != nil {
		t.Fatal(err)
	}
	decoded.ValidationRequest.ViewportWidth = 1
	if descriptor.ValidationRequest.ViewportWidth == 1 {
		t.Fatal("descriptor request aliases parsed value")
	}
	unknown := bytes.Replace(encoded, []byte(`"protocol":`), []byte(`"unknown":true,"protocol":`), 1)
	requireRuntimeDiagnostic(t, parseDescriptorError(unknown), "runtime.descriptor_malformed")

	report := validRuntimeV2Report(descriptor, "exec-v2")
	reportBytes, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	parsedReport, err := ParseRuntimeReport(reportBytes)
	if err != nil {
		t.Fatal(err)
	}
	parsedReport.ValidationIdentity.EngineName = "changed"
	if report.ValidationIdentity.EngineName == "changed" {
		t.Fatal("report identity aliases parsed value")
	}
	unknownReport := bytes.Replace(reportBytes, []byte(`"protocol":`), []byte(`"unknown":true,"protocol":`), 1)
	requireRuntimeDiagnostic(t, parseReportError(unknownReport), "runtime.report_malformed")
}

func validRuntimeValidationRequest() *RuntimeValidationRequest {
	return &RuntimeValidationRequest{
		ViewportWidth: 1440, ViewportHeight: 900, DeviceScaleFactor: 1, Zoom: 1,
		BrowserProfile: "chromium-deck-v1", ExpectedFontBundleDigest: strings.Repeat("0", 64),
	}
}

func validRuntimeV2Descriptor() RuntimeDescriptor {
	instance := RenderInstanceID("ri-00000042")
	firstDigest := strings.Repeat("0", 64)
	secondDigest := strings.Repeat("1", 64)
	return RuntimeDescriptor{
		Protocol: RuntimeProtocolV2, DocumentFingerprint: DocumentFingerprint{1}, RenderInstanceID: instance,
		ValidationRequest: validRuntimeValidationRequest(),
		Tasks: []RuntimeTask{
			{ID: string(instance) + ":deck-layout-screen:00000000:" + firstDigest, Kind: "deck-layout-screen", InputSHA256: firstDigest, DependsOn: []string{}},
			{ID: string(instance) + ":deck-layout-print-dom:00000000:" + secondDigest, Kind: "deck-layout-print-dom", InputSHA256: secondDigest, DependsOn: []string{}},
		},
	}
}

func validRuntimeV2Report(descriptor RuntimeDescriptor, executionID ExecutionID) RuntimeReport {
	report := validRuntimeReport(descriptor, executionID)
	report.Protocol = RuntimeProtocolV2
	report.ValidationIdentity = &RuntimeValidationIdentity{
		BrowserProfile: "chromium-deck-v1", EngineName: "Chromium", EngineVersion: "123.0.0.0",
		PlatformProfile: "darwin-arm64", FontBundleDigest: descriptor.ValidationRequest.ExpectedFontBundleDigest,
	}
	return report
}

func cloneRuntimeV2Report(value RuntimeReport) RuntimeReport {
	clone := cloneRuntimeReport(value)
	if value.ValidationIdentity != nil {
		identity := *value.ValidationIdentity
		clone.ValidationIdentity = &identity
	}
	return clone
}
