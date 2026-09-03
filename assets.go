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

	goshtosoassets "github.com/araihu/goshtoso/assets"
)

// embeddedAssets contains only the reviewed, library-scoped assets. Runtime
// and social assets are owned by later tasks and are intentionally absent.
//
//go:embed assets/*.css assets/*.js assets/*.svg
var embeddedAssets embed.FS

const (
	HTMLStylesURL       = "/margo-assets/document.css"
	TableSortRuntimeURL = "/margo-assets/table-sort.js"
)

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
	if name == "" || strings.ContainsAny(name, "\\\x00") || path.IsAbs(name) || path.Clean(name) != name || name == "." || strings.HasPrefix(name, "../") || strings.Contains(name, "/../") {
		return fmt.Errorf("margo: invalid asset path %q", name)
	}
	return nil
}

func assetMediaType(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
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
	fileServer := http.StripPrefix("/assets/", http.FileServer(http.FS(assetFS)))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.EqualFold(path.Ext(request.URL.Path), ".js") {
			http.NotFound(writer, request)
			return
		}
		fileServer.ServeHTTP(writer, request)
	})
}

func HTMLAssetHandler() http.Handler {
	assetFS, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		panic(err)
	}
	fileServer := http.StripPrefix("/margo-assets/", http.FileServer(http.FS(assetFS)))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasSuffix(strings.ToLower(request.URL.Path), ".js") {
			writer.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		}
		fileServer.ServeHTTP(writer, request)
	})
}

func coreHTMLRequirements(includeTableSort, includeCodeCopy bool) ([]HTMLRequirement, error) {
	goshtosoStyles, err := goshtosoassets.StylesCSS()
	if err != nil {
		return nil, fmt.Errorf("margo: load Goshtoso styles: %w", err)
	}
	manifest := goshtosoassets.DefaultRuntimeManifest()
	documentStyles, err := EmbeddedAsset("document.css")
	if err != nil {
		return nil, err
	}
	requirements := []HTMLRequirement{
		{
			ID: "goshtoso.styles", Kind: HTMLStylesheet,
			LocalURL: manifest.Stylesheet.LocalURL, Integrity: manifest.Stylesheet.Integrity,
			Inline: AssetRef{Path: "styles.css", MediaType: "text/css", Content: goshtosoStyles},
		},
		{
			ID: "margo.document.styles", Kind: HTMLStylesheet,
			LocalURL: HTMLStylesURL, LoadAfter: []string{"goshtoso.styles"}, Inline: documentStyles,
		},
	}
	if includeTableSort {
		tableSort, err := EmbeddedAsset("table-sort.js")
		if err != nil {
			return nil, err
		}
		requirements = append(requirements, HTMLRequirement{
			ID: "margo.table-sort", Kind: HTMLScript,
			LocalURL: TableSortRuntimeURL, LoadAfter: []string{"margo.document.styles"}, Inline: tableSort,
		})
	}
	if includeCodeCopy {
		codeCopyPath := strings.TrimPrefix(goshtosoassets.CodeBlockURL, "/assets/")
		codeCopy, err := goshtosoassets.ReadFile(codeCopyPath)
		if err != nil {
			return nil, fmt.Errorf("margo: load Goshtoso code-block runtime: %w", err)
		}
		requirements = append(requirements, HTMLRequirement{
			ID: "goshtoso.runtime.code-block", Kind: HTMLScript,
			LocalURL: goshtosoassets.CodeBlockURL, LoadAfter: []string{"margo.document.styles"},
			Inline: AssetRef{Path: codeCopyPath, MediaType: "application/javascript", Content: codeCopy},
		})
	}
	return requirements, nil
}
