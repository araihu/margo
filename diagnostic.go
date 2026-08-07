package margo

import (
	"fmt"
	"strings"
)

// DiagnosticError carries one or more stable diagnostics without exposing
// parser internals.
type DiagnosticError struct {
	Diagnostics []Diagnostic
}

func (e *DiagnosticError) Error() string {
	if e == nil || len(e.Diagnostics) == 0 {
		return "margo: diagnostic error"
	}
	parts := make([]string, 0, len(e.Diagnostics))
	for _, diagnostic := range e.Diagnostics {
		parts = append(parts, diagnostic.Code)
	}
	return "margo: " + strings.Join(parts, ", ")
}

func newDiagnosticError(diagnostic Diagnostic) error {
	if diagnostic.Severity == "" {
		diagnostic.Severity = SeverityError
	}
	return &DiagnosticError{Diagnostics: []Diagnostic{diagnostic}}
}

func diagnosticAt(code, source, pointer, message string, line, column int) error {
	return newDiagnosticError(Diagnostic{
		Code: code, Severity: SeverityError, Source: source, Pointer: pointer,
		Message: message, Line: line, Column: column,
	})
}

func diagnosticCode(err error) string {
	if diagnosticErr, ok := err.(*DiagnosticError); ok && len(diagnosticErr.Diagnostics) > 0 {
		return diagnosticErr.Diagnostics[0].Code
	}
	return fmt.Sprint(err)
}
