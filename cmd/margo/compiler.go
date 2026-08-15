package main

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/charts"
	"github.com/spf13/cobra"
)

func newCompiler(options ...margo.Option) *margo.Compiler {
	return newCompilerWithChartOptions(nil, options...)
}

func newCompilerWithChartOptions(chartOptions []charts.Option, options ...margo.Option) *margo.Compiler {
	chartOptions = append([]charts.Option{charts.WithExternalizedControlRuntime(true)}, chartOptions...)
	compilerOptions := []margo.Option{margo.WithExtension(charts.Extension(chartOptions...))}
	compilerOptions = append(compilerOptions, options...)
	return margo.New(compilerOptions...)
}

type compiledStandalone struct {
	HTML   []byte
	Render *margo.RenderResult
}

type standaloneMetadataFlags struct {
	Title    string
	Language string
}

func (options *standaloneMetadataFlags) bind(command *cobra.Command) {
	command.Flags().StringVar(&options.Title, "title", "", "override the document title (maximum 256 UTF-8 bytes)")
	command.Flags().StringVar(&options.Language, "lang", "", "override the document BCP 47 language tag")
}

func (options standaloneMetadataFlags) standaloneOptions(command *cobra.Command) []margo.StandaloneOption {
	result := make([]margo.StandaloneOption, 0, 2)
	if command.Flags().Changed("title") {
		result = append(result, margo.WithPageTitle(options.Title))
	}
	if command.Flags().Changed("lang") {
		result = append(result, margo.WithPageLanguage(options.Language))
	}
	return result
}

func compileStandalone(ctx context.Context, deps Dependencies, input string, options ...margo.StandaloneOption) (compiledStandalone, error) {
	return compileStandaloneWithCompiler(ctx, deps, input, newCompiler(), margo.TargetHTML, options...)
}

func compileStandaloneWithCompiler(ctx context.Context, deps Dependencies, input string, compiler *margo.Compiler, target margo.RenderTarget, options ...margo.StandaloneOption) (compiledStandalone, error) {
	source, err := readInput(ctx, deps.SourceReader, deps.Stdin, input)
	if err != nil {
		return compiledStandalone{}, err
	}
	if source.Name != "<stdin>" {
		absolute, pathErr := filepath.Abs(source.Name)
		if pathErr != nil {
			return compiledStandalone{}, fmt.Errorf("cli.input_path_invalid: %w", pathErr)
		}
		source.BaseURL = filepath.Dir(absolute)
	}
	document, err := compiler.Compile(ctx, source)
	if err != nil {
		return compiledStandalone{}, err
	}
	rendered, err := compiler.Render(ctx, document, margo.WithTableSort(margo.TableSortClient), margo.WithRenderTarget(target))
	if err != nil {
		return compiledStandalone{}, err
	}
	renderOptions := make([]any, len(options))
	for index := range options {
		renderOptions[index] = options[index]
	}
	component, err := margo.RenderStandalone(rendered, renderOptions...)
	if err != nil {
		return compiledStandalone{}, err
	}
	var output bytes.Buffer
	if err := component.Render(ctx, &output); err != nil {
		return compiledStandalone{}, fmt.Errorf("cli.html_serialize: %w", err)
	}
	materialized, err := materializeLocalImages(output.Bytes(), source.Name, deps.WorkingDirectory)
	if err != nil {
		return compiledStandalone{}, err
	}
	return compiledStandalone{HTML: materialized, Render: rendered}, nil
}
