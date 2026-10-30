package site

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/net/html"
)

func TestStructuredDataValidation(t *testing.T) {
	for name, raw := range map[string]string{
		"null": "null", "scalar": `"schema"`, "array": `[]`,
		"context":       `{"@context":"javascript:alert(1)","@type":"WebPage"}`,
		"missing type":  `{"@context":"https://schema.org"}`,
		"empty type":    `{"@context":"https://schema.org","@type":[""]}`,
		"bad graph":     `{"@context":"https://schema.org","@graph":{}}`,
		"empty graph":   `{"@context":"https://schema.org","@graph":[]}`,
		"untyped graph": `{"@context":"https://schema.org","@graph":[{"name":"Home"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			var value any
			if err := json.Unmarshal([]byte(raw), &value); err != nil {
				t.Fatal(err)
			}
			if _, err := structuredDataFor(Page{Source: "page.md"}, margo.Metadata{Additional: map[string]any{"structured_data": value}}); err == nil {
				t.Fatal("accepted invalid structured data")
			}
		})
	}
	if data, err := structuredDataFor(Page{Source: "page.md"}, margo.Metadata{}); err != nil || data != nil {
		t.Fatalf("absent declaration: %s %v", data, err)
	}
	value := map[string]any{"@context": "https://schema.org", "@type": "WebPage", "name": strings.Repeat("a", 65536)}
	if _, err := structuredDataFor(Page{Source: "page.md"}, margo.Metadata{Additional: map[string]any{"structured_data": value}}); err == nil {
		t.Fatal("accepted oversized declaration")
	}
}

func TestStructuredDataSafeProjection(t *testing.T) {
	value := map[string]any{"@context": "https://schema.org", "@graph": []any{map[string]any{"@type": []any{"WebPage", "AboutPage"}, "name": "</script><script>alert(1)</script>&\u2028"}}}
	data, err := structuredDataFor(Page{Source: "about.md"}, margo.Metadata{Additional: map[string]any{"structured_data": value}})
	if err != nil {
		t.Fatal(err)
	}
	page := Page{Source: "about.md", StructuredData: data}
	input := []byte(`<html><head><title>About</title></head><body><main>About</main></body></html>`)
	projected, err := projectPageMetadata(input, page)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(projected, []byte("</script>")) != 1 || bytes.Contains(projected, []byte("<script>alert")) {
		t.Fatalf("unsafe JSON-LD: %s", projected)
	}
	root, err := html.Parse(bytes.NewReader(projected))
	if err != nil {
		t.Fatal(err)
	}
	script := firstElement(firstElement(root, "head"), "script")
	if script == nil || script.FirstChild == nil || !json.Valid([]byte(script.FirstChild.Data)) {
		t.Fatal("missing valid JSON-LD in head")
	}
	if firstElement(root, "header") != nil {
		t.Fatal("structured data added an article header")
	}
	if _, err := projectPageMetadata(projected, page); err == nil {
		t.Fatal("accepted duplicate JSON-LD")
	}
	bodyScript := []byte(`<html><head></head><body><script type="application/ld+json">{}</script></body></html>`)
	if _, err := projectPageMetadata(bodyScript, page); err == nil {
		t.Fatal("accepted body JSON-LD conflict")
	}
}

func TestStructuredDataDirectoryBuildWithoutUnsafeHTML(t *testing.T) {
	result, err := Build(context.Background(), Request{Sources: []Source{{Path: "about.md", Content: []byte("---\ntitle: About\nstructured_data:\n  '@context': https://schema.org\n  '@type': AboutPage\n  url: https://example.com/about/\n---\n# About\n")}}, Compiler: margo.New(), Assets: AssetsInline})
	if err != nil {
		t.Fatal(err)
	}
	content := []byte(artifactContent(t, result, "about.html"))
	if bytes.Count(content, []byte(`type="application/ld+json"`)) != 1 {
		t.Fatal("missing structured data")
	}
	if len(result.Pages[0].StructuredData) == 0 {
		t.Fatal("route report lost structured data")
	}
}

func TestStructuredDataConfiguredLayouts(t *testing.T) {
	for _, kind := range []string{"docs", "article", "landing"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "---\ntitle: Home\nstructured_data:\n  '@context': https://schema.org\n  '@graph':\n    - '@type': WebSite\n      name: Test site\n      url: https://example.com/\n    - '@type': WebPage\n      url: https://example.com/\n---\n# Home\n\nWelcome.\n")
			writeConfigFile(t, filepath.Join(root, "docs", "post.md"), "---\ntitle: Post\nlanguage: en\nauthors: [Ana]\npublishedAt: \"2026-10-27T10:00:00-03:00\"\nstructured_data:\n  type: BlogPosting\n  author_urls:\n    Ana: https://example.com/about/\n---\n# Post\n\nA real post.\n")
			copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
			copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
			config := `version: 1
source: docs
output: dist
assets: local
offline: true
site:
  name: Test site
  base_url: https://example.com
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Test site
layout:
  kind: ` + kind + `
locales:
  default: en
  supported: [en]
theme:
  builtin: true
  name: modern
`
			writeConfigFile(t, filepath.Join(root, "site.yaml"), config)
			result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml"), Compiler: margo.New()})
			if err != nil {
				t.Fatal(err)
			}
			post := artifactContent(t, result, "post.html")
			if !strings.Contains(post, `"@type":"BlogPosting"`) || !strings.Contains(post, `"url":"https://example.com/post.html"`) || !strings.Contains(post, `"datePublished":"2026-10-27T10:00:00-03:00"`) {
				t.Fatalf("article projection missing from %s layout", kind)
			}
			content := []byte(artifactContent(t, result, "index.html"))
			head, _, ok := bytes.Cut(content, []byte("</head>"))
			if !ok || bytes.Count(head, []byte(`type="application/ld+json"`)) != 1 || bytes.Contains(content, []byte(`data-margo-publication-metadata`)) {
				t.Fatalf("bad metadata projection for %s", kind)
			}
		})
	}
}

func TestStructuredDataArticleProjection(t *testing.T) {
	page := Page{Source: "post.md", Title: "Two authors </script>", Description: "A real summary", Canonical: "https://example.com/docs/pt/post/", Locale: "pt-BR", Authors: []string{"Ana", "Rui"}, PublishedAt: "2026-10-26T23:00:00-03:00", ModifiedAt: "2026-10-27T03:00:00Z"}
	for _, kind := range []string{"Article", "BlogPosting"} {
		options := map[string]any{"type": kind, "image": "https://example.com/cover%20image.jpg", "author_urls": map[string]any{"Ana": "https://example.com/about/"}}
		raw, err := structuredDataFor(page, margo.Metadata{Additional: map[string]any{"structured_data": options}})
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if got["@type"] != kind || got["headline"] != page.Title || got["url"] != page.Canonical || got["mainEntityOfPage"] != page.Canonical || got["inLanguage"] != "pt-BR" || got["datePublished"] != page.PublishedAt || got["dateModified"] != page.ModifiedAt || got["image"] != options["image"] {
			t.Fatalf("projection drift: %s", raw)
		}
		authors := got["author"].([]any)
		if len(authors) != 2 || authors[0].(map[string]any)["url"] != "https://example.com/about/" || authors[1].(map[string]any)["url"] != nil {
			t.Fatalf("author profiles leaked: %s", raw)
		}
	}
	minimal := Page{Source: "guide.md", Title: "Guide", Canonical: "https://example.com/guide/"}
	data, err := articleStructuredData(minimal, map[string]any{"type": "Article"})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"author", "image", "datePublished", "dateModified", "description", "inLanguage"} {
		if _, exists := data[key]; exists {
			t.Fatalf("invented %s", key)
		}
	}
	for name, options := range map[string]map[string]any{
		"type": {"type": "WebPage"}, "typo": {"type": "Article", "images": "x"},
		"image": {"type": "Article", "image": "/cover.png"}, "unsafe URL": {"type": "Article", "image": "javascript:alert(1)"},
		"author":               {"type": "Article", "author_urls": map[string]any{"Guest": "https://example.com/guest/"}},
		"profile":              {"type": "Article", "author_urls": map[string]any{"Ana": "data:text/html,x"}},
		"author type":          {"type": "Article", "author_types": map[string]any{"Ana": "Corporation"}},
		"unknown typed author": {"type": "Article", "author_types": map[string]any{"Guest": "Organization"}},
		"invalid type map":     {"type": "Article", "author_types": "Organization"},
	} {
		if _, err := articleStructuredData(page, options); err == nil {
			t.Fatalf("accepted %s", name)
		}
	}
	for _, date := range []string{"yesterday", "2026-10-26T22:59:59-03:00"} {
		invalid := page
		invalid.ModifiedAt = date
		if _, err := articleStructuredData(invalid, map[string]any{"type": "BlogPosting"}); err == nil {
			t.Fatalf("accepted date %s", date)
		}
	}
	if _, err := articleStructuredData(Page{}, map[string]any{"type": "Article"}); err == nil {
		t.Fatal("invented canonical URL")
	}
}

func TestStructuredDataOutputContracts(t *testing.T) {
	page := Page{Source: "page.md", Output: "page.html", StructuredData: json.RawMessage(`{"@context":"https://schema.org","@type":"WebPage"}`)}
	for _, kind := range []margo.SchemaKind{margo.SchemaSiteReport, margo.SchemaSiteManifest} {
		schemaBytes, err := margo.OutputSchema(kind)
		if err != nil {
			t.Fatal(err)
		}
		document, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
		if err != nil {
			t.Fatal(err)
		}
		compiler := jsonschema.NewCompiler()
		if err := compiler.AddResource("https://example.com/schema.json", document); err != nil {
			t.Fatal(err)
		}
		schema, err := compiler.Compile("https://example.com/schema.json")
		if err != nil {
			t.Fatal(err)
		}
		var value any
		if kind == margo.SchemaSiteReport {
			value = map[string]any{"schemaVersion": "margo-site-report/v1", "artifacts": 1, "manifest": strings.Repeat("a", 64), "pages": []Page{page}}
		} else {
			value = map[string]any{"schemaVersion": "margo-site-manifest/v1", "digest": strings.Repeat("a", 64), "entries": []any{}, "routes": []Page{page}}
		}
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := schema.Validate(instance); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
	}
}
