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

const (
	defaultDocumentMarginTop    = 24
	defaultDocumentMarginRight  = 22
	defaultDocumentMarginBottom = 26
	defaultDocumentMarginLeft   = 22
)

func defaultDocumentPageFlags() pageFlags {
	return pageFlags{
		Size:        string(pdf.PageA4),
		Orientation: string(pdf.Portrait),
		Top:         defaultDocumentMarginTop,
		Right:       defaultDocumentMarginRight,
		Bottom:      defaultDocumentMarginBottom,
		Left:        defaultDocumentMarginLeft,
	}
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
	pageOptions := defaultDocumentPageFlags()
	linkOptions := pdfLinkFlags{Policy: string(pdf.RelativeLinksStrip)}
	metadata := standaloneMetadataFlags{}
	policyOptions := policyFlags{}
	diagnostics := string(diagnosticText)
	command := &cobra.Command{
		Use:   "pdf INPUT",
		Short: "Render a PDF document",
		Long:  "Render a PDF document. Run margo check before conversion and margo doctor when no PDF engine is discovered.",
		Example: "  margo check guide.md\n" +
			"  margo doctor\n" +
			"  margo pdf guide.md --output guide.pdf\n" +
			"  margo pdf guide.md --output guide.pdf --base-url https://docs.example.com/guide/",
		Args: diagnosticExactArgs(1, &diagnostics),
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
			policy, err := policyOptions.load(command.Context(), deps.SourceReader)
			if err != nil {
				return reportCommandError(command, format, err)
			}
			artifact, err := renderPDF(command, deps, args[0], engineOptions, pageOptions, linkConfig, policy, metadata.standaloneOptions(command))
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
	metadata.bind(command)
	policyOptions.bind(command)
	bindDiagnosticFlagErrors(command, &diagnostics)
	return command
}

func renderPDF(command *cobra.Command, deps Dependencies, input string, engineOptions engineFlags, pageOptions pageFlags, linkConfig pdfLinkConfig, policy *loadedPolicy, standaloneOptions []margo.StandaloneOption) ([]byte, error) {
	compiled, err := compileStandaloneWithCompiler(command.Context(), deps, input, compilerForPolicy(policy, policyTargetPDF), margo.TargetPDF, standaloneOptions...)
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
	pageConfig, err := pageOptions.config(command, compiled.Render.Metadata())
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
	command.Flags().Float64Var(&options.Top, "margin-top", options.Top, "top margin in millimeters")
	command.Flags().Float64Var(&options.Right, "margin-right", options.Right, "right margin in millimeters")
	command.Flags().Float64Var(&options.Bottom, "margin-bottom", options.Bottom, "bottom margin in millimeters")
	command.Flags().Float64Var(&options.Left, "margin-left", options.Left, "left margin in millimeters")
}

func (options pageFlags) config(command *cobra.Command, metadata margo.Metadata) (pdf.PageConfig, error) {
	size := options.Size
	orientation := options.Orientation
	top, right, bottom, left := options.Top, options.Right, options.Bottom, options.Left
	if metadata.Margo.Page != nil {
		if !command.Flags().Changed("page-size") && metadata.Margo.Page.Size != "" {
			size = metadata.Margo.Page.Size
		}
		if !command.Flags().Changed("orientation") && metadata.Margo.Page.Orientation != "" {
			orientation = metadata.Margo.Page.Orientation
		}
		if margins := metadata.Margo.Page.Margins; margins != nil {
			if !command.Flags().Changed("margin-top") && margins.Top != nil {
				top = *margins.Top
			}
			if !command.Flags().Changed("margin-right") && margins.Right != nil {
				right = *margins.Right
			}
			if !command.Flags().Changed("margin-bottom") && margins.Bottom != nil {
				bottom = *margins.Bottom
			}
			if !command.Flags().Changed("margin-left") && margins.Left != nil {
				left = *margins.Left
			}
		}
	}
	config := pdf.PageConfig{
		Size:        pdf.PageSize(size),
		Orientation: pdf.Orientation(orientation),
		Margins: pdf.Margins{
			Top: pdf.Millimeters(top), Right: pdf.Millimeters(right),
			Bottom: pdf.Millimeters(bottom), Left: pdf.Millimeters(left),
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
