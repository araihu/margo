package margo

import (
	"fmt"
	"regexp"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var (
	sourceLanguagePattern = regexp.MustCompile(`^[A-Za-z]{2,8}(?:-[A-Za-z0-9]{1,8})*$`)
	sourceSlugPattern     = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

type normalizedMarkdown struct {
	frontmatter frontmatterResult
	root        ast.Node
	headingIDs  []string
}

func init() { normalizeSource = normalizeMarkdownSource }

func normalizeMarkdownSource(source Source) (sourceNormalization, error) {
	frontmatter, err := parseFrontmatter(source)
	if err != nil {
		return sourceNormalization{}, err
	}
	root := newMarkdownParser().Parse(text.NewReader(frontmatter.body))
	headingIDs := collectHeadingIDs(root)
	metadata, err := normalizeSourceMetadata(source, frontmatter.values)
	if err != nil {
		return sourceNormalization{}, err
	}
	return sourceNormalization{
		metadata: metadata,
		parsed:   normalizedMarkdown{frontmatter: frontmatter, root: root, headingIDs: headingIDs},
	}, nil
}

func newMarkdownParser() parser.Parser {
	return goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.Footnote,
			extension.Linkify,
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	).Parser()
}

func normalizeSourceMetadata(source Source, values map[string]any) (Metadata, error) {
	metadata := Metadata{Name: source.Name, BaseURL: source.BaseURL}
	scalarFields := []struct {
		key         string
		destination *string
	}{
		{key: "title", destination: &metadata.Title},
		{key: "description", destination: &metadata.Description},
		{key: "language", destination: &metadata.Language},
		{key: "slug", destination: &metadata.Slug},
	}
	for _, field := range scalarFields {
		value, exists := values[field.key]
		if !exists {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return Metadata{}, invalidSourceMetadata(source.Name, "/"+field.key, fmt.Sprintf("%s must be a string", field.key))
		}
		*field.destination = text
	}
	if metadata.Language != "" && !sourceLanguagePattern.MatchString(metadata.Language) {
		return Metadata{}, invalidSourceMetadata(source.Name, "/language", "language must be a valid BCP 47-style tag")
	}
	if metadata.Slug != "" && !sourceSlugPattern.MatchString(metadata.Slug) {
		return Metadata{}, invalidSourceMetadata(source.Name, "/slug", "slug must contain lowercase letters, digits, and single hyphens")
	}

	var err error
	if metadata.Authors, err = sourceStringList(source.Name, values, "authors"); err != nil {
		return Metadata{}, err
	}
	if metadata.Tags, err = sourceStringList(source.Name, values, "tags"); err != nil {
		return Metadata{}, err
	}
	if metadata.PublishedAt, err = sourceDate(source.Name, values, "publishedAt"); err != nil {
		return Metadata{}, err
	}
	if metadata.ModifiedAt, err = sourceDate(source.Name, values, "modifiedAt"); err != nil {
		return Metadata{}, err
	}
	return metadata, nil
}

func sourceStringList(source string, values map[string]any, key string) ([]string, error) {
	value, exists := values[key]
	if !exists {
		return nil, nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil, invalidSourceMetadata(source, "/"+key, fmt.Sprintf("%s must be a list of strings", key))
	}
	result := make([]string, len(items))
	for index, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, invalidSourceMetadata(source, fmt.Sprintf("/%s/%d", key, index), fmt.Sprintf("%s entries must be strings", key))
		}
		result[index] = text
	}
	return result, nil
}

func sourceDate(source string, values map[string]any, key string) (string, error) {
	value, exists := values[key]
	if !exists {
		return "", nil
	}
	text, ok := value.(string)
	if !ok {
		return "", invalidSourceMetadata(source, "/"+key, fmt.Sprintf("%s must be an RFC 3339 string", key))
	}
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return "", invalidSourceMetadata(source, "/"+key, fmt.Sprintf("%s must be an RFC 3339 string", key))
	}
	return parsed.Format(time.RFC3339), nil
}

func invalidSourceMetadata(source, pointer, message string) error {
	return diagnosticAt("source.metadata_invalid", source, pointer, message, 1, 1)
}
