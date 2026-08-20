package deck

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/araihu/goshtoso/components/icon/heroicons"
	"github.com/araihu/margo"
)

// RenderOption overrides one immutable deck render setting.
type RenderOption func(*renderOptions) error

// PaginationIconPlacement fixes the icon's position relative to the ordinal.
type PaginationIconPlacement string

const (
	PaginationIconBefore PaginationIconPlacement = "before"
	PaginationIconAfter  PaginationIconPlacement = "after"
)

// PaginationIconConfig describes one trusted Goshtoso catalog icon in the
// bottom-right pagination cluster.
type PaginationIconConfig struct {
	Symbol     string
	Placement  PaginationIconPlacement
	Label      string
	Decorative bool
}

type renderOptions struct {
	validationRequest         *margo.RuntimeValidationRequest
	theme                     *margo.ThemeName
	colorMode                 *margo.ColorMode
	geometry                  *DeckGeometry
	idAllocator               margo.RenderIDAllocator
	confidentialityBadgeLabel string
	paginationIcon            *PaginationIconConfig
}

// WithConfidentialityBadge adds trusted host chrome before each visible page
// ordinal. Markdown cannot set or override this label.
func WithConfidentialityBadge(label string) RenderOption {
	return func(options *renderOptions) error {
		label = strings.TrimSpace(label)
		if label == "" || len([]byte(label)) > 64 || strings.ContainsAny(label, "<>\x00\n\r\t") {
			return deckError("deck.confidentiality_badge_invalid", "", 1, "confidentiality badge must be bounded inline text")
		}
		if options.confidentialityBadgeLabel != "" && options.confidentialityBadgeLabel != label {
			return deckError("deck.presentation_conflict", "", 1, "confidentiality badge overrides disagree")
		}
		options.confidentialityBadgeLabel = label
		return nil
	}
}

// WithPaginationIcon adds one trusted Goshtoso icon to each paginated slide.
// Placement is explicit; informative icons must provide a label.
func WithPaginationIcon(config PaginationIconConfig) RenderOption {
	return func(options *renderOptions) error {
		normalized, err := normalizePaginationIcon(config)
		if err != nil {
			return err
		}
		if options.paginationIcon != nil && *options.paginationIcon != normalized {
			return deckError("deck.presentation_conflict", "", 1, "pagination icon overrides disagree")
		}
		options.paginationIcon = &normalized
		return nil
	}
}

func normalizePaginationIcon(config PaginationIconConfig) (PaginationIconConfig, error) {
	config.Symbol = strings.TrimSpace(config.Symbol)
	config.Label = strings.TrimSpace(config.Label)
	if config.Symbol == "" || !goshtosoIconSymbol(config.Symbol) {
		return PaginationIconConfig{}, deckError("deck.pagination_icon_invalid", "", 1, "pagination icon symbol is not in the Goshtoso catalog")
	}
	if config.Placement != PaginationIconBefore && config.Placement != PaginationIconAfter {
		return PaginationIconConfig{}, deckError("deck.pagination_icon_invalid", "", 1, "pagination icon placement must be before or after")
	}
	if len([]byte(config.Label)) > 64 || strings.ContainsAny(config.Label, "<>\x00\n\r\t") {
		return PaginationIconConfig{}, deckError("deck.pagination_icon_invalid", "", 1, "pagination icon label must be bounded inline text")
	}
	if !config.Decorative && config.Label == "" {
		return PaginationIconConfig{}, deckError("deck.pagination_icon_invalid", "", 1, "informative pagination icons require a label")
	}
	return config, nil
}

func goshtosoIconSymbol(symbol string) bool {
	for _, glyph := range heroicons.Glyphs {
		if string(glyph.Symbol) == symbol {
			return true
		}
	}
	return false
}

func WithTheme(theme margo.ThemeName) RenderOption {
	return func(options *renderOptions) error {
		if options.theme != nil && *options.theme != theme {
			return deckError("deck.presentation_conflict", "", 1, "theme overrides disagree")
		}
		copy := theme
		options.theme = &copy
		return nil
	}
}

func WithColorMode(mode margo.ColorMode) RenderOption {
	return func(options *renderOptions) error {
		if options.colorMode != nil && *options.colorMode != mode {
			return deckError("deck.presentation_conflict", "", 1, "color mode overrides disagree")
		}
		copy := mode
		options.colorMode = &copy
		return nil
	}
}

func WithGeometry(geometry DeckGeometry) RenderOption {
	return func(options *renderOptions) error {
		if err := geometry.Validate(); err != nil {
			return err
		}
		if options.geometry != nil && !options.geometry.Equal(geometry) {
			return deckError("deck.geometry_conflict", "", 1, "geometry overrides disagree")
		}
		copy := geometry
		options.geometry = &copy
		return nil
	}
}

// WithRenderIDAllocator supplies a trusted render-wide identity registry to
// every slide and structural slot.
func WithRenderIDAllocator(allocator margo.RenderIDAllocator) RenderOption {
	return func(options *renderOptions) error {
		if allocator == nil {
			return deckError("deck.extension_id_unsafe", "", 1, "render ID allocator is nil")
		}
		if options.idAllocator != nil && options.idAllocator != allocator {
			return deckError("deck.presentation_conflict", "", 1, "render ID allocators disagree")
		}
		options.idAllocator = allocator
		return nil
	}
}

// WithValidationRequest selects a pinned viewport/profile for runtime
// evidence. The expected font digest is derived from the selected theme lock;
// a caller may only repeat that value, never replace it.
func WithValidationRequest(request margo.RuntimeValidationRequest) RenderOption {
	return func(options *renderOptions) error {
		if options.validationRequest != nil && !reflect.DeepEqual(*options.validationRequest, request) {
			return fmt.Errorf("deck.presentation_conflict: validation requests disagree")
		}
		copy := request
		options.validationRequest = &copy
		return nil
	}
}

func applyRenderOptions(options []RenderOption) (renderOptions, error) {
	result := renderOptions{}
	for index, option := range options {
		if option == nil {
			return renderOptions{}, fmt.Errorf("deck.option_invalid: nil render option at index %d", index)
		}
		if err := option(&result); err != nil {
			return renderOptions{}, err
		}
	}
	return result, nil
}

func resolveValidationRequest(theme margo.ThemeName, override *margo.RuntimeValidationRequest) (margo.RuntimeValidationRequest, error) {
	digest, err := bundledFontDigest(theme)
	if err != nil {
		return margo.RuntimeValidationRequest{}, err
	}
	request := margo.RuntimeValidationRequest{
		ViewportWidth: 1440, ViewportHeight: 900, DeviceScaleFactor: 1, Zoom: 1,
		BrowserProfile: "chromium-deck-v1", ExpectedFontBundleDigest: digest,
	}
	if override != nil {
		if override.ViewportWidth != 0 {
			request.ViewportWidth = override.ViewportWidth
		}
		if override.ViewportHeight != 0 {
			request.ViewportHeight = override.ViewportHeight
		}
		if override.DeviceScaleFactor != 0 {
			request.DeviceScaleFactor = override.DeviceScaleFactor
		}
		if override.Zoom != 0 {
			request.Zoom = override.Zoom
		}
		if override.BrowserProfile != "" {
			if override.BrowserProfile != "chromium-deck-v1" {
				return margo.RuntimeValidationRequest{}, deckError("deck.validator_profile_mismatch", "", 1, "browser profile is not registered for the v0.0.1 deck profile")
			}
			request.BrowserProfile = override.BrowserProfile
		}
		if override.ExpectedFontBundleDigest != "" && override.ExpectedFontBundleDigest != digest {
			return margo.RuntimeValidationRequest{}, deckError("deck.validator_profile_mismatch", "", 1, "expected font bundle digest does not match the selected theme lock")
		}
	}
	if err := request.Validate(); err != nil {
		return margo.RuntimeValidationRequest{}, deckError("deck.validator_profile_mismatch", "", 1, err.Error())
	}
	return request, nil
}
