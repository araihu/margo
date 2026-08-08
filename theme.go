package margo

import (
	"fmt"
	"regexp"
	"strings"
)

// ThemeName is the closed set of themes available to the standalone shell.
type ThemeName string

const (
	ThemeModern   = "modern"
	ThemeGoshtoso = "goshtoso"
	ThemeMinimal  = "minimal"
)

// ColorMode selects the light or dark Goshtoso token family independently
// from the document theme.
type ColorMode string

const (
	ColorModeLight ColorMode = "light"
	ColorModeDark  ColorMode = "dark"
)

func validateColorMode(mode ColorMode) error {
	if mode != ColorModeLight && mode != ColorModeDark {
		return fmt.Errorf("margo: unknown color mode %q", mode)
	}
	return nil
}

// DocumentToken is the versioned, bounded CSS custom-property surface.
type DocumentToken string

const (
	TokenFontBody       DocumentToken = "--document-font-body"
	TokenFontHeading    DocumentToken = "--document-font-heading"
	TokenContentWidth   DocumentToken = "--document-content-width"
	TokenLineHeight     DocumentToken = "--document-line-height"
	TokenCodeTheme      DocumentToken = "--document-code-theme"
	TokenPageBackground DocumentToken = "--document-page-background"
)

var tokenValuePattern = regexp.MustCompile(`^[A-Za-z0-9 ._#(),%+\-/'":]+$`)

var supportedDocumentTokens = map[DocumentToken]struct{}{
	TokenFontBody: {}, TokenFontHeading: {}, TokenContentWidth: {},
	TokenLineHeight: {}, TokenCodeTheme: {}, TokenPageBackground: {},
}

// ValidateDocumentToken validates both the versioned key and a conservative
// value grammar. CSS declarations, URLs, braces, and control characters are
// never accepted through this API.
func ValidateDocumentToken(token DocumentToken, value string) error {
	if _, ok := supportedDocumentTokens[token]; !ok {
		return fmt.Errorf("margo: unsupported document token %q", token)
	}
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 160 || !tokenValuePattern.MatchString(value) || strings.ContainsAny(value, "{};<>\\\n\r\t") {
		return fmt.Errorf("margo: invalid value for document token %q", token)
	}
	return nil
}

func defaultThemeTokens(theme ThemeName) (map[DocumentToken]string, error) {
	switch theme {
	case "", ThemeModern, ThemeGoshtoso:
		return map[DocumentToken]string{
			TokenFontBody:       "var(--font-body)",
			TokenFontHeading:    "var(--font-title)",
			TokenContentWidth:   "72rem",
			TokenLineHeight:     "1.6",
			TokenCodeTheme:      "chroma",
			TokenPageBackground: "var(--color-surface)",
		}, nil
	case ThemeMinimal:
		return map[DocumentToken]string{
			TokenFontBody:       "var(--font-body)",
			TokenFontHeading:    "var(--font-title)",
			TokenContentWidth:   "64rem",
			TokenLineHeight:     "1.55",
			TokenCodeTheme:      "chroma",
			TokenPageBackground: "var(--color-surface)",
		}, nil
	default:
		return nil, fmt.Errorf("margo: unknown theme %q", theme)
	}
}

func validateThemeTokens(tokens map[DocumentToken]string) error {
	for token, value := range tokens {
		if err := ValidateDocumentToken(token, value); err != nil {
			return err
		}
	}
	return nil
}
