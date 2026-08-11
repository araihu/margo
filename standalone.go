package margo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html"
	"strings"

	"github.com/a-h/templ"
	nethtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const standalonePrintPreparationScript = `<script data-margo-print-preparation>
(() => {
  "use strict";
  const mermaidSources = () => [...document.querySelectorAll(".margo-mermaid__source")];
  const article = document.querySelector(".goshtoso-document .margo-document");
  if (!article && mermaidSources().length === 0) return;

  const originalDetailsState = new WeakMap();
  const staticPrintReplacements = [];

  const restorePrintState = () => {
    for (let index = staticPrintReplacements.length - 1; index >= 0; index -= 1) {
      const { original, replacement, preserveChildren } = staticPrintReplacements[index];
      if (!replacement.isConnected) continue;
      if (preserveChildren) {
        while (replacement.firstChild) original.appendChild(replacement.firstChild);
      }
      replacement.replaceWith(original);
    }
    staticPrintReplacements.length = 0;
    for (const details of mermaidSources()) {
      if (!originalDetailsState.has(details)) continue;
      details.open = originalDetailsState.get(details);
      originalDetailsState.delete(details);
    }
  };

  const replaceForStaticPrint = (element, kind, text) => {
    const replacement = document.createElement("span");
    replacement.dataset.margoPrintStatic = kind;
    replacement.className = "margo-print-" + kind;
    const preserveChildren = text === undefined;
    if (preserveChildren) {
      for (const attribute of element.attributes) {
        const name = attribute.name.toLowerCase();
        if (name === "id" || name === "class" || name === "style" || name === "title" || name === "lang" || name === "dir" || name.startsWith("data-") || name.startsWith("aria-")) {
          replacement.setAttribute(attribute.name, attribute.value);
        }
      }
      replacement.classList.add("margo-print-" + kind);
      replacement.dataset.margoPrintStatic = kind;
      while (element.firstChild) replacement.appendChild(element.firstChild);
    } else {
      replacement.textContent = text;
    }
    staticPrintReplacements.push({ original: element, replacement, preserveChildren });
    element.replaceWith(replacement);
  };

  const prepare = () => {
    for (const details of mermaidSources()) {
      if (!originalDetailsState.has(details)) originalDetailsState.set(details, details.open);
      details.open = true;
    }
    if (!article || staticPrintReplacements.length > 0) return;
    for (const element of [...article.querySelectorAll("strong, b")]) {
      replaceForStaticPrint(element, "strong");
    }
    for (const element of [...article.querySelectorAll("em, i")]) {
      replaceForStaticPrint(element, "emphasis");
    }
    for (const button of [...article.querySelectorAll(".margo-table-sort-button")]) {
      replaceForStaticPrint(button, "table-label", button.textContent);
    }
    for (const checkbox of [...article.querySelectorAll('input[type="checkbox"]')]) {
      replaceForStaticPrint(checkbox, "checkbox", checkbox.checked ? "☑" : "☐");
    }
  };

  window.margoPreparePrint = prepare;
  window.margoRestorePrintState = restorePrintState;
  window.addEventListener("beforeprint", prepare);
  window.addEventListener("afterprint", restorePrintState);
  const printMedia = window.matchMedia("print");
  if (typeof printMedia.addEventListener === "function") {
    printMedia.addEventListener("change", (event) => event.matches ? prepare() : restorePrintState());
  }
  if (printMedia.matches) prepare();
})();
</script>`

type standaloneConfig struct {
	lang            string
	title           string
	description     string
	theme           ThemeName
	colorMode       ColorMode
	tokens          map[DocumentToken]string
	brand           Brand
	assetOverrides  map[string]AssetRef
	tableOfContents bool
}

// StandaloneOption configures the self-contained HTML shell.
type StandaloneOption func(*standaloneConfig) error

func defaultStandaloneConfig(metadata HTMLMetadata) (standaloneConfig, error) {
	tokens, err := defaultThemeTokens(ThemeModern)
	if err != nil {
		return standaloneConfig{}, err
	}
	title := metadata.Title
	if title == "" {
		title = "Margo document"
	}
	lang := metadata.Language
	if lang == "" {
		lang = "en"
	}
	return standaloneConfig{
		lang: lang, title: title, description: metadata.Description,
		theme: ThemeModern, colorMode: ColorModeLight, tokens: tokens,
		assetOverrides: make(map[string]AssetRef),
	}, nil
}

// WithStandaloneColorMode selects the light or dark Goshtoso token family for
// both screen rendering and print/PDF projection.
func WithStandaloneColorMode(mode ColorMode) StandaloneOption {
	return func(config *standaloneConfig) error {
		if err := validateColorMode(mode); err != nil {
			return err
		}
		config.colorMode = mode
		return nil
	}
}

// WithStandaloneTheme selects the closed theme set for standalone output.
func WithStandaloneTheme(theme ThemeName) StandaloneOption {
	return func(config *standaloneConfig) error {
		tokens, err := defaultThemeTokens(theme)
		if err != nil {
			return err
		}
		config.theme, config.tokens = theme, tokens
		return nil
	}
}

// WithPageTitle sets the escaped document title.
func WithPageTitle(title string) StandaloneOption {
	return func(config *standaloneConfig) error {
		if strings.TrimSpace(title) == "" || len([]byte(title)) > 256 {
			return newDiagnosticError(Diagnostic{
				Code: "html.metadata_invalid", Severity: SeverityError, Pointer: "/title",
				Message: "standalone title must contain text and be at most 256 UTF-8 bytes",
				Hint:    "Use a non-empty title of at most 256 UTF-8 bytes.",
			})
		}
		config.title = title
		return nil
	}
}

// WithPageLanguage sets the document language using a BCP 47 language tag.
func WithPageLanguage(language string) StandaloneOption {
	normalized := strings.TrimSpace(language)
	return func(config *standaloneConfig) error {
		if !sourceLanguagePattern.MatchString(normalized) {
			return newDiagnosticError(Diagnostic{
				Code: "html.metadata_invalid", Severity: SeverityError, Pointer: "/language",
				Message: "standalone language must be a valid BCP 47 language tag",
				Hint:    "Use a BCP 47 language tag such as en or pt-BR.",
			})
		}
		config.lang = normalized
		return nil
	}
}

// WithPageDescription sets the optional escaped description.
func WithPageDescription(description string) StandaloneOption {
	return func(config *standaloneConfig) error {
		if len([]byte(description)) > 512 {
			return fmt.Errorf("margo: standalone description is too long")
		}
		config.description = description
		return nil
	}
}

// WithTableOfContents inserts one deterministic navigation landmark before the
// article. Entries cover heading levels two through four and reuse compiled IDs.
func WithTableOfContents() StandaloneOption {
	return func(config *standaloneConfig) error {
		config.tableOfContents = true
		return nil
	}
}

// WithThemeTokens applies only the supported, bounded token keys.
func WithThemeTokens(tokens map[DocumentToken]string) StandaloneOption {
	return func(config *standaloneConfig) error {
		if err := validateThemeTokens(tokens); err != nil {
			return err
		}
		for key, value := range tokens {
			config.tokens[key] = value
		}
		return nil
	}
}

// WithBrand applies trusted header/footer components and validated declarative
// brand values.
func WithBrand(brand Brand) StandaloneOption {
	return func(config *standaloneConfig) error {
		if err := brand.Validate(); err != nil {
			return err
		}
		config.brand = brand.clone()
		return nil
	}
}

// WithAssetOverride supplies already-materialized bytes for one embedded asset.
func WithAssetOverride(name string, asset AssetRef) StandaloneOption {
	return func(config *standaloneConfig) error {
		if name != asset.Path {
			return fmt.Errorf("margo: asset override name/path mismatch")
		}
		if err := asset.validate(); err != nil {
			return err
		}
		config.assetOverrides[name] = asset.clone()
		return nil
	}
}

// RenderStandalone assembles a deterministic, offline HTML component. The
// variadic any accepts both standalone options and the existing compiler
// WithTheme option for ergonomic compatibility; unsupported compiler options
// are rejected.
func RenderStandalone(result *RenderResult, options ...any) (templ.Component, error) {
	if result == nil || result.Content() == nil {
		return nil, fmt.Errorf("margo: standalone render requires a result")
	}
	editorial, err := RenderHTML(result)
	if err != nil {
		return nil, err
	}
	config, err := defaultStandaloneConfig(editorial.Metadata())
	if err != nil {
		return nil, err
	}
	for index, option := range options {
		switch typed := option.(type) {
		case StandaloneOption:
			if typed == nil {
				return nil, fmt.Errorf("margo: nil standalone option at index %d", index)
			}
			if err := typed(&config); err != nil {
				return nil, err
			}
		case Option:
			// Existing WithTheme is an Option. Apply it to a temporary config
			// and import only its immutable theme value.
			compilerConfig := newCompilerConfig()
			if err := typed(&compilerConfig); err != nil {
				return nil, err
			}
			if theme, ok := compilerConfig.values["theme"].(string); ok {
				if err := WithStandaloneTheme(ThemeName(theme))(&config); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("margo: compiler option is not valid for standalone output")
			}
		default:
			return nil, fmt.Errorf("margo: unsupported standalone option %T at index %d", option, index)
		}
	}
	if err := config.brand.Validate(); err != nil {
		return nil, err
	}
	asset, ok := config.assetOverrides["document.css"]
	if !ok {
		asset, err = EmbeddedAsset("document.css")
		if err != nil {
			return nil, err
		}
	}
	shellAsset, ok := config.assetOverrides["standalone.css"]
	if !ok {
		shellAsset, err = EmbeddedAsset("standalone.css")
		if err != nil {
			return nil, err
		}
	}
	content, err := renderComponentBytes(editorial.Fragment())
	if err != nil {
		return nil, fmt.Errorf("margo: render standalone content: %w", err)
	}
	hash := sha256.Sum256(append([]byte("margo/standalone-document/v1\n"), content...))
	fingerprint := hex.EncodeToString(hash[:])
	asset = materializedStandaloneAsset(asset, []byte(applyThemeTokens(string(asset.Content), config.tokens)))
	shellAsset = materializedStandaloneAsset(shellAsset, []byte(applyThemeTokens(string(shellAsset.Content), config.tokens)))
	toc := templ.Component(nil)
	if config.tableOfContents {
		toc, err = tableOfContentsComponent(content)
		if err != nil {
			return nil, err
		}
	}
	logoURL := assetDataURL(config.brand.Logo)
	backdropURL := assetDataURL(config.brand.Backdrop)
	requirements := editorial.Requirements().List()
	for index := range requirements {
		if requirements[index].ID == "margo.document.styles" {
			requirements[index].Inline = asset.clone()
		}
	}
	merged, err := mergeHTMLRequirements(requirements)
	if err != nil {
		return nil, err
	}
	standaloneHTML := *editorial
	standaloneHTML.metadata.Title = config.title
	standaloneHTML.metadata.Description = config.description
	standaloneHTML.metadata.Language = config.lang
	standaloneHTML.requirements = merged
	standaloneHTML.fragmentBytes = append([]byte(nil), content...)
	body := standalonePublicationBody(fingerprint, editorial.Fingerprint().String(), config.brand, logoURL, backdropURL, toc, standaloneHTML.Fragment())
	return RenderHTMLPage(&standaloneHTML, HTMLPageInput{
		Theme: config.theme, ColorMode: config.colorMode,
		DependencyMode: HTMLDependenciesInline, ThemeStylesheet: shellAsset,
		body: body,
		legacyStyles: map[string]string{
			"goshtoso.styles": "goshtoso", "margo.document.styles": "document",
			"margo.theme." + string(config.theme): "shell",
		},
	})
}

// Standalone is a short alias for RenderStandalone.
func Standalone(result *RenderResult, options ...any) (templ.Component, error) {
	return RenderStandalone(result, options...)
}

func renderComponentBytes(component templ.Component) ([]byte, error) {
	var buffer bytes.Buffer
	if err := component.Render(nilContext(), &buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func nilContext() context.Context { return context.Background() }

func applyThemeTokens(css string, tokens map[DocumentToken]string) string {
	declarations := make([]string, 0, len(tokens))
	for _, token := range []DocumentToken{TokenFontBody, TokenFontHeading, TokenContentWidth, TokenLineHeight, TokenCodeTheme, TokenPageBackground} {
		if value, ok := tokens[token]; ok {
			declarations = append(declarations, fmt.Sprintf("%s:%s", token, value))
		}
	}
	return strings.ReplaceAll(css, "/* MARGO_THEME_TOKENS */", strings.Join(declarations, ";")+";")
}

func materializedStandaloneAsset(asset AssetRef, content []byte) AssetRef {
	digest := sha256.Sum256(content)
	asset.Content = append([]byte(nil), content...)
	asset.SHA256 = hex.EncodeToString(digest[:])
	return asset
}

func assetDataURL(asset AssetRef) string {
	if len(asset.Content) == 0 || asset.MediaType == "" {
		return ""
	}
	return "data:" + asset.MediaType + ";base64," + base64.StdEncoding.EncodeToString(asset.Content)
}

func tableOfContentsComponent(content []byte) (templ.Component, error) {
	contextNode := &nethtml.Node{Type: nethtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := nethtml.ParseFragment(bytes.NewReader(content), contextNode)
	if err != nil {
		return nil, fmt.Errorf("margo: parse standalone content for table of contents: %w", err)
	}
	type entry struct {
		level int
		id    string
		label string
	}
	entries := make([]entry, 0)
	var walk func(*nethtml.Node)
	walk = func(node *nethtml.Node) {
		if node.Type == nethtml.ElementNode && len(node.Data) == 2 && node.Data[0] == 'h' && node.Data[1] >= '2' && node.Data[1] <= '4' {
			id := ""
			for _, attribute := range node.Attr {
				if attribute.Key == "id" {
					id = attribute.Val
				}
			}
			if id != "" {
				entries = append(entries, entry{level: int(node.Data[1] - '0'), id: id, label: nodeText(node)})
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	for _, node := range nodes {
		walk(node)
	}
	var markup strings.Builder
	markup.WriteString(`<nav class="goshtoso-document__toc" aria-label="Table of contents"><p class="goshtoso-document__toc-title">Contents</p><ol>`)
	for _, entry := range entries {
		fmt.Fprintf(&markup, `<li data-level="%d"><a href="#%s">%s</a></li>`, entry.level, html.EscapeString(entry.id), html.EscapeString(entry.label))
	}
	markup.WriteString(`</ol></nav>`)
	return templ.Raw(markup.String()), nil
}

func nodeText(node *nethtml.Node) string {
	var text strings.Builder
	var walk func(*nethtml.Node)
	walk = func(current *nethtml.Node) {
		if current.Type == nethtml.TextNode {
			text.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return strings.TrimSpace(text.String())
}
