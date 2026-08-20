package site

import (
	"errors"
	"reflect"
	"testing"

	margo "github.com/araihu/margo"
)

func TestLayoutRegistryDefaultsValidate(t *testing.T) {
	for _, kind := range []LayoutKind{LayoutArticle, LayoutLanding, LayoutDocs} {
		t.Run(string(kind), func(t *testing.T) {
			entry, ok := builtinLayoutRegistry().lookup(kind)
			if !ok {
				t.Fatalf("missing kind %q", kind)
			}
			if _, err := entry.validateValues(entry.defaults, layoutValueSiteDefault, "/layout/default"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLayoutRegistryRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		config  LayoutConfig
		code    string
		pointer string
	}{
		{
			name:    "unknown kind",
			config:  LayoutConfig{Kind: LayoutKind("catalog")},
			code:    "site.layout_unknown",
			pointer: "/layout/kind",
		},
		{
			name: "unknown value",
			config: LayoutConfig{
				Kind:    LayoutDocs,
				Default: map[string]any{"sidebaar": true},
			},
			code:    "site.layout_value_unknown",
			pointer: "/layout/default/sidebaar",
		},
		{
			name: "wrong scalar type",
			config: LayoutConfig{
				Kind:    LayoutDocs,
				Default: map[string]any{"sidebar": "yes"},
			},
			code:    "site.layout_value_invalid",
			pointer: "/layout/default/sidebar",
		},
		{
			name: "wrong array element type",
			config: LayoutConfig{
				Kind:    LayoutDocs,
				Default: map[string]any{"families": []any{"default", true}},
			},
			code:    "site.layout_value_invalid",
			pointer: "/layout/default/families/1",
		},
		{
			name: "invalid enum",
			config: LayoutConfig{
				Kind: LayoutLanding,
				Default: map[string]any{
					"content": map[string]any{"layout": "landing"},
				},
			},
			code:    "site.layout_value_invalid",
			pointer: "/layout/default/content/layout",
		},
		{
			name: "site override cannot declare families",
			config: LayoutConfig{
				Kind:   LayoutDocs,
				Values: map[string]any{"families": []any{"module"}},
			},
			code:    "site.layout_value_invalid",
			pointer: "/layout/values/families",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := resolveSiteLayout(test.config, "site.yaml")
			requirePresentationDiagnostic(t, err, test.code, "site.yaml", test.pointer)
		})
	}
}

func TestLayoutRegistryValidationOrderIsStable(t *testing.T) {
	first := map[string]any{"zeta": true, "alpha": true}
	second := map[string]any{"alpha": true, "zeta": true}
	for _, values := range []map[string]any{first, second} {
		_, err := resolveSiteLayout(LayoutConfig{Kind: LayoutArticle, Default: values}, "site.yaml")
		requirePresentationDiagnostic(t, err, "site.layout_value_unknown", "site.yaml", "/layout/default/alpha")
	}
}

func TestLayoutValuesMergeMapsAndReplaceArrays(t *testing.T) {
	base := map[string]any{
		"families": []any{"default", "module"},
		"sidebar":  true,
		"content":  map[string]any{"layout": "article"},
	}
	patch := map[string]any{
		"families": []any{"cli"},
		"sidebar":  false,
	}

	got := mergeLayoutValues(base, patch)

	assertLayoutValues(t, got, map[string]any{
		"families": []any{"cli"},
		"sidebar":  false,
		"content":  map[string]any{"layout": "article"},
	})

	got["families"].([]any)[0] = "changed"
	got["content"].(map[string]any)["layout"] = "changed"
	assertLayoutValues(t, base, map[string]any{
		"families": []any{"default", "module"},
		"sidebar":  true,
		"content":  map[string]any{"layout": "article"},
	})
	assertLayoutValues(t, patch, map[string]any{
		"families": []any{"cli"},
		"sidebar":  false,
	})
}

func TestLayoutValuesIdentityIgnoresMapInsertionOrder(t *testing.T) {
	first := map[string]any{
		"sidebar":  true,
		"content":  map[string]any{"layout": "article"},
		"families": []any{"default", "module"},
	}
	second := map[string]any{
		"families": []any{"default", "module"},
		"content":  map[string]any{"layout": "article"},
		"sidebar":  true,
	}

	firstIdentity, err := layoutValuesIdentity(LayoutDocs, first)
	if err != nil {
		t.Fatal(err)
	}
	secondIdentity, err := layoutValuesIdentity(LayoutDocs, second)
	if err != nil {
		t.Fatal(err)
	}
	if firstIdentity == "" || firstIdentity != secondIdentity {
		t.Fatalf("identities differ: %q != %q", firstIdentity, secondIdentity)
	}

	reorderedArray := mergeLayoutValues(second, map[string]any{"families": []any{"module", "default"}})
	reorderedIdentity, err := layoutValuesIdentity(LayoutDocs, reorderedArray)
	if err != nil {
		t.Fatal(err)
	}
	if reorderedIdentity == firstIdentity {
		t.Fatal("array order did not affect layout identity")
	}
}

func TestLayoutCascadePreservesKindBuckets(t *testing.T) {
	cascade, err := resolveSiteLayout(LayoutConfig{
		Kind: LayoutDocs,
		Default: map[string]any{
			"families": []any{"default", "module"},
		},
		Values: map[string]any{
			"family": "module",
			"toc":    false,
		},
	}, "site.yaml")
	if err != nil {
		t.Fatal(err)
	}

	docsBefore := cascade.resolved()
	assertLayoutValues(t, docsBefore.Values, map[string]any{
		"families": []any{"default", "module"},
		"family":   "module",
		"sidebar":  true,
		"toc":      false,
		"content":  map[string]any{"layout": "article"},
	})

	cascade, err = cascade.apply(LayoutPatch{Kind: LayoutLanding, Source: "index.md", Base: "/layout"})
	if err != nil {
		t.Fatal(err)
	}
	landing := cascade.resolved()
	if landing.Kind != LayoutLanding {
		t.Fatalf("landing kind = %q", landing.Kind)
	}
	assertLayoutValues(t, landing.Values, map[string]any{
		"shell":   false,
		"content": map[string]any{"layout": "article"},
	})

	cascade, err = cascade.apply(LayoutPatch{
		Kind:   LayoutDocs,
		Values: map[string]any{"sidebar": false},
		Source: "module/index.md",
		Base:   "/layout",
	})
	if err != nil {
		t.Fatal(err)
	}
	docsAfter := cascade.resolved()
	assertLayoutValues(t, docsAfter.Values, map[string]any{
		"families": []any{"default", "module"},
		"family":   "module",
		"sidebar":  false,
		"toc":      false,
		"content":  map[string]any{"layout": "article"},
	})
	if docsBefore.Identity == "" || docsBefore.Identity == docsAfter.Identity {
		t.Fatalf("docs identities before=%q after=%q", docsBefore.Identity, docsAfter.Identity)
	}
}

func TestLayoutDocsFamilyDefaultsIncludeDefault(t *testing.T) {
	cascade, err := resolveSiteLayout(LayoutConfig{
		Kind:    LayoutDocs,
		Default: map[string]any{"families": []any{"module", "cli"}},
	}, "site.yaml")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := cascade.resolved().Values["families"], []any{"default", "module", "cli"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("families = %#v, want %#v", got, want)
	}
}

func TestLayoutDocsFamilySelectionRequiresCentralDeclaration(t *testing.T) {
	tests := []struct {
		name    string
		apply   func(layoutCascade) error
		source  string
		pointer string
	}{
		{
			name: "site default",
			apply: func(layoutCascade) error {
				_, err := resolveSiteLayout(LayoutConfig{
					Kind: LayoutDocs,
					Default: map[string]any{
						"families": []any{"module"},
						"family":   "missing",
					},
				}, "site.yaml")
				return err
			},
			source:  "site.yaml",
			pointer: "/layout/default/family",
		},
		{
			name: "site values",
			apply: func(layoutCascade) error {
				_, err := resolveSiteLayout(LayoutConfig{
					Kind:    LayoutDocs,
					Default: map[string]any{"families": []any{"module"}},
					Values:  map[string]any{"family": "missing"},
				}, "site.yaml")
				return err
			},
			source:  "site.yaml",
			pointer: "/layout/values/family",
		},
		{
			name: "directory patch",
			apply: func(cascade layoutCascade) error {
				_, err := cascade.apply(LayoutPatch{
					Values: map[string]any{"family": "missing"},
					Source: "module/_layout.yaml",
					Base:   "/",
				})
				return err
			},
			source:  "module/_layout.yaml",
			pointer: "/values/family",
		},
		{
			name: "Markdown patch",
			apply: func(cascade layoutCascade) error {
				_, err := cascade.apply(LayoutPatch{
					Values: map[string]any{"family": "missing"},
					Source: "guide.md",
					Base:   "/layout",
				})
				return err
			},
			source:  "guide.md",
			pointer: "/layout/values/family",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cascade, err := resolveSiteLayout(LayoutConfig{
				Kind:    LayoutDocs,
				Default: map[string]any{"families": []any{"module"}},
			}, "site.yaml")
			if err != nil {
				t.Fatal(err)
			}
			requirePresentationDiagnostic(t, test.apply(cascade), "site.family_undeclared", test.source, test.pointer)
		})
	}
}

func TestLayoutDocsFamilyDeclarationsAreSiteDefaultOnly(t *testing.T) {
	tests := []struct {
		name    string
		patch   LayoutPatch
		pointer string
	}{
		{
			name: "directory patch",
			patch: LayoutPatch{
				Values: map[string]any{"families": []any{"default", "module"}},
				Source: "module/_layout.yaml",
				Base:   "/",
			},
			pointer: "/values/families",
		},
		{
			name: "Markdown patch",
			patch: LayoutPatch{
				Values: map[string]any{"families": []any{"default", "module"}},
				Source: "guide.md",
				Base:   "/layout",
			},
			pointer: "/layout/values/families",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cascade, err := resolveSiteLayout(LayoutConfig{Kind: LayoutDocs}, "site.yaml")
			if err != nil {
				t.Fatal(err)
			}
			_, err = cascade.apply(test.patch)
			requirePresentationDiagnostic(t, err, "site.layout_value_invalid", test.patch.Source, test.pointer)
		})
	}
}

func requirePresentationDiagnostic(t *testing.T, err error, code, source, pointer string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected diagnostic %q", code)
	}
	var diagnosticError *margo.DiagnosticError
	if !errors.As(err, &diagnosticError) || len(diagnosticError.Diagnostics) != 1 {
		t.Fatalf("error = %v, want one presentation diagnostic", err)
	}
	diagnostic := diagnosticError.Diagnostics[0]
	if diagnostic.Code != code || diagnostic.Source != source || diagnostic.Pointer != pointer {
		t.Fatalf("diagnostic = %+v, want code=%q source=%q pointer=%q", diagnostic, code, source, pointer)
	}
}

func assertLayoutValues(t *testing.T, got, want map[string]any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("layout values = %#v, want %#v", got, want)
	}
}
