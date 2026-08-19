package site

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/pdf"
	"github.com/araihu/margo/ssg"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// Config is the v1 declarative documentation-site configuration.
type Config struct {
	Version    int                      `yaml:"version"`
	Source     string                   `yaml:"source"`
	Output     string                   `yaml:"output"`
	Assets     string                   `yaml:"assets"`
	Offline    *bool                    `yaml:"offline"`
	BasePath   string                   `yaml:"base_path"`
	Site       SiteConfig               `yaml:"site"`
	Layout     *LayoutConfig            `yaml:"layout"`
	Frame      *LayoutSelection         `yaml:"frame"`
	Shell      *LayoutSelection         `yaml:"shell"`
	Locales    LocaleConfig             `yaml:"locales"`
	Navigation NavigationConfig         `yaml:"navigation"`
	Bindings   map[string]BindingConfig `yaml:"bindings"`
	Themes     []ThemeConfig            `yaml:"themes"`
	CustomCSS  []CSSConfig              `yaml:"custom_css"`
	Theme      ThemeSelection           `yaml:"theme"`
}

type SiteConfig struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	Version       string            `yaml:"version"`
	RepositoryURL string            `yaml:"repository_url"`
	BaseURL       string            `yaml:"base_url"`
	Home          string            `yaml:"home"`
	Logo          string            `yaml:"logo"`
	Icon          string            `yaml:"icon"`
	SocialImage   SocialImageConfig `yaml:"social_image"`
}

type SocialImageConfig struct {
	Path string `yaml:"path"`
	Alt  string `yaml:"alt"`
}

type LayoutSelection struct {
	Builtin  string          `yaml:"builtin"`
	Command  string          `yaml:"command"`
	Protocol string          `yaml:"protocol"`
	Values   map[string]any  `yaml:"values"`
	GoModule *GoModuleConfig `yaml:"go_module"`
}

type GoModuleConfig struct {
	Import      string         `yaml:"import"`
	Version     string         `yaml:"version"`
	Constructor string         `yaml:"constructor"`
	Values      map[string]any `yaml:"values"`
}

type LocaleConfig struct {
	Default   string   `yaml:"default"`
	Supported []string `yaml:"supported"`
}

type NavigationConfig struct {
	Mode    string   `yaml:"mode"`
	Exclude []string `yaml:"exclude"`
}

type BindingConfig struct {
	Area  string         `yaml:"area"`
	Slot  string         `yaml:"slot"`
	Props map[string]any `yaml:"props"`
}

type ThemeConfig struct {
	Name         string `yaml:"name"`
	CSSURL       string `yaml:"css_url"`
	TokenCatalog string `yaml:"token_catalog"`
}

type CSSConfig struct {
	CSSURL string `yaml:"css_url"`
}

type ThemeSelection struct {
	Builtin          bool   `yaml:"builtin"`
	Name             string `yaml:"name"`
	AllowSwitchTheme bool   `yaml:"allow_switch_theme"`
	ColorMode        string `yaml:"color_mode"`
}

// ConfigRequest builds a site.yaml without publishing it.
type ConfigRequest struct {
	ConfigPath  string
	Compiler    *margo.Compiler
	AssetReader margo.CheckAssetReader
	PDFEngine   pdf.Engine
}

func LoadConfig(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("site.config_read: %w", err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, diagnostic("site.config_invalid", err.Error(), "Correct site.yaml and rebuild.", filename)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return Config{}, diagnostic("site.config_invalid", "site.yaml contains more than one YAML document", "Keep one configuration document.", filename)
	} else if !errors.Is(err, io.EOF) {
		return Config{}, diagnostic("site.config_invalid", err.Error(), "Correct site.yaml and rebuild.", filename)
	}
	if err := config.validate(); err != nil {
		return Config{}, attachConfigPath(err, filename)
	}
	return config, nil
}

func (config *Config) validate() error {
	if config.Version != 1 {
		return diagnostic("site.config_version", fmt.Sprintf("unsupported config version %d", config.Version), "Use version: 1.", "")
	}
	if config.Source == "" || !validConfigPath(config.Source) {
		return diagnostic("site.source_invalid", "source must be a normalized relative directory", "Set source to a directory below site.yaml.", "")
	}
	if config.Output == "" {
		config.Output = "dist"
	}
	if !validConfigPath(config.Output) {
		return diagnostic("site.output_invalid", "output must be a normalized relative directory", "Set output to a directory below site.yaml.", "")
	}
	if config.Assets == "" {
		config.Assets = string(AssetsLocal)
	}
	if config.Assets != string(AssetsLocal) && config.Assets != string(AssetsInline) {
		return diagnostic("site.assets_invalid", "assets must be local or inline", "Use assets: local.", "")
	}
	if config.BasePath != "" {
		if _, err := normalizeBasePath(config.BasePath); err != nil {
			return diagnostic("site.base_path_invalid", err.Error(), "Use / or a normalized path such as /docs.", "")
		}
	}
	if config.Site.BaseURL != "" {
		if err := validateOrigin(config.Site.BaseURL); err != nil {
			return diagnostic("site.base_url_invalid", err.Error(), "Use an absolute HTTPS origin without a path.", "")
		}
	}
	if config.Site.Version != "" && (strings.ContainsAny(config.Site.Version, "\x00\r\n") || len([]byte(config.Site.Version)) > 64) {
		return diagnostic("site.version_invalid", "site.version is empty, too long, or contains control characters", "Use a concise release or development label.", "site.version")
	}
	if config.Site.RepositoryURL != "" {
		repository, repositoryErr := url.Parse(config.Site.RepositoryURL)
		if repositoryErr != nil || repository.Scheme != "https" || repository.Host == "" || repository.User != nil || repository.RawQuery != "" || repository.Fragment != "" {
			return diagnostic("site.repository_invalid", "site.repository_url must be an absolute HTTPS URL", "Use the public source repository URL.", "site.repository_url")
		}
	}
	for name, value := range map[string]string{"name": config.Site.Name, "logo": config.Site.Logo, "icon": config.Site.Icon, "social_image.path": config.Site.SocialImage.Path, "social_image.alt": config.Site.SocialImage.Alt} {
		if strings.TrimSpace(value) == "" {
			return diagnostic("site.identity_required", fmt.Sprintf("site.%s is required", name), "Declare complete site identity and social image metadata.", "")
		}
	}
	for field, value := range map[string]string{"logo": config.Site.Logo, "icon": config.Site.Icon, "social_image.path": config.Site.SocialImage.Path} {
		if !validConfigPath(value) {
			return diagnostic("site.asset_path_invalid", fmt.Sprintf("site.%s must be a normalized relative path", field), "Use a local asset path below site.yaml.", "")
		}
	}
	if strings.ContainsAny(config.Site.SocialImage.Alt, "\x00\r\n") || len([]byte(config.Site.SocialImage.Alt)) > 160 {
		return diagnostic("site.social_image_invalid", "social image alt text is empty, too long, or contains control characters", "Use concise alternative text.", "")
	}
	if config.Site.Home == "" {
		config.Site.Home = "index.md"
	}
	if _, ok := validSourcePath(config.Site.Home); !ok {
		return diagnostic("site.home_invalid", fmt.Sprintf("site.home %q is not a normalized Markdown path", config.Site.Home), "Use a relative Markdown path below source, such as index.md.", "")
	}
	if config.Offline == nil {
		offline := true
		config.Offline = &offline
	}
	if config.Layout != nil {
		if config.Frame != nil || config.Shell != nil {
			return newPresentationDiagnostic("site.layout_conflict", "layout cannot be combined with frame or shell", "Remove frame or shell when using typed layout.", "/layout")
		}
		if err := validateSiteLayout(config.Layout); err != nil {
			return err
		}
	}
	if config.Frame != nil && config.Shell != nil {
		return diagnostic("site.layout_conflict", "frame and shell are mutually exclusive", "Select one layout kind.", "")
	}
	if config.Frame != nil {
		if err := validateLayoutSelection(*config.Frame, false); err != nil {
			return err
		}
	}
	if config.Shell != nil {
		if err := validateLayoutSelection(*config.Shell, true); err != nil {
			return err
		}
	}
	if config.Locales.Default == "" {
		config.Locales.Default = "en"
	}
	if len(config.Locales.Supported) == 0 {
		config.Locales.Supported = []string{config.Locales.Default}
	}
	if err := validateLocales(config.Locales); err != nil {
		return err
	}
	if config.Navigation.Mode == "" {
		config.Navigation.Mode = "file-tree"
	}
	if config.Navigation.Mode != "file-tree" {
		return diagnostic("site.navigation_invalid", fmt.Sprintf("unsupported navigation mode %q", config.Navigation.Mode), "Use navigation.mode: file-tree.", "")
	}
	if config.Theme.Name == "" {
		config.Theme.Name = "modern"
	}
	if config.Theme.ColorMode == "" {
		config.Theme.ColorMode = "system"
	}
	if config.Theme.ColorMode != "system" && config.Theme.ColorMode != "light" && config.Theme.ColorMode != "dark" {
		return diagnostic("site.theme_invalid", fmt.Sprintf("unsupported color_mode %q", config.Theme.ColorMode), "Use system, light, or dark.", "")
	}
	if !validThemeName(config.Theme.Name) {
		return diagnostic("site.theme_invalid", fmt.Sprintf("invalid theme name %q", config.Theme.Name), "Use lowercase letters, numbers, hyphens, and underscores.", "")
	}
	themeNames := map[string]struct{}{"modern": {}}
	for _, theme := range config.Themes {
		if !validThemeName(theme.Name) || theme.CSSURL == "" || !validConfigPathOrHTTPS(theme.CSSURL) || !validConfigPath(theme.TokenCatalog) {
			return diagnostic("site.theme_invalid", fmt.Sprintf("invalid custom theme %q", theme.Name), "Declare a valid name and local css_url.", "")
		}
		if _, exists := themeNames[theme.Name]; exists {
			return diagnostic("site.theme_duplicate", fmt.Sprintf("theme %q is declared more than once", theme.Name), "Use one name per theme catalog.", "")
		}
		themeNames[theme.Name] = struct{}{}
	}
	for _, css := range config.CustomCSS {
		if css.CSSURL == "" || !validConfigPathOrHTTPS(css.CSSURL) {
			return diagnostic("site.css_invalid", fmt.Sprintf("invalid custom CSS %q", css.CSSURL), "Declare a normalized local css_url.", "")
		}
	}
	if _, exists := themeNames[config.Theme.Name]; !exists {
		return diagnostic("site.theme_unavailable", fmt.Sprintf("configured theme %q is not in the catalog", config.Theme.Name), "Add the theme to themes or select modern.", "")
	}
	for kind, binding := range config.Bindings {
		if !knownBindingKind(kind) || binding.Area == "" {
			return diagnostic("site.binding_invalid", fmt.Sprintf("invalid binding %q", kind), "Use a known semantic provider kind and area.", "")
		}
	}
	for _, pattern := range config.Navigation.Exclude {
		cleaned := path.Clean(pattern)
		if _, err := path.Match(pattern, ""); err != nil || pattern == "" || strings.HasPrefix(pattern, "/") || strings.ContainsAny(pattern, "\\\x00\r\n") || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
			return diagnostic("site.navigation_invalid", fmt.Sprintf("invalid navigation.exclude pattern %q", pattern), "Use normalized paths such as drafts/**.", "")
		}
	}
	return nil
}

func validateSiteLayout(layout *LayoutConfig) error {
	if layout == nil {
		return nil
	}
	if _, err := resolveSiteLayout(*layout, ""); err != nil {
		return err
	}
	if layout.Kind != LayoutDocs {
		return nil
	}

	if layout.Default == nil {
		layout.Default = make(map[string]any)
	}
	rawFamilies, configured := layout.Default["families"]
	if !configured {
		layout.Default["families"] = []any{"default"}
		return nil
	}
	families, _ := layoutListValues(rawFamilies)
	seen := make(map[string]int, len(families))
	normalized := make([]any, 0, len(families)+1)
	for index, value := range families {
		family := strings.TrimSpace(value.(string))
		pointer := fmt.Sprintf("/layout/default/families/%d", index)
		if family == "" {
			return newPresentationDiagnostic("site.family_invalid", "family identifier must not be empty", "Declare a stable non-empty family identifier.", pointer)
		}
		if previous, exists := seen[family]; exists {
			return newPresentationDiagnostic("site.family_duplicate", fmt.Sprintf("family %q is declared more than once (entries %d and %d)", family, previous, index), "Declare each family once.", pointer)
		}
		seen[family] = index
		if family != "default" {
			normalized = append(normalized, family)
		}
	}
	layout.Default["families"] = append([]any{"default"}, normalized...)
	return nil
}

func validateLayoutSelection(selection LayoutSelection, shell bool) error {
	count := 0
	if selection.Builtin != "" {
		count++
	}
	if selection.Command != "" {
		count++
	}
	if selection.GoModule != nil {
		count++
	}
	if count != 1 {
		return diagnostic("site.layout_invalid", "exactly one of builtin, command, or go_module is required", "Select one distribution mode.", "")
	}
	if selection.Command != "" && selection.Protocol == "" {
		return diagnostic("site.layout_protocol_required", "command layouts require protocol", "Declare margo.ssg.frame/v1 or margo.ssg.shell/v1.", "")
	}
	if selection.Command != "" && selection.Protocol != map[bool]string{false: ssg.FrameContract, true: ssg.ShellContract}[shell] {
		return diagnostic("site.layout_protocol_invalid", fmt.Sprintf("layout protocol %q does not match selected kind", selection.Protocol), "Use the selected frame or shell contract.", "")
	}
	if selection.Builtin != "" && selection.Protocol != "" {
		return diagnostic("site.layout_protocol_invalid", "builtin layouts must not declare a command protocol", "Declare protocol only for command layouts.", "")
	}
	if selection.GoModule != nil {
		if selection.Protocol != "" {
			return diagnostic("site.layout_protocol_invalid", "Go-module layouts must not declare a command protocol", "Declare protocol only for command layouts.", "")
		}
		if strings.TrimSpace(selection.GoModule.Import) == "" || strings.TrimSpace(selection.GoModule.Version) == "" || strings.TrimSpace(selection.GoModule.Constructor) == "" {
			return diagnostic("site.layout_invalid", "Go-module layouts require import, version, and constructor", "Declare all Go-module constructor fields.", "")
		}
	}
	if selection.Builtin != "" && !shell {
		found := false
		for _, name := range ssg.BuiltinFrameNames() {
			if selection.Builtin == name {
				found = true
				break
			}
		}
		if !found {
			return diagnostic("site.layout_unknown", fmt.Sprintf("unknown builtin frame %q", selection.Builtin), "Choose one of the six builtin frames.", "")
		}
	}
	if selection.Builtin != "" && shell && selection.Builtin != "componentdocshell" {
		return diagnostic("site.layout_unknown", fmt.Sprintf("unknown builtin shell %q", selection.Builtin), "Choose the supported Goshtoso shell: componentdocshell.", "")
	}
	if selection.Builtin == "componentdocshell" && len(selection.Values) > 0 {
		return diagnostic("site.layout_values_invalid", "componentdocshell does not expose builtin structural values", "Remove shell.values or use a frame with published option paths.", "shell.values")
	}
	return nil
}

func validateLocales(config LocaleConfig) error {
	defaultTag, err := language.Parse(config.Default)
	if err != nil || defaultTag.String() != config.Default {
		return diagnostic("site.locale_invalid", fmt.Sprintf("invalid default locale %q", config.Default), "Use canonical BCP 47 spelling.", "")
	}
	foundDefault := false
	seen := map[string]struct{}{}
	for _, value := range config.Supported {
		tag, parseErr := language.Parse(value)
		if parseErr != nil || tag.String() != value {
			return diagnostic("site.locale_invalid", fmt.Sprintf("invalid supported locale %q", value), "Use canonical BCP 47 spelling.", "")
		}
		if _, exists := seen[value]; exists {
			return diagnostic("site.locale_duplicate", fmt.Sprintf("locale %q is repeated", value), "Declare each locale once.", "")
		}
		seen[value] = struct{}{}
		if value == config.Default {
			foundDefault = true
		}
	}
	if !foundDefault {
		return diagnostic("site.locale_default_missing", fmt.Sprintf("default locale %q is not supported", config.Default), "Include default in locales.supported.", "")
	}
	return nil
}

func validateOrigin(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" && parsed.Path != "/" {
		return fmt.Errorf("base_url must be an absolute HTTPS origin")
	}
	return nil
}

func normalizeBasePath(value string) (string, error) {
	if value == "" || value == "/" {
		return "/", nil
	}
	if !strings.HasPrefix(value, "/") || strings.ContainsAny(value, "\\\x00\r\n") {
		return "", fmt.Errorf("base_path must begin with /")
	}
	cleaned := path.Clean(value)
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, "/../") {
		return "", fmt.Errorf("base_path must not escape root")
	}
	return strings.TrimSuffix(cleaned, "/"), nil
}

func validConfigPath(value string) bool {
	return value != "" && !strings.HasPrefix(value, "/") && !strings.ContainsAny(value, "\\\x00\r\n") && path.Clean(value) == value && value != "." && value != ".." && !strings.HasPrefix(value, "../") && !strings.Contains(value, "/../")
}

func validConfigPathOrHTTPS(value string) bool {
	if validConfigPath(value) {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.Path != "" && parsed.RawQuery == "" && parsed.Fragment == ""
}

func validThemeName(value string) bool {
	if value == "" || value[0] < 'a' || value[0] > 'z' || len(value) > 64 {
		return false
	}
	for _, character := range value[1:] {
		if character != '-' && character != '_' && (character < 'a' || character > 'z') && (character < '0' || character > '9') {
			return false
		}
	}
	return true
}

func knownBindingKind(value string) bool {
	switch value {
	case "navigation", "site_navigation", "breadcrumbs", "pagination", "theme_controls", "locale_controls", "toc", "footer":
		return true
	default:
		return false
	}
}

func attachConfigPath(err error, filename string) error {
	if err == nil || filename == "" {
		return err
	}
	var diagnosticError *margo.DiagnosticError
	if errors.As(err, &diagnosticError) {
		for index := range diagnosticError.Diagnostics {
			if diagnosticError.Diagnostics[index].Source == "" {
				diagnosticError.Diagnostics[index].Source = filename
			}
		}
		return diagnosticError
	}
	return diagnostic("site.config_invalid", err.Error(), "Correct site.yaml and rebuild.", filename)
}

// BuildConfig reads site.yaml and builds the deterministic artifact set.
func BuildConfig(ctx context.Context, request ConfigRequest) (Result, error) {
	if request.ConfigPath == "" {
		return Result{}, diagnostic("site.config_required", "config path is required", "Pass site.yaml.", "")
	}
	config, err := LoadConfig(request.ConfigPath)
	if err != nil {
		return Result{}, err
	}
	return buildConfigured(ctx, request, config)
}
