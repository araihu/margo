package ssg

import (
	"fmt"
	"html"
	"net/url"
	"sort"
	"strings"
	"unicode"

	"github.com/araihu/margo/internal/canonicaljson"
)

// ResourceMarkup is the deterministic markup emitted for frame resources.
// Head and BodyEnd contain tags only; neither value includes an HTML wrapper.
type ResourceMarkup struct {
	Head    string
	BodyEnd string
}

// ValidateResources verifies the resource portion of a frame or shell
// contract. It does not resolve URLs: the site builder owns offline closure,
// vendoring, and remote-resource policy.
func ValidateResources(resources []ResourceRequirement) error {
	seen := make(map[string]struct{}, len(resources))
	for index, resource := range resources {
		if err := validateResource(index, resource); err != nil {
			return err
		}
		identity, err := canonicaljson.Marshal(resource)
		if err != nil {
			return fmt.Errorf("ssg.resource_identity: resource %d: %w", index, err)
		}
		key := string(identity)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("ssg.resource_duplicate: resource %d repeats an earlier resource", index)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// RenderResources turns validated requirements into stable head/body-end
// tags. Ordering is declaration order within each placement.
func RenderResources(resources []ResourceRequirement) (ResourceMarkup, error) {
	if err := ValidateResources(resources); err != nil {
		return ResourceMarkup{}, err
	}
	var head, bodyEnd []string
	for _, resource := range resources {
		markup, err := renderResource(resource)
		if err != nil {
			return ResourceMarkup{}, err
		}
		if resource.Placement == "head" {
			head = append(head, markup)
		} else {
			bodyEnd = append(bodyEnd, markup)
		}
	}
	return ResourceMarkup{
		Head:    strings.Join(head, "\n"),
		BodyEnd: strings.Join(bodyEnd, "\n"),
	}, nil
}

func validateResource(index int, resource ResourceRequirement) error {
	if resource.Placement != "head" && resource.Placement != "body-end" {
		return fmt.Errorf("ssg.resource_placement: resource %d has invalid placement %q", index, resource.Placement)
	}
	switch resource.Kind {
	case "stylesheet", "script", "module", "preload":
	default:
		return fmt.Errorf("ssg.resource_kind: resource %d has invalid kind %q", index, resource.Kind)
	}
	hasURL := resource.URL != ""
	hasInline := resource.Inline != ""
	if hasURL == hasInline {
		return fmt.Errorf("ssg.resource_source: resource %d must declare exactly one URL or Inline source", index)
	}
	if hasURL {
		if err := validateResourceURL(resource.URL); err != nil {
			return fmt.Errorf("ssg.resource_url: resource %d: %w", index, err)
		}
	}
	if hasInline {
		if resource.Kind != "script" && resource.Kind != "module" {
			return fmt.Errorf("ssg.resource_inline: resource %d kind %q cannot use Inline source", index, resource.Kind)
		}
		if resource.Integrity != "" {
			return fmt.Errorf("ssg.resource_integrity: resource %d cannot attach Integrity to Inline source", index)
		}
		if strings.Contains(strings.ToLower(resource.Inline), "</script") {
			return fmt.Errorf("ssg.resource_inline: resource %d contains a closing script tag", index)
		}
	}
	if resource.Placement == "body-end" && resource.Kind != "script" && resource.Kind != "module" {
		return fmt.Errorf("ssg.resource_placement: resource %d kind %q is only valid in head", index, resource.Kind)
	}
	if resource.Kind == "stylesheet" || resource.Kind == "preload" {
		if resource.Placement != "head" {
			return fmt.Errorf("ssg.resource_placement: resource %d kind %q is only valid in head", index, resource.Kind)
		}
	}
	if resource.Integrity != "" {
		if containsControlOrSpace(resource.Integrity) {
			return fmt.Errorf("ssg.resource_integrity: resource %d contains whitespace or control characters", index)
		}
	}
	if err := validateResourceAttributes(index, resource.Attributes); err != nil {
		return err
	}
	return nil
}

func validateResourceURL(value string) error {
	if containsControlOrSpace(value) || strings.HasPrefix(value, "//") {
		return fmt.Errorf("URL must be a relative path or an absolute HTTP(S) URL")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.IsAbs() {
		if parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("URL scheme must be http or https")
		}
		if parsed.User != nil {
			return fmt.Errorf("URL user info is not allowed")
		}
	}
	if parsed.Host != "" && parsed.Scheme == "" {
		return fmt.Errorf("network-path URLs are not allowed")
	}
	if parsed.Fragment != "" {
		return fmt.Errorf("URL fragments are not allowed")
	}
	return nil
}

func validateResourceAttributes(index int, attributes map[string]string) error {
	seen := make(map[string]string, len(attributes))
	for name, value := range attributes {
		if !validHTMLAttributeName(name) {
			return fmt.Errorf("ssg.resource_attribute: resource %d has invalid attribute name %q", index, name)
		}
		lower := strings.ToLower(name)
		if previous, exists := seen[lower]; exists {
			return fmt.Errorf("ssg.resource_attribute: resource %d repeats attributes %q and %q", index, previous, name)
		}
		seen[lower] = name
		switch lower {
		case "href", "src", "rel", "integrity", "type":
			return fmt.Errorf("ssg.resource_attribute: resource %d cannot override controlled attribute %q", index, name)
		}
		if strings.HasPrefix(lower, "on") {
			return fmt.Errorf("ssg.resource_attribute: resource %d cannot declare event handler %q", index, name)
		}
		if containsControl(value) {
			return fmt.Errorf("ssg.resource_attribute: resource %d attribute %q contains control characters", index, name)
		}
	}
	return nil
}

func validHTMLAttributeName(value string) bool {
	if value == "" {
		return false
	}
	for index, r := range value {
		if index == 0 {
			if !(r == ':' || r == '_' || unicode.IsLetter(r)) {
				return false
			}
			continue
		}
		if !(r == ':' || r == '_' || r == '-' || r == '.' || unicode.IsLetter(r) || unicode.IsDigit(r)) {
			return false
		}
	}
	return true
}

func renderResource(resource ResourceRequirement) (string, error) {
	var builder strings.Builder
	writeAttribute := func(name, value string) {
		builder.WriteByte(' ')
		builder.WriteString(name)
		builder.WriteString(`="`)
		builder.WriteString(html.EscapeString(value))
		builder.WriteByte('"')
	}
	attributes := sortedResourceAttributes(resource.Attributes)
	switch resource.Kind {
	case "stylesheet", "preload":
		builder.WriteString("<link")
		writeAttribute("rel", resource.Kind)
		writeAttribute("href", resource.URL)
		if resource.Integrity != "" {
			writeAttribute("integrity", resource.Integrity)
		}
		for _, attribute := range attributes {
			writeAttribute(attribute.name, attribute.value)
		}
		builder.WriteString(">")
	case "script", "module":
		builder.WriteString("<script")
		if resource.Kind == "module" {
			writeAttribute("type", "module")
		}
		if resource.URL != "" {
			writeAttribute("src", resource.URL)
		}
		if resource.Integrity != "" {
			writeAttribute("integrity", resource.Integrity)
		}
		for _, attribute := range attributes {
			writeAttribute(attribute.name, attribute.value)
		}
		builder.WriteString(">")
		if resource.Inline != "" {
			builder.WriteString(resource.Inline)
		}
		builder.WriteString("</script>")
	default:
		return "", fmt.Errorf("ssg.resource_kind: cannot render resource kind %q", resource.Kind)
	}
	return builder.String(), nil
}

type resourceAttribute struct {
	name  string
	value string
}

func sortedResourceAttributes(attributes map[string]string) []resourceAttribute {
	output := make([]resourceAttribute, 0, len(attributes))
	for name, value := range attributes {
		output = append(output, resourceAttribute{name: name, value: value})
	}
	sort.Slice(output, func(i, j int) bool {
		left, right := strings.ToLower(output[i].name), strings.ToLower(output[j].name)
		if left != right {
			return left < right
		}
		return output[i].name < output[j].name
	})
	return output
}

func containsControlOrSpace(value string) bool {
	for _, r := range value {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func containsControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
