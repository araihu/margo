package margo

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRuntimeRejectsTerminalTransition(t *testing.T) {
	state, err := newRuntimeState(validRuntimeDescriptor())
	if err != nil {
		t.Fatalf("newRuntimeState() error = %v", err)
	}
	completeRuntimeState(t, state)
	requireRuntimeDiagnostic(t, state.fail("late"), "runtime.transition_invalid")
}

func TestRuntimeRejectsEveryInvalidTransition(t *testing.T) {
	descriptor := validRuntimeDescriptor()
	state, err := newRuntimeState(descriptor)
	if err != nil {
		t.Fatalf("newRuntimeState() error = %v", err)
	}
	requireRuntimeDiagnostic(t, state.ready(), "runtime.transition_invalid")
	requireRuntimeDiagnostic(t, state.startTask(descriptor.Tasks[0].ID), "runtime.transition_invalid")
	if err := state.start(); err != nil {
		t.Fatalf("start() error = %v", err)
	}
	requireRuntimeDiagnostic(t, state.start(), "runtime.transition_invalid")
	requireRuntimeDiagnostic(t, state.succeedTask(descriptor.Tasks[0].ID, strings.Repeat("a", 64), 10), "runtime.transition_invalid")
	requireRuntimeDiagnostic(t, state.startTask(descriptor.Tasks[2].ID), "runtime.dependency_pending")
	if err := state.startTask(descriptor.Tasks[0].ID); err != nil {
		t.Fatalf("startTask() error = %v", err)
	}
	if err := state.succeedTask(descriptor.Tasks[0].ID, strings.Repeat("a", 64), 10); err != nil {
		t.Fatalf("succeedTask() error = %v", err)
	}
	requireRuntimeDiagnostic(t, state.succeedTask(descriptor.Tasks[0].ID, strings.Repeat("a", 64), 10), "runtime.transition_invalid")
	requireRuntimeDiagnostic(t, state.startTask("unknown"), "runtime.task_unknown")
}

func TestRuntimeTaskFailureIsIsolatedByInstance(t *testing.T) {
	firstDescriptor := validRuntimeDescriptor()
	secondDescriptor := cloneRuntimeDescriptor(firstDescriptor)
	secondDescriptor.RenderInstanceID = "ri-00000001"
	for index := range secondDescriptor.Tasks {
		secondDescriptor.Tasks[index].ID = strings.Replace(secondDescriptor.Tasks[index].ID, "ri-00000000", "ri-00000001", 1)
		for dependencyIndex := range secondDescriptor.Tasks[index].DependsOn {
			secondDescriptor.Tasks[index].DependsOn[dependencyIndex] = strings.Replace(secondDescriptor.Tasks[index].DependsOn[dependencyIndex], "ri-00000000", "ri-00000001", 1)
		}
	}
	first, err := newRuntimeState(firstDescriptor)
	if err != nil {
		t.Fatalf("first state error = %v", err)
	}
	second, err := newRuntimeState(secondDescriptor)
	if err != nil {
		t.Fatalf("second state error = %v", err)
	}
	if err := first.start(); err != nil {
		t.Fatalf("first.start() error = %v", err)
	}
	if err := first.startTask(firstDescriptor.Tasks[0].ID); err != nil {
		t.Fatalf("first.startTask() error = %v", err)
	}
	if err := first.failTask(firstDescriptor.Tasks[0].ID, "mermaid.render_failed"); err != nil {
		t.Fatalf("first.failTask() error = %v", err)
	}
	completeRuntimeState(t, second)
	if first.status != RuntimeFailed {
		t.Fatalf("first status = %q", first.status)
	}
	if second.status != RuntimeReady {
		t.Fatalf("second status = %q", second.status)
	}
}

func TestRuntimeReportRejectsForgedAndMalformedValues(t *testing.T) {
	descriptor := validRuntimeDescriptor()
	report := validRuntimeReport(descriptor, "exec-a")
	if err := ValidateRuntimeReport(descriptor, "exec-a", report); err != nil {
		t.Fatalf("ValidateRuntimeReport() error = %v", err)
	}

	tests := []struct {
		name string
		code string
		edit func(*RuntimeReport)
	}{
		{name: "protocol", code: "runtime.report_forged", edit: func(value *RuntimeReport) { value.Protocol = "margo-runtime/v2" }},
		{name: "document", code: "runtime.report_forged", edit: func(value *RuntimeReport) { value.DocumentFingerprint = DocumentFingerprint{2} }},
		{name: "instance", code: "runtime.report_forged", edit: func(value *RuntimeReport) { value.RenderInstanceID = "ri-00000001" }},
		{name: "execution", code: "runtime.report_forged", edit: func(value *RuntimeReport) { value.ExecutionID = "exec-b" }},
		{name: "duplicate task", code: "runtime.task_duplicate", edit: func(value *RuntimeReport) { value.Tasks = append(value.Tasks, value.Tasks[0]) }},
		{name: "missing task", code: "runtime.task_missing", edit: func(value *RuntimeReport) { value.Tasks = value.Tasks[:2] }},
		{name: "unknown task", code: "runtime.task_unknown", edit: func(value *RuntimeReport) { value.Tasks[0].ID = "unknown" }},
		{name: "input hash", code: "runtime.report_forged", edit: func(value *RuntimeReport) { value.Tasks[0].InputSHA256 = strings.Repeat("f", 64) }},
		{name: "nonterminal task", code: "runtime.report_malformed", edit: func(value *RuntimeReport) { value.Tasks[0].Status = RuntimeTaskRunning }},
		{name: "ready failed task", code: "runtime.report_malformed", edit: func(value *RuntimeReport) {
			value.Tasks[0].Status = RuntimeTaskFailed
			value.Tasks[0].OutputSHA256 = ""
			value.Tasks[0].OutputBytes = 0
			value.Tasks[0].ErrorCode = "mermaid.failed"
		}},
		{name: "failed without diagnostic", code: "runtime.report_malformed", edit: func(value *RuntimeReport) { value.Status = RuntimeFailed }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := cloneRuntimeReport(report)
			test.edit(&candidate)
			requireRuntimeDiagnostic(t, ValidateRuntimeReport(descriptor, "exec-a", candidate), test.code)
		})
	}

	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	unknown := bytes.Replace(encoded, []byte(`"protocol":`), []byte(`"unknown":true,"protocol":`), 1)
	requireRuntimeDiagnostic(t, parseReportError(unknown), "runtime.report_malformed")
	duplicate := bytes.Replace(encoded, []byte(`"status":`), []byte(`"status":"ready","status":`), 1)
	requireRuntimeDiagnostic(t, parseReportError(duplicate), "runtime.report_malformed")
	requireRuntimeDiagnostic(t, parseReportError(append(encoded, []byte(` []`)...)), "runtime.report_malformed")
}

func TestCanonicalRuntimeProjectionExcludesExecutionIDAndSortsSlices(t *testing.T) {
	descriptor := validRuntimeDescriptor()
	first := validRuntimeReport(descriptor, "exec-a")
	second := cloneRuntimeReport(first)
	second.ExecutionID = "exec-b"
	second.Tasks[0], second.Tasks[2] = second.Tasks[2], second.Tasks[0]

	firstBytes, err := CanonicalRuntimeProjection(first)
	if err != nil {
		t.Fatalf("CanonicalRuntimeProjection(first) error = %v", err)
	}
	secondBytes, err := CanonicalRuntimeProjection(second)
	if err != nil {
		t.Fatalf("CanonicalRuntimeProjection(second) error = %v", err)
	}
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatalf("execution ID or task order changed projection:\n%s\n%s", firstBytes, secondBytes)
	}
	if bytes.Contains(firstBytes, []byte("exec-")) {
		t.Fatalf("projection contains execution ID: %s", firstBytes)
	}
	want := canonicalProjectionLiteral(descriptor)
	if string(firstBytes) != want {
		t.Fatalf("projection =\n%s\nwant =\n%s", firstBytes, want)
	}
}

func validRuntimeReport(descriptor RuntimeDescriptor, executionID ExecutionID) RuntimeReport {
	tasks := make([]RuntimeTaskReport, len(descriptor.Tasks))
	for index, task := range descriptor.Tasks {
		tasks[index] = RuntimeTaskReport{
			ID:           task.ID,
			Kind:         task.Kind,
			InputSHA256:  task.InputSHA256,
			OutputSHA256: strings.Repeat(string(rune('a'+index)), 64),
			OutputBytes:  int64(100 + index),
			Status:       RuntimeTaskSucceeded,
		}
	}
	return RuntimeReport{
		Protocol:            RuntimeProtocolV1,
		DocumentFingerprint: descriptor.DocumentFingerprint,
		RenderInstanceID:    descriptor.RenderInstanceID,
		ExecutionID:         executionID,
		Status:              RuntimeReady,
		Tasks:               tasks,
		FontChecks:          []FontCheck{{Family: "Inter", Query: "12px Inter", Loaded: true}},
		BlockedRequests:     []BlockedRequest{},
		Layout:              LayoutMetrics{ScrollWidth: 1280, ScrollHeight: 720},
	}
}

func cloneRuntimeReport(value RuntimeReport) RuntimeReport {
	clone := value
	if value.Tasks != nil {
		clone.Tasks = append([]RuntimeTaskReport{}, value.Tasks...)
	}
	if value.FontChecks != nil {
		clone.FontChecks = append([]FontCheck{}, value.FontChecks...)
	}
	if value.BlockedRequests != nil {
		clone.BlockedRequests = append([]BlockedRequest{}, value.BlockedRequests...)
	}
	if value.Diagnostic != nil {
		diagnostic := *value.Diagnostic
		clone.Diagnostic = &diagnostic
	}
	return clone
}

func completeRuntimeState(t *testing.T, state *runtimeState) {
	t.Helper()
	if err := state.start(); err != nil {
		t.Fatalf("start() error = %v", err)
	}
	for _, task := range state.descriptor.Tasks {
		if err := state.startTask(task.ID); err != nil {
			t.Fatalf("startTask(%q) error = %v", task.ID, err)
		}
		if err := state.succeedTask(task.ID, strings.Repeat("a", 64), 1); err != nil {
			t.Fatalf("succeedTask(%q) error = %v", task.ID, err)
		}
	}
	if err := state.ready(); err != nil {
		t.Fatalf("ready() error = %v", err)
	}
}

func parseReportError(encoded []byte) error {
	_, err := ParseRuntimeReport(encoded)
	return err
}

func canonicalProjectionLiteral(descriptor RuntimeDescriptor) string {
	return `{"blockedRequests":[],"diagnosticCode":"","documentFingerprint":"` + descriptor.DocumentFingerprint.String() + `","fontChecks":[{"family":"Inter","loaded":true,"query":"12px Inter"}],"layout":{"scrollHeight":720,"scrollWidth":1280},"protocol":"margo-runtime/v1","renderInstanceID":"ri-00000000","status":"ready","tasks":[` +
		`{"errorCode":"","id":"` + descriptor.Tasks[2].ID + `","inputSHA256":"` + descriptor.Tasks[2].InputSHA256 + `","kind":"deck-layout","outputBytes":102,"outputSHA256":"` + strings.Repeat("c", 64) + `","status":"succeeded"},` +
		`{"errorCode":"","id":"` + descriptor.Tasks[0].ID + `","inputSHA256":"` + descriptor.Tasks[0].InputSHA256 + `","kind":"mermaid","outputBytes":100,"outputSHA256":"` + strings.Repeat("a", 64) + `","status":"succeeded"},` +
		`{"errorCode":"","id":"` + descriptor.Tasks[1].ID + `","inputSHA256":"` + descriptor.Tasks[1].InputSHA256 + `","kind":"mermaid","outputBytes":101,"outputSHA256":"` + strings.Repeat("b", 64) + `","status":"succeeded"}]}`
}
