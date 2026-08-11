package margo

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestPreV1HostPolicyAloneAuthorizesSanitizedRawHTML(t *testing.T) {
	compiler := New(WithHostPolicy(Policy{RawHTML: RawHTMLSanitized, OutputBytes: MaxOutputBytes}))
	if _, err := compiler.Compile(context.Background(), Source{Name: "raw.md", Content: []byte("<span>allowed</span>\n")}); err != nil {
		t.Fatalf("host-authorized raw HTML required document ceremony: %v", err)
	}
}

func TestPreV1CommentsAreHarmlessAndDiscarded(t *testing.T) {
	compiler := New()
	document, err := compiler.Compile(context.Background(), Source{
		Name:    "comments.md",
		Content: []byte("<!-- markdownlint-disable MD013 -->\n\n# Hello\n\nText <!-- author note --> remains.\n"),
	})
	if err != nil {
		t.Fatalf("Compile() rejected authoring comments: %v", err)
	}
	rendered, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, rendered.Content())
	if strings.Contains(markup, "<!--") || strings.Contains(markup, "author note") || strings.Contains(markup, "markdownlint") {
		t.Fatalf("authoring comment leaked into output: %s", markup)
	}
}

func TestPreV1MalformedCommentIsPositioned(t *testing.T) {
	_, err := New().Compile(context.Background(), Source{Name: "broken.md", Content: []byte("# Heading\n\n<!-- never closed\n")})
	diagnostic := unwrapDiagnostic(err)
	if diagnostic == nil || len(diagnostic.Diagnostics) != 1 {
		t.Fatalf("malformed comment was not diagnosed: %v", err)
	}
	got := diagnostic.Diagnostics[0]
	if got.Code != "source.html_comment_malformed" || got.Source != "broken.md" || got.Line != 3 {
		t.Fatalf("diagnostic = %+v", got)
	}
}

func TestPreV1MultilineCommentedScriptIsInertButAdjacentHTMLIsNot(t *testing.T) {
	comment := Source{Name: "comment.md", Content: []byte("before\n<!--\n<script>alert(1)</script>\n-->\nafter\n")}
	compiler := New()
	document, err := compiler.Compile(context.Background(), comment)
	if err != nil {
		t.Fatalf("commented script became active: %v", err)
	}
	rendered, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, rendered.Content())
	if strings.Contains(markup, "script") || strings.Contains(markup, "alert(1)") {
		t.Fatalf("comment contents leaked: %s", markup)
	}

	_, err = compiler.Compile(context.Background(), Source{Name: "mixed.md", Content: []byte("<!-- note --><span>real</span>\n")})
	if got := diagnosticCode(err); got != "policy.raw_html.denied" {
		t.Fatalf("adjacent raw HTML diagnostic = %q, error = %v", got, err)
	}
}

func TestPreV1TrustedEmbedFenceHasMigrationDiagnostic(t *testing.T) {
	_, err := New().Compile(context.Background(), Source{Name: "embed.md", Content: []byte("```trusted-embed\nkind: iframe\nurl: https://video.example.com/watch/123\ntitle: Architecture overview\n```\n")})
	if got := diagnosticCode(err); got != "source.trusted_embed_removed" {
		t.Fatalf("diagnostic = %q, error = %v", got, err)
	}
}

func TestPreV1LegacyGoshtosoNamespaceHasMigrationDiagnostic(t *testing.T) {
	_, err := New().Compile(context.Background(), Source{Name: "legacy.md", Content: []byte("---\ngoshtoso:\n  page:\n    size: A4\n---\n# Legacy\n")})
	if got := diagnosticCode(err); got != "frontmatter.goshtoso_removed" {
		t.Fatalf("diagnostic = %q, error = %v", got, err)
	}
}

func TestPreV1ProductionHasNoPredictivePaginator(t *testing.T) {
	source, err := os.ReadFile("standalone.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"window.innerHeight", "data-margo-print-break-before", "markCrossPageBlocks", "prepareNestedHeadingGroups"} {
		if strings.Contains(string(source), forbidden) {
			t.Errorf("production paginator still contains %q", forbidden)
		}
	}
}

func TestPreV1DefaultRejectsRemoteMarkdownImagesButKeepsLinks(t *testing.T) {
	if _, err := New().Compile(context.Background(), Source{Name: "link.md", Content: []byte("[ordinary link](https://example.com)\n")}); err != nil {
		t.Fatalf("ordinary link failed: %v", err)
	}
	_, err := New().Compile(context.Background(), Source{Name: "image.md", Content: []byte("![remote](https://cdn.example.com/image.png)\n")})
	if got := diagnosticCode(err); got != "policy.remote_image.denied" {
		t.Fatalf("remote image diagnostic = %q, error = %v", got, err)
	}
}
