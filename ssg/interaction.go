package ssg

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ValidateSwapOverride validates an HX-Reswap-style runtime override against
// one normalized area. Empty override means the area's declared default.
func ValidateSwapOverride(area AreaDescriptor, override string) (SwapMode, error) {
	return resolveSwapOverride(area, override, false)
}

// ValidateRootSwapOverride applies the additional subtree boundary rule used
// when the target is the root of a frame mount.
func ValidateRootSwapOverride(area AreaDescriptor, override string) (SwapMode, error) {
	return resolveSwapOverride(area, override, true)
}

// ValidateSwapResponse validates an override and, for outerHTML, proves that
// the response re-declares the qualified target and frame hooks.
func ValidateSwapResponse(area AreaDescriptor, override, qualifiedTarget string, response []byte) error {
	return validateSwapResponse(area, override, qualifiedTarget, response, false)
}

// ValidateRootSwapResponse is the subtree-boundary variant of
// ValidateSwapResponse.
func ValidateRootSwapResponse(area AreaDescriptor, override, qualifiedTarget string, response []byte) error {
	return validateSwapResponse(area, override, qualifiedTarget, response, true)
}

func resolveSwapOverride(area AreaDescriptor, override string, rootMount bool) (SwapMode, error) {
	if area.ID == "" {
		return "", fmt.Errorf("ssg.swap_area: area ID is required")
	}
	if area.Target == "" {
		area.Target = area.ID
	}
	if area.Swap == "" {
		area.Swap = SwapInnerHTML
	}
	if area.Live == "" {
		area.Live = "off"
	}
	if area.AllowedSwaps == nil {
		area.AllowedSwaps = stableSwaps()
	}
	for _, swap := range area.AllowedSwaps {
		if !containsSwap(allSwapModes, swap) {
			return "", fmt.Errorf("ssg.swap_invalid: area %q allows %q", area.ID, swap)
		}
		if area.Live != "off" && isBoundarySwap(swap) {
			return "", fmt.Errorf("ssg.swap_boundary: live area %q cannot allow %q", area.ID, swap)
		}
		if rootMount && isBoundarySwap(swap) {
			return "", fmt.Errorf("ssg.swap_boundary: root mount area %q cannot allow %q", area.ID, swap)
		}
	}
	if !containsSwap(area.AllowedSwaps, area.Swap) {
		return "", fmt.Errorf("ssg.swap_default_invalid: area %q default %q is not allowed", area.ID, area.Swap)
	}
	selected := area.Swap
	if override != "" {
		selected = SwapMode(override)
	}
	if !containsSwap(allSwapModes, selected) {
		return "", fmt.Errorf("ssg.swap_override_invalid: area %q received unsupported swap %q", area.ID, selected)
	}
	if rootMount && isBoundarySwap(selected) {
		return "", fmt.Errorf("ssg.swap_override_boundary: root mount area %q cannot use %q", area.ID, selected)
	}
	if area.Live != "off" && isBoundarySwap(selected) {
		return "", fmt.Errorf("ssg.swap_override_boundary: live area %q cannot use %q", area.ID, selected)
	}
	if !containsSwap(area.AllowedSwaps, selected) {
		return "", fmt.Errorf("ssg.swap_override_disallowed: area %q does not allow %q", area.ID, selected)
	}
	return selected, nil
}

func validateSwapResponse(area AreaDescriptor, override, qualifiedTarget string, response []byte, rootMount bool) error {
	swap, err := resolveSwapOverride(area, override, rootMount)
	if err != nil {
		return err
	}
	if swap != SwapOuterHTML {
		return nil
	}
	return ValidateOuterHTMLResponse(area, qualifiedTarget, response)
}

// ValidateOuterHTMLResponse requires one response root with the same
// qualified target and the hooks needed for future swaps.
func ValidateOuterHTMLResponse(area AreaDescriptor, qualifiedTarget string, response []byte) error {
	target := strings.TrimPrefix(qualifiedTarget, "#")
	if target == "" {
		return fmt.Errorf("ssg.swap_response_target: qualified target is required")
	}
	contextNode := &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"}
	nodes, err := html.ParseFragment(bytes.NewReader(response), contextNode)
	if err != nil {
		return fmt.Errorf("ssg.swap_response_parse: %w", err)
	}
	var elements []*html.Node
	for _, node := range nodes {
		if node.Type == html.TextNode && strings.TrimSpace(node.Data) == "" {
			continue
		}
		if node.Type != html.ElementNode {
			return fmt.Errorf("ssg.swap_response_root: response contains a non-element root")
		}
		elements = append(elements, node)
	}
	if len(elements) != 1 {
		return fmt.Errorf("ssg.swap_response_root: outerHTML response needs exactly one root element")
	}
	attributes := make(map[string]string, len(elements[0].Attr))
	for _, attribute := range elements[0].Attr {
		name := strings.ToLower(attribute.Key)
		if _, exists := attributes[name]; exists {
			return fmt.Errorf("ssg.swap_response_hooks: response repeats attribute %q", name)
		}
		attributes[name] = attribute.Val
	}
	required := map[string]string{
		"id":                target,
		"data-margo-area":   area.ID,
		"data-margo-target": "#" + target,
		"data-margo-swap":   "outerHTML",
		"hx-target":         "#" + target,
		"hx-swap":           "outerHTML",
	}
	for name, expected := range required {
		if attributes[name] != expected {
			return fmt.Errorf("ssg.swap_response_hooks: %s must be %q", name, expected)
		}
	}
	if !strings.Contains(","+attributes["data-margo-allowed-swaps"]+",", ",outerHTML,") {
		return fmt.Errorf("ssg.swap_response_hooks: data-margo-allowed-swaps must retain outerHTML")
	}
	if len(area.Triggers) > 0 {
		if attributes["hx-trigger"] != strings.Join(area.Triggers, ",") {
			return fmt.Errorf("ssg.swap_response_hooks: hx-trigger was not re-declared")
		}
	}
	return nil
}

func containsSwap(values []SwapMode, wanted SwapMode) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func isBoundarySwap(swap SwapMode) bool {
	return swap == SwapBeforeBegin || swap == SwapAfterEnd
}
