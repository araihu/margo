package ssg

import (
	"strings"
	"testing"
)

func TestValidateSwapOverrideEnforcesAreaAndLivePolicy(t *testing.T) {
	area := AreaDescriptor{ID: "results", Target: "results", AllowedSwaps: []SwapMode{SwapInnerHTML, SwapOuterHTML, SwapBeforeBegin}, Swap: SwapInnerHTML}
	if got, err := ValidateSwapOverride(area, ""); err != nil || got != SwapInnerHTML {
		t.Fatalf("default swap = %q, error = %v", got, err)
	}
	if _, err := ValidateSwapOverride(area, string(SwapAfterEnd)); err == nil || !strings.Contains(err.Error(), "disallowed") {
		t.Fatalf("disallowed override error = %v", err)
	}
	area.Live = "polite"
	if _, err := ValidateSwapOverride(area, string(SwapBeforeBegin)); err == nil || !strings.Contains(err.Error(), "boundary") {
		t.Fatalf("live boundary error = %v", err)
	}
	if _, err := ValidateRootSwapOverride(AreaDescriptor{ID: "root", AllowedSwaps: []SwapMode{SwapInnerHTML, SwapBeforeBegin}, Swap: SwapInnerHTML}, string(SwapBeforeBegin)); err == nil || !strings.Contains(err.Error(), "root mount") {
		t.Fatalf("root boundary error = %v", err)
	}
}

func TestValidateOuterHTMLResponseRequiresQualifiedHooks(t *testing.T) {
	area := AreaDescriptor{ID: "results", Target: "results", Triggers: []string{"refresh"}, AllowedSwaps: []SwapMode{SwapInnerHTML, SwapOuterHTML}, Swap: SwapInnerHTML}
	valid := `<section id="child--results" data-margo-area="results" data-margo-target="#child--results" data-margo-swap="outerHTML" data-margo-allowed-swaps="innerHTML,outerHTML" hx-target="#child--results" hx-swap="outerHTML" hx-trigger="refresh"></section>`
	if err := ValidateSwapResponse(area, string(SwapOuterHTML), "child--results", []byte(valid)); err != nil {
		t.Fatalf("valid response rejected: %v", err)
	}
	for _, response := range []string{
		valid + `<p>second root</p>`,
		`<section id="results" data-margo-area="results" data-margo-target="#results" data-margo-swap="outerHTML" data-margo-allowed-swaps="innerHTML,outerHTML" hx-target="#results" hx-swap="outerHTML" hx-trigger="refresh"></section>`,
		`<section id="child--results" data-margo-area="results" data-margo-target="#child--results" data-margo-swap="outerHTML" data-margo-allowed-swaps="innerHTML,outerHTML" hx-target="#child--results" hx-swap="innerHTML" hx-trigger="refresh"></section>`,
	} {
		if err := ValidateSwapResponse(area, string(SwapOuterHTML), "child--results", []byte(response)); err == nil {
			t.Fatalf("invalid response unexpectedly succeeded: %s", response)
		}
	}
}
