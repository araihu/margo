package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	margo "github.com/araihu/margo"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type diagnosticFormat string

const (
	diagnosticText diagnosticFormat = "text"
	diagnosticJSON diagnosticFormat = "json"
)

type diagnosticProjection struct {
	Code     string         `json:"code"`
	Message  string         `json:"message"`
	Severity margo.Severity `json:"severity,omitempty"`
	Source   string         `json:"source,omitempty"`
	Line     int            `json:"line,omitempty"`
	Column   int            `json:"column,omitempty"`
	Pointer  string         `json:"pointer,omitempty"`
	Hint     string         `json:"hint,omitempty"`
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
	if projection.Source != "" {
		severity := projection.Severity
		if severity == "" {
			severity = margo.SeverityError
		}
		if _, err := fmt.Fprintf(writer, "%s:%d:%d: %s %s: %s", projection.Source, projection.Line, projection.Column, severity, projection.Code, projection.Message); err != nil {
			return err
		}
		if projection.Pointer != "" {
			if _, err := fmt.Fprintf(writer, " [%s]", projection.Pointer); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return err
		}
		if projection.Hint != "" {
			_, err := fmt.Fprintf(writer, "  hint: %s\n", projection.Hint)
			return err
		}
		return nil
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
		return diagnosticProjection{
			Code: diagnostic.Code, Message: message, Severity: diagnostic.Severity,
			Source: diagnostic.Source, Line: diagnostic.Line, Column: diagnostic.Column,
			Pointer: diagnostic.Pointer, Hint: diagnostic.Hint,
		}
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

func (failure reportedError) Unwrap() error { return failure.error }

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

func diagnosticExactArgs(expected int, formatValue *string) cobra.PositionalArgs {
	return func(command *cobra.Command, args []string) error {
		if len(args) == expected {
			return nil
		}
		format := reportingDiagnosticFormat(formatValue)
		return reportCommandError(command, format, fmt.Errorf("cli.arguments_invalid: expected exactly %d input path(s), received %d", expected, len(args)))
	}
}

func diagnosticNoArgs(formatValue *string) cobra.PositionalArgs {
	return func(command *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		return reportCommandError(command, reportingDiagnosticFormat(formatValue), fmt.Errorf("cli.arguments_invalid: command does not accept positional arguments; received %d", len(args)))
	}
}

func bindDiagnosticFlagErrors(command *cobra.Command, formatValue *string) {
	const hiddenUnknownFlag = "__margo_unknown_flag"
	unknownValue := ignoredUnknownFlagValue{}
	command.Flags().Var(&unknownValue, hiddenUnknownFlag, "")
	_ = command.Flags().MarkHidden(hiddenUnknownFlag)
	if flag := command.Flags().Lookup(hiddenUnknownFlag); flag != nil {
		flag.NoOptDefVal = "true"
	}
	command.InitDefaultHelpFlag()
	allowed := make(map[string]struct{})
	command.Flags().VisitAll(func(flag *pflag.Flag) { allowed[flag.Name] = struct{}{} })
	unknown := make([]string, 0)
	command.Flags().SetNormalizeFunc(func(_ *pflag.FlagSet, name string) pflag.NormalizedName {
		if _, exists := allowed[name]; exists {
			return pflag.NormalizedName(name)
		}
		if len(unknown) == 0 || unknown[len(unknown)-1] != name {
			unknown = append(unknown, name)
		}
		return pflag.NormalizedName(hiddenUnknownFlag)
	})
	originalArgs := command.Args
	command.Args = func(command *cobra.Command, args []string) error {
		if len(unknown) > 0 {
			return reportCommandError(command, reportingDiagnosticFormat(formatValue), fmt.Errorf("cli.flag_invalid: unknown flag: --%s", unknown[0]))
		}
		if originalArgs != nil {
			return originalArgs(command, args)
		}
		return nil
	}
	command.SetFlagErrorFunc(func(command *cobra.Command, failure error) error {
		return reportCommandError(command, reportingDiagnosticFormat(formatValue), fmt.Errorf("cli.flag_invalid: %s", failure))
	})
}

type ignoredUnknownFlagValue struct{}

func (*ignoredUnknownFlagValue) String() string   { return "" }
func (*ignoredUnknownFlagValue) Set(string) error { return nil }
func (*ignoredUnknownFlagValue) Type() string     { return "unknown" }

func reportingDiagnosticFormat(value *string) diagnosticFormat {
	if value != nil && diagnosticFormat(*value) == diagnosticJSON {
		return diagnosticJSON
	}
	return diagnosticText
}
