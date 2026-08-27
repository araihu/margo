package main

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/charts"
	"github.com/araihu/margo/deck"
	"github.com/araihu/margo/pdf"
	"github.com/spf13/cobra"
)

func newDeckCommand(deps Dependencies) *cobra.Command {
	formatValue := "html"
	output := outputOptions{Path: "-"}
	engineOptions := engineFlags{Mode: "auto"}
	pageOptions := pageFlags{Size: "A4", Orientation: "portrait"}
	var slideOptions slideGeometryFlags
	printChartData := false
	confidentialityBadge := ""
	paginationIconSymbol := ""
	paginationIconPlacement := ""
	paginationIconLabel := ""
	paginationIconDecorative := false
	diagnostics := string(diagnosticText)
	var policyOptions policyFlags
	command := &cobra.Command{
		Use:   "deck INPUT",
		Short: "Render the versioned Margo deck profile as HTML or PDF",
		Long: "Render the versioned Margo Marpit-compatible deck profile. Slides are\n" +
			"separated by thematic breaks; YAML frontmatter and closed directives select\n" +
			"themes, geometry, compositions, and presenter notes. Deck charts are static\n" +
			"and arbitrary HTML/CSS or remote assets are rejected. See\n" +
			"https://margo.araihu.com/cli/deck/ for the authoring catalog and examples.",
		Example: "  margo check slides.md --target deck\n" +
			"  mkdir -p build && margo deck slides.md --format html --output build/slides.html\n" +
			"  margo deck slides.md --format pdf --output build/slides.pdf --slide-size 16:9\n" +
			"  margo deck slides.md --format pdf --output - --print-chart-data > slides.pdf",
		Args: diagnosticExactArgs(1, &diagnostics),
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
			policy, err := policyOptions.load(command.Context(), deps.SourceReader)
			if err != nil {
				return reportCommandError(command, format, err)
			}
			artifact, err := renderDeckArtifact(command, deps, args[0], formatValue, engineOptions, pageOptions, slideOptions, printChartData, confidentialityBadge, paginationIconSymbol, paginationIconPlacement, paginationIconLabel, paginationIconDecorative, policy)
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
	command.Flags().BoolVar(&printChartData, "print-chart-data", false, "print accessible exact-data tables after charts")
	command.Flags().StringVar(&confidentialityBadge, "confidentiality-badge", "", "host-owned badge label shown before page ordinals")
	command.Flags().StringVar(&paginationIconSymbol, "pagination-icon", "", "host-owned Goshtoso icon symbol in the page ordinal cluster")
	command.Flags().StringVar(&paginationIconPlacement, "pagination-icon-placement", "", "pagination icon placement: before or after")
	command.Flags().StringVar(&paginationIconLabel, "pagination-icon-label", "", "accessible label for an informative pagination icon")
	command.Flags().BoolVar(&paginationIconDecorative, "pagination-icon-decorative", false, "hide the pagination icon from assistive technology")
	command.Flags().StringVar(&diagnostics, "diagnostics", string(diagnosticText), "diagnostic format: text or json")
	engineOptions.bind(command)
	pageOptions.bind(command)
	slideOptions.bind(command)
	policyOptions.bind(command)
	bindDiagnosticFlagErrors(command, &diagnostics)
	return command
}

type slideGeometryFlags struct {
	Size   string
	Width  float64
	Height float64
	Unit   string
}

func (options *slideGeometryFlags) bind(command *cobra.Command) {
	command.Flags().StringVar(&options.Size, "slide-size", "", "deck slide size: 16:9, 4:3, or custom")
	command.Flags().Float64Var(&options.Width, "slide-width", 0, "custom deck slide width")
	command.Flags().Float64Var(&options.Height, "slide-height", 0, "custom deck slide height")
	command.Flags().StringVar(&options.Unit, "slide-unit", "px", "custom deck slide unit: px, mm, cm, in, pt, pc, or Q")
}

func (options slideGeometryFlags) explicit(command *cobra.Command) bool {
	return command.Flags().Changed("slide-size") || command.Flags().Changed("slide-width") || command.Flags().Changed("slide-height") || command.Flags().Changed("slide-unit")
}

func (options slideGeometryFlags) geometry(command *cobra.Command) (deck.DeckGeometry, bool, error) {
	if !options.explicit(command) {
		return deck.DeckGeometry{}, false, nil
	}
	size := strings.TrimSpace(options.Size)
	if size == "16:9" || size == "4:3" {
		if command.Flags().Changed("slide-width") || command.Flags().Changed("slide-height") || command.Flags().Changed("slide-unit") {
			return deck.DeckGeometry{}, false, fmt.Errorf("cli.deck_geometry_conflict: preset slide size cannot include custom width, height, or unit")
		}
		geometry, err := deck.ParseDeckGeometry(size)
		return geometry, true, err
	}
	if size != "" && size != "custom" {
		return deck.DeckGeometry{}, false, fmt.Errorf("cli.deck_geometry_invalid: --slide-size must be 16:9, 4:3, or custom")
	}
	if size != "custom" {
		return deck.DeckGeometry{}, false, fmt.Errorf("cli.deck_geometry_invalid: custom slide width and height require --slide-size custom")
	}
	if options.Width <= 0 || options.Height <= 0 {
		return deck.DeckGeometry{}, false, fmt.Errorf("cli.deck_geometry_invalid: custom slide width and height are required")
	}
	if !command.Flags().Changed("slide-unit") {
		return deck.DeckGeometry{}, false, fmt.Errorf("cli.deck_geometry_invalid: --slide-unit is required for custom slide geometry")
	}
	value := strconv.FormatFloat(options.Width, 'f', -1, 64) + "x" + strconv.FormatFloat(options.Height, 'f', -1, 64) + options.Unit
	geometry, err := deck.ParseDeckGeometry(value)
	return geometry, true, err
}

func renderDeckArtifact(command *cobra.Command, deps Dependencies, input, format string, engineOptions engineFlags, pageOptions pageFlags, slideOptions slideGeometryFlags, printChartData bool, confidentialityBadge, paginationIconSymbol, paginationIconPlacement, paginationIconLabel string, paginationIconDecorative bool, policy *loadedPolicy) ([]byte, error) {
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
	compiler := compilerForPolicy(policy, policyTargetDeck, charts.WithPrintableAccessibleData(printChartData))
	metadataDocument, err := compiler.Compile(command.Context(), source)
	if err != nil {
		return nil, err
	}
	geometry, geometryExplicit, err := slideOptions.geometry(command)
	if err != nil {
		return nil, err
	}
	renderOptions := []deck.RenderOption(nil)
	if command.Flags().Changed("confidentiality-badge") {
		renderOptions = append(renderOptions, deck.WithConfidentialityBadge(confidentialityBadge))
	}
	if command.Flags().Changed("pagination-icon") || command.Flags().Changed("pagination-icon-placement") || command.Flags().Changed("pagination-icon-label") || command.Flags().Changed("pagination-icon-decorative") {
		renderOptions = append(renderOptions, deck.WithPaginationIcon(deck.PaginationIconConfig{
			Symbol: paginationIconSymbol, Placement: deck.PaginationIconPlacement(paginationIconPlacement),
			Label: paginationIconLabel, Decorative: paginationIconDecorative,
		}))
	}
	result, err := deck.Render(command.Context(), compiler, deck.RenderInput{
		Name: source.Name, Markdown: source.Content, BaseURL: baseURL, Geometry: geometry,
	}, renderOptions...)
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
	pageConfig, err := pageOptions.config(command, metadataDocument.Metadata())
	if err != nil {
		return nil, err
	}
	legacyPageExplicit := command.Flags().Changed("page-size") || command.Flags().Changed("orientation")
	if legacyPageExplicit {
		legacyWidth, legacyHeight, err := pageConfigCSSGeometry(pageConfig)
		if err != nil {
			return nil, err
		}
		if math.Abs(legacyWidth-result.Geometry().Width) > 0.0001 || math.Abs(legacyHeight-result.Geometry().Height) > 0.0001 {
			return nil, fmt.Errorf("cli.deck_geometry_conflict: legacy page geometry does not match the selected slide geometry")
		}
	}
	if geometryExplicit || result.Geometry().Preset != "16:9" || metadataDocument.Metadata().Margo.Page == nil {
		pageConfig, err = deckGeometryPageConfig(result.Geometry(), pageConfig.ImageOverflow)
		if err != nil {
			return nil, err
		}
	}
	geometryForArtifact := result.Geometry()
	return exportPDFArtifact(command.Context(), deps, html, descriptor, executionID, pageConfig, engineOptions, pdfLinkConfig{Policy: pdf.RelativeLinksStrip}, &geometryForArtifact)
}

func deckGeometryPageConfig(geometry deck.DeckGeometry, overflow pdf.ImageOverflowPolicy) (pdf.PageConfig, error) {
	widthMM := pdf.Millimeters(geometry.Width * 25.4 / 96)
	heightMM := pdf.Millimeters(geometry.Height * 25.4 / 96)
	config := pdf.PageConfig{Custom: &pdf.CustomPageSize{WidthMM: widthMM, HeightMM: heightMM}, ImageOverflow: overflow}
	if err := config.Validate(); err != nil {
		return pdf.PageConfig{}, err
	}
	return config, nil
}
