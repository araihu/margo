package main

import "testing"

func TestPreV1PolicyRejectsDuplicateJSONKeys(t *testing.T) {
	_, err := parsePolicyDocument([]byte(`{"schemaVersion":"margo-policy/v1","rawHTML":"deny","rawHTML":"sanitized"}`))
	if err == nil {
		t.Fatal("duplicate policy property was accepted")
	}
}

func TestPreV1PolicyRejectsExplicitZeroResourceLimit(t *testing.T) {
	_, err := parsePolicyDocument([]byte(`{"schemaVersion":"margo-policy/v1","rawHTML":"deny","outputBytes":0}`))
	if err == nil {
		t.Fatal("explicit zero selected a default instead of failing")
	}
}
