package margo

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestMermaidSourceStartsCollapsedButRemainsDisclosure(t *testing.T) {
	result := mustRenderSource(t, "```mermaid\nflowchart TD\n  A --> B\n```\n")
	markup := renderComponent(t, result.Content())
	want := `<details class="margo-mermaid__source"><summary>Mermaid source</summary>`
	if !bytes.Contains([]byte(markup), []byte(want)) {
		t.Fatalf("Mermaid source disclosure is not collapsed by default:\n%s", markup)
	}
	if bytes.Contains([]byte(markup), []byte(`<details open class="margo-mermaid__source">`)) {
		t.Fatalf("Mermaid source disclosure is expanded by default:\n%s", markup)
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

func TestC0RootModuleTransferRemainsHistorical(t *testing.T) {
	const (
		expectedSchemaVersion   = "margo/root-module-transfer/v1"
		expectedModulePath      = "github.com/araihu/margo"
		expectedModuleVersion   = "v0.0.1"
		expectedGoModPostSHA256 = "0eb36e99f0c59989a8c8772899acafa7b30dd205c241801b2d1c52ad775617fe"
		expectedGoSumPostSHA256 = "1c7ae9b89ad246a943998c8e7a4a4f19bd59a53f84409e0d930ebe9b1670ddbb"
	)
	transferBytes, err := os.ReadFile("integration/root-module-transfer.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var transfer struct {
		SchemaVersion         string
		ModulePath            string
		ModuleVersion         string
		GoModPostSHA256       string
		GoSumPostSHA256       string
		C5ExpectedGoModSHA256 string
		C5ExpectedGoSumSHA256 string
	}
	if err := json.Unmarshal(transferBytes, &transfer); err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name string
		got  string
		want string
	}{
		{name: "schemaVersion", got: transfer.SchemaVersion, want: expectedSchemaVersion},
		{name: "modulePath", got: transfer.ModulePath, want: expectedModulePath},
		{name: "moduleVersion", got: transfer.ModuleVersion, want: expectedModuleVersion},
		{name: "goModPostSHA256", got: transfer.GoModPostSHA256, want: expectedGoModPostSHA256},
		{name: "c5ExpectedGoModSHA256", got: transfer.C5ExpectedGoModSHA256, want: expectedGoModPostSHA256},
		{name: "goSumPostSHA256", got: transfer.GoSumPostSHA256, want: expectedGoSumPostSHA256},
		{name: "c5ExpectedGoSumSHA256", got: transfer.C5ExpectedGoSumSHA256, want: expectedGoSumPostSHA256},
	}
	for _, check := range checks {
		if check.got == "" || check.got != check.want {
			t.Fatalf("historical transfer %s = %q, want %q", check.name, check.got, check.want)
		}
	}
}
