package margo

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func TestPagesSitePublishesCompleteSchemaTree(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	want := map[string][]byte{}
	err := filepath.WalkDir("schema/v1", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		writePagesFixture(t, root, path, data)
		want[path] = data
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// New nested schemas must not require another publisher allowlist update.
	future := filepath.Join("schema", "v1", "output", "nested", "future-report.json")
	want[future] = []byte("{\n  \"type\": \"object\"\n}\n")
	writePagesFixture(t, root, future, want[future])
	writePagesFixture(t, root, "README.md", []byte("not a schema"))
	writePagesFixture(t, root, "_site/index.html", []byte("stale site output"))
	writePagesFixture(t, root, "_site/schema/v1/obsolete.json", []byte("stale schema"))

	if output, err := runPagesScript(t, root); err != nil {
		t.Fatalf("stage schema tree: %v\n%s", err, output)
	}
	seen := 0
	err = filepath.WalkDir(filepath.Join(root, "_site"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(filepath.Join(root, "_site"), path)
		if err != nil {
			return err
		}
		expected, ok := want[rel]
		if !ok {
			t.Errorf("unexpected published file %s", rel)
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Equal(data, expected) {
			t.Errorf("published bytes differ for %s", rel)
		}
		seen++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen != len(want) {
		t.Fatalf("published %d schemas, want %d", seen, len(want))
	}
}

func TestPagesSiteRejectsInvalidSchemaTrees(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"empty", "non-json file"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "schema/v1"), 0o755); err != nil {
				t.Fatal(err)
			}
			if name == "non-json file" {
				writePagesFixture(t, root, "schema/v1/document.json", []byte("{}\n"))
				writePagesFixture(t, root, "schema/v1/unrelated.html", []byte("not a schema"))
			}
			if output, err := runPagesScript(t, root); err == nil {
				t.Fatalf("invalid schema tree was accepted:\n%s", output)
			}
		})
	}
}

func writePagesFixture(t *testing.T, root, path string, data []byte) {
	t.Helper()
	target := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func runPagesScript(t *testing.T, root string) ([]byte, error) {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("POSIX shell is unavailable")
	}
	source, err := parser.ParseFile(token.NewFileSet(), "dagger/main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var script string
	ast.Inspect(source, func(node ast.Node) bool {
		value, ok := node.(*ast.ValueSpec)
		if !ok || len(value.Names) != 1 || value.Names[0].Name != "pagesScript" {
			return true
		}
		literal, ok := value.Values[0].(*ast.BasicLit)
		if !ok {
			t.Fatal("pagesScript must be a string literal")
		}
		script, err = strconv.Unquote(literal.Value)
		if err != nil {
			t.Fatal(err)
		}
		return false
	})
	if script == "" {
		t.Fatal("pagesScript was not found")
	}
	cmd := exec.Command(sh, "-euc", script)
	cmd.Dir = root
	return cmd.CombinedOutput()
}
