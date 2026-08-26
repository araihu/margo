// Package site builds deterministic multi-page HTML sites from Markdown inputs.
package site

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/internal/staticimage"
	"github.com/araihu/margo/pdf"
	"github.com/araihu/margo/ssg"
	"golang.org/x/net/html"
)

// AssetMode controls whether source and runtime assets are emitted separately
// or embedded in each HTML document.
type AssetMode string

const (
	AssetsLocal  AssetMode = "local"
	AssetsInline AssetMode = "inline"
	// ManifestPath is reserved for the CLI-owned exact-byte manifest.
	ManifestPath = "margo-manifest.json"
	// SitemapPath is the generated XML index of configured public routes.
	SitemapPath = "sitemap.xml"
	// LLMSPath is the generated Markdown index for language-model consumers.
	LLMSPath = "llms.txt"
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
	PDFEngine   pdf.Engine
}

// Artifact is one site-relative output and its exact bytes.
type Artifact struct {
	Path    string
	Content []byte
}

// Page records one deterministic source-to-output mapping.
type Page struct {
	Source         string             `json:"source"`
	Output         string             `json:"output"`
	Locale         string             `json:"locale,omitempty"`
	Family         string             `json:"family,omitempty"`
	Layout         string             `json:"layout,omitempty"`
	Title          string             `json:"title,omitempty"`
	Description    string             `json:"description,omitempty"`
	Authors        []string           `json:"authors,omitempty"`
	PublishedAt    string             `json:"publishedAt,omitempty"`
	ModifiedAt     string             `json:"modifiedAt,omitempty"`
	Tags           []string           `json:"tags,omitempty"`
	Canonical      string             `json:"canonical,omitempty"`
	DocumentDigest string             `json:"documentDigest,omitempty"`
	ImageOverflow  string             `json:"imageOverflow,omitempty"`
	Actions        *margo.PageActions `json:"actions,omitempty"`
	Alternates     []Alternate        `json:"alternates,omitempty"`
}

type Alternate struct {
	Locale string `json:"locale"`
	URL    string `json:"url"`
}

// SiteManifest carries config and route identity in addition to artifact
// digests. Directory builds populate Routes with the same stable page records
// used by the site report; configured builds add their canonical identities.
type SiteManifest struct {
	ConfigVersion       int    `json:"configVersion,omitempty"`
	Layout              string `json:"layout,omitempty"`
	LayoutSchemaHash    string `json:"layoutSchemaHash,omitempty"`
	BaseURL             string `json:"baseURL,omitempty"`
	BasePath            string `json:"basePath,omitempty"`
	DocumentStyleDigest string `json:"documentStyleDigest,omitempty"`
	Routes              []Page `json:"routes,omitempty"`
}

// Result contains sorted artifacts and their exact-byte manifest.
type Result struct {
	Artifacts []Artifact
	Manifest  margo.Manifest
	Pages     []Page
	Site      SiteManifest
}

type builder struct {
	request          Request
	config           *Config
	configSource     string
	configDir        string
	sourceDir        string
	siteManifest     SiteManifest
	frame            ssg.Frame
	frameSchema      ssg.FrameSchema
	frameHash        string
	frameValues      ssg.Values
	layoutName       string
	shellMode        bool
	shellName        string
	shellAssetPrefix string
	socialMediaType  string
	layoutPatches    []LayoutPatch
	configured       map[string]configuredPage
	configPages      []Page
	docsFamilies     []docsFamily
	sources          map[string]Source
	outputs          map[string]string
	artifacts        map[string][]byte
	artifactKeys     map[string]string
	assets           map[string]cachedAsset
	configuredAssets map[string]cachedAsset
	dependencies     map[string]string
	pages            []Page
	references       []siteReference
	assetBytes       int64
	pdfEngine        pdf.Engine
	pdfInstances     *margo.InstanceAllocator
}

// docsFamily is derived from completed docs pages. Config owns only ordered
// identifiers; the overview page owns the public label and href.
type docsFamily struct {
	ID       string
	Locale   string
	Overview Page
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
		artifactKeys: make(map[string]string), assets: make(map[string]cachedAsset), configuredAssets: make(map[string]cachedAsset), dependencies: make(map[string]string),
		pdfEngine: request.PDFEngine, pdfInstances: margo.NewInstanceAllocator(),
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
		output := b.pageOutput(normalized)
		outputKey := strings.ToLower(output)
		if previous, exists := b.outputs[outputKey]; exists {
			return nil, diagnostic("site.output_collision", fmt.Sprintf("%q and %q map to the same output", previous, normalized), "Rename one source so each page has a unique output path.", normalized)
		}
		if previous, exists := pathPrefixCollision(b.outputs, outputKey); exists {
			return nil, diagnostic("site.artifact_collision", fmt.Sprintf("outputs for %q and %q have a file/directory path collision", previous, normalized), "Rename one source so no page output is another output's parent directory.", normalized)
		}
		b.sources[key] = source
		b.outputs[outputKey] = normalized
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	return ordered, nil
}

func (b *builder) renderSource(ctx context.Context, source Source) (failure error) {
	defer func() { failure = attachSource(failure, source.Path) }()
	if b.config != nil {
		return b.renderConfiguredSource(ctx, source)
	}
	base := filepath.Join(b.request.SourceRoot, filepath.FromSlash(path.Dir(source.Path)))
	document, err := b.request.Compiler.Compile(ctx, margo.Source{Name: source.Path, Content: source.Content, BaseURL: base})
	if err != nil {
		return err
	}
	rendered, err := b.request.Compiler.Render(ctx, document, margo.WithTableSort(margo.TableSortClient), margo.WithRenderTarget(margo.TargetSite))
	if err != nil {
		return err
	}
	htmlResult, err := margo.RenderHTML(rendered)
	if err != nil {
		return err
	}
	publication, err := publicationMetadataFor(source.Path, document.Metadata(), htmlResult.Metadata())
	if err != nil {
		return err
	}
	output := b.pageOutput(source.Path)
	authors, publishedAt, modifiedAt, tags := pagePublicationMetadata(publication)
	page := Page{
		Source: source.Path, Output: output, Locale: htmlResult.Metadata().Language, Title: htmlResult.Metadata().Title,
		Description: htmlResult.Metadata().Description, Authors: authors, PublishedAt: publishedAt, ModifiedAt: modifiedAt,
		Tags: tags, ImageOverflow: pageImageOverflowForMetadata(document.Metadata()), Actions: pageActionsForMetadata(document.Metadata()),
	}
	if page.Locale == "" {
		page.Locale = "en"
	}

	dependencyMode := margo.HTMLDependenciesLocal
	if b.request.Assets == AssetsInline {
		dependencyMode = margo.HTMLDependenciesInline
	}
	component, err := margo.RenderHTMLPage(htmlResult, margo.HTMLPageInput{Theme: margo.ThemeModern, ColorMode: margo.ColorModeLight, DependencyMode: dependencyMode})
	if err != nil {
		return err
	}
	var componentBytes bytes.Buffer
	if err := component.Render(ctx, &componentBytes); err != nil {
		return err
	}
	if b.request.Assets == AssetsLocal {
		for _, requirement := range htmlResult.Requirements().List() {
			if requirement.ID == "goshtoso.styles" {
				continue
			}
			assetPath := strings.TrimPrefix(requirement.LocalURL, "/")
			if assetPath == "" || len(requirement.Inline.Content) == 0 {
				continue
			}
			if err := b.addArtifact(assetPath, requirement.Inline.Content); err != nil {
				return err
			}
			b.dependencies[strings.ToLower(assetPath)] = assetPath
		}
		if err := b.stageChartIconSprite(htmlResult.Requirements()); err != nil {
			return err
		}
	}

	projected, err := projectPublicationMetadata(componentBytes.Bytes(), page)
	if err != nil {
		return err
	}
	rewritten, err := b.rewriteHTML(ctx, source, projected)
	if err != nil {
		return err
	}
	if err := b.addDeclaredPageArtifacts(ctx, source, page, document); err != nil {
		return err
	}
	if err := b.addArtifact(output, rewritten); err != nil {
		return err
	}
	b.pages = append(b.pages, page)
	return nil
}

func attachSource(failure error, source string) error {
	if failure == nil {
		return nil
	}
	var diagnosticError *margo.DiagnosticError
	if errors.As(failure, &diagnosticError) && len(diagnosticError.Diagnostics) > 0 {
		diagnostics := append([]margo.Diagnostic(nil), diagnosticError.Diagnostics...)
		for index := range diagnostics {
			if diagnostics[index].Source == "" {
				diagnostics[index].Source = source
			}
			if diagnostics[index].Hint == "" {
				diagnostics[index].Hint = "Correct the source document and run margo check before rebuilding the site."
			}
		}
		return &margo.DiagnosticError{Diagnostics: diagnostics}
	}
	code, message, found := strings.Cut(failure.Error(), ":")
	if found && validDiagnosticCode(code) {
		return diagnostic(code, strings.TrimSpace(message), "Correct the source document and run margo check before rebuilding the site.", source)
	}
	return failure
}

func validDiagnosticCode(value string) bool {
	if value == "" || !strings.Contains(value, ".") {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '.' && character != '_' {
			return false
		}
	}
	return true
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
			if err := b.rewriteDependency(source, node); err != nil {
				return err
			}
			if b.config == nil && attributeValue(node, "data-margo-requirement") == "goshtoso.styles" {
				node.Parent.RemoveChild(node)
				return nil
			}
			switch node.Data {
			case "a":
				if err := b.rewriteLink(ctx, source, node); err != nil {
					return err
				}
			case "img":
				if err := b.rewriteImage(ctx, source, node); err != nil {
					return err
				}
			}
		}
		for child := node.FirstChild; child != nil; {
			next := child.NextSibling
			if err := visit(child); err != nil {
				return err
			}
			child = next
		}
		return nil
	}
	if err := visit(root); err != nil {
		return nil, err
	}
	if b.usesComponentDocShell(source) {
		decorateComponentDocShellHeadings(root)
		if err := b.decorateComponentDocShellNavigation(root, source); err != nil {
			return nil, err
		}
	}
	var output bytes.Buffer
	if err := html.Render(&output, root); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func (b *builder) rewriteDependency(source Source, node *html.Node) error {
	var attribute string
	switch node.Data {
	case "link":
		attribute = "href"
	case "script":
		attribute = "src"
	default:
		return nil
	}
	index := attributeIndex(node, attribute)
	if index < 0 {
		return nil
	}
	parsed, err := url.Parse(node.Attr[index].Val)
	if err != nil || !strings.HasPrefix(parsed.Path, "/") {
		return nil
	}
	dependencyKey := strings.ToLower(strings.TrimPrefix(parsed.Path, "/"))
	dependency, exists := b.dependencies[dependencyKey]
	if !exists && b.config != nil {
		basePath := strings.TrimPrefix(normalizedBasePath(b.config.BasePath), "/")
		if basePath != "" {
			dependency, exists = b.dependencies[strings.TrimPrefix(dependencyKey, basePath+"/")]
		}
	}
	if !exists {
		return nil
	}
	relative, err := relativeSitePath(path.Dir(b.pageOutput(source.Path)), dependency)
	if err != nil {
		return err
	}
	parsed.Path = relative
	parsed.RawPath = ""
	node.Attr[index].Val = parsed.String()
	return nil
}

func (b *builder) rewriteLink(ctx context.Context, source Source, node *html.Node) error {
	index := attributeIndex(node, "href")
	if index < 0 {
		return nil
	}
	value := node.Attr[index].Val
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Path == "" || strings.HasPrefix(parsed.Path, "/") {
		return nil
	}
	if !isMarkdownPath(parsed.Path) {
		target := path.Clean(path.Join(path.Dir(source.Path), parsed.Path))
		if _, exists := b.outputs[strings.ToLower(target)]; exists {
			// Rendered page links are already site-relative and may carry a
			// query or fragment. They are not source-root assets.
			return nil
		}
		return b.rewriteAssetLink(ctx, source, node, index, parsed, value)
	}
	target := path.Clean(path.Join(path.Dir(source.Path), parsed.Path))
	if target == ".." || strings.HasPrefix(target, "../") {
		return diagnostic("site.link_outside_root", fmt.Sprintf("link %q escapes the site root", value), "Link to a Markdown page within the input directory.", source.Path)
	}
	targetSource, exists := b.sources[strings.ToLower(target)]
	if !exists {
		return diagnostic("site.link_missing", fmt.Sprintf("Markdown link target %q does not exist", target), "Add the target document or correct the relative link.", source.Path)
	}
	// Configured sites publish directory routes for index artifacts. The
	// in-memory legacy builder has no public site configuration, so it retains
	// its artifact-relative link contract.
	if b.usesPublicRoutes() {
		parsed.Path = b.publicPagePath(targetSource.Path)
	} else {
		relative, err := relativeSitePath(path.Dir(b.pageOutput(source.Path)), b.pageOutput(targetSource.Path))
		if err != nil {
			return err
		}
		parsed.Path = relative
	}
	parsed.RawPath = ""
	node.Attr[index].Val = parsed.String()
	b.references = append(b.references, siteReference{source: source.Path, target: b.pageOutput(targetSource.Path), fragment: parsed.Fragment})
	return nil
}

func (b *builder) rewriteAssetLink(ctx context.Context, source Source, node *html.Node, index int, parsed *url.URL, value string) error {
	assetPath := path.Clean(path.Join(path.Dir(source.Path), parsed.Path))
	if assetPath == ".." || strings.HasPrefix(assetPath, "../") {
		return diagnostic("site.asset_outside_root", fmt.Sprintf("asset link %q escapes the site root", value), "Link to a local asset below the input directory.", source.Path)
	}
	asset, cached := b.assets[assetPath]
	if !cached {
		remaining := margo.MaxDocumentBytes - b.assetBytes
		data, readErr := b.request.AssetReader.ReadAsset(ctx, b.request.SourceRoot, assetPath, remaining)
		if readErr != nil {
			// A relative href is not necessarily a downloadable asset. Keep
			// missing and non-regular targets as ordinary links; only an
			// existing regular file is promoted to a published artifact.
			if errors.Is(readErr, os.ErrNotExist) {
				if configured, exists := b.configuredAssets[assetPath]; exists {
					asset, cached = configured, true
				} else {
					return nil
				}
			} else if errors.Is(readErr, margo.ErrCheckAssetNotRegular) {
				return nil
			} else {
				return linkedAssetReadDiagnostic(source, value, readErr)
			}
		}
		if !cached && int64(len(data)) > remaining {
			return diagnostic("site.asset_too_large", fmt.Sprintf("local asset %q exceeds the remaining site asset limit", value), "Reduce the asset size or remove other local assets.", source.Path)
		}
		if !cached {
			var cacheErr error
			asset, cacheErr = b.cacheSourceAsset(ctx, assetPath, data)
			if cacheErr != nil {
				return cacheErr
			}
		}
	}
	if err := b.addArtifact(assetPath, asset.content); err != nil {
		return err
	}
	relative, err := relativeSitePath(path.Dir(b.pageOutput(source.Path)), assetPath)
	if err != nil {
		return err
	}
	parsed.Path = relative
	parsed.RawPath = ""
	node.Attr[index].Val = parsed.String()
	return nil
}

func linkedAssetReadDiagnostic(source Source, value string, readErr error) error {
	if errors.Is(readErr, context.Canceled) || errors.Is(readErr, context.DeadlineExceeded) {
		return readErr
	}
	if errors.Is(readErr, margo.ErrCheckAssetOutsideRoot) {
		return diagnostic("site.asset_outside_root", fmt.Sprintf("asset link %q resolves outside the site root", value), "Link to a local asset below the input directory.", source.Path)
	}
	if errors.Is(readErr, margo.ErrCheckAssetTooLarge) {
		return diagnostic("site.asset_too_large", fmt.Sprintf("local asset %q exceeds the remaining site asset limit", value), "Reduce the asset size or remove other local assets.", source.Path)
	}
	return diagnostic("site.asset_unreadable", fmt.Sprintf("cannot read local asset %q: %v", value, readErr), "Add a readable regular asset below the input directory.", source.Path)
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
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Path == "" {
		return diagnostic("site.asset_external", fmt.Sprintf("image %q is not a local site asset", value), "Download the image into the site source tree.", source.Path)
	}
	if strings.HasPrefix(parsed.Path, "/") {
		dependency, exists := b.dependencies[strings.ToLower(strings.TrimPrefix(parsed.Path, "/"))]
		if !exists {
			return diagnostic("site.asset_external", fmt.Sprintf("image %q is not a local site asset", value), "Download the image into the site source tree.", source.Path)
		}
		relative, relativeErr := relativeSitePath(path.Dir(b.pageOutput(source.Path)), dependency)
		if relativeErr != nil {
			return relativeErr
		}
		parsed.Path = relative
		parsed.RawPath = ""
		node.Attr[index].Val = parsed.String()
		return nil
	}
	assetPath := path.Clean(path.Join(path.Dir(source.Path), parsed.Path))
	if assetPath == ".." || strings.HasPrefix(assetPath, "../") {
		return diagnostic("site.asset_outside_root", fmt.Sprintf("image %q escapes the site root", value), "Use an image below the input directory.", source.Path)
	}
	asset, cached := b.assets[assetPath]
	sourceCached := cached
	if !cached {
		remaining := margo.MaxDocumentBytes - b.assetBytes
		data, readErr := b.request.AssetReader.ReadAsset(ctx, b.request.SourceRoot, assetPath, remaining)
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) {
				if configured, exists := b.configuredAssets[assetPath]; exists {
					asset, cached = configured, true
				} else {
					return diagnostic("site.asset_unreadable", fmt.Sprintf("cannot read image %q: %v", assetPath, readErr), "Add a readable regular image below the input directory.", source.Path)
				}
			} else {
				return diagnostic("site.asset_unreadable", fmt.Sprintf("cannot read image %q: %v", assetPath, readErr), "Add a readable regular image below the input directory.", source.Path)
			}
		}
		if !cached {
			mediaType, detectErr := staticimage.DetectContext(ctx, data)
			if detectErr != nil {
				return diagnostic("site.asset_invalid", fmt.Sprintf("invalid image %q: %v", assetPath, detectErr), "Use PNG, JPEG, GIF, WebP, or safe SVG.", source.Path)
			}
			asset = cachedAsset{content: append([]byte(nil), data...), mediaType: mediaType}
			b.assets[assetPath] = asset
			b.assetBytes += int64(len(data))
		}
	}
	if asset.mediaType == "" {
		mediaType, detectErr := staticimage.DetectContext(ctx, asset.content)
		if detectErr != nil {
			return diagnostic("site.asset_invalid", fmt.Sprintf("invalid image %q: %v", assetPath, detectErr), "Use PNG, JPEG, GIF, WebP, or safe SVG.", source.Path)
		}
		asset.mediaType = mediaType
		if sourceCached {
			b.assets[assetPath] = asset
		}
	}
	setImageIntrinsicDimensions(node, asset.content)
	if b.request.Assets == AssetsInline {
		node.Attr[index].Val = "data:" + asset.mediaType + ";base64," + base64.StdEncoding.EncodeToString(asset.content)
		return nil
	}
	if err := b.addArtifact(assetPath, asset.content); err != nil {
		return err
	}
	return nil
}

// cacheSourceAsset keeps source-root assets separate from configured assets.
// A configured asset may share a published path with a source asset, but it
// must never satisfy a source-relative reference before the source is read.
// Media type detection is intentionally best-effort here: ordinary links may
// target documents such as PDFs, while image references validate the cached
// bytes in rewriteImage before they are embedded.
func (b *builder) cacheSourceAsset(ctx context.Context, assetPath string, data []byte) (cachedAsset, error) {
	asset := cachedAsset{content: append([]byte(nil), data...)}
	mediaType, detectErr := staticimage.DetectContext(ctx, data)
	if detectErr != nil {
		if errors.Is(detectErr, context.Canceled) || errors.Is(detectErr, context.DeadlineExceeded) {
			return cachedAsset{}, detectErr
		}
	} else {
		asset.mediaType = mediaType
	}
	b.assets[assetPath] = asset
	b.assetBytes += int64(len(data))
	return asset, nil
}

func setImageIntrinsicDimensions(node *html.Node, content []byte) {
	if attributeIndex(node, "width") >= 0 || attributeIndex(node, "height") >= 0 {
		return
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return
	}
	node.Attr = append(node.Attr,
		html.Attribute{Key: "width", Val: fmt.Sprint(config.Width)},
		html.Attribute{Key: "height", Val: fmt.Sprint(config.Height)},
	)
}

func (b *builder) addArtifact(name string, content []byte) error {
	cleaned := path.Clean(strings.TrimPrefix(name, "/"))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return diagnostic("site.artifact_invalid", fmt.Sprintf("invalid artifact path %q", name), "Use a normalized site-relative artifact path.", name)
	}
	reservedKey := strings.ToLower(ManifestPath)
	cleanedKey := strings.ToLower(cleaned)
	if cleanedKey == reservedKey || strings.HasPrefix(cleanedKey, reservedKey+"/") {
		return diagnostic("site.artifact_reserved", fmt.Sprintf("artifact path %q is reserved for the site manifest", cleaned), "Rename the source asset.", cleaned)
	}
	key := cleanedKey
	canonical, caseExists := b.artifactKeys[key]
	if caseExists && canonical != cleaned {
		return diagnostic("site.artifact_collision", fmt.Sprintf("artifacts %q and %q collide on case-insensitive filesystems", canonical, cleaned), "Rename one source asset or page.", cleaned)
	}
	if previous, exists := pathPrefixCollision(b.artifactKeys, key); exists {
		return diagnostic("site.artifact_collision", fmt.Sprintf("artifacts %q and %q have a file/directory path collision", previous, cleaned), "Rename one source asset or page.", cleaned)
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

func pathPrefixCollision(paths map[string]string, candidate string) (string, bool) {
	for existing, original := range paths {
		if strings.HasPrefix(candidate, existing+"/") || strings.HasPrefix(existing, candidate+"/") {
			return original, true
		}
	}
	return "", false
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
	pages := b.pages
	if b.config != nil {
		pages = b.configPages
	}
	siteManifest := b.siteManifest
	if b.config == nil {
		siteManifest.Routes = append([]Page(nil), pages...)
	}
	result := Result{
		Artifacts: make([]Artifact, 0, len(paths)), Manifest: margo.Manifest{Entries: make([]margo.ManifestEntry, 0, len(paths))},
		Pages: append([]Page(nil), pages...), Site: siteManifest,
	}
	for _, name := range paths {
		content := append([]byte(nil), b.artifacts[name]...)
		result.Artifacts = append(result.Artifacts, Artifact{Path: name, Content: content})
		result.Manifest.Entries = append(result.Manifest.Entries, margo.ManifestEntry{Path: name, Digest: margo.ArtifactDigestOf(content)})
	}
	return result
}

func (b *builder) pageOutput(name string) string {
	if b.config == nil {
		return outputPath(name)
	}
	return configuredOutputPath(name, b.config.Locales)
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

// publicRoutePath projects an artifact path into its public URL identity.
// Directory index artifacts remain index.html on disk but are advertised at
// the directory route, so one page has one canonical public identity.
func publicRoutePath(output string) string {
	output = strings.TrimPrefix(path.Clean(strings.TrimSpace(output)), "/")
	if output == "." || output == "" {
		return "/"
	}
	if path.Base(output) == "index.html" {
		directory := path.Dir(output)
		if directory == "." || directory == "" {
			return "/"
		}
		return "/" + strings.Trim(directory, "/") + "/"
	}
	return "/" + output
}

func (b *builder) publicOutputPath(output string, home bool) string {
	route := publicRoutePath(output)
	if home {
		route = "/"
	}
	basePath := normalizedBasePath("")
	if b.config != nil {
		basePath = normalizedBasePath(b.config.BasePath)
	}
	if basePath != "/" {
		route = strings.TrimSuffix(basePath, "/") + route
	}
	return route
}

func (b *builder) usesPublicRoutes() bool {
	return b.config != nil && b.config.Layout != nil
}

func (b *builder) publicPagePath(source string) string {
	output := b.pageOutput(source)
	locale, _ := sourceLocale(source, b.config.Locales)
	home := source == b.config.Site.Home && locale == b.config.Locales.Default
	return b.publicOutputPath(output, home)
}

func diagnostic(code, message, hint, source string) error {
	return &margo.DiagnosticError{Diagnostics: []margo.Diagnostic{{
		Code: code, Severity: margo.SeverityError, Source: source,
		Message: message, Hint: hint,
	}}}
}
