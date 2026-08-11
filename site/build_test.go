package site

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

type mapAssetReader map[string][]byte

func (reader mapAssetReader) ReadAsset(ctx context.Context, root, name string, limit int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, ok := reader[filepath.Clean(filepath.Join(root, filepath.FromSlash(name)))]
	if !ok {
		return nil, os.ErrNotExist
	}
	if int64(len(data)) > limit {
		return nil, margo.ErrCheckAssetTooLarge
	}
	return append([]byte(nil), data...), nil
}

func TestBuildLocalSiteRewritesLinksAndSharesAssets(t *testing.T) {
	root := filepath.Clean("/workspace/docs")
	request := Request{
		SourceRoot: root,
		Sources: []Source{
			{Path: "index.md", Content: []byte("# Home\n\n[Guide](guide.md)\n\n![Logo](assets/logo.png)\n")},
			{Path: "guide.md", Content: []byte("# Guide\n\n[Home](index.md#home)\n")},
		},
		Compiler: margo.New(), Assets: AssetsLocal,
		AssetReader: mapAssetReader{filepath.Join(root, "assets/logo.png"): []byte("\x89PNG\r\n\x1a\n")},
	}
	result, err := Build(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	index := artifactContent(t, result, "index.html")
	guide := artifactContent(t, result, "guide.html")
	for _, assertion := range []struct{ document, fragment string }{
		{index, `href="guide.html"`},
		{index, `src="assets/logo.png"`},
		{guide, `href="index.html#home"`},
	} {
		if !strings.Contains(assertion.document, assertion.fragment) {
			t.Fatalf("document missing %q: %s", assertion.fragment, assertion.document)
		}
	}
	if got := artifactBytes(t, result, "assets/logo.png"); !bytes.Equal(got, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("logo = %x", got)
	}
	shared := 0
	for _, artifact := range result.Artifacts {
		if strings.HasPrefix(artifact.Path, "assets/") || strings.HasPrefix(artifact.Path, "margo-assets/") {
			shared++
		}
	}
	if shared < 3 {
		t.Fatalf("shared assets = %d, artifacts = %+v", shared, result.Artifacts)
	}
	if err := result.Manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(result.Manifest.Entries) != len(result.Artifacts) {
		t.Fatalf("manifest entries = %d artifacts = %d", len(result.Manifest.Entries), len(result.Artifacts))
	}
}

func TestBuildInlineSiteEmbedsAssetsAndIsDeterministic(t *testing.T) {
	root := filepath.Clean("/workspace/docs")
	sources := []Source{
		{Path: "guide.md", Content: []byte("# Guide\n\n[Home](index.md)\n")},
		{Path: "index.md", Content: []byte("# Home\n\n![Logo](logo.png)\n")},
	}
	request := Request{SourceRoot: root, Sources: sources, Compiler: margo.New(), Assets: AssetsInline, AssetReader: mapAssetReader{filepath.Join(root, "logo.png"): []byte("\x89PNG\r\n\x1a\n")}}
	first, err := Build(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	request.Sources = []Source{sources[1], sources[0]}
	second, err := Build(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("site output depends on source order")
	}
	if len(first.Artifacts) != 2 {
		t.Fatalf("inline artifacts = %+v", first.Artifacts)
	}
	index := artifactContent(t, first, "index.html")
	if !strings.Contains(index, "data:image/png;base64,") || !strings.Contains(index, "<style") {
		t.Fatalf("inline page is not self-contained: %s", index)
	}
}

func TestBuildRejectsOutputCollisionAndBrokenMarkdownLink(t *testing.T) {
	_, err := Build(context.Background(), Request{Sources: []Source{{Path: "README.md", Content: []byte("# one")}, {Path: "README.MD", Content: []byte("# two")}}})
	requireSiteCode(t, err, "site.output_collision")

	_, err = Build(context.Background(), Request{Sources: []Source{{Path: "index.md", Content: []byte("# Home\n\n[Missing](missing.md)\n")}}})
	requireSiteCode(t, err, "site.link_missing")

	_, err = Build(context.Background(), Request{Sources: []Source{
		{Path: "index.md", Content: []byte("# Home\n\n[Missing section](guide.md#absent)\n")},
		{Path: "guide.md", Content: []byte("# Guide\n")},
	}})
	requireSiteCode(t, err, "site.anchor_missing")
}

func TestBuildRejectsReservedAndPathPrefixArtifacts(t *testing.T) {
	root := filepath.Clean("/workspace/docs")
	_, err := Build(context.Background(), Request{
		SourceRoot: root,
		Sources:    []Source{{Path: "index.md", Content: []byte("# Home\n\n![bad](margo-manifest.json)\n")}},
		AssetReader: mapAssetReader{
			filepath.Join(root, "margo-manifest.json"): []byte("\x89PNG\r\n\x1a\n"),
		},
	})
	requireSiteCode(t, err, "site.artifact_reserved")

	_, err = Build(context.Background(), Request{Sources: []Source{
		{Path: "margo-manifest.json/page.md", Content: []byte("# Reserved prefix\n")},
	}})
	requireSiteCode(t, err, "site.artifact_reserved")

	_, err = Build(context.Background(), Request{Sources: []Source{
		{Path: "page.md", Content: []byte("# Page\n")},
		{Path: "page.html/nested.md", Content: []byte("# Nested\n")},
	}})
	requireSiteCode(t, err, "site.artifact_collision")
}

func TestBuildLocalSiteUsesPageRelativeDependencyURLs(t *testing.T) {
	result, err := Build(context.Background(), Request{
		Sources: []Source{{Path: "nested/index.md", Content: []byte("# Nested\n\n| A |\n| - |\n| 1 |\n")}},
		Assets:  AssetsLocal,
	})
	if err != nil {
		t.Fatal(err)
	}
	page := artifactContent(t, result, "nested/index.html")
	for _, expected := range []string{`href="../assets/styles.css"`, `href="../margo-assets/document.css"`, `src="../margo-assets/table-sort.js"`} {
		if !strings.Contains(page, expected) {
			t.Fatalf("nested page missing relative dependency %q: %s", expected, page)
		}
	}
	if strings.Contains(page, `href="/`) || strings.Contains(page, `src="/margo-assets/`) {
		t.Fatalf("nested page retains root-absolute dependency: %s", page)
	}
}

func TestBuildAttachesSourceToCompilerDiagnostics(t *testing.T) {
	_, err := Build(context.Background(), Request{Sources: []Source{{Path: "index.md", Content: []byte("# Home\n\n<div>raw</div>\n")}}})
	var diagnostic *margo.DiagnosticError
	if !errors.As(err, &diagnostic) || len(diagnostic.Diagnostics) == 0 {
		t.Fatalf("error = %v", err)
	}
	got := diagnostic.Diagnostics[0]
	if got.Code != "policy.raw_html.denied" || got.Source != "index.md" || got.Hint == "" {
		t.Fatalf("diagnostic = %+v", got)
	}
}

func artifactContent(t *testing.T, result Result, name string) string {
	t.Helper()
	return string(artifactBytes(t, result, name))
}

func artifactBytes(t *testing.T, result Result, name string) []byte {
	t.Helper()
	index := sort.Search(len(result.Artifacts), func(index int) bool { return result.Artifacts[index].Path >= name })
	if index >= len(result.Artifacts) || result.Artifacts[index].Path != name {
		t.Fatalf("artifact %q missing: %+v", name, result.Artifacts)
	}
	return result.Artifacts[index].Content
}

func requireSiteCode(t *testing.T, err error, code string) {
	t.Helper()
	var diagnostic *margo.DiagnosticError
	if !errors.As(err, &diagnostic) || len(diagnostic.Diagnostics) == 0 || diagnostic.Diagnostics[0].Code != code {
		t.Fatalf("error = %v, want %s", err, code)
	}
}
