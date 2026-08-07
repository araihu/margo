package margo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/a-h/templ"
)

func mustRenderSource(t *testing.T, source string, options ...RenderOption) *RenderResult {
	t.Helper()
	compiler := New(WithHostPolicy(Policy{RawHTML: RawHTMLDeny, OutputBytes: MaxOutputBytes}))
	document, err := compiler.Compile(context.Background(), Source{Name: "render.md", Content: []byte(source)})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	result, err := compiler.Render(context.Background(), document, options...)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	return result
}

func renderComponent(t *testing.T, component templ.Component) string {
	t.Helper()
	var buffer bytes.Buffer
	if err := component.Render(context.Background(), &buffer); err != nil {
		t.Fatalf("component.Render() error = %v", err)
	}
	return buffer.String()
}

func hashFileForTest(t *testing.T, name string) string {
	t.Helper()
	file, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func TestSemanticRender(t *testing.T) {
	result := mustRenderSource(t, "# Hello\n\nA [safe link](https://example.com).\n\n- one\n- two\n\n> quoted\n")
	markup := renderComponent(t, result.Content())
	for _, want := range []string{
		`<h1 id="hello">Hello</h1>`,
		`<p>A <a href="https://example.com">safe link</a>.</p>`,
		`<ul><li>one</li><li>two</li></ul>`,
		`<blockquote><p>quoted</p></blockquote>`,
	} {
		if !bytes.Contains([]byte(markup), []byte(want)) {
			t.Fatalf("rendered markup missing %q:\n%s", want, markup)
		}
	}
}

func TestSemanticRenderUsesFrontmatterBodyForFenceLanguage(t *testing.T) {
	result := mustRenderSource(t, "---\ntitle: With code\n---\n\n~~~go\nfmt.Println(\"hello\")\n~~~\n")
	markup := renderComponent(t, result.Content())
	if !bytes.Contains([]byte(markup), []byte(`aria-label="Copy go code"`)) {
		t.Fatalf("code fence language was not preserved after frontmatter:\n%s", markup)
	}
}

func TestMermaidSourceStartsExpandedButRemainsDisclosure(t *testing.T) {
	result := mustRenderSource(t, "```mermaid\nflowchart TD\n  A --> B\n```\n")
	markup := renderComponent(t, result.Content())
	want := `<details open class="margo-mermaid__source"><summary>Mermaid source</summary>`
	if !bytes.Contains([]byte(markup), []byte(want)) {
		t.Fatalf("Mermaid source disclosure is not expanded by default:\n%s", markup)
	}
}

func TestSemanticRenderMatchesGolden(t *testing.T) {
	got := renderComponent(t, mustRenderSource(t, "# Hello\n\nA [safe link](https://example.com).\n\n- one\n- two\n\n> quoted\n").Content())
	wantBytes, err := os.ReadFile("testdata/render/semantic.html")
	if err != nil {
		t.Fatal(err)
	}
	if got != string(wantBytes) {
		t.Fatalf("semantic golden mismatch:\nwant %s\n got %s", wantBytes, got)
	}
}

func TestC5ConsumesC0RootModuleTransferReadOnly(t *testing.T) {
	transferBytes, err := os.ReadFile("integration/root-module-transfer.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var transfer struct {
		GoModPostSHA256       string
		GoSumPostSHA256       string
		C5ExpectedGoModSHA256 string
		C5ExpectedGoSumSHA256 string
	}
	if err := json.Unmarshal(transferBytes, &transfer); err != nil {
		t.Fatal(err)
	}
	if got := hashFileForTest(t, "go.mod"); got != transfer.GoModPostSHA256 || got != transfer.C5ExpectedGoModSHA256 {
		t.Fatalf("go.mod transfer hash = %s, want %s", got, transfer.GoModPostSHA256)
	}
	if got := hashFileForTest(t, "go.sum"); got != transfer.GoSumPostSHA256 || got != transfer.C5ExpectedGoSumSHA256 {
		t.Fatalf("go.sum transfer hash = %s, want %s", got, transfer.GoSumPostSHA256)
	}
}
