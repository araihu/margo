// Package site builds the checked blog-style HTML example.
package site

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-h/templ"
	"github.com/araihu/margo"
)

//go:embed assets/* content/*.md
var exampleFiles embed.FS

var imageAssets = []string{
	"atelier-hero.avif",
	"atelier-hero.jpg",
	"atelier-hero.webp",
	"format-study.gif",
	"format-study.png",
}

// Generate writes the complete blog example to outputDirectory.
func Generate(outputDirectory string) error {
	if strings.TrimSpace(outputDirectory) == "" {
		return fmt.Errorf("blog output directory is required")
	}
	assetDirectory := filepath.Join(outputDirectory, "assets")
	if err := os.MkdirAll(assetDirectory, 0o755); err != nil {
		return fmt.Errorf("create blog asset directory: %w", err)
	}
	for _, name := range imageAssets {
		data, err := fs.ReadFile(exampleFiles, "assets/"+name)
		if err != nil {
			return fmt.Errorf("read embedded blog asset %q: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(assetDirectory, name), data, 0o644); err != nil {
			return fmt.Errorf("write blog asset %q: %w", name, err)
		}
	}

	stylesheet, err := embeddedStylesheet()
	if err != nil {
		return err
	}
	header := templ.Raw(`<nav class="blog-nav" aria-label="Primary"><a class="blog-nav__brand" href="index.html">Margo Field Notes</a><span class="blog-nav__links"><a href="index.html">Home</a><a href="field-notes.html">Latest story</a></span></nav>`)
	footer := templ.Raw(`<div class="blog-footer"><span>Built from semantic Markdown.</span><span>AVIF · WebP · JPEG · PNG · GIF</span></div>`)
	page := margo.HTMLPageInput{
		Theme: margo.ThemeName("margo-blog"), ColorMode: margo.ColorModeLight,
		DependencyMode: margo.HTMLDependenciesInline, ThemeStylesheet: stylesheet,
		Header: header, Footer: footer,
	}

	indexResult, err := renderMarkdown("content/index.md")
	if err != nil {
		return err
	}
	indexPage, err := margo.RenderHTMLPage(indexResult, page)
	if err != nil {
		return fmt.Errorf("compose blog index: %w", err)
	}
	if err := writeComponent(filepath.Join(outputDirectory, "index.html"), indexPage); err != nil {
		return err
	}

	articleResult, err := renderMarkdown("content/field-notes.md")
	if err != nil {
		return err
	}
	articlePage := page
	articlePage.BeforeContent = blogHero()
	article, err := renderBlogPublication(articleResult, blogPublicationInput{
		CanonicalURL:  "https://margo.invalid/guide",
		SiteName:      "Margo Field Notes",
		Locale:        "en_US",
		ImageURL:      "https://margo.invalid/assets/social/margo-v0.0.1.png",
		ImageMIMEType: "image/png",
		ImageWidth:    1280,
		ImageHeight:   640,
		ImageAlt:      "Margo Field Notes preview.",
		Page:          articlePage,
	})
	if err != nil {
		return fmt.Errorf("compose blog article: %w", err)
	}
	if err := writeComponent(filepath.Join(outputDirectory, "field-notes.html"), article); err != nil {
		return err
	}
	return nil
}

func embeddedStylesheet() (margo.AssetRef, error) {
	content, err := fs.ReadFile(exampleFiles, "assets/blog.css")
	if err != nil {
		return margo.AssetRef{}, fmt.Errorf("read blog stylesheet: %w", err)
	}
	digest := sha256.Sum256(content)
	return margo.AssetRef{
		Path: "assets/blog.css", MediaType: "text/css",
		SHA256: hex.EncodeToString(digest[:]), Content: content,
	}, nil
}

func renderMarkdown(name string) (*margo.HTMLResult, error) {
	content, err := fs.ReadFile(exampleFiles, name)
	if err != nil {
		return nil, fmt.Errorf("read blog source %q: %w", name, err)
	}
	compiler := margo.New()
	document, err := compiler.Compile(context.Background(), margo.Source{Name: filepath.Base(name), Content: content})
	if err != nil {
		return nil, fmt.Errorf("compile blog source %q: %w", name, err)
	}
	rendered, err := compiler.Render(context.Background(), document)
	if err != nil {
		return nil, fmt.Errorf("render blog source %q: %w", name, err)
	}
	result, err := margo.RenderHTML(rendered)
	if err != nil {
		return nil, fmt.Errorf("project blog source %q: %w", name, err)
	}
	return result, nil
}

func blogHero() templ.Component {
	return templ.Raw(`<figure class="blog-hero"><picture><source type="image/avif" srcset="assets/atelier-hero.avif"><source type="image/webp" srcset="assets/atelier-hero.webp"><img src="assets/atelier-hero.jpg" width="1738" height="906" alt="A pink gopher editor reviews an article in a warm publishing atelier."></picture><figcaption>One generated hero, delivered as AVIF, WebP, and JPEG fallback.</figcaption></figure>`)
}

func writeComponent(name string, component templ.Component) error {
	var output bytes.Buffer
	if err := component.Render(context.Background(), &output); err != nil {
		return fmt.Errorf("render %q: %w", name, err)
	}
	if err := os.WriteFile(name, output.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %q: %w", name, err)
	}
	return nil
}
