package margo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryHasOneGoModule(t *testing.T) {
	var nested []string
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "vendor") {
			return filepath.SkipDir
		}
		if path != "go.mod" && entry.Name() == "go.mod" {
			nested = append(nested, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(nested) != 0 {
		t.Fatalf("nested modules: %v", nested)
	}
}

func TestRootPackageDoesNotImportOptionalPackages(t *testing.T) {
	entries, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range entries {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{
			"github.com/araihu/margo/charts",
			"github.com/araihu/margo/deck",
			"github.com/araihu/margo/pdf",
			"github.com/spf13/cobra",
		} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("%s imports %s", path, forbidden)
			}
		}
	}
}
