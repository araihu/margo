package margo

import (
	"testing"

	"github.com/araihu/margo/internal/canonicaljson"
)

func TestCanonicalJSONIsStable(t *testing.T) {
	got, err := canonicaljson.Marshal(map[string]any{"z": 1, "a": true})
	if err != nil {
		t.Fatalf("canonicaljson.Marshal() error = %v", err)
	}
	if string(got) != `{"a":true,"z":1}` {
		t.Fatalf("canonical JSON = %q", got)
	}
}
