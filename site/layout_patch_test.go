package site

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscoverConfiguredInputsSeparatesLayoutPatches(t *testing.T) {
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "index.md"), "# Home\n")
	writeConfigFile(t, filepath.Join(root, "_layout.yaml.md"), "# Public Markdown\n")
	writeConfigFile(t, filepath.Join(root, "_layout.yaml"), "values:\n  toc: false\n")
	writeConfigFile(t, filepath.Join(root, "module", "_layout.yaml"), "values:\n  family: module\n")
	writeConfigFile(t, filepath.Join(root, "module", "advanced", "_layout.yaml"), "kind: article\n")

	inputs, err := discoverConfiguredInputs(context.Background(), root, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertSourcePaths(t, inputs.Sources, []string{"_layout.yaml.md", "index.md"})
	assertPatchPaths(t, inputs.Patches, []string{
		"_layout.yaml",
		"module/_layout.yaml",
		"module/advanced/_layout.yaml",
	})
}

func TestDirectoryLayoutPatchRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "patch-target.yaml")
	writeConfigFile(t, target, "kind: article\n")
	if err := os.Symlink(target, filepath.Join(root, "_layout.yaml")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	_, err := discoverConfiguredInputs(context.Background(), root, nil)
	requirePresentationDiagnostic(t, err, "site.layout_patch_invalid", "_layout.yaml", "")
}

func TestDirectoryLayoutPatchRejectsInvalidDocuments(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		code    string
		pointer string
	}{
		{
			name:    "duplicate YAML key",
			yaml:    "kind: docs\nvalues:\n  sidebar: true\n  sidebar: false\n",
			code:    "site.layout_patch_invalid",
			pointer: "/values/sidebar",
		},
		{
			name:    "multiple documents",
			yaml:    "kind: article\n---\nkind: landing\n",
			code:    "site.layout_patch_invalid",
			pointer: "",
		},
		{
			name:    "non-mapping root",
			yaml:    "- kind: article\n",
			code:    "site.layout_patch_invalid",
			pointer: "",
		},
		{
			name:    "directory default",
			yaml:    "default:\n  sidebar: true\n",
			code:    "site.layout_patch_invalid",
			pointer: "/default",
		},
		{
			name:    "unknown root property",
			yaml:    "profile: docs\n",
			code:    "site.layout_patch_invalid",
			pointer: "/profile",
		},
		{
			name:    "invalid kind",
			yaml:    "kind: magazine\n",
			code:    "site.layout_unknown",
			pointer: "/kind",
		},
		{
			name:    "empty kind",
			yaml:    "kind: \"\"\n",
			code:    "site.layout_unknown",
			pointer: "/kind",
		},
		{
			name:    "unknown value for explicit kind",
			yaml:    "kind: docs\nvalues:\n  sidebar_position: left\n",
			code:    "site.layout_value_unknown",
			pointer: "/values/sidebar_position",
		},
		{
			name:    "directory-owned families",
			yaml:    "kind: docs\nvalues:\n  families: [default, module]\n",
			code:    "site.layout_value_invalid",
			pointer: "/values/families",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			const source = "module/_layout.yaml"
			_, err := decodeDirectoryLayoutPatch(source, []byte(test.yaml))
			requirePresentationDiagnostic(t, err, test.code, source, test.pointer)
		})
	}
}

func TestDirectoryLayoutPatchDefersValueOnlySchemaValidation(t *testing.T) {
	const source = "module/_layout.yaml"
	patch, err := decodeDirectoryLayoutPatch(source, []byte("values:\n  family: module\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := LayoutPatch{
		Values: map[string]any{"family": "module"},
		Source: source,
		Base:   "/",
	}
	if !reflect.DeepEqual(patch, want) {
		t.Fatalf("patch = %#v, want %#v", patch, want)
	}
}

func TestLayoutPatchChainUsesExactAncestors(t *testing.T) {
	patches := []LayoutPatch{
		{Source: "_layout.yaml"},
		{Source: "module/_layout.yaml"},
		{Source: "module/advanced/_layout.yaml"},
		{Source: "module/advanced/deeper/_layout.yaml"},
		{Source: "modulex/_layout.yaml"},
	}

	chain := layoutPatchChain("module/advanced/page.md", patches)
	assertPatchPaths(t, chain, []string{
		"_layout.yaml",
		"module/_layout.yaml",
		"module/advanced/_layout.yaml",
	})
	assertPatchPaths(t, layoutPatchChain("index.md", patches), []string{"_layout.yaml"})
}

func assertSourcePaths(t *testing.T, sources []Source, want []string) {
	t.Helper()
	got := make([]string, len(sources))
	for index := range sources {
		got[index] = sources[index].Path
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("source paths = %q, want %q", got, want)
	}
}

func assertPatchPaths(t *testing.T, patches []LayoutPatch, want []string) {
	t.Helper()
	got := make([]string, len(patches))
	for index := range patches {
		got[index] = patches[index].Source
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("patch paths = %q, want %q", got, want)
	}
}
