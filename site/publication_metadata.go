package site

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	margo "github.com/araihu/margo"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// publicationMetadata is the site-owned projection of editorial metadata.
// The generic HTML API remains consumer-agnostic; configured and directory
// sites use this small projection for route records and article markup.
type publicationMetadata struct {
	Authors     []string
	PublishedAt string
	ModifiedAt  string
	Tags        []string
}

func publicationMetadataFor(source string, document margo.Metadata, rendered margo.HTMLMetadata) (publicationMetadata, error) {
	metadata := publicationMetadata{
		Authors:     append([]string(nil), rendered.Authors...),
		PublishedAt: strings.TrimSpace(rendered.PublishedAt),
		ModifiedAt:  strings.TrimSpace(rendered.ModifiedAt),
		Tags:        append([]string(nil), rendered.Tags...),
	}

	// `author` and `date` were common names in existing blog sources. They are
	// consumer-owned root properties in the document schema, so accept them as
	// site aliases without changing the generic document contract. Canonical
	// fields win when both forms are present.
	if len(metadata.Authors) == 0 {
		if value, exists := document.Additional["author"]; exists {
			author, ok := value.(string)
			if !ok {
				return publicationMetadata{}, publicationDiagnostic(source, "author must be a string", "Use authors: [Name] or author: Name in site frontmatter.")
			}
			metadata.Authors = []string{normalizePublicationText(author)}
		}
	}
	if metadata.PublishedAt == "" {
		if value, exists := document.Additional["date"]; exists {
			var date string
			switch typed := value.(type) {
			case string:
				date = typed
			case time.Time:
				date = typed.Format("2006-01-02")
			default:
				return publicationMetadata{}, publicationDiagnostic(source, "date must be an RFC 3339 or YYYY-MM-DD string", "Use publishedAt: 2026-08-25T12:00:00Z or date: 2026-08-25 in site frontmatter.")
			}
			date, err := normalizePublicationDate(date)
			if err != nil {
				return publicationMetadata{}, publicationDiagnostic(source, "date must be an RFC 3339 or YYYY-MM-DD string", "Use publishedAt: 2026-08-25T12:00:00Z or date: 2026-08-25 in site frontmatter.")
			}
			metadata.PublishedAt = date
		}
	}
	if err := validatePublicationList(source, "authors", metadata.Authors); err != nil {
		return publicationMetadata{}, err
	}
	if err := validatePublicationList(source, "tags", metadata.Tags); err != nil {
		return publicationMetadata{}, err
	}
	return metadata, nil
}

func publicationDiagnostic(source, message, hint string) error {
	return diagnostic("site.metadata_invalid", message, hint, source)
}

func normalizePublicationDate(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.Format(time.RFC3339), nil
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed.Format("2006-01-02"), nil
	}
	return "", fmt.Errorf("invalid publication date")
}

func validatePublicationList(source, name string, values []string) error {
	if len(values) > 64 {
		return publicationDiagnostic(source, fmt.Sprintf("%s exceeds the 64-item site limit", name), "Reduce the publication metadata list to at most 64 entries.")
	}
	for _, value := range values {
		value = normalizePublicationText(value)
		if value == "" || len([]byte(value)) > 256 {
			return publicationDiagnostic(source, fmt.Sprintf("%s contains an empty or oversized entry", name), "Use non-empty publication metadata entries within the documented size limit.")
		}
	}
	return nil
}

func normalizePublicationText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func pagePublicationMetadata(metadata publicationMetadata) (authors []string, publishedAt, modifiedAt string, tags []string) {
	return append([]string(nil), metadata.Authors...), metadata.PublishedAt, metadata.ModifiedAt, append([]string(nil), metadata.Tags...)
}

func hasPublicationMetadata(page Page) bool {
	return len(page.Authors) > 0 || page.PublishedAt != "" || page.ModifiedAt != "" || len(page.Tags) > 0
}

// projectPublicationMetadata adds one deterministic semantic article header
// with labeled publication dates and machine-readable article metadata to a
// complete page. It is a no-op for pages without publication metadata,
// preserving existing bytes and contracts.
func projectPublicationMetadata(document []byte, page Page) ([]byte, error) {
	if !hasPublicationMetadata(page) {
		return document, nil
	}
	root, err := html.Parse(bytes.NewReader(document))
	if err != nil {
		return nil, diagnostic("site.html_invalid", err.Error(), "Report the generated publication metadata defect.", page.Source)
	}
	head := firstElement(root, "head")
	if head == nil {
		return nil, diagnostic("site.head_missing", "generated page has no head for publication metadata", "Render a complete HTML document.", page.Source)
	}
	article := findDocumentArticle(root)
	if article == nil {
		return nil, diagnostic("site.metadata_invalid", "generated page has no document article for publication metadata", "Keep one article.margo-document in the generated page.", page.Source)
	}
	if firstElementWithAttribute(article, "data-margo-publication-metadata") == nil {
		metadata := publicationHeader(page)
		if metadata != nil {
			insertPublicationHeader(article, metadata)
		}
	}
	projectPublicationHead(head, page)
	var output bytes.Buffer
	if err := html.Render(&output, root); err != nil {
		return nil, diagnostic("site.html_invalid", err.Error(), "Report the generated publication metadata defect.", page.Source)
	}
	return output.Bytes(), nil
}

func publicationHeader(page Page) *html.Node {
	if !hasPublicationMetadata(page) {
		return nil
	}
	header := &html.Node{Type: html.ElementNode, DataAtom: atom.Header, Data: "header", Attr: []html.Attribute{
		{Key: "class", Val: "margo-document__metadata"},
		{Key: "data-margo-publication-metadata", Val: "true"},
		{Key: "aria-label", Val: "Publication details"},
	}}
	if len(page.Authors) > 0 {
		address := &html.Node{Type: html.ElementNode, DataAtom: atom.Address, Data: "address", Attr: []html.Attribute{{Key: "aria-label", Val: "Authors"}}}
		for index, author := range page.Authors {
			if index > 0 {
				address.AppendChild(&html.Node{Type: html.TextNode, Data: ", "})
			}
			span := &html.Node{Type: html.ElementNode, DataAtom: atom.Span, Data: "span", Attr: []html.Attribute{{Key: "rel", Val: "author"}}}
			span.AppendChild(&html.Node{Type: html.TextNode, Data: normalizePublicationText(author)})
			address.AppendChild(span)
		}
		header.AppendChild(address)
	}
	if page.PublishedAt != "" || page.ModifiedAt != "" {
		dates := &html.Node{Type: html.ElementNode, DataAtom: atom.Span, Data: "span", Attr: []html.Attribute{
			{Key: "class", Val: "margo-document__publication-dates"},
			{Key: "role", Val: "group"},
			{Key: "aria-label", Val: localizedLabel(page.Locale, "publication_dates")},
		}}
		if page.PublishedAt != "" {
			dates.AppendChild(publicationDate("published", page.PublishedAt, localizedLabel(page.Locale, "published"), false))
		}
		if page.ModifiedAt != "" {
			dates.AppendChild(publicationDate("modified", page.ModifiedAt, localizedLabel(page.Locale, "updated"), page.PublishedAt != ""))
		}
		header.AppendChild(dates)
	}
	if len(page.Tags) > 0 {
		list := &html.Node{Type: html.ElementNode, DataAtom: atom.Ul, Data: "ul", Attr: []html.Attribute{{Key: "aria-label", Val: "Tags"}, {Key: "data-margo-publication-tags", Val: "true"}}}
		for _, tag := range page.Tags {
			item := &html.Node{Type: html.ElementNode, DataAtom: atom.Li, Data: "li", Attr: []html.Attribute{{Key: "data-margo-publication-tag", Val: normalizePublicationText(tag)}}}
			item.AppendChild(&html.Node{Type: html.TextNode, Data: normalizePublicationText(tag)})
			list.AppendChild(item)
		}
		header.AppendChild(list)
	}
	return header
}

func publicationDate(kind, value, label string, separator bool) *html.Node {
	date := &html.Node{Type: html.ElementNode, DataAtom: atom.Span, Data: "span", Attr: []html.Attribute{
		{Key: "class", Val: "margo-document__publication-date"},
		{Key: "data-margo-publication-kind", Val: kind},
	}}
	if separator {
		separatorNode := &html.Node{Type: html.ElementNode, DataAtom: atom.Span, Data: "span", Attr: []html.Attribute{
			{Key: "class", Val: "margo-document__publication-separator"},
			{Key: "aria-hidden", Val: "true"},
			{Key: "data-margo-publication-separator", Val: "true"},
		}}
		separatorNode.AppendChild(&html.Node{Type: html.TextNode, Data: " · "})
		date.AppendChild(separatorNode)
	}
	labelNode := &html.Node{Type: html.ElementNode, DataAtom: atom.Span, Data: "span", Attr: []html.Attribute{
		{Key: "class", Val: "margo-document__publication-label"},
		{Key: "data-margo-publication-label", Val: kind},
	}}
	labelNode.AppendChild(&html.Node{Type: html.TextNode, Data: label})
	date.AppendChild(labelNode)
	date.AppendChild(&html.Node{Type: html.TextNode, Data: " "})
	date.AppendChild(publicationTime(kind, value))
	return date
}

func publicationTime(kind, value string) *html.Node {
	timeNode := &html.Node{Type: html.ElementNode, DataAtom: atom.Time, Data: "time", Attr: []html.Attribute{
		{Key: "datetime", Val: value}, {Key: "data-margo-publication-date", Val: kind},
	}}
	timeNode.AppendChild(&html.Node{Type: html.TextNode, Data: value})
	return timeNode
}

func insertPublicationHeader(article, metadata *html.Node) {
	h1 := firstElement(article, "h1")
	if h1 == nil || h1.Parent == nil {
		article.InsertBefore(metadata, article.FirstChild)
		return
	}
	parent := h1.Parent
	next := h1.NextSibling
	for next != nil && next.Type == html.TextNode && strings.TrimSpace(next.Data) == "" {
		next = next.NextSibling
	}
	if next != nil && next.Type == html.ElementNode && hasClass(next, "margo-document__lead") {
		next = next.NextSibling
	}
	if next == nil {
		parent.AppendChild(metadata)
		return
	}
	parent.InsertBefore(metadata, next)
}

func projectPublicationHead(head *html.Node, page Page) {
	if typeNode := firstHeadMeta(head, "property", "og:type"); typeNode != nil {
		setHTMLAttribute(typeNode, "content", "article")
	}
	if page.PublishedAt != "" {
		appendHeadMeta(head, "property", "article:published_time", page.PublishedAt)
	}
	if page.ModifiedAt != "" {
		appendHeadMeta(head, "property", "article:modified_time", page.ModifiedAt)
	}
	for _, author := range page.Authors {
		appendHeadMeta(head, "property", "article:author", normalizePublicationText(author))
	}
	for _, tag := range page.Tags {
		appendHeadMeta(head, "property", "article:tag", normalizePublicationText(tag))
	}
}

func firstHeadMeta(head *html.Node, key, value string) *html.Node {
	for child := head.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode || child.Data != "meta" {
			continue
		}
		if attributeValue(child, key) == value {
			return child
		}
	}
	return nil
}

func appendHeadMeta(head *html.Node, key, name, value string) {
	for child := head.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode || child.Data != "meta" || attributeValue(child, key) != name {
			continue
		}
		if attributeValue(child, "content") == value {
			return
		}
	}
	meta := &html.Node{Type: html.ElementNode, DataAtom: atom.Meta, Data: "meta", Attr: []html.Attribute{{Key: key, Val: name}, {Key: "content", Val: value}}}
	head.AppendChild(meta)
}
