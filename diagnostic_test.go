package margo

import "testing"

func TestDiagnosticErrorCarriesStableCode(t *testing.T) {
	err := diagnosticAt("frontmatter.invalid", "x.md", "/", "bad", 2, 3)
	if got := diagnosticCode(err); got != "frontmatter.invalid" {
		t.Fatalf("diagnostic code = %q", got)
	}
	if diagnosticErr := unwrapDiagnostic(err); diagnosticErr == nil || diagnosticErr.Diagnostics[0].Line != 2 {
		t.Fatalf("diagnostic location was not retained: %#v", diagnosticErr)
	}
}
