package site

import (
	"encoding/json"
	"net/url"
	"strings"

	margo "github.com/araihu/margo"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// structuredDataFor serializes consumer-owned JSON-LD without granting raw HTML
// capabilities. Vocabulary semantics and public identities belong to the host.
func structuredDataFor(page Page, metadata margo.Metadata) (json.RawMessage, error) {
	value, exists := metadata.Additional["structured_data"]
	if !exists {
		return nil, nil
	}
	invalid := func(message string) (json.RawMessage, error) {
		return nil, diagnostic("site.structured_data_invalid", message, "Use article options with type: Article or BlogPosting, or a JSON-LD object with an absolute HTTP(S) @context and @type or @graph.", page.Source)
	}
	if options, ok := value.(map[string]any); ok {
		if _, shorthand := options["type"]; shorthand {
			var err error
			value, err = articleStructuredData(page, options)
			if err != nil {
				return invalid(err.Error())
			}
		}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return invalid("structured_data must contain JSON-compatible values")
	}
	if len(data) > 64*1024 {
		return invalid("structured_data exceeds the 64 KiB limit")
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil || object == nil {
		return invalid("structured_data must be an object")
	}
	context, _ := object["@context"].(string)
	u, err := url.Parse(context)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return invalid("structured_data.@context must be an absolute HTTP(S) URL")
	}
	if graph, exists := object["@graph"]; exists {
		nodes, ok := graph.([]any)
		if !ok || len(nodes) == 0 {
			return invalid("structured_data.@graph must be a non-empty array")
		}
		for _, node := range nodes {
			object, ok := node.(map[string]any)
			if !ok || !structuredDataType(object["@type"]) {
				return invalid("each structured_data.@graph node must have a non-empty @type")
			}
		}
	} else if !structuredDataType(object["@type"]) {
		return invalid("structured_data must have a non-empty @type or @graph")
	}
	// Marshal keeps HTML escaping enabled, including script end tags and U+2028.
	return data, nil
}

func structuredDataType(value any) bool {
	switch value := value.(type) {
	case string:
		return strings.TrimSpace(value) != ""
	case []any:
		if len(value) == 0 {
			return false
		}
		for _, item := range value {
			text, ok := item.(string)
			if !ok || strings.TrimSpace(text) == "" {
				return false
			}
		}
		return true
	}
	return false
}

func projectStructuredData(root, head *html.Node, page Page) error {
	if len(page.StructuredData) == 0 {
		return nil
	}
	var duplicate func(*html.Node) bool
	duplicate = func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "script" && strings.EqualFold(strings.TrimSpace(attributeValue(node, "type")), "application/ld+json") {
			return true
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if duplicate(child) {
				return true
			}
		}
		return false
	}
	if duplicate(root) {
		return diagnostic("site.structured_data_duplicate", "structured_data conflicts with an existing JSON-LD script", "Consolidate JSON-LD in structured_data; use @graph for multiple entities.", page.Source)
	}
	script := &html.Node{Type: html.ElementNode, DataAtom: atom.Script, Data: "script", Attr: []html.Attribute{{Key: "type", Val: "application/ld+json"}}}
	script.AppendChild(&html.Node{Type: html.TextNode, Data: string(page.StructuredData)})
	head.AppendChild(script)
	return nil
}
