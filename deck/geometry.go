package deck

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// DeckUnit is the finite absolute-unit vocabulary accepted for custom deck
// geometry. Public geometry is normalized to logical CSS pixels.
type DeckUnit string

const (
	DeckUnitPX DeckUnit = "px"
	DeckUnitMM DeckUnit = "mm"
	DeckUnitCM DeckUnit = "cm"
	DeckUnitIN DeckUnit = "in"
	DeckUnitPT DeckUnit = "pt"
	DeckUnitPC DeckUnit = "pc"
	DeckUnitQ  DeckUnit = "Q"
)

// DeckGeometry is the fixed logical canvas used by screen and print output.
type DeckGeometry struct {
	Preset string
	Width  float64
	Height float64
	Unit   DeckUnit
}

var customGeometryPattern = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)x([0-9]+(?:\.[0-9]+)?)(px|mm|cm|in|pt|pc|Q)?$`)

func DefaultDeckGeometry() DeckGeometry {
	return DeckGeometry{Preset: "16:9", Width: 1280, Height: 720, Unit: DeckUnitPX}
}

// ParseDeckGeometry parses a normalized preset or an absolute custom size.
func ParseDeckGeometry(value string) (DeckGeometry, error) {
	value = strings.TrimSpace(value)
	switch value {
	case "", "16:9":
		return DefaultDeckGeometry(), nil
	case "4:3":
		return DeckGeometry{Preset: "4:3", Width: 960, Height: 720, Unit: DeckUnitPX}, nil
	}
	matches := customGeometryPattern.FindStringSubmatch(value)
	if len(matches) == 0 {
		return DeckGeometry{}, deckError("deck.size_invalid", "", 1, "size must be 16:9, 4:3, or absolute custom dimensions")
	}
	width, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return DeckGeometry{}, deckError("deck.size_invalid", "", 1, "custom width is not finite")
	}
	height, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		return DeckGeometry{}, deckError("deck.size_invalid", "", 1, "custom height is not finite")
	}
	unit := DeckUnit(matches[3])
	if unit == "" {
		unit = DeckUnitPX
	}
	factor, ok := deckUnitFactor(unit)
	if !ok {
		return DeckGeometry{}, deckError("deck.size_invalid", "", 1, "custom size unit is not supported")
	}
	geometry := DeckGeometry{Preset: "custom", Width: width * factor, Height: height * factor, Unit: DeckUnitPX}
	if err := geometry.Validate(); err != nil {
		return DeckGeometry{}, err
	}
	return geometry, nil
}

func (geometry DeckGeometry) Validate() error {
	if geometry.Preset == "16:9" {
		if geometry.Width != 1280 || geometry.Height != 720 {
			return fmt.Errorf("deck.size_invalid: 16:9 geometry must be 1280x720")
		}
		return nil
	}
	if geometry.Preset == "4:3" {
		if geometry.Width != 960 || geometry.Height != 720 {
			return fmt.Errorf("deck.size_invalid: 4:3 geometry must be 960x720")
		}
		return nil
	}
	if geometry.Preset != "custom" {
		return fmt.Errorf("deck.size_invalid: unknown geometry preset %q", geometry.Preset)
	}
	if geometry.Unit != DeckUnitPX {
		return fmt.Errorf("deck.size_invalid: normalized geometry must use px")
	}
	if math.IsNaN(geometry.Width) || math.IsInf(geometry.Width, 0) || math.IsNaN(geometry.Height) || math.IsInf(geometry.Height, 0) || geometry.Width < 320 || geometry.Width > 7680 || geometry.Height < 320 || geometry.Height > 7680 {
		return fmt.Errorf("deck.size_invalid: custom dimensions must be 320 through 7680 logical CSS pixels")
	}
	ratio := geometry.Width / geometry.Height
	if ratio < 0.25 || ratio > 4 {
		return fmt.Errorf("deck.size_invalid: custom aspect ratio must be between 1:4 and 4:1")
	}
	return nil
}

func (geometry DeckGeometry) Equal(other DeckGeometry) bool {
	return geometry.Preset == other.Preset && geometry.Width == other.Width && geometry.Height == other.Height && geometry.Unit == other.Unit
}

func deckUnitFactor(unit DeckUnit) (float64, bool) {
	switch unit {
	case DeckUnitPX:
		return 1, true
	case DeckUnitMM:
		return 96 / 25.4, true
	case DeckUnitCM:
		return 96 / 2.54, true
	case DeckUnitIN:
		return 96, true
	case DeckUnitPT:
		return 96.0 / 72.0, true
	case DeckUnitPC:
		return 16, true
	case DeckUnitQ:
		return 96 / 101.6, true
	default:
		return 0, false
	}
}
