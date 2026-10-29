package site

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

const publicationLinksSource = `---
authors: [Ana, Rui]
tags: [go]
publishedAt: "2026-10-28T23:30:00-03:00"
publication_links:
  authors:
    Ana: https://example.com/about/?a=1&b=2
  tags:
    go: https://example.com/tags/go/
---
# Post
`

func TestPublicationLinksInNativeHeader(t *testing.T) {
	for _, configured := range []bool{false, true} {
		t.Run(map[bool]string{true: "configured", false: "directory"}[configured], func(t *testing.T) {
			var result Result
			var err error
			if configured {
				root := t.TempDir()
				writeTypedLayoutBuildConfig(t, root, "layout:\n  kind: docs\npublication:\n  date_format: '02/01/2006'\n")
				writeConfigFile(t, filepath.Join(root, "docs/index.md"), publicationLinksSource)
				result, err = BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
			} else {
				result, err = Build(context.Background(), Request{Sources: []Source{{Path: "index.md", Content: []byte(publicationLinksSource)}}, Assets: AssetsInline})
			}
			if err != nil {
				t.Fatalf("%#v", err)
			}
			page := artifactContent(t, result, "index.html")
			for _, want := range []string{`<a href="https://example.com/about/?a=1&amp;b=2" rel="author">Ana</a>`, `<span rel="author">Rui</span>`, `<a href="https://example.com/tags/go/">go</a>`} {
				if !strings.Contains(page, want) {
					t.Errorf("missing %s", want)
				}
			}
			if configured && !strings.Contains(page, `>28/10/2026</time>`) {
				t.Error("linked header lost native dates")
			}
		})
	}
}

func TestPublicationLinksRejectUnsafeOrInventedIdentities(t *testing.T) {
	for _, links := range []any{
		"invalid", map[string]any{"unknown": map[string]any{}},
		map[string]any{"authors": map[string]any{"Unknown": "https://example.com/"}},
		map[string]any{"authors": map[string]any{"Ana": "javascript:alert(1)"}},
		map[string]any{"tags": map[string]any{"go": "//example.com/"}},
		map[string]any{"tags": map[string]any{"go": "https://example.com/\" onmouseover=\"x"}},
	} {
		page := Page{Authors: []string{"Ana"}, Tags: []string{"go"}}
		if err := applyPublicationLinks(&page, margo.Metadata{Additional: map[string]any{"publication_links": links}}); err == nil {
			t.Fatalf("accepted invalid links: %#v", links)
		}
	}
}
