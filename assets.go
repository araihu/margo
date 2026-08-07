package margo

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// embeddedAssets contains only the reviewed, library-scoped assets. Runtime
// and social assets are owned by later tasks and are intentionally absent.
//
//go:embed assets/*.css assets/*.svg
var embeddedAssets embed.FS

// AssetRef identifies a validated asset and, for overrides, carries its
// already-materialized bytes. Callers cannot make an override fetch at render
// time.
type AssetRef struct {
	Path      string
	MediaType string
	SHA256    string
	Content   []byte
}

func (a AssetRef) clone() AssetRef {
	a.Content = append([]byte(nil), a.Content...)
	return a
}

// EmbeddedAsset returns one of the assets reviewed into the binary.
func EmbeddedAsset(name string) (AssetRef, error) {
	if err := validateAssetPath(name); err != nil {
		return AssetRef{}, err
	}
	data, err := fs.ReadFile(embeddedAssets, path.Join("assets", name))
	if err != nil {
		return AssetRef{}, fmt.Errorf("margo: embedded asset %q: %w", name, err)
	}
	hash := sha256.Sum256(data)
	return AssetRef{
		Path:      name,
		MediaType: assetMediaType(name),
		SHA256:    hex.EncodeToString(hash[:]),
		Content:   append([]byte(nil), data...),
	}, nil
}

func (a AssetRef) validate() error {
	if err := validateAssetPath(a.Path); err != nil {
		return err
	}
	if len(a.Content) == 0 {
		return fmt.Errorf("margo: asset %q has no materialized bytes", a.Path)
	}
	if a.MediaType != "" && a.MediaType != assetMediaType(a.Path) {
		return fmt.Errorf("margo: asset %q has media type %q, want %q", a.Path, a.MediaType, assetMediaType(a.Path))
	}
	hash := sha256.Sum256(a.Content)
	actual := hex.EncodeToString(hash[:])
	if a.SHA256 != "" && a.SHA256 != actual {
		return fmt.Errorf("margo: asset %q SHA-256 mismatch", a.Path)
	}
	return nil
}

func validateAssetPath(name string) error {
	if name == "" || strings.ContainsAny(name, `\\\x00`) || path.IsAbs(name) || path.Clean(name) != name || name == "." || strings.HasPrefix(name, "../") || strings.Contains(name, "/../") {
		return fmt.Errorf("margo: invalid asset path %q", name)
	}
	return nil
}

func assetMediaType(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".css":
		return "text/css"
	case ".svg":
		return "image/svg+xml"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	default:
		return "application/octet-stream"
	}
}

// AssetHandler serves only the embedded non-runtime asset set. Mount it at
// /assets/ in an embedded application.
func AssetHandler() http.Handler {
	assetFS, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/assets/", http.FileServer(http.FS(assetFS)))
}
