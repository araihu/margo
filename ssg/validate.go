package ssg

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/a-h/templ"
	"github.com/araihu/margo/internal/canonicaljson"
)

var allSwapModes = []SwapMode{SwapInnerHTML, SwapOuterHTML, SwapBeforeBegin, SwapAfterBegin, SwapBeforeEnd, SwapAfterEnd}

// NormalizeSchema fills defaults that participate in the schema identity.
func NormalizeSchema(input FrameSchema) (FrameSchema, error) {
	output := input
	output.Areas = append([]AreaDescriptor(nil), input.Areas...)
	output.Options = append([]FrameOptionDescriptor(nil), input.Options...)
	for index := range output.Options {
		output.Options[index].Allowed = append([]string(nil), input.Options[index].Allowed...)
	}
	output.Mounts = append([]FrameMountDescriptor(nil), input.Mounts...)
	output.Resources = append([]ResourceRequirement(nil), input.Resources...)
	for index := range output.Resources {
		output.Resources[index].Attributes = cloneStringMap(input.Resources[index].Attributes)
	}
	output.BindingDefaults = cloneStringMap(input.BindingDefaults)
	output.BindingOrder = cloneStringSliceMap(input.BindingOrder)
	for index := range output.Areas {
		area := &output.Areas[index]
		area.Accepts = append([]string(nil), area.Accepts...)
		area.Triggers = append([]string(nil), area.Triggers...)
		area.AllowedSwaps = append([]SwapMode(nil), area.AllowedSwaps...)
		area.Slots = append([]SlotDescriptor(nil), area.Slots...)
		if area.Target == "" {
			area.Target = area.ID
		}
		if area.Swap == "" {
			area.Swap = SwapInnerHTML
		}
		if area.AllowedSwaps == nil {
			area.AllowedSwaps = stableSwaps()
		}
		if area.Live == "" {
			area.Live = "off"
		}
		if area.Focus == "" {
			area.Focus = "retain"
		}
		if area.MaxBindings == 0 && !area.Multiple && area.Role != "document" {
			area.MaxBindings = 1
		}
	}
	return output, nil
}

// ValidateFrameSchema verifies the structural contract. profile may be empty
// for generic /ssg callers; DocsProfile enables the document/navigation rules.
func ValidateFrameSchema(input FrameSchema, profile string) error {
	schema, err := NormalizeSchema(input)
	if err != nil {
		return err
	}
	if schema.Contract != FrameContract {
		return fmt.Errorf("ssg.schema_contract: got %q, want %q", schema.Contract, FrameContract)
	}
	if len(schema.Areas) == 0 {
		return fmt.Errorf("ssg.schema_areas: frame must declare at least one area")
	}
	if err := ValidateFrameOptions(schema.Options); err != nil {
		return err
	}
	if err := ValidateResources(schema.Resources); err != nil {
		return err
	}
	areaIDs := make(map[string]struct{}, len(schema.Areas))
	targets := make(map[string]struct{}, len(schema.Areas))
	documentAreas := 0
	for _, area := range schema.Areas {
		if area.ID == "" || strings.ContainsAny(area.ID, " <>\t\r\n") {
			return fmt.Errorf("ssg.area_invalid: invalid area ID %q", area.ID)
		}
		if _, exists := areaIDs[area.ID]; exists {
			return fmt.Errorf("ssg.area_duplicate: area %q is declared more than once", area.ID)
		}
		areaIDs[area.ID] = struct{}{}
		if _, exists := targets[area.Target]; exists {
			return fmt.Errorf("ssg.target_duplicate: target %q is declared more than once", area.Target)
		}
		targets[area.Target] = struct{}{}
		if area.Role == "document" {
			documentAreas++
			if !area.Required || area.Multiple || !slices.Contains(area.Accepts, "document") {
				return fmt.Errorf("ssg.document_area_invalid: area %q must be required, singular, and accept document", area.ID)
			}
		}
		if area.MaxBindings < 0 {
			return fmt.Errorf("ssg.area_limit_invalid: area %q has negative MaxBindings", area.ID)
		}
		if area.Swap != SwapInnerHTML && !slices.Contains(allSwapModes, area.Swap) {
			return fmt.Errorf("ssg.swap_invalid: area %q has invalid swap %q", area.ID, area.Swap)
		}
		if area.Live != "off" && area.Live != "polite" && area.Live != "assertive" {
			return fmt.Errorf("ssg.live_invalid: area %q has invalid live mode %q", area.ID, area.Live)
		}
		if area.Focus != "retain" && area.Focus != "first-focusable" && area.Focus != "trigger" && area.Focus != "area" {
			return fmt.Errorf("ssg.focus_invalid: area %q has invalid focus policy %q", area.ID, area.Focus)
		}
		seenKinds := map[string]struct{}{}
		for _, kind := range area.Accepts {
			if kind == "" {
				return fmt.Errorf("ssg.accepts_invalid: area %q has an empty payload kind", area.ID)
			}
			if _, exists := seenKinds[kind]; exists {
				return fmt.Errorf("ssg.accepts_duplicate: area %q repeats payload kind %q", area.ID, kind)
			}
			seenKinds[kind] = struct{}{}
		}
		seenSlots := map[string]int{}
		for _, slot := range area.Slots {
			if slot.ID == "" || slot.Order < 1 {
				return fmt.Errorf("ssg.slot_invalid: area %q has invalid slot %q", area.ID, slot.ID)
			}
			if _, exists := seenSlots[slot.ID]; exists {
				return fmt.Errorf("ssg.slot_duplicate: area %q repeats slot %q", area.ID, slot.ID)
			}
			seenSlots[slot.ID] = slot.Order
			for _, kind := range slot.Accepts {
				if kind == "" || !slices.Contains(area.Accepts, kind) && kind != "pagination" {
					return fmt.Errorf("ssg.slot_accepts_invalid: area %q slot %q accepts unsupported kind %q", area.ID, slot.ID, kind)
				}
			}
		}
		orders := make([]int, 0, len(seenSlots))
		for _, order := range seenSlots {
			orders = append(orders, order)
		}
		sort.Ints(orders)
		for index, order := range orders {
			if order != index+1 {
				return fmt.Errorf("ssg.slot_order_invalid: area %q slots must use contiguous one-based order", area.ID)
			}
		}
		for _, swap := range area.AllowedSwaps {
			if !slices.Contains(allSwapModes, swap) {
				return fmt.Errorf("ssg.swap_invalid: area %q allows invalid swap %q", area.ID, swap)
			}
			if area.Live != "off" && (swap == SwapBeforeBegin || swap == SwapAfterEnd) {
				return fmt.Errorf("ssg.swap_boundary: live area %q cannot allow %q", area.ID, swap)
			}
		}
		if !slices.Contains(area.AllowedSwaps, area.Swap) {
			return fmt.Errorf("ssg.swap_default_invalid: area %q default swap %q is not allowed", area.ID, area.Swap)
		}
	}
	if profile == DocsProfile {
		if documentAreas != 1 {
			return fmt.Errorf("ssg.docs_document_area: documentation profile needs exactly one document area")
		}
		navigation := false
		for _, area := range schema.Areas {
			if slices.Contains(area.Accepts, "navigation") {
				navigation = true
				break
			}
		}
		if !navigation {
			return fmt.Errorf("ssg.docs_navigation_area: documentation profile needs a navigation-capable area")
		}
	}
	if err := validatePlacements(schema, areaIDs); err != nil {
		return err
	}
	if err := validateBreakpoints(schema.Layout.Breakpoints); err != nil {
		return err
	}
	if err := validateMounts(schema, areaIDs, targets); err != nil {
		return err
	}
	if err := validateBindingOrder(schema, areaIDs); err != nil {
		return err
	}
	return nil
}

func validateMounts(schema FrameSchema, areaIDs map[string]struct{}, areaTargets map[string]struct{}) error {
	seenIDs := make(map[string]struct{}, len(schema.Mounts))
	seenTargets := make(map[string]struct{}, len(schema.Mounts))
	for _, mount := range schema.Mounts {
		if mount.ID == "" || strings.ContainsAny(mount.ID, " <>\t\r\n") {
			return fmt.Errorf("ssg.mount_invalid: invalid mount ID %q", mount.ID)
		}
		if _, exists := seenIDs[mount.ID]; exists {
			return fmt.Errorf("ssg.mount_duplicate: mount %q is declared more than once", mount.ID)
		}
		seenIDs[mount.ID] = struct{}{}
		if _, exists := areaIDs[mount.HostArea]; !exists {
			return fmt.Errorf("ssg.mount_host: mount %q names unknown host area %q", mount.ID, mount.HostArea)
		}
		if mount.Target == "" || strings.ContainsAny(mount.Target, " <>\t\r\n") {
			return fmt.Errorf("ssg.mount_target: mount %q has invalid target %q", mount.ID, mount.Target)
		}
		if _, exists := areaTargets[mount.Target]; exists {
			return fmt.Errorf("ssg.mount_target_collision: mount %q target %q collides with an area target", mount.ID, mount.Target)
		}
		if _, exists := seenTargets[mount.Target]; exists {
			return fmt.Errorf("ssg.mount_target_collision: mount target %q is declared more than once", mount.Target)
		}
		seenTargets[mount.Target] = struct{}{}
		if mount.Contract != FrameContract {
			return fmt.Errorf("ssg.mount_contract: mount %q requires contract %q, got %q", mount.ID, FrameContract, mount.Contract)
		}
	}
	return nil
}

func validateBindingOrder(schema FrameSchema, areaIDs map[string]struct{}) error {
	mounts := make(map[string]FrameMountDescriptor, len(schema.Mounts))
	for _, mount := range schema.Mounts {
		mounts[mount.ID] = mount
	}
	for area := range schema.BindingOrder {
		if _, exists := areaIDs[area]; !exists {
			return fmt.Errorf("ssg.binding_order_area: binding order names unknown area %q", area)
		}
	}
	for _, descriptor := range schema.Areas {
		order, exists := schema.BindingOrder[descriptor.ID]
		if !exists {
			return fmt.Errorf("ssg.binding_order_missing: area %q has no binding order", descriptor.ID)
		}
		seen := map[string]struct{}{}
		expected := map[string]struct{}{}
		for _, kind := range descriptor.Accepts {
			expected[kind] = struct{}{}
		}
		for _, slot := range descriptor.Slots {
			for _, kind := range slot.Accepts {
				expected[kind] = struct{}{}
			}
		}
		for _, mount := range schema.Mounts {
			if mount.HostArea == descriptor.ID {
				expected["mount:"+mount.ID] = struct{}{}
			}
		}
		for _, kind := range order {
			if _, exists := seen[kind]; exists {
				return fmt.Errorf("ssg.binding_order_duplicate: area %q repeats %q", descriptor.ID, kind)
			}
			seen[kind] = struct{}{}
			if strings.HasPrefix(kind, "mount:") {
				mountID := strings.TrimPrefix(kind, "mount:")
				mount, exists := mounts[mountID]
				if !exists || mount.HostArea != descriptor.ID {
					return fmt.Errorf("ssg.binding_order_mount: area %q orders unknown mount %q", descriptor.ID, mountID)
				}
				continue
			}
			if _, exists := expected[kind]; !exists {
				return fmt.Errorf("ssg.binding_order_kind: area %q orders unsupported kind %q", descriptor.ID, kind)
			}
		}
		for kind := range expected {
			if _, exists := seen[kind]; !exists {
				return fmt.Errorf("ssg.binding_order_incomplete: area %q omits %q", descriptor.ID, kind)
			}
		}
	}
	return nil
}

// ValidateShellSchema verifies the shell contract and its declared
// resources. Shell areas remain layout-neutral and are validated by the
// shell adapter that owns their interaction policy.
func ValidateShellSchema(input ShellSchema) error {
	if input.Contract != ShellContract {
		return fmt.Errorf("ssg.shell_contract: got %q, want %q", input.Contract, ShellContract)
	}
	return ValidateResources(input.Resources)
}

func validatePlacements(schema FrameSchema, areaIDs map[string]struct{}) error {
	placements := []FramePlacement{schema.Layout.Wide, schema.Layout.Mid, schema.Layout.Narrow}
	for index, placement := range placements {
		name := []string{"wide", "mid", "narrow"}[index]
		if len(placement.Rows) == 0 || len(placement.Columns) == 0 {
			return fmt.Errorf("ssg.layout_tracks: %s placement must declare rows and columns", name)
		}
		if len(placement.Regions) != len(areaIDs) || len(placement.SourceOrder) != len(areaIDs) {
			return fmt.Errorf("ssg.layout_regions: %s placement must enumerate every area exactly once", name)
		}
		seen := map[string]struct{}{}
		occupied := map[[2]int]string{}
		for _, region := range placement.Regions {
			if _, exists := areaIDs[region.Area]; !exists {
				return fmt.Errorf("ssg.layout_area: %s placement names unknown area %q", name, region.Area)
			}
			if _, exists := seen[region.Area]; exists {
				return fmt.Errorf("ssg.layout_duplicate: %s placement repeats area %q", name, region.Area)
			}
			seen[region.Area] = struct{}{}
			if region.RowStart < 1 || region.RowEnd <= region.RowStart || region.RowEnd > len(placement.Rows)+1 || region.ColumnStart < 1 || region.ColumnEnd <= region.ColumnStart || region.ColumnEnd > len(placement.Columns)+1 {
				return fmt.Errorf("ssg.layout_span: %s placement has invalid span for area %q", name, region.Area)
			}
			if region.Collapse != "none" && region.Collapse != "stack-before" && region.Collapse != "stack-after" && region.Collapse != "drawer-inline-start" && region.Collapse != "drawer-inline-end" {
				return fmt.Errorf("ssg.layout_collapse: %s placement has invalid collapse %q", name, region.Collapse)
			}
			for row := region.RowStart; row < region.RowEnd; row++ {
				for column := region.ColumnStart; column < region.ColumnEnd; column++ {
					key := [2]int{row, column}
					if previous, exists := occupied[key]; exists {
						return fmt.Errorf("ssg.layout_overlap: %s placement areas %q and %q overlap", name, previous, region.Area)
					}
					occupied[key] = region.Area
				}
			}
		}
		for _, area := range placement.SourceOrder {
			if _, exists := seen[area]; !exists {
				return fmt.Errorf("ssg.layout_order: %s placement omits area %q", name, area)
			}
			delete(seen, area)
		}
		if len(seen) != 0 {
			return fmt.Errorf("ssg.layout_order: %s placement source order contains duplicates", name)
		}
	}
	return nil
}

func validateBreakpoints(breakpoints []ContentBreakpoint) error {
	if len(breakpoints) == 0 {
		return fmt.Errorf("ssg.breakpoints: at least one content breakpoint is required")
	}
	lastMax := 0
	seen := map[string]struct{}{}
	for index, breakpoint := range breakpoints {
		if breakpoint.Name == "" || breakpoint.MinCSSPx < 0 || breakpoint.MaxCSSPx != nil && *breakpoint.MaxCSSPx <= breakpoint.MinCSSPx {
			return fmt.Errorf("ssg.breakpoint_invalid: invalid breakpoint %q", breakpoint.Name)
		}
		if _, exists := seen[breakpoint.Name]; exists {
			return fmt.Errorf("ssg.breakpoint_duplicate: breakpoint %q is repeated", breakpoint.Name)
		}
		seen[breakpoint.Name] = struct{}{}
		if index == 0 && breakpoint.MinCSSPx != 0 {
			return fmt.Errorf("ssg.breakpoint_gap: breakpoints must begin at 0 CSS pixels")
		}
		if breakpoint.MinCSSPx != lastMax {
			return fmt.Errorf("ssg.breakpoint_gap: breakpoint %q does not continue previous interval", breakpoint.Name)
		}
		if breakpoint.MaxCSSPx != nil {
			lastMax = *breakpoint.MaxCSSPx
		} else {
			lastMax = breakpoint.MinCSSPx
		}
	}
	return nil
}

// SchemaHash returns the domain-separated SHA-256 of the normalized schema.
func SchemaHash(input FrameSchema) (string, error) {
	return SchemaHashForValues(input, nil)
}

// NewAreaBinding freezes a rendered semantic fragment into the identity used
// by the layout boundary.
func NewAreaBinding(schemaHash, route string, spec BindingSpec, ordinal int, component templ.Component) (AreaBinding, error) {
	if schemaHash == "" || spec.Kind == "" || component == nil || ordinal < 0 {
		return AreaBinding{}, fmt.Errorf("ssg.binding_invalid: schema hash, kind, component, and ordinal are required")
	}
	var fragment bytes.Buffer
	if err := component.Render(context.Background(), &fragment); err != nil {
		return AreaBinding{}, fmt.Errorf("ssg.binding_render: %w", err)
	}
	digest := payloadDigest("margo.ssg.area-payload/v1", fragment.Bytes())
	key, err := canonicaljson.Marshal(struct {
		SchemaHash string   `json:"schemaHash"`
		Route      string   `json:"route"`
		Path       []string `json:"compositionPath"`
		Area       string   `json:"area"`
		Slot       string   `json:"slot"`
		Ordinal    int      `json:"ordinal"`
		Kind       string   `json:"kind"`
		Digest     string   `json:"digest"`
	}{schemaHash, route, spec.CompositionPath, spec.Area, spec.Slot, ordinal, spec.Kind, digest})
	if err != nil {
		return AreaBinding{}, fmt.Errorf("ssg.binding_token: %w", err)
	}
	tokenHash := sha256.New()
	_, _ = tokenHash.Write([]byte("margo.ssg.area-marker/v1\x00"))
	_, _ = tokenHash.Write(key)
	return AreaBinding{Kind: spec.Kind, CompositionPath: append([]string(nil), spec.CompositionPath...), Slot: spec.Slot, Token: hex.EncodeToString(tokenHash.Sum(nil)), Digest: digest, Component: component}, nil
}

// ValidateBindings enforces capability, multiplicity, slot, and document
// invariants for one frame instance.
func ValidateBindings(schema FrameSchema, bindings map[string][]AreaBinding) error {
	normalized, err := NormalizeSchema(schema)
	if err != nil {
		return err
	}
	seenTokens := map[string]struct{}{}
	for areaID, values := range bindings {
		area := findArea(normalized, areaID)
		if area.ID == "" {
			return fmt.Errorf("ssg.binding_area: binding targets unknown area %q", areaID)
		}
		areaCounts := map[string]int{}
		slotCounts := map[string]int{}
		for _, binding := range values {
			if binding.Kind == "" || binding.Token == "" || binding.Digest == "" || binding.Component == nil {
				return fmt.Errorf("ssg.binding_invalid: area %q contains an incomplete %q binding", areaID, binding.Kind)
			}
			var payload bytes.Buffer
			if err := binding.Component.Render(context.Background(), &payload); err != nil {
				return fmt.Errorf("ssg.binding_render: area %q %s: %w", areaID, binding.Kind, err)
			}
			if got := payloadDigest("margo.ssg.area-payload/v1", payload.Bytes()); got != binding.Digest {
				return fmt.Errorf("ssg.binding_digest: area %q %s payload digest mismatch", areaID, binding.Kind)
			}
			accepted := slices.Contains(area.Accepts, binding.Kind)
			if binding.Slot != "" {
				accepted = false
				for _, slot := range area.Slots {
					if slot.ID == binding.Slot && slices.Contains(slot.Accepts, binding.Kind) {
						accepted = true
						slotCounts[binding.Slot]++
						break
					}
				}
				if !accepted {
					return fmt.Errorf("ssg.binding_slot: area %q does not accept %s in slot %q", areaID, binding.Kind, binding.Slot)
				}
			} else {
				areaCounts[binding.Kind]++
			}
			if !accepted {
				return fmt.Errorf("ssg.binding_kind: area %q does not accept %q", areaID, binding.Kind)
			}
			if _, exists := seenTokens[binding.Token]; exists {
				return fmt.Errorf("ssg.binding_duplicate: token %q is used more than once", binding.Token)
			}
			seenTokens[binding.Token] = struct{}{}
		}
		if !area.Multiple && len(areaCounts) > 1 {
			// A document area may have one area binding plus declared slots.
			if area.Role != "document" {
				return fmt.Errorf("ssg.binding_multiple: area %q accepts only one binding", areaID)
			}
		}
		if !area.Multiple {
			for kind, count := range areaCounts {
				if count > 1 {
					return fmt.Errorf("ssg.binding_multiple: area %q repeats %q", areaID, kind)
				}
			}
		}
		for kind, count := range areaCounts {
			if limit := area.MaxBindingsByKind[kind]; limit > 0 && count > limit {
				return fmt.Errorf("ssg.binding_limit: area %q exceeds %q limit %d", areaID, kind, limit)
			}
		}
		if area.MaxBindings > 0 && len(values)-len(slotCounts) > area.MaxBindings {
			return fmt.Errorf("ssg.binding_limit: area %q exceeds maximum %d", areaID, area.MaxBindings)
		}
		for slot, count := range slotCounts {
			if count > 1 {
				return fmt.Errorf("ssg.binding_slot_multiple: area %q repeats slot %q", areaID, slot)
			}
		}
		if area.Role == "document" && areaCounts["document"] != 1 {
			return fmt.Errorf("ssg.document_binding: area %q needs exactly one document binding", areaID)
		}
	}
	for _, area := range normalized.Areas {
		if area.Required && len(bindings[area.ID]) == 0 {
			return fmt.Errorf("ssg.binding_required: required area %q has no binding", area.ID)
		}
	}
	return nil
}

func payloadDigest(domain string, payload []byte) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte(domain))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write(payload)
	return hex.EncodeToString(hash.Sum(nil))
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneStringSliceMap(input map[string][]string) map[string][]string {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string][]string, len(input))
	for key, values := range input {
		output[key] = append([]string(nil), values...)
	}
	return output
}
