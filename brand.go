package margo

import (
	"fmt"
	"strings"

	"github.com/a-h/templ"
)

const maxBrandWatermarkBytes = 256

// Brand is the trusted, declarative subset of standalone branding. Header and
// Footer are Go components; document-authored markup never populates them.
type Brand struct {
	Header    templ.Component
	Footer    templ.Component
	Logo      AssetRef
	Watermark string
	Tokens    map[DocumentToken]string
}

func (b Brand) clone() Brand {
	b.Logo = b.Logo.clone()
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
	}
	return validateThemeTokens(b.Tokens)
}
