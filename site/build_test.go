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
	if shared < 2 {
		t.Fatalf("shared assets = %d, artifacts = %+v", shared, result.Artifacts)
	}
	if err := result.Manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(result.Manifest.Entries) != len(result.Artifacts) {
		t.Fatalf("manifest entries = %d artifacts = %d", len(result.Manifest.Entries), len(result.Artifacts))
	}
}

func TestBuildLocalSiteRewritesLinksInsideTableCells(t *testing.T) {
	result, err := Build(context.Background(), Request{
		SourceRoot: "/workspace/docs",
		Sources: []Source{
			{Path: "index.md", Content: []byte("---\ntitle: Table links\nlanguage: en\n---\n\n# Table links\n\n| Resource | Destination |\n| --- | --- |\n| Guide | [Open guide](guide.md) |\n\n[Open guide outside table](guide.md).\n")},
			{Path: "guide.md", Content: []byte("---\ntitle: Guide\nlanguage: en\n---\n\n# Guide\n")},
		},
		Compiler: margo.New(),
		Assets:   AssetsLocal,
	})
	if err != nil {
		t.Fatal(err)
	}
	index := artifactContent(t, result, "index.html")
	for _, want := range []string{
		`<a href="guide.html">Open guide</a>`,
		`<a href="guide.html">Open guide outside table</a>`,
	} {
		if !strings.Contains(index, want) {
			t.Fatalf("index missing %q:\n%s", want, index)
		}
	}
}

func TestBuildLocalSiteProjectsPublicationMetadata(t *testing.T) {
	result, err := Build(context.Background(), Request{
		Sources:  []Source{{Path: "posts/launch.md", Content: []byte("---\ntitle: Launch notes\ndescription: The launch summary.\nlanguage: en\nauthors: [Ana Silva, Rui Costa]\npublishedAt: \"2026-08-25T12:00:00Z\"\nmodifiedAt: \"2026-08-26T12:00:00Z\"\ntags: [operations, release]\n---\n# Launch notes\n\nThe launch summary.\n")}},
		Compiler: margo.New(), Assets: AssetsInline,
	})
	if err != nil {
		t.Fatal(err)
	}
	page := artifactContent(t, result, "posts/launch.html")
	for _, required := range []string{
		`data-margo-publication-metadata="true"`,
		`<address aria-label="Authors"><span rel="author">Ana Silva</span>, <span rel="author">Rui Costa</span></address>`,
		`<time datetime="2026-08-25T12:00:00Z" data-margo-publication-date="published">2026-08-25T12:00:00Z</time>`,
		`<time datetime="2026-08-26T12:00:00Z" data-margo-publication-date="modified">2026-08-26T12:00:00Z</time>`,
		`<li data-margo-publication-tag="operations">operations</li>`,
		`<meta property="article:published_time" content="2026-08-25T12:00:00Z"`,
		`<meta property="article:modified_time" content="2026-08-26T12:00:00Z"`,
		`<meta property="article:author" content="Ana Silva"`,
		`<meta property="article:tag" content="release"`,
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("publication projection missing %q:\n%s", required, page)
		}
	}
	if len(result.Pages) != 1 || len(result.Site.Routes) != 1 {
		t.Fatalf("page projection = pages=%+v routes=%+v", result.Pages, result.Site.Routes)
	}
	for _, projected := range [][]string{result.Pages[0].Authors, result.Site.Routes[0].Authors} {
		if !reflect.DeepEqual(projected, []string{"Ana Silva", "Rui Costa"}) {
			t.Fatalf("authors = %#v", projected)
		}
	}
	if result.Pages[0].PublishedAt != "2026-08-25T12:00:00Z" || !reflect.DeepEqual(result.Pages[0].Tags, []string{"operations", "release"}) {
		t.Fatalf("page publication metadata = %+v", result.Pages[0])
	}
}

func TestBuildLocalSiteAcceptsBlogAuthorDateAliases(t *testing.T) {
	result, err := Build(context.Background(), Request{
		Sources:  []Source{{Path: "post.md", Content: []byte("---\ntitle: A blog post\nlanguage: en\nauthor: Ana Silva\ndate: 2026-08-25\ntags: [operations]\n---\n# A blog post\n\nPost body.\n")}},
		Compiler: margo.New(), Assets: AssetsInline,
	})
	if err != nil {
		t.Fatal(err)
	}
	page := artifactContent(t, result, "post.html")
	for _, required := range []string{`rel="author">Ana Silva</span>`, `datetime="2026-08-25"`, `article:published_time" content="2026-08-25"`, `data-margo-publication-tag="operations"`} {
		if !strings.Contains(page, required) {
			t.Fatalf("blog alias projection missing %q:\n%s", required, page)
		}
	}
	if got := result.Pages[0]; got.PublishedAt != "2026-08-25" || !reflect.DeepEqual(got.Authors, []string{"Ana Silva"}) {
		t.Fatalf("alias page = %+v", got)
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

func TestBuildLocalSiteProjectsPlainHTMLWithoutLayoutChrome(t *testing.T) {
	result, err := Build(context.Background(), Request{
		Sources: []Source{{Path: "guide.md", Content: []byte(`---
title: Guide
margo:
  actions:
    markdown: true
---
# Guide

The source stays available beside the rendered page.
`)}},
		Compiler: margo.New(), Assets: AssetsLocal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(artifactBytes(t, result, "guide.md")); !strings.Contains(got, "The source stays available") {
		t.Fatalf("retained Markdown = %q", got)
	}
	page := artifactContent(t, result, "guide.html")
	for _, forbidden := range []string{
		"margo-page-actions", "margo-breadcrumbs", "margo-pagination",
		`id="left-nav"`, `id="right-nav"`, "data-margo-layout",
		"margo-assets/site.css", "goshtoso",
	} {
		if strings.Contains(page, forbidden) {
			t.Fatalf("plain HTML contains %q: %s", forbidden, page)
		}
	}
	for _, forbiddenArtifact := range []string{
		pageActionsScriptPath, pageActionsStylePath, pageActionsIconSpritePath,
		"margo-assets/site.css", "assets/styles.css",
	} {
		if artifactExists(result, forbiddenArtifact) {
			t.Fatalf("unexpected artifact %q", forbiddenArtifact)
		}
	}
}

func TestBuildInlineSiteProjectsPlainHTMLWithoutGoshtosoCSS(t *testing.T) {
	result, err := Build(context.Background(), Request{
		Sources:  []Source{{Path: "guide.md", Content: []byte("# Guide\n\nPlain projection.\n")}},
		Compiler: margo.New(), Assets: AssetsInline,
	})
	if err != nil {
		t.Fatal(err)
	}
	page := artifactContent(t, result, "guide.html")
	for _, forbidden := range []string{
		"tailwindcss v4.3.3", "--font-sans: -apple-system",
		`data-margo-requirement="goshtoso.styles"`,
	} {
		if strings.Contains(page, forbidden) {
			t.Fatalf("inline HTML contains Goshtoso CSS %q", forbidden)
		}
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
	for _, expected := range []string{`href="../margo-assets/document.css"`, `src="../margo-assets/table-sort.js"`} {
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
