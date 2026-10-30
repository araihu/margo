package site

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"image"
	"net/url"
	"path"
	"strings"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/internal/staticimage"
	_ "golang.org/x/image/webp"
)

// socialPreview describes a validated, published local preview image.
type socialPreview struct {
	URL, Alt, MediaType string
	Width, Height       int
}

func (b *builder) publicAssetURL(name string) string {
	prefix := strings.TrimSuffix(normalizedBasePath(b.config.BasePath), "/")
	return strings.TrimSuffix(b.config.Site.BaseURL, "/") + (&url.URL{Path: prefix + "/" + name}).String()
}

func (b *builder) preparePageSocialImage(ctx context.Context, source Source, metadata margo.Metadata) (*socialPreview, error) {
	raw, exists := metadata.Additional["social_image"]
	if !exists {
		return nil, nil
	}
	invalid := func(message string) (*socialPreview, error) {
		return nil, diagnostic("site.social_image_invalid", message, "Use social_image with a source-relative PNG, JPEG, GIF or WebP path and non-empty alt text.", source.Path)
	}
	if b.config.Site.BaseURL == "" {
		return invalid("page social images require an absolute HTTPS site.base_url")
	}
	object, ok := raw.(map[string]any)
	if !ok || len(object) != 2 {
		return invalid("social_image requires only path and alt")
	}
	filename, pathOK := object["path"].(string)
	alt, altOK := object["alt"].(string)
	relative := strings.TrimPrefix(filename, "./")
	if !pathOK || !validConfigPath(relative) || strings.ContainsAny(relative, ":?#%") || !altOK || strings.TrimSpace(alt) == "" || len([]byte(alt)) > 512 || strings.ContainsAny(alt, "\x00\r\n") {
		return invalid("social_image requires a normalized local path and plain alternative text of at most 512 UTF-8 bytes")
	}
	assetPath := path.Join(path.Dir(source.Path), relative)
	asset, cached := b.assets[assetPath]
	if !cached {
		limit := min(int64(8<<20), b.sourceAssetLimit())
		if limit <= 0 {
			return invalid("social image exceeds the remaining site asset byte budget")
		}
		data, err := b.request.AssetReader.ReadAsset(ctx, b.request.SourceRoot, assetPath, limit)
		if err != nil {
			return invalid(fmt.Sprintf("cannot read social image %q: %v", filename, err))
		}
		if int64(len(data)) > limit {
			return invalid("social image exceeds the remaining site asset byte budget or 8 MiB")
		}
		asset = cachedAsset{content: data}
	}
	if len(asset.content) == 0 || len(asset.content) > 8<<20 {
		return invalid("social image must be non-empty and at most 8 MiB")
	}
	media, err := staticimage.DetectContext(ctx, asset.content)
	if err != nil {
		return invalid(fmt.Sprintf("invalid social image: %v", err))
	}
	switch media {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
	default:
		return invalid("unsupported social image format")
	}
	dimensions, _, err := image.DecodeConfig(bytes.NewReader(asset.content))
	if err != nil || dimensions.Width <= 0 || dimensions.Height <= 0 {
		return invalid("social image has invalid dimensions")
	}
	if !cached {
		if _, err := b.cacheSourceAsset(ctx, assetPath, asset.content); err != nil {
			return nil, err
		}
	}
	// Social crawlers need an HTTP URL even when document assets are inline.
	if err := b.addArtifact(assetPath, asset.content); err != nil {
		return nil, err
	}
	return &socialPreview{URL: b.publicAssetURL(assetPath), Alt: alt, MediaType: media, Width: dimensions.Width, Height: dimensions.Height}, nil
}

func (b *builder) pageSocialPreview(page Page) socialPreview {
	preview := socialPreview{URL: b.socialURL(), Alt: b.config.Site.SocialImage.Alt, MediaType: b.socialMediaType, Width: 1280, Height: 640}
	if image := b.configured[page.Source].socialImage; image != nil {
		preview = *image
	}
	return preview
}

func (b *builder) socialHead(page Page) string {
	preview := b.pageSocialPreview(page)
	var out strings.Builder
	for _, tag := range []struct{ key, value string }{
		{"og:url", page.Canonical}, {"og:type", "website"}, {"og:title", page.Title}, {"og:description", page.Description},
		{"og:site_name", b.config.Site.Name}, {"og:image", preview.URL}, {"og:image:type", preview.MediaType},
		{"og:image:width", fmt.Sprint(preview.Width)}, {"og:image:height", fmt.Sprint(preview.Height)}, {"og:image:alt", preview.Alt}, {"og:locale", openGraphLocale(page.Locale)},
	} {
		fmt.Fprintf(&out, `<meta property="%s" content="%s">`, tag.key, html.EscapeString(tag.value))
	}
	for _, tag := range []struct{ key, value string }{
		{"twitter:card", "summary_large_image"}, {"twitter:title", page.Title}, {"twitter:description", page.Description}, {"twitter:image", preview.URL}, {"twitter:image:alt", preview.Alt},
	} {
		fmt.Fprintf(&out, `<meta name="%s" content="%s">`, tag.key, html.EscapeString(tag.value))
	}
	return out.String()
}
