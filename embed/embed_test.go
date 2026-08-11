package embed_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	margo "github.com/araihu/margo"
	margoembed "github.com/araihu/margo/embed"
)

func TestExtensionRendersInteractiveIframeFromTypedAllowedRequest(t *testing.T) {
	registration := margoembed.Extension(margoembed.Policy{
		Projection:     margoembed.ProjectionInteractive,
		AllowedKinds:   []margoembed.Kind{margoembed.KindIframe},
		AllowedOrigins: []string{"https://video.example.com"},
		IframeSandbox: []margoembed.SandboxToken{
			margoembed.SandboxAllowPresentation,
			margoembed.SandboxAllowScripts,
		},
		ReferrerPolicy: margoembed.ReferrerNoReferrer,
	})
	compiler := margo.New(margo.WithExtension(registration))
	source := margo.Source{Name: "embed.md", Content: []byte("```trusted-embed\nkind: iframe\nurl: https://video.example.com/watch/123\ntitle: Architecture overview\nwidth: 800\nheight: 450\n```\n")}
	document, err := compiler.Compile(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := rendered.Content().Render(context.Background(), &output); err != nil {
		t.Fatal(err)
	}
	markup := output.String()
	for _, want := range []string{
		`<iframe class="margo-trusted-embed__frame"`,
		`src="https://video.example.com/watch/123"`,
		`title="Architecture overview"`,
		`width="800"`, `height="450"`,
		`sandbox="allow-presentation allow-scripts"`,
		`referrerpolicy="no-referrer"`,
		`loading="lazy"`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("interactive embed missing %q: %s", want, markup)
		}
	}
}

func TestExtensionProjectsAuthorizedEmbedAsStaticLink(t *testing.T) {
	markup, err := render(t, margoembed.Policy{
		Projection:     margoembed.ProjectionStaticLink,
		AllowedKinds:   []margoembed.Kind{margoembed.KindIframe},
		AllowedOrigins: []string{"https://video.example.com"},
	}, "kind: iframe\nurl: https://video.example.com/watch/123\ntitle: Architecture overview\n")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(markup, "<iframe") {
		t.Fatalf("static projection contains iframe: %s", markup)
	}
	for _, want := range []string{
		`<a class="margo-trusted-embed__link" href="https://video.example.com/watch/123"`,
		`rel="noreferrer"`,
		`referrerpolicy="no-referrer"`,
		`>Architecture overview</a>`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("static embed missing %q: %s", want, markup)
		}
	}
}

func TestExtensionFailsClosedWhenProjectionDeniesEmbeds(t *testing.T) {
	_, err := render(t, margoembed.Policy{
		Projection:     margoembed.ProjectionDeny,
		AllowedKinds:   []margoembed.Kind{margoembed.KindIframe},
		AllowedOrigins: []string{"https://video.example.com"},
	}, "kind: iframe\nurl: https://video.example.com/watch/123\ntitle: Architecture overview\n")
	if code := diagnosticCode(err); code != "embed.policy_denied" {
		t.Fatalf("diagnostic = %q, error = %v", code, err)
	}
}

func TestNormalizePolicyRejectsInteractiveVideoWithoutEnforceableReferrer(t *testing.T) {
	_, err := margoembed.NormalizePolicy(margoembed.Policy{
		Projection:     margoembed.ProjectionInteractive,
		AllowedKinds:   []margoembed.Kind{margoembed.KindVideo},
		AllowedOrigins: []string{"https://media.example.com"},
	})
	if err == nil || !strings.Contains(err.Error(), "embed.policy_invalid") || !strings.Contains(err.Error(), "video") {
		t.Fatalf("error = %v", err)
	}
}

func TestExtensionProjectsAuthorizedVideoAsNoreferrerStaticLink(t *testing.T) {
	markup, err := render(t, margoembed.Policy{
		Projection:     margoembed.ProjectionStaticLink,
		AllowedKinds:   []margoembed.Kind{margoembed.KindVideo},
		AllowedOrigins: []string{"https://media.example.com"},
	}, "kind: video\nurl: https://media.example.com/demo.mp4\ntitle: Product demonstration\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`href="https://media.example.com/demo.mp4"`, `rel="noreferrer"`, `referrerpolicy="no-referrer"`} {
		if !strings.Contains(markup, want) {
			t.Fatalf("static video link missing %q: %s", want, markup)
		}
	}
}

func TestCheckExtensionUsesSameOriginPolicyBeforeRendering(t *testing.T) {
	registration := margoembed.Extension(margoembed.Policy{
		Projection:     margoembed.ProjectionInteractive,
		AllowedKinds:   []margoembed.Kind{margoembed.KindIframe},
		AllowedOrigins: []string{"https://video.example.com"},
	})
	source := margo.Source{Name: "embed.md", Content: []byte("---\nlanguage: en\n---\n\n```trusted-embed\nkind: iframe\nurl: https://evil.example/watch\ntitle: Untrusted origin\n```\n")}
	diagnostics, err := margo.Check(context.Background(), source, margo.WithCheckExtension(registration))
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "embed.origin_denied" || diagnostics[0].Source != "embed.md" || diagnostics[0].Line != 6 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestNormalizePolicyRejectsCapabilitiesWithoutAllowlist(t *testing.T) {
	tests := []struct {
		name   string
		policy margoembed.Policy
	}{
		{
			name: "missing kinds",
			policy: margoembed.Policy{
				Projection: margoembed.ProjectionInteractive, AllowedOrigins: []string{"https://video.example.com"},
			},
		},
		{
			name: "missing origins",
			policy: margoembed.Policy{
				Projection: margoembed.ProjectionStaticLink, AllowedKinds: []margoembed.Kind{margoembed.KindIframe},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := margoembed.NormalizePolicy(test.policy); err == nil || !strings.Contains(err.Error(), "embed.policy_invalid") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestExtensionRejectsUnsafeRequests(t *testing.T) {
	policy := margoembed.Policy{
		Projection:     margoembed.ProjectionInteractive,
		AllowedKinds:   []margoembed.Kind{margoembed.KindIframe},
		AllowedOrigins: []string{"https://video.example.com"},
	}
	tests := []struct {
		name    string
		payload string
		code    string
	}{
		{name: "http URL", payload: "kind: iframe\nurl: http://video.example.com/watch\ntitle: Video\n", code: "embed.url_invalid"},
		{name: "credentials", payload: "kind: iframe\nurl: https://user@video.example.com/watch\ntitle: Video\n", code: "embed.url_invalid"},
		{name: "invalid port", payload: "kind: iframe\nurl: https://video.example.com:bad/watch\ntitle: Video\n", code: "embed.url_invalid"},
		{name: "denied origin", payload: "kind: iframe\nurl: https://evil.example/watch\ntitle: Video\n", code: "embed.origin_denied"},
		{name: "missing title", payload: "kind: iframe\nurl: https://video.example.com/watch\n", code: "embed.title_invalid"},
		{name: "oversized dimensions", payload: "kind: iframe\nurl: https://video.example.com/watch\ntitle: Video\nwidth: 4097\n", code: "embed.dimensions_invalid"},
		{name: "unknown field", payload: "kind: iframe\nurl: https://video.example.com/watch\ntitle: Video\nautoplay: true\n", code: "embed.request_invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := render(t, policy, test.payload)
			if got := diagnosticCode(err); got != test.code {
				t.Fatalf("diagnostic = %q, want %q, error = %v", got, test.code, err)
			}
		})
	}
}

func TestExtensionEscapesTrustedRequestValues(t *testing.T) {
	markup, err := render(t, margoembed.Policy{
		Projection:     margoembed.ProjectionInteractive,
		AllowedKinds:   []margoembed.Kind{margoembed.KindIframe},
		AllowedOrigins: []string{"https://video.example.com"},
	}, "kind: iframe\nurl: 'https://video.example.com/watch?q=\"unsafe\"&next=<x>'\ntitle: '\" onload=\"alert(1)'\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{` onload="alert(1)`, `next=<x>`} {
		if strings.Contains(markup, forbidden) {
			t.Fatalf("unescaped request value %q in %s", forbidden, markup)
		}
	}
	for _, want := range []string{`q=&#34;unsafe&#34;&amp;next=&lt;x&gt;`, `title="&#34; onload=&#34;alert(1)"`} {
		if !strings.Contains(markup, want) {
			t.Fatalf("escaped markup missing %q: %s", want, markup)
		}
	}
}

func TestExtensionRegistrationIsSafeForConcurrentReuse(t *testing.T) {
	registration := margoembed.Extension(margoembed.Policy{
		Projection:     margoembed.ProjectionInteractive,
		AllowedKinds:   []margoembed.Kind{margoembed.KindIframe},
		AllowedOrigins: []string{"https://video.example.com"},
	})
	compiler := margo.New(margo.WithExtension(registration))
	document, err := compiler.Compile(context.Background(), margo.Source{
		Name: "embed.md", Content: []byte("```trusted-embed\nkind: iframe\nurl: https://video.example.com/watch/123\ntitle: Architecture overview\n```\n"),
	})
	if err != nil {
		t.Fatal(err)
	}

	const workers = 24
	failures := make(chan error, workers)
	var group sync.WaitGroup
	for index := 0; index < workers; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			result, renderErr := compiler.Render(context.Background(), document)
			if renderErr != nil {
				failures <- renderErr
				return
			}
			var output bytes.Buffer
			if renderErr = result.Content().Render(context.Background(), &output); renderErr != nil {
				failures <- renderErr
				return
			}
			if !strings.Contains(output.String(), `src="https://video.example.com/watch/123"`) {
				failures <- errors.New("rendered embed changed under concurrent reuse")
			}
		}()
	}
	group.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
}

func TestPolicyConfigurationHashBindsProjectionAndOrigin(t *testing.T) {
	base := margoembed.Policy{
		Projection:     margoembed.ProjectionInteractive,
		AllowedKinds:   []margoembed.Kind{margoembed.KindIframe},
		AllowedOrigins: []string{"https://video.example.com"},
	}
	first, err := base.ConfigurationHash()
	if err != nil {
		t.Fatal(err)
	}
	changedProjection := base
	changedProjection.Projection = margoembed.ProjectionStaticLink
	second, err := changedProjection.ConfigurationHash()
	if err != nil {
		t.Fatal(err)
	}
	changedOrigin := base
	changedOrigin.AllowedOrigins = []string{"https://media.example.com"}
	third, err := changedOrigin.ConfigurationHash()
	if err != nil {
		t.Fatal(err)
	}
	if first == second || first == third || second == third {
		t.Fatalf("distinct capabilities share a hash: %q %q %q", first, second, third)
	}
}

func TestPolicyCanonicalizesDefaultHTTPSPortForMatchingAndHashing(t *testing.T) {
	withoutPort := margoembed.Policy{
		Projection:     margoembed.ProjectionStaticLink,
		AllowedKinds:   []margoembed.Kind{margoembed.KindIframe},
		AllowedOrigins: []string{"https://video.example.com"},
	}
	withPort := withoutPort
	withPort.AllowedOrigins = []string{"https://video.example.com:443"}
	first, err := withoutPort.ConfigurationHash()
	if err != nil {
		t.Fatal(err)
	}
	second, err := withPort.ConfigurationHash()
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("browser-equivalent origins have different hashes: %q != %q", first, second)
	}

	for _, test := range []struct {
		name   string
		policy margoembed.Policy
		url    string
	}{
		{name: "policy has port", policy: withPort, url: "https://video.example.com/watch"},
		{name: "request has port", policy: withoutPort, url: "https://video.example.com:443/watch"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := render(t, test.policy, "kind: iframe\nurl: "+test.url+"\ntitle: Video\n")
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPolicyCanonicalizesBrowserEquivalentHostsAndRejectsWildcards(t *testing.T) {
	base := margoembed.Policy{
		Projection:   margoembed.ProjectionStaticLink,
		AllowedKinds: []margoembed.Kind{margoembed.KindIframe},
	}
	for _, origin := range []string{"https://*.example.com", "https://127.1"} {
		policy := base
		policy.AllowedOrigins = []string{origin}
		if _, err := margoembed.NormalizePolicy(policy); err == nil || !strings.Contains(err.Error(), "embed.policy_invalid") {
			t.Fatalf("noncanonical origin %q error = %v", origin, err)
		}
	}

	equivalent := []struct {
		name  string
		left  string
		right string
	}{
		{name: "IDN", left: "https://bücher.example", right: "https://xn--bcher-kva.example"},
		{name: "IPv6", left: "https://[2001:0db8:0:0:0:0:0:1]", right: "https://[2001:db8::1]"},
	}
	for _, test := range equivalent {
		t.Run(test.name, func(t *testing.T) {
			left := base
			left.AllowedOrigins = []string{test.left}
			right := base
			right.AllowedOrigins = []string{test.right}
			leftHash, err := left.ConfigurationHash()
			if err != nil {
				t.Fatal(err)
			}
			rightHash, err := right.ConfigurationHash()
			if err != nil {
				t.Fatal(err)
			}
			if leftHash != rightHash {
				t.Fatalf("browser-equivalent origins have different hashes: %q != %q", leftHash, rightHash)
			}
		})
	}

	policy := base
	policy.AllowedOrigins = []string{"https://bücher.example"}
	if _, err := render(t, policy, "kind: iframe\nurl: https://xn--bcher-kva.example/watch\ntitle: Video\n"); err != nil {
		t.Fatalf("IDN request did not match canonical policy origin: %v", err)
	}
}

func TestExtensionBoundsRequestURLLength(t *testing.T) {
	policy := margoembed.Policy{
		Projection:     margoembed.ProjectionStaticLink,
		AllowedKinds:   []margoembed.Kind{margoembed.KindIframe},
		AllowedOrigins: []string{"https://video.example.com"},
	}
	const limit = 4096
	prefix := "https://video.example.com/"
	accepted := prefix + strings.Repeat("a", limit-len(prefix))
	if _, err := render(t, policy, "kind: iframe\nurl: "+accepted+"\ntitle: Video\n"); err != nil {
		t.Fatalf("URL at documented limit rejected: %v", err)
	}
	rejected := accepted + "a"
	if _, err := render(t, policy, "kind: iframe\nurl: "+rejected+"\ntitle: Video\n"); diagnosticCode(err) != "embed.url_invalid" {
		t.Fatalf("oversized URL error = %v", err)
	}
}

func render(t *testing.T, policy margoembed.Policy, payload string) (string, error) {
	t.Helper()
	compiler := margo.New(margo.WithExtension(margoembed.Extension(policy)))
	document, err := compiler.Compile(context.Background(), margo.Source{Name: "embed.md", Content: []byte("```trusted-embed\n" + payload + "```\n")})
	if err != nil {
		return "", err
	}
	result, err := compiler.Render(context.Background(), document)
	if err != nil {
		return "", err
	}
	var output bytes.Buffer
	if err := result.Content().Render(context.Background(), &output); err != nil {
		return "", err
	}
	return output.String(), nil
}

func diagnosticCode(err error) string {
	var diagnostic *margo.DiagnosticError
	if errors.As(err, &diagnostic) && len(diagnostic.Diagnostics) > 0 {
		return diagnostic.Diagnostics[0].Code
	}
	return ""
}
