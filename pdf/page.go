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

// PageConfig is the engine-neutral physical page contract. Headers, footers,
// watermarks, and page numbers remain part of the supplied core HTML/CSS.
type PageConfig struct {
	Size        PageSize    `json:"size"`
	Orientation Orientation `json:"orientation"`
	Margins     Margins     `json:"margins"`
}

// Validate rejects values that cannot be represented consistently by every
// engine covered by the v1 contract.
func (config PageConfig) Validate() error {
	switch config.Size {
	case PageA4, PageLetter:
	default:
		return pageValidationError("pdf.page_size_unsupported", "page size must be A4 or Letter")
	}

	switch config.Orientation {
	case "", Portrait, Landscape:
	default:
		return pageValidationError("pdf.orientation_unsupported", "orientation must be portrait or landscape")
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
