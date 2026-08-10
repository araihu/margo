package mermaid_test

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/araihu/margo/assets"
	"github.com/araihu/margo/internal/canonicaljson"
)

type runtimeProfile struct {
	AssetCount     int    `json:"assetCount"`
	AssetSetDigest string `json:"assetSetDigest"`
	MermaidVersion string `json:"mermaidVersion"`
	MermaidDigest  string `json:"mermaidDigest"`
}

type assetIdentity struct {
	Hash string `json:"hash"`
	Name string `json:"name"`
	Path string `json:"path"`
}

func TestMermaidRuntimeIdentityIsPinned(t *testing.T) {
	resource, ok := assets.MuambaResourceByName("mermaid")
	if !ok {
		t.Fatal("Mermaid resource missing from generated Muamba inventory")
	}
	if resource.Version != "11.16.1" {
		t.Fatalf("Mermaid version = %q, want 11.16.1", resource.Version)
	}

	runtimeHash, ok := assets.MuambaHash("mermaid", "runtime")
	if !ok {
		t.Fatal("Mermaid runtime missing from generated Muamba inventory")
	}
	profileBytes, err := os.ReadFile("../../profiles/margo-mermaid-svg-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var profile runtimeProfile
	if err := json.Unmarshal(profileBytes, &profile); err != nil {
		t.Fatal(err)
	}
	if profile.MermaidVersion != resource.Version {
		t.Fatalf("profile Mermaid version = %q, inventory = %q", profile.MermaidVersion, resource.Version)
	}
	if profile.MermaidDigest != runtimeHash {
		t.Fatalf("profile Mermaid digest = %q, inventory = %q", profile.MermaidDigest, runtimeHash)
	}
	identities := make([]assetIdentity, 0, len(resource.Downloads))
	for _, download := range resource.Downloads {
		identities = append(identities, assetIdentity{Hash: download.Hash, Name: download.Name, Path: download.Path})
	}
	sort.Slice(identities, func(i, j int) bool { return identities[i].Name < identities[j].Name })
	canonical, err := canonicaljson.Marshal(identities)
	if err != nil {
		t.Fatal(err)
	}
	assetPreimage := append([]byte("margo/mermaid-asset-set/v1\n"), canonical...)
	assetDigest := sha256.Sum256(assetPreimage)
	gotAssetDigest := "sha256:" + hex.EncodeToString(assetDigest[:])
	if profile.AssetCount != len(identities) {
		t.Fatalf("profile asset count = %d, inventory = %d", profile.AssetCount, len(identities))
	}
	if profile.AssetSetDigest != gotAssetDigest {
		t.Fatalf("profile asset set digest = %q, inventory = %q", profile.AssetSetDigest, gotAssetDigest)
	}

	runtimeFile, err := assets.MuambaOpen("mermaid", "runtime")
	if err != nil {
		t.Fatal(err)
	}
	defer runtimeFile.Close()
	runtimeBytes, err := io.ReadAll(runtimeFile)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha512.Sum384(runtimeBytes)
	got := "sha384:" + hex.EncodeToString(digest[:])
	if got != runtimeHash {
		t.Fatalf("embedded Mermaid digest = %q, inventory = %q", got, runtimeHash)
	}
}

func TestMermaidSupportedFamilyESMClosureIsEmbedded(t *testing.T) {
	resource, ok := assets.MuambaResourceByName("mermaid")
	if !ok {
		t.Fatal("Mermaid resource missing from generated Muamba inventory")
	}
	downloadByPath := map[string]string{}
	for _, download := range resource.Downloads {
		if download.Name == "license" || download.Name == "browser-bundle" {
			continue
		}
		const prefix = "assets/mermaid/11.16.1/"
		if !strings.HasPrefix(download.Path, prefix) {
			t.Fatalf("download %s path = %q", download.Name, download.Path)
		}
		downloadByPath[strings.TrimPrefix(download.Path, prefix)] = download.Name
	}

	staticImport := regexp.MustCompile(`(?:from|import)["'](\./[^"']+)["']`)
	dynamicImport := regexp.MustCompile(`import\(["'](\./[^"']+)["']\)`)
	selectedDynamic := map[string]bool{
		"chunks/mermaid.esm.min/flowDiagram-BWE6NHOH.mjs":     true,
		"chunks/mermaid.esm.min/sequenceDiagram-URATNSBD.mjs": true,
	}
	seen := map[string]bool{}
	pending := []string{"mermaid.esm.min.mjs"}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[current] {
			continue
		}
		downloadName, ok := downloadByPath[current]
		if !ok {
			t.Fatalf("ESM dependency %q is not embedded", current)
		}
		seen[current] = true
		file, err := assets.MuambaOpen("mermaid", downloadName)
		if err != nil {
			t.Fatal(err)
		}
		moduleBytes, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range staticImport.FindAllSubmatch(moduleBytes, -1) {
			pending = append(pending, path.Clean(path.Join(path.Dir(current), string(match[1]))))
		}
		for _, match := range dynamicImport.FindAllSubmatch(moduleBytes, -1) {
			dependency := path.Clean(path.Join(path.Dir(current), string(match[1])))
			if current != "mermaid.esm.min.mjs" || selectedDynamic[dependency] {
				pending = append(pending, dependency)
			}
		}
	}
	if len(seen) != len(downloadByPath) {
		var extras []string
		for embeddedPath := range downloadByPath {
			if !seen[embeddedPath] {
				extras = append(extras, embeddedPath)
			}
		}
		sort.Strings(extras)
		t.Fatalf("unreferenced embedded ESM files: %v", extras)
	}
}
