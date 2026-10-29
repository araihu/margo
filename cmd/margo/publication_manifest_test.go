package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/site"
)

func TestPublishSiteUsesPublicManifestComposition(t *testing.T) {
	result, err := site.Build(context.Background(), site.Request{Sources: []site.Source{{Path: "index.md", Content: []byte("# Home\n")}}, Compiler: margo.New(), Assets: site.AssetsInline})
	if err != nil {
		t.Fatal(err)
	}
	result.Artifacts = append(result.Artifacts, site.Artifact{Path: "index.xml", Content: []byte("<rss/>")})
	// Leave the original manifest stale to exercise final-byte composition.
	want, err := site.MarshalManifest(result, "policy-identity")
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "publication")
	if err := publishSite(context.Background(), target, result, "policy-identity"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(target, site.ManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("CLI manifest differs from public API")
	}
	var document struct {
		Entries []struct{ Path, Digest string }
	}
	if err := json.Unmarshal(got, &document); err != nil {
		t.Fatal(err)
	}
	for _, entry := range document.Entries {
		data, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(entry.Path)))
		if err != nil {
			t.Fatal(err)
		}
		if margo.ArtifactDigestOf(data).String() != entry.Digest {
			t.Errorf("stale digest for %s", entry.Path)
		}
	}
}
