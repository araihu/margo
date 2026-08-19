package chromium

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/deck"
	"github.com/araihu/margo/pdf"
)

func TestNewRejectsMissingExecutable(t *testing.T) {
	_, err := New(Config{ExecutablePath: filepath.Join(t.TempDir(), "missing")})
	if code(err) != "pdf.chromium.path_invalid" {
		t.Fatalf("New() error = %v", err)
	}
}

func TestExportRejectsInvalidRequestBeforeLaunch(t *testing.T) {
	path := fakeExecutable(t)
	engine, err := New(Config{ExecutablePath: path})
	if err != nil {
		t.Fatal(err)
	}
	_, err = engine.Export(context.Background(), pdf.Request{})
	if code(err) != "pdf.request_invalid" {
		t.Fatalf("Export() error = %v", err)
	}
}

func TestOfflineHTMLAllowsLinksButRejectsRenderTimeNetworkAssets(t *testing.T) {
	if err := validateOfflineHTML([]byte(`<p>Read <a href="https://example.com/post">the post</a>.</p><img src="data:image/png;base64,AA==">`)); err != nil {
		t.Fatalf("offline document rejected: %v", err)
	}
	for _, document := range []string{
		`<img src="https://example.com/image.png">`,
		`<script src="/app.js"></script>`,
		`<style>.hero{background:url(https://example.com/hero.webp)}</style>`,
	} {
		if err := validateOfflineHTML([]byte(document)); code(err) != "pdf.network_forbidden" {
			t.Fatalf("validateOfflineHTML(%q) error = %v", document, err)
		}
	}
}

func TestRewriteDocumentLinksUsesSafeDeterministicPolicies(t *testing.T) {
	document := []byte(`<html><body>
<a id="relative" href="guides/start.md?mode=full#install">Guide</a>
<a id="root" href="/reference/index.md">Reference</a>
<a id="fragment" href="#local">Local</a>
<a id="external" href="https://example.com/docs">External</a>
<a id="mail" href="mailto:docs@example.com">Mail</a>
<a id="phone" href="tel:+15551234567">Phone</a>
</body></html>`)

	tests := []struct {
		name    string
		policy  pdf.RelativeLinkPolicy
		baseURL string
		want    []string
		absent  []string
	}{
		{
			name:   "safe default strips relative targets",
			policy: "",
			want: []string{
				`id="relative"`, `id="root"`, `href="#local"`,
				`href="https://example.com/docs"`, `href="mailto:docs@example.com"`, `href="tel:+15551234567"`,
			},
			absent: []string{`href="guides/start.md?mode=full#install"`, `href="/reference/index.md"`},
		},
		{
			name:   "explicit keep preserves relative targets",
			policy: pdf.RelativeLinksKeep,
			want:   []string{`href="guides/start.md?mode=full#install"`, `href="/reference/index.md"`},
		},
		{
			name:    "resolve uses public base URL",
			policy:  pdf.RelativeLinksResolve,
			baseURL: "https://docs.example.com/manual/",
			want: []string{
				`href="https://docs.example.com/manual/guides/start.md?mode=full#install"`,
				`href="https://docs.example.com/reference/index.md"`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := rewriteDocumentLinks(document, test.policy, test.baseURL)
			if err != nil {
				t.Fatal(err)
			}
			markup := string(got)
			for _, want := range test.want {
				if !strings.Contains(markup, want) {
					t.Errorf("rewritten document missing %q: %s", want, markup)
				}
			}
			for _, absent := range test.absent {
				if strings.Contains(markup, absent) {
					t.Errorf("rewritten document retained %q: %s", absent, markup)
				}
			}
		})
	}
}

func TestRewriteDocumentLinksRejectsUnsafePolicyConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		document string
		policy   pdf.RelativeLinkPolicy
		baseURL  string
		code     string
	}{
		{name: "error policy finds relative link", policy: pdf.RelativeLinksError, code: "pdf.relative_link_forbidden"},
		{name: "resolve requires base URL", policy: pdf.RelativeLinksResolve, code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects local base URL", policy: pdf.RelativeLinksResolve, baseURL: "http://127.0.0.1:9000/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects shortened loopback base", policy: pdf.RelativeLinksResolve, baseURL: "http://127.1/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects empty hexadecimal loopback part", policy: pdf.RelativeLinksResolve, baseURL: "http://127.0x/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects split hexadecimal loopback base", policy: pdf.RelativeLinksResolve, baseURL: "http://0x7f.0x/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects octal loopback with empty hexadecimal part", policy: pdf.RelativeLinksResolve, baseURL: "http://0177.0x/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects integer loopback base", policy: pdf.RelativeLinksResolve, baseURL: "http://2130706433/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects hexadecimal loopback base", policy: pdf.RelativeLinksResolve, baseURL: "http://0x7f000001/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects invalid octal numeric base", policy: pdf.RelativeLinksResolve, baseURL: "http://09/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects too many IPv4 parts", policy: pdf.RelativeLinksResolve, baseURL: "http://1.2.3.4.5/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects mixed domain ending in number", policy: pdf.RelativeLinksResolve, baseURL: "http://example.255/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects unspecified IPv4 base", policy: pdf.RelativeLinksResolve, baseURL: "http://0/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects unspecified IPv6 base", policy: pdf.RelativeLinksResolve, baseURL: "http://[::]/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects trailing-dot localhost base", policy: pdf.RelativeLinksResolve, baseURL: "http://localhost./docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects trailing-dot localhost subdomain", policy: pdf.RelativeLinksResolve, baseURL: "http://x.localhost./docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects ideographic-dot localhost", policy: pdf.RelativeLinksResolve, baseURL: "http://localhost。/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects fullwidth-dot localhost subdomain", policy: pdf.RelativeLinksResolve, baseURL: "http://x.localhost．/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects halfwidth-dot loopback", policy: pdf.RelativeLinksResolve, baseURL: "http://127｡1/docs/", code: "pdf.relative_link_base_invalid"},
		{name: "resolve rejects empty base hostname", policy: pdf.RelativeLinksResolve, baseURL: "https://:443/manual/", code: "pdf.relative_link_base_invalid"},
		{name: "unknown policy", policy: pdf.RelativeLinkPolicy("surprise"), code: "pdf.relative_link_policy_invalid"},
		{name: "active link scheme", document: `<a href="javascript:alert(1)">Run</a>`, code: "pdf.link_scheme_forbidden"},
		{name: "network-path link", document: `<a href="//example.com/docs">Docs</a>`, code: "pdf.relative_link_invalid"},
		{name: "backslash network-path link", document: `<a href="\\evil.example/docs">Docs</a>`, policy: pdf.RelativeLinksKeep, code: "pdf.relative_link_invalid"},
		{name: "hostless HTTP link", document: `<a href="http:guides/start.md">Docs</a>`, code: "pdf.link_absolute_invalid"},
		{name: "single-slash HTTP link", document: `<a href="http:/guides/start.md">Docs</a>`, code: "pdf.link_absolute_invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := test.document
			if document == "" {
				document = `<a href="guide.md">Guide</a>`
			}
			_, err := rewriteDocumentLinks([]byte(document), test.policy, test.baseURL)
			if code(err) != test.code {
				t.Fatalf("error = %v, want %s", err, test.code)
			}
		})
	}
}

func TestRewriteDocumentLinksCanonicalizesNonLoopbackNumericBase(t *testing.T) {
	got, err := rewriteDocumentLinks(
		[]byte(`<a href="guide.md">Guide</a>`),
		pdf.RelativeLinksResolve,
		"http://0x08080808:8080/manual/",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `href="http://8.8.8.8:8080/manual/guide.md"`) {
		t.Fatalf("rewritten document did not canonicalize numeric base: %s", got)
	}
}

func TestBrowserIPv4AddressMatchesURLStandardNumericForms(t *testing.T) {
	tests := []struct {
		hostname    string
		wantAddress string
		wantNumeric bool
		wantValid   bool
	}{
		{hostname: "0", wantAddress: "0.0.0.0", wantNumeric: true, wantValid: true},
		{hostname: "0x", wantAddress: "0.0.0.0", wantNumeric: true, wantValid: true},
		{hostname: "0xffffffff", wantAddress: "255.255.255.255", wantNumeric: true, wantValid: true},
		{hostname: "127.0x", wantAddress: "127.0.0.0", wantNumeric: true, wantValid: true},
		{hostname: "09", wantNumeric: true, wantValid: false},
		{hostname: "example.255", wantNumeric: true, wantValid: false},
		{hostname: "1.2.3.4.5", wantNumeric: true, wantValid: false},
		{hostname: "example.com", wantNumeric: false, wantValid: true},
	}

	for _, test := range tests {
		t.Run(test.hostname, func(t *testing.T) {
			address, numeric, valid := browserIPv4Address(test.hostname)
			gotAddress := ""
			if address != nil {
				gotAddress = address.String()
			}
			if gotAddress != test.wantAddress || numeric != test.wantNumeric || valid != test.wantValid {
				t.Fatalf("browserIPv4Address(%q) = (%q, %t, %t), want (%q, %t, %t)", test.hostname, gotAddress, numeric, valid, test.wantAddress, test.wantNumeric, test.wantValid)
			}
		})
	}
}

func TestExportWithInstalledChromium(t *testing.T) {
	path := installedChromium()
	if path == "" {
		t.Skip("no installed Chromium-family browser")
	}
	engine, err := New(Config{ExecutablePath: path, Timeout: 20 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	descriptor := margo.RuntimeDescriptor{
		Protocol:            margo.RuntimeProtocolV1,
		DocumentFingerprint: margo.DocumentFingerprint{1},
		RenderInstanceID:    "ri-00000000",
		Tasks:               []margo.RuntimeTask{},
	}
	result, err := engine.Export(context.Background(), pdf.Request{
		HTML:        []byte("<!doctype html><html><body><h1>Margo Chromium E2E</h1></body></html>"),
		Runtime:     descriptor,
		ExecutionID: "chromium-e2e",
		Page:        pdf.PageConfig{Size: pdf.PageA4, Orientation: pdf.Portrait},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(result.PDF), "%PDF-") || len(result.PDF) < 1000 {
		t.Fatalf("PDF bytes = %d prefix = %q", len(result.PDF), result.PDF[:min(8, len(result.PDF))])
	}
	if err := margo.ValidateRuntimeReport(descriptor, "chromium-e2e", result.Runtime); err != nil {
		t.Fatalf("runtime report: %v", err)
	}
	if result.Engine.Name != "chromium" || result.Engine.Version == "" {
		t.Fatalf("engine info = %+v", result.Engine)
	}
}

func TestExportAwaitsPrintPreparation(t *testing.T) {
	path := installedChromium()
	if path == "" {
		t.Skip("no installed Chromium-family browser")
	}
	engine, err := New(Config{ExecutablePath: path, Timeout: 20 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	descriptor := margo.RuntimeDescriptor{
		Protocol:            margo.RuntimeProtocolV1,
		DocumentFingerprint: margo.DocumentFingerprint{3},
		RenderInstanceID:    "ri-00000002",
		Tasks:               []margo.RuntimeTask{},
	}
	_, err = engine.Export(context.Background(), pdf.Request{
		HTML: []byte(`<!doctype html><html><body><script>
window.margoPreparePrint = async () => { throw new Error("interactive chart capture failed"); };
</script><h1>Chart</h1></body></html>`),
		Runtime: descriptor, ExecutionID: "chromium-print-preparation",
		Page: pdf.PageConfig{Size: pdf.PageA4},
	})
	if code(err) != "pdf.chromium.export_failed" || !strings.Contains(err.Error(), "interactive chart capture failed") {
		t.Fatalf("Export() error = %v", err)
	}
}

func TestExportExecutesMermaidRuntimeTaskWithInstalledChromium(t *testing.T) {
	path := installedChromium()
	if path == "" {
		t.Skip("no installed Chromium-family browser")
	}
	const input = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	descriptor := margo.RuntimeDescriptor{
		Protocol:            margo.RuntimeProtocolV1,
		DocumentFingerprint: margo.DocumentFingerprint{2},
		RenderInstanceID:    "ri-00000001",
		Tasks: []margo.RuntimeTask{{
			ID:          "ri-00000001:mermaid:00000000:" + input,
			Kind:        "mermaid",
			InputSHA256: input,
			DependsOn:   []string{},
		}},
	}
	engine, err := New(Config{ExecutablePath: path, Timeout: 20 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Export(context.Background(), pdf.Request{
		HTML:    []byte(`<!doctype html><html><body><figure class="margo-runtime-task margo-mermaid" data-margo-runtime-task="mermaid" data-margo-runtime-task-ordinal="0"><div class="margo-mermaid__canvas"></div><details class="margo-mermaid__source"><summary>Source</summary><pre><code>graph TD; A--&gt;B</code></pre></details></figure></body></html>`),
		Runtime: descriptor, ExecutionID: "chromium-mermaid-e2e",
		Page: pdf.PageConfig{Size: pdf.PageA4},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := margo.ValidateRuntimeReport(descriptor, "chromium-mermaid-e2e", result.Runtime); err != nil {
		t.Fatal(err)
	}
	if len(result.Runtime.Tasks) != 1 || result.Runtime.Tasks[0].OutputBytes < 100 || result.Runtime.Tasks[0].OutputSHA256 == "" {
		t.Fatalf("runtime tasks = %+v", result.Runtime.Tasks)
	}
}

func TestExportDeckReportsObservedFontBundle(t *testing.T) {
	path := installedChromium()
	if path == "" {
		t.Skip("no installed Chromium-family browser")
	}
	result, err := deck.Render(context.Background(), margo.New(), deck.RenderInput{
		Name:     "font-check.md",
		Markdown: []byte("---\nlang: en\n---\n# Font check\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := result.RuntimeDescriptor("ri-00000042")
	if err != nil {
		t.Fatal(err)
	}
	engine, err := New(Config{ExecutablePath: path, Timeout: 20 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	report, err := engine.Export(context.Background(), pdf.Request{
		HTML:        result.HTML(),
		Runtime:     descriptor,
		ExecutionID: "chromium-font-e2e",
		Page: pdf.PageConfig{Custom: &pdf.CustomPageSize{
			WidthMM:  pdf.Millimeters(338.6666667),
			HeightMM: pdf.Millimeters(190.5),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Runtime.FontChecks) != 6 {
		t.Fatalf("font checks = %#v", report.Runtime.FontChecks)
	}
	if report.Runtime.ValidationIdentity == nil || report.Runtime.ValidationIdentity.FontBundleDigest != descriptor.ValidationRequest.ExpectedFontBundleDigest {
		t.Fatalf("validation identity = %#v want digest %q", report.Runtime.ValidationIdentity, descriptor.ValidationRequest.ExpectedFontBundleDigest)
	}
}

func fakeExecutable(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "browser")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 1\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func installedChromium() string {
	candidates := []string{"/opt/homebrew/bin/chromium"}
	if runtime.GOOS == "darwin" {
		candidates = append(candidates, "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome")
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
			return candidate
		}
	}
	return ""
}

func code(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if index := strings.IndexByte(message, ':'); index >= 0 {
		return message[:index]
	}
	return message
}
