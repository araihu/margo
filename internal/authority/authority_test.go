package authority

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestAuthorityRecordFixture(t *testing.T) {
	data, err := os.ReadFile("../../testdata/authority/record.json")
	if err != nil {
		t.Fatal(err)
	}
	record, err := VerifyAuthorityRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if record.CanonicalOrigin == "" || record.Asset.Width != 1280 || record.Asset.Height != 640 {
		t.Fatalf("unexpected authority record: %#v", record)
	}
}

func TestAuthorityRecordRejectsUnknownFieldAndDigestMismatch(t *testing.T) {
	data, err := os.ReadFile("../../testdata/authority/record.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	raw["unexpected"] = true
	unknown, _ := json.Marshal(raw)
	if _, err := VerifyAuthorityRecord(unknown); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("unknown field error = %v", err)
	}
	raw, _ = mapFromBytes(data)
	raw["recordDigest"] = strings.Repeat("0", 64)
	mutated, _ := json.Marshal(raw)
	if _, err := VerifyAuthorityRecord(mutated); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("digest error = %v", err)
	}
}

func mapFromBytes(data []byte) (map[string]any, error) {
	var raw map[string]any
	return raw, json.Unmarshal(data, &raw)
}
