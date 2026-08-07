package svgprofile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

const expectedFingerprint = "6e4899904bf55acdd2b5c39a290dbac378a7f6fdf8e904b41c38c4d9c3fdda75"

func TestProfile(t *testing.T) {
	profileBytes, err := os.ReadFile("../../profiles/margo-mermaid-svg-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := Decode(profileBytes)
	if err != nil {
		t.Fatal(err)
	}
	if profile.SchemaVersion != "margo-mermaid-svg/v1" {
		t.Fatalf("schemaVersion = %q", profile.SchemaVersion)
	}
	if profile.NormalizationAlgorithm != "margo-mermaid-svg-normalization/v2" {
		t.Fatalf("normalizationAlgorithm = %q", profile.NormalizationAlgorithm)
	}
	wantLengthUnits := []string{"", "%", "em", "pt", "px", "rem"}
	if !reflect.DeepEqual(profile.ValueGrammarParameters.LengthUnits, wantLengthUnits) {
		t.Fatalf("length units = %v, want %v", profile.ValueGrammarParameters.LengthUnits, wantLengthUnits)
	}
	fingerprint, err := Fingerprint(profileBytes)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprint != expectedFingerprint {
		t.Fatalf("fingerprint = %s, want %s", fingerprint, expectedFingerprint)
	}
}

func TestPositiveCorpus(t *testing.T) {
	profile := loadProfile(t)
	got := make([]string, 0, 8)
	for _, family := range profile.SupportedFamilies {
		for _, fixture := range family.Fixtures {
			fixturePath := filepath.Join("../..", filepath.FromSlash(fixture.Path))
			contents, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(contents)
			if actual := hex.EncodeToString(digest[:]); actual != fixture.SHA256 {
				t.Fatalf("%s SHA-256 = %s, want %s", fixture.Path, actual, fixture.SHA256)
			}
			got = append(got, family.Name+"/"+fixture.Variant)
		}
	}
	sort.Strings(got)
	want := []string{
		"flowchart/basic", "flowchart/conditional", "flowchart/id-reference-heavy", "flowchart/style-heavy",
		"sequence/basic", "sequence/conditional", "sequence/id-reference-heavy", "sequence/style-heavy",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("positive corpus = %v, want %v", got, want)
	}
}

func TestNegativeCorpus(t *testing.T) {
	manifestBytes, err := os.ReadFile("../../testdata/mermaid/negative/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	vectors, err := DecodeNegativeCorpus(manifestBytes)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(vectors))
	for _, vector := range vectors {
		if _, err := os.Stat(filepath.Join("../../testdata/mermaid/negative", filepath.FromSlash(vector.Path))); err != nil {
			t.Fatal(err)
		}
		got = append(got, vector.Name)
	}
	sort.Strings(got)
	want := []string{
		"cross-svg-url", "css-at-rule", "css-attribute-selector", "css-body", "css-custom-property",
		"css-forbidden-function", "css-pseudo", "css-sibling-selector", "css-universal-selector",
		"css-unknown-property", "event-handler", "external-link", "foreign-object", "invalid-data-points",
		"invalid-length-unit", "invalid-opacity", "script", "unknown-attribute", "unknown-element", "unknown-namespace", "unrooted-id",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("negative corpus = %v, want %v", got, want)
	}

	var raw []map[string]any
	if err := json.Unmarshal(manifestBytes, &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw) != len(vectors) {
		t.Fatalf("decoded vectors = %d, raw rows = %d", len(vectors), len(raw))
	}
}

func loadProfile(t *testing.T) Profile {
	t.Helper()
	profileBytes, err := os.ReadFile("../../profiles/margo-mermaid-svg-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := Decode(profileBytes)
	if err != nil {
		t.Fatal(err)
	}
	return profile
}
