package site

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

func TestConfiguredNoIndexPreservesPageAndOmitsDiscovery(t *testing.T) {
	root := t.TempDir()
	writeTypedLayoutBuildConfig(t, root, "layout:\n  kind: docs\n  values:\n    sidebar: false\n    pagination: false\n")
	writeConfigFile(t, filepath.Join(root, "docs/index.md"), "---\ntitle: Home\nnoindex: false\n---\n# Home\n")
	writeConfigFile(t, filepath.Join(root, "docs/utility.md"), "---\ntitle: Utility\nnoindex: true\n---\n# Utility\n")
	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml"), Compiler: margo.New()})
	if err != nil {
		t.Fatal(err)
	}
	page := string(configArtifact(t, result, "utility.html"))
	if !strings.Contains(page, `<meta name="robots" content="noindex, follow"`) {
		t.Fatal("utility page does not prevent indexing")
	}
	if strings.Contains(string(configArtifact(t, result, "index.html")), `name="robots"`) {
		t.Fatal("indexable page unexpectedly declares robots metadata")
	}
	for _, name := range []string{SitemapPath, LLMSPath} {
		content := string(configArtifact(t, result, name))
		if strings.Contains(content, "https://margo.example/utility.html") || !strings.Contains(content, "https://margo.example/") {
			t.Fatalf("%s advertises a noindex page or omits the public home", name)
		}
	}
	metadata, err := json.Marshal(result.Site)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Pages) != 2 || len(result.Site.Routes) != 2 || !strings.Contains(string(metadata), `"noindex":true`) {
		t.Fatal("noindex page disappeared from route metadata")
	}
}

func TestDirectoryNoIndexProjectsRobots(t *testing.T) {
	result, err := Build(context.Background(), Request{
		Sources:  []Source{{Path: "utility.md", Content: []byte("---\nnoindex: true\n---\n# Utility\n")}},
		Compiler: margo.New(), Assets: AssetsInline,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(artifactContent(t, result, "utility.html")), `<meta name="robots" content="noindex, follow"`) {
		t.Fatal("directory page does not prevent indexing")
	}
}

func TestNoIndexRejectsNonBooleanMetadata(t *testing.T) {
	for _, value := range []any{"true", 1, nil, map[string]any{}} {
		_, err := publicationMetadataFor("utility.md", margo.Metadata{Additional: map[string]any{"noindex": value}}, margo.HTMLMetadata{})
		requireSiteCode(t, err, "site.indexing_invalid")
	}
}
