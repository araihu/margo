package margo

import (
	"context"
	"strings"
	"testing"
)

func testIframePolicy() Policy {
	return Policy{
		RawHTML: RawHTMLDeny, OutputBytes: MaxOutputBytes,
		Iframe: &IframePolicy{
			AllowedOrigins: []string{"https://video.example.com"},
			Sandbox:        []SandboxToken{SandboxAllowScripts, SandboxAllowPresentation},
			Projections: TargetProjections{
				HTML: ProjectionInteractive, Site: ProjectionInteractive,
				PDF: ProjectionStaticLink, Deck: ProjectionDeny,
			},
		},
	}
}

func TestNaturalIframeIsCanonicalAndTargetProjected(t *testing.T) {
	compiler := New(WithHostPolicy(testIframePolicy()))
	source := Source{Name: "embed.md", Content: []byte(`<iframe height="450" src="https://video.example.com/watch/123" title="Architecture overview" width="800">
</iframe>
`)}
	document, err := compiler.Compile(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}

	htmlResult, err := compiler.Render(context.Background(), document, WithRenderTarget(TargetHTML))
	if err != nil {
		t.Fatal(err)
	}
	interactive := renderComponent(t, htmlResult.Content())
	for _, want := range []string{
		`<iframe class="margo-embed__frame"`,
		`src="https://video.example.com/watch/123"`,
		`title="Architecture overview"`, `width="800"`, `height="450"`,
		`sandbox="allow-presentation allow-scripts"`, `referrerpolicy="no-referrer"`,
		`allow=""`,
	} {
		if !strings.Contains(interactive, want) {
			t.Fatalf("interactive iframe missing %q: %s", want, interactive)
		}
	}
	if strings.Contains(interactive, `height="450" src=`) {
		t.Fatalf("source attribute order leaked instead of canonical projection: %s", interactive)
	}

	pdfResult, err := compiler.Render(context.Background(), document, WithRenderTarget(TargetPDF))
	if err != nil {
		t.Fatal(err)
	}
	static := renderComponent(t, pdfResult.Content())
	if strings.Contains(static, "<iframe") || !strings.Contains(static, `<a class="margo-embed__link" href="https://video.example.com/watch/123"`) {
		t.Fatalf("PDF projection is not a static link: %s", static)
	}

	if _, err := compiler.Render(context.Background(), document, WithRenderTarget(TargetDeck)); err == nil || !strings.Contains(err.Error(), "policy.iframe_denied") {
		t.Fatalf("deck deny projection error = %v", err)
	}
}

func TestNaturalIframeRejectsUnauthorizedOrUnsupportedInput(t *testing.T) {
	for _, source := range []string{
		`<iframe src="https://other.example.com/watch" title="Other"></iframe>`,
		`<iframe src="http://video.example.com/watch" title="HTTP"></iframe>`,
		`<iframe src="https://user:pass@video.example.com/watch" title="Credentials"></iframe>`,
		`<iframe src="https://video.example.com/watch" title="Extra" allow="fullscreen"></iframe>`,
		`<div><iframe src="https://video.example.com/watch" title="Nested"></iframe></div>`,
		`<iframe src="https://video.example.com/watch" title="Bad width" width="0"></iframe>`,
		`<iframe src="https://video.example.com/watch" title="Bad height" height="4097"></iframe>`,
	} {
		_, err := New(WithHostPolicy(testIframePolicy())).Compile(context.Background(), Source{Name: "invalid.md", Content: []byte(source)})
		if err == nil {
			t.Fatalf("invalid iframe compiled: %s", source)
		}
	}
}

func TestNaturalIframePolicyRejectsWildcardAndDuplicateCanonicalOrigins(t *testing.T) {
	for _, origins := range [][]string{
		{"https://*.example.com"},
		{"https://2130706433"},
		{"https://video.example.com", "https://video.example.com:443/"},
	} {
		policy := testIframePolicy()
		policy.Iframe.AllowedOrigins = origins
		_, err := New(WithHostPolicy(policy)).Compile(context.Background(), Source{Name: "plain.md", Content: []byte("# Plain\n")})
		if got := diagnosticCode(err); got != "policy.iframe_invalid" {
			t.Fatalf("origins %v diagnostic = %q, error = %v", origins, got, err)
		}
	}
}

func TestNaturalIframeNeedsHostPolicy(t *testing.T) {
	_, err := New().Compile(context.Background(), Source{Name: "denied.md", Content: []byte(`<iframe src="https://video.example.com/watch" title="Denied"></iframe>`)})
	if got := diagnosticCode(err); got != "policy.iframe_denied" {
		t.Fatalf("diagnostic = %q, error = %v", got, err)
	}
}

func TestCheckWarnsWhenIframeTitleIsMissing(t *testing.T) {
	diagnostics, err := Check(context.Background(), Source{Name: "untitled.md", Content: []byte(`<iframe src="https://video.example.com/watch"></iframe>`)}, WithCheckPolicy(testIframePolicy()), WithCheckTarget(TargetHTML))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, diagnostic := range diagnostics {
		found = found || (diagnostic.Code == "check.iframe_title_missing" && diagnostic.Severity == SeverityWarning)
	}
	if !found {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}
