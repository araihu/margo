package margo

import (
	"crypto/sha256"
	"strings"
	"testing"
)

func TestRenderResultProjectsRuntimeDescriptor(t *testing.T) {
	result := mustRenderSource(t, "```mermaid\ngraph TD; A-->B\n```\n")
	descriptor, err := result.RuntimeDescriptor("ri-00000000")
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.DocumentFingerprint != result.DocumentFingerprint() {
		t.Fatal("fingerprint mismatch")
	}
	if len(descriptor.Tasks) != 1 || descriptor.Tasks[0].Kind != "mermaid" {
		t.Fatalf("tasks = %#v", descriptor.Tasks)
	}
	if err := ValidateRuntimeDescriptor(descriptor); err != nil {
		t.Fatal(err)
	}
}

func TestRenderResultProjectsExplicitEmptyTaskList(t *testing.T) {
	result := mustRenderSource(t, "# Static\n")
	descriptor, err := result.RuntimeDescriptor("ri-00000001")
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.Tasks == nil || len(descriptor.Tasks) != 0 {
		t.Fatalf("tasks = %#v", descriptor.Tasks)
	}
}

func TestComposeRuntimeDescriptorsRebasesTaskIdentities(t *testing.T) {
	first := projectionDescriptor("ri-00000000", "mermaid", strings.Repeat("a", 64))
	second := projectionDescriptor("ri-00000001", "mermaid", strings.Repeat("b", 64))
	fingerprint := DocumentFingerprint(sha256.Sum256([]byte("deck")))
	merged, err := ComposeRuntimeDescriptors(fingerprint, "ri-00000002", first, second)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Tasks) != 2 {
		t.Fatalf("tasks = %d", len(merged.Tasks))
	}
	if merged.Tasks[0].ID == first.Tasks[0].ID || merged.Tasks[1].ID == second.Tasks[0].ID {
		t.Fatal("task IDs were not rebased")
	}
	if merged.Tasks[0].ID == merged.Tasks[1].ID {
		t.Fatal("rebased task IDs collide")
	}
	if err := ValidateRuntimeDescriptor(merged); err != nil {
		t.Fatal(err)
	}
}

func TestComposeRuntimeDescriptorsRewritesDependencies(t *testing.T) {
	fingerprint := DocumentFingerprint(sha256.Sum256([]byte("source")))
	part := RuntimeDescriptor{
		Protocol:            RuntimeProtocolV1,
		DocumentFingerprint: fingerprint,
		RenderInstanceID:    "ri-00000003",
		Tasks: []RuntimeTask{
			{ID: "ri-00000003:mermaid:00000000:" + strings.Repeat("c", 64), Kind: "mermaid", InputSHA256: strings.Repeat("c", 64), DependsOn: []string{}},
			{ID: "ri-00000003:mermaid:00000001:" + strings.Repeat("d", 64), Kind: "mermaid", InputSHA256: strings.Repeat("d", 64), DependsOn: []string{"ri-00000003:mermaid:00000000:" + strings.Repeat("c", 64)}},
		},
	}
	merged, err := ComposeRuntimeDescriptors(fingerprint, "ri-00000004", part)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := merged.Tasks[1].DependsOn, []string{merged.Tasks[0].ID}; !equalStrings(got, want) {
		t.Fatalf("dependencies = %v want %v", got, want)
	}
}

func projectionDescriptor(instance RenderInstanceID, kind, digest string) RuntimeDescriptor {
	fingerprint := DocumentFingerprint(sha256.Sum256([]byte(instance)))
	return RuntimeDescriptor{
		Protocol:            RuntimeProtocolV1,
		DocumentFingerprint: fingerprint,
		RenderInstanceID:    instance,
		Tasks: []RuntimeTask{{
			ID:          string(instance) + ":" + kind + ":00000000:" + digest,
			Kind:        kind,
			InputSHA256: digest,
			DependsOn:   []string{},
		}},
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
