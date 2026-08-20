package margo

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRuntimeDescriptorAcceptsValidDependencyGraph(t *testing.T) {
	descriptor := validRuntimeDescriptor()
	if err := ValidateRuntimeDescriptor(descriptor); err != nil {
		t.Fatalf("ValidateRuntimeDescriptor() error = %v", err)
	}
}

func TestRuntimeDescriptorRejectsInvalidIdentityAndTaskGraphs(t *testing.T) {
	tests := []struct {
		name string
		code string
		edit func(*RuntimeDescriptor)
	}{
		{name: "missing v2 request", code: "runtime.validation_request_missing", edit: func(value *RuntimeDescriptor) { value.Protocol = "margo-runtime/v2" }},
		{name: "zero document", code: "runtime.document_fingerprint_invalid", edit: func(value *RuntimeDescriptor) { value.DocumentFingerprint = DocumentFingerprint{} }},
		{name: "instance", code: "runtime.instance_invalid", edit: func(value *RuntimeDescriptor) { value.RenderInstanceID = "ri-ABC" }},
		{name: "null tasks", code: "runtime.descriptor_malformed", edit: func(value *RuntimeDescriptor) { value.Tasks = nil }},
		{name: "task outside instance", code: "runtime.task_invalid", edit: func(value *RuntimeDescriptor) {
			value.Tasks[0].ID = "ri-00000001:mermaid:00000000:" + strings.Repeat("a", 64)
		}},
		{name: "task hash", code: "runtime.task_invalid", edit: func(value *RuntimeDescriptor) { value.Tasks[0].InputSHA256 = "A" + strings.Repeat("a", 63) }},
		{name: "duplicate task", code: "runtime.task_duplicate", edit: func(value *RuntimeDescriptor) { value.Tasks = append(value.Tasks, value.Tasks[0]) }},
		{name: "duplicate dependency", code: "runtime.dependency_duplicate", edit: func(value *RuntimeDescriptor) {
			value.Tasks[2].DependsOn = []string{value.Tasks[0].ID, value.Tasks[0].ID}
		}},
		{name: "unsorted dependencies", code: "runtime.dependency_unsorted", edit: func(value *RuntimeDescriptor) {
			value.Tasks[2].DependsOn = []string{value.Tasks[1].ID, value.Tasks[0].ID}
		}},
		{name: "missing dependency", code: "runtime.dependency_missing", edit: func(value *RuntimeDescriptor) { value.Tasks[2].DependsOn = []string{"ri-00000000:missing"} }},
		{name: "dependency cycle", code: "runtime.dependency_cycle", edit: func(value *RuntimeDescriptor) {
			value.Tasks[0].DependsOn = []string{value.Tasks[1].ID}
			value.Tasks[1].DependsOn = []string{value.Tasks[0].ID}
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			descriptor := cloneRuntimeDescriptor(validRuntimeDescriptor())
			test.edit(&descriptor)
			requireRuntimeDiagnostic(t, ValidateRuntimeDescriptor(descriptor), test.code)
		})
	}
}

func TestRuntimeDescriptorJSONIsStrictAndUsesHexFingerprint(t *testing.T) {
	descriptor := validRuntimeDescriptor()
	encoded, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !bytes.Contains(encoded, []byte(`"documentFingerprint":"0100000000000000000000000000000000000000000000000000000000000000"`)) {
		t.Fatalf("document fingerprint is not lowercase hex: %s", encoded)
	}
	decoded, err := ParseRuntimeDescriptor(encoded)
	if err != nil {
		t.Fatalf("ParseRuntimeDescriptor() error = %v", err)
	}
	if decoded.DocumentFingerprint != descriptor.DocumentFingerprint {
		t.Fatalf("fingerprint = %s", decoded.DocumentFingerprint)
	}

	unknown := bytes.Replace(encoded, []byte(`"protocol":`), []byte(`"unknown":true,"protocol":`), 1)
	requireRuntimeDiagnostic(t, parseDescriptorError(unknown), "runtime.descriptor_malformed")
	duplicate := bytes.Replace(encoded, []byte(`"protocol":`), []byte(`"protocol":"margo-runtime/v1","protocol":`), 1)
	requireRuntimeDiagnostic(t, parseDescriptorError(duplicate), "runtime.descriptor_malformed")
	requireRuntimeDiagnostic(t, parseDescriptorError(append(encoded, []byte(` {}`)...)), "runtime.descriptor_malformed")
	uppercase := bytes.Replace(encoded, []byte(`"0100000000000000000000000000000000000000000000000000000000000000"`), []byte(`"A100000000000000000000000000000000000000000000000000000000000000"`), 1)
	requireRuntimeDiagnostic(t, parseDescriptorError(uppercase), "runtime.descriptor_malformed")
}

func parseDescriptorError(encoded []byte) error {
	_, err := ParseRuntimeDescriptor(encoded)
	return err
}

func validRuntimeDescriptor() RuntimeDescriptor {
	instance := RenderInstanceID("ri-00000000")
	first := string(instance) + ":mermaid:00000000:" + strings.Repeat("a", 64)
	second := string(instance) + ":mermaid:00000001:" + strings.Repeat("b", 64)
	layout := string(instance) + ":deck-layout:00000002:" + strings.Repeat("c", 64)
	return RuntimeDescriptor{
		Protocol:            RuntimeProtocolV1,
		DocumentFingerprint: DocumentFingerprint{1},
		RenderInstanceID:    instance,
		Tasks: []RuntimeTask{
			{ID: first, Kind: "mermaid", InputSHA256: strings.Repeat("1", 64), DependsOn: []string{}},
			{ID: second, Kind: "mermaid", InputSHA256: strings.Repeat("2", 64), DependsOn: []string{}},
			{ID: layout, Kind: "deck-layout", InputSHA256: strings.Repeat("3", 64), DependsOn: []string{first, second}},
		},
	}
}

func cloneRuntimeDescriptor(value RuntimeDescriptor) RuntimeDescriptor {
	clone := value
	clone.Tasks = make([]RuntimeTask, len(value.Tasks))
	for index, task := range value.Tasks {
		clone.Tasks[index] = task
		if task.DependsOn != nil {
			clone.Tasks[index].DependsOn = append([]string{}, task.DependsOn...)
		}
	}
	return clone
}

func requireRuntimeDiagnostic(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected diagnostic %q", code)
	}
	if got := diagnosticCode(err); got != code {
		t.Fatalf("diagnostic code = %q, want %q, err = %v", got, code, err)
	}
}
