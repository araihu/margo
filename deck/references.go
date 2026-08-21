package deck

import (
	"regexp"
	"strings"
)

var (
	definitionPattern         = regexp.MustCompile(`^\s{0,3}\[([^\]^]+)\]:`)
	footnoteDefinitionPattern = regexp.MustCompile(`^\s{0,3}\[\^([^\]^]+)\]:`)
	referencePattern          = regexp.MustCompile(`\[[^\]\n]+\]\[([^\]\n]*)\]`)
	footnoteReferencePattern  = regexp.MustCompile(`\[\^([^\]^\n]+)\]`)
)

type referenceLocation struct {
	slide int
	slot  string
}

func validateDeckReferences(name string, slides []Slide) error {
	definitions := make(map[string]referenceLocation)
	type scope struct {
		location referenceLocation
		markdown []byte
	}
	var scopes []scope
	for slideIndex, slide := range slides {
		layout := slide.Layout()
		if layout == nil {
			scopes = append(scopes, scope{location: referenceLocation{slide: slideIndex}, markdown: slide.Markdown()})
			continue
		}
		for _, slot := range layout.Slots {
			scopes = append(scopes, scope{location: referenceLocation{slide: slideIndex, slot: slot.Name}, markdown: slot.Markdown})
		}
	}
	for _, current := range scopes {
		for _, line := range strings.Split(string(current.markdown), "\n") {
			if match := footnoteDefinitionPattern.FindStringSubmatch(line); len(match) == 2 {
				if err := registerReferenceDefinition(name, definitions, normalizeReferenceLabel("^"+match[1]), current.location); err != nil {
					return err
				}
				continue
			}
			if match := definitionPattern.FindStringSubmatch(line); len(match) == 2 {
				if err := registerReferenceDefinition(name, definitions, normalizeReferenceLabel(match[1]), current.location); err != nil {
					return err
				}
			}
		}
	}
	for _, current := range scopes {
		text := string(current.markdown)
		for _, match := range referencePattern.FindAllStringSubmatch(text, -1) {
			label := strings.TrimSpace(match[1])
			if label == "" {
				continue
			}
			if err := checkReferenceScope(name, definitions, normalizeReferenceLabel(label), current.location); err != nil {
				return err
			}
		}
		for _, match := range footnoteReferencePattern.FindAllStringSubmatch(text, -1) {
			if err := checkReferenceScope(name, definitions, normalizeReferenceLabel("^"+match[1]), current.location); err != nil {
				return err
			}
		}
	}
	return nil
}

func registerReferenceDefinition(name string, definitions map[string]referenceLocation, key string, location referenceLocation) error {
	if previous, exists := definitions[key]; exists && previous != location {
		code := "deck.cross_slide_reference"
		if previous.slide == location.slide {
			code = "deck.cross_slot_reference"
		}
		return deckError(code, name, 1, "reference definition crosses a deck boundary")
	}
	definitions[key] = location
	return nil
}

func checkReferenceScope(name string, definitions map[string]referenceLocation, key string, location referenceLocation) error {
	definition, exists := definitions[key]
	if !exists || definition == location {
		return nil
	}
	code := "deck.cross_slide_reference"
	if definition.slide == location.slide {
		code = "deck.cross_slot_reference"
	}
	return deckError(code, name, 1, "reference crosses a deck boundary")
}

func normalizeReferenceLabel(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}
