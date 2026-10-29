package site

import (
	"fmt"
	"slices"

	margo "github.com/araihu/margo"
)

// applyPublicationLinks preserves host-owned identities and archive routes in
// the native publication header. It never derives routes from display names.
func applyPublicationLinks(page *Page, metadata margo.Metadata) error {
	raw, exists := metadata.Additional["publication_links"]
	if !exists {
		return nil
	}
	invalid := func(message string) error {
		return publicationDiagnostic(page.Source, message, "Map declared author and tag names to absolute HTTP(S) URLs in publication_links.authors and publication_links.tags.")
	}
	groups, ok := raw.(map[string]any)
	if !ok {
		return invalid("publication_links must be an object")
	}
	for name, raw := range groups {
		var declared []string
		links := map[string]string{}
		switch name {
		case "authors":
			declared = page.Authors
		case "tags":
			declared = page.Tags
		default:
			return invalid(fmt.Sprintf("unknown publication_links group %q", name))
		}
		mapping, ok := raw.(map[string]any)
		if !ok || len(mapping) > 64 {
			return invalid("publication link groups must contain at most 64 named URLs")
		}
		for label, raw := range mapping {
			destination, ok := raw.(string)
			if !ok || !structuredDataURL(destination) || !slices.Contains(declared, label) {
				return invalid(fmt.Sprintf("invalid URL or undeclared %s name %q", name, label))
			}
			links[label] = destination
		}
		if name == "authors" {
			page.authorLinks = links
		} else {
			page.tagLinks = links
		}
	}
	return nil
}
