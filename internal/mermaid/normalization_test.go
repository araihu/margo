package mermaid

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type normalizationVectors struct {
	SchemaVersion string `json:"schemaVersion"`
	Positive      []struct {
		Path             string `json:"path"`
		Family           string `json:"family"`
		SourceRootID     string `json:"sourceRootID"`
		RenderInstanceID string `json:"renderInstanceID"`
		BlockOrdinal     int    `json:"blockOrdinal"`
		NormalizedRootID string `json:"normalizedRootID"`
		DescendantCount  int    `json:"descendantCount"`
	} `json:"positive"`
	Negative []struct {
		Path         string `json:"path"`
		SourceRootID string `json:"sourceRootID"`
		ErrorCode    string `json:"errorCode"`
	} `json:"negative"`
}

func TestNormalizationVectors(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "mermaid", "normalization")
	data, err := os.ReadFile(filepath.Join(root, "vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var vectors normalizationVectors
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	if vectors.SchemaVersion != "margo-mermaid-normalization-vectors/v1" {
		t.Fatalf("unexpected schema: %q", vectors.SchemaVersion)
	}
	if len(vectors.Positive) == 0 || len(vectors.Negative) < 6 {
		t.Fatalf("incomplete normalization corpus: positive=%d negative=%d", len(vectors.Positive), len(vectors.Negative))
	}
	seen := map[string]bool{}
	for _, vector := range vectors.Positive {
		if vector.Family != "flowchart" && vector.Family != "sequence" {
			t.Fatalf("unsupported positive vector family: %+v", vector)
		}
		if vector.SourceRootID == "" || vector.RenderInstanceID == "" || vector.NormalizedRootID == "" || vector.DescendantCount <= 0 {
			t.Fatalf("incomplete positive vector: %+v", vector)
		}
		assertNormalizationFixture(t, root, vector.Path, seen)
	}
	wantCodes := map[string]bool{
		"svg.normalize.id_duplicate":           false,
		"svg.normalize.reference_unresolved":   false,
		"svg.normalize.reference_external":     false,
		"svg.normalize.reference_site_unknown": false,
		"svg.normalize.root_id_mismatch":       false,
	}
	for _, vector := range vectors.Negative {
		if vector.SourceRootID == "" || vector.ErrorCode == "" {
			t.Fatalf("incomplete negative vector: %+v", vector)
		}
		if _, ok := wantCodes[vector.ErrorCode]; ok {
			wantCodes[vector.ErrorCode] = true
		}
		assertNormalizationFixture(t, root, vector.Path, seen)
	}
	for code, covered := range wantCodes {
		if !covered {
			t.Errorf("missing normalization vector for %s", code)
		}
	}
}

func assertNormalizationFixture(t *testing.T, root, name string, seen map[string]bool) {
	t.Helper()
	if filepath.Base(name) != name || filepath.Ext(name) != ".svg" || seen[name] {
		t.Fatalf("invalid or duplicate fixture path %q", name)
	}
	seen[name] = true
	info, err := os.Stat(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatalf("empty fixture %q", name)
	}
}
