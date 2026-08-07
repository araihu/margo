package margo

import "testing"

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
