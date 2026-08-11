package margo

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type iframeEmbed struct {
	URL    string
	Origin string
	Title  string
	Width  int
	Height int
}

func parseIframeFragment(fragment []byte) (iframeEmbed, bool, error) {
	contextNode := &xhtml.Node{Type: xhtml.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := xhtml.ParseFragment(bytes.NewReader(fragment), contextNode)
	if err != nil {
		return iframeEmbed{}, false, err
	}
	meaningful := make([]*xhtml.Node, 0, len(nodes))
	for _, node := range nodes {
		if node.Type == xhtml.TextNode && strings.TrimSpace(node.Data) == "" {
			continue
		}
		meaningful = append(meaningful, node)
	}
	if len(meaningful) != 1 || meaningful[0].Type != xhtml.ElementNode || strings.ToLower(meaningful[0].Data) != "iframe" {
		if containsIframe(nodes) {
			return iframeEmbed{}, true, fmt.Errorf("iframe must be the only top-level element in its HTML fragment")
		}
		return iframeEmbed{}, false, nil
	}
	node := meaningful[0]
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != xhtml.TextNode || strings.TrimSpace(child.Data) != "" {
			return iframeEmbed{}, true, fmt.Errorf("iframe content must be empty")
		}
	}
	attributes := make(map[string]string, len(node.Attr))
	for _, attribute := range node.Attr {
		name := strings.ToLower(attribute.Key)
		if attribute.Namespace != "" {
			return iframeEmbed{}, true, fmt.Errorf("namespaced iframe attribute %q is not supported", attribute.Key)
		}
		if _, duplicate := attributes[name]; duplicate {
			return iframeEmbed{}, true, fmt.Errorf("duplicate iframe attribute %q", name)
		}
		switch name {
		case "src", "title", "width", "height":
			attributes[name] = attribute.Val
		default:
			return iframeEmbed{}, true, fmt.Errorf("unsupported iframe attribute %q; documents may use only src, title, width, and height", name)
		}
	}
	normalizedURL, origin, err := canonicalIframeURL(attributes["src"])
	if err != nil {
		return iframeEmbed{}, true, err
	}
	title := strings.TrimSpace(attributes["title"])
	if len([]byte(title)) > 256 {
		return iframeEmbed{}, true, fmt.Errorf("iframe title must be at most 256 UTF-8 bytes")
	}
	width, err := iframeDimension(attributes["width"], 640, "width")
	if err != nil {
		return iframeEmbed{}, true, err
	}
	height, err := iframeDimension(attributes["height"], 360, "height")
	if err != nil {
		return iframeEmbed{}, true, err
	}
	return iframeEmbed{URL: normalizedURL, Origin: origin, Title: title, Width: width, Height: height}, true, nil
}

func containsIframe(nodes []*xhtml.Node) bool {
	for _, node := range nodes {
		if node.Type == xhtml.ElementNode && strings.EqualFold(node.Data, "iframe") {
			return true
		}
		children := make([]*xhtml.Node, 0)
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			children = append(children, child)
		}
		if containsIframe(children) {
			return true
		}
	}
	return false
}

func iframeDimension(value string, fallback int, name string) (int, error) {
	if value == "" {
		return fallback, nil
	}
	number, err := strconv.Atoi(value)
	if err != nil || number < 1 || number > 4096 {
		return 0, fmt.Errorf("iframe %s must be an integer between 1 and 4096", name)
	}
	return number, nil
}

func canonicalIframeURL(value string) (string, string, error) {
	if len([]byte(value)) == 0 || len([]byte(value)) > 4096 || strings.Contains(value, `\`) {
		return "", "", fmt.Errorf("iframe src must contain 1 to 4096 UTF-8 bytes without backslashes")
	}
	parsed, err := url.Parse(value)
	if err != nil || strings.ToLower(parsed.Scheme) != "https" || parsed.Host == "" || parsed.User != nil {
		return "", "", fmt.Errorf("iframe src must be absolute HTTPS without credentials")
	}
	hostname, err := canonicalPolicyHost(parsed.Hostname())
	if err != nil {
		return "", "", fmt.Errorf("iframe src has invalid hostname: %w", err)
	}
	port := parsed.Port()
	if port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return "", "", fmt.Errorf("iframe src has invalid port")
		}
		if number == 443 {
			port = ""
		}
	}
	host := hostname
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	parsed.Scheme = "https"
	parsed.Host = host
	origin := "https://" + host
	return parsed.String(), origin, nil
}

func authorizeIframe(policy *IframePolicy, embed iframeEmbed) error {
	if policy == nil {
		return fmt.Errorf("iframe embeds are denied by the host policy")
	}
	index := sort.SearchStrings(policy.AllowedOrigins, embed.Origin)
	if index >= len(policy.AllowedOrigins) || policy.AllowedOrigins[index] != embed.Origin {
		return fmt.Errorf("iframe origin %q is not allowed by the host policy", embed.Origin)
	}
	return nil
}

func iframeProjection(policy *IframePolicy, target RenderTarget) Projection {
	if policy == nil {
		return ProjectionDeny
	}
	switch target {
	case TargetHTML:
		return policy.Projections.HTML
	case TargetSite:
		return policy.Projections.Site
	case TargetPDF:
		return policy.Projections.PDF
	case TargetDeck:
		return policy.Projections.Deck
	default:
		return ProjectionDeny
	}
}

func renderIframe(output io.Writer, embed iframeEmbed, policy *IframePolicy, target RenderTarget) error {
	if err := authorizeIframe(policy, embed); err != nil {
		return fmt.Errorf("policy.iframe_denied: %w", err)
	}
	projection := iframeProjection(policy, target)
	switch projection {
	case ProjectionStaticLink:
		label := embed.Title
		if label == "" {
			label = embed.URL
		}
		_, err := fmt.Fprintf(output, `<figure class="margo-embed margo-embed--static"><a class="margo-embed__link" href="%s" rel="noreferrer" referrerpolicy="%s">%s</a></figure>`, html.EscapeString(embed.URL), html.EscapeString(string(policy.ReferrerPolicy)), html.EscapeString(label))
		return err
	case ProjectionInteractive:
		sandbox := make([]string, len(policy.Sandbox))
		for index, token := range policy.Sandbox {
			sandbox[index] = string(token)
		}
		_, err := fmt.Fprintf(output, `<figure class="margo-embed margo-embed--iframe"><iframe class="margo-embed__frame" src="%s"`, html.EscapeString(embed.URL))
		if err != nil {
			return err
		}
		if embed.Title != "" {
			if _, err := fmt.Fprintf(output, ` title="%s"`, html.EscapeString(embed.Title)); err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(output, ` width="%d" height="%d" sandbox="%s" referrerpolicy="%s" allow="" loading="lazy"></iframe></figure>`, embed.Width, embed.Height, html.EscapeString(strings.Join(sandbox, " ")), html.EscapeString(string(policy.ReferrerPolicy)))
		return err
	default:
		return fmt.Errorf("policy.iframe_denied: iframe projection is deny for target %s", target)
	}
}
