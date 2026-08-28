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

func TestBuildLocalSitePublishesRelativeIframeHTMLWhenUnsafeHTMLIsEnabled(t *testing.T) {
	root := filepath.Clean("/workspace/docs")
	preview := []byte(`<!doctype html><html><body><h1>Interactive preview</h1></body></html>`)
	request := Request{
		SourceRoot:  root,
		Sources:     []Source{{Path: "index.md", Content: []byte("# Home\n\n<div class=\"preview\"><iframe src=\"preview.html\" title=\"Interactive preview\"></iframe></div>\n")}},
		Compiler:    margo.New(margo.WithUnsafeHTML()),
		Assets:      AssetsLocal,
		AssetReader: mapAssetReader{filepath.Join(root, "preview.html"): preview},
	}
	result, err := Build(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	page := artifactContent(t, result, "index.html")
	if !strings.Contains(page, `<iframe src="preview.html" title="Interactive preview"></iframe>`) {
		t.Fatalf("local iframe was not preserved: %s", page)
	}
	if got := artifactBytes(t, result, "preview.html"); !bytes.Equal(got, preview) {
		t.Fatalf("preview artifact = %q, want %q", got, preview)
	}
}

func TestBuildInlineSiteEmbedsRelativeIframeHTMLWhenUnsafeHTMLIsEnabled(t *testing.T) {
	root := filepath.Clean("/workspace/docs")
	preview := []byte(`<main>Inline preview</main>`)
	result, err := Build(context.Background(), Request{
		SourceRoot: root,
		Sources:    []Source{{Path: "index.md", Content: []byte("# Home\n\n<iframe src=\"preview.html\" title=\"Inline preview\"></iframe>\n")}},
		Compiler:   margo.New(margo.WithUnsafeHTML()), Assets: AssetsInline,
		AssetReader: mapAssetReader{filepath.Join(root, "preview.html"): preview},
	})
	if err != nil {
		t.Fatal(err)
	}
	page := artifactContent(t, result, "index.html")
	if !strings.Contains(page, `srcdoc="&lt;main&gt;Inline preview&lt;/main&gt;"`) || strings.Contains(page, `src="preview.html"`) {
		t.Fatalf("inline iframe was not embedded: %s", page)
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

func TestBuildLocalSitePublishesExistingLinkedAssets(t *testing.T) {
	root := t.TempDir()
	assetPath := filepath.Join(root, "assets", "format-study.pdf")
	assetContent := []byte("format study")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(assetPath, assetContent, 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Build(context.Background(), Request{
		SourceRoot: root,
		Sources: []Source{
			{Path: "nested/index.md", Content: []byte("# Home\n\n[Download](../assets/format-study.pdf?download=1#page=2)\n\n[Guide](guide.md#guide)\n\n[External](https://example.com/file.pdf)\n\n[Missing](../assets/missing.pdf)\n")},
			{Path: "nested/guide.md", Content: []byte("# Guide\n")},
		},
		Compiler: margo.New(), Assets: AssetsLocal,
	})
	if err != nil {
		t.Fatal(err)
	}
	page := artifactContent(t, result, "nested/index.html")
	for _, want := range []string{
		`href="../assets/format-study.pdf?download=1#page=2"`,
		`href="guide.html#guide"`,
		`href="https://example.com/file.pdf"`,
		`href="../assets/missing.pdf"`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("page missing %q:\n%s", want, page)
		}
	}
	if got := artifactBytes(t, result, "assets/format-study.pdf"); !bytes.Equal(got, assetContent) {
		t.Fatalf("linked asset = %q, want %q", got, assetContent)
	}
	for _, artifact := range result.Artifacts {
		if artifact.Path == "assets/missing.pdf" {
			t.Fatalf("missing linked asset was published: %+v", result.Artifacts)
		}
	}
}

func TestBuildLocalSiteRejectsLinkedAssetOutsideSourceRoot(t *testing.T) {
	_, err := Build(context.Background(), Request{
		SourceRoot: t.TempDir(),
		Sources:    []Source{{Path: "index.md", Content: []byte("# Home\n\n[Escape](../secret.pdf)\n")}},
		Compiler:   margo.New(), Assets: AssetsLocal,
	})
	requireSiteCode(t, err, "site.asset_outside_root")
}

func TestBuildLocalSiteDoesNotReplacePageWithSameNamedHTMLAsset(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "guide.html"), []byte("source asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Build(context.Background(), Request{
		SourceRoot: root,
		Sources: []Source{
			{Path: "guide.md", Content: []byte("# Guide\n")},
			{Path: "index.md", Content: []byte("# Home\n\n[Guide](guide.html)\n")},
		},
		Compiler: margo.New(), Assets: AssetsLocal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if page := artifactContent(t, result, "index.html"); !strings.Contains(page, `href="guide.html"`) {
		t.Fatalf("page link was changed unexpectedly: %s", page)
	}
	if page := artifactContent(t, result, "guide.html"); strings.Contains(page, "source asset") {
		t.Fatalf("same-named source asset replaced the generated page: %s", page)
	}
}

func TestBuildLocalSiteRejectsImageAssetCollidingWithPage(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "guide.html"), []byte("\x89PNG\r\n\x1a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Build(context.Background(), Request{
		SourceRoot: root,
		Sources: []Source{
			{Path: "guide.md", Content: []byte("# Guide\n")},
			{Path: "index.md", Content: []byte("# Home\n\n![Guide](guide.html)\n")},
		},
		Compiler: margo.New(), Assets: AssetsLocal,
	})
	requireSiteCode(t, err, "site.artifact_collision")
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
		`class="margo-document__publication-dates" role="group" aria-label="Publication dates"`,
		`data-margo-publication-label="published">Published</span>`,
		`data-margo-publication-label="modified">Updated</span>`,
		`class="margo-document__publication-separator" aria-hidden="true" data-margo-publication-separator="true"> · </span>`,
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

func TestBuildLocalSitePublicationDateLabelsHandleSingleDate(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		label      string
		date       string
		articleTag string
	}{
		{
			name:       "published only",
			source:     "---\ntitle: Published only\nlanguage: en\npublishedAt: \"2026-08-25T12:00:00Z\"\n---\n# Published only\n",
			label:      `data-margo-publication-label="published">Published</span>`,
			date:       `data-margo-publication-date="published">2026-08-25T12:00:00Z</time>`,
			articleTag: `article:published_time" content="2026-08-25T12:00:00Z"`,
		},
		{
			name:       "modified only",
			source:     "---\ntitle: Modified only\nlanguage: en\nmodifiedAt: \"2026-08-26T12:00:00Z\"\n---\n# Modified only\n",
			label:      `data-margo-publication-label="modified">Updated</span>`,
			date:       `data-margo-publication-date="modified">2026-08-26T12:00:00Z</time>`,
			articleTag: `article:modified_time" content="2026-08-26T12:00:00Z"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Build(context.Background(), Request{
				Sources:  []Source{{Path: "post.md", Content: []byte(test.source)}},
				Compiler: margo.New(), Assets: AssetsInline,
			})
			if err != nil {
				t.Fatal(err)
			}
			page := artifactContent(t, result, "post.html")
			for _, required := range []string{test.label, test.date, test.articleTag} {
				if !strings.Contains(page, required) {
					t.Fatalf("single publication date missing %q:\n%s", required, page)
				}
			}
			if strings.Contains(page, `data-margo-publication-separator="true"`) {
				t.Fatalf("single publication date unexpectedly contains a separator:\n%s", page)
			}
			if test.name == "published only" && strings.Contains(page, `data-margo-publication-label="modified"`) {
				t.Fatalf("published-only page unexpectedly contains an Updated label:\n%s", page)
			}
			if test.name == "modified only" && strings.Contains(page, `data-margo-publication-label="published"`) {
				t.Fatalf("modified-only page unexpectedly contains a Published label:\n%s", page)
			}
		})
	}
}

func TestBuildLocalSiteEscapesPublicationMetadata(t *testing.T) {
	result, err := Build(context.Background(), Request{
		Sources:  []Source{{Path: "post.md", Content: []byte("---\ntitle: Escaping\nlanguage: en\nauthors: [\"Ana <Admin>\"]\npublishedAt: \"2026-08-25T12:00:00Z\"\ntags: [\"<script>alert(1)</script>\", \"C++ & Go\"]\n---\n# Escaping\n")}},
		Compiler: margo.New(), Assets: AssetsInline,
	})
	if err != nil {
		t.Fatal(err)
	}
	page := artifactContent(t, result, "post.html")
	for _, escaped := range []string{
		`Ana &lt;Admin&gt;`,
		`&lt;script&gt;alert(1)&lt;/script&gt;`,
		`C++ &amp; Go`,
	} {
		if !strings.Contains(page, escaped) {
			t.Fatalf("publication metadata is not escaped as %q:\n%s", escaped, page)
		}
	}
	if strings.Contains(page, "<script>alert(1)</script>") {
		t.Fatalf("publication metadata emitted executable markup:\n%s", page)
	}
}

func TestBuildLocalSiteLocalizesPublicationDateLabels(t *testing.T) {
	result, err := Build(context.Background(), Request{
		Sources:  []Source{{Path: "post.md", Content: []byte("---\ntitle: Notas de lançamento\nlanguage: pt-BR\npublishedAt: \"2026-08-25T12:00:00Z\"\nmodifiedAt: \"2026-08-26T12:00:00Z\"\n---\n# Notas de lançamento\n")}},
		Compiler: margo.New(), Assets: AssetsInline,
	})
	if err != nil {
		t.Fatal(err)
	}
	page := artifactContent(t, result, "post.html")
	for _, required := range []string{
		`aria-label="Datas de publicação"`,
		`data-margo-publication-label="published">Publicado</span>`,
		`data-margo-publication-label="modified">Atualizado</span>`,
	} {
		if !strings.Contains(page, required) {
			t.Fatalf("Portuguese publication label missing %q:\n%s", required, page)
		}
	}
}

func TestPublicationMetadataStylesWrapAtNarrowWidths(t *testing.T) {
	for name, stylesheet := range map[string]string{
		"directory site":           configuredSiteCSS,
		"typed layout site":        configuredTypedSiteCSS,
		"shared publication rules": publicationMetadataCSS,
	} {
		t.Run(name, func(t *testing.T) {
			for _, required := range []string{
				".margo-document__publication-dates",
				"flex-wrap: wrap;",
				"min-inline-size: 0;",
				"overflow-wrap: anywhere;",
			} {
				if !strings.Contains(stylesheet, required) {
					t.Fatalf("stylesheet missing narrow-date rule %q:\n%s", required, stylesheet)
				}
			}
		})
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

func TestBuildInlineSiteLinkBeforeImagePreservesMediaType(t *testing.T) {
	root := filepath.Clean("/workspace/docs")
	result, err := Build(context.Background(), Request{
		SourceRoot: root,
		Sources:    []Source{{Path: "index.md", Content: []byte("# Home\n\n[Download logo](logo.png)\n\n![Logo](logo.png)\n")}},
		Compiler:   margo.New(),
		Assets:     AssetsInline,
		AssetReader: mapAssetReader{
			filepath.Join(root, "logo.png"): []byte("\x89PNG\r\n\x1a\n"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	page := artifactContent(t, result, "index.html")
	if !strings.Contains(page, `href="logo.png"`) {
		t.Fatalf("linked image asset was not published as a link: %s", page)
	}
	if !strings.Contains(page, `src="data:image/png;base64,`) {
		t.Fatalf("image did not preserve its detected media type after a link: %s", page)
	}
	if strings.Contains(page, `src="data:;base64,`) {
		t.Fatalf("image emitted an empty media type after a link: %s", page)
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
