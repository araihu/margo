package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newHTMLCommand(deps Dependencies) *cobra.Command {
	options := outputOptions{Path: "-"}
	metadata := standaloneMetadataFlags{}
	diagnostics := string(diagnosticText)
	command := &cobra.Command{
		Use:   "html INPUT",
		Short: "Render standalone HTML",
		Args:  diagnosticExactArgs(1, &diagnostics),
		RunE: func(command *cobra.Command, args []string) error {
			format, err := parseDiagnosticFormat(diagnostics)
			if err != nil {
				return err
			}
			compiled, err := compileStandalone(command.Context(), deps, args[0], metadata.standaloneOptions(command)...)
			if err == nil {
				_, err = publish(command.Context(), compiled.HTML, options, command.OutOrStdout())
			}
			if err != nil {
				return reportCommandError(command, format, err)
			}
			return nil
		},
	}
	command.Flags().StringVarP(&options.Path, "output", "o", "-", "output path, or - for stdout")
	command.Flags().BoolVarP(&options.Force, "force", "f", false, "replace an existing output file")
	command.Flags().StringVar(&diagnostics, "diagnostics", string(diagnosticText), "diagnostic format: text or json")
	metadata.bind(command)
	bindDiagnosticFlagErrors(command, &diagnostics)
	return command
}

func parseDiagnosticFormat(value string) (diagnosticFormat, error) {
	format := diagnosticFormat(value)
	if format != diagnosticText && format != diagnosticJSON {
		return "", fmt.Errorf("cli.diagnostics_invalid: diagnostics must be text or json")
	}
	return format, nil
}
