package margo

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

const RuntimeProtocolV1 = "margo-runtime/v1"

type RenderInstanceID string

type ExecutionID string

type RuntimeTask struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	InputSHA256 string   `json:"inputSHA256"`
	DependsOn   []string `json:"dependsOn"`
}

type RuntimeDescriptor struct {
	Protocol            string                    `json:"protocol"`
	DocumentFingerprint DocumentFingerprint       `json:"documentFingerprint"`
	RenderInstanceID    RenderInstanceID          `json:"renderInstanceID"`
	Tasks               []RuntimeTask             `json:"tasks"`
	ValidationRequest   *RuntimeValidationRequest `json:"validationRequest,omitempty"`
}

var (
	renderInstancePattern = regexp.MustCompile(`^ri-[0-9a-z]{8,32}$`)
	runtimeKindPattern    = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
	runtimeDigestPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

func (f DocumentFingerprint) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.String())
}

func (f *DocumentFingerprint) UnmarshalJSON(data []byte) error {
	if f == nil {
		return fmt.Errorf("document fingerprint receiver is nil")
	}
	var encoded string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return fmt.Errorf("document fingerprint must be a string: %w", err)
	}
	decoded, err := hex.DecodeString(encoded)
	if err != nil || len(decoded) != len(f) || encoded != strings.ToLower(encoded) {
		return fmt.Errorf("document fingerprint must be 64 lowercase hexadecimal characters")
	}
	copy(f[:], decoded)
	return nil
}

func ValidateRenderInstanceID(value RenderInstanceID) error {
	if !renderInstancePattern.MatchString(string(value)) {
		return runtimeDiagnostic("runtime.instance_invalid", "render instance ID does not match the v1 grammar")
	}
	return nil
}

func ValidateRuntimeDescriptor(descriptor RuntimeDescriptor) error {
	if descriptor.Protocol != RuntimeProtocolV1 && descriptor.Protocol != RuntimeProtocolV2 {
		return runtimeDiagnostic("runtime.protocol_invalid", "runtime protocol is not margo-runtime/v1")
	}
	if descriptor.Protocol == RuntimeProtocolV2 {
		if descriptor.ValidationRequest == nil {
			return runtimeDiagnostic("runtime.validation_request_missing", "v2 runtime descriptor requires a validation request")
		}
		if err := descriptor.ValidationRequest.Validate(); err != nil {
			return err
		}
	} else if descriptor.ValidationRequest != nil {
		return runtimeDiagnostic("runtime.descriptor_malformed", "v1 runtime descriptor cannot contain a validation request")
	}
	if descriptor.DocumentFingerprint == (DocumentFingerprint{}) {
		return runtimeDiagnostic("runtime.document_fingerprint_invalid", "document fingerprint is zero")
	}
	if err := ValidateRenderInstanceID(descriptor.RenderInstanceID); err != nil {
		return err
	}
	if descriptor.Tasks == nil {
		return runtimeDiagnostic("runtime.descriptor_malformed", "runtime tasks must be an explicit array")
	}

	tasks := make(map[string]RuntimeTask, len(descriptor.Tasks))
	prefix := string(descriptor.RenderInstanceID) + ":"
	for _, task := range descriptor.Tasks {
		if !strings.HasPrefix(task.ID, prefix) || !validRuntimeTaskID(task.ID, descriptor.RenderInstanceID) || !runtimeKindPattern.MatchString(task.Kind) || !runtimeDigestPattern.MatchString(task.InputSHA256) {
			return runtimeDiagnostic("runtime.task_invalid", "runtime task identity is invalid")
		}
		if _, exists := tasks[task.ID]; exists {
			return runtimeDiagnostic("runtime.task_duplicate", "runtime task ID is duplicated")
		}
		if task.DependsOn == nil {
			return runtimeDiagnostic("runtime.task_invalid", "runtime dependencies must be an explicit array")
		}
		for index, dependency := range task.DependsOn {
			if index > 0 {
				switch strings.Compare(task.DependsOn[index-1], dependency) {
				case 0:
					return runtimeDiagnostic("runtime.dependency_duplicate", "runtime dependency is duplicated")
				case 1:
					return runtimeDiagnostic("runtime.dependency_unsorted", "runtime dependencies are not sorted")
				}
			}
		}
		tasks[task.ID] = task
	}
	for _, task := range descriptor.Tasks {
		for _, dependency := range task.DependsOn {
			if _, exists := tasks[dependency]; !exists {
				return runtimeDiagnostic("runtime.dependency_missing", "runtime dependency is not in the descriptor")
			}
		}
	}
	if runtimeDependencyCycle(tasks) {
		return runtimeDiagnostic("runtime.dependency_cycle", "runtime task graph contains a cycle")
	}
	return nil
}

func ParseRuntimeDescriptor(data []byte) (RuntimeDescriptor, error) {
	var descriptor RuntimeDescriptor
	if err := decodeRuntimeJSON(data, &descriptor); err != nil {
		return RuntimeDescriptor{}, runtimeDiagnostic("runtime.descriptor_malformed", err.Error())
	}
	if err := ValidateRuntimeDescriptor(descriptor); err != nil {
		return RuntimeDescriptor{}, err
	}
	return cloneRuntimeDescriptorValue(descriptor), nil
}

func validRuntimeTaskID(value string, instance RenderInstanceID) bool {
	parts := strings.Split(value, ":")
	return len(parts) == 4 && parts[0] == string(instance) && runtimeKindPattern.MatchString(parts[1]) && len(parts[2]) == 8 && allDecimal(parts[2]) && runtimeDigestPattern.MatchString(parts[3])
}

func allDecimal(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func runtimeDependencyCycle(tasks map[string]RuntimeTask) bool {
	const (
		unvisited = iota
		visiting
		visited
	)
	states := make(map[string]int, len(tasks))
	ids := make([]string, 0, len(tasks))
	for id := range tasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var visit func(string) bool
	visit = func(id string) bool {
		switch states[id] {
		case visiting:
			return true
		case visited:
			return false
		}
		states[id] = visiting
		for _, dependency := range tasks[id].DependsOn {
			if visit(dependency) {
				return true
			}
		}
		states[id] = visited
		return false
	}
	for _, id := range ids {
		if visit(id) {
			return true
		}
	}
	return false
}

func decodeRuntimeJSON(data []byte, destination any) error {
	if err := rejectDuplicateRuntimeJSONKeys(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("runtime JSON contains a second value")
		}
		return err
	}
	return nil
}

func rejectDuplicateRuntimeJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := consumeRuntimeJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("runtime JSON contains a second value")
		}
		return err
	}
	return nil
}

func consumeRuntimeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, container := token.(json.Delim)
	if !container {
		return nil
	}
	switch delimiter {
	case '{':
		keys := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("runtime JSON object key is not a string")
			}
			if _, duplicate := keys[key]; duplicate {
				return fmt.Errorf("runtime JSON contains duplicate key %q", key)
			}
			keys[key] = struct{}{}
			if err := consumeRuntimeJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim('}') {
			return fmt.Errorf("runtime JSON object is not closed")
		}
	case '[':
		for decoder.More() {
			if err := consumeRuntimeJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim(']') {
			return fmt.Errorf("runtime JSON array is not closed")
		}
	default:
		return fmt.Errorf("runtime JSON has unexpected delimiter %q", delimiter)
	}
	return nil
}

func cloneRuntimeDescriptorValue(value RuntimeDescriptor) RuntimeDescriptor {
	clone := value
	if value.ValidationRequest != nil {
		request := *value.ValidationRequest
		clone.ValidationRequest = &request
	}
	clone.Tasks = make([]RuntimeTask, len(value.Tasks))
	for index, task := range value.Tasks {
		clone.Tasks[index] = task
		if task.DependsOn != nil {
			clone.Tasks[index].DependsOn = append([]string{}, task.DependsOn...)
		}
	}
	return clone
}

func runtimeDiagnostic(code, message string) error {
	return newDiagnosticError(Diagnostic{Code: code, Severity: SeverityError, Message: message})
}
