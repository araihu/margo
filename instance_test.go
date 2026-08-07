package margo

import "testing"

func TestInstanceAllocatorUsesDeterministicBase36RenderOrder(t *testing.T) {
	allocator := NewInstanceAllocator()
	want := map[int]RenderInstanceID{
		0:  "ri-00000000",
		1:  "ri-00000001",
		35: "ri-0000000z",
		36: "ri-00000010",
	}
	for index := 0; index <= 36; index++ {
		got, err := allocator.Next()
		if err != nil {
			t.Fatalf("Next(%d) error = %v", index, err)
		}
		if expected, ok := want[index]; ok && got != expected {
			t.Fatalf("Next(%d) = %q, want %q", index, got, expected)
		}
	}
}

func TestInstanceAllocatorRegistryRejectsInvalidAndDuplicateIDs(t *testing.T) {
	for _, value := range []RenderInstanceID{"", "ri-0000000A", "ri-0000000", "ri-00000000/child", "xx-00000000"} {
		requireRuntimeDiagnostic(t, ValidateRenderInstanceID(value), "runtime.instance_invalid")
	}
	registry := NewInstanceRegistry()
	if err := registry.Reserve("ri-00000000"); err != nil {
		t.Fatalf("Reserve(first) error = %v", err)
	}
	requireRuntimeDiagnostic(t, registry.Reserve("ri-00000000"), "runtime.instance_duplicate")
	if err := registry.Reserve("ri-00000001"); err != nil {
		t.Fatalf("Reserve(second) error = %v", err)
	}
}
