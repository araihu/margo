package charts

import (
	"context"
	"errors"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

func TestExtensionCheckMatchesChartTargetMatrix(t *testing.T) {
	payload := []byte(`schemaVersion: 1
type: line
renderer: interactive
title: Weekly revenue
categories: [Mon, Tue]
series:
  - name: Revenue
    values: [12, 18]
`)
	registration := Extension()
	if registration.Check == nil {
		t.Fatal("chart extension does not expose a compatibility checker")
	}
	for _, target := range []margo.RenderTarget{margo.TargetHTML, margo.TargetSite, margo.TargetPDF} {
		t.Run(string(target), func(t *testing.T) {
			if err := registration.Check(context.Background(), margo.ExtensionNode{Fence: "goshtosochart", Payload: payload, Target: target}); err != nil {
				t.Fatalf("interactive chart rejected for %s: %v", target, err)
			}
		})
	}

	err := registration.Check(context.Background(), margo.ExtensionNode{Fence: "goshtosochart", Payload: payload, Target: margo.TargetDeck})
	if err == nil || !strings.Contains(err.Error(), "chart.renderer_target_unsupported") {
		t.Fatalf("deck check error = %v, want chart.renderer_target_unsupported", err)
	}
	var diagnostic *margo.DiagnosticError
	if !errors.As(err, &diagnostic) || len(diagnostic.Diagnostics) != 1 {
		t.Fatalf("deck check diagnostics = %v", err)
	}
	if got := diagnostic.Diagnostics[0]; got.Pointer != "/renderer" || got.Hint == "" {
		t.Fatalf("deck check diagnostic = %+v", got)
	}

	staticPayload := strings.Replace(string(payload), "renderer: interactive\n", "renderer: static\n", 1)
	if err := registration.Check(context.Background(), margo.ExtensionNode{Fence: "goshtosochart", Payload: []byte(staticPayload), Target: margo.TargetDeck}); err != nil {
		t.Fatalf("static deck chart rejected: %v", err)
	}
}

func TestExtensionCheckHonorsDisabledControlWrapper(t *testing.T) {
	payload := []byte(`schemaVersion: 1
type: bar
renderer: interactive
title: Weekly revenue
categories: [Mon]
series:
  - name: Revenue
    values: [12]
`)
	registration := Extension(WithControlWrapper(false))
	err := registration.Check(context.Background(), margo.ExtensionNode{Fence: "goshtosochart", Payload: payload, Target: margo.TargetHTML})
	if err == nil || !strings.Contains(err.Error(), "chart.renderer_controls_required") {
		t.Fatalf("check error = %v, want chart.renderer_controls_required", err)
	}
}

func TestExtensionCheckRejectsInvalidChartBeforeTargetProjection(t *testing.T) {
	payload := []byte(`schemaVersion: 1
type: bar
renderer: interactive
title: Weekly revenue
categories: [Mon, Tue]
series:
  - name: Revenue
    values: [12]
`)
	err := Extension().Check(context.Background(), margo.ExtensionNode{Fence: "goshtosochart", Payload: payload, Target: margo.TargetDeck})
	if err == nil || !strings.Contains(err.Error(), "chart.semantic_alignment_invalid") {
		t.Fatalf("check error = %v, want chart.semantic_alignment_invalid", err)
	}
}
