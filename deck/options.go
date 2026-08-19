package deck

import (
	"fmt"
	"reflect"

	"github.com/araihu/margo"
)

// RenderOption overrides one immutable deck render setting.
type RenderOption func(*renderOptions) error

type renderOptions struct {
	validationRequest *margo.RuntimeValidationRequest
	theme             *margo.ThemeName
	colorMode         *margo.ColorMode
	geometry          *DeckGeometry
	idAllocator       margo.RenderIDAllocator
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
