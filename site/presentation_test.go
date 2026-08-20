package site

import (
	"errors"
	"testing"

	margo "github.com/araihu/margo"
)

func TestKnownBindingKindIncludesSiteNavigation(t *testing.T) {
	if !knownBindingKind("site_navigation") {
		t.Fatal("site_navigation is not an allowed binding kind")
	}
}

func presentationDiagnostic(t *testing.T, err error) margo.Diagnostic {
	t.Helper()
	if err == nil {
		t.Fatal("expected diagnostic")
	}
	var diagnosticError *margo.DiagnosticError
	if !errors.As(err, &diagnosticError) || len(diagnosticError.Diagnostics) != 1 {
		t.Fatalf("error = %v, want one diagnostic", err)
	}
	return diagnosticError.Diagnostics[0]
}
