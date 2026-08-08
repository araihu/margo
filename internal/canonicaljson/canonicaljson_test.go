package canonicaljson

import (
	"math"
	"testing"
)

func TestMarshalCanonicalObject(t *testing.T) {
	got, err := Marshal(map[string]any{"b": 2, "a": "<x>"})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(got) != `{"a":"<x>","b":2}` {
		t.Fatalf("canonical JSON = %s", got)
	}
}

func TestMarshalRejectsNonFinite(t *testing.T) {
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := Marshal(value); err == nil {
			t.Fatalf("Marshal(%v) unexpectedly succeeded", value)
		}
	}
}
