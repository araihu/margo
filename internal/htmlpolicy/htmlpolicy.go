// Package htmlpolicy implements the closed margo-html-v1 fragment profile.
package htmlpolicy

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var allowedElements = map[string]struct{}{
	"a": {}, "abbr": {}, "b": {}, "blockquote": {}, "br": {}, "cite": {},
	"code": {}, "dd": {}, "del": {}, "details": {}, "dfn": {}, "dl": {},
	"dt": {}, "em": {}, "h1": {}, "h2": {}, "h3": {}, "h4": {}, "h5": {},
	"h6": {}, "hr": {}, "i": {}, "kbd": {}, "li": {}, "mark": {}, "ol": {},
	"p": {}, "pre": {}, "q": {}, "s": {}, "samp": {}, "small": {},
	"span": {}, "strong": {}, "sub": {}, "summary": {}, "sup": {},
	"table": {}, "tbody": {}, "td": {}, "tfoot": {}, "th": {}, "thead": {},
	"tr": {}, "u": {}, "ul": {}, "var": {},
}

var globalAttributes = map[string]struct{}{"title": {}, "lang": {}, "dir": {}}

var elementAttributes = map[string]map[string]struct{}{
	"a":       {"href": {}},
	"details": {"open": {}},
	"ol":      {"start": {}, "reversed": {}, "type": {}},
	"li":      {"value": {}},
	"td":      {"abbr": {}, "colspan": {}, "headers": {}, "rowspan": {}, "scope": {}},
	"th":      {"abbr": {}, "colspan": {}, "headers": {}, "rowspan": {}, "scope": {}},
}

// Validate parses and validates a fragment. It does not perform lossy
// rewriting: disallowed syntax is an error and must be rejected by the caller.
func Validate(fragment []byte) error {
	if len(fragment) == 0 {
		return nil
	}
	for _, r := range string(fragment) {
		if r == '\x00' || (r >= 0x01 && r <= 0x08) || (r >= 0x0b && r <= 0x0c) || (r >= 0x0e && r <= 0x1f) || r == '\x7f' {
			return fmt.Errorf("control character is not allowed")
		}
	}
	context := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := html.ParseFragment(strings.NewReader(string(fragment)), context)
	if err != nil {
		return fmt.Errorf("parse fragment: %w", err)
	}
	for _, node := range nodes {
		if err := validateNode(node); err != nil {
			return err
		}
	}
	return nil
}

func validateNode(node *html.Node) error {
	switch node.Type {
	case html.TextNode:
		return nil
	case html.ElementNode:
		name := strings.ToLower(node.Data)
		if node.Namespace != "" {
			return fmt.Errorf("namespace %q is not allowed", node.Namespace)
		}
		if _, ok := allowedElements[name]; !ok {
			return fmt.Errorf("element %q is not allowed", node.Data)
		}
		if err := validateAttributes(name, node.Attr); err != nil {
			return err
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if err := validateNode(child); err != nil {
				return err
			}
		}
		return nil
	case html.CommentNode:
		return fmt.Errorf("comments are not allowed")
	default:
		return fmt.Errorf("node type %d is not allowed", node.Type)
	}
}

func validateAttributes(element string, attributes []html.Attribute) error {
	seen := make(map[string]struct{}, len(attributes))
	for _, attribute := range attributes {
		if attribute.Namespace != "" {
			return fmt.Errorf("namespaced attribute %q is not allowed", attribute.Key)
		}
		name := strings.ToLower(attribute.Key)
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("duplicate attribute %q", name)
		}
		seen[name] = struct{}{}
		if _, ok := globalAttributes[name]; !ok {
			allowed := elementAttributes[element]
			if _, ok := allowed[name]; !ok {
				return fmt.Errorf("attribute %q is not allowed on <%s>", attribute.Key, element)
			}
		}
		if err := validateAttributeValue(element, name, attribute.Val); err != nil {
			return err
		}
	}
	return nil
}

func validateAttributeValue(element, name, value string) error {
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("attribute %q contains a control character", name)
		}
	}
	switch name {
	case "href":
		return validateURL(value)
	case "dir":
		if value != "ltr" && value != "rtl" && value != "auto" {
			return fmt.Errorf("invalid dir value")
		}
	case "open", "reversed":
		if value != "" && value != name {
			return fmt.Errorf("invalid boolean attribute %q", name)
		}
	case "start", "value", "colspan", "rowspan":
		number, err := strconv.Atoi(value)
		if err != nil || number < 1 {
			return fmt.Errorf("invalid positive integer attribute %q", name)
		}
	case "type":
		if element == "ol" {
			if _, ok := map[string]struct{}{"1": {}, "a": {}, "A": {}, "i": {}, "I": {}}[value]; !ok {
				return fmt.Errorf("invalid ordered-list type")
			}
		}
	case "scope":
		if value != "row" && value != "col" && value != "rowgroup" && value != "colgroup" {
			return fmt.Errorf("invalid table-cell scope")
		}
	case "headers":
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("headers cannot be empty")
		}
	}
	return nil
}

func validateURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme == "" {
		if strings.HasPrefix(value, "//") || parsed.Host != "" {
			return fmt.Errorf("network-path URLs are not allowed")
		}
		return nil
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "mailto", "tel":
		return nil
	default:
		return fmt.Errorf("URL scheme %q is not allowed", parsed.Scheme)
	}
}
