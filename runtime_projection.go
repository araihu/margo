package margo

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/araihu/margo/internal/canonicaljson"
)

type runtimeTaskTemplate struct {
	kind        string
	inputSHA256 string
	dependsOn   []int
}

func runtimeTaskTemplates(plan renderPlan) ([]runtimeTaskTemplate, error) {
	templates := make([]runtimeTaskTemplate, 0)
	for _, node := range plan.nodes {
		compiled, ok := node.compiled.(mermaidCompiledTask)
		if !ok {
			continue
		}
		encoded, err := canonicaljson.Marshal(compiled.descriptor)
		if err != nil {
			return nil, fmt.Errorf("runtime.task_projection_failed: %w", err)
		}
		digest := sha256.Sum256(encoded)
		templates = append(templates, runtimeTaskTemplate{
			kind:        compiled.descriptor.Kind,
			inputSHA256: hex.EncodeToString(digest[:]),
			dependsOn:   []int{},
		})
	}
	return templates, nil
}

func cloneRuntimeTaskTemplates(templates []runtimeTaskTemplate) []runtimeTaskTemplate {
	cloned := make([]runtimeTaskTemplate, len(templates))
	for index, template := range templates {
		cloned[index] = template
		cloned[index].dependsOn = append([]int(nil), template.dependsOn...)
	}
	return cloned
}

func (r *RenderResult) DocumentFingerprint() DocumentFingerprint {
	if r == nil {
		return DocumentFingerprint{}
	}
	return r.documentFingerprint
}

func (r *RenderResult) RuntimeDescriptor(instance RenderInstanceID) (RuntimeDescriptor, error) {
	if r == nil {
		return RuntimeDescriptor{}, runtimeDiagnostic("runtime.result_required", "render result is required")
	}
	if err := ValidateRenderInstanceID(instance); err != nil {
		return RuntimeDescriptor{}, err
	}
	templates := cloneRuntimeTaskTemplates(r.runtimeTasks)
	descriptor := RuntimeDescriptor{
		Protocol:            RuntimeProtocolV1,
		DocumentFingerprint: r.documentFingerprint,
		RenderInstanceID:    instance,
		Tasks:               make([]RuntimeTask, len(templates)),
	}
	ordinals := make(map[string]uint32)
	for index, template := range templates {
		ordinal := ordinals[template.kind]
		id, err := projectedRuntimeTaskID(instance, template.kind, ordinal, template.inputSHA256)
		if err != nil {
			return RuntimeDescriptor{}, err
		}
		ordinals[template.kind] = ordinal + 1
		descriptor.Tasks[index] = RuntimeTask{
			ID:          id,
			Kind:        template.kind,
			InputSHA256: template.inputSHA256,
			DependsOn:   make([]string, len(template.dependsOn)),
		}
	}
	for index, template := range templates {
		for dependencyIndex, templateIndex := range template.dependsOn {
			if templateIndex < 0 || templateIndex >= len(descriptor.Tasks) {
				return RuntimeDescriptor{}, runtimeDiagnostic("runtime.template_invalid", "runtime template dependency is out of range")
			}
			descriptor.Tasks[index].DependsOn[dependencyIndex] = descriptor.Tasks[templateIndex].ID
		}
		sort.Strings(descriptor.Tasks[index].DependsOn)
	}
	if err := ValidateRuntimeDescriptor(descriptor); err != nil {
		return RuntimeDescriptor{}, err
	}
	return cloneRuntimeDescriptorValue(descriptor), nil
}

func ComposeRuntimeDescriptors(document DocumentFingerprint, instance RenderInstanceID, parts ...RuntimeDescriptor) (RuntimeDescriptor, error) {
	if document == (DocumentFingerprint{}) {
		return RuntimeDescriptor{}, runtimeDiagnostic("runtime.document_fingerprint_invalid", "document fingerprint is zero")
	}
	if err := ValidateRenderInstanceID(instance); err != nil {
		return RuntimeDescriptor{}, err
	}
	result := RuntimeDescriptor{
		Protocol:            RuntimeProtocolV1,
		DocumentFingerprint: document,
		RenderInstanceID:    instance,
		Tasks:               []RuntimeTask{},
	}
	ordinals := make(map[string]uint32)
	for _, part := range parts {
		if err := ValidateRuntimeDescriptor(part); err != nil {
			return RuntimeDescriptor{}, err
		}
		mapping := make(map[string]string, len(part.Tasks))
		start := len(result.Tasks)
		for _, task := range part.Tasks {
			ordinal := ordinals[task.Kind]
			id, err := projectedRuntimeTaskID(instance, task.Kind, ordinal, task.InputSHA256)
			if err != nil {
				return RuntimeDescriptor{}, err
			}
			ordinals[task.Kind] = ordinal + 1
			mapping[task.ID] = id
			result.Tasks = append(result.Tasks, RuntimeTask{
				ID:          id,
				Kind:        task.Kind,
				InputSHA256: task.InputSHA256,
				DependsOn:   make([]string, len(task.DependsOn)),
			})
		}
		for index, task := range part.Tasks {
			for dependencyIndex, dependency := range task.DependsOn {
				mapped, ok := mapping[dependency]
				if !ok {
					return RuntimeDescriptor{}, runtimeDiagnostic("runtime.dependency_missing", "runtime dependency is not in the descriptor")
				}
				result.Tasks[start+index].DependsOn[dependencyIndex] = mapped
			}
			sort.Strings(result.Tasks[start+index].DependsOn)
		}
	}
	if err := ValidateRuntimeDescriptor(result); err != nil {
		return RuntimeDescriptor{}, err
	}
	return cloneRuntimeDescriptorValue(result), nil
}

func projectedRuntimeTaskID(instance RenderInstanceID, kind string, ordinal uint32, digest string) (string, error) {
	if ordinal > 99999999 {
		return "", runtimeDiagnostic("runtime.task_exhausted", "runtime task ordinal exceeds the v1 grammar")
	}
	return fmt.Sprintf("%s:%s:%08d:%s", instance, kind, ordinal, digest), nil
}
