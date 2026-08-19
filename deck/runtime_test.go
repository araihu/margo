package deck

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/araihu/margo"
)

func TestRuntimeDescriptorBindsV2ProfileAndBothLayoutModes(t *testing.T) {
	result := mustRenderDeck(t, "# One\n---\n# Two\n")
	descriptor, err := result.RuntimeDescriptor("ri-00000042")
	if err != nil {
		t.Fatal(err)
	}
	if descriptor.Protocol != margo.RuntimeProtocolV2 || descriptor.ValidationRequest == nil {
		t.Fatalf("descriptor = %#v", descriptor)
	}
	if descriptor.ValidationRequest.ExpectedFontBundleDigest != result.ValidationRequest().ExpectedFontBundleDigest {
		t.Fatal("descriptor lost immutable validation request")
	}
	var screen, printDOM int
	for _, task := range descriptor.Tasks {
		if strings.HasPrefix(task.Kind, "deck-layout-") {
			switch task.Kind {
			case "deck-layout-screen":
				screen++
			case "deck-layout-print-dom":
				printDOM++
			default:
				t.Fatalf("unknown layout task kind %q", task.Kind)
			}
		}
	}
	if screen != 1 || printDOM != 1 {
		t.Fatalf("layout tasks = screen %d print %d", screen, printDOM)
	}
	if err := margo.ValidateRuntimeDescriptor(descriptor); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeDescriptorIncludesEveryStructuralSlot(t *testing.T) {
	result := mustRenderDeck(t, "<!-- class: columns -->\n<!-- layout: columns -->\n<!-- slot: left -->\n```mermaid\ngraph TD; A-->B\n```\n<!-- slot: right -->\n```mermaid\ngraph TD; C-->D\n```\n<!-- /layout -->\n")
	descriptor, err := result.RuntimeDescriptor("ri-00000042")
	if err != nil {
		t.Fatal(err)
	}
	mermaidTasks := 0
	for _, task := range descriptor.Tasks {
		if task.Kind == "mermaid" {
			mermaidTasks++
		}
	}
	if mermaidTasks != 2 {
		t.Fatalf("mermaid tasks = %d, tasks = %#v", mermaidTasks, descriptor.Tasks)
	}
}

func TestScreenRuntimeDescriptorOmitsPrintTask(t *testing.T) {
	result := mustRenderDeck(t, "# One\n")
	descriptor, err := result.ScreenRuntimeDescriptor("ri-00000042")
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range descriptor.Tasks {
		if task.Kind == "deck-layout-print-dom" {
			t.Fatal("screen descriptor contains print task")
		}
	}
	if descriptor.Protocol != margo.RuntimeProtocolV2 || descriptor.ValidationRequest == nil {
		t.Fatalf("screen descriptor = %#v", descriptor)
	}
}

func TestRuntimeLayoutEnvelopeQuantizesLogicalMetrics(t *testing.T) {
	envelope := LayoutValidationEnvelope{
		Version:           1,
		Mode:              LayoutValidationModeScreen,
		LogicalCanvas:     LogicalCanvas{Width: 1280, Height: 720},
		Stage:             StageMetrics{Scale: 1.1251, OriginX: 0.001, OriginY: 12.999},
		ValidationRequest: validDeckValidationRequest(t),
	}
	encoded, err := CanonicalLayoutValidationEnvelope(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"scale":1.125`) || !strings.Contains(string(encoded), `"originY":13`) {
		t.Fatalf("envelope = %s", encoded)
	}
}

func TestRuntimeLayoutEnvelopeRejectsNonFiniteEvidence(t *testing.T) {
	envelope := LayoutValidationEnvelope{
		Mode:              LayoutValidationModeScreen,
		LogicalCanvas:     LogicalCanvas{Width: math.NaN(), Height: 720},
		ValidationRequest: validDeckValidationRequest(t),
	}
	if _, err := CanonicalLayoutValidationEnvelope(envelope); err == nil {
		t.Fatal("non-finite layout evidence unexpectedly accepted")
	}
}

func TestRuntimeDescriptorRequestIsDefensive(t *testing.T) {
	result := mustRenderDeck(t, "# One\n")
	descriptor, err := result.RuntimeDescriptor("ri-00000042")
	if err != nil {
		t.Fatal(err)
	}
	descriptor.ValidationRequest.ViewportWidth = 1
	again, err := result.RuntimeDescriptor("ri-00000042")
	if err != nil {
		t.Fatal(err)
	}
	if again.ValidationRequest.ViewportWidth == 1 {
		t.Fatal("runtime descriptor aliases validation request")
	}
}

func mustRenderRuntimeDeck(t *testing.T, source string) *Result {
	t.Helper()
	result, err := Render(context.Background(), margo.New(), RenderInput{Name: "runtime.md", Markdown: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func validDeckValidationRequest(t *testing.T) margo.RuntimeValidationRequest {
	t.Helper()
	digest, err := bundledFontDigest(margo.ThemeModern)
	if err != nil {
		t.Fatal(err)
	}
	return margo.RuntimeValidationRequest{ViewportWidth: 1440, ViewportHeight: 900, DeviceScaleFactor: 1, Zoom: 1, BrowserProfile: "chromium-deck-v1", ExpectedFontBundleDigest: digest}
}
