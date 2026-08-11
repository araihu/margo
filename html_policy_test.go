package margo

import "testing"

func TestHTMLPolicyRejectsScriptAndUnsafeURLs(t *testing.T) {
	for _, input := range []string{`<script>alert(1)</script>`, `</script>`, `<a href="javascript:alert(1)">x</a>`, `<div onclick="x">x</div>`} {
		if err := ValidateHTML(input); err == nil {
			t.Fatalf("unsafe HTML unexpectedly accepted: %s", input)
		}
	}
}

func TestHTMLPolicyAcceptsSemanticSubset(t *testing.T) {
	if err := ValidateHTML(`<p title="ok"><a href="/guide">Guide</a></p>`); err != nil {
		t.Fatalf("safe HTML rejected: %v", err)
	}
}
