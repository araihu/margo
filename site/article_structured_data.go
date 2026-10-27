package site

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
)

// articleStructuredData uses the same normalized page metadata as the visible
// article. Images and author profiles must be explicitly supplied by the host.
func articleStructuredData(page Page, options map[string]any) (map[string]any, error) {
	for key := range options {
		if key != "type" && key != "image" && key != "author_urls" {
			return nil, fmt.Errorf("unknown article structured_data property %q", key)
		}
	}
	kind, ok := options["type"].(string)
	if !ok || (kind != "Article" && kind != "BlogPosting") {
		return nil, fmt.Errorf("structured_data.type must be Article or BlogPosting")
	}
	if !structuredDataURL(page.Canonical) {
		return nil, fmt.Errorf("article structured_data requires a configured canonical HTTP(S) URL")
	}
	data := map[string]any{"@context": "https://schema.org", "@type": kind, "@id": page.Canonical + "#" + strings.ToLower(kind), "url": page.Canonical, "mainEntityOfPage": page.Canonical, "headline": page.Title}
	if page.Description != "" {
		data["description"] = page.Description
	}
	if page.Locale != "" {
		data["inLanguage"] = page.Locale
	}
	var published, modified time.Time
	for _, date := range []struct {
		key, value string
		parsed     *time.Time
	}{{"datePublished", page.PublishedAt, &published}, {"dateModified", page.ModifiedAt, &modified}} {
		if date.value == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, date.value)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", date.value)
		}
		if err != nil {
			return nil, fmt.Errorf("invalid article %s", date.key)
		}
		*date.parsed = parsed
		data[date.key] = date.value
	}
	if !published.IsZero() && !modified.IsZero() && modified.Before(published) {
		return nil, fmt.Errorf("article modifiedAt precedes publishedAt")
	}
	profiles := map[string]any{}
	if value, exists := options["author_urls"]; exists {
		var ok bool
		profiles, ok = value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("structured_data.author_urls must map author names to absolute HTTP(S) profile URLs")
		}
		for name, value := range profiles {
			profile, ok := value.(string)
			if !ok || !structuredDataURL(profile) || !slices.Contains(page.Authors, name) {
				return nil, fmt.Errorf("structured_data.author_urls contains an invalid URL or an unknown author")
			}
		}
	}
	if len(page.Authors) > 0 {
		authors := make([]map[string]string, 0, len(page.Authors))
		for _, name := range page.Authors {
			author := map[string]string{"@type": "Person", "name": name}
			if profile, ok := profiles[name].(string); ok {
				author["url"] = profile
			}
			authors = append(authors, author)
		}
		data["author"] = authors
	}
	if value, exists := options["image"]; exists {
		image, ok := value.(string)
		if !ok || !structuredDataURL(image) {
			return nil, fmt.Errorf("structured_data.image must be an absolute HTTP(S) URL")
		}
		data["image"] = image
	}
	return data, nil
}

func structuredDataURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Host != "" && parsed.User == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && !strings.ContainsAny(value, "\r\n\t ")
}
