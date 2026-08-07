package margo

import (
	"strings"
	"testing"
)

func TestThemeTokens(t *testing.T) {
	for _, token := range []DocumentToken{
		TokenFontBody,
		TokenFontHeading,
		TokenContentWidth,
		TokenLineHeight,
		TokenCodeTheme,
		TokenPageBackground,
	} {
		if err := ValidateDocumentToken(token, "var(--safe-token)"); err != nil {
			t.Fatalf("ValidateDocumentToken(%q) error = %v", token, err)
		}
	}
	if err := ValidateDocumentToken(DocumentToken("--not-supported"), "red"); err == nil {
		t.Fatal("unknown document token unexpectedly accepted")
	}
	if err := ValidateDocumentToken(TokenPageBackground, "red; background: url(https://evil.example)"); err == nil {
		t.Fatal("unsafe token value unexpectedly accepted")
	}
}

func TestModernIsDefaultStandaloneTheme(t *testing.T) {
	tokens, err := defaultThemeTokens(ThemeModern)
	if err != nil {
		t.Fatal(err)
	}
	if got := tokens[TokenFontBody]; got != "var(--font-body)" {
		t.Fatalf("body font = %q", got)
	}
	if got := tokens[TokenPageBackground]; got != "var(--color-surface)" {
		t.Fatalf("page background = %q", got)
	}
}

func TestApplyThemeTokensTerminatesLastDeclaration(t *testing.T) {
	tokens, err := defaultThemeTokens(ThemeModern)
	if err != nil {
		t.Fatal(err)
	}
	css := applyThemeTokens(`.document{/* MARGO_THEME_TOKENS */color:red}`, tokens)
	want := string(TokenPageBackground) + ":var(--color-surface);color:red"
	if !strings.Contains(css, want) {
		t.Fatalf("last theme declaration is not terminated: %s", css)
	}
}
