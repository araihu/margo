package margo

import "testing"

func TestResourcePolicyRejectsOversizedDocument(t *testing.T) {
	if err := ValidateResourceSize(MaxDocumentBytes+1, ResourceLimits{DocumentBytes: MaxDocumentBytes}); err == nil {
		t.Fatal("oversized document unexpectedly accepted")
	}
}
