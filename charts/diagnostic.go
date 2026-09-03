package charts

import margo "github.com/araihu/margo"

func chartDiagnostic(code, message string) error {
	return &margo.DiagnosticError{Diagnostics: []margo.Diagnostic{{
		Code: code, Severity: margo.SeverityError, Message: message,
	}}}
}

func chartDiagnosticAt(code, message string, line, column int) error {
	return chartDiagnosticAtPointer(code, message, "", line, column)
}

func chartDiagnosticAtPointer(code, message, pointer string, line, column int) error {
	return &margo.DiagnosticError{Diagnostics: []margo.Diagnostic{{
		Code: code, Severity: margo.SeverityError, Pointer: pointer, Line: line, Column: column, Message: message,
	}}}
}
