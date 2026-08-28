package margo

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestDocumentSecurityNamespaceIsRejectedBeforePolicyEvaluation(t *testing.T) {
	compiler := New(WithHostPolicy(Policy{RawHTML: RawHTMLDeny, OutputBytes: 4096}))
	_, err := compiler.Compile(context.Background(), Source{Name: "x.md", Content: []byte("---\ngoshtoso:\n  security:\n    rawHTML: sanitized\n---\n<span>ok</span>")})
	if got := diagnosticCode(err); got != "frontmatter.goshtoso_removed" {
		t.Fatalf("diagnostic code = %q, err = %v", got, err)
	}
}

func TestInputCeilingRunsBeforeMarkdownParsing(t *testing.T) {
	compiler := New(WithHostPolicy(Policy{InputBytes: 32, OutputBytes: MaxOutputBytes}))
	_, err := compiler.Compile(context.Background(), Source{Name: "large.md", Content: bytes.Repeat([]byte{'x'}, 33)})
	if got := diagnosticCode(err); got != "policy.resource.document_too_large" {
		t.Fatalf("diagnostic = %q, error = %v", got, err)
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

func TestRawHTMLFailsClosedWithoutHostGrant(t *testing.T) {
	for _, markup := range []string{"<span>raw</span>", "<script>alert(1)</script>\n"} {
		_, err := New().Compile(context.Background(), Source{Name: "x.md", Content: []byte(markup)})
		if got := diagnosticCode(err); got != "policy.raw_html.denied" {
			t.Fatalf("markup %q diagnostic code = %q, err = %v", markup, got, err)
		}
	}
}

func TestUnsafeHTMLOptInPassesThroughArbitraryMarkup(t *testing.T) {
	source := Source{Name: "unsafe.md", Content: []byte("---\nlanguage: en\n---\n\n<div data-example=\"yes\"><iframe src=\"preview.html\" allow=\"fullscreen\">fallback</iframe><script>window.example = true</script></div>")}
	compiler := New(WithUnsafeHTML())
	document, err := compiler.Compile(context.Background(), source)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	result, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	markup := renderComponent(t, result.Content())
	for _, want := range []string{
		`<div data-example="yes">`,
		`<iframe src="preview.html" allow="fullscreen">fallback</iframe>`,
		`<script>window.example = true</script>`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("unsafe markup missing %q: %s", want, markup)
		}
	}

	checkDiagnostics, err := Check(context.Background(), source, WithCheckUnsafeHTML())
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(checkDiagnostics) != 0 {
		t.Fatalf("unsafe check diagnostics = %+v", checkDiagnostics)
	}
}
