package site

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	margo "github.com/araihu/margo"
)

func TestFrontmatterLayoutPatchUsesMarkdownSourcePointers(t *testing.T) {
	metadata := margo.Metadata{Additional: map[string]any{
		"layout": map[string]any{"values": map[string]any{"family": "missing"}},
	}}
	patch, err := layoutPatchFromMetadata(metadata, "guide.md")
	if err != nil {
		t.Fatal(err)
	}
	if patch.Source != "guide.md" || patch.Base != "/layout" {
		t.Fatalf("patch = %+v", patch)
	}
}

func TestFrontmatterLayoutPatchAcceptsKindOrValues(t *testing.T) {
	tests := []struct {
		name     string
		layout   map[string]any
		wantKind LayoutKind
		want     map[string]any
	}{
		{name: "kind only", layout: map[string]any{"kind": "landing"}, wantKind: LayoutLanding},
		{name: "values only", layout: map[string]any{"values": map[string]any{"toc": false}}, want: map[string]any{"toc": false}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			patch, err := layoutPatchFromMetadata(margo.Metadata{Additional: map[string]any{"layout": test.layout}}, "guide.md")
			if err != nil {
				t.Fatal(err)
			}
			if patch.Kind != test.wantKind || !reflect.DeepEqual(patch.Values, test.want) || patch.Source != "guide.md" || patch.Base != "/layout" {
				t.Fatalf("patch = %#v, want kind=%q values=%#v", patch, test.wantKind, test.want)
			}
		})
	}
}

func TestFrontmatterLayoutPatchRejectsExplicitEmptyKind(t *testing.T) {
	_, err := layoutPatchFromMetadata(margo.Metadata{Additional: map[string]any{
		"layout": map[string]any{"kind": ""},
	}}, "guide.md")
	requirePresentationDiagnostic(t, err, "site.layout_unknown", "guide.md", "/layout/kind")
}

func TestFrontmatterLayoutPatchRejectsInvalidShape(t *testing.T) {
	tests := []struct {
		name    string
		layout  any
		pointer string
	}{
		{name: "non-mapping layout", layout: "landing", pointer: "/layout"},
		{name: "unknown root property", layout: map[string]any{"profile": "docs"}, pointer: "/layout/profile"},
		{name: "non-string kind", layout: map[string]any{"kind": true}, pointer: "/layout/kind"},
		{name: "non-mapping values", layout: map[string]any{"values": false}, pointer: "/layout/values"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := layoutPatchFromMetadata(margo.Metadata{Additional: map[string]any{"layout": test.layout}}, "guide.md")
			requirePresentationDiagnostic(t, err, "site.layout_patch_invalid", "guide.md", test.pointer)
		})
	}
}

func TestFrontmatterLayoutPatchDefersActiveKindValidation(t *testing.T) {
	cascade, err := resolveSiteLayout(LayoutConfig{Kind: LayoutDocs}, "site.yaml")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		values  map[string]any
		code    string
		pointer string
	}{
		{name: "unknown value", values: map[string]any{"sidebaar": true}, code: "site.layout_value_unknown", pointer: "/layout/values/sidebaar"},
		{name: "Markdown family declaration", values: map[string]any{"families": []any{"default", "module"}}, code: "site.layout_value_invalid", pointer: "/layout/values/families"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			patch, patchErr := layoutPatchFromMetadata(margo.Metadata{Additional: map[string]any{
				"layout": map[string]any{"values": test.values},
			}}, "guide.md")
			if patchErr != nil {
				t.Fatal(patchErr)
			}
			_, applyErr := cascade.apply(patch)
			requirePresentationDiagnostic(t, applyErr, test.code, "guide.md", test.pointer)
		})
	}
}

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
