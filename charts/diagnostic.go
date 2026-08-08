package charts

import margo "github.com/araihu/margo"

func chartDiagnostic(code, message string) error {
	return &margo.DiagnosticError{Diagnostics: []margo.Diagnostic{{
		Code: code, Severity: margo.SeverityError, Message: message,
	}}}
}

func chartDiagnosticAt(code, message string, line, column int) error {
	return &margo.DiagnosticError{Diagnostics: []margo.Diagnostic{{
		Code: code, Severity: margo.SeverityError, Line: line, Column: column, Message: message,
	}}}
}
