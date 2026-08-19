package deck

import (
	"strings"
	"testing"

	"github.com/araihu/margo"
)

func TestThemeCatalogFreezesTokensAndTypography(t *testing.T) {
	modern, err := ThemeCatalog(margo.ThemeModern, margo.ColorModeLight)
	if err != nil {
		t.Fatal(err)
	}
	if modern.Tokens.Surface != "#ffffff" || modern.Tokens.Ink != "#111827" || modern.Typography.H1Size != 64 {
		t.Fatalf("modern catalog = %#v", modern)
	}
	minimal, err := ThemeCatalog(margo.ThemeMinimal, margo.ColorModeDark)
	if err != nil {
		t.Fatal(err)
	}
	if minimal.Typography.HeadingFamily != "Margo Serif" || minimal.Tokens.Surface != "#000000" {
		t.Fatalf("minimal catalog = %#v", minimal)
	}
}

func TestThemeContrastMatrixRejectsUnlistedPairs(t *testing.T) {
	if err := ValidateThemeColorPair(margo.ThemeModern, margo.ColorModeLight, "ink", "surface"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateThemeColorPair(margo.ThemeModern, margo.ColorModeLight, "accent", "surface"); err == nil {
		t.Fatal("accent foreground unexpectedly accepted")
	}
	if err := ValidateThemeColorPair(margo.ThemeModern, margo.ColorModeLight, "ink", "negative"); err == nil {
		t.Fatal("unlisted foreground/background pair unexpectedly accepted")
	}
}

func TestThemeCSSKeepsNamedFacesAndInteractiveFallbacks(t *testing.T) {
	entry, err := ThemeCatalog(margo.ThemeModern, margo.ColorModeLight)
	if err != nil {
		t.Fatal(err)
	}
	css := themeCSS(entry)
	for _, fragment := range []string{
		"--margo-font-body:'Margo Sans'",
		"'SF Pro Text'",
		"ui-sans-serif",
		"--margo-font-code:'Margo Mono'",
		"ui-monospace",
	} {
		if !strings.Contains(css, fragment) {
			t.Fatalf("theme CSS missing %q: %s", fragment, css)
		}
	}
}
