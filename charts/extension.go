package charts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/a-h/templ"
	chartassets "github.com/araihu/goshtoso-charts/assets"
	goshtosoassets "github.com/araihu/goshtoso/assets"
	margo "github.com/araihu/margo"
	"gopkg.in/yaml.v3"
)

type familyHandler func(margo.RenderContext, any, chartRenderOptions) (templ.Component, error)

// chartRenderOptions are frozen when the extension registration is created.
// Keeping the setting out of RenderContext makes it part of the extension
// identity rather than a mutable per-render concern.
type chartRenderOptions struct {
	controlWrapper             bool
	externalizedControlRuntime bool
	printAccessibleData        bool
	deckProjection             bool
}

var defaultChartRenderOptions = chartRenderOptions{controlWrapper: true}

// Option configures the optional charts extension.
type Option func(*chartRenderOptions)

// WithControlWrapper controls the server-rendered chart controls wrapper.
// The wrapper is enabled by default; passing false opts into HTML containing
// only the static SVG and its accessible data table.
func WithControlWrapper(enabled bool) Option {
	return func(options *chartRenderOptions) {
		options.controlWrapper = enabled
	}
}

// WithChartControlWrapper is a descriptive alias for WithControlWrapper.
func WithChartControlWrapper(enabled bool) Option {
	return WithControlWrapper(enabled)
}

// WithExternalizedControlRuntime declares chart control dependencies through
// the extension capability envelope and suppresses the exact upstream loader.
// The default remains false for consumers pinned to an older Margo root.
func WithExternalizedControlRuntime(enabled bool) Option {
	return func(options *chartRenderOptions) {
		options.externalizedControlRuntime = enabled
	}
}

// WithPrintableAccessibleData includes one formatted exact-data table after
// each chart in print/PDF output. Exact data remains available in HTML when
// this option is false, but print output hides it by default.
func WithPrintableAccessibleData(enabled bool) Option {
	return func(options *chartRenderOptions) {
		options.printAccessibleData = enabled
	}
}

// WithDeckProjection marks a compiler used by the CLI deck projection. Decks
// intentionally omit browser chart controls; the marker lets an interactive
// chart fail with the target-specific contract diagnostic instead of exposing
// the lower-level missing-wrapper error.
func WithDeckProjection(enabled bool) Option {
	return func(options *chartRenderOptions) {
		options.deckProjection = enabled
	}
}

type chartSession struct {
	context  margo.RenderContext
	handlers map[string]familyHandler
	options  chartRenderOptions
}

var familyRegistry = struct {
	sync.RWMutex
	items map[string]familyHandler
}{items: make(map[string]familyHandler)}

// Extension registers the reserved goshtosochart fence for optional use by a
// host compiler. The chart control wrapper is enabled unless explicitly
// disabled with WithControlWrapper(false).
func Extension(options ...Option) margo.ExtensionRegistration {
	config := defaultChartRenderOptions
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	capabilities := []string{"static-svg", "accessible-data", "NamespacedIDsV1"}
	var capabilityErr error
	if config.controlWrapper && config.externalizedControlRuntime {
		var runtimeCapabilities []string
		runtimeCapabilities, capabilityErr = chartControlRequirementCapabilities()
		if capabilityErr == nil {
			capabilities = append(capabilities, runtimeCapabilities...)
		}
	}
	factory := extensionFactoryFor(config)
	if capabilityErr != nil {
		factory = func(margo.RenderContext) (margo.ExtensionSession, error) {
			return nil, fmt.Errorf("chart.runtime_materialization_failed: %w", capabilityErr)
		}
	}
	return margo.ExtensionRegistration{
		Identity: margo.ExtensionIdentity{
			Name:              "margo-charts",
			Version:           "v0.0.1-dev",
			ConfigurationHash: config.configurationHash(),
			Capabilities:      capabilities,
		},
		Fences:  []string{"goshtosochart"},
		Factory: factory,
		Check:   func(ctx context.Context, node margo.ExtensionNode) error { return checkChart(ctx, node, config) },
	}
}

func extensionFactory(rc margo.RenderContext) (margo.ExtensionSession, error) {
	return extensionFactoryWithOptions(rc, defaultChartRenderOptions)
}

func extensionFactoryFor(options chartRenderOptions) margo.ExtensionFactory {
	return func(rc margo.RenderContext) (margo.ExtensionSession, error) {
		return extensionFactoryWithOptions(rc, options)
	}
}

func extensionFactoryWithOptions(rc margo.RenderContext, options chartRenderOptions) (margo.ExtensionSession, error) {
	familyRegistry.RLock()
	handlers := make(map[string]familyHandler, len(familyRegistry.items))
	for name, handler := range familyRegistry.items {
		handlers[name] = handler
	}
	familyRegistry.RUnlock()
	return chartSession{context: rc, handlers: handlers, options: options}, nil
}

func (options chartRenderOptions) configurationHash() string {
	value := fmt.Sprintf("control-wrapper=%t\ndeck-projection=%t\nexternalized-control-runtime=%t\nprint-accessible-data=%t", options.controlWrapper, options.deckProjection, options.externalizedControlRuntime, options.printAccessibleData)
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

type encodedRequirementCapability struct {
	ID        string                            `json:"id"`
	Kind      string                            `json:"kind"`
	LocalURL  string                            `json:"localURL"`
	Integrity string                            `json:"integrity,omitempty"`
	LoadAfter []string                          `json:"loadAfter,omitempty"`
	Inline    encodedRequirementCapabilityAsset `json:"inline"`
}

type encodedRequirementCapabilityAsset struct {
	Path      string `json:"path"`
	MediaType string `json:"mediaType"`
	SHA256    string `json:"sha256"`
	Content   []byte `json:"content"`
}

type controlRuntimeAsset struct {
	id         string
	localURL   string
	inlinePath string
	integrity  string
	loadAfter  []string
	handler    http.Handler
	prefix     string
}

func chartControlRequirementCapabilities() ([]string, error) {
	manifest := goshtosoassets.DefaultRuntimeManifest()
	focus, found := runtimeAssetByRole(manifest, goshtosoassets.RuntimeRoleAlpineFocus)
	if !found {
		return nil, fmt.Errorf("Goshtoso runtime role %q is unavailable", goshtosoassets.RuntimeRoleAlpineFocus)
	}
	firstParty, found := runtimeAssetByRole(manifest, goshtosoassets.RuntimeRoleFirstParty)
	if !found {
		return nil, fmt.Errorf("Goshtoso runtime role %q is unavailable", goshtosoassets.RuntimeRoleFirstParty)
	}
	alpine, found := runtimeAssetByRole(manifest, goshtosoassets.RuntimeRoleAlpineJS)
	if !found {
		return nil, fmt.Errorf("Goshtoso runtime role %q is unavailable", goshtosoassets.RuntimeRoleAlpineJS)
	}
	assets := []controlRuntimeAsset{
		{id: "goshtoso.runtime.alpine-focus", localURL: focus.LocalURL, integrity: focus.Integrity, loadAfter: []string{"margo.document.styles"}, handler: goshtosoassets.Handler(), prefix: "/assets/"},
		{id: "goshtoso.runtime.first-party", localURL: firstParty.LocalURL, integrity: firstParty.Integrity, loadAfter: []string{"goshtoso.runtime.alpine-focus"}, handler: goshtosoassets.Handler(), prefix: "/assets/"},
		{id: "goshtoso.runtime.alpine", localURL: alpine.LocalURL, integrity: alpine.Integrity, loadAfter: []string{"goshtoso.runtime.first-party"}, handler: goshtosoassets.Handler(), prefix: "/assets/"},
		{id: "goshtoso-charts.runtime", localURL: chartassets.RuntimeURL, inlinePath: "charts-runtime.js", loadAfter: []string{"margo.document.styles"}, handler: chartassets.Handler(), prefix: chartassets.Prefix},
		{id: "goshtoso-charts.controls", localURL: chartassets.ControlRuntimeURL, loadAfter: []string{"goshtoso-charts.runtime", "goshtoso.runtime.alpine"}, handler: chartassets.Handler(), prefix: chartassets.Prefix},
	}
	capabilities := make([]string, 0, len(assets))
	for _, asset := range assets {
		capability, err := materializeRequirementCapability(asset)
		if err != nil {
			return nil, err
		}
		capabilities = append(capabilities, capability)
	}
	return capabilities, nil
}

func runtimeAssetByRole(manifest goshtosoassets.RuntimeManifest, role goshtosoassets.RuntimeAssetRole) (goshtosoassets.RuntimeAsset, bool) {
	for _, asset := range manifest.Dependencies {
		if asset.Role == role && asset.Enabled {
			return asset, true
		}
	}
	return goshtosoassets.RuntimeAsset{}, false
}

func materializeRequirementCapability(asset controlRuntimeAsset) (string, error) {
	if !strings.HasPrefix(asset.localURL, asset.prefix) {
		return "", fmt.Errorf("runtime %q URL %q is outside mount %q", asset.id, asset.localURL, asset.prefix)
	}
	recorder := httptest.NewRecorder()
	asset.handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, asset.localURL, nil))
	if recorder.Code != http.StatusOK {
		return "", fmt.Errorf("runtime %q returned HTTP %d", asset.id, recorder.Code)
	}
	content := append([]byte(nil), recorder.Body.Bytes()...)
	if len(content) == 0 {
		return "", fmt.Errorf("runtime %q returned no bytes", asset.id)
	}
	digest := sha256.Sum256(content)
	inlinePath := asset.inlinePath
	if inlinePath == "" {
		inlinePath = strings.TrimPrefix(asset.localURL, asset.prefix)
	}
	value := encodedRequirementCapability{
		ID: asset.id, Kind: "script", LocalURL: asset.localURL,
		Integrity: asset.integrity, LoadAfter: append([]string(nil), asset.loadAfter...),
		Inline: encodedRequirementCapabilityAsset{
			Path: inlinePath, MediaType: "application/javascript",
			SHA256: hex.EncodeToString(digest[:]), Content: content,
		},
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("runtime %q capability: %w", asset.id, err)
	}
	return "margo-html-requirement/v1:" + base64.RawURLEncoding.EncodeToString(data), nil
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
	component, err := handler(s.context, envelope.Model, s.options)
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
	if err := validateChartSchema(chartType, node); err != nil {
		return decodedEnvelope{}, err
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
