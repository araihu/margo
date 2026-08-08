package margo

import (
	"path/filepath"
	"testing"
)

func TestOutputMappers(t *testing.T) {
	adjacent, err := (AdjacentMapper{}).Map("docs/guide.md")
	if err != nil || adjacent != filepath.Join("docs", "guide.html") {
		t.Fatalf("adjacent mapping = %q, err = %v", adjacent, err)
	}
	preserve, err := (PreserveMapper{SourceRoot: "/repo/docs", OutputDir: "/repo/dist"}).Map("/repo/docs/guide.md")
	if err != nil || preserve != filepath.Join("/repo/dist", "guide.html") {
		t.Fatalf("preserve mapping = %q, err = %v", preserve, err)
	}
	flat, err := (FlatMapper{OutputDir: "/repo/dist"}).Map("/repo/docs/guide.md")
	if err != nil || flat != filepath.Join("/repo/dist", "guide.html") {
		t.Fatalf("flat mapping = %q, err = %v", flat, err)
	}
	if _, err := (PreserveMapper{SourceRoot: "/repo/docs", OutputDir: "/repo/dist"}).Map("/other/guide.md"); err == nil {
		t.Fatal("source outside root unexpectedly mapped")
	}
}
