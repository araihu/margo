package margo

import "testing"

func TestTokenPolicyRejectsCSSChannels(t *testing.T) {
	for _, value := range []string{"var(--x)", "url(http://x)", "@import url(x)"} {
		if err := ValidateToken("color", value); err == nil {
			t.Fatalf("token %q unexpectedly accepted", value)
		}
	}
}
