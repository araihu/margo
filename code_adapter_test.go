package margo

import (
	"bytes"
	"testing"
)

func TestCodeBlockUsesGoshtosoChroma(t *testing.T) {
	markup := renderComponent(t, mustRenderSource(t, "```go\npackage main\n\nfunc main() {}\n```\n").Content())
	if !bytes.Contains([]byte(markup), []byte(`<pre`)) || !bytes.Contains([]byte(markup), []byte(`chroma`)) {
		t.Fatalf("code block is not semantic Chroma output:\n%s", markup)
	}
	if bytes.Contains([]byte(markup), []byte(`<script`)) {
		t.Fatalf("code block unexpectedly emitted script markup:\n%s", markup)
	}
}

func TestCodeBlockCopyCanBeDisabledByFenceInfo(t *testing.T) {
	markup := renderComponent(t, mustRenderSource(t, "```text:copy_disabled\nordinary Markdown\n```\n").Content())

	if !bytes.Contains([]byte(markup), []byte(`>text</span>`)) {
		t.Fatalf("copy-disabled fence leaked its option into the language label:\n%s", markup)
	}
	if bytes.Contains([]byte(markup), []byte(`<button`)) {
		t.Fatalf("copy-disabled fence still rendered a button:\n%s", markup)
	}
}

func TestCodeBlockCopyRemainsEnabledByDefault(t *testing.T) {
	markup := renderComponent(t, mustRenderSource(t, "```text\nordinary Markdown\n```\n").Content())

	for _, want := range []string{
		`aria-label="Copy text code"`,
		`data-code-block-copy`,
		`data-code-block-target=`,
		`data-code-block-copy-status`,
		`aria-live="polite"`,
	} {
		if !bytes.Contains([]byte(markup), []byte(want)) {
			t.Fatalf("regular fence lost copy affordance marker %q:\n%s", want, markup)
		}
	}
}
