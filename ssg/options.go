package ssg

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/araihu/margo/internal/canonicaljson"
)

var cssLengthPattern = regexp.MustCompile(`^(?:0|(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:px|rem|em|ch|vi|vb|vw|vh|vmin|vmax|%)?)$`)

// ValidateFrameOptions checks the descriptor catalog exposed by a frame.
func ValidateFrameOptions(options []FrameOptionDescriptor) error {
	seen := make(map[string]struct{}, len(options))
	for index, option := range options {
		if err := validateOptionPath(option.Path); err != nil {
			return fmt.Errorf("ssg.option_path: option %d: %w", index, err)
		}
		if _, exists := seen[option.Path]; exists {
			return fmt.Errorf("ssg.option_duplicate: option %q is declared more than once", option.Path)
		}
		seen[option.Path] = struct{}{}
		if strings.TrimSpace(option.Description) == "" {
			return fmt.Errorf("ssg.option_description: option %q needs a description", option.Path)
		}
		switch option.Type {
		case "boolean", "enum", "length", "breakpoint", "number", "string":
		default:
			return fmt.Errorf("ssg.option_type: option %q has unsupported type %q", option.Path, option.Type)
		}
		if option.Type != "enum" && option.Type != "breakpoint" && len(option.Allowed) != 0 {
			return fmt.Errorf("ssg.option_allowed: option %q only enum and breakpoint options may declare Allowed values", option.Path)
		}
		allowed := make(map[string]struct{}, len(option.Allowed))
		for _, value := range option.Allowed {
			if value == "" {
				return fmt.Errorf("ssg.option_allowed: option %q contains an empty allowed value", option.Path)
			}
			if _, exists := allowed[value]; exists {
				return fmt.Errorf("ssg.option_allowed: option %q repeats allowed value %q", option.Path, value)
			}
			allowed[value] = struct{}{}
		}
		if option.Type == "enum" || option.Type == "breakpoint" {
			if len(option.Allowed) == 0 {
				return fmt.Errorf("ssg.option_allowed: option %q needs allowed values", option.Path)
			}
		}
		if option.Min != nil && math.IsNaN(*option.Min) || option.Max != nil && math.IsNaN(*option.Max) {
			return fmt.Errorf("ssg.option_range: option %q has a non-finite range", option.Path)
		}
		if option.Min != nil && option.Max != nil && *option.Min > *option.Max {
			return fmt.Errorf("ssg.option_range: option %q has Min greater than Max", option.Path)
		}
		if option.Default != nil {
			if _, err := normalizeOptionValue(option, option.Default); err != nil {
				return fmt.Errorf("ssg.option_default: option %q: %w", option.Path, err)
			}
		}
	}
	return nil
}

// ResolveFrameValues fills declared defaults and rejects unknown or malformed
// structural values. The returned tree contains only declared option paths.
func ResolveFrameValues(schema FrameSchema, input Values) (Values, error) {
	if err := ValidateFrameOptions(schema.Options); err != nil {
		return nil, err
	}
	options := make(map[string]FrameOptionDescriptor, len(schema.Options))
	for _, option := range schema.Options {
		options[option.Path] = option
	}
	flat := make(map[string]any)
	if err := flattenValues("", input, flat); err != nil {
		return nil, err
	}
	for path := range flat {
		if _, exists := options[path]; !exists {
			return nil, fmt.Errorf("ssg.option_unknown: structural value %q is not declared by the frame", path)
		}
	}
	resolved := Values{}
	for _, option := range schema.Options {
		value, exists := flat[option.Path]
		if !exists {
			if option.Default == nil {
				return nil, fmt.Errorf("ssg.option_required: structural value %q has no value or default", option.Path)
			}
			value = option.Default
		}
		normalized, err := normalizeOptionValue(option, value)
		if err != nil {
			return nil, fmt.Errorf("ssg.option_value: %s: %w", option.Path, err)
		}
		if err := setValuePath(resolved, option.Path, normalized); err != nil {
			return nil, err
		}
	}
	if len(resolved) == 0 {
		return nil, nil
	}
	return resolved, nil
}

// SchemaHashForValues binds normalized structural values to the layout
// schema. SchemaHash is the equivalent call with omitted values/defaults.
func SchemaHashForValues(input FrameSchema, values Values) (string, error) {
	normalized, err := NormalizeSchema(input)
	if err != nil {
		return "", err
	}
	if err := ValidateFrameSchema(normalized, ""); err != nil {
		return "", err
	}
	resolved, err := ResolveFrameValues(normalized, values)
	if err != nil {
		return "", err
	}
	encoded, err := canonicaljson.Marshal(struct {
		Schema FrameSchema `json:"schema"`
		Values Values      `json:"values,omitempty"`
	}{normalized, resolved})
	if err != nil {
		return "", fmt.Errorf("ssg.schema_hash: %w", err)
	}
	return hashBytes("margo.ssg.layout-schema/v1", encoded), nil
}

func validateOptionPath(value string) error {
	if value == "" {
		return fmt.Errorf("path is empty")
	}
	for _, segment := range strings.Split(value, ".") {
		if segment == "" || strings.ContainsAny(segment, " \t\r\n\x00") {
			return fmt.Errorf("path %q is not a dotted structural path", value)
		}
	}
	return nil
}

func flattenValues(prefix string, value any, output map[string]any) error {
	switch typed := value.(type) {
	case nil:
		if prefix == "" {
			return nil
		}
		output[prefix] = nil
	case Values:
		for key, child := range typed {
			if err := flattenValues(joinValuePath(prefix, key), child, output); err != nil {
				return err
			}
		}
	case map[string]any:
		for key, child := range typed {
			if err := flattenValues(joinValuePath(prefix, key), child, output); err != nil {
				return err
			}
		}
	default:
		if prefix == "" {
			return fmt.Errorf("ssg.option_values: root values must be an object")
		}
		output[prefix] = value
	}
	return nil
}

func joinValuePath(prefix, segment string) string {
	if prefix == "" {
		return segment
	}
	return prefix + "." + segment
}

func setValuePath(values Values, path string, value any) error {
	segments := strings.Split(path, ".")
	current := values
	for _, segment := range segments[:len(segments)-1] {
		child, exists := current[segment]
		if !exists {
			nested := Values{}
			current[segment] = nested
			current = nested
			continue
		}
		nested, ok := child.(Values)
		if !ok {
			return fmt.Errorf("ssg.option_path_collision: %q is both a value and an object", path)
		}
		current = nested
	}
	current[segments[len(segments)-1]] = value
	return nil
}

func normalizeOptionValue(option FrameOptionDescriptor, value any) (any, error) {
	switch option.Type {
	case "boolean":
		if typed, ok := value.(bool); ok {
			return typed, nil
		}
		return nil, fmt.Errorf("expected boolean, got %T", value)
	case "enum", "breakpoint", "string":
		typed, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}
		if typed == "top" && containsString(option.Allowed, "block-start") {
			typed = "block-start"
		}
		if typed == "bottom" && containsString(option.Allowed, "block-end") {
			typed = "block-end"
		}
		if (option.Type == "enum" || option.Type == "breakpoint") && !containsString(option.Allowed, typed) {
			return nil, fmt.Errorf("value %q is not allowed", typed)
		}
		if option.Type == "string" && typed == "" {
			return nil, fmt.Errorf("value must not be empty")
		}
		return typed, nil
	case "length":
		if typed, ok := numericValue(value); ok {
			if err := validateNumberRange(option, typed); err != nil {
				return nil, err
			}
			return typed, nil
		}
		typed, ok := value.(string)
		if !ok || !cssLengthPattern.MatchString(typed) && !strings.HasPrefix(typed, "token://") {
			return nil, fmt.Errorf("expected bounded CSS length or token:// reference, got %T", value)
		}
		return typed, nil
	case "number":
		typed, ok := numericValue(value)
		if !ok {
			return nil, fmt.Errorf("expected finite number, got %T", value)
		}
		if err := validateNumberRange(option, typed); err != nil {
			return nil, err
		}
		return typed, nil
	default:
		return nil, fmt.Errorf("unsupported type %q", option.Type)
	}
}

func numericValue(value any) (float64, bool) {
	var number float64
	switch typed := value.(type) {
	case int:
		number = float64(typed)
	case int8:
		number = float64(typed)
	case int16:
		number = float64(typed)
	case int32:
		number = float64(typed)
	case int64:
		number = float64(typed)
	case uint:
		number = float64(typed)
	case uint8:
		number = float64(typed)
	case uint16:
		number = float64(typed)
	case uint32:
		number = float64(typed)
	case uint64:
		number = float64(typed)
	case float32:
		number = float64(typed)
	case float64:
		number = typed
	default:
		return 0, false
	}
	return number, !math.IsNaN(number) && !math.IsInf(number, 0)
}

func validateNumberRange(option FrameOptionDescriptor, value float64) error {
	if option.Min != nil && value < *option.Min {
		return fmt.Errorf("value %v is below minimum %v", value, *option.Min)
	}
	if option.Max != nil && value > *option.Max {
		return fmt.Errorf("value %v is above maximum %v", value, *option.Max)
	}
	return nil
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func valueAt(values Values, path ...string) (any, bool) {
	var current any = values
	for _, segment := range path {
		switch typed := current.(type) {
		case Values:
			var exists bool
			current, exists = typed[segment]
			if !exists {
				return nil, false
			}
		case map[string]any:
			var exists bool
			current, exists = typed[segment]
			if !exists {
				return nil, false
			}
		default:
			return nil, false
		}
	}
	return current, true
}

func hashBytes(domain string, payload []byte) string {
	return payloadDigest(domain, payload)
}
