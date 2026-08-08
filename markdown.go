package margo

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
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
	root := goldmark.New(
		goldmark.WithExtensions(
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.Footnote,
			extension.Linkify,
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	).Parser().Parse(text.NewReader(frontmatter.body))
	headingIDs := collectHeadingIDs(root)
	metadata := Metadata{Name: source.Name, BaseURL: source.BaseURL}
	if title, ok := frontmatter.values["title"].(string); ok {
		metadata.Title = title
	}
	if description, ok := frontmatter.values["description"].(string); ok {
		metadata.Description = description
	}
	return sourceNormalization{
		metadata: metadata,
		parsed:   normalizedMarkdown{frontmatter: frontmatter, root: root, headingIDs: headingIDs},
	}, nil
}
