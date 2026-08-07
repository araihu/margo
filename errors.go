package margo

import "errors"

func isDiagnosticCode(err error, code string) bool { return diagnosticCode(err) == code }

func unwrapDiagnostic(err error) *DiagnosticError {
	var diagnosticErr *DiagnosticError
	if errors.As(err, &diagnosticErr) {
		return diagnosticErr
	}
	return nil
}
