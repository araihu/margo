package margo

import (
	"fmt"
	"strings"

	"github.com/a-h/templ"
)

const maxBrandWatermarkBytes = 256

const (
	maxBrandAltBytes   = 160
	maxBrandStamps     = 8
	maxBrandStampBytes = 64
)

// Brand is the trusted, declarative subset of standalone branding. Header and
// Footer are Go components; document-authored markup never populates them.
type Brand struct {
	Header    templ.Component
	Footer    templ.Component
	Logo      AssetRef
	LogoAlt   string
	Backdrop  AssetRef
	Watermark string
	Stamps    []string
	Tokens    map[DocumentToken]string
}

func (b Brand) clone() Brand {
	b.Logo = b.Logo.clone()
	b.Backdrop = b.Backdrop.clone()
	b.Stamps = append([]string(nil), b.Stamps...)
	b.Tokens = make(map[DocumentToken]string, len(b.Tokens))
	for key, value := range b.Tokens {
		b.Tokens[key] = value
	}
	return b
}

// Validate rejects invalid overrides instead of falling back to embedded
// assets or silently dropping unsafe customization.
func (b Brand) Validate() error {
	if len([]byte(b.Watermark)) > maxBrandWatermarkBytes {
		return fmt.Errorf("margo: brand watermark exceeds %d bytes", maxBrandWatermarkBytes)
	}
	if strings.ContainsAny(b.Watermark, "<>\x00\n\r") {
		return fmt.Errorf("margo: brand watermark contains forbidden characters")
	}
	if b.Logo.Path != "" {
		if err := b.Logo.validate(); err != nil {
			return err
		}
		if b.Logo.MediaType != "image/svg+xml" {
			return fmt.Errorf("margo: brand logo must be SVG")
		}
		if strings.TrimSpace(b.LogoAlt) == "" || len([]byte(b.LogoAlt)) > maxBrandAltBytes || strings.ContainsAny(b.LogoAlt, "<>\x00\n\r") {
			return fmt.Errorf("margo: brand logo requires bounded alternative text")
		}
	} else if b.LogoAlt != "" {
		return fmt.Errorf("margo: brand logo alternative text requires a logo")
	}
	if b.Backdrop.Path != "" {
		if err := b.Backdrop.validate(); err != nil {
			return err
		}
		if b.Backdrop.MediaType != "image/svg+xml" {
			return fmt.Errorf("margo: brand backdrop must be SVG")
		}
	}
	if len(b.Stamps) > maxBrandStamps {
		return fmt.Errorf("margo: brand stamps exceed %d entries", maxBrandStamps)
	}
	for _, stamp := range b.Stamps {
		if strings.TrimSpace(stamp) == "" || len([]byte(stamp)) > maxBrandStampBytes || strings.ContainsAny(stamp, "<>\x00\n\r") {
			return fmt.Errorf("margo: invalid brand stamp")
		}
		if strings.EqualFold(strings.TrimSpace(stamp), "human review") {
			return fmt.Errorf("margo: brand review stamp must state pending, completed, or required")
		}
	}
	return validateThemeTokens(b.Tokens)
}
