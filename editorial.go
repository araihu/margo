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

type EditorialMetadata struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Language    string   `json:"language"`
	Slug        string   `json:"slug"`
	Authors     []string `json:"authors,omitempty"`
	PublishedAt string   `json:"publishedAt,omitempty"`
	ModifiedAt  string   `json:"modifiedAt,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

func (m EditorialMetadata) clone() EditorialMetadata {
	m.Authors = append([]string(nil), m.Authors...)
	m.Tags = append([]string(nil), m.Tags...)
	return m
}

type EditorialResult struct {
	fragmentBytes []byte
	plainText     string
	metadata      EditorialMetadata
	requirements  HTMLRequirements
	diagnostics   []Diagnostic
	fingerprint   EditorialFingerprint
}

type editorialConfig struct {
	header bool
}

type EditorialOption func(*editorialConfig) error

func WithEditorialHeader() EditorialOption {
	return func(config *editorialConfig) error {
		config.header = true
		return nil
	}
}

func RenderEditorial(result *RenderResult, options ...EditorialOption) (*EditorialResult, error) {
	if result == nil || result.Content() == nil {
		return nil, editorialError("editorial.result_required", "render result and content are required")
	}
	config := editorialConfig{}
	for index, option := range options {
		if option == nil {
			return nil, editorialError("editorial.metadata_invalid", fmt.Sprintf("nil editorial option at index %d", index))
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}

	metadata, err := normalizeEditorialResultMetadata(result.Metadata())
	if err != nil {
		return nil, err
	}
	fragment, err := renderEditorialComponentBytes(result.Content())
	if err != nil {
		return nil, fmt.Errorf("editorial.fragment_render: %w", err)
	}
	inspection, err := inspectEditorialFragment(fragment)
	if err != nil {
		return nil, err
	}
	if metadata.Title == "" {
		metadata.Title = inspection.firstH1
	}
	diagnostics := result.Diagnostics()
	if metadata.Title != "" && inspection.firstH1 != "" && metadata.Title != inspection.firstH1 {
		diagnostics = append(diagnostics, Diagnostic{
			Code: "editorial.title_conflict", Severity: SeverityInfo,
			Message: fmt.Sprintf("metadata title %q differs from body heading %q", metadata.Title, inspection.firstH1),
		})
	}
	if config.header && inspection.firstH1 == "" && metadata.Title != "" {
		fragment, err = insertEditorialHeader(fragment, metadata.Title)
		if err != nil {
			return nil, err
		}
		inspection, err = inspectEditorialFragment(fragment)
		if err != nil {
			return nil, err
		}
	}
	requirements := result.editorialHTMLRequirements()
	fingerprint, err := editorialFingerprint(fragment, metadata, requirements, config)
	if err != nil {
		return nil, err
	}
	return &EditorialResult{
		fragmentBytes: append([]byte(nil), fragment...),
		plainText:     inspection.plainText,
		metadata:      metadata.clone(),
		requirements:  HTMLRequirements{requirements: requirements.List()},
		diagnostics:   cloneDiagnostics(diagnostics),
		fingerprint:   fingerprint,
	}, nil
}

func (r *EditorialResult) Fragment() templ.Component {
	if r == nil {
		return nil
	}
	data := append([]byte(nil), r.fragmentBytes...)
	return templ.ComponentFunc(func(_ context.Context, out io.Writer) error {
		_, err := out.Write(data)
		return err
	})
}

func (r *EditorialResult) PlainText() string {
	if r == nil {
		return ""
	}
	return r.plainText
}

func (r *EditorialResult) Metadata() EditorialMetadata {
	if r == nil {
		return EditorialMetadata{}
	}
	return r.metadata.clone()
}

func (r *EditorialResult) Requirements() HTMLRequirements {
	if r == nil {
		return HTMLRequirements{}
	}
	return HTMLRequirements{requirements: r.requirements.List()}
}

func (r *EditorialResult) Diagnostics() []Diagnostic {
	if r == nil {
		return nil
	}
	return cloneDiagnostics(r.diagnostics)
}

func (r *EditorialResult) Fingerprint() EditorialFingerprint {
	if r == nil {
		return EditorialFingerprint{}
	}
	return r.fingerprint
}

func renderEditorialComponentBytes(component templ.Component) ([]byte, error) {
	var buffer bytes.Buffer
	if err := component.Render(context.Background(), &buffer); err != nil {
		return nil, err
	}
	return append([]byte(nil), buffer.Bytes()...), nil
}

type editorialFragmentInspection struct {
	firstH1   string
	plainText string
}

func inspectEditorialFragment(fragment []byte) (editorialFragmentInspection, error) {
	contextNode := &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := xhtml.ParseFragment(bytes.NewReader(fragment), contextNode)
	if err != nil {
		return editorialFragmentInspection{}, editorialError("editorial.metadata_invalid", fmt.Sprintf("invalid editorial fragment: %v", err))
	}
	var article *xhtml.Node
	for _, node := range nodes {
		if node.Type == xhtml.TextNode && strings.TrimSpace(node.Data) == "" {
			continue
		}
		if node.Type != xhtml.ElementNode || node.Data != "article" || article != nil {
			return editorialFragmentInspection{}, editorialError("editorial.metadata_invalid", "editorial fragment must contain exactly one top-level article")
		}
		article = node
	}
	if article == nil || !htmlClassContains(article, "margo-document") {
		return editorialFragmentInspection{}, editorialError("editorial.metadata_invalid", "editorial fragment must contain one margo-document article")
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
				return editorialError("editorial.metadata_invalid", fmt.Sprintf("editorial fragment contains forbidden <%s>", node.Data))
			case "style":
				if !htmlAttributeEquals(node, "data-margo-extension-style", "charts") {
					return editorialError("editorial.metadata_invalid", "editorial fragment contains an unowned <style>")
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
			semanticText = isEditorialTextElement(node.Data)
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
		return editorialFragmentInspection{}, err
	}
	if articleCount != 1 {
		return editorialFragmentInspection{}, editorialError("editorial.metadata_invalid", "editorial fragment must contain exactly one article")
	}
	return editorialFragmentInspection{firstH1: firstH1, plainText: normalizeEditorialText(text.String())}, nil
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
	return normalizeEditorialText(text.String())
}

func isEditorialTextElement(tag string) bool {
	switch tag {
	case "address", "caption", "code", "dd", "dt", "figcaption", "h1", "h2", "h3", "h4", "h5", "h6", "li", "p", "pre", "td", "th":
		return true
	default:
		return false
	}
}

func insertEditorialHeader(fragment []byte, title string) ([]byte, error) {
	articleStart := bytes.Index(fragment, []byte("<article"))
	if articleStart < 0 {
		return nil, editorialError("editorial.metadata_invalid", "editorial article start is missing")
	}
	openingEnd := bytes.IndexByte(fragment[articleStart:], '>')
	if openingEnd < 0 {
		return nil, editorialError("editorial.metadata_invalid", "editorial article opening tag is malformed")
	}
	openingEnd += articleStart + 1
	header := []byte(`<header class="margo-document__header"><h1>` + stdhtml.EscapeString(title) + `</h1></header>`)
	result := make([]byte, 0, len(fragment)+len(header))
	result = append(result, fragment[:openingEnd]...)
	result = append(result, header...)
	result = append(result, fragment[openingEnd:]...)
	return result, nil
}

func normalizeEditorialResultMetadata(metadata Metadata) (EditorialMetadata, error) {
	result := EditorialMetadata{
		Title: normalizeEditorialText(metadata.Title), Description: normalizeEditorialText(metadata.Description),
		Language: strings.TrimSpace(metadata.Language), Slug: strings.TrimSpace(metadata.Slug),
		PublishedAt: strings.TrimSpace(metadata.PublishedAt), ModifiedAt: strings.TrimSpace(metadata.ModifiedAt),
	}
	if len([]byte(result.Title)) > 256 || len([]byte(result.Description)) > 512 || len([]byte(result.Language)) > 64 || len([]byte(result.Slug)) > 128 {
		return EditorialMetadata{}, editorialError("editorial.metadata_invalid", "editorial scalar metadata exceeds its byte limit")
	}
	invalidLanguage := result.Language != "" && !editorialLanguagePattern.MatchString(result.Language)
	invalidSlug := result.Slug != "" && !editorialSlugPattern.MatchString(result.Slug)
	if invalidLanguage || invalidSlug {
		return EditorialMetadata{}, editorialError("editorial.metadata_invalid", "editorial language or slug is invalid")
	}
	var err error
	if result.Authors, err = normalizeEditorialList(metadata.Authors, "authors"); err != nil {
		return EditorialMetadata{}, err
	}
	if result.Tags, err = normalizeEditorialList(metadata.Tags, "tags"); err != nil {
		return EditorialMetadata{}, err
	}
	for _, date := range []string{result.PublishedAt, result.ModifiedAt} {
		if date == "" {
			continue
		}
		if _, err := time.Parse(time.RFC3339, date); err != nil {
			return EditorialMetadata{}, editorialError("editorial.metadata_invalid", "editorial date is not RFC 3339")
		}
	}
	return result, nil
}

func normalizeEditorialList(values []string, name string) ([]string, error) {
	if len(values) > 64 {
		return nil, editorialError("editorial.metadata_invalid", fmt.Sprintf("editorial %s exceed the item limit", name))
	}
	result := make([]string, len(values))
	for index, value := range values {
		result[index] = normalizeEditorialText(value)
		if result[index] == "" || len([]byte(result[index])) > 128 {
			return nil, editorialError("editorial.metadata_invalid", fmt.Sprintf("editorial %s entry is invalid", name))
		}
	}
	return result, nil
}

func normalizeEditorialText(value string) string {
	return strings.Join(strings.FieldsFunc(value, unicode.IsSpace), " ")
}

func editorialError(code, message string) error {
	return newDiagnosticError(Diagnostic{Code: code, Severity: SeverityError, Message: message})
}
