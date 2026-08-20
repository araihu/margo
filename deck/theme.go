package deck

import (
	"fmt"
	"math"
	"strings"

	"github.com/araihu/margo"
)

// ThemeTokens is the finite semantic color surface exposed to deck CSS.
type ThemeTokens struct {
	Surface      string
	SurfaceAlt   string
	Ink          string
	InkMuted     string
	Accent       string
	AccentStrong string
	Positive     string
	Warning      string
	Negative     string
	Info         string
}

type ThemeTypography struct {
	BodyFamily    string
	HeadingFamily string
	CodeFamily    string
	BodySize      float64
	BodyLine      float64
	H1Size        float64
	H1Line        float64
	H2Size        float64
	H2Line        float64
	H3Size        float64
	H3Line        float64
}

// ThemeCatalogEntry freezes one theme/mode's visual contract.
type ThemeCatalogEntry struct {
	Theme       margo.ThemeName
	ColorMode   margo.ColorMode
	Tokens      ThemeTokens
	Typography  ThemeTypography
	PaddingX    float64
	PaddingY    float64
	HeaderZone  float64
	FooterZone  float64
	ContentGap  float64
	CardRadius  float64
	BorderWidth float64
	SlotPadding float64
	CardPadding float64
}

var themeColorRows = map[margo.ThemeName]map[margo.ColorMode]ThemeTokens{
	margo.ThemeModern: {
		margo.ColorModeLight: {Surface: "#ffffff", SurfaceAlt: "#f3f4f6", Ink: "#111827", InkMuted: "#4b5563", Accent: "#0f766e", AccentStrong: "#115e59", Positive: "#166534", Warning: "#92400e", Negative: "#b91c1c", Info: "#1d4ed8"},
		margo.ColorModeDark:  {Surface: "#111827", SurfaceAlt: "#1f2937", Ink: "#f9fafb", InkMuted: "#d1d5db", Accent: "#5eead4", AccentStrong: "#99f6e4", Positive: "#86efac", Warning: "#fde68a", Negative: "#fca5a5", Info: "#93c5fd"},
	},
	margo.ThemeGoshtoso: {
		margo.ColorModeLight: {Surface: "#ffffff", SurfaceAlt: "#f1f5f9", Ink: "#0f172a", InkMuted: "#475569", Accent: "#0e7490", AccentStrong: "#155e75", Positive: "#166534", Warning: "#92400e", Negative: "#b91c1c", Info: "#1d4ed8"},
		margo.ColorModeDark:  {Surface: "#0f172a", SurfaceAlt: "#1e293b", Ink: "#f8fafc", InkMuted: "#cbd5e1", Accent: "#67e8f9", AccentStrong: "#22d3ee", Positive: "#86efac", Warning: "#fde68a", Negative: "#fca5a5", Info: "#93c5fd"},
	},
	margo.ThemeMinimal: {
		margo.ColorModeLight: {Surface: "#ffffff", SurfaceAlt: "#fafafa", Ink: "#000000", InkMuted: "#404040", Accent: "#1f2937", AccentStrong: "#111827", Positive: "#14532d", Warning: "#854d0e", Negative: "#991b1b", Info: "#1e3a8a"},
		margo.ColorModeDark:  {Surface: "#000000", SurfaceAlt: "#171717", Ink: "#fafafa", InkMuted: "#d4d4d4", Accent: "#a3a3a3", AccentStrong: "#e5e5e5", Positive: "#86efac", Warning: "#fde68a", Negative: "#fca5a5", Info: "#93c5fd"},
	},
}

var themeTypography = map[margo.ThemeName]ThemeTypography{
	margo.ThemeModern:   {BodyFamily: "Margo Sans", HeadingFamily: "Margo Sans", CodeFamily: "Margo Mono", BodySize: 24, BodyLine: 1.35, H1Size: 64, H1Line: 1.05, H2Size: 40, H2Line: 1.10, H3Size: 28, H3Line: 1.20},
	margo.ThemeGoshtoso: {BodyFamily: "Margo Sans", HeadingFamily: "Margo Sans", CodeFamily: "Margo Mono", BodySize: 22, BodyLine: 1.40, H1Size: 60, H1Line: 1.05, H2Size: 38, H2Line: 1.12, H3Size: 26, H3Line: 1.22},
	margo.ThemeMinimal:  {BodyFamily: "Margo Sans", HeadingFamily: "Margo Serif", CodeFamily: "Margo Mono", BodySize: 22, BodyLine: 1.40, H1Size: 58, H1Line: 1.05, H2Size: 36, H2Line: 1.15, H3Size: 26, H3Line: 1.25},
}

// ThemeCatalog returns a defensive copy of the selected frozen row.
func ThemeCatalog(theme margo.ThemeName, mode margo.ColorMode) (ThemeCatalogEntry, error) {
	if theme == "" {
		theme = margo.ThemeModern
	}
	if mode == "" {
		mode = margo.ColorModeLight
	}
	rows, ok := themeColorRows[theme]
	if !ok {
		return ThemeCatalogEntry{}, deckError("deck.theme_invalid", "", 1, "unsupported deck theme")
	}
	tokens, ok := rows[mode]
	if !ok {
		return ThemeCatalogEntry{}, deckError("deck.color_mode_invalid", "", 1, "unsupported deck color mode")
	}
	typography := themeTypography[theme]
	entry := ThemeCatalogEntry{
		Theme: theme, ColorMode: mode, Tokens: tokens, Typography: typography,
		PaddingX: 64, PaddingY: 56, HeaderZone: 32, FooterZone: 24, ContentGap: 16,
		CardRadius: 20, BorderWidth: 1, SlotPadding: 24, CardPadding: 24,
	}
	if theme == margo.ThemeGoshtoso {
		entry.PaddingX, entry.PaddingY, entry.CardRadius = 56, 48, 12
	} else if theme == margo.ThemeMinimal {
		entry.PaddingX, entry.PaddingY, entry.HeaderZone, entry.FooterZone = 72, 64, 28, 20
		entry.CardRadius, entry.SlotPadding, entry.CardPadding = 0, 16, 16
	}
	return entry, nil
}

func (tokens ThemeTokens) Value(name string) (string, bool) {
	switch name {
	case "surface":
		return tokens.Surface, true
	case "surface-alt":
		return tokens.SurfaceAlt, true
	case "ink":
		return tokens.Ink, true
	case "ink-muted":
		return tokens.InkMuted, true
	case "accent":
		return tokens.Accent, true
	case "accent-strong":
		return tokens.AccentStrong, true
	case "positive":
		return tokens.Positive, true
	case "warning":
		return tokens.Warning, true
	case "negative":
		return tokens.Negative, true
	case "info":
		return tokens.Info, true
	default:
		return "", false
	}
}

func ValidateThemeColorPair(theme margo.ThemeName, mode margo.ColorMode, foreground, background string) error {
	entry, err := ThemeCatalog(theme, mode)
	if err != nil {
		return err
	}
	if foreground == "transparent" || background == "transparent" {
		return deckError("deck.contrast_invalid", "", 1, "transparent cannot be selected as an explicit foreground/background")
	}
	_, fgKnown := entry.Tokens.Value(foreground)
	_, bgKnown := entry.Tokens.Value(background)
	if !fgKnown || !bgKnown {
		return deckError("deck.contrast_invalid", "", 1, "unknown theme color token")
	}
	accentStatus := map[string]struct{}{"ink": {}, "ink-muted": {}, "accent-strong": {}, "positive": {}, "warning": {}, "negative": {}, "info": {}}
	_, fgAccent := accentStatus[foreground]
	_, bgAccent := accentStatus[background]
	allowed := (fgAccent && (background == "surface" || background == "surface-alt")) || (bgAccent && (foreground == "surface" || foreground == "surface-alt")) || (background == "accent" && (foreground == "surface" || foreground == "surface-alt"))
	if !allowed {
		return deckError("deck.contrast_invalid", "", 1, "foreground/background token pair is not in the frozen matrix")
	}
	fg, _ := entry.Tokens.Value(foreground)
	bg, _ := entry.Tokens.Value(background)
	if contrastRatio(fg, bg) < 4.5 {
		return deckError("deck.contrast_invalid", "", 1, fmt.Sprintf("contrast ratio %.3f is below WCAG AA", contrastRatio(fg, bg)))
	}
	return nil
}

func contrastRatio(foreground, background string) float64 {
	fg := hexLuminance(foreground)
	bg := hexLuminance(background)
	if fg < bg {
		fg, bg = bg, fg
	}
	return (fg + 0.05) / (bg + 0.05)
}

func hexLuminance(value string) float64 {
	if len(value) != 7 || value[0] != '#' {
		return 0
	}
	channel := func(offset int) float64 {
		parsed := 0
		for _, digit := range value[offset : offset+2] {
			parsed *= 16
			switch {
			case digit >= '0' && digit <= '9':
				parsed += int(digit - '0')
			case digit >= 'a' && digit <= 'f':
				parsed += int(digit-'a') + 10
			case digit >= 'A' && digit <= 'F':
				parsed += int(digit-'A') + 10
			}
		}
		linear := float64(parsed) / 255
		if linear <= 0.03928 {
			return linear / 12.92
		}
		return math.Pow((linear+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(1) + 0.7152*channel(3) + 0.0722*channel(5)
}

func themeCSS(entry ThemeCatalogEntry) string {
	return strings.Join([]string{
		fmt.Sprintf("--margo-surface:%s", entry.Tokens.Surface),
		fmt.Sprintf("--margo-surface-alt:%s", entry.Tokens.SurfaceAlt),
		fmt.Sprintf("--margo-ink:%s", entry.Tokens.Ink),
		fmt.Sprintf("--margo-ink-muted:%s", entry.Tokens.InkMuted),
		fmt.Sprintf("--margo-accent:%s", entry.Tokens.Accent),
		fmt.Sprintf("--margo-accent-strong:%s", entry.Tokens.AccentStrong),
		fmt.Sprintf("--margo-positive:%s", entry.Tokens.Positive),
		fmt.Sprintf("--margo-warning:%s", entry.Tokens.Warning),
		fmt.Sprintf("--margo-negative:%s", entry.Tokens.Negative),
		fmt.Sprintf("--margo-info:%s", entry.Tokens.Info),
		fmt.Sprintf("--margo-font-body:%s", fontStack(entry.Typography.BodyFamily)),
		fmt.Sprintf("--margo-font-heading:%s", fontStack(entry.Typography.HeadingFamily)),
		fmt.Sprintf("--margo-font-code:%s", fontStack(entry.Typography.CodeFamily)),
		fmt.Sprintf("--margo-body-size:%gpx", entry.Typography.BodySize),
		fmt.Sprintf("--margo-body-line:%g", entry.Typography.BodyLine),
		fmt.Sprintf("--margo-h1-size:%gpx", entry.Typography.H1Size),
		fmt.Sprintf("--margo-h1-line:%g", entry.Typography.H1Line),
		fmt.Sprintf("--margo-h2-size:%gpx", entry.Typography.H2Size),
		fmt.Sprintf("--margo-h2-line:%g", entry.Typography.H2Line),
		fmt.Sprintf("--margo-h3-size:%gpx", entry.Typography.H3Size),
		fmt.Sprintf("--margo-h3-line:%g", entry.Typography.H3Line),
		fmt.Sprintf("--margo-padding-x:%gpx", entry.PaddingX),
		fmt.Sprintf("--margo-padding-y:%gpx", entry.PaddingY),
		fmt.Sprintf("--margo-header-zone:%gpx", entry.HeaderZone),
		fmt.Sprintf("--margo-footer-zone:%gpx", entry.FooterZone),
		fmt.Sprintf("--margo-content-gap:%gpx", entry.ContentGap),
		fmt.Sprintf("--margo-card-radius:%gpx", entry.CardRadius),
		fmt.Sprintf("--margo-border-width:%gpx", entry.BorderWidth),
		fmt.Sprintf("--margo-slot-padding:%gpx", entry.SlotPadding),
		fmt.Sprintf("--margo-card-padding:%gpx", entry.CardPadding),
	}, ";")
}

// fontStack keeps the named theme face as the first choice while ensuring a
// missing optional interactive face never falls through to a platform serif.
// Canonical validators still bind the expected versioned bundle digest; these
// stacks are the documented interactive fallback path.
func fontStack(family string) string {
	switch family {
	case "Margo Serif":
		return "'Margo Serif', 'New York', Georgia, 'Times New Roman', serif"
	case "Margo Mono":
		return "'Margo Mono', 'SFMono-Regular', ui-monospace, Menlo, Consolas, monospace"
	case "Margo Sans":
		fallthrough
	default:
		return "'Margo Sans', 'SF Pro Text', 'Helvetica Neue', ui-sans-serif, system-ui, sans-serif"
	}
}
