package main

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/charts"
)

func newCompiler() *margo.Compiler {
	return margo.New(margo.WithExtension(charts.Extension(charts.WithExternalizedControlRuntime(true))))
}

type compiledStandalone struct {
	HTML   []byte
	Render *margo.RenderResult
}

func compileStandalone(ctx context.Context, deps Dependencies, input string) (compiledStandalone, error) {
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
	compiler := newCompiler()
	document, err := compiler.Compile(ctx, source)
	if err != nil {
		return compiledStandalone{}, err
	}
	rendered, err := compiler.Render(ctx, document, margo.WithTableSort(margo.TableSortClient))
	if err != nil {
		return compiledStandalone{}, err
	}
	component, err := margo.RenderStandalone(rendered)
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
