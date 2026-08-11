// Package site builds deterministic multi-page HTML sites from Markdown inputs.
package site

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strings"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/internal/staticimage"
	"golang.org/x/net/html"
)

// AssetMode controls whether source and runtime assets are emitted separately
// or embedded in each HTML document.
type AssetMode string

const (
	AssetsLocal  AssetMode = "local"
	AssetsInline AssetMode = "inline"
)

// Source is one site-relative Markdown input.
type Source struct {
	Path    string
	Content []byte
}

// Request is one immutable site build request.
type Request struct {
	SourceRoot  string
	Sources     []Source
	Compiler    *margo.Compiler
	Assets      AssetMode
	AssetReader margo.CheckAssetReader
}

// Artifact is one site-relative output and its exact bytes.
type Artifact struct {
	Path    string
	Content []byte
}

// Page records one deterministic source-to-output mapping.
type Page struct {
	Source string `json:"source"`
	Output string `json:"output"`
}

// Result contains sorted artifacts and their exact-byte manifest.
type Result struct {
	Artifacts []Artifact
	Manifest  margo.Manifest
	Pages     []Page
}

type builder struct {
	request      Request
	sources      map[string]Source
	outputs      map[string]string
	artifacts    map[string][]byte
	artifactKeys map[string]string
	assets       map[string]cachedAsset
	pages        []Page
	references   []siteReference
	assetBytes   int64
}

type cachedAsset struct {
	content   []byte
	mediaType string
}

type siteReference struct {
	source   string
	target   string
	fragment string
}

// Build renders, links, and packages a deterministic site without writing it.
func Build(ctx context.Context, request Request) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if request.Compiler == nil {
		request.Compiler = margo.New()
	}
	if request.Assets == "" {
		request.Assets = AssetsLocal
	}
	if request.Assets != AssetsLocal && request.Assets != AssetsInline {
		return Result{}, diagnostic("site.assets_invalid", "asset mode must be local or inline", "Choose --assets local or --assets inline.", "")
	}
	if request.AssetReader == nil {
		request.AssetReader = margo.FilesystemCheckAssetReader{}
	}

	b := &builder{
		request: request, sources: make(map[string]Source, len(request.Sources)),
		outputs: make(map[string]string, len(request.Sources)), artifacts: make(map[string][]byte),
		artifactKeys: make(map[string]string), assets: make(map[string]cachedAsset),
	}
	ordered, err := b.indexSources()
	if err != nil {
		return Result{}, err
	}
	for _, source := range ordered {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		if err := b.renderSource(ctx, source); err != nil {
			return Result{}, err
		}
	}
	if err := b.validateReferences(); err != nil {
		return Result{}, err
	}
	return b.result(), nil
}

func (b *builder) indexSources() ([]Source, error) {
	ordered := make([]Source, len(b.request.Sources))
	copy(ordered, b.request.Sources)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	for _, source := range ordered {
		normalized, ok := validSourcePath(source.Path)
		if !ok {
			return nil, diagnostic("site.source_invalid", fmt.Sprintf("invalid Markdown source path %q", source.Path), "Use a normalized relative .md or .markdown path.", source.Path)
		}
		source.Path = normalized
		key := strings.ToLower(normalized)
		if _, exists := b.sources[key]; exists {
			return nil, diagnostic("site.output_collision", fmt.Sprintf("multiple sources map to %q", outputPath(normalized)), "Rename one source so output paths are unique even on case-insensitive filesystems.", normalized)
		}
		output := outputPath(normalized)
		outputKey := strings.ToLower(output)
		if previous, exists := b.outputs[outputKey]; exists {
			return nil, diagnostic("site.output_collision", fmt.Sprintf("%q and %q map to the same output", previous, normalized), "Rename one source so each page has a unique output path.", normalized)
		}
		b.sources[key] = source
		b.outputs[outputKey] = normalized
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	return ordered, nil
}

func (b *builder) renderSource(ctx context.Context, source Source) error {
	base := filepath.Join(b.request.SourceRoot, filepath.FromSlash(path.Dir(source.Path)))
	document, err := b.request.Compiler.Compile(ctx, margo.Source{Name: source.Path, Content: source.Content, BaseURL: base})
	if err != nil {
		return err
	}
	rendered, err := b.request.Compiler.Render(ctx, document, margo.WithTableSort(margo.TableSortClient))
	if err != nil {
		return err
	}

	var componentBytes bytes.Buffer
	if b.request.Assets == AssetsInline {
		component, renderErr := margo.RenderStandalone(rendered)
		if renderErr != nil {
			return renderErr
		}
		if renderErr = component.Render(ctx, &componentBytes); renderErr != nil {
			return renderErr
		}
	} else {
		htmlResult, renderErr := margo.RenderHTML(rendered)
		if renderErr != nil {
			return renderErr
		}
		component, renderErr := margo.RenderHTMLPage(htmlResult, margo.HTMLPageInput{Theme: margo.ThemeModern, ColorMode: margo.ColorModeLight, DependencyMode: margo.HTMLDependenciesLocal})
		if renderErr != nil {
			return renderErr
		}
		if renderErr = component.Render(ctx, &componentBytes); renderErr != nil {
			return renderErr
		}
		for _, requirement := range htmlResult.Requirements().List() {
			assetPath := strings.TrimPrefix(requirement.LocalURL, "/")
			if assetPath == "" || len(requirement.Inline.Content) == 0 {
				continue
			}
			if err := b.addArtifact(assetPath, requirement.Inline.Content); err != nil {
				return err
			}
		}
	}

	rewritten, err := b.rewriteHTML(ctx, source, componentBytes.Bytes())
	if err != nil {
		return err
	}
	output := outputPath(source.Path)
	if err := b.addArtifact(output, rewritten); err != nil {
		return err
	}
	b.pages = append(b.pages, Page{Source: source.Path, Output: output})
	return nil
}

func (b *builder) rewriteHTML(ctx context.Context, source Source, document []byte) ([]byte, error) {
	root, err := html.Parse(bytes.NewReader(document))
	if err != nil {
		return nil, diagnostic("site.html_invalid", err.Error(), "Report this renderer defect with the source document.", source.Path)
	}
	var visit func(*html.Node) error
	visit = func(node *html.Node) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if node.Type == html.ElementNode {
			switch node.Data {
			case "a":
				if err := b.rewriteLink(source, node); err != nil {
					return err
				}
			case "img":
				if err := b.rewriteImage(ctx, source, node); err != nil {
					return err
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := html.Render(&output, root); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func (b *builder) rewriteLink(source Source, node *html.Node) error {
	index := attributeIndex(node, "href")
	if index < 0 {
		return nil
	}
	value := node.Attr[index].Val
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Path == "" || strings.HasPrefix(parsed.Path, "/") || !isMarkdownPath(parsed.Path) {
		return nil
	}
	target := path.Clean(path.Join(path.Dir(source.Path), parsed.Path))
	if target == ".." || strings.HasPrefix(target, "../") {
		return diagnostic("site.link_outside_root", fmt.Sprintf("link %q escapes the site root", value), "Link to a Markdown page within the input directory.", source.Path)
	}
	targetSource, exists := b.sources[strings.ToLower(target)]
	if !exists {
		return diagnostic("site.link_missing", fmt.Sprintf("Markdown link target %q does not exist", target), "Add the target document or correct the relative link.", source.Path)
	}
	relative, err := relativeSitePath(path.Dir(outputPath(source.Path)), outputPath(targetSource.Path))
	if err != nil {
		return err
	}
	parsed.Path = relative
	parsed.RawPath = ""
	node.Attr[index].Val = parsed.String()
	b.references = append(b.references, siteReference{source: source.Path, target: outputPath(targetSource.Path), fragment: parsed.Fragment})
	return nil
}

func (b *builder) rewriteImage(ctx context.Context, source Source, node *html.Node) error {
	index := attributeIndex(node, "src")
	if index < 0 || node.Attr[index].Val == "" {
		return nil
	}
	value := node.Attr[index].Val
	if strings.HasPrefix(strings.ToLower(value), "data:") {
		_, _, err := staticimage.ValidateDataURL(ctx, value, margo.MaxDocumentBytes-b.assetBytes)
		if err != nil {
			return diagnostic("site.asset_invalid", err.Error(), "Use a supported static image with matching media type.", source.Path)
		}
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(parsed.Path, "/") || parsed.Path == "" {
		return diagnostic("site.asset_external", fmt.Sprintf("image %q is not a local site asset", value), "Download the image into the site source tree.", source.Path)
	}
	assetPath := path.Clean(path.Join(path.Dir(source.Path), parsed.Path))
	if assetPath == ".." || strings.HasPrefix(assetPath, "../") {
		return diagnostic("site.asset_outside_root", fmt.Sprintf("image %q escapes the site root", value), "Use an image below the input directory.", source.Path)
	}
	asset, cached := b.assets[assetPath]
	if !cached {
		remaining := margo.MaxDocumentBytes - b.assetBytes
		data, readErr := b.request.AssetReader.ReadAsset(ctx, b.request.SourceRoot, assetPath, remaining)
		if readErr != nil {
			return diagnostic("site.asset_unreadable", fmt.Sprintf("cannot read image %q: %v", assetPath, readErr), "Add a readable regular image below the input directory.", source.Path)
		}
		mediaType, detectErr := staticimage.DetectContext(ctx, data)
		if detectErr != nil {
			return diagnostic("site.asset_invalid", fmt.Sprintf("invalid image %q: %v", assetPath, detectErr), "Use PNG, JPEG, GIF, WebP, or safe SVG.", source.Path)
		}
		asset = cachedAsset{content: append([]byte(nil), data...), mediaType: mediaType}
		b.assets[assetPath] = asset
		b.assetBytes += int64(len(data))
	}
	if b.request.Assets == AssetsInline {
		node.Attr[index].Val = "data:" + asset.mediaType + ";base64," + base64.StdEncoding.EncodeToString(asset.content)
		return nil
	}
	if err := b.addArtifact(assetPath, asset.content); err != nil {
		return err
	}
	return nil
}

func (b *builder) addArtifact(name string, content []byte) error {
	cleaned := path.Clean(strings.TrimPrefix(name, "/"))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return diagnostic("site.artifact_invalid", fmt.Sprintf("invalid artifact path %q", name), "Use a normalized site-relative artifact path.", name)
	}
	key := strings.ToLower(cleaned)
	canonical, caseExists := b.artifactKeys[key]
	if caseExists && canonical != cleaned {
		return diagnostic("site.artifact_collision", fmt.Sprintf("artifacts %q and %q collide on case-insensitive filesystems", canonical, cleaned), "Rename one source asset or page.", cleaned)
	}
	if previous, exists := b.artifacts[cleaned]; exists {
		if bytes.Equal(previous, content) {
			return nil
		}
		return diagnostic("site.artifact_collision", fmt.Sprintf("different content maps to artifact %q", cleaned), "Rename the source asset or page.", cleaned)
	}
	b.artifacts[cleaned] = append([]byte(nil), content...)
	b.artifactKeys[key] = cleaned
	return nil
}

func (b *builder) validateReferences() error {
	anchors := make(map[string]map[string]struct{})
	for _, reference := range b.references {
		if reference.fragment == "" {
			continue
		}
		ids, exists := anchors[reference.target]
		if !exists {
			document, ok := b.artifacts[reference.target]
			if !ok {
				return diagnostic("site.link_missing", fmt.Sprintf("generated link target %q does not exist", reference.target), "Correct the Markdown link target.", reference.source)
			}
			root, err := html.Parse(bytes.NewReader(document))
			if err != nil {
				return diagnostic("site.html_invalid", err.Error(), "Report this renderer defect with the source document.", reference.source)
			}
			ids = make(map[string]struct{})
			var walk func(*html.Node)
			walk = func(node *html.Node) {
				if node.Type == html.ElementNode {
					if index := attributeIndex(node, "id"); index >= 0 && node.Attr[index].Val != "" {
						ids[node.Attr[index].Val] = struct{}{}
					}
				}
				for child := node.FirstChild; child != nil; child = child.NextSibling {
					walk(child)
				}
			}
			walk(root)
			anchors[reference.target] = ids
		}
		if _, exists := ids[reference.fragment]; !exists {
			return diagnostic("site.anchor_missing", fmt.Sprintf("fragment %q does not exist in %q", reference.fragment, reference.target), "Correct the fragment or add the target heading.", reference.source)
		}
	}
	return nil
}

func (b *builder) result() Result {
	paths := make([]string, 0, len(b.artifacts))
	for name := range b.artifacts {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	result := Result{
		Artifacts: make([]Artifact, 0, len(paths)), Manifest: margo.Manifest{Entries: make([]margo.ManifestEntry, 0, len(paths))},
		Pages: append([]Page(nil), b.pages...),
	}
	for _, name := range paths {
		content := append([]byte(nil), b.artifacts[name]...)
		result.Artifacts = append(result.Artifacts, Artifact{Path: name, Content: content})
		result.Manifest.Entries = append(result.Manifest.Entries, margo.ManifestEntry{Path: name, Digest: margo.ArtifactDigestOf(content)})
	}
	return result
}

func validSourcePath(name string) (string, bool) {
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
		return "", false
	}
	cleaned := path.Clean(name)
	if cleaned != name || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || !isMarkdownPath(cleaned) {
		return "", false
	}
	return cleaned, true
}

func isMarkdownPath(name string) bool {
	extension := strings.ToLower(path.Ext(name))
	return extension == ".md" || extension == ".markdown"
}

func outputPath(name string) string {
	return strings.TrimSuffix(name, path.Ext(name)) + ".html"
}

func attributeIndex(node *html.Node, key string) int {
	for index := range node.Attr {
		if node.Attr[index].Key == key {
			return index
		}
	}
	return -1
}

func relativeSitePath(fromDirectory, target string) (string, error) {
	relative, err := filepath.Rel(filepath.FromSlash(fromDirectory), filepath.FromSlash(target))
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(relative), nil
}

func diagnostic(code, message, hint, source string) error {
	return &margo.DiagnosticError{Diagnostics: []margo.Diagnostic{{
		Code: code, Severity: margo.SeverityError, Source: source,
		Message: message, Hint: hint,
	}}}
}
