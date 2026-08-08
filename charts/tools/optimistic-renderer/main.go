// Command optimistic-renderer creates a deterministic standalone HTML review
// artifact with the optional Goshtoso Charts extension enabled.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-h/templ"
	"github.com/araihu/goshtoso-charts/assets"
	goshtosoassets "github.com/araihu/goshtoso/assets"
	margo "github.com/araihu/margo"
	"github.com/araihu/margo/charts"
)

const (
	defaultTitle       = "Margo v0.0.1 optimistic benchmark with charts"
	defaultDescription = "Offline Margo Markdown benchmark with interactive Goshtoso Charts projections."
)

type generatorConfig struct {
	SourcePath       string
	ChartsSourcePath string
	OutputPath       string
	Title            string
	Description      string
	ColorMode        margo.ColorMode
}

func main() {
	var config generatorConfig
	flag.StringVar(&config.SourcePath, "source", "", "Markdown source path")
	flag.StringVar(&config.ChartsSourcePath, "charts-source", "", "optional Markdown appendix containing goshtosochart fences")
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
	if config.ChartsSourcePath != "" {
		chartsSourcePath, err := filepath.Abs(config.ChartsSourcePath)
		if err != nil {
			return fmt.Errorf("optimistic-renderer: resolve charts source: %w", err)
		}
		appendix, err := os.ReadFile(chartsSourcePath)
		if err != nil {
			return fmt.Errorf("optimistic-renderer: read charts source: %w", err)
		}
		source = append(append(source, '\n', '\n'), appendix...)
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

	compiler := margo.New(
		margo.WithHostPolicy(margo.Policy{
			RawHTML:     margo.RawHTMLSanitized,
			OutputBytes: margo.MaxOutputBytes,
		}),
		margo.WithExtension(charts.Extension()),
	)
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
			Header:    templ.Raw(`<span><strong>Margo</strong> · full feature benchmark</span><span>Markdown for Goshtoso + Charts</span>`),
			Footer:    templ.Raw(`<span>Source: ` + html.EscapeString(filepath.Base(sourcePath)) + `</span><span>theme modern · human artifact</span>`),
			Logo:      logo,
			LogoAlt:   "Margo mark used as a compact vector figure",
			Backdrop:  logo,
			Watermark: "Optimistic benchmark",
			Stamps:    []string{"v0.0.1", "optimistic", "charts", "human review"},
		}),
	)
	if err != nil {
		return fmt.Errorf("optimistic-renderer: assemble standalone: %w", err)
	}
	var output bytes.Buffer
	if err := component.Render(ctx, &output); err != nil {
		return fmt.Errorf("optimistic-renderer: serialize standalone: %w", err)
	}
	interactive, err := inlineChartControlRuntime(output.Bytes())
	if err != nil {
		return fmt.Errorf("optimistic-renderer: inline chart controls: %w", err)
	}
	if err := writeAtomic(outputPath, interactive); err != nil {
		return fmt.Errorf("optimistic-renderer: write output: %w", err)
	}
	fmt.Printf("wrote %d bytes to %s\n", len(interactive), outputPath)
	return nil
}

func inlineChartControlRuntime(markup []byte) ([]byte, error) {
	marker := `<script src="` + assets.ControlRuntimeURL + `" defer></script>`
	if !strings.Contains(string(markup), marker) {
		return append([]byte(nil), markup...), nil
	}
	chartRuntime, err := chartControlRuntime()
	if err != nil {
		return nil, err
	}
	withoutExternal := strings.ReplaceAll(string(markup), marker, "")
	goshtosoRuntime, err := inlineGoshtosoRuntime()
	if err != nil {
		return nil, err
	}
	injection := goshtosoRuntime + `<script data-margo-chart-controls-inline="v5">` + safeScriptText(chartRuntime) + `</script>`
	if index := strings.Index(withoutExternal, "</body>"); index >= 0 {
		withoutExternal = withoutExternal[:index] + injection + withoutExternal[index:]
	} else {
		withoutExternal += injection
	}
	return []byte(withoutExternal), nil
}

func inlineGoshtosoRuntime() (string, error) {
	manifest := goshtosoassets.DefaultRuntimeManifest()
	wanted := map[goshtosoassets.RuntimeAssetRole]bool{
		goshtosoassets.RuntimeRoleAlpineFocus: true,
		goshtosoassets.RuntimeRoleFirstParty:  true,
		goshtosoassets.RuntimeRoleAlpineJS:    true,
	}
	var output strings.Builder
	for _, dependency := range manifest.Dependencies {
		if !wanted[dependency.Role] {
			continue
		}
		runtime, err := goshtosoRuntimeAsset(dependency.LocalURL)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&output, `<script data-margo-goshtoso-runtime=%q>`, string(dependency.Role))
		output.WriteString(safeScriptText(runtime))
		output.WriteString(`</script>`)
	}
	return output.String(), nil
}

func goshtosoRuntimeAsset(path string) ([]byte, error) {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	recorder := httptest.NewRecorder()
	goshtosoassets.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		return nil, fmt.Errorf("Goshtoso asset handler returned HTTP %d for %s", recorder.Code, path)
	}
	return append([]byte(nil), recorder.Body.Bytes()...), nil
}

func safeScriptText(data []byte) string {
	return strings.ReplaceAll(string(data), "</script", "<\\/script")
}

func chartControlRuntime() ([]byte, error) {
	request := httptest.NewRequest(http.MethodGet, assets.ControlRuntimeURL, nil)
	recorder := httptest.NewRecorder()
	assets.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		return nil, fmt.Errorf("asset handler returned HTTP %d for %s", recorder.Code, assets.ControlRuntimeURL)
	}
	return append([]byte(nil), recorder.Body.Bytes()...), nil
}

func writeAtomic(path string, data []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".margo-render-charts-*")
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
