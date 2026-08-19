package pdf

import (
	"fmt"
	"math"
)

// PageSize identifies one physical page size supported by the v1 contract.
type PageSize string

const (
	PageA4     PageSize = "A4"
	PageLetter PageSize = "Letter"
)

// Orientation identifies the physical orientation of a page. An omitted
// orientation uses portrait mode.
type Orientation string

const (
	Portrait  Orientation = "portrait"
	Landscape Orientation = "landscape"
)

// ImageOverflowPolicy controls whether print images are capped to keep one
// image from spanning several pages. The zero value uses ImageOverflowLimit.
type ImageOverflowPolicy string

const (
	ImageOverflowLimit ImageOverflowPolicy = "limit"
	ImageOverflowAllow ImageOverflowPolicy = "allow"

	// DefaultImageMaxHeightPercent is the default maximum image height during
	// print projection, expressed as a percentage of the page viewport.
	DefaultImageMaxHeightPercent = 70
)

// Millimeters is a physical page length. Page margins use millimeters so all
// engines receive the same renderer-neutral values.
type Millimeters float64

// Margins defines the four non-negative physical page margins.
type Margins struct {
	Top    Millimeters `json:"top"`
	Right  Millimeters `json:"right"`
	Bottom Millimeters `json:"bottom"`
	Left   Millimeters `json:"left"`
}

// CustomPageSize is an absolute physical PDF page size in millimetres.
type CustomPageSize struct {
	WidthMM  Millimeters `json:"widthMm"`
	HeightMM Millimeters `json:"heightMm"`
}

// PageConfig is the engine-neutral physical page contract. Headers, footers,
// watermarks, and page numbers remain part of the supplied core HTML/CSS.
type PageConfig struct {
	Size          PageSize            `json:"size"`
	Orientation   Orientation         `json:"orientation"`
	Custom        *CustomPageSize     `json:"custom,omitempty"`
	Margins       Margins             `json:"margins"`
	ImageOverflow ImageOverflowPolicy `json:"imageOverflow,omitempty"`
}

func (config PageConfig) Clone() PageConfig {
	if config.Custom != nil {
		custom := *config.Custom
		config.Custom = &custom
	}
	return config
}

// EffectiveImageOverflowPolicy returns the safe default for an omitted policy.
func (config PageConfig) EffectiveImageOverflowPolicy() ImageOverflowPolicy {
	if config.ImageOverflow == "" {
		return ImageOverflowLimit
	}
	return config.ImageOverflow
}

// Validate rejects values that cannot be represented consistently by every
// engine covered by the v1 contract.
func (config PageConfig) Validate() error {
	if config.Custom != nil {
		if config.Size != "" || config.Orientation != "" {
			return pageValidationError("pdf.page_size_conflict", "custom page size cannot be combined with a named size or orientation")
		}
		for _, value := range []struct {
			name string
			mm   Millimeters
		}{{"width", config.Custom.WidthMM}, {"height", config.Custom.HeightMM}} {
			if math.IsNaN(float64(value.mm)) || math.IsInf(float64(value.mm), 0) || value.mm <= 0 {
				return pageValidationError("pdf.custom_size_invalid", value.name+" must be finite and positive")
			}
		}
	} else {
		switch config.Size {
		case PageA4, PageLetter:
		default:
			return pageValidationError("pdf.page_size_unsupported", "page size must be A4 or Letter")
		}
	}

	switch config.Orientation {
	case "", Portrait, Landscape:
	default:
		return pageValidationError("pdf.orientation_unsupported", "orientation must be portrait or landscape")
	}

	switch config.ImageOverflow {
	case "", ImageOverflowLimit, ImageOverflowAllow:
	default:
		return pageValidationError("pdf.image_overflow_unsupported", "image overflow policy must be limit or allow")
	}

	for _, marginValue := range []struct {
		name  string
		value Millimeters
	}{
		{name: "top", value: config.Margins.Top},
		{name: "right", value: config.Margins.Right},
		{name: "bottom", value: config.Margins.Bottom},
		{name: "left", value: config.Margins.Left},
	} {
		margin := float64(marginValue.value)
		if math.IsNaN(margin) || math.IsInf(margin, 0) || margin < 0 {
			return pageValidationError("pdf.margin_invalid", fmt.Sprintf("%s margin must be finite and non-negative", marginValue.name))
		}
	}
	return nil
}

func pageValidationError(code, message string) error {
	return fmt.Errorf("%s: %s", code, message)
}
