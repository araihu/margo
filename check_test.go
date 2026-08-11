package margo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type checkMapReader map[string][]byte

func (reader checkMapReader) ReadFile(name string) ([]byte, error) {
	data, ok := reader[filepath.Clean(name)]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), data...), nil
}

func TestCheckReportsActionableCompatibilityDiagnostics(t *testing.T) {
	root := filepath.Clean("/workspace/docs")
	source := Source{
		Name:    filepath.Join(root, "guide.md"),
		BaseURL: root,
		Content: []byte("---\nlanguage: en_US\n---\n\n<span>raw</span>\n\n![remote](https://cdn.example.com/image.png)\n![missing](missing.png)\n![unsafe](unsafe.svg)\n![](ok.png)\n[Guide](other.md)\n\n```mermaid\n%%{init: {\"theme\": \"dark\"}}%%\ngraph TD; A-->B\n```\n"),
	}
	reader := checkMapReader{
		filepath.Join(root, "unsafe.svg"): []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		filepath.Join(root, "ok.png"):     []byte("not inspected by preflight"),
	}

	diagnostics, err := Check(context.Background(), source, WithCheckAssetReader(reader))
	if err != nil {
		t.Fatal(err)
	}
	wantCodes := []string{
		"source.metadata_invalid",
		"check.raw_html",
		"check.asset_remote",
		"check.asset_missing",
		"check.svg_incompatible",
		"check.image_alt_empty",
		"check.link_relative",
		"mermaid.configuration_forbidden",
	}
	gotCodes := make([]string, len(diagnostics))
	for index, diagnostic := range diagnostics {
		gotCodes[index] = diagnostic.Code
		if diagnostic.Source != source.Name || diagnostic.Line <= 0 || diagnostic.Column <= 0 {
			t.Errorf("diagnostic %q has incomplete source position: %+v", diagnostic.Code, diagnostic)
		}
		if diagnostic.Pointer == "" || diagnostic.Hint == "" {
			t.Errorf("diagnostic %q is not actionable: %+v", diagnostic.Code, diagnostic)
		}
	}
	if !reflect.DeepEqual(gotCodes, wantCodes) {
		t.Fatalf("codes = %#v, want %#v\ndiagnostics: %+v", gotCodes, wantCodes, diagnostics)
	}
	if diagnostics[0].Line != 2 || diagnostics[0].Pointer != "/language" {
		t.Fatalf("metadata diagnostic = %+v", diagnostics[0])
	}
	if diagnostics[7].Line != 14 || diagnostics[7].Pointer != "/mermaid/configuration" {
		t.Fatalf("Mermaid diagnostic = %+v", diagnostics[7])
	}
	if diagnostics[5].Severity != SeverityWarning || diagnostics[6].Severity != SeverityWarning {
		t.Fatalf("advisory severities = %q, %q", diagnostics[5].Severity, diagnostics[6].Severity)
	}
}

func TestCheckIsDeterministicAndDoesNotMutateSource(t *testing.T) {
	source := Source{Name: "clean.md", BaseURL: "/workspace", Content: []byte("# Clean\n\n[external](https://example.com)\n")}
	original := append([]byte(nil), source.Content...)
	first, err := Check(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Check(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 0 || !reflect.DeepEqual(first, second) || !reflect.DeepEqual(source.Content, original) {
		t.Fatalf("first=%+v second=%+v source=%q", first, second, source.Content)
	}
}

func TestCheckHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Check(ctx, Source{Name: "x.md", Content: []byte("# x")}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}
