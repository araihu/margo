package ssg

import (
	"strings"
	"testing"
)

func TestResolveFrameValuesDefaultsAliasesAndHash(t *testing.T) {
	schema := compositionTestSchema(false, nil)
	schema.Options = []FrameOptionDescriptor{
		{Path: "areas.top-nav.sticky.enabled", Type: "boolean", Default: false, Description: "Enable sticky."},
		{Path: "areas.top-nav.sticky.edge", Type: "enum", Default: "block-start", Allowed: []string{"block-start", "block-end"}, Description: "Sticky edge."},
		{Path: "areas.top-nav.sticky.offset", Type: "length", Default: "0", Description: "Sticky offset."},
	}
	values, err := ResolveFrameValues(schema, Values{"areas": map[string]any{
		"top-nav": map[string]any{"sticky": map[string]any{"enabled": true, "edge": "top", "offset": "1rem"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	edge, ok := valueAt(values, "areas", "top-nav", "sticky", "edge")
	if !ok || edge != "block-start" {
		t.Fatalf("resolved edge = %#v", edge)
	}
	if _, err := ResolveFrameValues(schema, Values{"unknown": true}); err == nil || !strings.Contains(err.Error(), "option_unknown") {
		t.Fatalf("unknown option error = %v", err)
	}
	first, err := SchemaHashForValues(schema, Values{"areas": map[string]any{"top-nav": map[string]any{"sticky": map[string]any{"enabled": true}}}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := SchemaHashForValues(schema, Values{"areas": map[string]any{"top-nav": map[string]any{"sticky": map[string]any{"enabled": false}}}})
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("structural values did not affect schema hash")
	}
}

func TestValidateFrameOptionsRejectsIncompleteDescriptors(t *testing.T) {
	bad := []FrameOptionDescriptor{
		{Path: "areas..sticky", Type: "boolean", Default: false, Description: "bad path"},
		{Path: "density", Type: "enum", Default: "compact", Description: "missing allowed"},
		{Path: "width", Type: "number", Min: float64Ptr(3), Max: float64Ptr(2), Description: "bad range"},
	}
	for index, option := range bad {
		if err := ValidateFrameOptions([]FrameOptionDescriptor{option}); err == nil {
			t.Fatalf("case %d unexpectedly succeeded", index)
		}
	}
}

func float64Ptr(value float64) *float64 { return &value }
