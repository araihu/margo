package main

import (
	"bytes"
	"fmt"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/pdf"
	"github.com/spf13/cobra"
)

type pageFlags struct {
	Size        string
	Orientation string
	Top         float64
	Right       float64
	Bottom      float64
	Left        float64
}

func newPDFCommand(deps Dependencies) *cobra.Command {
	output := outputOptions{}
	engineOptions := engineFlags{Mode: "auto"}
	pageOptions := pageFlags{Size: string(pdf.PageA4), Orientation: string(pdf.Portrait)}
	diagnostics := string(diagnosticText)
	command := &cobra.Command{
		Use:   "pdf INPUT",
		Short: "Render a PDF document",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			format, err := parseDiagnosticFormat(diagnostics)
			if err != nil {
				return err
			}
			if output.Path == "" {
				return reportCommandError(command, format, fmt.Errorf("cli.output_required: PDF requires --output PATH or --output -"))
			}
			artifact, err := renderPDF(command, deps, args[0], engineOptions, pageOptions)
			if err == nil {
				_, err = publish(command.Context(), artifact, output, command.OutOrStdout())
			}
			if err != nil {
				return reportCommandError(command, format, err)
			}
			return nil
		},
	}
	command.Flags().StringVarP(&output.Path, "output", "o", "", "required output path, or - for binary stdout")
	command.Flags().BoolVarP(&output.Force, "force", "f", false, "replace an existing output file")
	command.Flags().StringVar(&diagnostics, "diagnostics", string(diagnosticText), "diagnostic format: text or json")
	engineOptions.bind(command)
	command.Flags().StringVar(&pageOptions.Size, "page-size", string(pdf.PageA4), "page size: A4 or Letter")
	command.Flags().StringVar(&pageOptions.Orientation, "orientation", string(pdf.Portrait), "orientation: portrait or landscape")
	command.Flags().Float64Var(&pageOptions.Top, "margin-top", 0, "top margin in millimeters")
	command.Flags().Float64Var(&pageOptions.Right, "margin-right", 0, "right margin in millimeters")
	command.Flags().Float64Var(&pageOptions.Bottom, "margin-bottom", 0, "bottom margin in millimeters")
	command.Flags().Float64Var(&pageOptions.Left, "margin-left", 0, "left margin in millimeters")
	return command
}

func renderPDF(command *cobra.Command, deps Dependencies, input string, engineOptions engineFlags, pageOptions pageFlags) ([]byte, error) {
	compiled, err := compileStandalone(command.Context(), deps, input)
	if err != nil {
		return nil, err
	}
	instance, err := margo.NewInstanceAllocator().Next()
	if err != nil {
		return nil, err
	}
	descriptor, err := compiled.Render.RuntimeDescriptor(instance)
	if err != nil {
		return nil, err
	}
	executionID := deps.NextExecutionID()
	if executionID == "" {
		return nil, fmt.Errorf("cli.execution_id_invalid: execution ID source returned an empty ID")
	}
	pageConfig := pdf.PageConfig{
		Size:        pdf.PageSize(pageOptions.Size),
		Orientation: pdf.Orientation(pageOptions.Orientation),
		Margins: pdf.Margins{
			Top: pdf.Millimeters(pageOptions.Top), Right: pdf.Millimeters(pageOptions.Right),
			Bottom: pdf.Millimeters(pageOptions.Bottom), Left: pdf.Millimeters(pageOptions.Left),
		},
	}
	if err := pageConfig.Validate(); err != nil {
		return nil, err
	}
	engine, _, err := selectEngine(command.Context(), deps.EngineProbe, engineOptions)
	if err != nil {
		return nil, err
	}
	result, err := engine.Export(command.Context(), pdf.Request{
		HTML: compiled.HTML, Runtime: descriptor, ExecutionID: executionID, Page: pageConfig,
	})
	if err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(result.PDF, []byte("%PDF-")) {
		return nil, fmt.Errorf("pdf.output_invalid: selected engine returned invalid PDF bytes")
	}
	if err := margo.ValidateRuntimeReport(descriptor, executionID, result.Runtime); err != nil {
		return nil, fmt.Errorf("pdf.runtime_report_invalid: %w", err)
	}
	if err := result.Engine.Validate(); err != nil {
		return nil, err
	}
	return append([]byte(nil), result.PDF...), nil
}
