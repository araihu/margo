package deck

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/araihu/margo"
)

func renderDeckPage(metadata Metadata, theme margo.ThemeName, colorMode margo.ColorMode, article []byte, requirements margo.HTMLRequirements) ([]byte, error) {
	dependencies, err := margo.RenderHTMLDependencies(requirements, margo.HTMLDependenciesInline)
	if err != nil {
		return nil, err
	}
	var dependencyMarkup bytes.Buffer
	if err := dependencies.Render(context.Background(), &dependencyMarkup); err != nil {
		return nil, fmt.Errorf("deck.dependencies_render: %w", err)
	}
	title := strings.TrimSpace(metadata.Title)
	if title == "" {
		title = "Margo deck"
	}
	var output bytes.Buffer
	_, _ = output.WriteString("<!doctype html><html lang=\"en\" data-theme=\"")
	_, _ = output.WriteString(html.EscapeString(string(theme)))
	_, _ = output.WriteString("\" data-color-mode=\"")
	_, _ = output.WriteString(html.EscapeString(string(colorMode)))
	_, _ = output.WriteString("\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
	_, _ = output.WriteString(`<meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src 'self' data: blob:; style-src 'unsafe-inline'; script-src 'unsafe-inline'; font-src data:">`)
	_, _ = output.WriteString("<title>")
	_, _ = output.WriteString(html.EscapeString(title))
	_, _ = output.WriteString("</title>")
	if description := strings.TrimSpace(metadata.Description); description != "" {
		_, _ = output.WriteString(`<meta name="description" content="`)
		_, _ = output.WriteString(html.EscapeString(description))
		_, _ = output.WriteString(`">`)
	}
	_, _ = output.Write(dependencyMarkup.Bytes())
	_, _ = output.WriteString(`<style data-margo-deck-styles>`)
	_, _ = output.WriteString(strings.ReplaceAll(deckCSS, "</style", `<\/style`))
	_, _ = output.WriteString(`</style></head><body>`)
	_, _ = output.WriteString(`<nav class="margo-deck-controls" aria-label="Slide controls"><button type="button" data-margo-deck-previous>Previous</button><output aria-live="polite" data-margo-deck-status></output><button type="button" data-margo-deck-next>Next</button><button type="button" data-margo-deck-print>Print</button></nav>`)
	_, _ = output.Write(article)
	_, _ = output.WriteString(`<script data-margo-deck-navigation>`)
	_, _ = output.WriteString(strings.ReplaceAll(deckJavaScript, "</script", `<\/script`))
	_, _ = output.WriteString(`</script></body></html>`)
	return append([]byte(nil), output.Bytes()...), nil
}
