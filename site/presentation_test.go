package site

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
)

func TestResolveFamilyUsesMostSpecificSegmentPrefix(t *testing.T) {
	families := []FamilyConfig{
		{ID: "tour", Source: "."},
		{ID: "cli", Source: "cli"},
	}
	got, err := resolveFamily("cli/index.md", LocaleConfig{Default: "en", Supported: []string{"en"}}, families)
	if err != nil || got.ID != "cli" {
		t.Fatalf("family = %+v, err = %v", got, err)
	}
	if got, err := resolveFamily("client.md", LocaleConfig{Default: "en", Supported: []string{"en"}}, families); err != nil || got.ID != "tour" {
		t.Fatalf("client fallback = %+v, err = %v", got, err)
	}
}

func TestResolveFamilyStripsLocaleBeforeMatching(t *testing.T) {
	families := []FamilyConfig{{ID: "tour", Source: "."}, {ID: "cli", Source: "cli"}}
	got, err := resolveFamily("pt-BR/cli/index.md", LocaleConfig{Default: "en", Supported: []string{"en", "pt-BR"}}, families)
	if err != nil || got.ID != "cli" {
		t.Fatalf("family = %+v, err = %v", got, err)
	}
}

func TestResolveFamilyReportsUnmatchedSource(t *testing.T) {
	_, err := resolveFamily("guide.md", LocaleConfig{Default: "en", Supported: []string{"en"}}, []FamilyConfig{{ID: "cli", Source: "cli"}})
	diagnostic := presentationDiagnostic(t, err)
	if diagnostic.Code != "site.family_unresolved" || diagnostic.Pointer != "/navigation/families" {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
	if !strings.Contains(diagnostic.Hint, "guide.md") {
		t.Fatalf("diagnostic hint = %q", diagnostic.Hint)
	}
}

func TestResolveLayoutUsesPageFamilyThenSiteDefault(t *testing.T) {
	layouts := LayoutProfiles{
		Default: "docs",
		Profiles: map[string]LayoutProfile{
			"landing": {Frame: LayoutSelection{Builtin: "top-main-footer"}},
			"docs":    {Frame: LayoutSelection{Builtin: "top-left-main-right-footer"}},
		},
	}
	family := FamilyConfig{ID: "module", Layout: "docs"}
	if got, err := resolveLayout(margo.Metadata{}, family, layouts); err != nil || got != "docs" {
		t.Fatalf("family layout = %q, err = %v", got, err)
	}
	if got, err := resolveLayout(margo.Metadata{Margo: margo.DocumentPreferences{Site: &margo.SitePreference{Layout: "landing"}}}, family, layouts); err != nil || got != "landing" {
		t.Fatalf("page layout = %q, err = %v", got, err)
	}
	if got, err := resolveLayout(margo.Metadata{}, FamilyConfig{}, layouts); err != nil || got != "docs" {
		t.Fatalf("default layout = %q, err = %v", got, err)
	}
}

func TestResolveLayoutRejectsUnknownProfile(t *testing.T) {
	_, err := resolveLayout(margo.Metadata{Name: "guide.md", Margo: margo.DocumentPreferences{Site: &margo.SitePreference{Layout: "missing"}}}, FamilyConfig{ID: "docs"}, LayoutProfiles{Default: "docs", Profiles: map[string]LayoutProfile{"docs": {}}})
	diagnostic := presentationDiagnostic(t, err)
	if diagnostic.Code != "site.layout_unknown" || diagnostic.Pointer != "/margo/site/layout" {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
	if diagnostic.Hint == "" || !containsAll(diagnostic.Hint, "guide.md", "missing") {
		t.Fatalf("diagnostic hint = %q", diagnostic.Hint)
	}
}

func TestResolveLayoutRejectsMissingDefault(t *testing.T) {
	_, err := resolveLayout(margo.Metadata{}, FamilyConfig{}, LayoutProfiles{})
	diagnostic := presentationDiagnostic(t, err)
	if diagnostic.Code != "site.layout_required" || diagnostic.Pointer != "/layouts/default" {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
}

func TestValidateAcceptsLayoutProfilesAndFamilies(t *testing.T) {
	config := validPresentationConfig()
	config.Layouts = LayoutProfiles{
		Default: "docs",
		Profiles: map[string]LayoutProfile{
			"landing": {Frame: LayoutSelection{Builtin: "top-main-footer"}},
			"docs":    {Frame: LayoutSelection{Builtin: "top-left-main-right-footer"}},
		},
	}
	config.Navigation.Families = []FamilyConfig{
		{ID: "tour", Label: "Tour", Source: ".", Overview: "index.md", Layout: "landing"},
		{ID: "module", Label: "Module", Source: "module", Overview: "module/index.md", Layout: "docs"},
	}
	if err := config.validate(); err != nil {
		t.Fatalf("validate() error = %v", err)
	}
	if config.Layouts.Default != "docs" || config.Navigation.Families[1].Source != "module" {
		t.Fatalf("config normalization changed valid values: %+v", config)
	}
}

func TestValidatePreservesLegacyDefaultsWithoutPresentationFields(t *testing.T) {
	config := validPresentationConfig()
	if err := config.validate(); err != nil {
		t.Fatalf("validate() error = %v", err)
	}
	if config.Navigation.Mode != "file-tree" || config.Layouts.Default != "" {
		t.Fatalf("legacy defaults = navigation=%q layouts=%+v", config.Navigation.Mode, config.Layouts)
	}
}

func TestValidateRejectsLayoutsWithTopLevelFrame(t *testing.T) {
	config := validPresentationConfig()
	config.Frame = &LayoutSelection{Builtin: "top-main-footer"}
	config.Layouts = LayoutProfiles{Default: "docs", Profiles: map[string]LayoutProfile{"docs": {Frame: LayoutSelection{Builtin: "top-main-footer"}}}}
	diagnostic := presentationDiagnostic(t, config.validate())
	if diagnostic.Code != "site.layout_conflict" || diagnostic.Pointer != "/layouts" {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
}

func TestLoadConfigTreatsExplicitEmptyLayoutsAsProfileMode(t *testing.T) {
	root := t.TempDir()
	filename := filepath.Join(root, "site.yaml")
	writeConfigFile(t, filename, `version: 1
source: docs
site:
  name: Margo
  base_url: https://margo.example
  logo: assets/logo.svg
  icon: assets/icon.svg
  social_image:
    path: assets/social.jpg
    alt: Margo preview
layouts: {}
`)
	_, err := LoadConfig(filename)
	diagnostic := presentationDiagnostic(t, err)
	if diagnostic.Code != "site.layout_default_required" || diagnostic.Pointer != "/layouts/default" {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
}

func TestLoadConfigRejectsExplicitEmptyLayoutsWithLegacyFrame(t *testing.T) {
	root := t.TempDir()
	filename := filepath.Join(root, "site.yaml")
	writeConfigFile(t, filename, `version: 1
source: docs
site:
  name: Margo
  base_url: https://margo.example
  logo: assets/logo.svg
  icon: assets/icon.svg
  social_image:
    path: assets/social.jpg
    alt: Margo preview
frame:
  builtin: top-main-footer
layouts: {}
`)
	_, err := LoadConfig(filename)
	diagnostic := presentationDiagnostic(t, err)
	if diagnostic.Code != "site.layout_conflict" || diagnostic.Pointer != "/layouts" {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
}

func TestValidateRejectsLayoutProfileWithoutBuiltinFrame(t *testing.T) {
	config := validPresentationConfig()
	config.Layouts = LayoutProfiles{Default: "docs", Profiles: map[string]LayoutProfile{"docs": {}}}
	diagnostic := presentationDiagnostic(t, config.validate())
	if diagnostic.Code != "site.layout_invalid" || diagnostic.Pointer != "/layouts/profiles/docs/frame" {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
}

func TestValidateRejectsExternalLayoutProfileFrame(t *testing.T) {
	config := validPresentationConfig()
	config.Layouts = LayoutProfiles{Default: "docs", Profiles: map[string]LayoutProfile{
		"docs": {Frame: LayoutSelection{Command: "external-frame", Protocol: "margo.ssg.frame/v1"}},
	}}
	diagnostic := presentationDiagnostic(t, config.validate())
	if diagnostic.Code != "site.layout_unavailable" || diagnostic.Pointer != "/layouts/profiles/docs/frame" {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
}

func TestValidateRejectsUnknownDefaultAndFamilyLayout(t *testing.T) {
	config := validPresentationConfig()
	config.Layouts = LayoutProfiles{Default: "missing", Profiles: map[string]LayoutProfile{"docs": {Frame: LayoutSelection{Builtin: "top-main-footer"}}}}
	diagnostic := presentationDiagnostic(t, config.validate())
	if diagnostic.Code != "site.layout_unknown" || diagnostic.Pointer != "/layouts/default" {
		t.Fatalf("default diagnostic = %+v", diagnostic)
	}

	config.Layouts.Default = "docs"
	config.Navigation.Families = []FamilyConfig{{ID: "tour", Label: "Tour", Source: ".", Overview: "index.md", Layout: "missing"}}
	diagnostic = presentationDiagnostic(t, config.validate())
	if diagnostic.Code != "site.layout_unknown" || diagnostic.Pointer != "/navigation/families/0/layout" {
		t.Fatalf("family diagnostic = %+v", diagnostic)
	}
}

func TestValidateRejectsDuplicateFamilyIdentityAndRoots(t *testing.T) {
	config := validPresentationConfig()
	config.Layouts = LayoutProfiles{Default: "docs", Profiles: map[string]LayoutProfile{"docs": {Frame: LayoutSelection{Builtin: "top-main-footer"}}}}
	config.Navigation.Families = []FamilyConfig{
		{ID: "tour", Label: "Tour", Source: ".", Overview: "index.md"},
		{ID: "tour", Label: "Other", Source: "docs", Overview: "docs/index.md"},
	}
	diagnostic := presentationDiagnostic(t, config.validate())
	if diagnostic.Code != "site.family_duplicate" || diagnostic.Pointer != "/navigation/families/1/id" {
		t.Fatalf("duplicate ID diagnostic = %+v", diagnostic)
	}

	config.Navigation.Families[1].ID = "docs"
	config.Navigation.Families[1].Source = "./"
	diagnostic = presentationDiagnostic(t, config.validate())
	if diagnostic.Code != "site.family_source_duplicate" || diagnostic.Pointer != "/navigation/families/1/source" {
		t.Fatalf("duplicate root diagnostic = %+v", diagnostic)
	}
}

func TestValidateRejectsInvalidFamilySourceAndOverview(t *testing.T) {
	config := validPresentationConfig()
	config.Layouts = LayoutProfiles{Default: "docs", Profiles: map[string]LayoutProfile{"docs": {Frame: LayoutSelection{Builtin: "top-main-footer"}}}}
	config.Navigation.Families = []FamilyConfig{{ID: "docs", Label: "Docs", Source: "../docs", Overview: "../docs/index.md"}}
	diagnostic := presentationDiagnostic(t, config.validate())
	if diagnostic.Code != "site.family_source_invalid" || diagnostic.Pointer != "/navigation/families/0/source" {
		t.Fatalf("source diagnostic = %+v", diagnostic)
	}

	config.Navigation.Families[0].Source = "docs/../manual"
	diagnostic = presentationDiagnostic(t, config.validate())
	if diagnostic.Code != "site.family_source_invalid" || diagnostic.Pointer != "/navigation/families/0/source" {
		t.Fatalf("parent source diagnostic = %+v", diagnostic)
	}

	config.Navigation.Families[0].Source = "docs/.."
	diagnostic = presentationDiagnostic(t, config.validate())
	if diagnostic.Code != "site.family_source_invalid" || diagnostic.Pointer != "/navigation/families/0/source" {
		t.Fatalf("terminal parent source diagnostic = %+v", diagnostic)
	}

	config.Navigation.Families[0].Source = "docs"
	config.Navigation.Families[0].Overview = "index.md"
	diagnostic = presentationDiagnostic(t, config.validate())
	if diagnostic.Code != "site.family_overview_invalid" || diagnostic.Pointer != "/navigation/families/0/overview" {
		t.Fatalf("overview diagnostic = %+v", diagnostic)
	}
}

func TestKnownBindingKindIncludesSiteNavigation(t *testing.T) {
	if !knownBindingKind("site_navigation") {
		t.Fatal("site_navigation is not an allowed binding kind")
	}
}

func validPresentationConfig() Config {
	return Config{
		Version: 1,
		Source:  "docs",
		Site: SiteConfig{
			Name: "Margo", BaseURL: "https://margo.example", Logo: "assets/logo.svg", Icon: "assets/logo.svg",
			SocialImage: SocialImageConfig{Path: "assets/social.jpg", Alt: "Margo preview"},
		},
	}
}

func presentationDiagnostic(t *testing.T, err error) margo.Diagnostic {
	t.Helper()
	if err == nil {
		t.Fatal("expected diagnostic")
	}
	var diagnosticError *margo.DiagnosticError
	if !errors.As(err, &diagnosticError) || len(diagnosticError.Diagnostics) != 1 {
		t.Fatalf("error = %v, want one diagnostic", err)
	}
	return diagnosticError.Diagnostics[0]
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}
