package charts

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/a-h/templ"
	margo "github.com/araihu/margo"
	"gopkg.in/yaml.v3"
)

type familyHandler func(margo.RenderContext, any) (templ.Component, error)

type chartSession struct {
	context  margo.RenderContext
	handlers map[string]familyHandler
}

var familyRegistry = struct {
	sync.RWMutex
	items map[string]familyHandler
}{items: make(map[string]familyHandler)}

// Extension registers the reserved goshtosochart fence for optional use by a
// host compiler. Root remains independent until the host opts into this seam.
func Extension() margo.ExtensionRegistration {
	return margo.ExtensionRegistration{
		Identity: margo.ExtensionIdentity{
			Name:         "margo-charts",
			Version:      "v0.0.1-dev",
			Capabilities: []string{"static-svg", "accessible-data"},
		},
		Fences:  []string{"goshtosochart"},
		Factory: extensionFactory,
	}
}

func extensionFactory(rc margo.RenderContext) (margo.ExtensionSession, error) {
	familyRegistry.RLock()
	handlers := make(map[string]familyHandler, len(familyRegistry.items))
	for name, handler := range familyRegistry.items {
		handlers[name] = handler
	}
	familyRegistry.RUnlock()
	return chartSession{context: rc, handlers: handlers}, nil
}

func registerFamilyHandler(name string, handler familyHandler) {
	familyRegistry.Lock()
	defer familyRegistry.Unlock()
	familyRegistry.items[name] = handler
}

func (s chartSession) Render(ctx context.Context, node margo.ExtensionNode, out io.Writer) error {
	envelope, err := decodeEnvelope(node.Payload)
	if err != nil {
		return err
	}
	handler, ok := s.handlers[envelope.Type]
	if !ok {
		return chartDiagnostic("chart.type_unsupported", fmt.Sprintf("chart type %q is unsupported", envelope.Type))
	}
	component, err := handler(s.context, envelope.Model)
	if err != nil {
		return err
	}
	if component == nil {
		return chartDiagnostic("chart.component_invalid", "chart handler returned a nil component")
	}
	return component.Render(ctx, out)
}

type decodedEnvelope struct {
	Type  string
	Model any
}

func decodeEnvelope(payload []byte) (decodedEnvelope, error) {
	node, err := decodeDocument(payload)
	if err != nil {
		return decodedEnvelope{}, chartDiagnostic("chart.syntax_invalid", err.Error())
	}
	fields, err := mappingFields(node)
	if err != nil {
		return decodedEnvelope{}, chartDiagnosticAt("chart.schema_invalid", err.Error(), node.Line, node.Column)
	}
	versionNode, ok := fields["schemaVersion"]
	if !ok {
		return decodedEnvelope{}, chartDiagnosticAt("chart.schema_invalid", "schemaVersion is required", node.Line, node.Column)
	}
	var version int
	if err := versionNode.Decode(&version); err != nil || version != 1 {
		return decodedEnvelope{}, chartDiagnosticAt("chart.schema_version_unsupported", "schemaVersion must be integer 1", versionNode.Line, versionNode.Column)
	}
	typeNode, ok := fields["type"]
	if !ok {
		return decodedEnvelope{}, chartDiagnosticAt("chart.schema_invalid", "type is required", node.Line, node.Column)
	}
	var chartType string
	if err := typeNode.Decode(&chartType); err != nil || chartType == "" {
		return decodedEnvelope{}, chartDiagnosticAt("chart.schema_invalid", "type must be a non-empty string", typeNode.Line, typeNode.Column)
	}
	var model any
	switch chartType {
	case "bar":
		var typed barModel
		if err := decodeStrictNode(node, &typed); err != nil {
			return decodedEnvelope{}, chartDiagnosticAt("chart.schema_invalid", err.Error(), node.Line, node.Column)
		}
		model = typed
	case "line":
		var typed lineModel
		if err := decodeStrictNode(node, &typed); err != nil {
			return decodedEnvelope{}, chartDiagnosticAt("chart.schema_invalid", err.Error(), node.Line, node.Column)
		}
		model = typed
	case "pie", "doughnut":
		var typed pieModel
		if err := decodeStrictNode(node, &typed); err != nil {
			return decodedEnvelope{}, chartDiagnosticAt("chart.schema_invalid", err.Error(), node.Line, node.Column)
		}
		model = typed
	case "scatter":
		var typed scatterModel
		if err := decodeStrictNode(node, &typed); err != nil {
			return decodedEnvelope{}, chartDiagnosticAt("chart.schema_invalid", err.Error(), node.Line, node.Column)
		}
		model = typed
	default:
		return decodedEnvelope{}, chartDiagnostic("chart.type_unsupported", fmt.Sprintf("chart type %q is unsupported", chartType))
	}
	return decodedEnvelope{Type: chartType, Model: model}, nil
}

func decodeDocument(payload []byte) (*yaml.Node, error) {
	if len(bytes.TrimSpace(payload)) == 0 {
		return nil, fmt.Errorf("chart payload is empty")
	}
	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("chart payload must contain one document")
		}
		return nil, err
	}
	if len(document.Content) != 1 || document.Content[0] == nil {
		return nil, fmt.Errorf("chart payload must contain one document")
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("chart payload must be an object")
	}
	if err := rejectDuplicateMappings(root); err != nil {
		return nil, err
	}
	return root, nil
}

func rejectDuplicateMappings(node *yaml.Node) error {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case yaml.MappingNode:
		seen := make(map[string]struct{}, len(node.Content)/2)
		for index := 0; index+1 < len(node.Content); index += 2 {
			key := node.Content[index]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
				continue
			}
			if _, exists := seen[key.Value]; exists {
				return fmt.Errorf("duplicate field %q", key.Value)
			}
			seen[key.Value] = struct{}{}
			if err := rejectDuplicateMappings(node.Content[index+1]); err != nil {
				return err
			}
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if err := rejectDuplicateMappings(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func mappingFields(node *yaml.Node) (map[string]*yaml.Node, error) {
	if node == nil || node.Kind != yaml.MappingNode || len(node.Content)%2 != 0 {
		return nil, fmt.Errorf("chart payload mapping is malformed")
	}
	fields := make(map[string]*yaml.Node, len(node.Content)/2)
	for index := 0; index < len(node.Content); index += 2 {
		key := node.Content[index]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
			return nil, fmt.Errorf("chart object keys must be strings")
		}
		if _, exists := fields[key.Value]; exists {
			return nil, fmt.Errorf("duplicate field %q", key.Value)
		}
		fields[key.Value] = node.Content[index+1]
	}
	return fields, nil
}

func decodeStrictNode(node *yaml.Node, target any) error {
	payload, err := yaml.Marshal(node)
	if err != nil {
		return err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(payload))
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}
