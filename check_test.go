package margo

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type checkMapReader map[string][]byte

func (reader checkMapReader) ReadAsset(ctx context.Context, root, name string, limit int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, ok := reader[filepath.Clean(filepath.Join(root, filepath.FromSlash(name)))]
	if !ok {
		return nil, os.ErrNotExist
	}
	if int64(len(data)) > limit {
		return nil, ErrCheckAssetTooLarge
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
		filepath.Join(root, "ok.png"):     []byte("\x89PNG\r\n\x1a\n"),
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

func TestCheckDoesNotDropSameLineFindings(t *testing.T) {
	source := Source{Name: "/workspace/guide.md", BaseURL: "/workspace", Content: []byte("![one](one.png) ![two](two.png) [x](x.md) [y](y.md)\n")}
	diagnostics, err := Check(context.Background(), source, WithCheckAssetReader(checkMapReader{}))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"check.asset_missing", "check.asset_missing", "check.link_relative", "check.link_relative"}
	got := make([]string, len(diagnostics))
	columns := make(map[int]struct{})
	for index, diagnostic := range diagnostics {
		got[index] = diagnostic.Code
		columns[diagnostic.Column] = struct{}{}
	}
	if !reflect.DeepEqual(got, want) || len(columns) != 4 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestCheckLocatesLinkSyntaxInsteadOfEarlierDestinationText(t *testing.T) {
	source := Source{Name: "guide.md", Content: []byte("target appears here\n\n[link](target)\n")}
	diagnostics, err := Check(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "check.link_relative" || diagnostics[0].Line != 3 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestCheckEnforcesMetadataListLimitsAndSequencePositions(t *testing.T) {
	var many strings.Builder
	many.WriteString("---\nauthors:\n")
	for index := 0; index < 65; index++ {
		many.WriteString("  - author\n")
	}
	many.WriteString("---\n# x\n")
	diagnostics, err := Check(context.Background(), Source{Name: "many.md", Content: []byte(many.String())})
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "source.metadata_invalid" || diagnostics[0].Pointer != "/authors" || diagnostics[0].Line != 2 {
		t.Fatalf("list limit diagnostics = %+v", diagnostics)
	}

	invalid := Source{Name: "invalid.md", Content: []byte("---\nauthors:\n  - valid\n  - 42\n---\n")}
	diagnostics, err = Check(context.Background(), invalid)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Pointer != "/authors/1" || diagnostics[0].Line != 4 {
		t.Fatalf("sequence diagnostics = %+v", diagnostics)
	}
}

func TestFilesystemCheckAssetReaderRejectsEscapeAndOversize(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "docs")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.png")
	if err := os.WriteFile(outside, []byte("\x89PNG\r\n\x1a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape.png")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	huge := filepath.Join(root, "huge.png")
	file, err := os.Create(huge)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(MaxDocumentBytes + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	source := Source{Name: filepath.Join(root, "guide.md"), BaseURL: root, Content: []byte("![escape](escape.png)\n\n![huge](huge.png)\n")}
	diagnostics, err := Check(context.Background(), source, WithCheckAssetReader(FilesystemCheckAssetReader{}))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"check.asset_outside_root", "check.asset_too_large"}
	got := make([]string, len(diagnostics))
	for index := range diagnostics {
		got[index] = diagnostics[index].Code
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestCheckValidatesAssetContentAndDataSVG(t *testing.T) {
	root := filepath.Clean("/workspace")
	unsafeSVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	dataSVG := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(unsafeSVG)
	source := Source{Name: filepath.Join(root, "guide.md"), BaseURL: root, Content: []byte("![wrong name](unsafe.png)\n\n![bad bytes](broken.png)\n\n![data](" + dataSVG + ")\n")}
	reader := checkMapReader{
		filepath.Join(root, "unsafe.png"): unsafeSVG,
		filepath.Join(root, "broken.png"): []byte("not an image"),
	}
	diagnostics, err := Check(context.Background(), source, WithCheckAssetReader(reader))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"check.svg_incompatible", "check.asset_incompatible", "check.svg_incompatible"}
	got := make([]string, len(diagnostics))
	for index := range diagnostics {
		got[index] = diagnostics[index].Code
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

func TestCheckRejectsWindowsAbsoluteLink(t *testing.T) {
	diagnostics, err := Check(context.Background(), Source{Name: "guide.md", Content: []byte("[local](C:/docs/x.md)\n")})
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != "check.link_unsupported" || diagnostics[0].Severity != SeverityError {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
}

type blockingCheckReader struct{ started chan struct{} }

func (reader blockingCheckReader) ReadAsset(ctx context.Context, _, _ string, _ int64) ([]byte, error) {
	close(reader.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestCheckCancellationInterruptsAssetRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader := blockingCheckReader{started: make(chan struct{})}
	result := make(chan error, 1)
	go func() {
		_, err := Check(ctx, Source{Name: "/workspace/guide.md", BaseURL: "/workspace", Content: []byte("![x](x.png)\n")}, WithCheckAssetReader(reader))
		result <- err
	}()
	<-reader.started
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Check did not interrupt the asset read")
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
