package margo

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"testing"
)

func requirementAsset(path, mediaType, content string) AssetRef {
	data := []byte(content)
	digest := sha256.Sum256(data)
	return AssetRef{
		Path:      path,
		MediaType: mediaType,
		SHA256:    hex.EncodeToString(digest[:]),
		Content:   data,
	}
}

func requireRequirementIDs(t *testing.T, requirements HTMLRequirements, expected ...string) {
	t.Helper()
	list := requirements.List()
	if len(list) != len(expected) {
		t.Fatalf("requirement count = %d, want %d: %#v", len(list), len(expected), list)
	}
	for index, id := range expected {
		if list[index].ID != id {
			t.Fatalf("requirement IDs[%d] = %q, want %q: %#v", index, list[index].ID, id, list)
		}
	}
}

func TestHTMLRequirementsMergeOrdersDeduplicatesAndClones(t *testing.T) {
	styles := HTMLRequirement{
		ID:       "styles",
		Kind:     HTMLStylesheet,
		LocalURL: "/margo-assets/document.css",
		Inline:   requirementAsset("document.css", "text/css", ".margo-document{}"),
	}
	merged, err := mergeHTMLRequirements([]HTMLRequirement{
		{ID: "runtime", Kind: HTMLScript, LocalURL: "/margo-assets/runtime.js", LoadAfter: []string{"styles"}},
		styles,
		styles,
	})
	if err != nil {
		t.Fatal(err)
	}
	list := merged.List()
	if len(list) != 2 || list[0].ID != "styles" || list[1].ID != "runtime" {
		t.Fatalf("order = %#v", list)
	}
	list[0].ID = "mutated"
	list[0].Inline.Content[0] = 'X'
	list[1].LoadAfter[0] = "mutated"
	again := merged.List()
	if again[0].ID != "styles" || string(again[0].Inline.Content) != ".margo-document{}" || again[1].LoadAfter[0] != "styles" {
		t.Fatalf("requirements alias caller: %#v", again)
	}
}

func TestHTMLRequirementsRejectInvalidGraphs(t *testing.T) {
	tests := []struct {
		name         string
		requirements []HTMLRequirement
		code         string
	}{
		{
			name: "conflict",
			requirements: []HTMLRequirement{
				{ID: "styles", Kind: HTMLStylesheet, LocalURL: "/one.css"},
				{ID: "styles", Kind: HTMLStylesheet, LocalURL: "/two.css"},
			},
			code: "html.requirement_conflict",
		},
		{
			name:         "missing dependency",
			requirements: []HTMLRequirement{{ID: "runtime", Kind: HTMLScript, LocalURL: "/runtime.js", LoadAfter: []string{"styles"}}},
			code:         "html.requirement_dependency_missing",
		},
		{
			name: "cycle",
			requirements: []HTMLRequirement{
				{ID: "one", Kind: HTMLScript, LocalURL: "/one.js", LoadAfter: []string{"two"}},
				{ID: "two", Kind: HTMLScript, LocalURL: "/two.js", LoadAfter: []string{"one"}},
			},
			code: "html.requirement_cycle",
		},
		{
			name:         "unsafe URL",
			requirements: []HTMLRequirement{{ID: "runtime", Kind: HTMLScript, LocalURL: "javascript:alert(1)"}},
			code:         "html.requirement_invalid",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := mergeHTMLRequirements(test.requirements)
			if got := diagnosticCode(err); got != test.code {
				t.Fatalf("diagnostic code = %q, want %q; err = %v", got, test.code, err)
			}
		})
	}
}

func TestHTMLRequirementsCanonicalizeInlineAssetIdentity(t *testing.T) {
	content := []byte(".margo-document{}")
	digest := sha256.Sum256(content)
	merged, err := mergeHTMLRequirements([]HTMLRequirement{
		{ID: "styles", Kind: HTMLStylesheet, Inline: AssetRef{Path: "document.css", Content: append([]byte(nil), content...)}},
		{ID: "styles", Kind: HTMLStylesheet, Inline: AssetRef{Path: "document.css", MediaType: "text/css", SHA256: hex.EncodeToString(digest[:]), Content: append([]byte(nil), content...)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	list := merged.List()
	if len(list) != 1 || list[0].Inline.MediaType != "text/css" || list[0].Inline.SHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("canonical inline identity = %#v", list)
	}
}

func TestHTMLRequirementCapabilityRoundTripsMaterializedBytes(t *testing.T) {
	requirement := HTMLRequirement{
		ID:        "margo.document.styles",
		Kind:      HTMLStylesheet,
		LocalURL:  "/margo-assets/document.css",
		LoadAfter: []string{"goshtoso.styles"},
		Inline:    requirementAsset("document.css", "text/css", ".margo-document{display:block}"),
	}
	capability, err := HTMLRequirementCapability(requirement)
	if err != nil {
		t.Fatal(err)
	}
	decoded, recognized, err := decodeHTMLRequirementCapability(capability)
	if err != nil {
		t.Fatal(err)
	}
	if !recognized || !equalHTMLRequirement(decoded, requirement) {
		t.Fatalf("decoded = %#v, recognized = %v", decoded, recognized)
	}
	decoded.Inline.Content[0] = 'X'
	again, recognized, err := decodeHTMLRequirementCapability(capability)
	if err != nil || !recognized || string(again.Inline.Content) != ".margo-document{display:block}" {
		t.Fatalf("capability aliases decoded bytes: %#v, %v", again, err)
	}
}

func TestHTMLRequirementCapabilityDecoderIsStrict(t *testing.T) {
	encode := func(data string) string {
		return htmlRequirementCapabilityPrefix + base64.RawURLEncoding.EncodeToString([]byte(data))
	}
	tests := []string{
		htmlRequirementCapabilityPrefix + "%%%",
		encode(`{"id":"styles","kind":"stylesheet","localURL":"/styles.css","unknown":true}`),
		encode(`{"id":"styles","kind":"stylesheet","localURL":"/styles.css"} {}`),
		encode(`{"id":"styles","kind":"stylesheet","inline":{"path":"styles.css","mediaType":"text/css","sha256":"bad","content":"eA=="}}`),
	}
	for _, capability := range tests {
		if _, recognized, err := decodeHTMLRequirementCapability(capability); !recognized || diagnosticCode(err) != "html.requirement_invalid" {
			t.Fatalf("capability accepted: recognized=%v err=%v", recognized, err)
		}
	}
	if _, recognized, err := decodeHTMLRequirementCapability("opaque-capability"); err != nil || recognized {
		t.Fatalf("opaque capability changed: recognized=%v err=%v", recognized, err)
	}
}

func TestExtensionRequirementsAttachOnlyWhenUsed(t *testing.T) {
	stylesCapability, err := HTMLRequirementCapability(HTMLRequirement{
		ID: "demo.styles", Kind: HTMLStylesheet, LocalURL: "/demo/styles.css", LoadAfter: []string{"margo.document.styles"},
	})
	if err != nil {
		t.Fatal(err)
	}
	runtimeCapability, err := HTMLRequirementCapability(HTMLRequirement{
		ID: "demo.runtime", Kind: HTMLScript, LocalURL: "/demo/runtime.js", LoadAfter: []string{"demo.styles"},
	})
	if err != nil {
		t.Fatal(err)
	}
	compiler := New(WithExtension(ExtensionRegistration{
		Identity: ExtensionIdentity{
			Name: "demo", Version: "v1",
			Capabilities: []string{"opaque", runtimeCapability, stylesCapability},
		},
		Fences: []string{"demo"},
		Factory: func(RenderContext) (ExtensionSession, error) {
			return extensionSessionFunc(func(context.Context, ExtensionNode, io.Writer) error { return nil }), nil
		},
	}))

	unused, err := compiler.Compile(context.Background(), Source{Name: "unused.md", Content: []byte("Body\n")})
	if err != nil {
		t.Fatal(err)
	}
	requireRequirementIDs(t, unused.editorialHTMLRequirements(), "goshtoso.styles", "margo.document.styles")

	used, err := compiler.Compile(context.Background(), Source{Name: "used.md", Content: []byte("```demo\nfirst\n```\n\n```demo\nsecond\n```\n")})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.Render(context.Background(), used)
	if err != nil {
		t.Fatal(err)
	}
	list := result.editorialHTMLRequirements().List()
	if len(list) != 4 || list[0].ID != "goshtoso.styles" || list[1].ID != "margo.document.styles" || list[2].ID != "demo.styles" || list[3].ID != "demo.runtime" {
		t.Fatalf("used requirements = %#v", list)
	}
	list[3].LoadAfter[0] = "mutated"
	if result.editorialHTMLRequirements().List()[3].LoadAfter[0] != "demo.styles" {
		t.Fatal("result requirements alias caller")
	}
}

func TestRegistryRejectsMalformedHTMLRequirementCapability(t *testing.T) {
	config := newCompilerConfig()
	err := WithExtension(ExtensionRegistration{
		Identity: ExtensionIdentity{
			Name: "broken", Version: "v1",
			Capabilities: []string{htmlRequirementCapabilityPrefix + "%%%"},
		},
		Factory: func(RenderContext) (ExtensionSession, error) { return &testExtensionSession{}, nil },
	})(&config)
	if got := diagnosticCode(err); got != "html.requirement_invalid" {
		t.Fatalf("diagnostic code = %q, err = %v", got, err)
	}
}
