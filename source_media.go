package margo

import (
	"bytes"
	"net/url"
	"strings"

	goldast "github.com/yuin/goldmark/ast"
)

// rejectRemoteImages keeps the core artifact offline-capable. Ordinary links
// remain links; image destinations that would trigger a remote request fail.
func rejectRemoteImages(normalized sourceNormalization) error {
	parsed, ok := normalized.parsed.(normalizedMarkdown)
	if !ok || parsed.root == nil {
		return nil
	}
	var failure error
	_ = goldast.Walk(parsed.root, func(node goldast.Node, entering bool) (goldast.WalkStatus, error) {
		if !entering || failure != nil {
			return goldast.WalkContinue, nil
		}
		image, ok := node.(*goldast.Image)
		if !ok {
			return goldast.WalkContinue, nil
		}
		destination := string(image.Destination)
		parsedURL, err := url.Parse(destination)
		if err != nil || (strings.ToLower(parsedURL.Scheme) != "http" && strings.ToLower(parsedURL.Scheme) != "https") {
			return goldast.WalkContinue, nil
		}
		offset := bytes.Index(parsed.frontmatter.body, image.Destination)
		if offset < 0 {
			offset = 0
		}
		failure = diagnosticAt(
			"policy.remote_image.denied", normalized.metadata.Name, "/image/destination",
			"remote Markdown images are denied; download the image and use a local relative asset",
			parsed.frontmatter.bodyLines+lineAtOffset(parsed.frontmatter.body, offset), 1,
		)
		return goldast.WalkStop, nil
	})
	return failure
}
