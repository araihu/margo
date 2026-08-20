package site

import (
	"errors"
	"strings"

	margo "github.com/araihu/margo"
)

func newPresentationDiagnostic(code, message, hint, pointer string) error {
	err := diagnostic(code, message, hint, "")
	var diagnosticError *margo.DiagnosticError
	if errors.As(err, &diagnosticError) && pointer != "" {
		for index := range diagnosticError.Diagnostics {
			diagnosticError.Diagnostics[index].Pointer = pointer
		}
	}
	return err
}

func presentationSourceDiagnostic(err error, source string) error {
	if strings.TrimSpace(source) == "" {
		return err
	}
	var diagnosticError *margo.DiagnosticError
	if errors.As(err, &diagnosticError) {
		for index := range diagnosticError.Diagnostics {
			diagnosticError.Diagnostics[index].Source = source
		}
	}
	return err
}
