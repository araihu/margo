package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"

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

type pdfLinkFlags struct {
	Policy  string
	BaseURL string
}

type pdfLinkConfig struct {
	Policy  pdf.RelativeLinkPolicy
	BaseURL string
}

func newPDFCommand(deps Dependencies) *cobra.Command {
	output := outputOptions{}
	engineOptions := engineFlags{Mode: "auto"}
	pageOptions := pageFlags{Size: string(pdf.PageA4), Orientation: string(pdf.Portrait)}
	linkOptions := pdfLinkFlags{Policy: string(pdf.RelativeLinksStrip)}
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
			linkConfig, err := linkOptions.config(command.Flags().Changed("relative-links"))
			if err != nil {
				return reportCommandError(command, format, err)
			}
			artifact, err := renderPDF(command, deps, args[0], engineOptions, pageOptions, linkConfig)
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
	pageOptions.bind(command)
	linkOptions.bind(command)
	return command
}

func renderPDF(command *cobra.Command, deps Dependencies, input string, engineOptions engineFlags, pageOptions pageFlags, linkConfig pdfLinkConfig) ([]byte, error) {
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
	pageConfig, err := pageOptions.config()
	if err != nil {
		return nil, err
	}
	return exportPDFArtifact(command.Context(), deps, compiled.HTML, descriptor, executionID, pageConfig, engineOptions, linkConfig)
}

func (options *pdfLinkFlags) bind(command *cobra.Command) {
	command.Flags().StringVar(&options.Policy, "relative-links", string(pdf.RelativeLinksStrip), "relative PDF links: strip, error, keep, or resolve")
	command.Flags().StringVar(&options.BaseURL, "base-url", "", "absolute public http(s) base URL used to resolve relative PDF links")
}

func (options pdfLinkFlags) config(policyExplicit bool) (pdfLinkConfig, error) {
	policy := pdf.RelativeLinkPolicy(options.Policy)
	switch policy {
	case pdf.RelativeLinksStrip, pdf.RelativeLinksError, pdf.RelativeLinksKeep, pdf.RelativeLinksResolve:
	default:
		return pdfLinkConfig{}, fmt.Errorf("cli.relative_link_policy_invalid: --relative-links must be strip, error, keep, or resolve")
	}
	baseURL := strings.TrimSpace(options.BaseURL)
	if baseURL != "" && !policyExplicit {
		policy = pdf.RelativeLinksResolve
	}
	if policy == pdf.RelativeLinksResolve && baseURL == "" {
		return pdfLinkConfig{}, fmt.Errorf("cli.relative_link_base_required: --relative-links resolve requires --base-url URL")
	}
	if policy != pdf.RelativeLinksResolve && baseURL != "" {
		return pdfLinkConfig{}, fmt.Errorf("cli.relative_link_options_invalid: --base-url requires --relative-links resolve")
	}
	return pdfLinkConfig{Policy: policy, BaseURL: baseURL}, nil
}

func (options *pageFlags) bind(command *cobra.Command) {
	command.Flags().StringVar(&options.Size, "page-size", string(pdf.PageA4), "page size: A4 or Letter")
	command.Flags().StringVar(&options.Orientation, "orientation", string(pdf.Portrait), "orientation: portrait or landscape")
	command.Flags().Float64Var(&options.Top, "margin-top", 0, "top margin in millimeters")
	command.Flags().Float64Var(&options.Right, "margin-right", 0, "right margin in millimeters")
	command.Flags().Float64Var(&options.Bottom, "margin-bottom", 0, "bottom margin in millimeters")
	command.Flags().Float64Var(&options.Left, "margin-left", 0, "left margin in millimeters")
}

func (options pageFlags) config() (pdf.PageConfig, error) {
	config := pdf.PageConfig{
		Size:        pdf.PageSize(options.Size),
		Orientation: pdf.Orientation(options.Orientation),
		Margins: pdf.Margins{
			Top: pdf.Millimeters(options.Top), Right: pdf.Millimeters(options.Right),
			Bottom: pdf.Millimeters(options.Bottom), Left: pdf.Millimeters(options.Left),
		},
	}
	if err := config.Validate(); err != nil {
		return pdf.PageConfig{}, err
	}
	return config, nil
}

func exportPDFArtifact(ctx context.Context, deps Dependencies, html []byte, descriptor margo.RuntimeDescriptor, executionID margo.ExecutionID, pageConfig pdf.PageConfig, engineOptions engineFlags, linkConfig pdfLinkConfig) ([]byte, error) {
	engine, _, err := selectEngine(ctx, deps.EngineProbe, engineOptions)
	if err != nil {
		return nil, err
	}
	result, err := engine.Export(ctx, pdf.Request{
		HTML: html, Runtime: descriptor, ExecutionID: executionID, Page: pageConfig,
		RelativeLinks: linkConfig.Policy, BaseURL: linkConfig.BaseURL,
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
