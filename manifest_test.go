package margo

import "testing"

func TestManifestIsDefensiveAndCanonical(t *testing.T) {
	manifest := Manifest{Entries: []ManifestEntry{{Path: "index.html", Digest: ArtifactDigestOf([]byte("html"))}}}
	copy := manifest.Clone()
	copy.Entries[0].Path = "mutated.html"
	if manifest.Entries[0].Path != "index.html" {
		t.Fatal("manifest clone aliases entries")
	}
	if len(manifest.Digest()) != 64 {
		t.Fatal("manifest digest is not hexadecimal SHA-256")
	}
}
