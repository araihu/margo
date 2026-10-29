package site

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
)

// Finalize validates and copies the final application-owned artifacts, sorts
// them, and recomputes their manifest. It preserves page and site metadata;
// callers must update that metadata if they change page semantics. It neither
// reparses HTML nor writes files. The manifest file itself is excluded.
func Finalize(result Result) (Result, error) {
	b := &builder{artifacts: map[string][]byte{}, artifactKeys: map[string]string{}}
	seen := map[string]bool{}
	for _, artifact := range result.Artifacts {
		if !fs.ValidPath(artifact.Path) || artifact.Path == "." || strings.ContainsAny(artifact.Path, "\\:\x00\r\n") {
			return Result{}, diagnostic("site.artifact_invalid", fmt.Sprintf("invalid artifact path %q", artifact.Path), "Use normalized relative paths with forward slashes.", artifact.Path)
		}
		if seen[artifact.Path] {
			return Result{}, diagnostic("site.artifact_collision", "duplicate artifact path", "Keep one artifact for each output path.", artifact.Path)
		}
		seen[artifact.Path] = true
		if err := b.addArtifact(artifact.Path, artifact.Content); err != nil {
			return Result{}, err
		}
	}
	finalized := b.result()
	result.Artifacts = finalized.Artifacts
	result.Manifest = finalized.Manifest
	return result, nil
}

type manifestDocument struct {
	SchemaVersion       string          `json:"schemaVersion"`
	Digest              string          `json:"digest"`
	Policy              string          `json:"policy,omitempty"`
	ConfigVersion       int             `json:"configVersion,omitempty"`
	Layout              string          `json:"layout,omitempty"`
	LayoutSchemaHash    string          `json:"layoutSchemaHash,omitempty"`
	BaseURL             string          `json:"baseURL,omitempty"`
	BasePath            string          `json:"basePath,omitempty"`
	DocumentStyleDigest string          `json:"documentStyleDigest,omitempty"`
	Routes              []Page          `json:"routes,omitempty"`
	Entries             []manifestEntry `json:"entries"`
}

type manifestEntry struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

// MarshalManifest serializes the versioned site manifest for the result's
// current artifact bytes. A stale result.Manifest is recomputed, not trusted.
// policy is the optional trusted host policy identity. The returned JSON ends
// with a newline and does not include the manifest file's own digest.
func MarshalManifest(result Result, policy string) ([]byte, error) {
	finalized, err := Finalize(result)
	if err != nil {
		return nil, err
	}
	document := manifestDocument{
		SchemaVersion: "margo-site-manifest/v1", Digest: finalized.Manifest.Digest(), Policy: policy,
		ConfigVersion: result.Site.ConfigVersion, Layout: result.Site.Layout, LayoutSchemaHash: result.Site.LayoutSchemaHash,
		BaseURL: result.Site.BaseURL, BasePath: result.Site.BasePath, DocumentStyleDigest: result.Site.DocumentStyleDigest,
		Routes: result.Site.Routes, Entries: make([]manifestEntry, len(finalized.Manifest.Entries)),
	}
	for i, entry := range finalized.Manifest.Entries {
		document.Entries[i] = manifestEntry{Path: entry.Path, Digest: entry.Digest.String()}
	}
	data, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("site.manifest_invalid: %w", err)
	}
	return append(data, '\n'), nil
}
