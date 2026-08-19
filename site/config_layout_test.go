package site

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadConfigAcceptsTypedLayoutSelection(t *testing.T) {
	root := t.TempDir()
	filename := filepath.Join(root, "site.yaml")
	writeConfigFile(t, filename, `version: 1
source: docs
output: dist
site:
  name: Example
  base_url: https://margo.example
  logo: assets/logo.svg
  icon: assets/icon.svg
  social_image:
    path: assets/social.png
    alt: Example documentation preview
  home: index.md
layout:
  kind: docs
  default:
    families: [module, cli]
    sidebar: true
    toc: true
    content:
      layout: article
  values:
    family: default
`)

	config, err := LoadConfig(filename)
	if err != nil {
		t.Fatal(err)
	}
	if config.Layout == nil {
		t.Fatal("layout = nil")
	}
	if config.Layout.Kind != LayoutDocs {
		t.Fatalf("kind = %q", config.Layout.Kind)
	}
	if got, want := config.Layout.Default["families"], []any{"default", "module", "cli"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("families = %#v, want %#v", got, want)
	}

	cascade, err := resolveSiteLayout(*config.Layout, filename)
	if err != nil {
		t.Fatal(err)
	}
	assertLayoutValues(t, cascade.resolved().Values, map[string]any{
		"families": []any{"default", "module", "cli"},
		"family":   "default",
		"sidebar":  true,
		"toc":      true,
		"content":  map[string]any{"layout": "article"},
	})
}

func TestLoadConfigNormalizesDeclaredLayoutFamilies(t *testing.T) {
	filename := writeLayoutConfig(t, `layout:
  kind: docs
  default:
    families: [" module ", " default ", " cli "]
`)
	config, err := LoadConfig(filename)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := config.Layout.Default["families"], []any{"default", "module", "cli"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("families = %#v, want %#v", got, want)
	}
}

func TestLoadConfigRejectsInvalidTypedLayoutSelection(t *testing.T) {
	tests := []struct {
		name    string
		layout  string
		code    string
		pointer string
	}{
		{
			name: "unknown kind",
			layout: `layout:
  kind: catalog
`,
			code:    "site.layout_unknown",
			pointer: "/layout/kind",
		},
		{
			name: "unknown default key",
			layout: `layout:
  kind: docs
  default:
    sidebaar: true
`,
			code:    "site.layout_value_unknown",
			pointer: "/layout/default/sidebaar",
		},
		{
			name: "unknown values key",
			layout: `layout:
  kind: docs
  values:
    sidebaar: true
`,
			code:    "site.layout_value_unknown",
			pointer: "/layout/values/sidebaar",
		},
		{
			name: "invalid value type",
			layout: `layout:
  kind: docs
  values:
    sidebar: enabled
`,
			code:    "site.layout_value_invalid",
			pointer: "/layout/values/sidebar",
		},
		{
			name: "empty family",
			layout: `layout:
  kind: docs
  default:
    families: [module, "  "]
`,
			code:    "site.family_invalid",
			pointer: "/layout/default/families/1",
		},
		{
			name: "duplicate family after trimming",
			layout: `layout:
  kind: docs
  default:
    families: [module, " module "]
`,
			code:    "site.family_duplicate",
			pointer: "/layout/default/families/1",
		},
		{
			name: "duplicate default family",
			layout: `layout:
  kind: docs
  default:
    families: [default, module, " default "]
`,
			code:    "site.family_duplicate",
			pointer: "/layout/default/families/2",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filename := writeLayoutConfig(t, test.layout)
			_, err := LoadConfig(filename)
			requirePresentationDiagnostic(t, err, test.code, filename, test.pointer)
		})
	}
}

func TestLoadConfigRejectsTypedLayoutMixedWithLegacySelection(t *testing.T) {
	tests := []struct {
		name   string
		legacy string
	}{
		{name: "frame", legacy: "frame:\n  builtin: top-main-footer\n"},
		{name: "shell", legacy: "shell:\n  builtin: componentdocshell\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filename := writeLayoutConfig(t, `layout:
  kind: docs
`+test.legacy)
			_, err := LoadConfig(filename)
			requirePresentationDiagnostic(t, err, "site.layout_conflict", filename, "/layout")
		})
	}
}

func TestLoadConfigRejectsMissingBaseURL(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "site.yaml")
	writeConfigFile(t, filename, `version: 1
source: docs
site:
  name: Example
  logo: assets/logo.svg
  icon: assets/icon.svg
  social_image:
    path: assets/social.png
    alt: Example documentation preview
layout:
  kind: docs
`)

	_, err := LoadConfig(filename)
	requirePresentationDiagnostic(t, err, "site.base_url_invalid", filename, "")
}

func writeLayoutConfig(t *testing.T, extra string) string {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "site.yaml")
	writeConfigFile(t, filename, `version: 1
source: docs
site:
  name: Example
  base_url: https://margo.example
  logo: assets/logo.svg
  icon: assets/icon.svg
  social_image:
    path: assets/social.png
    alt: Example documentation preview
`+extra)
	return filename
}
