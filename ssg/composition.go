package ssg

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/araihu/margo/internal/canonicaljson"
)

// FrameResolver selects an in-process frame distribution and returns its
// executable contract. A resolver is deliberately separate from the site
// builder so module loading, registry policy, and Goshtoso integration remain
// outside the layout contract.
type FrameResolver func(FrameContext, string, Values) (Frame, error)

// ResolvedComposition is a validated, identity-bound frame tree. Root has an
// empty composition path; descendants use their instance IDs as path
// segments, which gives semantic bindings and mount targets a stable namespace.
type ResolvedComposition struct {
	Root FrameNodeResult
	Hash string
}

// FrameNodeResult is the resolved structural view of one frame instance.
// Children retain declaration order because that order is part of composition
// identity and is available to the parent renderer through mount bindings.
type FrameNodeResult struct {
	InstanceID      string
	MountID         string
	Selector        string
	Values          Values
	CompositionPath []string
	Schema          FrameSchema
	SchemaHash      string
	Children        []FrameNodeResult

	frame Frame
}

// ResolveComposition validates a frame tree bottom-up and computes one hash
// over selectors, structural values, paths, schema hashes, and child order.
// The returned tree can be rendered with Render.
func ResolveComposition(composition FrameComposition, base FrameContext, resolver FrameResolver) (ResolvedComposition, error) {
	if resolver == nil {
		return ResolvedComposition{}, fmt.Errorf("ssg.composition_resolver: resolver is required")
	}
	if composition.Root.InstanceID == "" || composition.Root.Selector == "" {
		return ResolvedComposition{}, fmt.Errorf("ssg.composition_root: root needs instanceID and selector")
	}
	if composition.Root.MountID != "" {
		return ResolvedComposition{}, fmt.Errorf("ssg.composition_root: root cannot declare a mount ID")
	}
	instances := make(map[string]string)
	root, err := resolveCompositionNode(composition.Root, base, resolver, nil, nil, true, instances, 0)
	if err != nil {
		return ResolvedComposition{}, err
	}
	identity, err := compositionIdentity(root)
	if err != nil {
		return ResolvedComposition{}, fmt.Errorf("ssg.composition_identity: %w", err)
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte("margo.ssg.composition/v1\x00"))
	_, _ = hash.Write(identity)
	return ResolvedComposition{Root: root, Hash: hex.EncodeToString(hash.Sum(nil))}, nil
}

func resolveCompositionNode(node FrameNode, base FrameContext, resolver FrameResolver, parent *FrameNodeResult, parentSchema *FrameSchema, root bool, instances map[string]string, depth int) (FrameNodeResult, error) {
	if depth > 256 {
		return FrameNodeResult{}, fmt.Errorf("ssg.composition_depth: composition exceeds 256 levels")
	}
	if err := validateCompositionName("instanceID", node.InstanceID); err != nil {
		return FrameNodeResult{}, err
	}
	if node.Selector == "" || strings.IndexFunc(node.Selector, func(r rune) bool { return r == 0 || r == '\n' || r == '\r' }) >= 0 {
		return FrameNodeResult{}, fmt.Errorf("ssg.composition_selector: instance %q has invalid selector %q", node.InstanceID, node.Selector)
	}
	if previous, exists := instances[node.InstanceID]; exists {
		return FrameNodeResult{}, fmt.Errorf("ssg.composition_instance_duplicate: instance %q repeats path %s", node.InstanceID, previous)
	}
	path := []string(nil)
	if parent != nil {
		path = append(path, parent.CompositionPath...)
		path = append(path, node.InstanceID)
		if node.MountID == "" {
			return FrameNodeResult{}, fmt.Errorf("ssg.composition_mount: child %q must declare a mount ID", node.InstanceID)
		}
		if parentSchema == nil {
			return FrameNodeResult{}, fmt.Errorf("ssg.composition_parent: child %q has no parent schema", node.InstanceID)
		}
		mount, ok := findMount(*parentSchema, node.MountID)
		if !ok {
			return FrameNodeResult{}, fmt.Errorf("ssg.composition_mount_unresolved: child %q references mount %q", node.InstanceID, node.MountID)
		}
		for _, sibling := range parent.Children {
			if sibling.MountID == node.MountID {
				return FrameNodeResult{}, fmt.Errorf("ssg.composition_mount_duplicate: mount %q has more than one child", node.MountID)
			}
		}
		_ = mount
	} else if len(path) != 0 {
		return FrameNodeResult{}, fmt.Errorf("ssg.composition_path: root must have an empty composition path")
	}
	ctx := base
	ctx.InstanceID = node.InstanceID
	ctx.CompositionPath = append([]string(nil), path...)
	ctx.Root = root
	if root {
		ctx.Profile = DocsProfile
	} else {
		ctx.Profile = ""
	}
	frame, err := resolver(ctx, node.Selector, node.Values)
	if err != nil {
		return FrameNodeResult{}, fmt.Errorf("ssg.composition_resolve: %s: %w", node.InstanceID, err)
	}
	if frame == nil {
		return FrameNodeResult{}, fmt.Errorf("ssg.composition_resolve: %s returned nil frame", node.InstanceID)
	}
	schema, err := frame.Schema(ctx)
	if err != nil {
		return FrameNodeResult{}, fmt.Errorf("ssg.composition_schema: %s: %w", node.InstanceID, err)
	}
	if err := ValidateFrameSchema(schema, ctx.Profile); err != nil {
		return FrameNodeResult{}, fmt.Errorf("ssg.composition_schema: %s: %w", node.InstanceID, err)
	}
	if !root && hasDocumentArea(schema) {
		return FrameNodeResult{}, fmt.Errorf("ssg.composition_document_child: child %q declares a document area", node.InstanceID)
	}
	resolvedValues, err := ResolveFrameValues(schema, node.Values)
	if err != nil {
		return FrameNodeResult{}, fmt.Errorf("ssg.composition_values: %s: %w", node.InstanceID, err)
	}
	hash, err := SchemaHashForValues(schema, resolvedValues)
	if err != nil {
		return FrameNodeResult{}, fmt.Errorf("ssg.composition_schema_hash: %s: %w", node.InstanceID, err)
	}
	normalized, err := NormalizeSchema(schema)
	if err != nil {
		return FrameNodeResult{}, fmt.Errorf("ssg.composition_schema: %s: %w", node.InstanceID, err)
	}
	result := FrameNodeResult{
		InstanceID:      node.InstanceID,
		MountID:         node.MountID,
		Selector:        node.Selector,
		Values:          resolvedValues,
		CompositionPath: path,
		Schema:          normalized,
		SchemaHash:      hash,
		frame:           frame,
	}
	instances[node.InstanceID] = compositionPathString(path)
	for _, child := range node.Children {
		resolved, childErr := resolveCompositionNode(child, base, resolver, &result, &result.Schema, false, instances, depth+1)
		if childErr != nil {
			return FrameNodeResult{}, childErr
		}
		result.Children = append(result.Children, resolved)
	}
	for _, mount := range result.Schema.Mounts {
		populated := false
		for _, child := range result.Children {
			if child.MountID == mount.ID {
				populated = true
				break
			}
		}
		if mount.Required && !populated {
			return FrameNodeResult{}, fmt.Errorf("ssg.composition_mount_required: frame %q does not populate required mount %q", result.InstanceID, mount.ID)
		}
	}
	return result, nil
}

// Render evaluates child frames before their parents and passes each child as
// a typed mount binding. bindings is keyed by globally unique instance ID.
func (composition ResolvedComposition) Render(bindings map[string]map[string][]AreaBinding) (FrameOutput, error) {
	if composition.Root.frame == nil || composition.Hash == "" {
		return FrameOutput{}, fmt.Errorf("ssg.composition_render: composition is not resolved")
	}
	seen := make(map[string]struct{})
	output, err := renderCompositionNode(composition.Root, composition.Hash, bindings, seen)
	if err != nil {
		return FrameOutput{}, err
	}
	for instanceID := range bindings {
		if _, exists := seen[instanceID]; !exists {
			return FrameOutput{}, fmt.Errorf("ssg.composition_binding_instance: bindings name unknown instance %q", instanceID)
		}
	}
	return output, nil
}

func renderCompositionNode(node FrameNodeResult, rootHash string, bindings map[string]map[string][]AreaBinding, seen map[string]struct{}) (FrameOutput, error) {
	if _, exists := seen[node.InstanceID]; exists {
		return FrameOutput{}, fmt.Errorf("ssg.composition_cycle: instance %q was visited twice", node.InstanceID)
	}
	seen[node.InstanceID] = struct{}{}
	nodeBindings := bindings[node.InstanceID]
	if nodeBindings == nil {
		nodeBindings = map[string][]AreaBinding{}
	}
	if err := ValidateBindings(node.Schema, nodeBindings); err != nil {
		return FrameOutput{}, fmt.Errorf("ssg.composition_bindings: %s: %w", node.InstanceID, err)
	}
	children := make(map[string]FrameChildBinding, len(node.Children))
	assets := AssetSet{}
	for _, child := range node.Children {
		childOutput, err := renderCompositionNode(child, rootHash, bindings, seen)
		if err != nil {
			return FrameOutput{}, err
		}
		if childOutput.Fragment == nil {
			return FrameOutput{}, fmt.Errorf("ssg.composition_fragment: child %q returned nil fragment", child.InstanceID)
		}
		var fragment bytes.Buffer
		if err := childOutput.Fragment.Render(context.Background(), &fragment); err != nil {
			return FrameOutput{}, fmt.Errorf("ssg.composition_fragment: child %q: %w", child.InstanceID, err)
		}
		if childOutput.SchemaHash != child.SchemaHash {
			return FrameOutput{}, fmt.Errorf("ssg.composition_schema_hash: child %q returned %q, want %q", child.InstanceID, childOutput.SchemaHash, child.SchemaHash)
		}
		mount, ok := findMount(node.Schema, child.MountID)
		if !ok {
			return FrameOutput{}, fmt.Errorf("ssg.composition_mount_unresolved: child %q references mount %q", child.InstanceID, child.MountID)
		}
		if _, exists := children[child.MountID]; exists {
			return FrameOutput{}, fmt.Errorf("ssg.composition_mount_duplicate: mount %q has more than one child", child.MountID)
		}
		children[child.MountID] = FrameChildBinding{
			MountID:         mount.ID,
			HostArea:        mount.HostArea,
			Target:          mount.Target,
			QualifiedTarget: qualifiedTarget(child.CompositionPath, mount.Target),
			InstanceID:      child.InstanceID,
			CompositionPath: append([]string(nil), child.CompositionPath...),
			SchemaHash:      child.SchemaHash,
			Digest:          payloadDigest("margo.ssg.frame-fragment/v1", fragment.Bytes()),
			Fragment:        childOutput.Fragment,
		}
		assets = mergeAssets(assets, childOutput.Assets)
	}
	for mountID, child := range children {
		mount, _ := findMount(node.Schema, mountID)
		if mount.Exclusive && len(nodeBindings[mount.HostArea]) > 0 {
			return FrameOutput{}, fmt.Errorf("ssg.composition_mount_collision: exclusive mount %q shares host area %q with semantic bindings", mountID, mount.HostArea)
		}
		_ = child
	}
	output, err := node.frame.Render(FrameInput{
		SchemaHash:          node.SchemaHash,
		RootCompositionHash: rootHash,
		InstanceID:          node.InstanceID,
		CompositionPath:     append([]string(nil), node.CompositionPath...),
		Values:              node.Values,
		Bindings:            nodeBindings,
		ChildrenByMount:     children,
	})
	if err != nil {
		return FrameOutput{}, fmt.Errorf("ssg.composition_render: %s: %w", node.InstanceID, err)
	}
	if output.Fragment == nil {
		return FrameOutput{}, fmt.Errorf("ssg.composition_fragment: frame %q returned nil fragment", node.InstanceID)
	}
	if output.SchemaHash != "" && output.SchemaHash != node.SchemaHash {
		return FrameOutput{}, fmt.Errorf("ssg.composition_schema_hash: frame %q returned %q, want %q", node.InstanceID, output.SchemaHash, node.SchemaHash)
	}
	if output.SchemaHash == "" {
		output.SchemaHash = node.SchemaHash
	}
	output.Assets = mergeAssets(assets, output.Assets)
	return output, nil
}

func compositionIdentity(node FrameNodeResult) ([]byte, error) {
	children := make([]any, 0, len(node.Children))
	for _, child := range node.Children {
		identity, err := compositionIdentity(child)
		if err != nil {
			return nil, err
		}
		children = append(children, string(identity))
	}
	return canonicaljson.Marshal(struct {
		InstanceID      string   `json:"instanceID"`
		MountID         string   `json:"mountID,omitempty"`
		Selector        string   `json:"selector"`
		Values          Values   `json:"values,omitempty"`
		CompositionPath []string `json:"compositionPath,omitempty"`
		SchemaHash      string   `json:"schemaHash"`
		Children        []any    `json:"children,omitempty"`
	}{node.InstanceID, node.MountID, node.Selector, node.Values, node.CompositionPath, node.SchemaHash, children})
}

func validateCompositionName(kind, value string) error {
	if value == "" || strings.IndexFunc(value, func(r rune) bool { return unicodeSpaceOrControl(r) }) >= 0 {
		return fmt.Errorf("ssg.composition_%s: invalid value %q", kind, value)
	}
	return nil
}

func unicodeSpaceOrControl(r rune) bool {
	return r == 0 || r == '\n' || r == '\r' || r == '\t' || r == ' '
}

func compositionPathString(path []string) string {
	if len(path) == 0 {
		return "<root>"
	}
	return strings.Join(path, "/")
}

func findMount(schema FrameSchema, id string) (FrameMountDescriptor, bool) {
	for _, mount := range schema.Mounts {
		if mount.ID == id {
			return mount, true
		}
	}
	return FrameMountDescriptor{}, false
}

func hasDocumentArea(schema FrameSchema) bool {
	for _, area := range schema.Areas {
		if area.Role == "document" {
			return true
		}
	}
	return false
}

func qualifiedTarget(path []string, target string) string {
	if len(path) == 0 {
		return target
	}
	parts := append(append([]string(nil), path...), target)
	return strings.Join(parts, "--")
}

func mergeAssets(left, right AssetSet) AssetSet {
	seen := make(map[string]struct{}, len(left.Paths)+len(right.Paths))
	paths := make([]string, 0, len(left.Paths)+len(right.Paths))
	for _, path := range append(append([]string(nil), left.Paths...), right.Paths...) {
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	// AssetSet is an identity-bearing output, so declaration order from
	// independent child branches must not leak into the result.
	sort.Strings(paths)
	return AssetSet{Paths: paths}
}
