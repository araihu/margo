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
	PageCount             int                      `json:"pageCount"`
	MediaBoxesMicrometers []PDFMediaBoxMicrometers `json:"mediaBoxesMicrometers"`
	EvidenceSHA256        string                   `json:"evidenceSHA256"`
	EvidenceBytes         int64                    `json:"evidenceBytes"`
	Valid                 bool                     `json:"valid"`
}

type PDFArtifactValidator interface {
	Validate(context.Context, []byte, DeckGeometry, int) (PDFArtifactReport, error)
}

var pdfMediaBoxPattern = regexp.MustCompile(`/MediaBox\s*\[([^\]]+)\]`)

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
		Expected   map[string]int64 `json:"expected"`
		Pages      []map[string]any `json:"pages"`
		SlideCount int              `json:"slideCount"`
		Version    int              `json:"version"`
	}{
		Expected: map[string]int64{"bottomMicrometers": 0, "leftMicrometers": 0, "rightMicrometers": right, "topMicrometers": top},
		Pages:    pageProjection, SlideCount: slideCount, Version: 1,
	}
	encoded, err := canonicaljson.Marshal(evidence)
	if err != nil {
		return PDFArtifactReport{}, err
	}
	digest := sha256.Sum256(encoded)
	report := PDFArtifactReport{PageCount: len(observed), MediaBoxesMicrometers: pages, EvidenceSHA256: hex.EncodeToString(digest[:]), EvidenceBytes: int64(len(encoded))}
	report.Valid = len(observed) == slideCount
	for index, page := range pages {
		if page.Index != index || !withinPDFTolerance(page.LeftMicrometers, 0) || !withinPDFTolerance(page.BottomMicrometers, 0) || !withinPDFTolerance(page.RightMicrometers, right) || !withinPDFTolerance(page.TopMicrometers, top) {
			report.Valid = false
		}
	}
	return report, nil
}

func withinPDFTolerance(actual, expected int64) bool {
	difference := actual - expected
	if difference < 0 {
		difference = -difference
	}
	return difference <= 10
}
