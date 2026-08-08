package svgprofile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"reflect"
	"strings"

	"github.com/araihu/margo/internal/canonicaljson"
)

const fingerprintDomain = "margo/mermaid-svg-profile/v1\n"

// Profile is the closed Mermaid SVG and CSS contract consumed by the browser
// validator. Maps retain the human-reviewed allowlists; no observed SVG may
// add entries to them at runtime.
type Profile struct {
	AssetCount              int                    `json:"assetCount"`
	AssetSetDigest          string                 `json:"assetSetDigest"`
	SchemaVersion           string                 `json:"schemaVersion"`
	MermaidVersion          string                 `json:"mermaidVersion"`
	MermaidDigest           string                 `json:"mermaidDigest"`
	NormalizationAlgorithm  string                 `json:"normalizationAlgorithm"`
	SupportedFamilies       []Family               `json:"supportedFamilies"`
	Namespaces              map[string]string      `json:"namespaces"`
	AllowedElements         []string               `json:"allowedElements"`
	GlobalAttributes        []string               `json:"globalAttributes"`
	ElementAttributes       map[string][]string    `json:"elementAttributes"`
	IDReferenceSites        IDReferenceSites       `json:"idReferenceSites"`
	SelectorGrammar         SelectorGrammar        `json:"selectorGrammar"`
	CSSProperties           map[string]string      `json:"cssProperties"`
	ValueGrammars           map[string]string      `json:"valueGrammars"`
	ValueGrammarParameters  ValueGrammarParameters `json:"valueGrammarParameters"`
	Limits                  Limits                 `json:"limits"`
	NormalizationReductions json.RawMessage        `json:"normalizationReductions"`
}

type ValueGrammarParameters struct {
	LengthUnits []string `json:"lengthUnits"`
}

type Family struct {
	Name     string    `json:"name"`
	Fixtures []Fixture `json:"fixtures"`
}

type Fixture struct {
	Variant string `json:"variant"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
}

type IDReferenceSites struct {
	FragmentAttributes     []string `json:"fragmentAttributes"`
	PresentationAttributes []string `json:"presentationAttributes"`
	ARIAIDREFAttributes    []string `json:"ariaIdrefAttributes"`
	CSS                    []string `json:"css"`
}

type SelectorGrammar struct {
	Selectors     []string `json:"selectors"`
	Combinators   []string `json:"combinators"`
	PseudoClasses []string `json:"pseudoClasses"`
}

type Limits struct {
	MaxSVGBytes      int `json:"maxSvgBytes"`
	MaxElements      int `json:"maxElements"`
	MaxAttributes    int `json:"maxAttributes"`
	MaxCSSRules      int `json:"maxCssRules"`
	MaxSelectorBytes int `json:"maxSelectorBytes"`
}

type NegativeVector struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Family string `json:"family"`
	Code   string `json:"code"`
}

func Decode(data []byte) (Profile, error) {
	var profile Profile
	if err := decodeExact(data, &profile); err != nil {
		return Profile{}, fmt.Errorf("decode Mermaid SVG profile: %w", err)
	}
	if profile.SchemaVersion != "margo-mermaid-svg/v1" || profile.NormalizationAlgorithm != "margo-mermaid-svg-normalization/v2" {
		return Profile{}, fmt.Errorf("unsupported Mermaid SVG profile identity")
	}
	if len(profile.SupportedFamilies) == 0 || len(profile.AllowedElements) == 0 || len(profile.GlobalAttributes) == 0 || len(profile.CSSProperties) == 0 {
		return Profile{}, fmt.Errorf("Mermaid SVG profile allowlists are empty")
	}
	wantLengthUnits := []string{"", "%", "em", "pt", "px", "rem"}
	if !reflect.DeepEqual(profile.ValueGrammarParameters.LengthUnits, wantLengthUnits) {
		return Profile{}, fmt.Errorf("Mermaid SVG profile length units are not the closed v1 set")
	}
	if profile.Limits.MaxSVGBytes <= 0 || profile.Limits.MaxElements <= 0 || profile.Limits.MaxAttributes <= 0 || profile.Limits.MaxCSSRules <= 0 || profile.Limits.MaxSelectorBytes <= 0 {
		return Profile{}, fmt.Errorf("Mermaid SVG profile limits must be positive")
	}
	return profile, nil
}

func Fingerprint(data []byte) (string, error) {
	var decoded any
	if err := decodeExact(data, &decoded); err != nil {
		return "", fmt.Errorf("decode fingerprint preimage: %w", err)
	}
	canonical, err := canonicaljson.Marshal(decoded)
	if err != nil {
		return "", fmt.Errorf("canonicalize fingerprint preimage: %w", err)
	}
	digest := sha256.Sum256(append([]byte(fingerprintDomain), canonical...))
	return hex.EncodeToString(digest[:]), nil
}

func DecodeNegativeCorpus(data []byte) ([]NegativeVector, error) {
	var vectors []NegativeVector
	if err := decodeExact(data, &vectors); err != nil {
		return nil, fmt.Errorf("decode negative Mermaid SVG corpus: %w", err)
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("negative Mermaid SVG corpus is empty")
	}
	seenNames := make(map[string]struct{}, len(vectors))
	seenPaths := make(map[string]struct{}, len(vectors))
	for _, vector := range vectors {
		if vector.Name == "" || vector.Path == "" || (vector.Family != "flowchart" && vector.Family != "sequence") || !strings.HasPrefix(vector.Code, "mermaid.svg_") {
			return nil, fmt.Errorf("invalid negative Mermaid SVG vector %q", vector.Name)
		}
		if path.Base(vector.Path) != vector.Path || path.Ext(vector.Path) != ".svg" {
			return nil, fmt.Errorf("negative Mermaid SVG vector path %q is not a local SVG filename", vector.Path)
		}
		if _, ok := seenNames[vector.Name]; ok {
			return nil, fmt.Errorf("duplicate negative Mermaid SVG vector name %q", vector.Name)
		}
		if _, ok := seenPaths[vector.Path]; ok {
			return nil, fmt.Errorf("duplicate negative Mermaid SVG vector path %q", vector.Path)
		}
		seenNames[vector.Name] = struct{}{}
		seenPaths[vector.Path] = struct{}{}
	}
	return vectors, nil
}

func decodeExact(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("trailing JSON value: %w", err)
	}
	return nil
}
