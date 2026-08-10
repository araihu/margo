package chromium

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var cssURLPattern = regexp.MustCompile(`(?i)url\(\s*['\"]?([^'\"\)]+)`)

var subresourceAttributes = map[string]map[string]struct{}{
	"audio":  {"src": {}},
	"embed":  {"src": {}},
	"iframe": {"src": {}},
	"img":    {"src": {}, "srcset": {}},
	"input":  {"src": {}},
	"link":   {"href": {}},
	"object": {"data": {}},
	"script": {"src": {}},
	"source": {"src": {}, "srcset": {}},
	"track":  {"src": {}},
	"video":  {"poster": {}, "src": {}},
}

// validateOfflineHTML rejects every render-time asset that is not already
// embedded in the document. Ordinary anchor links are content, not requests,
// and remain allowed.
func validateOfflineHTML(document []byte) error {
	root, err := html.Parse(bytes.NewReader(document))
	if err != nil {
		return chromiumError("pdf.request_invalid", "HTML cannot be parsed")
	}
	var walk func(*html.Node) error
	walk = func(node *html.Node) error {
		if node.Type == html.ElementNode {
			attributes := subresourceAttributes[strings.ToLower(node.Data)]
			for _, attribute := range node.Attr {
				name := strings.ToLower(attribute.Key)
				if _, subresource := attributes[name]; subresource && !embeddedURL(attribute.Val) {
					return chromiumError("pdf.network_forbidden", fmt.Sprintf("%s[%s] must be materialized as an embedded asset", node.Data, name))
				}
				if name == "style" {
					if err := validateOfflineCSS(attribute.Val); err != nil {
						return err
					}
				}
			}
		}
		if node.Type == html.TextNode && node.Parent != nil && strings.EqualFold(node.Parent.Data, "style") {
			if err := validateOfflineCSS(node.Data); err != nil {
				return err
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root)
}

func validateOfflineCSS(value string) error {
	for _, match := range cssURLPattern.FindAllStringSubmatch(value, -1) {
		if len(match) == 2 && !embeddedURL(match[1]) {
			return chromiumError("pdf.network_forbidden", "CSS url() assets must be materialized in the document")
		}
	}
	return nil
}

func embeddedURL(value string) bool {
	value = strings.TrimSpace(value)
	return value == "" || strings.HasPrefix(value, "#") || strings.HasPrefix(strings.ToLower(value), "data:")
}
