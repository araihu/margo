package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
	margoembed "github.com/araihu/margo/embed"
)

func TestParsePolicyDocumentNormalizesAndHashesExactCapabilities(t *testing.T) {
	input := []byte(`{
  "schemaVersion": "margo-policy/v1",
  "rawHTML": "sanitized",
  "trustedEmbeds": {
    "allowedKinds": ["video", "iframe"],
    "allowedOrigins": ["https://video.example.com/", "https://media.example.com"],
    "iframeSandbox": ["allow-scripts", "allow-presentation"],
    "projections": {
      "html": "interactive",
      "pdf": "static-link",
      "site": "interactive",
      "deck": "deny"
    }
  }
}`)
	policy, err := parsePolicyDocument(input)
	if err != nil {
		t.Fatal(err)
	}
	if policy.Host != (margo.Policy{RawHTML: margo.RawHTMLSanitized, OutputBytes: margo.MaxOutputBytes}) {
		t.Fatalf("host policy = %+v", policy.Host)
	}
	if policy.Digest != "sha256:3614aded7db067ed69d87ee913f5250400d54d4f12e17883648a138fec8ef93d" {
		t.Fatalf("policy digest = %q", policy.Digest)
	}
	for target, want := range map[policyTarget]margoembed.Projection{
		policyTargetHTML: margoembed.ProjectionInteractive,
		policyTargetPDF:  margoembed.ProjectionStaticLink,
		policyTargetSite: margoembed.ProjectionInteractive,
		policyTargetDeck: margoembed.ProjectionDeny,
	} {
		got, ok := policy.EmbedPolicy(target)
		if !ok || got.Projection != want {
			t.Fatalf("target %q policy = %+v, present=%t", target, got, ok)
		}
	}
}

func TestParsePolicyDocumentRejectsAmbiguousOrOverbroadInput(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{name: "unknown field", input: []byte(`{"schemaVersion":"margo-policy/v1","unexpected":true}`)},
		{name: "multiple objects", input: []byte(`{"schemaVersion":"margo-policy/v1"}{"schemaVersion":"margo-policy/v1"}`)},
		{name: "oversized", input: bytes.Repeat([]byte(" "), maxPolicyBytes+1)},
		{name: "invalid origin shape", input: []byte(`{"schemaVersion":"margo-policy/v1","trustedEmbeds":{"allowedKinds":["iframe"],"allowedOrigins":["https://video.example.com/path"],"projections":{"html":"interactive"}}}`)},
		{name: "non-deny missing kinds", input: []byte(`{"schemaVersion":"margo-policy/v1","trustedEmbeds":{"allowedOrigins":["https://video.example.com"],"projections":{"html":"interactive"}}}`)},
		{name: "non-deny missing origins", input: []byte(`{"schemaVersion":"margo-policy/v1","trustedEmbeds":{"allowedKinds":["iframe"],"projections":{"pdf":"static-link"}}}`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parsePolicyDocument(test.input); cliDiagnosticCode(err) != "cli.policy_invalid" {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestPolicyDigestIsIndependentOfAllowlistOrdering(t *testing.T) {
	left, err := parsePolicyDocument([]byte(`{"schemaVersion":"margo-policy/v1","trustedEmbeds":{"allowedKinds":["video","iframe"],"allowedOrigins":["https://video.example.com","https://media.example.com"],"projections":{"html":"interactive"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	right, err := parsePolicyDocument([]byte(`{"trustedEmbeds":{"projections":{"html":"interactive"},"allowedOrigins":["https://media.example.com/","https://video.example.com/"],"allowedKinds":["iframe","video"]},"schemaVersion":"margo-policy/v1"}`))
	if err != nil {
		t.Fatal(err)
	}
	if left.Digest != right.Digest {
		t.Fatalf("equivalent policies have different identities: %q != %q", left.Digest, right.Digest)
	}
}

func TestPolicyFlagsRejectStdinAsPolicyAuthority(t *testing.T) {
	_, err := (policyFlags{Path: "-"}).load(context.Background(), nil)
	if cliDiagnosticCode(err) != "cli.policy_invalid" || !strings.Contains(err.Error(), "file path") {
		t.Fatalf("error = %v", err)
	}
}
