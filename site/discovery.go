package site

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

const sitemapNamespace = "http://www.sitemaps.org/schemas/sitemap/0.9"

type sitemapDocument struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Location string `xml:"loc"`
}

func (b *builder) addPublicationDiscoveryArtifacts() error {
	if b.config == nil {
		return nil
	}

	sitemap, err := renderSitemap(b.configPages)
	if err != nil {
		return err
	}
	if err := b.addArtifact(SitemapPath, sitemap); err != nil {
		return err
	}

	llms, err := renderLLMSText(b.config.Site, b.configPages)
	if err != nil {
		return err
	}
	if err := b.addArtifact(LLMSPath, llms); err != nil {
		return err
	}
	return nil
}

func renderSitemap(pages []Page) ([]byte, error) {
	routes, err := publicationRoutes(pages)
	if err != nil {
		return nil, err
	}
	document := sitemapDocument{XMLNS: sitemapNamespace, URLs: make([]sitemapURL, 0, len(routes))}
	for _, page := range routes {
		document.URLs = append(document.URLs, sitemapURL{Location: page.Canonical})
	}
	data, err := xml.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("site.sitemap_invalid: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), append(data, '\n')...), nil
}

func renderLLMSText(siteConfig SiteConfig, pages []Page) ([]byte, error) {
	routes, err := publicationRoutes(pages)
	if err != nil {
		return nil, err
	}
	name := llmsInline(siteConfig.Name)
	if name == "" {
		name = "Documentation"
	}
	summary := normalizeLLMSLine(siteConfig.Description)
	if summary == "" {
		summary = name + " documentation."
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "# %s\n\n> %s\n\n## Documentation\n\n", name, summary)
	for _, page := range routes {
		title := llmsInline(page.Title)
		if title == "" {
			title = "Untitled page"
		}
		description := normalizeLLMSLine(page.Description)
		fmt.Fprintf(&builder, "- [%s](<%s>)", title, page.Canonical)
		if description != "" {
			fmt.Fprintf(&builder, ": %s", description)
		}
		builder.WriteByte('\n')
	}
	if repository := strings.TrimSpace(siteConfig.RepositoryURL); repository != "" {
		fmt.Fprintf(&builder, "\n## Project\n\n- [Source repository](<%s>)\n", repository)
	}
	return []byte(builder.String()), nil
}

func publicationRoutes(pages []Page) ([]Page, error) {
	routes := append([]Page(nil), pages...)
	sort.SliceStable(routes, func(left, right int) bool {
		return pageRouteLess(routes[left], routes[right])
	})
	seen := make(map[string]struct{}, len(routes))
	for _, page := range routes {
		canonical := strings.TrimSpace(page.Canonical)
		parsed, err := url.Parse(canonical)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, diagnostic("site.public_url_invalid", fmt.Sprintf("route %q has invalid canonical URL %q", page.Output, page.Canonical), "Use an absolute HTTPS base_url and generated route paths.", page.Source)
		}
		if _, exists := seen[canonical]; exists {
			return nil, diagnostic("site.route_duplicate", fmt.Sprintf("route %q has duplicate canonical URL %q", page.Output, canonical), "Give each public page a unique route.", page.Source)
		}
		seen[canonical] = struct{}{}
	}
	return routes, nil
}

func normalizeLLMSLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func llmsInline(value string) string {
	return strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]").Replace(normalizeLLMSLine(value))
}
