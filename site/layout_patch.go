package site

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"

	"gopkg.in/yaml.v3"
)

const directoryLayoutPatchName = "_layout.yaml"

type configuredInputs struct {
	Sources []Source
	Patches []LayoutPatch
}

func decodeDirectoryLayoutPatch(source string, data []byte) (LayoutPatch, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		if errors.Is(err, io.EOF) {
			return LayoutPatch{}, invalidDirectoryLayoutPatch(source, "", "layout patch is empty")
		}
		return LayoutPatch{}, invalidDirectoryLayoutPatch(source, "", err.Error())
	}

	var trailing yaml.Node
	if err := decoder.Decode(&trailing); err == nil {
		return LayoutPatch{}, invalidDirectoryLayoutPatch(source, "", "layout patch contains more than one YAML document")
	} else if !errors.Is(err, io.EOF) {
		return LayoutPatch{}, invalidDirectoryLayoutPatch(source, "", err.Error())
	}

	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return LayoutPatch{}, invalidDirectoryLayoutPatch(source, "", "layout patch root must be a mapping")
	}
	root := document.Content[0]
	if pointer, ok := duplicateYAMLKeyPointer(root, ""); ok {
		return LayoutPatch{}, invalidDirectoryLayoutPatch(source, pointer, "layout patch contains a duplicate YAML key")
	}

	patch := LayoutPatch{Source: source, Base: "/"}
	kindPresent := false
	for index := 0; index < len(root.Content); index += 2 {
		key := root.Content[index]
		value := root.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
			return LayoutPatch{}, invalidDirectoryLayoutPatch(source, "", "layout patch property names must be strings")
		}
		pointer := appendLayoutPointer("", key.Value)
		switch key.Value {
		case "kind":
			kindPresent = true
			if err := value.Decode(&patch.Kind); err != nil {
				return LayoutPatch{}, invalidDirectoryLayoutPatch(source, pointer, err.Error())
			}
		case "values":
			if value.Kind != yaml.MappingNode {
				return LayoutPatch{}, invalidDirectoryLayoutPatch(source, pointer, "layout patch values must be a mapping")
			}
			if err := value.Decode(&patch.Values); err != nil {
				return LayoutPatch{}, invalidDirectoryLayoutPatch(source, pointer, err.Error())
			}
		case "default":
			return LayoutPatch{}, invalidDirectoryLayoutPatch(source, pointer, "default is allowed only in site layout configuration")
		default:
			return LayoutPatch{}, invalidDirectoryLayoutPatch(source, pointer, fmt.Sprintf("unknown layout patch property %q", key.Value))
		}
	}

	if !kindPresent {
		return patch, nil
	}
	entry, ok := builtinLayoutRegistry().lookup(patch.Kind)
	if !ok {
		return LayoutPatch{}, presentationSourceDiagnostic(newPresentationDiagnostic(
			"site.layout_unknown",
			fmt.Sprintf("unknown layout kind %q", patch.Kind),
			"Choose article, landing, or docs.",
			"/kind",
		), source)
	}
	if patch.Values != nil {
		values, err := entry.validateValues(patch.Values, layoutValueOverride, "/values")
		if err != nil {
			return LayoutPatch{}, presentationSourceDiagnostic(err, source)
		}
		patch.Values = values
	}
	return patch, nil
}

func invalidDirectoryLayoutPatch(source, pointer, message string) error {
	return presentationSourceDiagnostic(newPresentationDiagnostic(
		"site.layout_patch_invalid",
		message,
		"Use one YAML mapping containing only kind and values.",
		pointer,
	), source)
}

func duplicateYAMLKeyPointer(node *yaml.Node, pointer string) (string, bool) {
	switch node.Kind {
	case yaml.MappingNode:
		seen := make(map[string]struct{}, len(node.Content)/2)
		for index := 0; index < len(node.Content); index += 2 {
			key := node.Content[index]
			value := node.Content[index+1]
			childPointer := pointer
			if key.Kind == yaml.ScalarNode {
				childPointer = appendLayoutPointer(pointer, key.Value)
				if _, exists := seen[key.Value]; exists {
					return childPointer, true
				}
				seen[key.Value] = struct{}{}
			}
			if duplicate, ok := duplicateYAMLKeyPointer(value, childPointer); ok {
				return duplicate, true
			}
		}
	case yaml.SequenceNode:
		for index, child := range node.Content {
			childPointer := appendLayoutPointer(pointer, fmt.Sprint(index))
			if duplicate, ok := duplicateYAMLKeyPointer(child, childPointer); ok {
				return duplicate, true
			}
		}
	}
	return "", false
}

func layoutPatchChain(sourcePath string, patches []LayoutPatch) []LayoutPatch {
	directory := path.Dir(path.Clean(sourcePath))
	ancestors := []string{"."}
	if directory != "." {
		current := ""
		for _, segment := range splitLayoutPatchDirectory(directory) {
			current = path.Join(current, segment)
			ancestors = append(ancestors, current)
		}
	}

	byDirectory := make(map[string]LayoutPatch, len(patches))
	for _, patch := range patches {
		byDirectory[path.Dir(patch.Source)] = patch
	}
	chain := make([]LayoutPatch, 0, len(ancestors))
	for _, ancestor := range ancestors {
		if patch, ok := byDirectory[ancestor]; ok {
			chain = append(chain, patch)
		}
	}
	return chain
}

func splitLayoutPatchDirectory(directory string) []string {
	if directory == "." || directory == "" {
		return nil
	}
	var segments []string
	for directory != "." && directory != "/" {
		segments = append([]string{path.Base(directory)}, segments...)
		directory = path.Dir(directory)
	}
	return segments
}
