package site

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/araihu/margo/ssg"
)

// LayoutKind identifies one of Margo's closed configured-site layouts.
type LayoutKind string

const (
	LayoutArticle LayoutKind = "article"
	LayoutLanding LayoutKind = "landing"
	LayoutDocs    LayoutKind = "docs"
)

// LayoutConfig selects a configured-site layout and its site-owned values.
type LayoutConfig struct {
	Kind    LayoutKind     `yaml:"kind"`
	Default map[string]any `yaml:"default"`
	Values  map[string]any `yaml:"values"`
}

// LayoutPatch is one directory or Markdown contribution to the typed cascade.
type LayoutPatch struct {
	Kind   LayoutKind
	Values map[string]any
	Source string
	Base   string
}

// ResolvedLayout carries the active kind and its isolated, normalized values.
// Rendering fields are populated when configured-site renderers consume it.
type ResolvedLayout struct {
	Kind         LayoutKind
	Values       map[string]any
	Family       string
	FrameName    string
	Frame        ssg.Frame
	FrameSchema  ssg.FrameSchema
	FrameValues  ssg.Values
	SchemaHash   string
	Identity     string
	renderer     layoutRenderer
	dependencies layoutDependencies
}

type layoutRenderer uint8

const (
	layoutRenderArticle layoutRenderer = iota
	layoutRenderLanding
	layoutRenderDocs
)

func (renderer layoutRenderer) String() string {
	switch renderer {
	case layoutRenderLanding:
		return "landing"
	case layoutRenderDocs:
		return "docs"
	default:
		return "article"
	}
}

type layoutDependencies struct {
	siteStyles         bool
	landingStyles      bool
	docsStyles         bool
	docsInteractions   bool
	goshtosoNavigation bool
	pageActions        bool
}

type layoutValueType uint8

const (
	layoutObject layoutValueType = iota
	layoutBool
	layoutString
	layoutStringList
)

type layoutValueScope uint8

const (
	layoutValueSiteDefault layoutValueScope = iota
	layoutValueOverride
)

type layoutValueSchema struct {
	Type            layoutValueType
	Properties      map[string]layoutValueSchema
	Enum            []string
	SiteDefaultOnly bool
}

type layoutRegistryEntry struct {
	kind         LayoutKind
	defaults     map[string]any
	schema       layoutValueSchema
	frameName    string
	frameProfile string
	renderer     layoutRenderer
	dependencies layoutDependencies
}

type layoutRegistry struct {
	order   []LayoutKind
	entries map[LayoutKind]layoutRegistryEntry
}

func builtinLayoutRegistry() layoutRegistry {
	content := func() layoutValueSchema {
		return layoutValueSchema{
			Type: layoutObject,
			Properties: map[string]layoutValueSchema{
				"layout": {Type: layoutString, Enum: []string{"article"}},
			},
		}
	}
	articleSchema := layoutValueSchema{
		Type: layoutObject,
		Properties: map[string]layoutValueSchema{
			"content": content(),
		},
	}
	docsSchema := layoutValueSchema{
		Type: layoutObject,
		Properties: map[string]layoutValueSchema{
			"families": {Type: layoutStringList, SiteDefaultOnly: true},
			"family":   {Type: layoutString},
			"sidebar":  {Type: layoutBool},
			"toc":      {Type: layoutBool},
			"content":  content(),
		},
	}

	entries := map[LayoutKind]layoutRegistryEntry{
		LayoutArticle: {
			kind:      LayoutArticle,
			defaults:  map[string]any{"content": map[string]any{"layout": "article"}},
			schema:    articleSchema,
			frameName: "main",
			renderer:  layoutRenderArticle,
			dependencies: layoutDependencies{
				siteStyles: true,
			},
		},
		LayoutLanding: {
			kind:      LayoutLanding,
			defaults:  map[string]any{"content": map[string]any{"layout": "article"}},
			schema:    articleSchema,
			frameName: "main",
			renderer:  layoutRenderLanding,
			dependencies: layoutDependencies{
				siteStyles:    true,
				landingStyles: true,
			},
		},
		LayoutDocs: {
			kind:         LayoutDocs,
			frameName:    "top-left-main-right-footer",
			frameProfile: ssg.DocsProfile,
			renderer:     layoutRenderDocs,
			dependencies: layoutDependencies{
				siteStyles:         true,
				docsStyles:         true,
				docsInteractions:   true,
				goshtosoNavigation: true,
				pageActions:        true,
			},
			defaults: map[string]any{
				"families": []any{"default"},
				"family":   "default",
				"sidebar":  true,
				"toc":      true,
				"content":  map[string]any{"layout": "article"},
			},
			schema: docsSchema,
		},
	}
	return layoutRegistry{
		order:   []LayoutKind{LayoutArticle, LayoutLanding, LayoutDocs},
		entries: entries,
	}
}

func (registry layoutRegistry) lookup(kind LayoutKind) (layoutRegistryEntry, bool) {
	entry, ok := registry.entries[kind]
	if ok {
		entry.defaults = mergeLayoutValues(entry.defaults, nil)
		entry.schema = cloneLayoutValueSchema(entry.schema)
	}
	return entry, ok
}

func cloneLayoutValueSchema(schema layoutValueSchema) layoutValueSchema {
	cloned := schema
	cloned.Enum = append([]string(nil), schema.Enum...)
	cloned.Properties = make(map[string]layoutValueSchema, len(schema.Properties))
	for key, property := range schema.Properties {
		cloned.Properties[key] = cloneLayoutValueSchema(property)
	}
	return cloned
}

func (entry layoutRegistryEntry) validateValues(values map[string]any, scope layoutValueScope, pointer string) (map[string]any, error) {
	if values == nil {
		return map[string]any{}, nil
	}
	normalized, err := validateLayoutValue(entry.schema, values, scope, pointer)
	if err != nil {
		return nil, err
	}
	return normalized.(map[string]any), nil
}

func validateLayoutValue(schema layoutValueSchema, value any, scope layoutValueScope, pointer string) (any, error) {
	switch schema.Type {
	case layoutObject:
		object, ok := value.(map[string]any)
		if !ok {
			return nil, invalidLayoutValue(pointer, "value must be an object")
		}
		keys := make([]string, 0, len(object))
		for key := range object {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		normalized := make(map[string]any, len(object))
		for _, key := range keys {
			propertyPointer := appendLayoutPointer(pointer, key)
			property, exists := schema.Properties[key]
			if !exists {
				return nil, newPresentationDiagnostic(
					"site.layout_value_unknown",
					fmt.Sprintf("layout value %q is not supported", key),
					"Use only values published by the selected layout kind.",
					propertyPointer,
				)
			}
			if property.SiteDefaultOnly && scope != layoutValueSiteDefault {
				return nil, invalidLayoutValue(propertyPointer, fmt.Sprintf("layout value %q is site-default only", key))
			}
			child, err := validateLayoutValue(property, object[key], scope, propertyPointer)
			if err != nil {
				return nil, err
			}
			normalized[key] = child
		}
		return normalized, nil

	case layoutBool:
		normalized, ok := value.(bool)
		if !ok {
			return nil, invalidLayoutValue(pointer, "value must be a boolean")
		}
		return normalized, nil

	case layoutString:
		normalized, ok := value.(string)
		if !ok {
			return nil, invalidLayoutValue(pointer, "value must be a string")
		}
		if len(schema.Enum) != 0 && !containsLayoutEnum(schema.Enum, normalized) {
			return nil, invalidLayoutValue(pointer, fmt.Sprintf("value %q is not one of %s", normalized, strings.Join(schema.Enum, ", ")))
		}
		return normalized, nil

	case layoutStringList:
		values, ok := layoutListValues(value)
		if !ok {
			return nil, invalidLayoutValue(pointer, "value must be an array of strings")
		}
		normalized := make([]any, len(values))
		for index, item := range values {
			text, ok := item.(string)
			if !ok {
				return nil, invalidLayoutValue(appendLayoutPointer(pointer, fmt.Sprint(index)), "array element must be a string")
			}
			normalized[index] = text
		}
		return normalized, nil

	default:
		return nil, invalidLayoutValue(pointer, "layout schema has an unsupported value type")
	}
}

func layoutListValues(value any) ([]any, bool) {
	switch typed := value.(type) {
	case []any:
		return typed, true
	case []string:
		values := make([]any, len(typed))
		for index := range typed {
			values[index] = typed[index]
		}
		return values, true
	default:
		return nil, false
	}
}

func containsLayoutEnum(allowed []string, value string) bool {
	for _, candidate := range allowed {
		if candidate == value {
			return true
		}
	}
	return false
}

func invalidLayoutValue(pointer, message string) error {
	return newPresentationDiagnostic(
		"site.layout_value_invalid",
		message,
		"Use a value with the type and choices published by the selected layout kind.",
		pointer,
	)
}

func resolveSiteLayout(config LayoutConfig, source string) (layoutCascade, error) {
	registry := builtinLayoutRegistry()
	entry, ok := registry.lookup(config.Kind)
	if !ok {
		return layoutCascade{}, presentationSourceDiagnostic(newPresentationDiagnostic(
			"site.layout_unknown",
			fmt.Sprintf("unknown layout kind %q", config.Kind),
			"Choose article, landing, or docs.",
			"/layout/kind",
		), source)
	}

	builtinValues, err := entry.validateValues(entry.defaults, layoutValueSiteDefault, "/layout/default")
	if err != nil {
		return layoutCascade{}, presentationSourceDiagnostic(err, source)
	}
	defaultValues, err := entry.validateValues(config.Default, layoutValueSiteDefault, "/layout/default")
	if err != nil {
		return layoutCascade{}, presentationSourceDiagnostic(err, source)
	}
	if config.Kind == LayoutDocs {
		defaultValues, err = normalizeDocsFamilyDeclarations(defaultValues, "/layout/default/families")
		if err != nil {
			return layoutCascade{}, presentationSourceDiagnostic(err, source)
		}
	}
	overrideValues, err := entry.validateValues(config.Values, layoutValueOverride, "/layout/values")
	if err != nil {
		return layoutCascade{}, presentationSourceDiagnostic(err, source)
	}

	values := mergeLayoutValues(builtinValues, defaultValues)
	if config.Kind == LayoutDocs {
		if err := validateDocsFamilySelection(defaultValues, values, "/layout/default/family"); err != nil {
			return layoutCascade{}, presentationSourceDiagnostic(err, source)
		}
		if err := validateDocsFamilySelection(overrideValues, values, "/layout/values/family"); err != nil {
			return layoutCascade{}, presentationSourceDiagnostic(err, source)
		}
	}
	values = mergeLayoutValues(values, overrideValues)
	return layoutCascade{
		registry: registry,
		active:   config.Kind,
		buckets:  map[LayoutKind]map[string]any{config.Kind: values},
	}, nil
}

type layoutCascade struct {
	registry layoutRegistry
	active   LayoutKind
	buckets  map[LayoutKind]map[string]any
}

func (cascade layoutCascade) apply(patch LayoutPatch) (layoutCascade, error) {
	next := cascade.clone()
	base := patch.Base
	if base == "" {
		base = "/layout"
	}

	kind := next.active
	if patch.Kind != "" {
		kind = patch.Kind
	}
	entry, ok := next.registry.lookup(kind)
	if !ok {
		return layoutCascade{}, presentationSourceDiagnostic(newPresentationDiagnostic(
			"site.layout_unknown",
			fmt.Sprintf("unknown layout kind %q", kind),
			"Choose article, landing, or docs.",
			appendLayoutPointer(base, "kind"),
		), patch.Source)
	}

	values, exists := next.buckets[kind]
	if !exists {
		var err error
		values, err = entry.validateValues(entry.defaults, layoutValueSiteDefault, appendLayoutPointer(base, "values"))
		if err != nil {
			return layoutCascade{}, presentationSourceDiagnostic(err, patch.Source)
		}
	}
	normalized, err := entry.validateValues(patch.Values, layoutValueOverride, appendLayoutPointer(base, "values"))
	if err != nil {
		return layoutCascade{}, presentationSourceDiagnostic(err, patch.Source)
	}
	if kind == LayoutDocs {
		if err := validateDocsFamilySelection(normalized, values, appendLayoutPointer(appendLayoutPointer(base, "values"), "family")); err != nil {
			return layoutCascade{}, presentationSourceDiagnostic(err, patch.Source)
		}
	}
	next.active = kind
	next.buckets[kind] = mergeLayoutValues(values, normalized)
	return next, nil
}

func normalizeDocsFamilyDeclarations(values map[string]any, pointer string) (map[string]any, error) {
	raw, configured := values["families"]
	if !configured {
		return values, nil
	}
	families, _ := layoutListValues(raw)
	seen := make(map[string]int, len(families))
	normalized := []any{"default"}
	for index, value := range families {
		family := strings.TrimSpace(value.(string))
		itemPointer := appendLayoutPointer(pointer, fmt.Sprint(index))
		if family == "" {
			return nil, newPresentationDiagnostic(
				"site.family_invalid",
				"family identifier must not be empty",
				"Declare a stable non-empty family identifier.",
				itemPointer,
			)
		}
		if previous, exists := seen[family]; exists {
			return nil, newPresentationDiagnostic(
				"site.family_duplicate",
				fmt.Sprintf("family %q is declared more than once (entries %d and %d)", family, previous, index),
				"Declare each family once.",
				itemPointer,
			)
		}
		seen[family] = index
		if family != "default" {
			normalized = append(normalized, family)
		}
	}
	values["families"] = normalized
	return values, nil
}

func validateDocsFamilySelection(selection, declarations map[string]any, pointer string) error {
	raw, explicit := selection["family"]
	if !explicit {
		return nil
	}
	family := strings.TrimSpace(raw.(string))
	declared, _ := layoutListValues(declarations["families"])
	for _, candidate := range declared {
		if candidate == family {
			selection["family"] = family
			return nil
		}
	}
	return newPresentationDiagnostic(
		"site.family_undeclared",
		fmt.Sprintf("docs family %q is not declared", family),
		"Declare the family in layout.default.families before selecting it.",
		pointer,
	)
}

func (cascade layoutCascade) clone() layoutCascade {
	buckets := make(map[LayoutKind]map[string]any, len(cascade.buckets))
	for kind, values := range cascade.buckets {
		buckets[kind] = mergeLayoutValues(values, nil)
	}
	return layoutCascade{registry: cascade.registry, active: cascade.active, buckets: buckets}
}

func (cascade layoutCascade) resolved() ResolvedLayout {
	values := mergeLayoutValues(cascade.buckets[cascade.active], nil)
	identity, _ := layoutValuesIdentity(cascade.active, values)
	family := ""
	if cascade.active == LayoutDocs {
		family, _ = values["family"].(string)
	}
	return ResolvedLayout{Kind: cascade.active, Values: values, Family: family, Identity: identity}
}

// mergeLayoutValues recursively merges objects. Scalars replace and arrays
// replace as whole values. The result shares no maps or arrays with its inputs.
func mergeLayoutValues(base, patch map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(patch))
	for key, value := range base {
		merged[key] = cloneLayoutValue(value)
	}
	for key, value := range patch {
		if patchMap, ok := value.(map[string]any); ok {
			if baseMap, ok := merged[key].(map[string]any); ok {
				merged[key] = mergeLayoutValues(baseMap, patchMap)
				continue
			}
		}
		merged[key] = cloneLayoutValue(value)
	}
	return merged
}

func cloneLayoutValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return mergeLayoutValues(typed, nil)
	case []any:
		values := make([]any, len(typed))
		for index := range typed {
			values[index] = cloneLayoutValue(typed[index])
		}
		return values
	case []string:
		values := make([]any, len(typed))
		for index := range typed {
			values[index] = typed[index]
		}
		return values
	default:
		return typed
	}
}

func layoutValuesIdentity(kind LayoutKind, values map[string]any) (string, error) {
	var canonical bytes.Buffer
	if err := writeCanonicalLayoutValue(&canonical, values); err != nil {
		return "", fmt.Errorf("site.layout_identity: %w", err)
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte("margo.site.layout-values/v1\x00"))
	_, _ = hash.Write([]byte(kind))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(canonical.Bytes())
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writeCanonicalLayoutValue(output *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		output.WriteString("null")
	case bool:
		if typed {
			output.WriteString("true")
		} else {
			output.WriteString("false")
		}
	case string:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return err
		}
		output.Write(encoded)
	case []string:
		values := make([]any, len(typed))
		for index := range typed {
			values[index] = typed[index]
		}
		return writeCanonicalLayoutValue(output, values)
	case []any:
		output.WriteByte('[')
		for index := range typed {
			if index != 0 {
				output.WriteByte(',')
			}
			if err := writeCanonicalLayoutValue(output, typed[index]); err != nil {
				return err
			}
		}
		output.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		output.WriteByte('{')
		for index, key := range keys {
			if index != 0 {
				output.WriteByte(',')
			}
			encodedKey, err := json.Marshal(key)
			if err != nil {
				return err
			}
			output.Write(encodedKey)
			output.WriteByte(':')
			if err := writeCanonicalLayoutValue(output, typed[key]); err != nil {
				return err
			}
		}
		output.WriteByte('}')
	default:
		return fmt.Errorf("unsupported canonical value type %T", value)
	}
	return nil
}

func appendLayoutPointer(base, segment string) string {
	base = strings.TrimSuffix(base, "/")
	segment = strings.ReplaceAll(segment, "~", "~0")
	segment = strings.ReplaceAll(segment, "/", "~1")
	return base + "/" + segment
}
