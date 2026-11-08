package margo

import (
	"bytes"
	"context"
	"testing"
)

func explicitChatHTML(t *testing.T, source string) string {
	t.Helper()
	component, err := RenderChatWithOptions(context.Background(), source, ChatOptions{ExplicitBlockquotes: true})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := component.Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}
func TestRenderChatExplicitBlockquotes(t *testing.T) {
	for _, tt := range []struct{ name, source, want string }{
		{"setext-equals", "> quoted\n===", "<blockquote><p>quoted</p></blockquote><p>===</p>"},
		{"setext-dash", "> quoted\n---", "<blockquote><p>quoted</p></blockquote><p>---</p>"},
		{"emoji", "> quoted line\n🙌🙌😀", "<blockquote><p>quoted line</p></blockquote><p>🙌🙌😀</p>"},
		{"depth", ">> nested\n> outer\nout", "<blockquote><blockquote><p>nested</p></blockquote><p>outer</p></blockquote><p>out</p>"},
		{"inline", "> **quoted**\n_unquoted_", "<blockquote><p><strong>quoted</strong></p></blockquote><p><em>unquoted</em></p>"},
		{"explicit", "> first\n> second", "<blockquote><p>first<br>second</p></blockquote>"},
		{"list-quote", "- > quoted\n  own", "<ul><li><blockquote><p>quoted</p></blockquote>own</li></ul>"},
		{"blank", "> quote\n\nplain", "<blockquote><p>quote</p></blockquote><p>plain</p>"},
		{"plain", "first\nsecond", "<p>first<br>second</p>"},
		{"list", "> - one\n> - two\nout", "<blockquote><ul><li>one</li><li>two</li></ul></blockquote><p>out</p>"},
		{"fence", "```\n> literal\nplain\n```", "<pre><code>&gt; literal\nplain\n</code></pre>"},
		{"quoted-fence", "> ```\n> literal\n> ```\nout", "<blockquote><pre><code>literal\n</code></pre></blockquote><p>out</p>"},
		{"indented", "> quote\n    code", "<blockquote><p>quote</p></blockquote><pre><code>code\n</code></pre>"},
		{"crlf", "> café\r\n😀", "<blockquote><p>café</p></blockquote><p>😀</p>"},
		{"escaped", "\\> literal\nplain", "<p>\\&gt; literal<br>plain</p>"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := explicitChatHTML(t, tt.source); got != tt.want {
				t.Fatalf("got %s\nwant%s", got, tt.want)
			}
		})
	}
}
func TestRenderChatDefaultStillAllowsLazyQuotes(t *testing.T) {
	source := "> quoted line\n🙌🙌😀"
	want := "<blockquote><p>quoted line<br>🙌🙌😀</p></blockquote>"
	if got := chatHTML(t, source); got != want {
		t.Fatal(got)
	}
	component, err := RenderChatWithOptions(context.Background(), source, ChatOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := component.Render(context.Background(), &out); err != nil || out.String() != want {
		t.Fatal(out.String(), err)
	}
}

func TestExplicitQuoteOptionKeepsOtherChatSyntax(t *testing.T) {
	for _, source := range []string{
		"plain\ncontinued", "- item\n  continued\n- second", "1. first\n2. second", "    indented\n    code", "```xml\n<a>content</a>\n```", "`a > b` **bold** _italic_ ~~strike~~", "[link](https://example.org) ![image](https://example.org/image.png)", "<script>literal</script>", "---\ntitle: visible\n---\nbody", "|a|b|\n|-|-|\n|c|d|",
	} {
		if got, want := explicitChatHTML(t, source), chatHTML(t, source); got != want {
			t.Errorf("non-quote syntax changed for %q: got%s want%s", source, got, want)
		}
	}
}
