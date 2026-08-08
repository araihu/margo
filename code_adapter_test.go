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
