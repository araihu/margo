package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	margo "github.com/araihu/margo"
	"github.com/spf13/cobra"
)

type checkReport struct {
	Diagnostics []margo.Diagnostic `json:"diagnostics"`
	Policy      string             `json:"policy,omitempty"`
	Errors      int                `json:"errors"`
	Warnings    int                `json:"warnings"`
}

func newCheckCommand(deps Dependencies) *cobra.Command {
	diagnostics := string(diagnosticText)
	target := string(margo.TargetHTML)
	policyOptions := policyFlags{}
	command := &cobra.Command{
		Use:   "check INPUT",
		Short: "Check Markdown compatibility without rendering",
		Args:  diagnosticExactArgs(1, &diagnostics),
		RunE: func(command *cobra.Command, args []string) error {
			format, err := parseDiagnosticFormat(diagnostics)
			if err != nil {
				return err
			}
			policy, err := policyOptions.load(command.Context(), deps.SourceReader)
			if err != nil {
				return reportCommandError(command, format, err)
			}
			source, err := readInput(command.Context(), deps.SourceReader, deps.Stdin, args[0])
			if err != nil {
				return reportCommandError(command, format, err)
			}
			if source.Name == "<stdin>" {
				source.BaseURL, err = filepath.Abs(deps.WorkingDirectory)
			} else {
				var absolute string
				absolute, err = filepath.Abs(source.Name)
				source.BaseURL = filepath.Dir(absolute)
			}
			if err != nil {
				return reportCommandError(command, format, fmt.Errorf("cli.input_path_invalid: %w", err))
			}
			checkOptions := []margo.CheckOption{margo.WithCheckAssetReader(deps.CheckAssetReader), margo.WithCheckTarget(margo.RenderTarget(target))}
			if policy != nil {
				checkOptions = append(checkOptions, margo.WithCheckPolicy(policy.Host))
			}
			findings, err := margo.Check(command.Context(), source, checkOptions...)
			if err != nil {
				return reportCommandError(command, format, err)
			}
			report := summarizeCheck(findings)
			if policy != nil {
				report.Policy = policy.Digest
			}
			if err := writeCheckReport(command.OutOrStdout(), format, report); err != nil {
				return err
			}
			if report.Errors > 0 {
				return reportedError{error: errors.New("check.failed")}
			}
			return nil
		},
	}
	command.Flags().StringVar(&diagnostics, "diagnostics", string(diagnosticText), "diagnostic format: text or json")
	command.Flags().StringVar(&target, "target", string(margo.TargetHTML), "output target: html, site, pdf, or deck")
	policyOptions.bind(command)
	bindDiagnosticFlagErrors(command, &diagnostics)
	return command
}

func summarizeCheck(diagnostics []margo.Diagnostic) checkReport {
	report := checkReport{Diagnostics: append([]margo.Diagnostic(nil), diagnostics...)}
	if report.Diagnostics == nil {
		report.Diagnostics = []margo.Diagnostic{}
	}
	for _, diagnostic := range diagnostics {
		switch diagnostic.Severity {
		case margo.SeverityError:
			report.Errors++
		case margo.SeverityWarning:
			report.Warnings++
		}
	}
	return report
}

func writeCheckReport(writer io.Writer, format diagnosticFormat, report checkReport) error {
	if writer == nil {
		return errors.New("cli.diagnostics_writer_required: diagnostic writer is unavailable")
	}
	if format == diagnosticJSON {
		return json.NewEncoder(writer).Encode(report)
	}
	if format != diagnosticText {
		return errors.New("cli.diagnostics_invalid: diagnostics must be text or json")
	}
	for _, diagnostic := range report.Diagnostics {
		if _, err := fmt.Fprintf(writer, "%s:%d:%d: %s %s: %s [%s]\n  hint: %s\n", diagnostic.Source, diagnostic.Line, diagnostic.Column, diagnostic.Severity, diagnostic.Code, diagnostic.Message, diagnostic.Pointer, diagnostic.Hint); err != nil {
			return err
		}
	}
	if report.Policy != "" {
		if _, err := fmt.Fprintf(writer, "policy %s\n", report.Policy); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(writer, "%d error(s), %d warning(s)\n", report.Errors, report.Warnings)
	return err
}
