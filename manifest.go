package margo

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/araihu/margo/internal/canonicaljson"
)

// ManifestEntry binds one output path to its exact artifact bytes.
type ManifestEntry struct {
	Path   string         `json:"path"`
	Digest ArtifactDigest `json:"digest"`
}

// Manifest is a defensive, deterministic collection of output identities.
type Manifest struct {
	Entries []ManifestEntry `json:"entries"`
}

// Clone returns a manifest whose entry storage is independent of the source.
func (m Manifest) Clone() Manifest {
	clone := Manifest{Entries: make([]ManifestEntry, len(m.Entries))}
	copy(clone.Entries, m.Entries)
	return clone
}

// Validate checks path and duplicate invariants before a manifest is emitted.
func (m Manifest) Validate() error {
	seen := make(map[string]struct{}, len(m.Entries))
	for _, entry := range m.Entries {
		if entry.Path == "" {
			return fmt.Errorf("margo manifest: empty path")
		}
		if _, ok := seen[entry.Path]; ok {
			return fmt.Errorf("margo manifest: duplicate path %q", entry.Path)
		}
		seen[entry.Path] = struct{}{}
	}
	return nil
}

// Digest returns the domain-separated SHA-256 of the canonical sorted
// manifest. It panics only when the in-memory value cannot be canonicalized;
// callers that accept external data should call Validate first.
func (m Manifest) Digest() string {
	entries := append([]ManifestEntry(nil), m.Entries...)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	bytes, err := canonicaljson.Marshal(struct {
		Entries []ManifestEntry `json:"entries"`
	}{Entries: entries})
	if err != nil {
		panic(err)
	}
	preimage := append([]byte("margo/manifest/v1\n"), bytes...)
	digest := sha256.Sum256(preimage)
	return hex.EncodeToString(digest[:])
}
