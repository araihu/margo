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
		{name: "unknown policy", policy: pdf.RelativeLinkPolicy("surprise"), code: "pdf.relative_link_policy_invalid"},
		{name: "active link scheme", document: `<a href="javascript:alert(1)">Run</a>`, code: "pdf.link_scheme_forbidden"},
		{name: "network-path link", document: `<a href="//example.com/docs">Docs</a>`, code: "pdf.relative_link_invalid"},
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
