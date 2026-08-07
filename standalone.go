package margo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/a-h/templ"
	goshtosoassets "github.com/araihu/goshtoso/assets"
)

// HeadOwnerSelection freezes one metadata owner before the social task. It is
// intentionally immutable from the standalone API: later tasks can verify it
// but cannot replace it.
type HeadOwnerSelection struct {
	SchemaVersion   string `json:"schemaVersion"`
	Owner           string `json:"owner"`
	Primitive       string `json:"primitive"`
	GoshtosoCommit  string `json:"goshtosoCommit"`
	GoshtosoTree    string `json:"goshtosoTree"`
	APISourcePath   string `json:"apiSourcePath"`
	APISourceSHA256 string `json:"apiSourceSHA256"`
}

var hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

var frozenHeadOwnerSelection = HeadOwnerSelection{
	SchemaVersion:   "margo/head-owner-selection/v1",
	Owner:           "margo",
	Primitive:       "socialMetadataTags",
	GoshtosoCommit:  "module:v0.1.2",
	GoshtosoTree:    "module-cache:v0.1.2",
	APISourcePath:   "components/head/component.go",
	APISourceSHA256: "833562eafa47d917587c21e300d28c45006b855a569266b96041123ca870b3fb",
}

// FrozenHeadOwnerSelection returns the C6 selection by value.
func FrozenHeadOwnerSelection() HeadOwnerSelection { return frozenHeadOwnerSelection }

// Validate checks the exact closed selection contract.
func (s HeadOwnerSelection) Validate() error {
	if s.SchemaVersion != "margo/head-owner-selection/v1" {
		return fmt.Errorf("margo: unsupported head-owner schema %q", s.SchemaVersion)
	}
	if (s.Owner == "goshtoso" && s.Primitive != "head.Metadata") || (s.Owner == "margo" && s.Primitive != "socialMetadataTags") || (s.Owner != "goshtoso" && s.Owner != "margo") {
		return fmt.Errorf("margo: invalid head owner/primitive pair %q/%q", s.Owner, s.Primitive)
	}
	if s.GoshtosoCommit == "" || s.GoshtosoTree == "" || s.APISourcePath == "" || !hex64.MatchString(s.APISourceSHA256) {
		return fmt.Errorf("margo: incomplete head-owner evidence")
	}
	if strings.ContainsAny(s.APISourcePath, "\\\x00\n\r") || strings.HasPrefix(s.APISourcePath, "/") || strings.Contains(s.APISourcePath, "..") {
		return fmt.Errorf("margo: invalid head-owner source path")
	}
	return nil
}

// ParseHeadOwnerSelection rejects unknown fields and non-canonical values.
func ParseHeadOwnerSelection(data []byte) (HeadOwnerSelection, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var selection HeadOwnerSelection
	if err := decoder.Decode(&selection); err != nil {
		return HeadOwnerSelection{}, fmt.Errorf("margo: parse head-owner selection: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return HeadOwnerSelection{}, fmt.Errorf("margo: trailing head-owner selection data")
		}
		return HeadOwnerSelection{}, fmt.Errorf("margo: trailing head-owner selection data: %w", err)
	}
	if err := selection.Validate(); err != nil {
		return HeadOwnerSelection{}, err
	}
	return selection, nil
}

type standaloneConfig struct {
	lang           string
	title          string
	description    string
	theme          ThemeName
	tokens         map[DocumentToken]string
	brand          Brand
	assetOverrides map[string]AssetRef
}

// StandaloneOption configures the self-contained HTML shell.
type StandaloneOption func(*standaloneConfig) error

func defaultStandaloneConfig(result *RenderResult) (standaloneConfig, error) {
	tokens, err := defaultThemeTokens(ThemeModern)
	if err != nil {
		return standaloneConfig{}, err
	}
	title := "Margo document"
	if result != nil && result.Metadata().Title != "" {
		title = result.Metadata().Title
	}
	return standaloneConfig{
		lang: "en", title: title, theme: ThemeModern, tokens: tokens,
		assetOverrides: make(map[string]AssetRef),
	}, nil
}

// WithStandaloneTheme selects the closed theme set for standalone output.
func WithStandaloneTheme(theme ThemeName) StandaloneOption {
	return func(config *standaloneConfig) error {
		tokens, err := defaultThemeTokens(theme)
		if err != nil {
			return err
		}
		config.theme, config.tokens = theme, tokens
		return nil
	}
}

// WithPageTitle sets the escaped document title.
func WithPageTitle(title string) StandaloneOption {
	return func(config *standaloneConfig) error {
		if strings.TrimSpace(title) == "" || len([]byte(title)) > 256 {
			return fmt.Errorf("margo: invalid standalone title")
		}
		config.title = title
		return nil
	}
}

// WithPageDescription sets the optional escaped description.
func WithPageDescription(description string) StandaloneOption {
	return func(config *standaloneConfig) error {
		if len([]byte(description)) > 512 {
			return fmt.Errorf("margo: standalone description is too long")
		}
		config.description = description
		return nil
	}
}

// WithThemeTokens applies only the supported, bounded token keys.
func WithThemeTokens(tokens map[DocumentToken]string) StandaloneOption {
	return func(config *standaloneConfig) error {
		if err := validateThemeTokens(tokens); err != nil {
			return err
		}
		for key, value := range tokens {
			config.tokens[key] = value
		}
		return nil
	}
}

// WithBrand applies trusted header/footer components and validated declarative
// brand values.
func WithBrand(brand Brand) StandaloneOption {
	return func(config *standaloneConfig) error {
		if err := brand.Validate(); err != nil {
			return err
		}
		config.brand = brand.clone()
		return nil
	}
}

// WithAssetOverride supplies already-materialized bytes for one embedded asset.
func WithAssetOverride(name string, asset AssetRef) StandaloneOption {
	return func(config *standaloneConfig) error {
		if name != asset.Path {
			return fmt.Errorf("margo: asset override name/path mismatch")
		}
		if err := asset.validate(); err != nil {
			return err
		}
		config.assetOverrides[name] = asset.clone()
		return nil
	}
}

// RenderStandalone assembles a deterministic, offline HTML component. The
// variadic any accepts both standalone options and the existing compiler
// WithTheme option for ergonomic compatibility; unsupported compiler options
// are rejected.
func RenderStandalone(result *RenderResult, options ...any) (templ.Component, error) {
	if result == nil || result.Content() == nil {
		return nil, fmt.Errorf("margo: standalone render requires a result")
	}
	config, err := defaultStandaloneConfig(result)
	if err != nil {
		return nil, err
	}
	for index, option := range options {
		switch typed := option.(type) {
		case StandaloneOption:
			if typed == nil {
				return nil, fmt.Errorf("margo: nil standalone option at index %d", index)
			}
			if err := typed(&config); err != nil {
				return nil, err
			}
		case Option:
			// Existing WithTheme is an Option. Apply it to a temporary config
			// and import only its immutable theme value.
			compilerConfig := newCompilerConfig()
			if err := typed(&compilerConfig); err != nil {
				return nil, err
			}
			if theme, ok := compilerConfig.values["theme"].(string); ok {
				if err := WithStandaloneTheme(ThemeName(theme))(&config); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("margo: compiler option is not valid for standalone output")
			}
		default:
			return nil, fmt.Errorf("margo: unsupported standalone option %T at index %d", option, index)
		}
	}
	if err := config.brand.Validate(); err != nil {
		return nil, err
	}
	selection := FrozenHeadOwnerSelection()
	if err := selection.Validate(); err != nil {
		return nil, err
	}
	goshtosoCSS, err := goshtosoassets.StylesCSS()
	if err != nil {
		return nil, fmt.Errorf("margo: read embedded Goshtoso stylesheet: %w", err)
	}
	asset, ok := config.assetOverrides["document.css"]
	if !ok {
		asset, err = EmbeddedAsset("document.css")
		if err != nil {
			return nil, err
		}
	}
	content, err := renderComponentBytes(result.Content())
	if err != nil {
		return nil, fmt.Errorf("margo: render standalone content: %w", err)
	}
	hash := sha256.Sum256(append([]byte("margo/standalone-document/v1\n"), content...))
	fingerprint := hex.EncodeToString(hash[:])
	css := applyThemeTokens(string(asset.Content), config.tokens)
	styles := templ.Raw(`<style data-margo-stylesheet="goshtoso">` + string(goshtosoCSS) + `</style><style data-margo-stylesheet="document">` + css + `</style>`)
	return standaloneDocument(config.lang, config.theme, config.title, config.description, fingerprint, styles, config.brand, templ.Raw(string(content))), nil
}

// Standalone is a short alias for RenderStandalone.
func Standalone(result *RenderResult, options ...any) (templ.Component, error) {
	return RenderStandalone(result, options...)
}

func renderComponentBytes(component templ.Component) ([]byte, error) {
	var buffer bytes.Buffer
	if err := component.Render(nilContext(), &buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func nilContext() context.Context { return context.Background() }

func applyThemeTokens(css string, tokens map[DocumentToken]string) string {
	declarations := make([]string, 0, len(tokens))
	for _, token := range []DocumentToken{TokenFontBody, TokenFontHeading, TokenContentWidth, TokenLineHeight, TokenCodeTheme, TokenPageBackground} {
		if value, ok := tokens[token]; ok {
			declarations = append(declarations, fmt.Sprintf("%s:%s", token, value))
		}
	}
	return strings.Replace(css, "/* MARGO_THEME_TOKENS */", strings.Join(declarations, ";")+";", 1)
}
