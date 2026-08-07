package margo

import (
	"context"
	"testing"
)

func TestRawHTMLRequirementCannotElevateHost(t *testing.T) {
	compiler := New(WithHostPolicy(Policy{RawHTML: RawHTMLDeny, OutputBytes: 4096}))
	_, err := compiler.Compile(context.Background(), Source{Name: "x.md", Content: []byte("---\ngoshtoso:\n  security:\n    rawHTML: sanitized\n---\n<span>ok</span>")})
	if got := diagnosticCode(err); got != "policy.raw_html.mismatch" {
		t.Fatalf("diagnostic code = %q, err = %v", got, err)
	}
}

func TestEffectiveOutputPolicyUsesHostCeiling(t *testing.T) {
	compiler := New(WithHostPolicy(Policy{RawHTML: RawHTMLDeny, OutputBytes: 4096}))
	doc, err := compiler.Compile(context.Background(), Source{Name: "x.md", Content: []byte("# x")})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if got := doc.effectivePolicy.OutputBytes; got != 4096 {
		t.Fatalf("effective output bytes = %d", got)
	}
}

func TestOutputBytesBounds(t *testing.T) {
	for _, value := range []int64{0, -1, MaxOutputBytes + 1} {
		_, err := New(WithHostPolicy(Policy{RawHTML: RawHTMLDeny, OutputBytes: value})).Compile(context.Background(), Source{Name: "x.md", Content: []byte("# x")})
		if got := diagnosticCode(err); got != "policy.output_bytes_invalid" {
			t.Fatalf("output bytes %d diagnostic = %q, err = %v", value, got, err)
		}
	}
}

func TestUndeclaredRawHTMLFailsClosed(t *testing.T) {
	_, err := New().Compile(context.Background(), Source{Name: "x.md", Content: []byte("<span>raw</span>")})
	if got := diagnosticCode(err); got != "policy.raw_html.undeclared" {
		t.Fatalf("diagnostic code = %q, err = %v", got, err)
	}
}
