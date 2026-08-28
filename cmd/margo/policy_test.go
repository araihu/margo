package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

func TestParsePolicyDocumentNormalizesAndHashesExactCapabilities(t *testing.T) {
	input := []byte(`{
  "schemaVersion": "margo-policy/v1",
  "rawHTML": "sanitized",
  "iframe": {
    "allowedOrigins": ["https://video.example.com/", "https://media.example.com"],
    "sandbox": ["allow-scripts", "allow-presentation"],
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
	if policy.Host.SchemaVersion != "margo-policy/v1" || policy.Host.RawHTML != margo.RawHTMLSanitized || policy.Host.InputBytes != margo.MaxDocumentBytes || policy.Host.OutputBytes != margo.MaxOutputBytes || policy.Host.Iframe == nil {
		t.Fatalf("host policy = %+v", policy.Host)
	}
	if !strings.HasPrefix(policy.Digest, "sha256:") || len(policy.Digest) != len("sha256:")+64 {
		t.Fatalf("policy digest = %q", policy.Digest)
	}
	for target, want := range map[policyTarget]margo.Projection{
		policyTargetHTML: margo.ProjectionInteractive,
		policyTargetPDF:  margo.ProjectionStaticLink,
		policyTargetSite: margo.ProjectionInteractive,
		policyTargetDeck: margo.ProjectionDeny,
	} {
		got := map[policyTarget]margo.Projection{
			policyTargetHTML: policy.Host.Iframe.Projections.HTML,
			policyTargetPDF:  policy.Host.Iframe.Projections.PDF,
			policyTargetSite: policy.Host.Iframe.Projections.Site,
			policyTargetDeck: policy.Host.Iframe.Projections.Deck,
		}[target]
		if got != want {
			t.Fatalf("target %q projection = %q", target, got)
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
		{name: "invalid origin shape", input: []byte(`{"schemaVersion":"margo-policy/v1","iframe":{"allowedOrigins":["https://video.example.com/path"],"projections":{"html":"interactive"}}}`)},
		{name: "missing origins", input: []byte(`{"schemaVersion":"margo-policy/v1","iframe":{"projections":{"pdf":"static-link"}}}`)},
		{name: "unsupported iframe field", input: []byte(`{"schemaVersion":"margo-policy/v1","iframe":{"allowedOrigins":["https://media.example.com"],"allow":["fullscreen"]}}`)},
		{name: "interactive PDF", input: []byte(`{"schemaVersion":"margo-policy/v1","iframe":{"allowedOrigins":["https://media.example.com"],"projections":{"pdf":"interactive"}}}`)},
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
	left, err := parsePolicyDocument([]byte(`{"schemaVersion":"margo-policy/v1","iframe":{"allowedOrigins":["https://video.example.com","https://media.example.com"],"projections":{"html":"static-link"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	right, err := parsePolicyDocument([]byte(`{"iframe":{"projections":{"html":"static-link"},"allowedOrigins":["https://media.example.com/","https://video.example.com/"]},"schemaVersion":"margo-policy/v1"}`))
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

func TestPolicyFlagsAllowUnsafeHTMLWithoutPolicyFile(t *testing.T) {
	policy, err := (policyFlags{AllowUnsafeHTML: true}).load(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if policy == nil || !policy.AllowUnsafeHTML || policy.Digest != "" {
		t.Fatalf("loaded policy = %+v", policy)
	}
}
