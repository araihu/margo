package margo

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
)

type HTMLRequirementKind string

const (
	HTMLStylesheet  HTMLRequirementKind = "stylesheet"
	HTMLScript      HTMLRequirementKind = "script"
	HTMLRuntimeRole HTMLRequirementKind = "runtime-role"
)

const (
	maxHTMLRequirementInlineBytes       = 4 << 20
	maxHTMLRequirementsTotalInlineBytes = 8 << 20
	maxHTMLRequirementCapabilityBytes   = 8 << 20
)

var htmlRequirementIDPattern = regexp.MustCompile(`^[a-z][a-z0-9.-]{0,127}$`)

const htmlRequirementCapabilityPrefix = "margo-html-requirement/v1:"

type HTMLRequirement struct {
	ID        string
	Kind      HTMLRequirementKind
	LocalURL  string
	Integrity string
	LoadAfter []string
	Inline    AssetRef
}

type HTMLRequirements struct {
	requirements []HTMLRequirement
}

type htmlRequirementCapabilityValue struct {
	ID        string                          `json:"id"`
	Kind      HTMLRequirementKind             `json:"kind"`
	LocalURL  string                          `json:"localURL,omitempty"`
	Integrity string                          `json:"integrity,omitempty"`
	LoadAfter []string                        `json:"loadAfter,omitempty"`
	Inline    *htmlRequirementCapabilityAsset `json:"inline,omitempty"`
}

type htmlRequirementCapabilityAsset struct {
	Path      string `json:"path"`
	MediaType string `json:"mediaType"`
	SHA256    string `json:"sha256"`
	Content   []byte `json:"content"`
}

func (r HTMLRequirements) List() []HTMLRequirement {
	return cloneHTMLRequirements(r.requirements)
}

func cloneHTMLRequirements(requirements []HTMLRequirement) []HTMLRequirement {
	if len(requirements) == 0 {
		return nil
	}
	cloned := make([]HTMLRequirement, len(requirements))
	for index, requirement := range requirements {
		cloned[index] = cloneHTMLRequirement(requirement)
	}
	return cloned
}

func cloneHTMLRequirement(requirement HTMLRequirement) HTMLRequirement {
	requirement.LoadAfter = append([]string(nil), requirement.LoadAfter...)
	requirement.Inline = requirement.Inline.clone()
	return requirement
}

func HTMLRequirementCapability(requirement HTMLRequirement) (string, error) {
	normalized, err := normalizeHTMLRequirement(requirement)
	if err != nil {
		return "", err
	}
	value := htmlRequirementCapabilityValue{
		ID:        normalized.ID,
		Kind:      normalized.Kind,
		LocalURL:  normalized.LocalURL,
		Integrity: normalized.Integrity,
		LoadAfter: append([]string(nil), normalized.LoadAfter...),
		Inline:    htmlRequirementCapabilityAssetFrom(normalized.Inline),
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", htmlRequirementError("html.requirement_invalid", err.Error())
	}
	return htmlRequirementCapabilityPrefix + base64.RawURLEncoding.EncodeToString(data), nil
}

func decodeHTMLRequirementCapability(capability string) (HTMLRequirement, bool, error) {
	if !strings.HasPrefix(capability, htmlRequirementCapabilityPrefix) {
		return HTMLRequirement{}, false, nil
	}
	encoded := strings.TrimPrefix(capability, htmlRequirementCapabilityPrefix)
	if len(encoded) > maxHTMLRequirementCapabilityBytes {
		return HTMLRequirement{}, true, htmlRequirementError("html.requirement_invalid", "capability exceeds the encoded byte limit")
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return HTMLRequirement{}, true, htmlRequirementError("html.requirement_invalid", "capability is not valid base64url")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var value htmlRequirementCapabilityValue
	if err := decoder.Decode(&value); err != nil {
		return HTMLRequirement{}, true, htmlRequirementError("html.requirement_invalid", fmt.Sprintf("capability JSON: %v", err))
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return HTMLRequirement{}, true, htmlRequirementError("html.requirement_invalid", "capability has trailing JSON data")
	}
	requirement := HTMLRequirement{
		ID:        value.ID,
		Kind:      value.Kind,
		LocalURL:  value.LocalURL,
		Integrity: value.Integrity,
		LoadAfter: append([]string(nil), value.LoadAfter...),
		Inline:    htmlRequirementAssetFromCapability(value.Inline),
	}
	normalized, err := normalizeHTMLRequirement(requirement)
	if err != nil {
		return HTMLRequirement{}, true, err
	}
	return normalized, true, nil
}

func htmlRequirementCapabilityAssetFrom(asset AssetRef) *htmlRequirementCapabilityAsset {
	if assetRefIsZero(asset) {
		return nil
	}
	return &htmlRequirementCapabilityAsset{
		Path: asset.Path, MediaType: asset.MediaType, SHA256: asset.SHA256,
		Content: append([]byte(nil), asset.Content...),
	}
}

func htmlRequirementAssetFromCapability(asset *htmlRequirementCapabilityAsset) AssetRef {
	if asset == nil {
		return AssetRef{}
	}
	return AssetRef{
		Path: asset.Path, MediaType: asset.MediaType, SHA256: asset.SHA256,
		Content: append([]byte(nil), asset.Content...),
	}
}

func assetRefIsZero(asset AssetRef) bool {
	return asset.Path == "" && asset.MediaType == "" && asset.SHA256 == "" && len(asset.Content) == 0
}

func mergeHTMLRequirements(input []HTMLRequirement) (HTMLRequirements, error) {
	byID := make(map[string]HTMLRequirement, len(input))
	for _, candidate := range input {
		normalized, err := normalizeHTMLRequirement(candidate)
		if err != nil {
			return HTMLRequirements{}, err
		}
		if existing, found := byID[normalized.ID]; found {
			if !equalHTMLRequirement(existing, normalized) {
				return HTMLRequirements{}, htmlRequirementError("html.requirement_conflict", fmt.Sprintf("requirement %q has conflicting identities", normalized.ID))
			}
			continue
		}
		byID[normalized.ID] = normalized
	}

	totalInlineBytes := 0
	for _, requirement := range byID {
		totalInlineBytes += len(requirement.Inline.Content)
		if totalInlineBytes > maxHTMLRequirementsTotalInlineBytes {
			return HTMLRequirements{}, htmlRequirementError("html.requirement_invalid", "merged inline requirement bytes exceed the limit")
		}
		for _, dependency := range requirement.LoadAfter {
			if _, found := byID[dependency]; !found {
				return HTMLRequirements{}, htmlRequirementError("html.requirement_dependency_missing", fmt.Sprintf("requirement %q depends on missing requirement %q", requirement.ID, dependency))
			}
		}
	}

	remaining := make(map[string]HTMLRequirement, len(byID))
	for id, requirement := range byID {
		remaining[id] = requirement
	}
	emitted := make(map[string]struct{}, len(byID))
	ordered := make([]HTMLRequirement, 0, len(byID))
	for len(remaining) > 0 {
		ready := make([]string, 0, len(remaining))
		for id, requirement := range remaining {
			dependenciesReady := true
			for _, dependency := range requirement.LoadAfter {
				if _, found := emitted[dependency]; !found {
					dependenciesReady = false
					break
				}
			}
			if dependenciesReady {
				ready = append(ready, id)
			}
		}
		if len(ready) == 0 {
			return HTMLRequirements{}, htmlRequirementError("html.requirement_cycle", "requirement dependency graph contains a cycle")
		}
		sort.Strings(ready)
		id := ready[0]
		ordered = append(ordered, cloneHTMLRequirement(remaining[id]))
		delete(remaining, id)
		emitted[id] = struct{}{}
	}
	return HTMLRequirements{requirements: ordered}, nil
}

func normalizeHTMLRequirement(requirement HTMLRequirement) (HTMLRequirement, error) {
	requirement = cloneHTMLRequirement(requirement)
	if !htmlRequirementIDPattern.MatchString(requirement.ID) {
		return HTMLRequirement{}, htmlRequirementError("html.requirement_invalid", fmt.Sprintf("invalid requirement ID %q", requirement.ID))
	}
	switch requirement.Kind {
	case HTMLStylesheet, HTMLScript, HTMLRuntimeRole:
	default:
		return HTMLRequirement{}, htmlRequirementError("html.requirement_invalid", fmt.Sprintf("requirement %q has invalid kind %q", requirement.ID, requirement.Kind))
	}
	if requirement.LocalURL == "" && assetRefIsZero(requirement.Inline) {
		return HTMLRequirement{}, htmlRequirementError("html.requirement_invalid", fmt.Sprintf("requirement %q has no local URL or inline bytes", requirement.ID))
	}
	if requirement.LocalURL != "" {
		if err := validateHTMLRequirementURL(requirement.LocalURL); err != nil {
			return HTMLRequirement{}, htmlRequirementError("html.requirement_invalid", fmt.Sprintf("requirement %q URL: %v", requirement.ID, err))
		}
	}
	if requirement.Integrity != "" {
		encoded := strings.TrimPrefix(requirement.Integrity, "sha384-")
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if !strings.HasPrefix(requirement.Integrity, "sha384-") || err != nil || len(decoded) != 48 {
			return HTMLRequirement{}, htmlRequirementError("html.requirement_invalid", fmt.Sprintf("requirement %q has invalid SHA-384 integrity", requirement.ID))
		}
	}
	if !assetRefIsZero(requirement.Inline) {
		if len(requirement.Inline.Content) > maxHTMLRequirementInlineBytes {
			return HTMLRequirement{}, htmlRequirementError("html.requirement_invalid", fmt.Sprintf("requirement %q inline bytes exceed the limit", requirement.ID))
		}
		if err := requirement.Inline.validate(); err != nil {
			return HTMLRequirement{}, htmlRequirementError("html.requirement_invalid", fmt.Sprintf("requirement %q inline asset: %v", requirement.ID, err))
		}
		if requirement.Inline.MediaType == "" {
			requirement.Inline.MediaType = assetMediaType(requirement.Inline.Path)
		}
		if requirement.Inline.SHA256 == "" {
			digest := sha256.Sum256(requirement.Inline.Content)
			requirement.Inline.SHA256 = hex.EncodeToString(digest[:])
		}
	}

	sort.Strings(requirement.LoadAfter)
	for index, dependency := range requirement.LoadAfter {
		if !htmlRequirementIDPattern.MatchString(dependency) || dependency == requirement.ID {
			return HTMLRequirement{}, htmlRequirementError("html.requirement_invalid", fmt.Sprintf("requirement %q has invalid dependency %q", requirement.ID, dependency))
		}
		if index > 0 && dependency == requirement.LoadAfter[index-1] {
			return HTMLRequirement{}, htmlRequirementError("html.requirement_invalid", fmt.Sprintf("requirement %q repeats dependency %q", requirement.ID, dependency))
		}
	}
	return requirement, nil
}

func validateHTMLRequirementURL(value string) error {
	if strings.ContainsAny(value, "\\\x00\n\r") {
		return fmt.Errorf("unsafe characters")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return err
	}
	if parsed.Scheme == "https" {
		if parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
			return fmt.Errorf("absolute HTTPS URL required")
		}
		return nil
	}
	if parsed.Scheme != "" || parsed.Host != "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || parsed.RawQuery != "" || parsed.Fragment != "" || path.Clean(parsed.Path) != parsed.Path {
		return fmt.Errorf("absolute local path or HTTPS URL required")
	}
	return nil
}

func equalHTMLRequirement(left, right HTMLRequirement) bool {
	return left.ID == right.ID && left.Kind == right.Kind && left.LocalURL == right.LocalURL && left.Integrity == right.Integrity &&
		slices.Equal(left.LoadAfter, right.LoadAfter) && left.Inline.Path == right.Inline.Path && left.Inline.MediaType == right.Inline.MediaType &&
		left.Inline.SHA256 == right.Inline.SHA256 && bytes.Equal(left.Inline.Content, right.Inline.Content)
}

func htmlRequirementError(code, message string) error {
	return newDiagnosticError(Diagnostic{Code: code, Severity: SeverityError, Message: message})
}
