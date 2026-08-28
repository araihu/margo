package margo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	jsonschemaValidator "github.com/santhosh-tekuri/jsonschema/v6"
)

// jsonSchemaFenceName is the canonical Markdown fence for publishing a
// human-readable property tree from a JSON Schema document. json-schema is
// retained as a spelling alias because it is the name used by most editors.
const jsonSchemaFenceName = "jsonschema"

var jsonSchemaFenceNames = map[string]struct{}{
	jsonSchemaFenceName: {},
	"json-schema":       {},
}

type jsonSchemaCompiled struct {
	schema    []byte
	reference string
}

func (compiled jsonSchemaCompiled) cloneCompiledExtensionNode() compiledExtensionNode {
	compiled.schema = append([]byte(nil), compiled.schema...)
	return compiled
}

type jsonSchemaSession struct{}

func (jsonSchemaSession) Render(ctx context.Context, node ExtensionNode, out io.Writer) error {
	compiled, ok := node.compiled.(jsonSchemaCompiled)
	if !ok || len(compiled.schema) == 0 {
		return fmt.Errorf("jsonschema.compiled_missing: JSON Schema fence was not compiled")
	}
	if err := contextError(ctx); err != nil {
		return err
	}
	return renderJSONSchemaDocument(ctx, out, compiled.schema, compiled.reference)
}

func jsonSchemaExtensionRegistration() ExtensionRegistration {
	return ExtensionRegistration{
		Identity: ExtensionIdentity{
			Name:         "margo-jsonschema",
			Version:      "v1",
			Capabilities: []string{"static-html", "NamespacedIDsV1"},
		},
		Fences:  []string{jsonSchemaFenceName, "json-schema"},
		Factory: func(RenderContext) (ExtensionSession, error) { return jsonSchemaSession{}, nil },
		Check:   checkJSONSchemaFence,
		compile: compileJSONSchemaNode,
	}
}

func compileJSONSchemaNode(compileContext extensionCompileContext, node ExtensionNode, _ uint32) (ExtensionNode, error) {
	data, reference, err := resolveJSONSchemaFence(context.Background(), node, compileContext.source)
	if err != nil {
		return ExtensionNode{}, jsonSchemaNodeError(node, "jsonschema.reference_invalid", err.Error())
	}
	if _, err := parseAndValidateJSONSchema(data); err != nil {
		return ExtensionNode{}, jsonSchemaNodeError(node, "jsonschema.schema_invalid", err.Error())
	}
	node.compiled = jsonSchemaCompiled{schema: append([]byte(nil), data...), reference: reference}
	return node, nil
}

func checkJSONSchemaFence(ctx context.Context, node ExtensionNode) error {
	data, _, err := resolveJSONSchemaFence(ctx, node, Source{Name: node.Source.Source, BaseURL: node.BaseURL})
	if err != nil {
		return jsonSchemaNodeError(node, "jsonschema.reference_invalid", err.Error())
	}
	if _, err := parseAndValidateJSONSchema(data); err != nil {
		return jsonSchemaNodeError(node, "jsonschema.schema_invalid", err.Error())
	}
	return nil
}

func jsonSchemaNodeError(node ExtensionNode, code, message string) error {
	return newDiagnosticError(Diagnostic{
		Code: code, Severity: SeverityError, Source: node.Source.Source,
		Line: node.Source.Line, Column: node.Source.Column, Pointer: "/jsonschema",
		Message: strings.TrimSpace(message), Hint: "Provide a valid inline JSON Schema, a contained ref=path.json file, or an embedded margo://schema/v1/output reference.",
	})
}

func isJSONSchemaFence(name string) bool {
	_, ok := jsonSchemaFenceNames[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

// resolveJSONSchemaFence returns the exact schema bytes used for the static
// tree. A fence with no ref= argument takes its body as inline JSON. A ref
// can be a path relative to the source directory or a margo:// embedded
// schema reference. The optional #/pointer suffix selects a nested schema.
func resolveJSONSchemaFence(ctx context.Context, node ExtensionNode, source Source) ([]byte, string, error) {
	if err := contextError(ctx); err != nil {
		return nil, "", err
	}
	reference, err := parseJSONSchemaFenceInfo(node.Info)
	if err != nil {
		return nil, "", err
	}
	if reference == "" {
		if len(bytes.TrimSpace(node.Payload)) == 0 {
			return nil, "", errors.New("JSON Schema fence body is empty")
		}
		return append([]byte(nil), node.Payload...), "inline", nil
	}
	if strings.HasPrefix(reference, "margo://") {
		data, err := embeddedJSONSchemaReference(reference)
		if err != nil {
			return nil, "", err
		}
		selected, err := selectJSONSchemaFragment(data, reference)
		return selected, reference, err
	}
	parsed, err := url.Parse(reference)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || strings.HasPrefix(reference, "//") {
		return nil, "", fmt.Errorf("schema reference %q must be a relative path or margo:// reference", reference)
	}
	fileReference, _ := splitJSONSchemaFragment(reference)
	if fileReference == "" || filepath.IsAbs(filepath.FromSlash(fileReference)) {
		return nil, "", fmt.Errorf("schema reference %q must be a non-empty relative path", reference)
	}
	if len(bytes.TrimSpace(node.Payload)) != 0 {
		return nil, "", errors.New("JSON Schema fence cannot combine ref= with an inline body")
	}
	root := node.BaseURL
	if root == "" {
		root = source.BaseURL
	}
	if root == "" {
		return nil, "", errors.New("relative schema reference requires Source.BaseURL")
	}
	reader := node.AssetReader
	if reader == nil {
		reader = FilesystemCheckAssetReader{}
	}
	data, err := reader.ReadAsset(ctx, root, fileReference, MaxDocumentBytes)
	if err != nil {
		return nil, "", fmt.Errorf("cannot read schema reference %q: %w", fileReference, err)
	}
	selected, err := selectJSONSchemaFragment(data, reference)
	if err != nil {
		return nil, "", err
	}
	return selected, reference, nil
}

func parseJSONSchemaFenceInfo(info string) (string, error) {
	trimmed := strings.TrimSpace(info)
	if trimmed == "" {
		return "", nil
	}
	firstEnd := strings.IndexAny(trimmed, " \t\r\n")
	if firstEnd < 0 {
		return "", nil
	}
	rest := strings.TrimSpace(trimmed[firstEnd:])
	if rest == "" {
		return "", nil
	}
	// The short form ```jsonschema path/to/schema.json``` is convenient in
	// prose. The explicit ref= and path= forms are equivalent and permit a
	// quoted path containing spaces.
	if strings.HasPrefix(rest, "=") {
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "="))
	}
	for _, key := range []string{"ref", "path"} {
		for _, separator := range []string{"=", ":"} {
			prefix := key + separator
			if strings.HasPrefix(strings.ToLower(rest), prefix) {
				rest = strings.TrimSpace(rest[len(prefix):])
				return parseJSONSchemaReferenceValue(rest)
			}
		}
	}
	if strings.ContainsAny(rest, " \t\r\n") {
		return "", errors.New("JSON Schema fence options must be ref=path or path")
	}
	return rest, nil
}

func parseJSONSchemaReferenceValue(value string) (string, error) {
	if value == "" {
		return "", errors.New("JSON Schema ref= value is empty")
	}
	if value[0] == '\'' {
		if len(value) < 2 || value[len(value)-1] != '\'' {
			return "", errors.New("JSON Schema ref= value has an unterminated quote")
		}
		// Shell-style single quotes are useful in Markdown fences, but Go's
		// strconv.Unquote treats a multi-character single-quoted value as an
		// invalid rune literal. Keep the deliberately small grammar here and
		// only unescape the quote itself; backslashes otherwise remain path data.
		return strings.ReplaceAll(value[1:len(value)-1], `\'`, `'`), nil
	}
	if value[0] == '"' {
		quote := value[0]
		if len(value) < 2 || value[len(value)-1] != quote {
			return "", errors.New("JSON Schema ref= value has an unterminated quote")
		}
		decoded, err := strconv.Unquote(value)
		if err != nil {
			return "", fmt.Errorf("JSON Schema ref= value is not validly quoted: %w", err)
		}
		return decoded, nil
	}
	if strings.ContainsAny(value, " \t\r\n") {
		return "", errors.New("JSON Schema ref= value with spaces must be quoted")
	}
	return value, nil
}

func splitJSONSchemaFragment(reference string) (string, string) {
	index := strings.IndexByte(reference, '#')
	if index < 0 {
		return reference, ""
	}
	return reference[:index], reference[index+1:]
}

func selectJSONSchemaFragment(data []byte, reference string) ([]byte, error) {
	_, fragment := splitJSONSchemaFragment(reference)
	if fragment == "" {
		return append([]byte(nil), data...), nil
	}
	if !strings.HasPrefix(fragment, "/") {
		return nil, fmt.Errorf("schema fragment %q must be a JSON pointer", fragment)
	}
	decodedFragment, err := url.PathUnescape(fragment)
	if err != nil {
		return nil, fmt.Errorf("schema fragment %q is not URI-encoded correctly: %w", fragment, err)
	}
	fragment = decodedFragment
	// Validate the complete source before selecting a fragment. Otherwise a
	// duplicate key or trailing JSON value outside the selected subtree could
	// be silently discarded by the decoder and evade the schema contract.
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return nil, fmt.Errorf("schema reference is not one unique JSON value: %w", err)
	}
	var root any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("schema reference is not valid JSON: %w", err)
	}
	current := root
	for _, rawSegment := range strings.Split(strings.TrimPrefix(fragment, "/"), "/") {
		segment := strings.ReplaceAll(strings.ReplaceAll(rawSegment, "~1", "/"), "~0", "~")
		switch typed := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = typed[segment]
			if !ok {
				return nil, fmt.Errorf("schema fragment %q does not exist", fragment)
			}
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, fmt.Errorf("schema fragment %q has an invalid array index", fragment)
			}
			current = typed[index]
		default:
			return nil, fmt.Errorf("schema fragment %q traverses a scalar", fragment)
		}
	}
	// Keep local definition catalogs available when a fragment points at a
	// property or definition that uses #/$defs/... references. Clone before
	// attaching the catalog: a selected definition is itself nested inside the
	// source catalog, and assigning that catalog directly would create a Go map
	// cycle that cannot be marshaled.
	selectedValue := cloneJSONSchemaValue(current)
	if selected, ok := selectedValue.(map[string]any); ok {
		if source, exists := root.(map[string]any); exists {
			for _, key := range []string{"$defs", "definitions"} {
				if _, present := selected[key]; !present {
					if definitions, found := source[key]; found {
						selected[key] = cloneJSONSchemaValue(definitions)
					}
				}
			}
		}
	}
	selected, err := json.Marshal(selectedValue)
	if err != nil {
		return nil, fmt.Errorf("schema fragment %q cannot be encoded: %w", fragment, err)
	}
	return selected, nil
}

func cloneJSONSchemaValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		clone := make(map[string]any, len(typed))
		for key, child := range typed {
			clone[key] = cloneJSONSchemaValue(child)
		}
		return clone
	case []any:
		clone := make([]any, len(typed))
		for index, child := range typed {
			clone[index] = cloneJSONSchemaValue(child)
		}
		return clone
	default:
		return value
	}
}

func parseAndValidateJSONSchema(data []byte) (map[string]any, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var root map[string]any
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}
	if root == nil {
		return nil, errors.New("JSON Schema root must be an object")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, errors.New("JSON Schema must contain one JSON value")
		}
		return nil, err
	}
	document, err := jsonschemaValidator.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	compiler := jsonschemaValidator.NewCompiler()
	digest := sha256.Sum256(data)
	resource := "margo://jsonschema/" + hex.EncodeToString(digest[:])
	if err := compiler.AddResource(resource, document); err != nil {
		return nil, err
	}
	if _, err := compiler.Compile(resource); err != nil {
		return nil, err
	}
	return root, nil
}

type jsonSchemaRow struct {
	path        string
	typeName    string
	required    bool
	description string
	constraints string
}

type jsonSchemaTreeNode struct {
	segment  string
	row      jsonSchemaRow
	children []*jsonSchemaTreeNode
}

func renderJSONSchemaDocument(ctx context.Context, out io.Writer, data []byte, reference string) error {
	root, err := parseAndValidateJSONSchema(data)
	if err != nil {
		return err
	}
	title, _ := root["title"].(string)
	if strings.TrimSpace(title) == "" {
		title = "JSON Schema"
	}
	description, _ := root["description"].(string)
	rows := collectJSONSchemaRows(root)
	if _, err := fmt.Fprintf(out, `<section class="margo-jsonschema" data-margo-jsonschema-reference="%s" aria-label="%s">`, html.EscapeString(reference), html.EscapeString(title)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, `<h3 class="margo-jsonschema__title">%s</h3>`, html.EscapeString(title)); err != nil {
		return err
	}
	if strings.TrimSpace(description) != "" {
		if _, err := fmt.Fprintf(out, `<p class="margo-jsonschema__description">%s</p>`, html.EscapeString(description)); err != nil {
			return err
		}
	}
	if len(rows) == 0 {
		_, err := io.WriteString(out, `<p class="margo-jsonschema__empty">This schema does not declare object properties.</p></section>`)
		return err
	}
	if _, err := io.WriteString(out, `<div class="margo-jsonschema__tree" aria-label="Properties defined by this schema"><ul class="margo-jsonschema__tree-list">`); err != nil {
		return err
	}
	for _, node := range buildJSONSchemaTree(rows) {
		if err := renderJSONSchemaTreeNode(ctx, out, node); err != nil {
			return err
		}
	}
	_, err = io.WriteString(out, `</ul></div></section>`)
	return err
}

func buildJSONSchemaTree(rows []jsonSchemaRow) []*jsonSchemaTreeNode {
	roots := make([]*jsonSchemaTreeNode, 0)
	byPath := make(map[string]*jsonSchemaTreeNode)
	for _, row := range rows {
		segments := strings.Split(strings.TrimPrefix(row.path, "/"), "/")
		parentPath := ""
		var parent *jsonSchemaTreeNode
		for _, segment := range segments {
			if segment == "" {
				continue
			}
			path := parentPath + "/" + segment
			node := byPath[path]
			if node == nil {
				node = &jsonSchemaTreeNode{segment: unescapeJSONPointerSegment(segment), row: jsonSchemaRow{path: path}}
				byPath[path] = node
				if parent == nil {
					roots = append(roots, node)
				} else {
					parent.children = append(parent.children, node)
				}
			}
			parent, parentPath = node, path
		}
		if parent != nil {
			parent.row = row
		}
	}
	return roots
}

func unescapeJSONPointerSegment(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~1", "/"), "~0", "~")
}

func renderJSONSchemaTreeNode(ctx context.Context, out io.Writer, node *jsonSchemaTreeNode) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	row := node.row
	if _, err := fmt.Fprintf(out, `<li class="margo-jsonschema__tree-node"><div class="margo-jsonschema__tree-row"><code class="margo-jsonschema__tree-path" title="%s">%s</code>`, html.EscapeString(row.path), html.EscapeString(node.segment)); err != nil {
		return err
	}
	if row.typeName != "" {
		if _, err := fmt.Fprintf(out, `<span class="margo-jsonschema__tree-type">%s</span>`, html.EscapeString(row.typeName)); err != nil {
			return err
		}
		if row.required {
			if _, err := io.WriteString(out, `<span class="margo-jsonschema__tree-required" title="required" aria-label="required">*</span>`); err != nil {
				return err
			}
		}
	}
	if _, err := io.WriteString(out, `</div>`); err != nil {
		return err
	}
	if strings.TrimSpace(row.description) != "" {
		if _, err := fmt.Fprintf(out, `<p class="margo-jsonschema__tree-description">%s</p>`, html.EscapeString(row.description)); err != nil {
			return err
		}
	}
	if strings.TrimSpace(row.constraints) != "" {
		if _, err := fmt.Fprintf(out, `<p class="margo-jsonschema__tree-constraints">%s</p>`, html.EscapeString(row.constraints)); err != nil {
			return err
		}
	}
	if len(node.children) > 0 {
		if _, err := io.WriteString(out, `<ul class="margo-jsonschema__tree-list">`); err != nil {
			return err
		}
		for _, child := range node.children {
			if err := renderJSONSchemaTreeNode(ctx, out, child); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, `</ul>`); err != nil {
			return err
		}
	}
	_, err := io.WriteString(out, `</li>`)
	return err
}

func collectJSONSchemaRows(root map[string]any) []jsonSchemaRow {
	rows := make([]jsonSchemaRow, 0)
	seen := make(map[string]struct{})
	var walk func(map[string]any, string, map[string]bool, map[string]bool, map[string]bool)
	walk = func(schema map[string]any, prefix string, inheritedRequired map[string]bool, stack map[string]bool, refStack map[string]bool) {
		// A legal JSON Schema may refer back to an enclosing object (for
		// example, a tree node's `children.items.$ref: "#"`). Stop following a
		// repeated local reference while retaining the row for the property that
		// introduced it; otherwise a documentation build could recurse forever.
		nextRefs := refStack
		if reference, ok := schema["$ref"].(string); ok && isLocalJSONSchemaReference(reference) {
			if refStack[reference] {
				return
			}
			nextRefs = make(map[string]bool, len(refStack)+1)
			for key, value := range refStack {
				nextRefs[key] = value
			}
			nextRefs[reference] = true
		}
		resolved := resolveJSONSchemaObject(root, schema)
		if resolved == nil {
			return
		}
		identity := fmt.Sprintf("%p:%s", resolved, prefix)
		if stack[identity] {
			return
		}
		nextStack := make(map[string]bool, len(stack)+1)
		for key, value := range stack {
			nextStack[key] = value
		}
		nextStack[identity] = true
		required := requiredJSONSchemaProperties(resolved)
		for key, value := range inheritedRequired {
			if value {
				required[key] = true
			}
		}
		properties := jsonSchemaProperties(root, resolved)
		keys := make([]string, 0, len(properties))
		for key := range properties {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			property := properties[key]
			property = resolveJSONSchemaObject(root, property)
			if property == nil {
				continue
			}
			rowPath := prefix + "/" + escapeJSONPointerSegment(key)
			rowKey := rowPath
			if _, exists := seen[rowKey]; !exists {
				rows = append(rows, jsonSchemaRow{
					path: rowPath, typeName: jsonSchemaTypeName(root, property), required: required[key],
					description: jsonSchemaString(property, "description"), constraints: jsonSchemaConstraints(property),
				})
				seen[rowKey] = struct{}{}
			}
			walk(property, rowPath, nil, nextStack, nextRefs)
			if items := jsonSchemaMap(property["items"]); items != nil {
				walk(items, rowPath+"/*", nil, nextStack, nextRefs)
			}
			for _, variant := range jsonSchemaVariants(property) {
				walk(variant, rowPath, nil, nextStack, nextRefs)
			}
		}
	}
	walk(root, "", nil, make(map[string]bool), make(map[string]bool))
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].path < rows[j].path })
	return rows
}

func escapeJSONPointerSegment(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}

func resolveJSONSchemaObject(root map[string]any, schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	resolved := schema
	if reference, ok := schema["$ref"].(string); ok && isLocalJSONSchemaReference(reference) {
		if target := resolveJSONSchemaPointer(root, reference); target != nil {
			if len(schema) == 1 {
				return target
			}
			merged := make(map[string]any, len(target)+len(schema))
			for key, value := range target {
				merged[key] = value
			}
			for key, value := range schema {
				merged[key] = value
			}
			return merged
		}
	}
	return resolved
}

func resolveJSONSchemaPointer(root map[string]any, reference string) map[string]any {
	if reference == "#" {
		return root
	}
	if !strings.HasPrefix(reference, "#/") {
		return nil
	}
	fragment, err := url.PathUnescape(strings.TrimPrefix(reference, "#"))
	if err != nil || !strings.HasPrefix(fragment, "/") {
		return nil
	}
	var current any = root
	for _, raw := range strings.Split(strings.TrimPrefix(fragment, "/"), "/") {
		segment := strings.ReplaceAll(strings.ReplaceAll(raw, "~1", "/"), "~0", "~")
		switch typed := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = typed[segment]
			if !ok {
				return nil
			}
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(typed) {
				return nil
			}
			current = typed[index]
		default:
			return nil
		}
	}
	result, _ := current.(map[string]any)
	return result
}

func isLocalJSONSchemaReference(reference string) bool {
	return reference == "#" || strings.HasPrefix(reference, "#/")
}

func jsonSchemaProperties(root map[string]any, schema map[string]any) map[string]map[string]any {
	result := make(map[string]map[string]any)
	if properties, ok := schema["properties"].(map[string]any); ok {
		for key, value := range properties {
			if typed, ok := value.(map[string]any); ok {
				result[key] = typed
			}
		}
	}
	for _, key := range []string{"allOf", "anyOf", "oneOf"} {
		variants, _ := schema[key].([]any)
		for _, value := range variants {
			variant, ok := value.(map[string]any)
			if !ok {
				continue
			}
			for name, property := range jsonSchemaProperties(root, resolveJSONSchemaObject(root, variant)) {
				if _, exists := result[name]; !exists {
					result[name] = property
				}
			}
		}
	}
	return result
}

func requiredJSONSchemaProperties(schema map[string]any) map[string]bool {
	result := make(map[string]bool)
	values, _ := schema["required"].([]any)
	for _, value := range values {
		if key, ok := value.(string); ok {
			result[key] = true
		}
	}
	// `allOf` composes object contracts, so a property required by any
	// conjunct is required by the effective schema. (For `anyOf`/`oneOf`, no
	// single branch is universally required and the table keeps the safer
	// optional label.)
	variants, _ := schema["allOf"].([]any)
	for _, value := range variants {
		if variant, ok := value.(map[string]any); ok {
			for key := range requiredJSONSchemaProperties(variant) {
				result[key] = true
			}
		}
	}
	return result
}

func jsonSchemaVariants(schema map[string]any) []map[string]any {
	result := make([]map[string]any, 0)
	for _, key := range []string{"allOf", "anyOf", "oneOf"} {
		values, _ := schema[key].([]any)
		for _, value := range values {
			if typed, ok := value.(map[string]any); ok {
				result = append(result, typed)
			}
		}
	}
	return result
}

func jsonSchemaTypeName(root map[string]any, schema map[string]any) string {
	if value, ok := schema["type"]; ok {
		switch typed := value.(type) {
		case string:
			return typed
		case []any:
			values := make([]string, 0, len(typed))
			for _, item := range typed {
				if text, ok := item.(string); ok {
					values = append(values, text)
				}
			}
			if len(values) > 0 {
				return strings.Join(values, " | ")
			}
		}
	}
	if reference, ok := schema["$ref"].(string); ok {
		if resolved := resolveJSONSchemaObject(root, schema); resolved != nil {
			if _, hasType := resolved["type"]; hasType {
				return jsonSchemaTypeName(root, resolved)
			}
			if _, hasProperties := resolved["properties"]; hasProperties {
				return "object"
			}
		}
		return "ref " + path.Base(strings.TrimPrefix(reference, "#/"))
	}
	for _, key := range []string{"oneOf", "anyOf"} {
		values, _ := schema[key].([]any)
		if len(values) > 0 {
			types := make([]string, 0, len(values))
			for _, value := range values {
				if typed, ok := value.(map[string]any); ok {
					types = append(types, jsonSchemaTypeName(root, typed))
				}
			}
			if len(types) > 0 {
				return strings.Join(types, " | ")
			}
		}
	}
	if _, ok := schema["properties"]; ok {
		return "object"
	}
	return "value"
}

func jsonSchemaMap(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}

func jsonSchemaString(schema map[string]any, key string) string {
	value, _ := schema[key].(string)
	return strings.TrimSpace(value)
}

func jsonSchemaConstraints(schema map[string]any) string {
	parts := make([]string, 0, 6)
	for _, key := range []string{"const", "enum", "default", "format", "pattern", "minimum", "maximum", "minLength", "maxLength", "minItems", "maxItems"} {
		value, ok := schema[key]
		if !ok {
			continue
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			continue
		}
		parts = append(parts, key+"="+string(encoded))
	}
	if unique, _ := schema["uniqueItems"].(bool); unique {
		parts = append(parts, "uniqueItems=true")
	}
	return strings.Join(parts, "; ")
}
