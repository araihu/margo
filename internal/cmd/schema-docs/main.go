package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type row struct {
	Path, Description, Type, Required, Default, Allowed, Limits, Targets, Precedence, Effect, Examples, Interactions string
}

func main() {
	for _, item := range []struct {
		schema string
		output string
		title  string
	}{
		{"schema/v1/policy.json", "docs/reference/policy.md", "Host policy reference"},
		{"schema/v1/document.json", "docs/reference/document-metadata.md", "Document metadata reference"},
	} {
		data, err := os.ReadFile(item.schema)
		must(err)
		var schema map[string]any
		must(json.Unmarshal(data, &schema))
		rows := collectRows(schema)
		var output bytes.Buffer
		fmt.Fprintf(&output, "# %s\n\n", item.title)
		fmt.Fprintf(&output, "Generated from [`%s`](../../%s). Do not edit by hand.\n\n", filepath.Base(item.schema), item.schema)
		fmt.Fprintln(&output, "| Path | Description | Type | Required | Default or fallback | Allowed values | Limits | Targets | Precedence | Security or privacy effect | Examples | Interactions |")
		fmt.Fprintln(&output, "| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |")
		for _, row := range rows {
			fmt.Fprintf(&output, "| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
				cell(row.Path), cell(row.Description), cell(row.Type), cell(row.Required), cell(row.Default), cell(row.Allowed),
				cell(row.Limits), cell(row.Targets), cell(row.Precedence), cell(row.Effect), cell(row.Examples), cell(row.Interactions))
		}
		must(os.MkdirAll(filepath.Dir(item.output), 0o755))
		must(os.WriteFile(item.output, output.Bytes(), 0o644))
	}
}

func collectRows(root map[string]any) []row {
	var rows []row
	var walk func(map[string]any, string, map[string]any)
	walk = func(schema map[string]any, path string, inherited map[string]any) {
		properties, _ := schema["properties"].(map[string]any)
		required := stringSet(schema["required"])
		keys := make([]string, 0, len(properties))
		for key := range properties {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			property, _ := properties[key].(map[string]any)
			resolved := resolve(root, property)
			merged := merge(inheritedAnnotations(inherited), resolved, property)
			propertyPath := path + "/" + key
			rows = append(rows, makeRow(propertyPath, merged, required[key]))
			walk(merged, propertyPath, merged)
		}
	}
	walk(root, "", map[string]any{})
	return rows
}

func makeRow(path string, schema map[string]any, required bool) row {
	defaultValue := "built-in or conceptual fallback; see description"
	if value, ok := schema["default"]; ok {
		defaultValue = jsonText(value)
	} else if required {
		defaultValue = "none"
	}
	allowed := "any value of declared type"
	if value, ok := schema["const"]; ok {
		allowed = jsonText(value)
	} else if value, ok := schema["enum"]; ok {
		allowed = jsonText(value)
	}
	limits := make([]string, 0)
	for _, key := range []string{"minimum", "maximum", "minLength", "maxLength", "minItems", "maxItems", "pattern", "format", "x-margo-maxUtf8Bytes"} {
		if value, ok := schema[key]; ok {
			limits = append(limits, key+"="+jsonText(value))
		}
	}
	if unique, _ := schema["uniqueItems"].(bool); unique {
		limits = append(limits, "uniqueItems=true")
	}
	typeName, _ := schema["type"].(string)
	if typeName == "" {
		typeName = "value"
	}
	requiredText := "optional"
	if required {
		requiredText = "required"
	}
	return row{
		Path: path, Description: textValue(schema["description"]), Type: typeName, Required: requiredText, Default: defaultValue,
		Allowed: allowed, Limits: strings.Join(limits, "; "),
		Targets:      fallback(jsonText(schema["x-margo-targets"]), "not target-specific"),
		Precedence:   fallback(textValue(schema["x-margo-precedence"]), "not applicable"),
		Effect:       fallback(textValue(schema["x-margo-security"]), "no capability or privacy effect"),
		Examples:     fallback(jsonText(schema["examples"]), "see enclosing schema examples"),
		Interactions: fallback(textValue(schema["x-margo-interactions"]), "none"),
	}
}

func fallback(value, alternative string) string {
	if value == "" {
		return alternative
	}
	return value
}

func resolve(root, schema map[string]any) map[string]any {
	reference, _ := schema["$ref"].(string)
	if !strings.HasPrefix(reference, "#/") {
		return schema
	}
	var current any = root
	for _, segment := range strings.Split(strings.TrimPrefix(reference, "#/"), "/") {
		mapping, _ := current.(map[string]any)
		current = mapping[segment]
	}
	resolved, _ := current.(map[string]any)
	return resolved
}

func merge(values ...map[string]any) map[string]any {
	result := make(map[string]any)
	for _, values := range values {
		for key, value := range values {
			if _, exists := result[key]; !exists || value != nil {
				result[key] = value
			}
		}
	}
	return result
}

func inheritedAnnotations(values map[string]any) map[string]any {
	result := make(map[string]any)
	for key, value := range values {
		if strings.HasPrefix(key, "x-margo-") {
			result[key] = value
		}
	}
	return result
}

func stringSet(value any) map[string]bool {
	result := make(map[string]bool)
	items, _ := value.([]any)
	for _, item := range items {
		if text, ok := item.(string); ok {
			result[text] = true
		}
	}
	return result
}

func jsonText(value any) string {
	if value == nil {
		return ""
	}
	data, _ := json.Marshal(value)
	return string(data)
}

func textValue(value any) string {
	text, _ := value.(string)
	return text
}

func cell(value string) string {
	if value == "" {
		return "—"
	}
	return strings.ReplaceAll(strings.ReplaceAll(value, "|", "\\|"), "\n", " ")
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
