// Command optimistic-renderer creates a deterministic standalone HTML review
// artifact from one Markdown source file. PDF printing remains a separate
// browser-owned step so this tool never downloads or starts a renderer.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html"
	"log"
	"os"
	"path/filepath"

	"github.com/a-h/templ"
	"github.com/araihu/margo"
)

const (
	defaultTitle       = "Margo v0.0.1 optimistic benchmark"
	defaultDescription = "Exhaustive human benchmark for Margo Markdown, HTML, PDF, extensions, and document composition."
)

type generatorConfig struct {
	SourcePath  string
	OutputPath  string
	Title       string
	Description string
	ColorMode   margo.ColorMode
}

func main() {
	var config generatorConfig
	flag.StringVar(&config.SourcePath, "source", "", "Markdown source path")
	flag.StringVar(&config.OutputPath, "output", "", "standalone HTML output path")
	flag.StringVar(&config.Title, "title", defaultTitle, "document title")
	flag.StringVar(&config.Description, "description", defaultDescription, "document description")
	flag.Var(colorModeFlag{value: &config.ColorMode}, "color-mode", "light or dark (default: light)")
	flag.Parse()
	if config.SourcePath == "" || config.OutputPath == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := generateHTML(context.Background(), config); err != nil {
		log.Fatal(err)
	}
}

type colorModeFlag struct{ value *margo.ColorMode }

func (f colorModeFlag) String() string {
	if f.value == nil || *f.value == "" {
		return string(margo.ColorModeLight)
	}
	return string(*f.value)
}

func (f colorModeFlag) Set(value string) error {
	*f.value = margo.ColorMode(value)
	return nil
}

func generateHTML(ctx context.Context, config generatorConfig) error {
	if ctx == nil {
		return fmt.Errorf("optimistic-renderer: context is required")
	}
	sourcePath, err := filepath.Abs(config.SourcePath)
	if err != nil {
		return fmt.Errorf("optimistic-renderer: resolve source: %w", err)
	}
	outputPath, err := filepath.Abs(config.OutputPath)
	if err != nil {
		return fmt.Errorf("optimistic-renderer: resolve output: %w", err)
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("optimistic-renderer: read source: %w", err)
	}
	title := config.Title
	if title == "" {
		title = defaultTitle
	}
	description := config.Description
	if description == "" {
		description = defaultDescription
	}
	colorMode := config.ColorMode
	if colorMode == "" {
		colorMode = margo.ColorModeLight
	}

	compiler := margo.New(margo.WithHostPolicy(margo.Policy{
		RawHTML:     margo.RawHTMLSanitized,
		OutputBytes: margo.MaxOutputBytes,
	}))
	document, err := compiler.Compile(ctx, margo.Source{
		Name:    filepath.Base(sourcePath),
		Content: source,
	})
	if err != nil {
		return fmt.Errorf("optimistic-renderer: compile source: %w", err)
	}
	result, err := compiler.Render(ctx, document, margo.WithTableSort(margo.TableSortClient))
	if err != nil {
		return fmt.Errorf("optimistic-renderer: render source: %w", err)
	}
	logo, err := margo.EmbeddedAsset("logo.svg")
	if err != nil {
		return fmt.Errorf("optimistic-renderer: load logo: %w", err)
	}
	component, err := margo.RenderStandalone(result,
		margo.WithPageTitle(title),
		margo.WithPageDescription(description),
		margo.WithTableOfContents(),
		margo.WithStandaloneColorMode(colorMode),
		margo.WithBrand(margo.Brand{
			Header:    templ.Raw(`<span><strong>Margo</strong> · full feature benchmark</span><span>Markdown for Goshtoso</span>`),
			Footer:    templ.Raw(`<span>Source: ` + html.EscapeString(filepath.Base(sourcePath)) + `</span><span>theme modern · human artifact</span>`),
			Logo:      logo,
			LogoAlt:   "Margo mark used as a compact vector figure",
			Backdrop:  logo,
			Watermark: "Optimistic benchmark",
			Stamps:    []string{"v0.0.1", "optimistic", "review required"},
		}),
	)
	if err != nil {
		return fmt.Errorf("optimistic-renderer: assemble standalone: %w", err)
	}
	var output bytes.Buffer
	if err := component.Render(ctx, &output); err != nil {
		return fmt.Errorf("optimistic-renderer: serialize standalone: %w", err)
	}
	if err := writeAtomic(outputPath, output.Bytes()); err != nil {
		return fmt.Errorf("optimistic-renderer: write output: %w", err)
	}
	fmt.Printf("wrote %d bytes to %s\n", output.Len(), outputPath)
	return nil
}

func writeAtomic(path string, data []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".margo-render-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
