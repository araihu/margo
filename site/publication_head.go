package site

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html"
	"image"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/internal/staticimage"
)

var iconSizes = regexp.MustCompile(`^(any|[1-9][0-9]{0,3}x[1-9][0-9]{0,3}( [1-9][0-9]{0,3}x[1-9][0-9]{0,3})*)$`)
var themeColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (config *Config) validatePublicationHead() error {
	invalid := func(message, field string) error {
		return diagnostic("site.publication_invalid", message, "Use documented publication and site head configuration.", field)
	}
	if value := config.Publication.DateFormat; value != "" && (strings.TrimSpace(value) == "" || len(value) > 128 || strings.ContainsAny(value, "\x00\r\n")) {
		return invalid("date_format must be a non-blank Go time layout of at most 128 bytes", "publication.date_format")
	}
	if value := config.Site.ApplicationName; value != "" && (strings.TrimSpace(value) == "" || len(value) > 256 || strings.ContainsAny(value, "\x00\r\n")) {
		return invalid("application_name must be plain text of at most 256 bytes", "site.application_name")
	}
	if value := config.Site.ThemeColor; value != "" && !themeColor.MatchString(value) {
		return invalid("theme_color must be a six-digit hexadecimal color", "site.theme_color")
	}
	if value := config.Site.WebManifest; value != "" && (!validConfigPath(value) || strings.ContainsAny(value, ":?#%")) {
		return invalid("webmanifest must be a normalized local path", "site.webmanifest")
	}
	if len(config.Site.Icons) > 32 {
		return invalid("icons exceeds the 32-entry limit", "site.icons")
	}
	seen := map[string]bool{}
	for i := range config.Site.Icons {
		icon := &config.Site.Icons[i]
		if icon.Rel == "" {
			icon.Rel = "icon"
		}
		if icon.Rel != "icon" && icon.Rel != "shortcut icon" && icon.Rel != "apple-touch-icon" {
			return invalid("icon rel must be icon, shortcut icon or apple-touch-icon", "site.icons")
		}
		if !validConfigPath(icon.Path) || strings.ContainsAny(icon.Path, ":?#%") || icon.Sizes != "" && !iconSizes.MatchString(icon.Sizes) {
			return invalid("icons require a normalized local path and valid sizes", "site.icons")
		}
		key := icon.Rel + "\x00" + icon.Path + "\x00" + icon.Sizes
		if seen[key] {
			return invalid("duplicate icon declaration", "site.icons")
		}
		seen[key] = true
	}
	return nil
}

func (b *builder) stagePublicationHeadAssets() error {
	for _, icon := range b.config.Site.Icons {
		data, err := (margo.FilesystemCheckAssetReader{}).ReadAsset(context.Background(), b.configDir, icon.Path, 1<<20)
		if err != nil {
			return diagnostic("site.asset_unreadable", err.Error(), "Add the declared icon beside site.yaml.", icon.Path)
		}
		if len(data) == 0 || len(data) > 1<<20 {
			return diagnostic("site.asset_invalid", "icon must be non-empty and at most 1 MiB", "Use a small browser icon.", icon.Path)
		}
		media, err := browserIconMedia(data)
		if err != nil {
			return diagnostic("site.asset_invalid", err.Error(), "Use PNG, safe SVG or ICO browser icons.", icon.Path)
		}
		if icon.Rel == "apple-touch-icon" && media != "image/png" {
			return diagnostic("site.asset_invalid", "Apple Touch icons must be PNG", "Use a PNG Apple Touch icon.", icon.Path)
		}
		if icon.Sizes != "" && media == "image/png" {
			size, _, decodeErr := image.DecodeConfig(bytes.NewReader(data))
			if decodeErr != nil || icon.Sizes != fmt.Sprintf("%dx%d", size.Width, size.Height) {
				return diagnostic("site.asset_invalid", "PNG icon sizes must match its decoded dimensions", "Declare the actual PNG dimensions.", icon.Path)
			}
		}
		if media == "image/svg+xml" && icon.Sizes != "" && icon.Sizes != "any" {
			return diagnostic("site.asset_invalid", "SVG icon sizes must be any or omitted", "Use sizes: any for SVG.", icon.Path)
		}
		if err := b.addArtifact(icon.Path, data); err != nil {
			return err
		}
		b.configuredAssets[icon.Path] = cachedAsset{content: data, mediaType: media}
	}
	if name := b.config.Site.WebManifest; name != "" {
		data, err := (margo.FilesystemCheckAssetReader{}).ReadAsset(context.Background(), b.configDir, name, 64<<10)
		if err != nil {
			return diagnostic("site.asset_unreadable", err.Error(), "Add the declared webmanifest beside site.yaml.", name)
		}
		var object map[string]json.RawMessage
		if len(data) > 64<<10 || json.Unmarshal(data, &object) != nil || object == nil {
			return diagnostic("site.asset_invalid", "webmanifest must be a JSON object of at most 64 KiB", "Supply a valid local webmanifest.", name)
		}
		if err := b.addArtifact(name, data); err != nil {
			return err
		}
	}
	return nil
}

func browserIconMedia(data []byte) (string, error) {
	if http.DetectContentType(data) == "image/x-icon" {
		if len(data) < 6 {
			return "", fmt.Errorf("invalid ICO header")
		}
		count := int(binary.LittleEndian.Uint16(data[4:6]))
		if count == 0 || count > (len(data)-6)/16 {
			return "", fmt.Errorf("invalid ICO directory")
		}
		for i := 0; i < count; i++ {
			entry := data[6+i*16 : 6+(i+1)*16]
			size, offset := uint64(binary.LittleEndian.Uint32(entry[8:12])), uint64(binary.LittleEndian.Uint32(entry[12:16]))
			if size == 0 || offset < uint64(6+count*16) || offset+size > uint64(len(data)) {
				return "", fmt.Errorf("invalid ICO image bounds")
			}
		}
		return "image/x-icon", nil
	}
	media, err := staticimage.DetectContext(context.Background(), data)
	if err != nil {
		return "", err
	}
	if media != "image/png" && media != "image/svg+xml" {
		return "", fmt.Errorf("unsupported browser icon format %q", media)
	}
	if media == "image/png" {
		if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
			return "", err
		}
	}
	return media, nil
}

func (b *builder) publicationBrandHead(defaultIconURL string) string {
	var out strings.Builder
	// A declaration of the same icon replaces the default link rather than duplicating it.
	defaultDeclared := false
	for _, icon := range b.config.Site.Icons {
		if icon.Rel == "icon" && icon.Path == b.config.Site.Icon {
			defaultDeclared = true
		}
	}
	if !defaultDeclared {
		fmt.Fprintf(&out, `<link rel="icon" href="%s">`, html.EscapeString(defaultIconURL))
	}
	for _, icon := range b.config.Site.Icons {
		fmt.Fprintf(&out, `<link rel="%s" href="%s" type="%s"`, icon.Rel, html.EscapeString(b.publicHeadAssetPath(icon.Path)), b.configuredAssets[icon.Path].mediaType)
		if icon.Sizes != "" {
			fmt.Fprintf(&out, ` sizes="%s"`, icon.Sizes)
		}
		out.WriteString(">")
	}
	if name := b.config.Site.WebManifest; name != "" {
		fmt.Fprintf(&out, `<link rel="manifest" href="%s" type="application/manifest+json">`, html.EscapeString(b.publicHeadAssetPath(name)))
	}
	if name := b.config.Site.ApplicationName; name != "" {
		fmt.Fprintf(&out, `<meta name="application-name" content="%s"><meta name="apple-mobile-web-app-title" content="%s">`, html.EscapeString(name), html.EscapeString(name))
	}
	if color := b.config.Site.ThemeColor; color != "" {
		fmt.Fprintf(&out, `<meta name="theme-color" content="%s">`, color)
	}
	return out.String()
}

func (b *builder) publicHeadAssetPath(name string) string {
	return (&url.URL{Path: strings.TrimSuffix(normalizedBasePath(b.config.BasePath), "/") + "/" + name}).String()
}
