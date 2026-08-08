package margo

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	colorTokenPattern  = regexp.MustCompile(`^(#[0-9a-fA-F]{3,8}|[a-zA-Z]+|rgba?\([^()\r\n]+\))$`)
	lengthTokenPattern = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:px|rem|em|ex|ch|vw|vh|vmin|vmax|%)$`)
)

// ValidateToken accepts only bounded, value-only theme tokens. In particular,
// it never accepts CSS functions that can resolve host state or external data.
func ValidateToken(name, value string) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(value) == "" {
		return fmt.Errorf("policy.token.invalid: token name and value are required")
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("policy.token.invalid: control character")
		}
	}
	lower := strings.ToLower(value)
	for _, forbidden := range []string{"var(", "url(", "@import", "@supports", "@media", ";", "{", "}"} {
		if strings.Contains(lower, forbidden) {
			return fmt.Errorf("policy.token.invalid: forbidden CSS channel %q", forbidden)
		}
	}
	switch strings.ToLower(name) {
	case "color", "background", "foreground", "border-color":
		if !colorTokenPattern.MatchString(value) {
			return fmt.Errorf("policy.token.invalid: bounded color grammar required")
		}
	case "length", "font-size", "radius", "spacing":
		if !lengthTokenPattern.MatchString(value) && value != "0" {
			return fmt.Errorf("policy.token.invalid: bounded length grammar required")
		}
	default:
		if strings.ContainsAny(value, "()[]<>\"'") {
			return fmt.Errorf("policy.token.invalid: unsupported token grammar")
		}
	}
	return nil
}
