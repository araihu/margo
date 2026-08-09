package margo

import (
	"bytes"
	"context"
	"fmt"
	stdhtml "html"
	"io"
	"strings"
	"time"
	"unicode"

	"github.com/a-h/templ"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type HTMLMetadata struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Language    string   `json:"language"`
	Slug        string   `json:"slug"`
	Authors     []string `json:"authors,omitempty"`
	PublishedAt string   `json:"publishedAt,omitempty"`
	ModifiedAt  string   `json:"modifiedAt,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

func (m HTMLMetadata) clone() HTMLMetadata {
	m.Authors = append([]string(nil), m.Authors...)
	m.Tags = append([]string(nil), m.Tags...)
	return m
}

type HTMLResult struct {
	fragmentBytes []byte
	plainText     string
	metadata      HTMLMetadata
	requirements  HTMLRequirements
	diagnostics   []Diagnostic
	fingerprint   HTMLFingerprint
}

type htmlConfig struct {
	header bool
}

type HTMLOption func(*htmlConfig) error

func WithHTMLHeader() HTMLOption {
	return func(config *htmlConfig) error {
		config.header = true
		return nil
	}
}

func RenderHTML(result *RenderResult, options ...HTMLOption) (*HTMLResult, error) {
	if result == nil || result.Content() == nil {
		return nil, htmlError("html.result_required", "render result and content are required")
	}
	config := htmlConfig{}
	for index, option := range options {
		if option == nil {
			return nil, htmlError("html.metadata_invalid", fmt.Sprintf("nil HTML option at index %d", index))
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}

	metadata, err := normalizeHTMLResultMetadata(result.Metadata())
	if err != nil {
		return nil, err
	}
	fragment, err := renderHTMLComponentBytes(result.Content())
	if err != nil {
		return nil, fmt.Errorf("html.fragment_render: %w", err)
	}
	inspection, err := inspectHTMLFragment(fragment)
	if err != nil {
		return nil, err
	}
	if metadata.Title == "" {
		metadata.Title = inspection.firstH1
	}
	diagnostics := result.Diagnostics()
	if metadata.Title != "" && inspection.firstH1 != "" && metadata.Title != inspection.firstH1 {
		diagnostics = append(diagnostics, Diagnostic{
			Code: "html.title_conflict", Severity: SeverityInfo,
			Message: fmt.Sprintf("metadata title %q differs from body heading %q", metadata.Title, inspection.firstH1),
		})
	}
	if config.header && inspection.firstH1 == "" && metadata.Title != "" {
		fragment, err = insertHTMLHeader(fragment, metadata.Title)
		if err != nil {
			return nil, err
		}
		inspection, err = inspectHTMLFragment(fragment)
		if err != nil {
			return nil, err
		}
	}
	requirements := result.projectedHTMLRequirements()
	fingerprint, err := htmlFingerprint(fragment, metadata, requirements, config)
	if err != nil {
		return nil, err
	}
	return &HTMLResult{
		fragmentBytes: append([]byte(nil), fragment...),
		plainText:     inspection.plainText,
		metadata:      metadata.clone(),
		requirements:  HTMLRequirements{requirements: requirements.List()},
		diagnostics:   cloneDiagnostics(diagnostics),
		fingerprint:   fingerprint,
	}, nil
}

func (r *HTMLResult) Fragment() templ.Component {
	if r == nil {
		return nil
	}
	data := append([]byte(nil), r.fragmentBytes...)
	return templ.ComponentFunc(func(_ context.Context, out io.Writer) error {
		_, err := out.Write(data)
		return err
	})
}

func (r *HTMLResult) PlainText() string {
	if r == nil {
		return ""
	}
	return r.plainText
}

func (r *HTMLResult) Metadata() HTMLMetadata {
	if r == nil {
		return HTMLMetadata{}
	}
	return r.metadata.clone()
}

func (r *HTMLResult) Requirements() HTMLRequirements {
	if r == nil {
		return HTMLRequirements{}
	}
	return HTMLRequirements{requirements: r.requirements.List()}
}

func (r *HTMLResult) Diagnostics() []Diagnostic {
	if r == nil {
		return nil
	}
	return cloneDiagnostics(r.diagnostics)
}

func (r *HTMLResult) Fingerprint() HTMLFingerprint {
	if r == nil {
		return HTMLFingerprint{}
	}
	return r.fingerprint
}

func renderHTMLComponentBytes(component templ.Component) ([]byte, error) {
	var buffer bytes.Buffer
	if err := component.Render(context.Background(), &buffer); err != nil {
		return nil, err
	}
	return append([]byte(nil), buffer.Bytes()...), nil
}

type htmlFragmentInspection struct {
	firstH1   string
	plainText string
}

func inspectHTMLFragment(fragment []byte) (htmlFragmentInspection, error) {
	contextNode := &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := xhtml.ParseFragment(bytes.NewReader(fragment), contextNode)
	if err != nil {
		return htmlFragmentInspection{}, htmlError("html.metadata_invalid", fmt.Sprintf("invalid HTML fragment: %v", err))
	}
	var article *xhtml.Node
	for _, node := range nodes {
		if node.Type == xhtml.TextNode && strings.TrimSpace(node.Data) == "" {
			continue
		}
		if node.Type != xhtml.ElementNode || node.Data != "article" || article != nil {
			return htmlFragmentInspection{}, htmlError("html.metadata_invalid", "HTML fragment must contain exactly one top-level article")
		}
		article = node
	}
	if article == nil || !htmlClassContains(article, "margo-document") {
		return htmlFragmentInspection{}, htmlError("html.metadata_invalid", "HTML fragment must contain one margo-document article")
	}
	articleCount := 0
	var firstH1 string
	var text strings.Builder
	var walk func(*xhtml.Node, bool, int) error
	walk = func(node *xhtml.Node, skip bool, semanticDepth int) error {
		semanticText := false
		if node.Type == xhtml.ElementNode {
			switch node.Data {
			case "html", "head", "body", "script":
				return htmlError("html.metadata_invalid", fmt.Sprintf("HTML fragment contains forbidden <%s>", node.Data))
			case "style":
				if !htmlAttributeEquals(node, "data-margo-extension-style", "charts") {
					return htmlError("html.metadata_invalid", "HTML fragment contains an unowned <style>")
				}
				skip = true
			case "article":
				articleCount++
			case "template", "button", "svg", "canvas":
				skip = true
			case "h1":
				if firstH1 == "" {
					firstH1 = normalizedNodeText(node)
				}
			}
			if htmlAttributeEquals(node, "aria-hidden", "true") {
				skip = true
			}
			semanticText = isHTMLTextElement(node.Data)
			if semanticText {
				semanticDepth++
			}
			if node.Data == "img" && !skip && semanticDepth > 0 {
				if alt, found := htmlAttributeValue(node, "alt"); found {
					text.WriteString(alt)
					text.WriteByte(' ')
				}
			}
		}
		if node.Type == xhtml.TextNode && !skip && semanticDepth > 0 {
			text.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if err := walk(child, skip, semanticDepth); err != nil {
				return err
			}
		}
		if semanticText && !skip {
			text.WriteByte(' ')
		}
		return nil
	}
	if err := walk(article, false, 0); err != nil {
		return htmlFragmentInspection{}, err
	}
	if articleCount != 1 {
		return htmlFragmentInspection{}, htmlError("html.metadata_invalid", "HTML fragment must contain exactly one article")
	}
	return htmlFragmentInspection{firstH1: firstH1, plainText: normalizeHTMLText(text.String())}, nil
}

func htmlAttributeEquals(node *xhtml.Node, name, expected string) bool {
	value, found := htmlAttributeValue(node, name)
	return found && value == expected
}

func htmlAttributeValue(node *xhtml.Node, name string) (string, bool) {
	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return attribute.Val, true
		}
	}
	return "", false
}

func htmlClassContains(node *xhtml.Node, expected string) bool {
	for _, attribute := range node.Attr {
		if attribute.Key != "class" {
			continue
		}
		for _, class := range strings.Fields(attribute.Val) {
			if class == expected {
				return true
			}
		}
	}
	return false
}

func normalizedNodeText(node *xhtml.Node) string {
	var text strings.Builder
	var visit func(*xhtml.Node)
	visit = func(current *xhtml.Node) {
		if current.Type == xhtml.TextNode {
			text.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(node)
	return normalizeHTMLText(text.String())
}

func isHTMLTextElement(tag string) bool {
	switch tag {
	case "address", "caption", "code", "dd", "dt", "figcaption", "h1", "h2", "h3", "h4", "h5", "h6", "li", "p", "pre", "td", "th":
		return true
	default:
		return false
	}
}

func insertHTMLHeader(fragment []byte, title string) ([]byte, error) {
	articleStart := bytes.Index(fragment, []byte("<article"))
	if articleStart < 0 {
		return nil, htmlError("html.metadata_invalid", "HTML article start is missing")
	}
	openingEnd := bytes.IndexByte(fragment[articleStart:], '>')
	if openingEnd < 0 {
		return nil, htmlError("html.metadata_invalid", "HTML article opening tag is malformed")
	}
	openingEnd += articleStart + 1
	header := []byte(`<header class="margo-document__header"><h1>` + stdhtml.EscapeString(title) + `</h1></header>`)
	result := make([]byte, 0, len(fragment)+len(header))
	result = append(result, fragment[:openingEnd]...)
	result = append(result, header...)
	result = append(result, fragment[openingEnd:]...)
	return result, nil
}

func normalizeHTMLResultMetadata(metadata Metadata) (HTMLMetadata, error) {
	result := HTMLMetadata{
		Title: normalizeHTMLText(metadata.Title), Description: normalizeHTMLText(metadata.Description),
		Language: strings.TrimSpace(metadata.Language), Slug: strings.TrimSpace(metadata.Slug),
		PublishedAt: strings.TrimSpace(metadata.PublishedAt), ModifiedAt: strings.TrimSpace(metadata.ModifiedAt),
	}
	if len([]byte(result.Title)) > 256 || len([]byte(result.Description)) > 512 || len([]byte(result.Language)) > 64 || len([]byte(result.Slug)) > 128 {
		return HTMLMetadata{}, htmlError("html.metadata_invalid", "HTML scalar metadata exceeds its byte limit")
	}
	invalidLanguage := result.Language != "" && !sourceLanguagePattern.MatchString(result.Language)
	invalidSlug := result.Slug != "" && !sourceSlugPattern.MatchString(result.Slug)
	if invalidLanguage || invalidSlug {
		return HTMLMetadata{}, htmlError("html.metadata_invalid", "HTML language or slug is invalid")
	}
	var err error
	if result.Authors, err = normalizeHTMLList(metadata.Authors, "authors"); err != nil {
		return HTMLMetadata{}, err
	}
	if result.Tags, err = normalizeHTMLList(metadata.Tags, "tags"); err != nil {
		return HTMLMetadata{}, err
	}
	for _, date := range []string{result.PublishedAt, result.ModifiedAt} {
		if date == "" {
			continue
		}
		if _, err := time.Parse(time.RFC3339, date); err != nil {
			return HTMLMetadata{}, htmlError("html.metadata_invalid", "HTML date is not RFC 3339")
		}
	}
	return result, nil
}

func normalizeHTMLList(values []string, name string) ([]string, error) {
	if len(values) > 64 {
		return nil, htmlError("html.metadata_invalid", fmt.Sprintf("HTML %s exceed the item limit", name))
	}
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = normalizeHTMLText(value)
		if result[index] == "" || len([]byte(result[index])) > 128 {
			return nil, htmlError("html.metadata_invalid", fmt.Sprintf("HTML %s entry is invalid", name))
		}
	}
	return result, nil
}

func normalizeHTMLText(value string) string {
	return strings.Join(strings.FieldsFunc(value, unicode.IsSpace), " ")
}

func htmlError(code, message string) error {
	return newDiagnosticError(Diagnostic{Code: code, Severity: SeverityError, Message: message})
}
