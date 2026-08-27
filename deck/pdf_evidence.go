package deck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/araihu/margo/internal/canonicaljson"
)

type PDFMediaBoxMicrometers struct {
	Index             int   `json:"index"`
	LeftMicrometers   int64 `json:"leftMicrometers"`
	BottomMicrometers int64 `json:"bottomMicrometers"`
	RightMicrometers  int64 `json:"rightMicrometers"`
	TopMicrometers    int64 `json:"topMicrometers"`
}

type PDFArtifactReport struct {
	PageCount                 int                      `json:"pageCount"`
	MediaBoxesMicrometers     []PDFMediaBoxMicrometers `json:"mediaBoxesMicrometers"`
	EvidenceSHA256            string                   `json:"evidenceSHA256"`
	EvidenceBytes             int64                    `json:"evidenceBytes"`
	Valid                     bool                     `json:"valid"`
	CompositionCatalogVersion string                   `json:"compositionCatalogVersion,omitempty"`
	Compositions              []PDFCompositionIdentity `json:"compositions,omitempty"`
}

type PDFCompositionIdentity struct {
	Name    string   `json:"name"`
	Variant string   `json:"variant"`
	Class   string   `json:"class"`
	Family  string   `json:"family"`
	Slots   []string `json:"slots,omitempty"`
}

type PDFArtifactValidator interface {
	Validate(context.Context, []byte, DeckGeometry, int) (PDFArtifactReport, error)
}

var pdfMediaBoxPattern = regexp.MustCompile(`/MediaBox\s*\[([^\]]+)\]`)

// PDFMediaBoxToleranceMicrometers covers the paper-size quantization introduced
// when Chromium converts CSS pixels to PDF points while preserving material
// geometry errors. Chromium's custom-page projection rounds to a 0.96-point
// grid (about 169 micrometres at its worst), so allow that renderer
// quantization without accepting a millimetre-scale mismatch.
const PDFMediaBoxToleranceMicrometers int64 = 170

// ParsePDFMediaBoxes extracts page media boxes from a PDF byte stream and
// normalizes PDF points into integer micrometres. Chromium emits one media box
// per page in the artifacts accepted by the deck profile.
func ParsePDFMediaBoxes(data []byte) ([]PDFMediaBoxMicrometers, error) {
	matches := pdfMediaBoxPattern.FindAllSubmatch(data, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("deck.pdf_media_box_missing: PDF contains no page media boxes")
	}
	boxes := make([]PDFMediaBoxMicrometers, 0, len(matches))
	for index, match := range matches {
		fields := strings.Fields(string(match[1]))
		if len(fields) != 4 {
			return nil, fmt.Errorf("deck.pdf_media_box_invalid: media box %d must contain four coordinates", index)
		}
		values := [4]float64{}
		for valueIndex, field := range fields {
			value, err := strconv.ParseFloat(field, 64)
			if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
				return nil, fmt.Errorf("deck.pdf_media_box_invalid: media box %d has a non-finite coordinate", index)
			}
			values[valueIndex] = value
		}
		converted := [4]int64{}
		for valueIndex, value := range values {
			converted[valueIndex] = int64(math.Round(value * 25.4 / 72 * 1000))
		}
		boxes = append(boxes, PDFMediaBoxMicrometers{
			Index: index, LeftMicrometers: converted[0], BottomMicrometers: converted[1],
			RightMicrometers: converted[2], TopMicrometers: converted[3],
		})
	}
	return boxes, nil
}

func CSSPixelsToMicrometers(value float64) (int64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("deck.pdf_geometry_invalid: value is not finite")
	}
	return int64(math.Round(value * 25.4 / 96 * 1000)), nil
}

// BuildPDFArtifactReport builds the non-recursive canonical evidence envelope
// for one normalized geometry and the observed page media boxes.
func BuildPDFArtifactReport(widthCSS, heightCSS float64, slideCount int, observed []PDFMediaBoxMicrometers) (PDFArtifactReport, error) {
	return buildPDFArtifactReport(widthCSS, heightCSS, slideCount, observed, nil)
}

// BuildPDFArtifactReportWithComposition adds the resolved R1 composition
// identity to the PDF evidence body while keeping the v0.0.1 four-argument
// builder byte-for-byte compatible when no composition is supplied.
func BuildPDFArtifactReportWithComposition(widthCSS, heightCSS float64, slideCount int, observed []PDFMediaBoxMicrometers, compositions []CompositionSpec) (PDFArtifactReport, error) {
	identities, err := normalizePDFCompositionIdentities(compositions, slideCount)
	if err != nil {
		return PDFArtifactReport{}, err
	}
	return buildPDFArtifactReport(widthCSS, heightCSS, slideCount, observed, identities)
}

func buildPDFArtifactReport(widthCSS, heightCSS float64, slideCount int, observed []PDFMediaBoxMicrometers, compositions []PDFCompositionIdentity) (PDFArtifactReport, error) {
	if slideCount < 0 {
		return PDFArtifactReport{}, fmt.Errorf("deck.pdf_page_count_invalid: slide count is negative")
	}
	right, err := CSSPixelsToMicrometers(widthCSS)
	if err != nil {
		return PDFArtifactReport{}, err
	}
	top, err := CSSPixelsToMicrometers(heightCSS)
	if err != nil {
		return PDFArtifactReport{}, err
	}
	pages := append([]PDFMediaBoxMicrometers(nil), observed...)
	sort.Slice(pages, func(i, j int) bool { return pages[i].Index < pages[j].Index })
	pageProjection := make([]map[string]any, len(pages))
	for index, page := range pages {
		pageProjection[index] = map[string]any{
			"bottomMicrometers": page.BottomMicrometers,
			"index":             page.Index,
			"leftMicrometers":   page.LeftMicrometers,
			"rightMicrometers":  page.RightMicrometers,
			"topMicrometers":    page.TopMicrometers,
		}
	}
	evidence := struct {
		Expected                  map[string]int64         `json:"expected"`
		Pages                     []map[string]any         `json:"pages"`
		SlideCount                int                      `json:"slideCount"`
		Version                   int                      `json:"version"`
		CompositionCatalogVersion string                   `json:"compositionCatalogVersion,omitempty"`
		Compositions              []PDFCompositionIdentity `json:"compositions,omitempty"`
	}{
		Expected: map[string]int64{"bottomMicrometers": 0, "leftMicrometers": 0, "rightMicrometers": right, "topMicrometers": top},
		Pages:    pageProjection, SlideCount: slideCount, Version: 1, Compositions: append([]PDFCompositionIdentity(nil), compositions...),
	}
	if len(compositions) > 0 {
		evidence.CompositionCatalogVersion = CompositionCatalogVersion
	}
	encoded, err := canonicaljson.Marshal(evidence)
	if err != nil {
		return PDFArtifactReport{}, err
	}
	digest := sha256.Sum256(encoded)
	report := PDFArtifactReport{PageCount: len(observed), MediaBoxesMicrometers: pages, EvidenceSHA256: hex.EncodeToString(digest[:]), EvidenceBytes: int64(len(encoded)), CompositionCatalogVersion: evidence.CompositionCatalogVersion, Compositions: append([]PDFCompositionIdentity(nil), compositions...)}
	report.Valid = len(observed) == slideCount
	for index, page := range pages {
		if page.Index != index || !withinPDFTolerance(page.LeftMicrometers, 0) || !withinPDFTolerance(page.BottomMicrometers, 0) || !withinPDFTolerance(page.RightMicrometers, right) || !withinPDFTolerance(page.TopMicrometers, top) {
			report.Valid = false
		}
	}
	return report, nil
}

func normalizePDFCompositionIdentities(specs []CompositionSpec, slideCount int) ([]PDFCompositionIdentity, error) {
	if len(specs) == 0 {
		return nil, nil
	}
	if slideCount != len(specs) {
		return nil, deckError("deck.composition_slots_required", "", 1, "PDF composition identity must cover every slide")
	}
	identities := make([]PDFCompositionIdentity, len(specs))
	for index, spec := range specs {
		if spec.Name == "" || spec.CatalogVersion != CompositionCatalogVersion {
			return nil, deckError("deck.composition_catalog_mismatch", "", index+1, "PDF composition identity is not registered R1")
		}
		resolved, err := ResolveComposition(spec.Name)
		if err != nil || resolved.CatalogVersion != spec.CatalogVersion || resolved.Variant != spec.Variant || resolved.LayoutClass != spec.LayoutClass {
			return nil, deckError("deck.composition_catalog_mismatch", "", index+1, "PDF composition identity does not match the registered catalog")
		}
		identity := PDFCompositionIdentity{Name: string(spec.Name), Variant: spec.Variant, Class: spec.LayoutClass, Family: spec.LayoutClass}
		if len(spec.Slots) > 0 {
			identity.Slots = make([]string, len(spec.Slots))
			for slotIndex, slot := range spec.Slots {
				identity.Slots[slotIndex] = slot.Name
			}
		}
		identities[index] = identity
	}
	return identities, nil
}

func withinPDFTolerance(actual, expected int64) bool {
	difference := actual - expected
	if difference < 0 {
		difference = -difference
	}
	return difference <= PDFMediaBoxToleranceMicrometers
}
