package engines

import (
	"fmt"
	"strings"
)

type Error struct {
	DiagnosticCode string      `json:"code"`
	Message        string      `json:"message"`
	Candidates     []Candidate `json:"candidates,omitempty"`
}

func (err *Error) Error() string {
	if err == nil {
		return "pdf.engine_error"
	}
	return fmt.Sprintf("%s: %s", err.DiagnosticCode, err.Message)
}

func (err *Error) Code() string {
	if err == nil {
		return "pdf.engine_error"
	}
	return err.DiagnosticCode
}

func engineError(code, message string, candidates []Candidate) error {
	return &Error{DiagnosticCode: code, Message: strings.TrimSpace(message), Candidates: append([]Candidate(nil), candidates...)}
}
