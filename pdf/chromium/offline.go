package chromium

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/araihu/margo/pdf"
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

func rewriteDocumentLinks(document []byte, policy pdf.RelativeLinkPolicy, baseURL string) ([]byte, error) {
	if policy == "" {
		policy = pdf.RelativeLinksStrip
	}
	switch policy {
	case pdf.RelativeLinksStrip, pdf.RelativeLinksError, pdf.RelativeLinksKeep, pdf.RelativeLinksResolve:
	default:
		return nil, chromiumError("pdf.relative_link_policy_invalid", "relative link policy must be strip, error, keep, or resolve")
	}

	var base *url.URL
	if policy == pdf.RelativeLinksResolve {
		parsed, err := publicDocumentBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		base = parsed
	}

	root, err := html.Parse(bytes.NewReader(document))
	if err != nil {
		return nil, chromiumError("pdf.request_invalid", "HTML cannot be parsed")
	}
	var walk func(*html.Node) error
	walk = func(node *html.Node) error {
		if node.Type == html.ElementNode && strings.EqualFold(node.Data, "a") {
			for index := range node.Attr {
				attribute := &node.Attr[index]
				if !strings.EqualFold(attribute.Key, "href") {
					continue
				}
				href := strings.TrimSpace(attribute.Val)
				if strings.Contains(href, `\`) {
					return chromiumError("pdf.relative_link_invalid", "anchor href must not contain backslashes")
				}
				parsed, parseErr := url.Parse(href)
				if parseErr != nil {
					return chromiumError("pdf.relative_link_invalid", "anchor href cannot be parsed")
				}
				if href == "" || strings.HasPrefix(href, "#") {
					break
				}
				if parsed.Scheme != "" {
					switch strings.ToLower(parsed.Scheme) {
					case "http", "https":
						if parsed.Host == "" || parsed.Hostname() == "" {
							return chromiumError("pdf.link_absolute_invalid", "http(s) anchor must include an absolute host")
						}
					case "mailto", "tel":
						break
					default:
						return chromiumError("pdf.link_scheme_forbidden", "anchor scheme "+parsed.Scheme+" is not allowed")
					}
					break
				}
				if parsed.Host != "" || strings.HasPrefix(href, "//") {
					return chromiumError("pdf.relative_link_invalid", "network-path anchor is not allowed")
				}
				switch policy {
				case pdf.RelativeLinksKeep:
				case pdf.RelativeLinksError:
					return chromiumError("pdf.relative_link_forbidden", "relative anchor "+href+" requires an explicit PDF link policy")
				case pdf.RelativeLinksResolve:
					attribute.Val = base.ResolveReference(parsed).String()
				case pdf.RelativeLinksStrip:
					node.Attr = append(node.Attr[:index], node.Attr[index+1:]...)
				}
				break
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := html.Render(&output, root); err != nil {
		return nil, chromiumError("pdf.request_invalid", "HTML cannot be serialized")
	}
	return output.Bytes(), nil
}

func publicDocumentBaseURL(value string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil || parsed.Host == "" || parsed.Hostname() == "" || (strings.ToLower(parsed.Scheme) != "http" && strings.ToLower(parsed.Scheme) != "https") {
		return nil, chromiumError("pdf.relative_link_base_invalid", "base URL must be an absolute http or https URL")
	}
	hostname := strings.TrimRight(strings.ToLower(parsed.Hostname()), ".")
	address := net.ParseIP(hostname)
	if address == nil {
		address, _ = browserIPv4Address(hostname)
	}
	if hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") || (address != nil && address.IsLoopback()) {
		return nil, chromiumError("pdf.relative_link_base_invalid", "base URL must not use a loopback host")
	}
	return parsed, nil
}

// browserIPv4Address recognizes the legacy numeric forms that Chromium's URL
// parser canonicalizes as IPv4, including 127.1 and a single 32-bit integer.
func browserIPv4Address(hostname string) (net.IP, bool) {
	parts := strings.Split(hostname, ".")
	if len(parts) == 0 || len(parts) > 4 {
		return nil, false
	}
	numbers := make([]uint64, len(parts))
	for index, part := range parts {
		if part == "" {
			return nil, false
		}
		base := 10
		digits := part
		lower := strings.ToLower(part)
		if strings.HasPrefix(lower, "0x") {
			base, digits = 16, part[2:]
		} else if len(part) > 1 && part[0] == '0' {
			base = 8
		}
		if digits == "" {
			return nil, false
		}
		value, err := strconv.ParseUint(digits, base, 32)
		if err != nil {
			return nil, false
		}
		numbers[index] = value
	}
	for _, value := range numbers[:len(numbers)-1] {
		if value > 255 {
			return nil, false
		}
	}
	remainingBytes := 5 - len(numbers)
	lastLimit := (uint64(1) << (8 * remainingBytes)) - 1
	if numbers[len(numbers)-1] > lastLimit {
		return nil, false
	}
	var value uint64
	for _, part := range numbers[:len(numbers)-1] {
		value = value*256 + part
	}
	value = value*(uint64(1)<<(8*remainingBytes)) + numbers[len(numbers)-1]
	return net.IPv4(byte(value>>24), byte(value>>16), byte(value>>8), byte(value)), true
}
