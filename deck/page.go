package deck

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/araihu/margo"
)

func renderDeckPage(metadata Metadata, theme margo.ThemeName, colorMode margo.ColorMode, lang string, geometry DeckGeometry, article []byte, requirements margo.HTMLRequirements, iconSprite []byte) ([]byte, error) {
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
	if lang == "" {
		lang = "en"
	}
	labels := localizedDeckLabels(lang)
	catalog, err := ThemeCatalog(theme, colorMode)
	if err != nil {
		return nil, err
	}
	fontDigest, err := bundledFontDigest(theme)
	if err != nil {
		return nil, err
	}
	fontCSS, err := embeddedDeckFontCSS()
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	_, _ = output.WriteString("<!doctype html><html lang=\"")
	_, _ = output.WriteString(html.EscapeString(lang))
	if colorMode == margo.ColorModeDark {
		// Goshtoso's dark utility and document selectors are keyed by the
		// canonical .dark class. The deck also exposes data-color-mode for
		// consumers, but that attribute alone leaves table body text on the
		// light --color-on-surface token in Chromium.
		_, _ = output.WriteString(`" class="dark`)
	}
	_, _ = output.WriteString("\" data-theme=\"")
	_, _ = output.WriteString(html.EscapeString(string(theme)))
	_, _ = output.WriteString("\" data-color-mode=\"")
	_, _ = output.WriteString(html.EscapeString(string(colorMode)))
	_, _ = output.WriteString("\" data-margo-font-bundle-digest=\"")
	_, _ = output.WriteString(html.EscapeString(fontDigest))
	_, _ = output.WriteString("\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
	scriptPolicy := "'self' 'unsafe-inline'"
	for _, requirement := range requirements.List() {
		if requirement.ID == "goshtoso.runtime.alpine" {
			// Goshtoso's bundled Alpine runtime evaluates trusted declarative
			// bindings from the rendered chart controls. Keep eval out of decks
			// that do not opt into those controls.
			scriptPolicy += " 'unsafe-eval'"
			break
		}
	}
	_, _ = output.WriteString(`<meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src 'self' data: blob:; style-src 'unsafe-inline'; script-src `)
	_, _ = output.WriteString(scriptPolicy)
	_, _ = output.WriteString(`; font-src data:">`)
	_, _ = output.WriteString("<title>")
	_, _ = output.WriteString(html.EscapeString(title))
	_, _ = output.WriteString("</title>")
	if description := strings.TrimSpace(metadata.Description); description != "" {
		_, _ = output.WriteString(`<meta name="description" content="`)
		_, _ = output.WriteString(html.EscapeString(description))
		_, _ = output.WriteString(`">`)
	}
	_, _ = output.WriteString(`<style data-margo-deck-fonts>`)
	_, _ = output.WriteString(strings.ReplaceAll(fontCSS, "</style", `<\/style`))
	_, _ = output.WriteString(`</style>`)
	_, _ = output.WriteString(`<style data-margo-deck-tokens>:root{`)
	_, _ = output.WriteString(themeCSS(catalog))
	_, _ = output.WriteString(fmt.Sprintf(";--margo-deck-width:%gpx;--margo-deck-height:%gpx", geometry.Width, geometry.Height))
	_, _ = output.WriteString(`}</style>`)
	_, _ = output.WriteString(`<style data-margo-deck-styles>`)
	_, _ = output.WriteString(strings.ReplaceAll(deckCSS, "</style", `<\/style`))
	_, _ = output.WriteString(`</style></head><body>`)
	if len(iconSprite) > 0 {
		_, _ = output.Write(iconSprite)
	}
	_, _ = output.WriteString(`<div class="margo-deck-stage">`)
	_, _ = output.Write(article)
	_, _ = output.WriteString(`<nav class="margo-deck-controls" aria-label="`)
	_, _ = output.WriteString(html.EscapeString(labels.Controls))
	_, _ = output.WriteString(`"><button type="button" data-margo-deck-previous aria-label="`)
	_, _ = output.WriteString(html.EscapeString(labels.Previous))
	_, _ = output.WriteString(`">`)
	_, _ = output.WriteString(html.EscapeString(labels.Previous))
	_, _ = output.WriteString(`</button><output aria-live="polite" data-margo-deck-status data-margo-label-slide="`)
	_, _ = output.WriteString(html.EscapeString(labels.Slide))
	_, _ = output.WriteString(`" data-margo-label-separator="`)
	_, _ = output.WriteString(html.EscapeString(labels.Separator))
	_, _ = output.WriteString(`"></output><button type="button" data-margo-deck-next aria-label="`)
	_, _ = output.WriteString(html.EscapeString(labels.Next))
	_, _ = output.WriteString(`">`)
	_, _ = output.WriteString(html.EscapeString(labels.Next))
	_, _ = output.WriteString(`</button><button type="button" data-margo-deck-print aria-label="`)
	_, _ = output.WriteString(html.EscapeString(labels.Print))
	_, _ = output.WriteString(`">`)
	_, _ = output.WriteString(html.EscapeString(labels.Print))
	_, _ = output.WriteString(`</button></nav></div>`)
	// Place extension dependencies after the body content. Inline runtimes such
	// as Alpine must see a live <body> during initialization; keeping them in
	// <head> makes the bundled chart controls race document parsing.
	_, _ = output.Write(dependencyMarkup.Bytes())
	_, _ = output.WriteString(`<script data-margo-deck-navigation>`)
	_, _ = output.WriteString(strings.ReplaceAll(deckJavaScript, "</script", `<\/script`))
	_, _ = output.WriteString(`</script></body></html>`)
	return append([]byte(nil), output.Bytes()...), nil
}

type deckLabelSet struct {
	Controls  string
	Previous  string
	Next      string
	Print     string
	Slide     string
	Separator string
}

func localizedDeckSlideLabel(lang string, index, total int) string {
	if strings.HasPrefix(strings.ToLower(lang), "pt") {
		return fmt.Sprintf("Slide %d de %d", index, total)
	}
	if strings.HasPrefix(strings.ToLower(lang), "es") {
		return fmt.Sprintf("Diapositiva %d de %d", index, total)
	}
	return fmt.Sprintf("Slide %d of %d", index, total)
}

func localizedDeckChapterLabel(lang string, number int) string {
	if strings.HasPrefix(strings.ToLower(lang), "pt") {
		return fmt.Sprintf("Capítulo %d", number)
	}
	if strings.HasPrefix(strings.ToLower(lang), "es") {
		return fmt.Sprintf("Capítulo %d", number)
	}
	return fmt.Sprintf("Chapter %d", number)
}

func localizedDeckLayoutLabel(lang, class string) string {
	if strings.HasPrefix(strings.ToLower(lang), "pt") {
		if class == "metrics" {
			return "Métricas"
		}
		return "Layout " + class
	}
	if strings.HasPrefix(strings.ToLower(lang), "es") {
		if class == "metrics" {
			return "Métricas"
		}
		return "Diseño " + class
	}
	if class == "metrics" {
		return "Metrics"
	}
	return "Layout " + class
}

func localizedDeckCompositionLabel(lang string, composition CompositionSpec) string {
	pt := strings.HasPrefix(strings.ToLower(lang), "pt")
	es := strings.HasPrefix(strings.ToLower(lang), "es")
	if pt {
		switch composition.Name {
		case "agenda":
			return "Agenda"
		case "steps":
			return "Etapas"
		case "compare-grid":
			return "Comparação"
		case "image-grid":
			return "Grade de imagens"
		case "media-split":
			return "Mídia e conteúdo"
		case "media-stage":
			return "Palco de mídia"
		case "highlight":
			return "Destaque"
		case "hero":
			return "Abertura"
		case "content":
			return "Conteúdo"
		}
	}
	if es {
		switch composition.Name {
		case "agenda":
			return "Agenda"
		case "steps":
			return "Etapas"
		case "compare-grid":
			return "Comparación"
		case "image-grid":
			return "Cuadrícula de imágenes"
		case "media-split":
			return "Medios y contenido"
		case "media-stage":
			return "Escenario multimedia"
		case "highlight":
			return "Destacado"
		case "hero":
			return "Apertura"
		case "content":
			return "Contenido"
		}
	}
	switch composition.Name {
	case "agenda":
		return "Agenda"
	case "steps":
		return "Steps"
	case "compare-grid":
		return "Comparison"
	case "image-grid":
		return "Image grid"
	case "media-split":
		return "Media and content"
	case "media-stage":
		return "Media stage"
	case "highlight":
		return "Highlight"
	case "hero":
		return "Hero"
	case "content":
		return "Content"
	default:
		return "Composition"
	}
}

func localizedDeckLabels(lang string) deckLabelSet {
	if strings.HasPrefix(strings.ToLower(lang), "pt") {
		return deckLabelSet{Controls: "Controles de slides", Previous: "Anterior", Next: "Próximo", Print: "Imprimir", Slide: "Slide", Separator: "de"}
	}
	if strings.HasPrefix(strings.ToLower(lang), "es") {
		return deckLabelSet{Controls: "Controles de diapositivas", Previous: "Anterior", Next: "Siguiente", Print: "Imprimir", Slide: "Diapositiva", Separator: "de"}
	}
	return deckLabelSet{Controls: "Slide controls", Previous: "Previous", Next: "Next", Print: "Print", Slide: "Slide", Separator: "of"}
}
