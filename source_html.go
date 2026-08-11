package margo

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	goldast "github.com/yuin/goldmark/ast"
	"golang.org/x/net/html"
)

// stripHTMLComments tokenizes one Goldmark raw-HTML span, removes only
// well-formed comments, and leaves adjacent real HTML subject to host policy.
func stripHTMLComments(fragment []byte) ([]byte, error) {
	tokenizer := html.NewTokenizer(bytes.NewReader(fragment))
	var output bytes.Buffer
	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			if tokenizer.Err() == io.EOF {
				return output.Bytes(), nil
			}
			return nil, fmt.Errorf("malformed HTML comment: %w", tokenizer.Err())
		case html.CommentToken:
			raw := tokenizer.Raw()
			if !bytes.HasPrefix(raw, []byte("<!--")) || !bytes.HasSuffix(raw, []byte("-->")) {
				return nil, fmt.Errorf("HTML comment must end with -->")
			}
		default:
			_, _ = output.Write(tokenizer.Raw())
		}
	}
}

func inspectSourceHTML(normalized sourceNormalization, iframePolicy *IframePolicy) (bool, error) {
	parsed, ok := normalized.parsed.(normalizedMarkdown)
	if !ok || parsed.root == nil {
		return false, nil
	}
	found := false
	var failure error
	_ = goldast.Walk(parsed.root, func(node goldast.Node, entering bool) (goldast.WalkStatus, error) {
		if !entering {
			return goldast.WalkContinue, nil
		}
		if textNode, ok := node.(*goldast.Text); ok {
			value := textNode.Value(parsed.frontmatter.body)
			if index := bytes.Index(value, []byte("<!--")); index >= 0 && !insideCodeSpan(textNode) {
				offset := parsed.frontmatter.bodyOffset + textNode.Segment.Start + index
				failure = htmlCommentDiagnostic(normalized.metadata.Name, parsed.frontmatter.body, parsed.frontmatter.bodyOffset, parsed.frontmatter.bodyLines, offset, "HTML comment must end with -->")
				return goldast.WalkStop, nil
			}
			return goldast.WalkContinue, nil
		}
		fragment, offset, raw := rawHTMLSource(node, parsed.frontmatter.body)
		if !raw {
			return goldast.WalkContinue, nil
		}
		remaining, err := stripHTMLComments(fragment)
		if err != nil {
			failure = htmlCommentDiagnostic(normalized.metadata.Name, parsed.frontmatter.body, parsed.frontmatter.bodyOffset, parsed.frontmatter.bodyLines, parsed.frontmatter.bodyOffset+offset, err.Error())
			return goldast.WalkStop, nil
		}
		if strings.TrimSpace(string(remaining)) == "" {
			return goldast.WalkContinue, nil
		}
		embed, recognized, embedErr := parseIframeFragment(remaining)
		if recognized {
			if embedErr == nil {
				embedErr = authorizeIframe(iframePolicy, embed)
			}
			if embedErr != nil {
				failure = diagnosticAt("policy.iframe_denied", normalized.metadata.Name, "/policy/iframe", embedErr.Error(), parsed.frontmatter.bodyLines+lineAtOffset(parsed.frontmatter.body, offset), 1)
				return goldast.WalkStop, nil
			}
			return goldast.WalkContinue, nil
		}
		found = true
		return goldast.WalkContinue, nil
	})
	return found, failure
}

func rawHTMLSource(node goldast.Node, source []byte) ([]byte, int, bool) {
	switch value := node.(type) {
	case *goldast.HTMLBlock:
		offset := segmentAtStart(value.Lines())
		return value.Text(source), offset, true
	case *goldast.RawHTML:
		offset := segmentAtStart(value.Segments)
		return value.Segments.Value(source), offset, true
	default:
		return nil, 0, false
	}
}

func insideCodeSpan(node goldast.Node) bool {
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		if _, ok := parent.(*goldast.CodeSpan); ok {
			return true
		}
	}
	return false
}

func htmlCommentDiagnostic(source string, body []byte, bodyOffset, bodyLines, absoluteOffset int, message string) error {
	relative := absoluteOffset - bodyOffset
	if relative < 0 {
		relative = 0
	}
	return diagnosticAt("source.html_comment_malformed", source, "/rawHTML", message, bodyLines+lineAtOffset(body, relative), 1)
}
