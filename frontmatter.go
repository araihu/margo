package margo

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type frontmatterResult struct {
	values   map[string]any
	goshtoso map[string]any
	body     []byte
}

var allowedGoshtosoFields = map[string]map[string]struct{}{
	"": {
		"theme": {}, "security": {}, "tables": {}, "page": {}, "brand": {},
	},
	"security": {"rawHTML": {}, "mermaid": {}},
	"tables":   {"sort": {}},
	"page":     {"size": {}, "orientation": {}},
	"brand":    {"logo": {}, "watermark": {}},
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
	if err := validateYAMLNode(&node, 0, 0); err != nil {
		return frontmatterResult{}, diagnosticAt("frontmatter.limits", source.Name, "", err.Error(), 1, 1)
	}
	if err := validateGoshtosoNode(source.Name, &node); err != nil {
		return frontmatterResult{}, err
	}
	if err := yaml.Unmarshal(bytes.Join(lines[1:closeIndex], nil), &result.values); err != nil {
		return frontmatterResult{}, diagnosticAt("frontmatter.invalid", source.Name, "", err.Error(), 1, 1)
	}
	if value, ok := result.values["goshtoso"].(map[string]any); ok {
		result.goshtoso = cloneStringAnyMap(value)
	}
	result.body = bytes.Join(lines[closeIndex+1:], nil)
	return result, nil
}

func validateGoshtosoNode(source string, root *yaml.Node) error {
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
		key, value := mapping.Content[i], mapping.Content[i+1]
		if key.Value != "goshtoso" {
			continue
		}
		if value.Kind != yaml.MappingNode {
			return diagnosticAt("frontmatter.goshtoso.mapping_required", source, "/goshtoso", "goshtoso must be a mapping", value.Line, value.Column)
		}
		if err := validateGoshtosoMapping(source, value, "", 0); err != nil {
			return err
		}
	}
	return nil
}

func validateGoshtosoMapping(source string, node *yaml.Node, parent string, depth int) error {
	allowed := allowedGoshtosoFields[parent]
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if _, ok := allowed[key.Value]; !ok {
			pointer := "/goshtoso"
			if parent != "" {
				pointer += "/" + parent
			}
			pointer += "/" + key.Value
			return diagnosticAt("frontmatter.goshtoso.unknown_field", source, pointer, fmt.Sprintf("unknown goshtoso field %q", key.Value), key.Line, key.Column)
		}
		if nestedAllowed, ok := allowedGoshtosoFields[key.Value]; ok && len(nestedAllowed) > 0 {
			if value.Kind != yaml.MappingNode {
				return diagnosticAt("frontmatter.goshtoso.mapping_required", source, "/goshtoso/"+key.Value, "field must be a mapping", value.Line, value.Column)
			}
			if err := validateGoshtosoMapping(source, value, key.Value, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateYAMLNode(node *yaml.Node, depth, aliases int) error {
	if depth > 32 {
		return fmt.Errorf("YAML nesting exceeds 32 levels")
	}
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode {
		aliases++
		if aliases > 64 {
			return fmt.Errorf("YAML aliases exceed 64")
		}
	}
	for _, child := range node.Content {
		if err := validateYAMLNode(child, depth+1, aliases); err != nil {
			return err
		}
	}
	return nil
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
