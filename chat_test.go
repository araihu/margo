package margo

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
)

func chatHTML(t *testing.T, source string) string {
	t.Helper()
	component, err := RenderChat(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := component.Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}
func TestRenderChatNestedQuotesAndExistingSyntax(t *testing.T) {
	got := chatHTML(t, ">> kDE: O documentário\n>> **bold** _italic_ ~~strike~~\n> Lembrando...\n> next\n\n*italic*, not a new dialect\n\n- one\n- two\n\n3. third\n\n`code`\n\n```mermaid\ngraph TD; A-->B\n```")
	for _, want := range []string{"<blockquote><blockquote>", "<strong>bold</strong>", "<em>italic</em>", "<del>strike</del>", "Lembrando...<br>next", "<ul><li>one</li>", `<ol start="3">`, "<code>code</code>", "<pre><code>graph TD; A--&gt;B"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	for _, forbidden := range []string{"<script", "data-runtime", "x-data", "id=", "mermaid-runtime"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("chat emitted document runtime: %s", got)
		}
	}
}
func TestRenderChatEscapesHTMLAndDoesNotFetchImages(t *testing.T) {
	got := chatHTML(t, `<script>alert(1)</script>

[unsafe](javascript:alert%281%29) ![photo](https://example.test/image.png) [safe](https://example.test/?x=1&y=2)

<img src="https://example.test/remote" onerror="alert(2)">

`+"```html\n<script>code</script>\n```")
	if strings.Contains(got, "<script>") || strings.Contains(got, "<img") || strings.Contains(got, `href="javascript:`) {
		t.Fatal(got)
	}
	for _, want := range []string{"&lt;script&gt;", `href="https://example.test/image.png"`, `rel="noopener noreferrer"`, "javascript:alert%281%29", "&lt;img"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
func TestRenderChatPreservesFrontmatterWithoutDocumentFeatures(t *testing.T) {
	got := chatHTML(t, "---\ntitle: Visible\n---\n# heading\n\nbody\nline")
	if !strings.Contains(got, "---<br>title: Visible<br>---<br>") || !strings.Contains(got, "# heading") || !strings.Contains(got, "body<br>line") {
		t.Fatal(got)
	}
	if strings.Contains(got, "<h") || strings.Contains(got, "id=") {
		t.Fatal(got)
	}
}
func TestRenderChatBoundsCancellationAndConcurrentReuse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RenderChat(ctx, "text"); err == nil {
		t.Fatal("cancelled render accepted")
	}
	if _, err := RenderChat(context.Background(), strings.Repeat("x", MaxChatBytes+1)); err == nil {
		t.Fatal("unbounded source accepted")
	}
	component, err := RenderChat(context.Background(), "**same**")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			var out bytes.Buffer
			if err := component.Render(context.Background(), &out); err != nil || out.String() != "<p><strong>same</strong></p>" {
				t.Errorf("reuse %s %v", out.String(), err)
			}
		})
	}
	wg.Wait()
	var out bytes.Buffer
	if err := component.Render(ctx, &out); err == nil {
		t.Fatal("cancelled component accepted")
	}
}
