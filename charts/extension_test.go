package charts

import (
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

func TestExtensionRegistrationOwnsReservedFence(t *testing.T) {
	registration := Extension()
	if registration.Identity.Name == "" || registration.Identity.Version == "" {
		t.Fatalf("identity = %#v", registration.Identity)
	}
	if len(registration.Fences) != 1 || registration.Fences[0] != "goshtosochart" {
		t.Fatalf("fences = %#v", registration.Fences)
	}
}

func TestExtensionFactoryBindsFrozenPolicy(t *testing.T) {
	rc := margo.RenderContext{EffectivePolicy: margo.EffectivePolicy{OutputBytes: 4096}}
	session, err := extensionFactory(rc)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := session.(chartSession)
	if !ok {
		t.Fatalf("session type = %T", session)
	}
	if got.context.EffectivePolicy.OutputBytes != 4096 {
		t.Fatalf("policy = %#v", got.context.EffectivePolicy)
	}
}

func TestDecodeEnvelopeRejectsUnsupportedVersionAndUnknownFields(t *testing.T) {
	for _, payload := range []string{
		"schemaVersion: 2\ntype: bar\ntitle: T\ncategories: [A]\nseries: [{name: S, values: [1]}]\n",
		"schemaVersion: 1\ntype: bar\nextra: true\n",
	} {
		if _, err := decodeEnvelope([]byte(payload)); err == nil {
			t.Fatalf("payload accepted: %s", payload)
		}
	}
	if _, err := decodeEnvelope([]byte("schemaVersion: 1\ntype: radar\n")); err == nil || !strings.Contains(err.Error(), "chart.type_unsupported") {
		t.Fatalf("unsupported type error = %v", err)
	}
	if _, err := decodeEnvelope([]byte("schemaVersion: 1\ntype: bar\ntitle: T\ncategories: [A]\nseries: [{name: S, values: [1], values: [2]}]\n")); err == nil || !strings.Contains(err.Error(), "chart.syntax_invalid") {
		t.Fatalf("duplicate nested field error = %v", err)
	}
}
