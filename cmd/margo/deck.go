package main

import (
	"fmt"
	"path/filepath"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/deck"
	"github.com/araihu/margo/pdf"
	"github.com/spf13/cobra"
)

func newDeckCommand(deps Dependencies) *cobra.Command {
	formatValue := "html"
	output := outputOptions{Path: "-"}
	engineOptions := engineFlags{Mode: "auto"}
	pageOptions := pageFlags{Size: "A4", Orientation: "portrait"}
	diagnostics := string(diagnosticText)
	command := &cobra.Command{
		Use:   "deck INPUT",
		Short: "Render an HTML or PDF presentation deck",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			format, err := parseDiagnosticFormat(diagnostics)
			if err != nil {
				return err
			}
			if formatValue != "html" && formatValue != "pdf" {
				return reportCommandError(command, format, fmt.Errorf("cli.format_invalid: deck format must be html or pdf"))
			}
			if formatValue == "pdf" && !command.Flags().Changed("output") {
				return reportCommandError(command, format, fmt.Errorf("cli.output_required: PDF deck requires --output PATH or --output -"))
			}
			artifact, err := renderDeckArtifact(command, deps, args[0], formatValue, engineOptions, pageOptions)
			if err == nil {
				_, err = publish(command.Context(), artifact, output, command.OutOrStdout())
			}
			if err != nil {
				return reportCommandError(command, format, err)
			}
			return nil
		},
	}
	command.Flags().StringVar(&formatValue, "format", "html", "deck format: html or pdf")
	command.Flags().StringVarP(&output.Path, "output", "o", "-", "output path, or - for stdout")
	command.Flags().BoolVarP(&output.Force, "force", "f", false, "replace an existing output file")
	command.Flags().StringVar(&diagnostics, "diagnostics", string(diagnosticText), "diagnostic format: text or json")
	engineOptions.bind(command)
	pageOptions.bind(command)
	return command
}

func renderDeckArtifact(command *cobra.Command, deps Dependencies, input, format string, engineOptions engineFlags, pageOptions pageFlags) ([]byte, error) {
	source, err := readInput(command.Context(), deps.SourceReader, deps.Stdin, input)
	if err != nil {
		return nil, err
	}
	baseURL := ""
	if source.Name != "<stdin>" {
		absolute, pathErr := filepath.Abs(source.Name)
		if pathErr != nil {
			return nil, fmt.Errorf("cli.input_path_invalid: %w", pathErr)
		}
		baseURL = filepath.Dir(absolute)
	}
	result, err := deck.Render(command.Context(), newCompiler(), deck.RenderInput{
		Name: source.Name, Markdown: source.Content, BaseURL: baseURL,
	})
	if err != nil {
		return nil, err
	}
	html, err := materializeLocalImages(result.HTML(), source.Name, deps.WorkingDirectory)
	if err != nil {
		return nil, err
	}
	if format == "html" {
		return html, nil
	}
	instance, err := margo.NewInstanceAllocator().Next()
	if err != nil {
		return nil, err
	}
	descriptor, err := result.RuntimeDescriptor(instance)
	if err != nil {
		return nil, err
	}
	executionID := deps.NextExecutionID()
	if executionID == "" {
		return nil, fmt.Errorf("cli.execution_id_invalid: execution ID source returned an empty ID")
	}
	pageConfig, err := pageOptions.config()
	if err != nil {
		return nil, err
	}
	return exportPDFArtifact(command.Context(), deps, html, descriptor, executionID, pageConfig, engineOptions, pdfLinkConfig{Policy: pdf.RelativeLinksStrip})
}
