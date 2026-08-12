package margo

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRepositoryHasRootAndDaggerGoModules(t *testing.T) {
	var modules []string
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "vendor") {
			return filepath.SkipDir
		}
		if entry.Name() == "go.mod" {
			modules = append(modules, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"dagger/go.mod", "go.mod"}
	if !reflect.DeepEqual(modules, want) {
		t.Fatalf("Go modules = %v, want %v", modules, want)
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
