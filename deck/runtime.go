package deck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"

	"github.com/araihu/margo"
	"github.com/araihu/margo/internal/canonicaljson"
)

// LayoutValidationMode identifies the two canonical deck layout tasks.
type LayoutValidationMode string

const (
	LayoutValidationModeScreen   LayoutValidationMode = "screen"
	LayoutValidationModePrintDOM LayoutValidationMode = "print-dom"
)

type LogicalCanvas struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type StageMetrics struct {
	ControlsReserved float64 `json:"controlsReserved"`
	OriginX          float64 `json:"originX"`
	OriginY          float64 `json:"originY"`
	Scale            float64 `json:"scale"`
}

type LayoutRect struct {
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
	Right  float64 `json:"right"`
	Top    float64 `json:"top"`
}

type SlideLayoutMetrics struct {
	ID              string     `json:"id"`
	ClientHeight    float64    `json:"clientHeight"`
	ClientWidth     float64    `json:"clientWidth"`
	ContentHeight   float64    `json:"contentHeight"`
	ContentWidth    float64    `json:"contentWidth"`
	DescendantUnion LayoutRect `json:"descendantUnion"`
	ScrollHeight    float64    `json:"scrollHeight"`
	ScrollWidth     float64    `json:"scrollWidth"`
}

// LayoutValidationEnvelope is the canonical mode-bound browser evidence body.
type LayoutValidationEnvelope struct {
	Version                   int                            `json:"version"`
	Mode                      LayoutValidationMode           `json:"mode"`
	LogicalCanvas             LogicalCanvas                  `json:"logicalCanvas"`
	Slides                    []SlideLayoutMetrics           `json:"slides"`
	Stage                     StageMetrics                   `json:"stage"`
	ValidationRequest         margo.RuntimeValidationRequest `json:"validationRequest"`
	CompositionCatalogVersion string                         `json:"compositionCatalogVersion,omitempty"`
}

type layoutSlideInput struct {
	ID          string   `json:"id"`
	Classes     []string `json:"classes"`
	Layout      string   `json:"layout"`
	Composition string   `json:"composition,omitempty"`
	Variant     string   `json:"variant,omitempty"`
	Class       string   `json:"class,omitempty"`
	Family      string   `json:"family,omitempty"`
	Slots       []string `json:"slots,omitempty"`
}

// LayoutValidator is the browser-owned screen/print validation seam.
type LayoutValidator interface {
	Validate(context.Context, []byte, margo.RuntimeDescriptor) (margo.RuntimeReport, error)
}

// CanonicalLayoutValidationEnvelope serializes quantized logical evidence.
func CanonicalLayoutValidationEnvelope(envelope LayoutValidationEnvelope) ([]byte, error) {
	envelope.Version = 1
	if envelope.Mode != LayoutValidationModeScreen && envelope.Mode != LayoutValidationModePrintDOM {
		return nil, deckError("deck.validation_mode_mismatch", "", 1, "unknown layout validation mode")
	}
	if envelope.CompositionCatalogVersion != "" && envelope.CompositionCatalogVersion != CompositionCatalogVersion {
		return nil, deckError("deck.composition_catalog_mismatch", "", 1, "unsupported composition catalog version")
	}
	if err := envelope.ValidationRequest.Validate(); err != nil {
		return nil, err
	}
	if err := validateLayoutEnvelope(envelope); err != nil {
		return nil, err
	}
	envelope.LogicalCanvas.Width = quantizeLogical(envelope.LogicalCanvas.Width)
	envelope.LogicalCanvas.Height = quantizeLogical(envelope.LogicalCanvas.Height)
	envelope.Stage.ControlsReserved = quantizeLogical(envelope.Stage.ControlsReserved)
	envelope.Stage.OriginX = quantizeLogical(envelope.Stage.OriginX)
	envelope.Stage.OriginY = quantizeLogical(envelope.Stage.OriginY)
	envelope.Stage.Scale = quantizeLogical(envelope.Stage.Scale)
	for index := range envelope.Slides {
		metrics := &envelope.Slides[index]
		metrics.ClientHeight = quantizeLogical(metrics.ClientHeight)
		metrics.ClientWidth = quantizeLogical(metrics.ClientWidth)
		metrics.ContentHeight = quantizeLogical(metrics.ContentHeight)
		metrics.ContentWidth = quantizeLogical(metrics.ContentWidth)
		metrics.ScrollHeight = quantizeLogical(metrics.ScrollHeight)
		metrics.ScrollWidth = quantizeLogical(metrics.ScrollWidth)
		metrics.DescendantUnion.Bottom = quantizeLogical(metrics.DescendantUnion.Bottom)
		metrics.DescendantUnion.Left = quantizeLogical(metrics.DescendantUnion.Left)
		metrics.DescendantUnion.Right = quantizeLogical(metrics.DescendantUnion.Right)
		metrics.DescendantUnion.Top = quantizeLogical(metrics.DescendantUnion.Top)
	}
	return canonicaljson.Marshal(envelope)
}

func validateLayoutEnvelope(envelope LayoutValidationEnvelope) error {
	finitePositive := func(value float64) bool {
		return !math.IsNaN(value) && !math.IsInf(value, 0) && value > 0
	}
	finiteNonNegative := func(value float64) bool {
		return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
	}
	if !finitePositive(envelope.LogicalCanvas.Width) || !finitePositive(envelope.LogicalCanvas.Height) ||
		!finiteNonNegative(envelope.Stage.ControlsReserved) || !finiteNonNegative(envelope.Stage.OriginX) ||
		!finiteNonNegative(envelope.Stage.OriginY) || !finitePositive(envelope.Stage.Scale) {
		return deckError("deck.layout_evidence_invalid", "", 1, "layout canvas and stage metrics must be finite")
	}
	seen := make(map[string]struct{}, len(envelope.Slides))
	for index, slide := range envelope.Slides {
		if slide.ID == "" {
			return deckError("deck.layout_evidence_invalid", "", index+1, "slide identity is required")
		}
		if _, exists := seen[slide.ID]; exists {
			return deckError("deck.layout_evidence_invalid", "", index+1, "slide identity is duplicated")
		}
		seen[slide.ID] = struct{}{}
		values := []float64{slide.ClientHeight, slide.ClientWidth, slide.ContentHeight, slide.ContentWidth, slide.ScrollHeight, slide.ScrollWidth, slide.DescendantUnion.Bottom, slide.DescendantUnion.Left, slide.DescendantUnion.Right, slide.DescendantUnion.Top}
		for _, value := range values {
			if !finiteNonNegative(value) {
				return deckError("deck.layout_evidence_invalid", "", index+1, "slide metrics must be finite and non-negative")
			}
		}
	}
	return nil
}

func quantizeLogical(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Round(value*64) / 64
}

func (r *Result) RuntimeDescriptor(instance margo.RenderInstanceID) (margo.RuntimeDescriptor, error) {
	return r.runtimeDescriptor(instance, true)
}

// ScreenRuntimeDescriptor returns the advisory profile-bound descriptor used
// by embedded hosts; it deliberately omits the print-DOM task.
func (r *Result) ScreenRuntimeDescriptor(instance margo.RenderInstanceID) (margo.RuntimeDescriptor, error) {
	return r.runtimeDescriptor(instance, false)
}

func (r *Result) runtimeDescriptor(instance margo.RenderInstanceID, includePrint bool) (margo.RuntimeDescriptor, error) {
	if r == nil {
		return margo.RuntimeDescriptor{}, deckError("deck.result_required", "", 1, "deck result is required")
	}
	parts := make([]margo.RuntimeDescriptor, len(r.slideResults))
	for index, result := range r.slideResults {
		privateInstance, err := slideInstanceID(index)
		if err != nil {
			return margo.RuntimeDescriptor{}, err
		}
		parts[index], err = result.RuntimeDescriptor(privateInstance)
		if err != nil {
			return margo.RuntimeDescriptor{}, err
		}
	}
	base, err := margo.ComposeRuntimeDescriptors(r.fingerprint, instance, parts...)
	if err != nil {
		return margo.RuntimeDescriptor{}, err
	}
	request := r.validationRequest
	base.Protocol = margo.RuntimeProtocolV2
	base.ValidationRequest = &request
	base.Tasks = append([]margo.RuntimeTask(nil), base.Tasks...)
	dependencies := make([]string, 0, len(base.Tasks))
	for _, task := range base.Tasks {
		dependencies = append(dependencies, task.ID)
	}
	sort.Strings(dependencies)
	for _, mode := range []LayoutValidationMode{LayoutValidationModeScreen, LayoutValidationModePrintDOM} {
		if mode == LayoutValidationModePrintDOM && !includePrint {
			break
		}
		digest, err := r.layoutTaskInputDigest(mode)
		if err != nil {
			return margo.RuntimeDescriptor{}, err
		}
		kind := "deck-layout-" + string(mode)
		taskID := fmt.Sprintf("%s:%s:%08d:%s", instance, kind, 0, digest)
		base.Tasks = append(base.Tasks, margo.RuntimeTask{ID: taskID, Kind: kind, InputSHA256: digest, DependsOn: append([]string{}, dependencies...)})
	}
	if err := margo.ValidateRuntimeDescriptor(base); err != nil {
		return margo.RuntimeDescriptor{}, err
	}
	return base, nil
}

func (r *Result) layoutTaskInputDigest(mode LayoutValidationMode) (string, error) {
	input := struct {
		Version                   int                            `json:"version"`
		Document                  string                         `json:"documentFingerprint"`
		Theme                     margo.ThemeName                `json:"theme"`
		ColorMode                 margo.ColorMode                `json:"colorMode"`
		Geometry                  DeckGeometry                   `json:"geometry"`
		Mode                      LayoutValidationMode           `json:"mode"`
		Slides                    []layoutSlideInput             `json:"slides"`
		ValidationRequest         margo.RuntimeValidationRequest `json:"validationRequest"`
		OverflowVersion           string                         `json:"overflowVersion"`
		CompositionCatalogVersion string                         `json:"compositionCatalogVersion,omitempty"`
	}{Version: 1, Document: r.fingerprint.String(), Theme: r.theme, ColorMode: r.colorMode, Geometry: r.geometry, Mode: mode, ValidationRequest: r.validationRequest, OverflowVersion: "logical-1/64-v1"}
	if r.compositionCatalogVersion != "" {
		input.CompositionCatalogVersion = r.compositionCatalogVersion
	}
	input.Slides = make([]layoutSlideInput, 0, r.slideCount)
	for index := 0; index < r.slideCount; index++ {
		if index < len(r.layoutInputs) {
			input.Slides = append(input.Slides, r.layoutInputs[index])
			continue
		}
		input.Slides = append(input.Slides, layoutSlideInput{ID: fmt.Sprintf("slide-%04d", index+1)})
	}
	encoded, err := canonicaljson.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("deck.layout_input_invalid: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
