package charts

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"slices"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

type requirementCapabilityValue struct {
	ID        string                      `json:"id"`
	Kind      string                      `json:"kind"`
	LocalURL  string                      `json:"localURL,omitempty"`
	Integrity string                      `json:"integrity,omitempty"`
	LoadAfter []string                    `json:"loadAfter,omitempty"`
	Inline    *requirementCapabilityAsset `json:"inline,omitempty"`
}

type requirementCapabilityAsset struct {
	Path      string `json:"path"`
	MediaType string `json:"mediaType"`
	SHA256    string `json:"sha256"`
	Content   []byte `json:"content"`
}

func decodeRequirementCapabilities(t *testing.T, capabilities []string) []requirementCapabilityValue {
	t.Helper()
	const prefix = "margo-html-requirement/v1:"
	values := make([]requirementCapabilityValue, 0, len(capabilities))
	for _, capability := range capabilities {
		if !strings.HasPrefix(capability, prefix) {
			continue
		}
		data, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(capability, prefix))
		if err != nil {
			t.Fatal(err)
		}
		var value requirementCapabilityValue
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&value); err != nil {
			t.Fatal(err)
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			t.Fatalf("trailing capability data: %v", err)
		}
		values = append(values, value)
	}
	return values
}

func requireCapabilityIDs(t *testing.T, values []requirementCapabilityValue, expected ...string) {
	t.Helper()
	got := make([]string, len(values))
	for index := range values {
		got[index] = values[index].ID
	}
	if !slices.Equal(got, expected) {
		t.Fatalf("capability IDs = %v, want %v", got, expected)
	}
}

func TestExtensionRegistrationOwnsReservedFence(t *testing.T) {
	registration := Extension()
	if registration.Identity.Name == "" || registration.Identity.Version == "" {
		t.Fatalf("identity = %#v", registration.Identity)
	}
	if len(registration.Fences) != 1 || registration.Fences[0] != "goshtosochart" {
		t.Fatalf("fences = %#v", registration.Fences)
	}
}

func TestExtensionDeclaresEnabledControlRequirements(t *testing.T) {
	registration := Extension(WithExternalizedControlRuntime(true))
	capabilities := decodeRequirementCapabilities(t, registration.Identity.Capabilities)
	requireCapabilityIDs(t, capabilities,
		"goshtoso.runtime.alpine-focus", "goshtoso.runtime.first-party",
		"goshtoso.runtime.alpine", "goshtoso-charts.controls",
	)
	loadAfter := [][]string{{"margo.document.styles"}, {"goshtoso.runtime.alpine-focus"}, {"goshtoso.runtime.first-party"}, {"goshtoso.runtime.alpine"}}
	for index, capability := range capabilities {
		if capability.Kind != "script" || capability.LocalURL == "" || capability.Inline == nil || len(capability.Inline.Content) == 0 || capability.Inline.MediaType != "application/javascript" {
			t.Fatalf("capability %d = %#v", index, capability)
		}
		digest := sha256.Sum256(capability.Inline.Content)
		if capability.Inline.SHA256 != hex.EncodeToString(digest[:]) {
			t.Fatalf("capability %s SHA-256 = %q", capability.ID, capability.Inline.SHA256)
		}
		if !slices.Equal(capability.LoadAfter, loadAfter[index]) {
			t.Fatalf("capability %s loadAfter = %v, want %v", capability.ID, capability.LoadAfter, loadAfter[index])
		}
	}
}

func TestExtensionDeclaresNoControlRequirementsWhenDisabled(t *testing.T) {
	for _, registration := range []margo.ExtensionRegistration{
		Extension(),
		Extension(WithExternalizedControlRuntime(false)),
		Extension(WithControlWrapper(false), WithExternalizedControlRuntime(true)),
	} {
		if capabilities := decodeRequirementCapabilities(t, registration.Identity.Capabilities); len(capabilities) != 0 {
			t.Fatalf("disabled requirements = %#v", capabilities)
		}
	}
}

func TestExternalizedControlRuntimeChangesConfigurationIdentity(t *testing.T) {
	ordinary := Extension().Identity.ConfigurationHash
	externalized := Extension(WithExternalizedControlRuntime(true)).Identity.ConfigurationHash
	if ordinary == "" || externalized == "" || ordinary == externalized {
		t.Fatalf("configuration hashes = ordinary %q externalized %q", ordinary, externalized)
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
