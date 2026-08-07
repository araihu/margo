package profiles_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/araihu/margo/internal/canonicaljson"
)

const expectedProfileFingerprint = "6e4899904bf55acdd2b5c39a290dbac378a7f6fdf8e904b41c38c4d9c3fdda75"

type profile struct {
	AssetCount              int                    `json:"assetCount"`
	AssetSetDigest          string                 `json:"assetSetDigest"`
	SchemaVersion           string                 `json:"schemaVersion"`
	MermaidVersion          string                 `json:"mermaidVersion"`
	MermaidDigest           string                 `json:"mermaidDigest"`
	NormalizationAlgorithm  string                 `json:"normalizationAlgorithm"`
	NormalizationReductions json.RawMessage        `json:"normalizationReductions"`
	SupportedFamilies       []family               `json:"supportedFamilies"`
	Namespaces              map[string]string      `json:"namespaces"`
	AllowedElements         []string               `json:"allowedElements"`
	GlobalAttributes        []string               `json:"globalAttributes"`
	ElementAttributes       map[string][]string    `json:"elementAttributes"`
	IDReferenceSites        idReferenceSites       `json:"idReferenceSites"`
	SelectorGrammar         selectorGrammar        `json:"selectorGrammar"`
	CSSProperties           map[string]string      `json:"cssProperties"`
	ValueGrammars           map[string]string      `json:"valueGrammars"`
	ValueGrammarParameters  valueGrammarParameters `json:"valueGrammarParameters"`
	Limits                  limits                 `json:"limits"`
}

type valueGrammarParameters struct {
	LengthUnits []string `json:"lengthUnits"`
}

type family struct {
	Name     string    `json:"name"`
	Fixtures []fixture `json:"fixtures"`
}

type fixture struct {
	Variant string `json:"variant"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
}

type idReferenceSites struct {
	FragmentAttributes     []string `json:"fragmentAttributes"`
	PresentationAttributes []string `json:"presentationAttributes"`
	ARIAIDREFAttributes    []string `json:"ariaIdrefAttributes"`
	CSS                    []string `json:"css"`
}

type selectorGrammar struct {
	Selectors     []string `json:"selectors"`
	Combinators   []string `json:"combinators"`
	PseudoClasses []string `json:"pseudoClasses"`
}

type limits struct {
	MaxSVGBytes      int `json:"maxSvgBytes"`
	MaxElements      int `json:"maxElements"`
	MaxAttributes    int `json:"maxAttributes"`
	MaxCSSRules      int `json:"maxCssRules"`
	MaxSelectorBytes int `json:"maxSelectorBytes"`
}

func TestProfilePreimageIsCanonicalAndPinned(t *testing.T) {
	profileBytes := readProfile(t)
	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(profileBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	canonical, err := canonicaljson.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	preimage := append([]byte("margo/mermaid-svg-profile/v1\n"), canonical...)
	digest := sha256.Sum256(preimage)
	got := hex.EncodeToString(digest[:])
	if got != expectedProfileFingerprint {
		t.Fatalf("profile fingerprint = %s, want %s", got, expectedProfileFingerprint)
	}
}

func TestFamilyCorpusIsClosedAndHasRequiredVariants(t *testing.T) {
	p := decodeProfile(t)
	if p.SchemaVersion != "margo-mermaid-svg/v1" {
		t.Fatalf("schemaVersion = %q", p.SchemaVersion)
	}
	if p.NormalizationAlgorithm != "margo-mermaid-svg-normalization/v2" {
		t.Fatalf("normalizationAlgorithm = %q", p.NormalizationAlgorithm)
	}
	assertNormalizationReductions(t, p.NormalizationReductions)
	wantFamilies := []string{"flowchart", "sequence"}
	wantVariants := []string{"basic", "conditional", "id-reference-heavy", "style-heavy"}
	var gotFamilies []string
	profilePaths := map[string]bool{}
	for _, family := range p.SupportedFamilies {
		gotFamilies = append(gotFamilies, family.Name)
		var variants []string
		for _, fixture := range family.Fixtures {
			variants = append(variants, fixture.Variant)
			if profilePaths[fixture.Path] {
				t.Fatalf("duplicate fixture path %q", fixture.Path)
			}
			profilePaths[fixture.Path] = true
			fixtureBytes, err := os.ReadFile(filepath.Join("..", fixture.Path))
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(fixtureBytes)
			if got := hex.EncodeToString(digest[:]); got != fixture.SHA256 {
				t.Fatalf("fixture %s SHA-256 = %s, want %s", fixture.Path, got, fixture.SHA256)
			}
		}
		sort.Strings(variants)
		if !reflect.DeepEqual(variants, wantVariants) {
			t.Fatalf("family %s variants = %v, want %v", family.Name, variants, wantVariants)
		}
	}
	if !reflect.DeepEqual(gotFamilies, wantFamilies) {
		t.Fatalf("families = %v, want %v", gotFamilies, wantFamilies)
	}

	matches, err := filepath.Glob("../testdata/mermaid/positive/*.mmd")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != len(profilePaths) {
		t.Fatalf("fixture count = %d, profile paths = %d", len(matches), len(profilePaths))
	}
	for _, match := range matches {
		path := strings.TrimPrefix(filepath.ToSlash(match), "../")
		if !profilePaths[path] {
			t.Fatalf("unclaimed positive fixture %q", path)
		}
	}
}

func assertNormalizationReductions(t *testing.T, got json.RawMessage) {
	t.Helper()
	if len(got) == 0 {
		t.Fatal("normalizationReductions is required")
	}
	wantBytes, err := os.ReadFile("../docs/proposals/MERMAID_NORMALIZATION_REDUCTIONS_V2.proposed.json")
	if err != nil {
		t.Fatal(err)
	}
	var gotValue, wantValue any
	for name, item := range map[string]struct {
		data []byte
		out  *any
	}{
		"profile normalizationReductions": {data: got, out: &gotValue},
		"approved reduction proposal":     {data: wantBytes, out: &wantValue},
	} {
		decoder := json.NewDecoder(bytes.NewReader(item.data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(item.out); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatal("normalizationReductions differs from the approved proposal")
	}
}

func TestProfileContainsClosedSecurityGrammar(t *testing.T) {
	p := decodeProfile(t)
	if len(p.AllowedElements) == 0 || len(p.GlobalAttributes) == 0 || len(p.ElementAttributes) == 0 {
		t.Fatal("SVG element and attribute allowlists must be non-empty")
	}
	wantLengthUnits := []string{"", "%", "em", "pt", "px", "rem"}
	if !reflect.DeepEqual(p.ValueGrammarParameters.LengthUnits, wantLengthUnits) {
		t.Fatalf("length units = %v, want %v", p.ValueGrammarParameters.LengthUnits, wantLengthUnits)
	}
	if len(p.IDReferenceSites.FragmentAttributes) == 0 || len(p.IDReferenceSites.PresentationAttributes) == 0 || len(p.IDReferenceSites.ARIAIDREFAttributes) == 0 || len(p.IDReferenceSites.CSS) == 0 {
		t.Fatal("ID reference registry must declare every site family")
	}
	for _, property := range []string{"background-color", "cursor", "text-align"} {
		if _, ok := p.CSSProperties[property]; !ok {
			t.Fatalf("required flowchart CSS property %q missing", property)
		}
	}
	if p.Limits.MaxSVGBytes <= 0 || p.Limits.MaxElements <= 0 || p.Limits.MaxAttributes <= 0 || p.Limits.MaxCSSRules <= 0 || p.Limits.MaxSelectorBytes <= 0 {
		t.Fatal("all profile limits must be positive")
	}
}

func readProfile(t *testing.T) []byte {
	t.Helper()
	profileBytes, err := os.ReadFile("margo-mermaid-svg-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	return profileBytes
}

func decodeProfile(t *testing.T) profile {
	t.Helper()
	var p profile
	decoder := json.NewDecoder(bytes.NewReader(readProfile(t)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&p); err != nil {
		t.Fatal(err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatal(fmt.Errorf("profile contains trailing JSON value: %w", err))
	}
	return p
}
