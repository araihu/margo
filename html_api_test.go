package margo

import (
	"strings"
	"testing"
)

func TestRenderHTMLProducesHostOwnedFragment(t *testing.T) {
	rendered := mustRenderSource(t, "---\ntitle: Host title\n---\n\nBody with **meaning**.\n")
	result, err := RenderHTML(rendered)
	if err != nil {
		t.Fatal(err)
	}

	var _ *HTMLResult = result
	var _ HTMLMetadata = result.Metadata()
	var _ HTMLFingerprint = result.Fingerprint()

	markup := renderComponent(t, result.Fragment())
	for _, forbidden := range []string{"<!doctype", "<html", "<head", "<body", "<script", "data-theme="} {
		if strings.Contains(strings.ToLower(markup), forbidden) {
			t.Fatalf("fragment owns shell %q: %s", forbidden, markup)
		}
	}
	if strings.Count(markup, `<article class="margo-document">`) != 1 {
		t.Fatalf("article contract: %s", markup)
	}
	if result.PlainText() != "Body with meaning." {
		t.Fatalf("plain text = %q", result.PlainText())
	}
	if result.Metadata().Title != "Host title" {
		t.Fatalf("title = %q", result.Metadata().Title)
	}
}

func TestRenderHTMLHeaderIsExplicit(t *testing.T) {
	rendered := mustRenderSource(t, "---\ntitle: Metadata title\n---\n\nBody\n")
	result, err := RenderHTML(rendered, WithHTMLHeader())
	if err != nil {
		t.Fatal(err)
	}
	if markup := renderComponent(t, result.Fragment()); strings.Count(markup, "<h1") != 1 || !strings.Contains(markup, ">Metadata title</h1>") {
		t.Fatalf("explicit header = %s", markup)
	}
}
