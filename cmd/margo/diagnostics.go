package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	margo "github.com/araihu/margo"
)

type diagnosticFormat string

const (
	diagnosticText diagnosticFormat = "text"
	diagnosticJSON diagnosticFormat = "json"
)

type diagnosticProjection struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeDiagnostic(writer io.Writer, format diagnosticFormat, failure error) error {
	if format != diagnosticText && format != diagnosticJSON {
		return fmt.Errorf("cli.diagnostics_invalid: diagnostics must be text or json")
	}
	if writer == nil {
		return fmt.Errorf("cli.diagnostics_writer_required: diagnostic writer is unavailable")
	}
	projection := projectDiagnostic(failure)
	if format == diagnosticJSON {
		return json.NewEncoder(writer).Encode(projection)
	}
	_, err := fmt.Fprintf(writer, "%s: %s\n", projection.Code, projection.Message)
	return err
}

func projectDiagnostic(failure error) diagnosticProjection {
	if failure == nil {
		return diagnosticProjection{Code: "cli.unknown", Message: "unknown failure"}
	}
	var margoFailure *margo.DiagnosticError
	if errors.As(failure, &margoFailure) && len(margoFailure.Diagnostics) > 0 {
		diagnostic := margoFailure.Diagnostics[0]
		message := diagnostic.Message
		if message == "" {
			message = diagnostic.Code
		}
		return diagnosticProjection{Code: diagnostic.Code, Message: message}
	}
	var coded interface{ Code() string }
	if errors.As(failure, &coded) && coded.Code() != "" {
		message := strings.TrimSpace(strings.TrimPrefix(failure.Error(), coded.Code()+":"))
		if message == "" {
			message = coded.Code()
		}
		return diagnosticProjection{Code: coded.Code(), Message: message}
	}
	message := failure.Error()
	if index := strings.IndexByte(message, ':'); index >= 0 {
		return diagnosticProjection{Code: strings.TrimSpace(message[:index]), Message: strings.TrimSpace(message[index+1:])}
	}
	if validDiagnosticCode(message) {
		return diagnosticProjection{Code: message, Message: message}
	}
	return diagnosticProjection{Code: "cli.error", Message: message}
}

func cliDiagnosticCode(failure error) string { return projectDiagnostic(failure).Code }

type reportedError struct{ error }

func reportCommandError(command interface{ ErrOrStderr() io.Writer }, format diagnosticFormat, failure error) error {
	if err := writeDiagnostic(command.ErrOrStderr(), format, failure); err != nil {
		return err
	}
	return reportedError{error: failure}
}

func validDiagnosticCode(value string) bool {
	if value == "" || strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '.' && character != '_' {
			return false
		}
	}
	return true
}
