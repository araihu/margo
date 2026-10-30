package site

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSiteAssetBudgetAcrossPages(t *testing.T) {
	root := t.TempDir()
	asset := bytes.Repeat([]byte("x"), 9<<20)
	for _, name := range []string{"a.bin", "b.bin"} {
		if err := os.WriteFile(filepath.Join(root, name), asset, 0600); err != nil {
			t.Fatal(err)
		}
	}
	request := Request{SourceRoot: root, Sources: []Source{
		{Path: "index.md", Content: []byte("# Home\n\n[One](a.bin) [Again](a.bin) [Next](next.md)\n")},
		{Path: "next.md", Content: []byte("# Next\n\n[Two](b.bin)\n")},
	}}
	if _, err := Build(context.Background(), request); err == nil {
		t.Fatal("default aggregate accepted 18 MiB")
	}
	request.AssetBudgetBytes = 18 << 20
	result, err := Build(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(artifactBytes(t, result, "b.bin"), asset) {
		t.Fatal("asset bytes changed")
	}
	request.AssetBudgetBytes--
	if _, err := Build(context.Background(), request); err == nil {
		t.Fatal("accepted aggregate over exact budget")
	}
	request.AssetBudgetBytes = 32 << 20
	request.Sources = []Source{{Path: "index.md", Content: []byte("# Home\n\n[Large](large.bin)\n")}}
	if err := os.WriteFile(filepath.Join(root, "large.bin"), make([]byte, 17<<20), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(context.Background(), request); err == nil {
		t.Fatal("aggregate override raised per-file limit")
	}
}

func TestConfiguredSiteAssetBudget(t *testing.T) {
	root := t.TempDir()
	writeTypedLayoutBuildConfig(t, root, "asset_budget_bytes: 32\nlayout:\n  kind: article\n")
	filename := filepath.Join(root, "site.yaml")
	// The fixture declares source: docs.
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "index.md"), []byte("# Home\n\n[Data](a.bin)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "a.bin"), make([]byte, 33), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filename}); err == nil || !strings.Contains(err.Error(), "site.asset_too_large") {
		t.Fatalf("configured aggregate error = %v", err)
	}
	config, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	config = bytes.Replace(config, []byte("asset_budget_bytes: 32"), []byte("asset_budget_bytes: 33"), 1)
	if err := os.WriteFile(filename, config, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filename}); err != nil {
		t.Fatal(err)
	}
	for _, budget := range []int64{-1, maxSiteAssetBudget + 1} {
		if _, err := Build(context.Background(), Request{AssetBudgetBytes: budget}); err == nil {
			t.Fatalf("accepted budget %d", budget)
		}
	}
}
