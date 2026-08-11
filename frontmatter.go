package margo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

type frontmatterResult struct {
	values     map[string]any
	margo      map[string]any
	body       []byte
	bodyOffset int
	bodyLines  int
}

func parseFrontmatter(source Source) (frontmatterResult, error) {
	result := frontmatterResult{values: make(map[string]any), body: append([]byte(nil), source.Content...)}
	lines := bytes.SplitAfter(source.Content, []byte("\n"))
	if len(lines) == 0 || strings.TrimSpace(string(lines[0])) != "---" {
		return result, nil
	}
	closeIndex := -1
	for i := 1; i < len(lines); i++ {
		marker := strings.TrimSpace(string(bytes.TrimSuffix(lines[i], []byte("\n"))))
		if marker == "---" || marker == "..." {
			closeIndex = i
			break
		}
	}
	if closeIndex < 0 {
		return frontmatterResult{}, diagnosticAt("frontmatter.unclosed", source.Name, "", "frontmatter delimiter is not closed", 1, 1)
	}
	var node yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(bytes.Join(lines[1:closeIndex], nil)))
	if err := decoder.Decode(&node); err != nil {
		return frontmatterResult{}, diagnosticAt("frontmatter.invalid", source.Name, "", err.Error(), 1, 1)
	}
	if err := validateYAMLNode(&node, 0, new(int)); err != nil {
		return frontmatterResult{}, diagnosticAt("frontmatter.limits", source.Name, "", err.Error(), 1, 1)
	}
	if err := rejectLegacyGoshtoso(source.Name, &node); err != nil {
		return frontmatterResult{}, err
	}
	if err := yaml.Unmarshal(bytes.Join(lines[1:closeIndex], nil), &result.values); err != nil {
		return frontmatterResult{}, diagnosticAt("frontmatter.invalid", source.Name, "", err.Error(), 1, 1)
	}
	serialized, err := json.Marshal(result.values)
	if err != nil {
		return frontmatterResult{}, diagnosticAt("frontmatter.schema_invalid", source.Name, "", err.Error(), 1, 1)
	}
	if _, err := validateJSONSchema(SchemaDocument, serialized); err != nil {
		pointer, line, column := frontmatterSchemaLocation(&node, err)
		return frontmatterResult{}, diagnosticAt("frontmatter.schema_invalid", source.Name, pointer, err.Error(), line, column)
	}
	if value, ok := result.values["margo"].(map[string]any); ok {
		result.margo = cloneStringAnyMap(value)
	}
	result.bodyOffset = 0
	for _, line := range lines[:closeIndex+1] {
		result.bodyOffset += len(line)
	}
	result.body = bytes.Join(lines[closeIndex+1:], nil)
	result.bodyLines = closeIndex + 1
	return result, nil
}

func rejectLegacyGoshtoso(source string, root *yaml.Node) error {
	mapping := root
	if root.Kind == yaml.DocumentNode && len(root.Content) == 1 {
		mapping = root.Content[0]
	}
	if mapping.Kind == 0 {
		return nil
	}
	if mapping.Kind != yaml.MappingNode {
		return diagnosticAt("frontmatter.mapping_required", source, "", "frontmatter must be a mapping", mapping.Line, mapping.Column)
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := mapping.Content[i]
		if key.Value != "goshtoso" {
			continue
		}
		return diagnosticAt("frontmatter.goshtoso_removed", source, "/goshtoso", "goshtoso frontmatter was removed: move page preferences to margo.page; move security, theme, and brand controls to trusted host/render configuration; remove tables and Mermaid configuration", key.Line, key.Column)
	}
	return nil
}

func validateYAMLNode(node *yaml.Node, depth int, aliases *int) error {
	if depth > 32 {
		return fmt.Errorf("YAML nesting exceeds 32 levels")
	}
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode {
		*aliases = *aliases + 1
		if *aliases > 64 {
			return fmt.Errorf("YAML aliases exceed 64")
		}
	}
	if node.Kind == yaml.MappingNode {
		seen := make(map[string]struct{}, len(node.Content)/2)
		for index := 0; index+1 < len(node.Content); index += 2 {
			key := node.Content[index]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
				return fmt.Errorf("YAML mapping keys must be strings")
			}
			if _, duplicate := seen[key.Value]; duplicate {
				return fmt.Errorf("duplicate YAML property %q", key.Value)
			}
			seen[key.Value] = struct{}{}
		}
	}
	for _, child := range node.Content {
		if err := validateYAMLNode(child, depth+1, aliases); err != nil {
			return err
		}
	}
	return nil
}

func frontmatterSchemaLocation(root *yaml.Node, failure error) (string, int, int) {
	var validation *jsonschema.ValidationError
	if !errors.As(failure, &validation) {
		return "", 1, 1
	}
	for len(validation.Causes) > 0 {
		validation = validation.Causes[0]
	}
	pointer := ""
	node := root
	if node.Kind == yaml.DocumentNode && len(node.Content) == 1 {
		node = node.Content[0]
	}
	for _, segment := range validation.InstanceLocation {
		pointer += "/" + strings.ReplaceAll(strings.ReplaceAll(segment, "~", "~0"), "/", "~1")
		if node.Kind == yaml.MappingNode {
			for index := 0; index+1 < len(node.Content); index += 2 {
				if node.Content[index].Value == segment {
					node = node.Content[index+1]
					break
				}
			}
		} else if node.Kind == yaml.SequenceNode {
			index, err := strconv.Atoi(segment)
			if err == nil && index >= 0 && index < len(node.Content) {
				node = node.Content[index]
			}
		}
	}
	line, column := node.Line, node.Column
	if line < 1 {
		line, column = 1, 1
	}
	return pointer, line + 1, column
}

func cloneStringAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneOptionValue(value)
	}
	return out
}
