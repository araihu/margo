package site

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	stdhtml "html"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/a-h/templ"
	chartassets "github.com/araihu/goshtoso-charts/assets"
	"github.com/araihu/goshtoso/components/breadcrumbs"
	margo "github.com/araihu/margo"
	"github.com/araihu/margo/internal/staticimage"
	"github.com/araihu/margo/ssg"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
	"gopkg.in/yaml.v3"
)

type configuredPage struct {
	page          Page
	layout        ResolvedLayout
	layoutSources []string
	presentation  PagePresentation
	article       []byte
	plainText     string
	result        *margo.HTMLResult
	document      *margo.Document
}

const configuredSiteCSS = `:root {
  color-scheme: light dark;
  --margo-surface: #ffffff;
  --margo-surface-alt: #f5f7fa;
  --margo-text: #17202a;
  --margo-text-strong: #0b1220;
  --margo-accent: #155eef;
  --margo-outline: #7b8794;
  --margo-gap: 1.5rem;
  --margo-reading-measure: 75ch;
}
[data-color-mode="dark"], html.dark {
  --margo-surface: #111827;
  --margo-surface-alt: #1f2937;
  --margo-text: #e5e7eb;
  --margo-text-strong: #ffffff;
  --margo-accent: #8ab4ff;
  --margo-outline: #9ca3af;
}
@media (prefers-color-scheme: dark) {
  [data-color-mode="system"], html.dark {
    --margo-surface: #111827;
    --margo-surface-alt: #1f2937;
    --margo-text: #e5e7eb;
    --margo-text-strong: #ffffff;
    --margo-accent: #8ab4ff;
    --margo-outline: #9ca3af;
  }
}
* { box-sizing: border-box; }
html { background: var(--margo-surface); color: var(--margo-text); }
body { margin: 0; min-inline-size: 0; font-family: system-ui, sans-serif; line-height: 1.6; }
:where(.margo-frame a, .margo-document a) { color: var(--margo-accent); }
.margo-skip-link { position: absolute; inset-inline-start: 0.75rem; inset-block-start: 0.75rem; transform: translateY(-200%); padding: 0.75rem 1rem; background: var(--margo-surface); color: var(--margo-text-strong); outline: 3px solid var(--margo-accent); z-index: 10; }
.margo-skip-link:focus { transform: none; }
.margo-frame { display: grid; gap: var(--margo-gap); max-inline-size: 100%; padding: 1rem; }
.margo-frame > .margo-area { min-inline-size: 0; }
.margo-area--main-content { max-inline-size: var(--margo-reading-measure); inline-size: 100%; margin-inline: auto; }
.margo-area--top-nav { grid-area: top-nav; }
.margo-area--left-nav { grid-area: left-nav; }
.margo-area--main-content { grid-area: main-content; }
.margo-area--right-nav { grid-area: right-nav; }
.margo-area--footer { grid-area: footer; }
.margo-area--left-nav, .margo-area--right-nav { overflow-inline: auto; }
.margo-area--left-nav ul, .margo-area--right-nav ul { padding-inline-start: 1.25rem; }
.margo-area--top-nav nav, .margo-area--left-nav nav, .margo-area--right-nav nav { min-block-size: 2.75rem; }
.margo-area--top-nav { display: flex; flex-wrap: wrap; align-items: center; gap: 1rem; }
.margo-area--top-nav > nav { flex: 1 1 auto; }
.margo-area--top-nav > [data-navbar-shell="true"] { flex: 1 1 100%; inline-size: 100%; min-inline-size: 0; }
[data-margo-sticky="true"] { position: sticky; z-index: 2; }
[data-margo-sticky="true"][data-margo-sticky-edge="block-start"] { inset-block-start: var(--margo-sticky-offset, 0); }
[data-margo-sticky="true"][data-margo-sticky-edge="block-end"] { inset-block-end: var(--margo-sticky-offset, 0); }
.margo-site-brand { display: inline-flex; align-items: center; gap: 0.5rem; min-block-size: 2.75rem; font-weight: 700; color: var(--margo-text-strong); text-decoration: none; }
.margo-site-brand { inline-size: max-content; min-inline-size: max-content; max-inline-size: none; flex: 0 0 auto; white-space: nowrap; }
.margo-site-brand img { max-block-size: 2rem; max-inline-size: 10rem; }
.margo-site-navbar { inline-size: 100%; min-inline-size: 0; }
.margo-breadcrumbs ol, .margo-pagination ul, .margo-locale-controls ul { display: flex; flex-wrap: wrap; gap: 0.75rem; margin: 0; padding: 0; list-style: none; }
.margo-pagination { margin-block-start: 2rem; padding-block-start: 1rem; border-block-start: 1px solid var(--margo-outline); }
.margo-showcase-article .margo-pagination ul { justify-content: space-between; column-gap: 2rem; row-gap: 0.75rem; }
.margo-footer { margin-block-start: 1rem; padding-block-start: 1rem; border-block-start: 1px solid var(--margo-outline); }
.margo-showcase-article .margo-pagination a { color: var(--margo-accent); font-weight: 600; text-decoration: none; }
.margo-showcase-article .margo-pagination a:hover, .margo-showcase-article .margo-pagination a:focus-visible { color: var(--margo-text-strong); text-decoration: underline; }
:where(.margo-frame button, .margo-frame a) { min-block-size: 2.75rem; }
:where(.margo-frame button) { color: inherit; background: var(--margo-surface-alt); border: 1px solid var(--margo-outline); border-radius: 0.25rem; padding-inline: 0.75rem; }
:where(.margo-frame :focus-visible, .margo-document :focus-visible) { outline: 3px solid var(--margo-accent); outline-offset: 2px; }
@media (min-width: 720px) {
  .margo-frame { grid-template-columns: minmax(12rem, 16rem) minmax(0, var(--margo-reading-measure)); grid-template-areas: "top-nav top-nav" "left-nav main-content" "footer footer"; justify-content: center; }
  .margo-frame--fluid { grid-template-columns: minmax(0, 1fr); grid-template-areas: "top-nav" "main-content" "footer"; }
}
@media (min-width: 1100px) {
  .margo-frame { grid-template-columns: 16rem minmax(0, var(--margo-reading-measure)); }
}
@media (max-width: 719px) {
  .margo-frame { display: block; padding: 0.75rem; }
  .margo-frame > .margo-area { margin-block-end: var(--margo-gap); }
  .margo-area--left-nav, .margo-area--right-nav { border-block: 1px solid var(--margo-outline); padding-block: 0.75rem; }
}
@media (prefers-reduced-motion: reduce) { *, *::before, *::after { scroll-behavior: auto !important; transition-duration: 0.01ms !important; } }
@media print { .margo-skip-link, .margo-area--top-nav, .margo-area--left-nav, .margo-area--right-nav, .margo-area--footer { display: none !important; } .margo-frame { display: block; padding: 0; } .margo-area--main-content { max-inline-size: none; } }
.margo-showcase-article { inline-size: min(100%, 78ch); margin-inline: auto; padding-block: clamp(1.5rem, 4vw, 4rem); }
.margo-showcase-article .margo-breadcrumbs { margin-block-end: 2rem; color: var(--margo-text); font-size: 0.875rem; }
.margo-showcase-article .margo-breadcrumbs a { color: inherit; text-decoration: none; }
.margo-showcase-article .margo-document { color: var(--margo-text); }
[data-margo-layout="landing"] .margo-showcase-article .margo-document h1 { color: var(--margo-text-strong); font-size: clamp(2.25rem, 6vw, 4.5rem); letter-spacing: -0.04em; line-height: 1.05; }
[data-margo-layout="landing"] .margo-showcase-article .margo-document__lead { color: var(--margo-accent); font-size: clamp(1.15rem, 2vw, 1.5rem); font-weight: 650; }
[data-margo-layout="landing"] .margo-showcase-article .margo-document h2 { margin-block-start: 3rem; color: var(--margo-text-strong); letter-spacing: -0.02em; }
[data-margo-layout="landing"] .margo-showcase-article .margo-document blockquote { margin-block-start: 1.25rem; margin-inline: 0; border-inline-start: 0.25rem solid var(--margo-accent); padding-inline: 1rem; color: var(--margo-text-strong); }
[data-margo-layout="landing"] .margo-showcase-article .margo-document img[alt^="Margo mascot"] { display: block; inline-size: min(100%, 24rem); aspect-ratio: 4 / 3; object-fit: cover; object-position: center 62%; margin-block: 1.5rem 2rem; margin-inline: auto; border: 1px solid var(--margo-outline); border-radius: 1rem; box-shadow: 0 1rem 2.5rem rgb(11 18 32 / 18%); }
.component-doc-shell__brand-mark { display: none !important; }
.margo-shell-search { width: clamp(11rem, 26vw, 18rem); min-width: 0; flex: 0 1 auto; }
.margo-shell-search-trigger { min-block-size: 2.75rem; }
@media (width < 30rem) {
  .component-doc-shell__brand-mark { display: grid !important; }
  .margo-shell-search { width: 2.75rem; flex: 0 0 2.75rem; }
  .margo-shell-search-trigger { width: 2.75rem; min-width: 2.75rem; padding-inline: 0; justify-content: center; }
  .margo-shell-search-trigger > span { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0, 0, 0, 0); clip-path: inset(50%); white-space: nowrap; }
  .margo-shell-search-trigger > kbd { display: none; }
}
.component-doc-shell__main { view-transition-name: margo-main-content; transition: opacity 160ms ease, transform 160ms ease; }
.component-doc-shell__main.htmx-swapping { opacity: 0; transform: translateY(0.35rem); }
.component-doc-shell__main.htmx-settling { animation: margo-main-content-enter 240ms cubic-bezier(0.2, 0.8, 0.2, 1) both; }
@keyframes margo-main-content-enter { from { opacity: 0; transform: translateY(0.35rem); } to { opacity: 1; transform: translateY(0); } }
@keyframes margo-main-content-exit { from { opacity: 1; transform: translateY(0); } to { opacity: 0; transform: translateY(-0.2rem); } }
::view-transition-old(margo-main-content) { animation: margo-main-content-exit 160ms ease both; }
::view-transition-new(margo-main-content) { animation: margo-main-content-enter 240ms cubic-bezier(0.2, 0.8, 0.2, 1) both; }
@media (prefers-reduced-motion: reduce) {
  .component-doc-shell__main { view-transition-name: none; transition: none; }
  .component-doc-shell__main.htmx-swapping, .component-doc-shell__main.htmx-settling { opacity: 1; transform: none; animation: none; }
  ::view-transition-old(margo-main-content), ::view-transition-new(margo-main-content) { animation: none; }
}
.margo-shell-footer { margin: 0; color: var(--margo-text); font-size: 0.875rem; }
`

// configuredProfileLayoutCSS is consumer-owned semantic presentation for
// layout-profile pages. It deliberately names only Margo frame/layout hooks;
// Goshtoso component internals and App Shell-private selectors stay opaque.
const configuredProfileLayoutCSS = `[data-margo-layout="landing"].margo-frame--top-main-footer {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  grid-template-areas: "top-nav" "main-content" "footer";
  justify-content: stretch;
}
[data-margo-layout="landing"] .margo-area--main-content { max-inline-size: none; }
[data-margo-layout="landing"] .margo-showcase-article { inline-size: 100%; max-inline-size: none; }
[data-margo-layout="landing"] .margo-document { max-inline-size: 100%; }
[data-margo-layout="landing"] .margo-area--top-nav { gap: 0; }
[data-margo-layout="docs"] .margo-area--main-content { max-inline-size: var(--margo-reading-measure); }
[data-margo-layout="docs"] .margo-showcase-article { inline-size: min(100%, var(--margo-reading-measure)); max-inline-size: 100%; }
[data-margo-layout="docs"] .margo-area--left-nav { min-inline-size: 0; padding-block: 0.5rem; }
[data-margo-layout="docs"] .margo-area--right-nav { min-inline-size: 0; padding-inline-start: 0.5rem; }
[data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--left-nav,
[data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--main-content,
[data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--right-nav { min-inline-size: 0; }
[data-margo-layout="landing"] :where(a, button):focus-visible,
[data-margo-layout="docs"] :where(a, button):focus-visible { outline: 3px solid var(--margo-accent); outline-offset: 2px; }
[data-margo-layout="landing"] .margo-site-family-links a,
[data-margo-layout="docs"] .margo-site-family-links a,
[data-margo-layout="landing"] .margo-site-search button,
[data-margo-layout="docs"] .margo-site-search button { min-block-size: 2.75rem; }
[data-margo-layout="landing"] .margo-area--top-nav > nav,
[data-margo-layout="docs"] .margo-area--top-nav > nav,
[data-margo-layout="landing"] .margo-site-family-links,
[data-margo-layout="docs"] .margo-site-family-links,
[data-margo-layout="landing"] .margo-area--top-nav > *,
[data-margo-layout="docs"] .margo-area--top-nav > * { min-inline-size: 0; max-inline-size: 100%; }
[data-margo-layout="landing"] .margo-site-navbar > div:first-of-type,
[data-margo-layout="docs"] .margo-site-navbar > div:first-of-type { min-inline-size: 0; flex: 1 1 auto; }
[data-margo-layout="landing"] .margo-site-navbar > div:first-of-type > a,
[data-margo-layout="docs"] .margo-site-navbar > div:first-of-type > a { inline-size: auto; max-inline-size: none; flex: 0 0 auto; min-inline-size: max-content; }
[data-margo-layout="landing"] .margo-site-brand,
[data-margo-layout="docs"] .margo-site-brand { display: inline-flex; align-items: center; gap: 0.5rem; inline-size: auto; flex: 0 0 auto; min-inline-size: max-content; white-space: nowrap; }
[data-margo-layout="landing"] .margo-site-brand img,
[data-margo-layout="docs"] .margo-site-brand img { inline-size: 2rem; block-size: 2rem; flex: 0 0 2rem; margin-inline-end: 0; object-fit: contain; vertical-align: middle; }
[data-margo-layout="landing"] .margo-site-repository,
[data-margo-layout="docs"] .margo-site-repository { display: inline-flex; align-items: center; justify-content: center; inline-size: 2.75rem; block-size: 2.75rem; color: var(--margo-accent); text-decoration: none; }
[data-margo-layout="landing"] .margo-site-repository svg,
[data-margo-layout="docs"] .margo-site-repository svg { inline-size: 1.5rem; block-size: 1.5rem; }
[data-margo-layout="landing"] .margo-site-repository:hover,
[data-margo-layout="landing"] .margo-site-repository:focus-visible,
[data-margo-layout="docs"] .margo-site-repository:hover,
[data-margo-layout="docs"] .margo-site-repository:focus-visible { color: var(--margo-text-strong); }
[data-margo-layout="landing"] .margo-site-navbar > div:first-of-type > .margo-site-search,
[data-margo-layout="docs"] .margo-site-navbar > div:first-of-type > .margo-site-search { inline-size: clamp(11rem, 26vw, 18rem); min-inline-size: 0; flex: 0 1 auto; }
[data-margo-layout="landing"] .margo-site-family-links,
[data-margo-layout="docs"] .margo-site-family-links { flex: 1 1 100%; }
[data-margo-layout="landing"] .margo-search-clear,
[data-margo-layout="docs"] .margo-search-clear { min-block-size: 2rem; border: 0; padding: 0.25rem 0.5rem; color: var(--margo-text); background: transparent; font-size: 0.875rem; white-space: nowrap; }
[data-margo-layout="landing"] .margo-search-clear:hover,
[data-margo-layout="landing"] .margo-search-clear:focus-visible,
[data-margo-layout="docs"] .margo-search-clear:hover,
[data-margo-layout="docs"] .margo-search-clear:focus-visible { color: var(--margo-text-strong); background: var(--margo-surface-alt); }
.margo-search-status { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip: rect(0 0 0 0); clip-path: inset(50%); white-space: nowrap; }
@media (width < 30rem) {
  [data-margo-layout="landing"] .margo-site-navbar > div:first-of-type > .margo-site-search,
  [data-margo-layout="docs"] .margo-site-navbar > div:first-of-type > .margo-site-search { inline-size: 2.75rem; flex: 0 0 2.75rem; }
  [data-margo-layout="landing"] .margo-site-search,
  [data-margo-layout="docs"] .margo-site-search { width: 2.75rem; flex: 0 0 2.75rem; }
  [data-margo-layout="landing"] .margo-site-search button,
  [data-margo-layout="docs"] .margo-site-search button { width: 2.75rem; min-width: 2.75rem; padding-inline: 0; justify-content: center; }
  [data-margo-layout="landing"] .margo-site-search button > span,
  [data-margo-layout="docs"] .margo-site-search button > span,
  [data-margo-layout="landing"] .margo-site-search button > kbd,
  [data-margo-layout="docs"] .margo-site-search button > kbd { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0, 0, 0, 0); clip-path: inset(50%); white-space: nowrap; }
}
@media (min-width: 1200px) {
  [data-margo-layout="docs"].margo-frame--top-left-main-right-footer {
    grid-template-columns: minmax(12rem, 16rem) minmax(0, var(--margo-reading-measure)) minmax(12rem, 16rem);
    grid-template-areas: "top-nav top-nav top-nav" "left-nav main-content right-nav" "footer footer footer";
    justify-content: center;
  }
  [data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--left-nav,
  [data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--main-content,
  [data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--right-nav { max-inline-size: 100%; }
}
@media (min-width: 880px) and (max-width: 1199px) {
  [data-margo-layout="docs"].margo-frame--top-left-main-right-footer {
    grid-template-columns: minmax(9rem, 11rem) minmax(0, 1fr) minmax(9rem, 11rem);
    grid-template-areas: "top-nav top-nav top-nav" "left-nav main-content right-nav" "footer footer footer";
    justify-content: center;
  }
}
@media (max-width: 879px) {
  [data-margo-layout="landing"] .margo-showcase-article,
  [data-margo-layout="docs"] .margo-showcase-article { inline-size: 100%; }
  [data-margo-layout="docs"].margo-frame--top-left-main-right-footer {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    grid-template-areas: "top-nav" "left-nav" "main-content" "right-nav" "footer";
    overflow-x: clip;
  }
  [data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--left-nav,
  [data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--main-content,
  [data-margo-layout="docs"].margo-frame--top-left-main-right-footer > .margo-area--right-nav {
    min-inline-size: 0;
    max-inline-size: 100%;
    overflow-inline: hidden;
  }
  [data-margo-layout="docs"] .margo-area--right-nav { padding-inline-start: 0; }
}
`

func configuredSiteStylesheet(profileMode bool) string {
	if !profileMode {
		return configuredSiteCSS + "\n" + pageActionsCSS
	}
	// Keep legacy shell CSS byte-for-byte for shell mode, while profile output
	// omits App Shell-private selectors entirely.
	profileCSS := configuredSiteCSS
	if index := strings.Index(profileCSS, ".component-doc-shell__brand-mark"); index >= 0 {
		profileCSS = profileCSS[:index]
	}
	return profileCSS + configuredProfileLayoutCSS + "\n" + pageActionsCSS
}

func buildConfigured(ctx context.Context, request ConfigRequest, config Config) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	absoluteConfig, err := filepath.Abs(request.ConfigPath)
	if err != nil {
		return Result{}, fmt.Errorf("site.config_invalid: %w", err)
	}
	configDir := filepath.Dir(absoluteConfig)
	sourceDir := filepath.Join(configDir, filepath.FromSlash(config.Source))
	info, err := os.Stat(sourceDir)
	if err != nil || !info.IsDir() {
		return Result{}, diagnostic("site.source_unreadable", fmt.Sprintf("source directory %q is unavailable", config.Source), "Create the configured source directory.", absoluteConfig)
	}
	inputs, err := discoverConfiguredInputs(ctx, sourceDir, config.Navigation.Exclude)
	if err != nil {
		return Result{}, err
	}
	sources := inputs.Sources
	if len(sources) == 0 {
		return Result{}, diagnostic("site.sources_empty", "configured source directory contains no public Markdown files", "Add a Markdown document or change navigation.exclude.", config.Source)
	}
	if config.Frame != nil && (config.Frame.Command != "" || config.Frame.GoModule != nil) {
		return Result{}, diagnostic("site.layout_unavailable", "external frame distribution is not implemented by this builder", "Use a builtin frame until the command and Go-module contracts are wired.", "frame")
	}
	locale := config.Locales.Default
	shellMode := config.Shell != nil
	profileMode := configUsesLayoutProfiles(config)
	typedLayoutMode := config.Layout != nil
	frameName := ""
	frame := ssg.Frame(nil)
	schema := ssg.FrameSchema{}
	frameValues := ssg.Values(nil)
	frameHash := ""
	layoutIdentity := ""
	layoutSchemaHash := ""
	presentations := map[string]PagePresentation(nil)
	if shellMode {
		layoutIdentity = "shell:" + config.Shell.Builtin
		frameHash = componentDocShellSchemaHash(config)
		layoutSchemaHash = frameHash
	} else if profileMode {
		var frameErr error
		presentations, frameErr = prepareFramePresentations(config)
		if frameErr != nil {
			return Result{}, frameErr
		}
		layoutIdentity, layoutSchemaHash = profileLayoutIdentity(presentations)
	} else {
		frameName = "top-left-main-footer"
		if config.Frame != nil && config.Frame.Builtin != "" {
			frameName = config.Frame.Builtin
		}
		var frameErr error
		frame, frameErr = ssg.BuiltinFrame(frameName)
		if frameErr != nil {
			return Result{}, diagnostic("site.layout_unknown", frameErr.Error(), "Select a supported builtin frame.", frameName)
		}
		schema, frameErr = frame.Schema(ssg.FrameContext{Locale: locale, Direction: localeDirection(locale), Profile: ssg.DocsProfile, Root: true, InstanceID: "root", Theme: ssg.ThemeContext{Name: config.Theme.Name, ColorMode: config.Theme.ColorMode}})
		if frameErr != nil {
			return Result{}, frameErr
		}
		if frameErr := ssg.ValidateFrameSchema(schema, ssg.DocsProfile); frameErr != nil {
			return Result{}, diagnostic("site.layout_invalid", frameErr.Error(), "Select a navigation-capable root frame.", frameName)
		}
		if config.Frame != nil {
			frameValues = ssg.Values(config.Frame.Values)
		}
		var valueErr error
		frameValues, valueErr = ssg.ResolveFrameValues(schema, frameValues)
		if valueErr != nil {
			return Result{}, diagnostic("site.layout_values_invalid", valueErr.Error(), "Use only option paths and values published by the selected frame.", "frame.values")
		}
		frameHash, err = ssg.SchemaHashForValues(schema, frameValues)
		if err != nil {
			return Result{}, err
		}
		if typedLayoutMode {
			layoutIdentity = "layout:" + string(config.Layout.Kind)
		} else {
			layoutIdentity = "frame:" + frameName
			layoutSchemaHash = frameHash
		}
	}
	if request.Compiler == nil {
		request.Compiler = margo.New()
	}
	if request.AssetReader == nil {
		request.AssetReader = margo.FilesystemCheckAssetReader{}
	}
	assets := AssetMode(config.Assets)
	shellAssetPrefix := ""
	shellName := ""
	if shellMode {
		shellAssetPrefix = componentDocShellAssetPrefix(config.BasePath)
		shellName = config.Shell.Builtin
	}
	b := &builder{
		request: requestToSiteRequest(request, sourceDir, sources, assets), config: &config,
		configSource: absoluteConfig, configDir: configDir, sourceDir: sourceDir, frame: frame, frameSchema: schema,
		frameHash: frameHash, frameValues: frameValues, profileMode: profileMode, presentations: presentations, layoutName: frameName, shellMode: shellMode,
		shellName: shellName, shellAssetPrefix: shellAssetPrefix,
		layoutPatches: append([]LayoutPatch(nil), inputs.Patches...),
		configured:    make(map[string]configuredPage),
		sources:       make(map[string]Source, len(sources)), outputs: make(map[string]string, len(sources)),
		artifacts: make(map[string][]byte), artifactKeys: make(map[string]string), assets: make(map[string]cachedAsset),
		dependencies: make(map[string]string), pdfEngine: request.PDFEngine, pdfInstances: margo.NewInstanceAllocator(),
		siteManifest: SiteManifest{ConfigVersion: 1, Layout: layoutIdentity, LayoutSchemaHash: layoutSchemaHash, BaseURL: strings.TrimSuffix(config.Site.BaseURL, "/"), BasePath: normalizedBasePath(config.BasePath)},
	}
	ordered, err := b.indexSources()
	if err != nil {
		return Result{}, err
	}
	if err := b.preflightConfigured(ctx, ordered); err != nil {
		return Result{}, err
	}
	if err := b.validateConfiguredFamilyOverviews(ordered); err != nil {
		return Result{}, err
	}
	homeFound := false
	for _, page := range b.configPages {
		if page.Source == config.Site.Home && page.Locale == config.Locales.Default {
			homeFound = true
			break
		}
	}
	if !homeFound {
		return Result{}, diagnostic("site.home_missing", fmt.Sprintf("configured home %q is not a public default-locale page", config.Site.Home), "Add the home Markdown file or select an existing default-locale route.", config.Site.Home)
	}
	if err := b.stageConfiguredAssets(config); err != nil {
		return Result{}, err
	}
	b.siteManifest.DocumentStyleDigest = b.documentStyleDigest()
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
	if err := b.addPublicationDiscoveryArtifacts(); err != nil {
		return Result{}, err
	}
	result := b.result()
	result.Site.Routes = append([]Page(nil), b.configPages...)
	return result, nil
}

func requestToSiteRequest(request ConfigRequest, sourceDir string, sources []Source, assets AssetMode) Request {
	return Request{SourceRoot: sourceDir, Sources: sources, Compiler: request.Compiler, Assets: assets, AssetReader: request.AssetReader, PDFEngine: request.PDFEngine}
}

func discoverConfiguredSources(ctx context.Context, root string, excludes []string) ([]Source, error) {
	inputs, err := discoverConfiguredInputs(ctx, root, excludes)
	if err != nil {
		return nil, err
	}
	return inputs.Sources, nil
}

func discoverConfiguredInputs(ctx context.Context, root string, excludes []string) (configuredInputs, error) {
	sourcePaths := make([]string, 0)
	patchPaths := make([]string, 0)
	err := filepath.WalkDir(root, func(name string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		relative = norm.NFC.String(filepath.ToSlash(relative))
		layoutPatch := entry.Name() == directoryLayoutPatchName
		if entry.Type()&os.ModeSymlink != 0 {
			if layoutPatch {
				return invalidDirectoryLayoutPatch(relative, "", "layout patch must be a regular file, not a symlink")
			}
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			if layoutPatch {
				return invalidDirectoryLayoutPatch(relative, "", "layout patch must be a regular file")
			}
			return fmt.Errorf("site.source_invalid: %s is not a regular file", name)
		}
		if layoutPatch {
			patchPaths = append(patchPaths, relative)
			return nil
		}
		if !isMarkdownPath(relative) || pathExcluded(relative, excludes) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > margo.MaxDocumentBytes {
			return fmt.Errorf("site.input_too_large: %s exceeds %d bytes", relative, margo.MaxDocumentBytes)
		}
		sourcePaths = append(sourcePaths, relative)
		return nil
	})
	if err != nil {
		return configuredInputs{}, fmt.Errorf("site.source_unreadable: %w", err)
	}
	sort.Strings(sourcePaths)
	sort.Strings(patchPaths)
	inputs := configuredInputs{
		Sources: make([]Source, 0, len(sourcePaths)),
		Patches: make([]LayoutPatch, 0, len(patchPaths)),
	}
	for _, relative := range sourcePaths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return configuredInputs{}, fmt.Errorf("site.input_read: %s: %w", relative, err)
		}
		inputs.Sources = append(inputs.Sources, Source{Path: relative, Content: data})
	}
	for _, relative := range patchPaths {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return configuredInputs{}, fmt.Errorf("site.input_read: %s: %w", relative, err)
		}
		patch, err := decodeDirectoryLayoutPatch(relative, data)
		if err != nil {
			return configuredInputs{}, err
		}
		inputs.Patches = append(inputs.Patches, patch)
	}
	return inputs, nil
}

func pathExcluded(value string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := path.Match(pattern, value)
		if err == nil && matched {
			return true
		}
		if strings.HasSuffix(pattern, "/**") && strings.HasPrefix(value, strings.TrimSuffix(pattern, "**")) {
			return true
		}
	}
	return false
}

func configuredOutputPath(source string, locales LocaleConfig) string {
	locale, rest := sourceLocale(source, locales)
	output := outputPath(rest)
	if locale == locales.Default {
		return output
	}
	return path.Join(strings.ToLower(locale), output)
}

func sourceLocale(source string, locales LocaleConfig) (string, string) {
	for _, candidate := range locales.Supported {
		prefix := strings.ToLower(candidate)
		lower := strings.ToLower(source)
		if lower == prefix || strings.HasPrefix(lower, prefix+"/") {
			rest := strings.TrimPrefix(source, source[:len(candidate)])
			rest = strings.TrimPrefix(rest, "/")
			return candidate, rest
		}
	}
	return locales.Default, source
}

func normalizedBasePath(value string) string {
	if value == "" || value == "/" {
		return "/"
	}
	cleaned, _ := normalizeBasePath(value)
	return cleaned
}

func localeDirection(value string) string {
	tag, err := language.Parse(value)
	if err != nil {
		return "ltr"
	}
	base, _ := tag.Base()
	if base.String() == "ar" || base.String() == "fa" || base.String() == "he" || base.String() == "ur" {
		return "rtl"
	}
	return "ltr"
}

func (b *builder) preflightConfigured(ctx context.Context, sources []Source) error {
	var siteCascade layoutCascade
	var siteLayout ResolvedLayout
	if b.config.Layout != nil {
		var err error
		siteCascade, err = resolveSiteLayout(*b.config.Layout, "")
		if err != nil {
			return err
		}
		siteLayout = siteCascade.resolved()
	}

	for _, source := range sources {
		base := filepath.Join(b.sourceDir, filepath.FromSlash(path.Dir(source.Path)))
		document, err := b.request.Compiler.Compile(ctx, margo.Source{Name: source.Path, Content: source.Content, BaseURL: base})
		if err != nil {
			return err
		}
		rendered, err := b.request.Compiler.Render(ctx, document, margo.WithTableSort(margo.TableSortClient), margo.WithRenderTarget(margo.TargetSite))
		if err != nil {
			return err
		}
		result, err := margo.RenderHTML(rendered)
		if err != nil {
			return err
		}
		article, err := renderComponentBytes(result.Fragment())
		if err != nil {
			return err
		}
		if count := countElements(article, "h1"); count != 1 {
			return diagnostic("site.h1_invalid", fmt.Sprintf("document must contain exactly one h1, found %d", count), "Add exactly one document-level h1.", source.Path)
		}
		locale, _ := sourceLocale(source.Path, b.config.Locales)
		metadata := result.Metadata()
		if metadata.Language != "" && !strings.EqualFold(metadata.Language, locale) {
			return diagnostic("site.locale_conflict", fmt.Sprintf("frontmatter language %q conflicts with source locale %q", metadata.Language, locale), "Align frontmatter language with the locale path.", source.Path)
		}
		title := strings.TrimSpace(metadata.Title)
		if title == "" {
			return diagnostic("site.title_required", "public page has no usable title", "Add frontmatter title or a document h1.", source.Path)
		}
		description := strings.TrimSpace(metadata.Description)
		if description == "" {
			if source.Path == b.config.Site.Home && locale == b.config.Locales.Default && strings.TrimSpace(b.config.Site.Description) != "" {
				description = strings.TrimSpace(b.config.Site.Description)
			} else {
				description = routeDescription(result.PlainText(), title, b.config.Site.Name)
			}
		}
		page := Page{Source: source.Path, Output: b.pageOutput(source.Path), Locale: locale, Title: title, Description: description, DocumentDigest: documentPayloadDigest(article), ImageOverflow: pageImageOverflowForMetadata(document.Metadata()), Actions: pageActionsForMetadata(document.Metadata())}
		resolvedLayout := ResolvedLayout{}
		layoutSources := []string(nil)
		presentation := PagePresentation{}
		family := FamilyConfig{}
		if b.config.Layout != nil {
			cascade := siteCascade
			for _, patch := range layoutPatchChain(source.Path, b.layoutPatches) {
				cascade, err = cascade.apply(patch)
				if err != nil {
					return err
				}
				layoutSources = append(layoutSources, patch.Source)
			}
			markdownPatch, patchErr := layoutPatchFromMetadata(document.Metadata(), source.Path)
			if patchErr != nil {
				return patchErr
			}
			if markdownPatch.Source != "" {
				cascade, err = cascade.apply(markdownPatch)
				if err != nil {
					return err
				}
				layoutSources = append(layoutSources, markdownPatch.Source)
			}
			resolvedLayout = cascade.resolved()
			entry, exists := cascade.registry.lookup(resolvedLayout.Kind)
			if !exists {
				return fmt.Errorf("site.layout_missing: resolved layout kind %q is unavailable", resolvedLayout.Kind)
			}
			resolvedLayout.Values, err = entry.validateValues(resolvedLayout.Values, layoutValueSiteDefault, "/layout/values")
			if err != nil {
				return presentationSourceDiagnostic(err, source.Path)
			}
			resolvedLayout.Family = ""
			if resolvedLayout.Kind == LayoutDocs {
				resolvedLayout.Family, _ = resolvedLayout.Values["family"].(string)
				page.Family = resolvedLayout.Family
			}
			resolvedLayout.Identity, err = configuredPageLayoutIdentity(source.Path, resolvedLayout, layoutSources)
			if err != nil {
				return err
			}
			page.Layout = string(resolvedLayout.Kind)
		} else if b.profileMode {
			if len(b.config.Navigation.Families) > 0 {
				family, err = resolveFamily(source.Path, b.config.Locales, b.config.Navigation.Families)
				if err != nil {
					return err
				}
			}
			layout, layoutErr := resolveLayout(document.Metadata(), family, b.config.Layouts)
			if layoutErr != nil {
				return layoutErr
			}
			var exists bool
			presentation, exists = b.presentations[layout]
			if !exists {
				return presentationSourceDiagnostic(newPresentationDiagnostic(
					"site.layout_unknown",
					fmt.Sprintf("layout profile %q is not prepared", layout),
					"Select a declared layouts.profiles name.",
					"/layouts/profiles",
				), source.Path)
			}
			presentation.FamilyID = family.ID
			page.Family = family.ID
			page.Layout = layout
		}
		page.Canonical = b.pageURL(page)
		b.configured[source.Path] = configuredPage{page: page, layout: resolvedLayout, layoutSources: layoutSources, presentation: presentation, article: article, plainText: result.PlainText(), result: result, document: document}
		b.configPages = append(b.configPages, page)
	}
	sort.Slice(b.configPages, func(i, j int) bool { return pageRouteLess(b.configPages[i], b.configPages[j]) })
	for index := range b.configPages {
		for _, candidate := range b.configPages {
			if candidate.Locale == b.configPages[index].Locale || routeKey(candidate.Source, b.config.Locales) != routeKey(b.configPages[index].Source, b.config.Locales) {
				continue
			}
			b.configPages[index].Alternates = append(b.configPages[index].Alternates, Alternate{Locale: candidate.Locale, URL: b.pageURL(candidate)})
		}
		sort.Slice(b.configPages[index].Alternates, func(left, right int) bool {
			return b.configPages[index].Alternates[left].Locale < b.configPages[index].Alternates[right].Locale
		})
		prepared := b.configured[b.configPages[index].Source]
		prepared.page = b.configPages[index]
		b.configured[b.configPages[index].Source] = prepared
	}
	if b.config.Layout != nil {
		if err := b.buildDocsFamilies(siteLayout); err != nil {
			return err
		}
		identity, err := configuredSiteLayoutIdentity(siteLayout, b.layoutPatches, b.configured)
		if err != nil {
			return err
		}
		b.siteManifest.LayoutSchemaHash = identity
	}
	return nil
}

func (b *builder) buildDocsFamilies(siteLayout ResolvedLayout) error {
	familyIDs := docsFamilyIDs(siteLayout)
	if siteLayout.Kind != LayoutDocs {
		for _, page := range b.configPages {
			if page.Layout == string(LayoutDocs) {
				familyIDs = docsFamilyIDs(b.configured[page.Source].layout)
				break
			}
		}
	}

	b.docsFamilies = nil
	for familyIndex, familyID := range familyIDs {
		byLocale := make(map[string][]Page)
		for _, page := range b.configPages {
			if page.Layout != string(LayoutDocs) || page.Family != familyID {
				continue
			}
			byLocale[page.Locale] = append(byLocale[page.Locale], page)
		}
		if familyID != "default" && len(byLocale) == 0 {
			return presentationSourceDiagnostic(newPresentationDiagnostic(
				"site.family_empty",
				fmt.Sprintf("docs family %q has no docs page", familyID),
				"Add a docs page selecting this family or remove the declaration.",
				fmt.Sprintf("/layout/default/families/%d", familyIndex),
			), b.configSource)
		}

		locales := make([]string, 0, len(byLocale))
		for locale := range byLocale {
			locales = append(locales, locale)
		}
		sort.Strings(locales)
		for _, locale := range locales {
			b.docsFamilies = append(b.docsFamilies, docsFamily{
				ID:       familyID,
				Locale:   locale,
				Overview: docsFamilyOverview(byLocale[locale]),
			})
		}
	}
	return nil
}

func docsFamilyIDs(layout ResolvedLayout) []string {
	raw, ok := layout.Values["families"]
	if !ok {
		return []string{"default"}
	}
	values, ok := layoutListValues(raw)
	if !ok || len(values) == 0 {
		return []string{"default"}
	}
	ids := make([]string, 0, len(values))
	for _, value := range values {
		id, ok := value.(string)
		if ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func docsFamilyOverview(pages []Page) Page {
	if len(pages) == 0 {
		return Page{}
	}
	for _, page := range pages {
		if strings.EqualFold(path.Base(page.Source), "index.md") {
			return page
		}
	}
	return pages[0]
}

func configuredPageLayoutIdentity(source string, layout ResolvedLayout, patchSources []string) (string, error) {
	sources := make([]any, len(patchSources))
	for index := range patchSources {
		sources[index] = patchSources[index]
	}
	payload := map[string]any{
		"source":        source,
		"kind":          string(layout.Kind),
		"values":        layout.Values,
		"patch_sources": sources,
	}
	var canonical bytes.Buffer
	if err := writeCanonicalLayoutValue(&canonical, payload); err != nil {
		return "", fmt.Errorf("site.layout_identity: %w", err)
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte("margo.site.page-layout/v1\x00"))
	_, _ = hash.Write(canonical.Bytes())
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func configuredSiteLayoutIdentity(siteLayout ResolvedLayout, patches []LayoutPatch, configured map[string]configuredPage) (string, error) {
	sources := make([]string, 0, len(configured))
	for source := range configured {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	pages := make([]any, 0, len(sources))
	for _, source := range sources {
		pages = append(pages, map[string]any{
			"source":   source,
			"identity": configured[source].layout.Identity,
		})
	}
	orderedPatches := append([]LayoutPatch(nil), patches...)
	sort.Slice(orderedPatches, func(left, right int) bool { return orderedPatches[left].Source < orderedPatches[right].Source })
	patchIdentities := make([]any, 0, len(orderedPatches))
	for _, patch := range orderedPatches {
		identity, err := json.Marshal(map[string]any{
			"source": patch.Source,
			"kind":   string(patch.Kind),
			"values": patch.Values,
		})
		if err != nil {
			return "", fmt.Errorf("site.layout_identity: %w", err)
		}
		patchIdentities = append(patchIdentities, string(identity))
	}
	payload := map[string]any{
		"registry": configuredLayoutRegistryIdentity(builtinLayoutRegistry()),
		"site": map[string]any{
			"kind":   string(siteLayout.Kind),
			"values": siteLayout.Values,
		},
		"directory_patches": patchIdentities,
		"pages":             pages,
	}
	var canonical bytes.Buffer
	if err := writeCanonicalLayoutValue(&canonical, payload); err != nil {
		return "", fmt.Errorf("site.layout_identity: %w", err)
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte("margo.site.configured-layout/v1\x00"))
	_, _ = hash.Write(canonical.Bytes())
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func configuredLayoutRegistryIdentity(registry layoutRegistry) []any {
	entries := make([]any, 0, len(registry.order))
	for _, kind := range registry.order {
		entry, exists := registry.lookup(kind)
		if !exists {
			continue
		}
		entries = append(entries, map[string]any{
			"kind":     string(kind),
			"defaults": entry.defaults,
			"schema":   configuredLayoutSchemaIdentity(entry.schema),
		})
	}
	return entries
}

func configuredLayoutSchemaIdentity(schema layoutValueSchema) map[string]any {
	properties := make(map[string]any, len(schema.Properties))
	for name, property := range schema.Properties {
		properties[name] = configuredLayoutSchemaIdentity(property)
	}
	enums := make([]any, len(schema.Enum))
	for index := range schema.Enum {
		enums[index] = schema.Enum[index]
	}
	return map[string]any{
		"type":              configuredLayoutValueTypeIdentity(schema.Type),
		"properties":        properties,
		"enum":              enums,
		"site_default_only": schema.SiteDefaultOnly,
	}
}

func configuredLayoutValueTypeIdentity(value layoutValueType) string {
	switch value {
	case layoutObject:
		return "object"
	case layoutBool:
		return "boolean"
	case layoutString:
		return "string"
	case layoutStringList:
		return "string-list"
	default:
		return "unknown"
	}
}

// validateConfiguredFamilyOverviews enforces the publication contract before
// any page is rendered. A configured family link must always resolve: the
// declared overview source must exist, and every locale represented by that
// family must have the same overview route.
func (b *builder) validateConfiguredFamilyOverviews(sources []Source) error {
	if !b.profileMode || len(b.config.Navigation.Families) == 0 {
		return nil
	}
	for index, family := range b.config.Navigation.Families {
		representedLocales := make(map[string]struct{})
		overviewFound := false
		for _, source := range sources {
			resolved, err := resolveFamily(source.Path, b.config.Locales, b.config.Navigation.Families)
			if err != nil {
				return err
			}
			if resolved.ID != family.ID {
				continue
			}
			locale, _ := sourceLocale(source.Path, b.config.Locales)
			representedLocales[locale] = struct{}{}
			if strings.EqualFold(source.Path, family.Overview) {
				overviewFound = true
			}
		}
		pointer := fmt.Sprintf("/navigation/families/%d/overview", index)
		if !overviewFound {
			return presentationSourceDiagnostic(newPresentationDiagnostic(
				"site.family_overview_missing",
				fmt.Sprintf("family %q overview %q was not discovered", family.ID, family.Overview),
				"Add the configured overview Markdown source before building the site.",
				pointer,
			), family.Overview)
		}
		overviewRoute := routeKey(family.Overview, b.config.Locales)
		locales := make([]string, 0, len(representedLocales))
		for locale := range representedLocales {
			locales = append(locales, locale)
		}
		sort.Strings(locales)
		for _, locale := range locales {
			found := false
			for _, source := range sources {
				candidateLocale, _ := sourceLocale(source.Path, b.config.Locales)
				if candidateLocale != locale || routeKey(source.Path, b.config.Locales) != overviewRoute {
					continue
				}
				resolved, err := resolveFamily(source.Path, b.config.Locales, b.config.Navigation.Families)
				if err != nil {
					return err
				}
				if resolved.ID == family.ID {
					found = true
					break
				}
			}
			if !found {
				return presentationSourceDiagnostic(newPresentationDiagnostic(
					"site.family_overview_locale_incomplete",
					fmt.Sprintf("family %q has pages in locale %q but no overview for that locale", family.ID, locale),
					"Add the configured overview route under the locale directory.",
					pointer,
				), family.Overview)
			}
		}
	}
	return nil
}

func routeKey(source string, locales LocaleConfig) string {
	_, rest := sourceLocale(source, locales)
	return outputPath(rest)
}

func pageRouteLess(left, right Page) bool {
	leftDirectory, rightDirectory := path.Dir(left.Output), path.Dir(right.Output)
	if leftDirectory != rightDirectory {
		return leftDirectory < rightDirectory
	}
	leftBase, rightBase := path.Base(left.Output), path.Base(right.Output)
	if leftBase == "index.html" && rightBase != "index.html" {
		return true
	}
	if rightBase == "index.html" && leftBase != "index.html" {
		return false
	}
	return left.Output < right.Output
}

func routeDescription(plainText, title, siteName string) string {
	value := strings.Join(strings.Fields(plainText), " ")
	value = strings.TrimSpace(strings.TrimPrefix(value, title))
	if index := strings.IndexAny(value, ".!?\n"); index >= 0 {
		value = value[:index+1]
	}
	if value == "" {
		value = fmt.Sprintf("%s documentation: %s.", siteName, title)
	}
	if len([]rune(value)) > 160 {
		value = string([]rune(value)[:157]) + "..."
	}
	return value
}

func (b *builder) pageURL(page Page) string {
	home := b.config.Site.Home
	if home == "" {
		home = "index.md"
	}
	route := "/" + page.Output
	if b.profileMode {
		route = b.publicOutputPath(page.Output, page.Source == home && page.Locale == b.config.Locales.Default)
	} else if page.Source == home && page.Locale == b.config.Locales.Default {
		route = "/"
	}
	if !b.profileMode {
		basePath := normalizedBasePath(b.config.BasePath)
		if basePath != "/" {
			route = strings.TrimSuffix(basePath, "/") + route
		}
	}
	return strings.TrimSuffix(b.config.Site.BaseURL, "/") + route
}

func (b *builder) stageConfiguredAssets(config Config) error {
	if err := b.stageConfigAsset(config.Site.Logo, "site.logo", "image/svg+xml"); err != nil {
		return err
	}
	if err := b.stageConfigAsset(config.Site.Icon, "site.icon", ""); err != nil {
		return err
	}
	if err := b.stageSocialImage(config.Site.SocialImage); err != nil {
		return err
	}
	if err := b.addArtifact("margo-assets/site.css", []byte(configuredSiteStylesheet(b.profileMode))); err != nil {
		return err
	}
	b.dependencies["margo-assets/site.css"] = "margo-assets/site.css"
	if err := b.addArtifact(pageActionsScriptPath, []byte(pageActionsScript)); err != nil {
		return err
	}
	b.dependencies[pageActionsScriptPath] = pageActionsScriptPath
	if b.profileMode {
		if err := b.addArtifact(profileInteractionScriptPath, []byte(profileInteractionScript)); err != nil {
			return err
		}
		b.dependencies[profileInteractionScriptPath] = profileInteractionScriptPath
	}
	for _, theme := range config.Themes {
		if strings.HasPrefix(theme.CSSURL, "https://") {
			return diagnostic("site.remote_resource", "remote theme CSS is not vendorized by this offline builder", "Copy the stylesheet into the site source tree.", theme.CSSURL)
		}
		if err := b.stageConfigAsset(theme.CSSURL, "theme.css", "text/css"); err != nil {
			return err
		}
		if err := b.stageConfigAsset(theme.TokenCatalog, "theme.token_catalog", "application/yaml"); err != nil {
			return err
		}
		if err := b.validateThemeCatalog(theme); err != nil {
			return err
		}
	}
	for _, css := range config.CustomCSS {
		if strings.HasPrefix(css.CSSURL, "https://") {
			return diagnostic("site.remote_resource", "remote custom CSS is not vendorized by this offline builder", "Copy the stylesheet into the site source tree.", css.CSSURL)
		}
		if err := b.stageConfigAsset(css.CSSURL, "custom_css", "text/css"); err != nil {
			return err
		}
	}
	if b.shellMode {
		if err := b.stageGoshtosoComponentDocShellAssets(); err != nil {
			return err
		}
	} else if b.profileMode {
		if err := b.stageGoshtosoNavigationAssets(); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) validateThemeCatalog(theme ThemeConfig) error {
	css, cssOK := b.assets[theme.CSSURL]
	catalog, catalogOK := b.assets[theme.TokenCatalog]
	if !cssOK || !catalogOK {
		return diagnostic("site.theme_catalog_invalid", fmt.Sprintf("theme %q assets were not staged", theme.Name), "Declare readable local CSS and token catalog assets.", theme.Name)
	}
	var document map[string]any
	if err := yaml.Unmarshal(catalog.content, &document); err != nil {
		return diagnostic("site.theme_catalog_invalid", fmt.Sprintf("theme %q token catalog is not valid YAML: %v", theme.Name, err), "Use the margo.theme.tokens/v1 catalog shape.", theme.TokenCatalog)
	}
	if document["schema"] != "margo.theme.tokens/v1" {
		return diagnostic("site.theme_catalog_invalid", fmt.Sprintf("theme %q has unsupported token catalog schema", theme.Name), "Use schema: margo.theme.tokens/v1.", theme.TokenCatalog)
	}
	cssDigest := strings.TrimPrefix(strings.TrimPrefix(fmt.Sprint(document["css_digest"]), "sha256-"), "sha256:")
	cssHash := sha256.Sum256(css.content)
	if cssDigest != hex.EncodeToString(cssHash[:]) {
		return diagnostic("site.theme_css_digest", fmt.Sprintf("theme %q css_digest does not match its CSS asset", theme.Name), "Recompute the catalog digest from the vendored stylesheet.", theme.TokenCatalog)
	}
	for _, key := range []string{"fonts", "minimum_text_size", "touch_target", "layout", "spacing", "breakpoints", "grid", "drawer", "colors", "states", "feedback", "contrast_pairs"} {
		if _, exists := document[key]; !exists {
			return diagnostic("site.theme_catalog_invalid", fmt.Sprintf("theme %q token catalog omits %s", theme.Name, key), "Declare every required semantic theme section.", theme.TokenCatalog)
		}
	}
	typography, ok := document["typography"].(map[string]any)
	if !ok {
		return diagnostic("site.theme_catalog_invalid", fmt.Sprintf("theme %q typography section is missing", theme.Name), "Declare display, headline, title, body, label, and caption roles.", theme.TokenCatalog)
	}
	for _, role := range []string{"display", "headline", "title", "body", "label", "caption"} {
		if _, exists := typography[role]; !exists {
			return diagnostic("site.theme_catalog_invalid", fmt.Sprintf("theme %q typography omits %s", theme.Name, role), "Declare every required typography role.", theme.TokenCatalog)
		}
	}
	colors, colorsOK := document["colors"].(map[string]any)
	states, statesOK := document["states"].(map[string]any)
	feedback, feedbackOK := document["feedback"].(map[string]any)
	if !colorsOK || !statesOK || !feedbackOK {
		return diagnostic("site.theme_catalog_invalid", fmt.Sprintf("theme %q state/color sections are malformed", theme.Name), "Declare light and dark semantic state catalogs.", theme.TokenCatalog)
	}
	for _, mode := range []string{"light", "dark"} {
		for _, section := range []struct {
			name string
			data map[string]any
		}{{name: "colors", data: colors}, {name: "states", data: states}, {name: "feedback", data: feedback}} {
			if _, exists := section.data[mode]; !exists {
				return diagnostic("site.theme_catalog_invalid", fmt.Sprintf("theme %q %s omits %s mode", theme.Name, section.name, mode), "Declare both light and dark semantic values.", theme.TokenCatalog)
			}
		}
	}
	return nil
}

func (b *builder) stageConfigAsset(name, subject, expectedMedia string) error {
	data, err := os.ReadFile(filepath.Join(b.configDir, filepath.FromSlash(name)))
	if err != nil {
		return diagnostic("site.asset_unreadable", fmt.Sprintf("cannot read %s %q: %v", subject, name, err), "Add the declared asset beside site.yaml.", name)
	}
	mediaType := expectedMedia
	if expectedMedia != "text/css" && expectedMedia != "application/yaml" {
		mediaType, err = staticimage.DetectContext(context.Background(), data)
	}
	if err != nil {
		return diagnostic("site.asset_invalid", fmt.Sprintf("%s %q: %v", subject, name, err), "Use a valid local asset.", name)
	}
	if expectedMedia != "" && mediaType != expectedMedia {
		return diagnostic("site.asset_invalid", fmt.Sprintf("%s %q has media type %q, want %q", subject, name, mediaType, expectedMedia), "Use the required asset type.", name)
	}
	if err := b.addArtifact(name, data); err != nil {
		return err
	}
	b.assets[name] = cachedAsset{content: append([]byte(nil), data...), mediaType: mediaType}
	b.dependencies[strings.ToLower(strings.TrimPrefix(name, "/"))] = strings.TrimPrefix(name, "/")
	return nil
}

func (b *builder) stageSocialImage(config SocialImageConfig) error {
	data, err := os.ReadFile(filepath.Join(b.configDir, filepath.FromSlash(config.Path)))
	if err != nil {
		return diagnostic("site.social_image_invalid", fmt.Sprintf("cannot read social image: %v", err), "Add the configured preview image.", config.Path)
	}
	if len(data) == 0 || len(data) > 1<<20 {
		return diagnostic("site.social_image_invalid", "social preview image must be non-empty and below 1 MiB", "Use a 1280x640 JPEG or PNG.", config.Path)
	}
	mediaType, err := staticimage.DetectContext(context.Background(), data)
	if err != nil || mediaType != "image/jpeg" && mediaType != "image/png" {
		return diagnostic("site.social_image_invalid", "social preview image must be JPEG or PNG", "Use a safe 1280x640 JPEG or PNG.", config.Path)
	}
	b.socialMediaType = mediaType
	decoded, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || decoded.Width != 1280 || decoded.Height != 640 || format != "jpeg" && format != "png" {
		return diagnostic("site.social_image_invalid", "social preview image must be exactly 1280x640 JPEG or PNG", "Create an intentional landscape preview image.", config.Path)
	}
	return b.addArtifact(config.Path, data)
}

func (b *builder) renderConfiguredSource(ctx context.Context, source Source) error {
	prepared, exists := b.configured[source.Path]
	if !exists {
		return fmt.Errorf("site.page_missing: preflight page %q is unavailable", source.Path)
	}
	dependencyBytes, err := b.configuredDependencyBytes(prepared)
	if err != nil {
		return err
	}
	if b.shellMode {
		return b.renderConfiguredShellSource(ctx, source, prepared, dependencyBytes)
	}
	bindings, err := b.bindingsForPage(prepared)
	if err != nil {
		return err
	}
	presentation, err := b.presentationForPage(prepared)
	if err != nil {
		return err
	}
	output, err := presentation.Frame.Render(ssg.FrameInput{SchemaHash: presentation.SchemaHash, RootCompositionHash: presentation.SchemaHash, InstanceID: "root", Values: presentation.Values, Bindings: bindings})
	if err != nil {
		return err
	}
	fragment, err := renderComponentBytes(output.Fragment)
	if err != nil {
		return err
	}
	fragment = addProfileLayoutHook(fragment, prepared.page.Layout)
	page := prepared.page
	iconURL, _ := relativeSitePath(path.Dir(page.Output), b.config.Site.Icon)
	head, err := b.renderPageHead(page, iconURL, dependencyBytes, prepared.result.Requirements())
	if err != nil {
		return err
	}
	body := `<a class="margo-skip-link" href="#margo-document">Skip to content</a>` + string(fragment)
	markup := []byte(`<!doctype html><html lang="` + stdhtml.EscapeString(page.Locale) + `" dir="` + localeDirection(page.Locale) + `" data-theme="` + stdhtml.EscapeString(b.config.Theme.Name) + `" data-color-mode="` + stdhtml.EscapeString(b.config.Theme.ColorMode) + `"><head>` + head + `</head><body>` + body + `</body></html>`)
	rewritten, err := b.rewriteHTML(ctx, source, markup)
	if err != nil {
		return err
	}
	rewritten, err = b.injectPageActions(ctx, rewritten, prepared.page)
	if err != nil {
		return err
	}
	if err := b.addDeclaredPageArtifacts(ctx, source, prepared.page, prepared.document); err != nil {
		return err
	}
	if err := validateConfiguredDocument(rewritten, prepared.page, bindings); err != nil {
		return err
	}
	if err := b.addArtifact(page.Output, rewritten); err != nil {
		return err
	}
	b.pages = append(b.pages, page)
	return nil
}

func (b *builder) presentationForPage(prepared configuredPage) (PagePresentation, error) {
	if b.profileMode {
		if prepared.presentation.Frame == nil {
			return PagePresentation{}, fmt.Errorf("site.layout_missing: page %q has no prepared layout presentation", prepared.page.Source)
		}
		return prepared.presentation, nil
	}
	if b.frame == nil {
		return PagePresentation{}, fmt.Errorf("site.layout_missing: page %q has no configured frame", prepared.page.Source)
	}
	return PagePresentation{FrameName: b.layoutName, Frame: b.frame, Schema: b.frameSchema, Values: b.frameValues, SchemaHash: b.frameHash}, nil
}

func (b *builder) configuredDependencyBytes(prepared configuredPage) ([]byte, error) {
	mode := margo.HTMLDependenciesLocal
	if b.request.Assets == AssetsInline {
		mode = margo.HTMLDependenciesInline
	}
	if err := b.stageChartIconSprite(prepared.result.Requirements()); err != nil {
		return nil, err
	}
	requirements, err := margo.RenderHTMLDependencies(prepared.result.Requirements(), mode)
	if err != nil {
		return nil, err
	}
	if b.request.Assets == AssetsLocal {
		for _, requirement := range prepared.result.Requirements().List() {
			assetPath := strings.TrimPrefix(requirement.LocalURL, "/")
			if assetPath == "" || len(requirement.Inline.Content) == 0 {
				continue
			}
			if err := b.addArtifact(assetPath, requirement.Inline.Content); err != nil {
				return nil, err
			}
			b.dependencies[strings.ToLower(assetPath)] = assetPath
		}
	}
	return renderComponentBytes(requirements)
}

func (b *builder) stageChartIconSprite(requirements margo.HTMLRequirements) error {
	if b.request.Assets != AssetsLocal {
		return nil
	}
	for _, requirement := range requirements.List() {
		if requirement.ID != "goshtoso-charts.controls" {
			continue
		}
		return b.stageHandlerAsset(chartassets.Handler(), chartassets.ChartIconsSpriteURL)
	}
	return nil
}

func (b *builder) bindingsForPage(prepared configuredPage) (map[string][]ssg.AreaBinding, error) {
	presentation, err := b.presentationForPage(prepared)
	if err != nil {
		return nil, err
	}
	schema := presentation.Schema
	schemaHash := presentation.SchemaHash
	bindings := map[string][]ssg.AreaBinding{}
	add := func(kind, defaultArea, slot, fragment string) error {
		configuration, configured := b.config.Bindings[kind]
		area := defaultArea
		if configured {
			area = configuration.Area
			if configuration.Slot != "" {
				slot = configuration.Slot
			}
		}
		if area == "" {
			return nil
		}
		binding, err := ssg.NewAreaBinding(schemaHash, prepared.page.Output, ssg.BindingSpec{Kind: kind, Area: area, Slot: slot}, len(bindings[area]), templ.Raw(fragment))
		if err != nil {
			return err
		}
		bindings[area] = append(bindings[area], binding)
		return nil
	}
	documentFragment := string(prepared.article)
	if b.profileMode {
		// Profile pages share one public article hook. Landing owns a wider
		// canvas while docs retain their reading measure through profile CSS.
		documentFragment = `<div class="margo-showcase-article" data-margo-showcase-article="true">` + documentFragment + `</div>`
	}
	if err := add("document", schema.BindingDefaults["document"], "", documentFragment); err != nil {
		return nil, err
	}
	if b.profileMode {
		siteNavigation, siteNavigationErr := b.siteNavigationFragment(prepared.page)
		if siteNavigationErr != nil {
			return nil, siteNavigationErr
		}
		if err := add("site_navigation", "top-nav", "", siteNavigation); err != nil {
			return nil, err
		}
		profileDocs := false
		for _, area := range schema.Areas {
			if area.ID == "left-nav" {
				profileDocs = true
				break
			}
		}
		if profileDocs {
			familyNavigation, familyNavigationErr := b.familyNavigationFragment(prepared.page)
			if familyNavigationErr != nil {
				return nil, familyNavigationErr
			}
			if err := add("navigation", "left-nav", "", familyNavigation); err != nil {
				return nil, err
			}
			if err := add("toc", schema.BindingDefaults["toc"], "", b.tocFragment(prepared.article, prepared.page.Locale)); err != nil {
				return nil, err
			}
			pagination := b.paginationFragment(prepared.page)
			if pagination != "" {
				if err := add("pagination", schema.BindingDefaults["pagination"], "after-article", pagination); err != nil {
					return nil, err
				}
			}
		}
	} else {
		if err := add("navigation", schema.BindingDefaults["navigation"], "", b.navigationFragment(prepared.page)); err != nil {
			return nil, err
		}
		if err := add("breadcrumbs", schema.BindingDefaults["breadcrumbs"], "", b.breadcrumbFragment(prepared.page)); err != nil {
			return nil, err
		}
		if err := add("pagination", schema.BindingDefaults["pagination"], "after-article", b.paginationFragment(prepared.page)); err != nil {
			return nil, err
		}
	}
	if b.config.Theme.AllowSwitchTheme {
		label := localizedLabel(prepared.page.Locale, "theme")
		if err := add("theme_controls", schema.BindingDefaults["theme_controls"], "", `<div class="margo-theme-controls"><button type="button" aria-label="`+stdhtml.EscapeString(label)+`" data-margo-theme-control="cycle">`+stdhtml.EscapeString(label)+`</button></div>`); err != nil {
			return nil, err
		}
	}
	if len(b.config.Locales.Supported) > 1 {
		if err := add("locale_controls", schema.BindingDefaults["locale_controls"], "", b.localeFragment(prepared.page)); err != nil {
			return nil, err
		}
	}
	if err := add("footer", schema.BindingDefaults["footer"], "", `<footer class="margo-footer"><p>`+stdhtml.EscapeString(b.config.Site.Name)+`</p></footer>`); err != nil {
		return nil, err
	}
	if err := ssg.ValidateBindings(schema, bindings); err != nil {
		return nil, err
	}
	return bindings, nil
}

func (b *builder) navigationFragment(page Page) string {
	var builder strings.Builder
	label := localizedLabel(page.Locale, "contents")
	builder.WriteString(`<nav aria-label="` + stdhtml.EscapeString(label) + `"><a class="margo-site-brand" href="`)
	rootURL := relativeAssetPath(path.Dir(page.Output), b.localeHomeOutput(page))
	builder.WriteString(stdhtml.EscapeString(rootURL) + `"><img src="` + stdhtml.EscapeString(relativeAssetPath(path.Dir(page.Output), b.config.Site.Logo)) + `" alt="` + stdhtml.EscapeString(b.config.Site.Name) + `"> <span>` + stdhtml.EscapeString(b.config.Site.Name) + `</span></a><ul>`)
	for _, candidate := range b.configPages {
		if candidate.Locale != page.Locale {
			continue
		}
		builder.WriteString(`<li><a href="` + stdhtml.EscapeString(relativeAssetPath(path.Dir(page.Output), candidate.Output)) + `"`)
		if candidate.Source == page.Source {
			builder.WriteString(` aria-current="page"`)
		}
		builder.WriteString(`>` + stdhtml.EscapeString(candidate.Title) + `</a></li>`)
	}
	builder.WriteString(`</ul></nav>`)
	return builder.String()
}

func (b *builder) breadcrumbFragment(page Page) string {
	if page.Output == b.localeHomeOutput(page) {
		return ""
	}
	_, route := sourceLocale(page.Source, b.config.Locales)
	parts := strings.Split(strings.TrimSuffix(outputPath(route), path.Ext(outputPath(route))), "/")
	current := ""
	for index, part := range parts {
		if part == "index" && index == len(parts)-1 {
			continue
		}
		if index == len(parts)-1 {
			current = page.Title
		}
	}
	homeHref := relativeAssetPath(path.Dir(page.Output), b.localeHomeOutput(page))
	if b.profileMode {
		homeHref = b.siteHomeHref(page)
	}
	markup, err := renderComponentBytes(breadcrumbs.Breadcrumbs(breadcrumbs.Config{
		Items: []breadcrumbs.Item{{
			Label: localizedLabel(page.Locale, "home"),
			Href:  homeHref,
		}},
		Current:   current,
		Separator: breadcrumbs.Chevron,
		NavClass:  "margo-breadcrumbs",
	}))
	if err != nil {
		return ""
	}
	return strings.Replace(string(markup), `aria-label="breadcrumb"`, `aria-label="`+stdhtml.EscapeString(localizedLabel(page.Locale, "breadcrumbs"))+`"`, 1)
}

func (b *builder) localeHomeOutput(page Page) string {
	if page.Locale == b.config.Locales.Default {
		return configuredOutputPath(b.config.Site.Home, b.config.Locales)
	}
	homeRoute := routeKey(b.config.Site.Home, b.config.Locales)
	for _, candidate := range b.configPages {
		if candidate.Locale == page.Locale && routeKey(candidate.Source, b.config.Locales) == homeRoute {
			return candidate.Output
		}
	}
	return configuredOutputPath(b.config.Site.Home, b.config.Locales)
}

func (b *builder) paginationFragment(page Page) string {
	localePages := make([]Page, 0, len(b.configPages))
	familyScoped := b.profileMode || (b.config.Layout != nil && page.Layout == string(LayoutDocs))
	if familyScoped {
		localePages = b.familyPages(page)
	} else {
		for _, candidate := range b.configPages {
			if candidate.Locale == page.Locale {
				localePages = append(localePages, candidate)
			}
		}
	}
	index := -1
	for i, candidate := range localePages {
		if candidate.Source == page.Source {
			index = i
			break
		}
	}
	if familyScoped && (index < 0 || len(localePages) < 2) {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(`<nav class="margo-pagination" aria-label="` + stdhtml.EscapeString(localizedLabel(page.Locale, "article_navigation")) + `"><ul>`)
	if index > 0 {
		previous := localePages[index-1]
		href := relativeAssetPath(path.Dir(page.Output), previous.Output)
		if familyScoped {
			href = b.sitePageHref(previous)
		}
		builder.WriteString(`<li><a rel="prev" href="` + stdhtml.EscapeString(href) + `">Previous: ` + stdhtml.EscapeString(previous.Title) + `</a></li>`)
	}
	if index >= 0 && index+1 < len(localePages) {
		next := localePages[index+1]
		href := relativeAssetPath(path.Dir(page.Output), next.Output)
		if familyScoped {
			href = b.sitePageHref(next)
		}
		builder.WriteString(`<li><a rel="next" href="` + stdhtml.EscapeString(href) + `">Next: ` + stdhtml.EscapeString(next.Title) + `</a></li>`)
	}
	builder.WriteString(`</ul></nav>`)
	return builder.String()
}

func (b *builder) localeFragment(page Page) string {
	var builder strings.Builder
	builder.WriteString(`<nav class="margo-locale-controls" aria-label="` + stdhtml.EscapeString(localizedLabel(page.Locale, "language")) + `"><ul>`)
	for _, alternate := range page.Alternates {
		href := b.relativeAlternate(page.Output, alternate.URL)
		if b.profileMode {
			href = b.publicAlternatePath(alternate.URL)
		}
		builder.WriteString(`<li><a hreflang="` + stdhtml.EscapeString(alternate.Locale) + `" href="` + stdhtml.EscapeString(href) + `">` + stdhtml.EscapeString(alternate.Locale) + `</a></li>`)
	}
	builder.WriteString(`</ul></nav>`)
	return builder.String()
}

func (b *builder) renderPageHead(page Page, iconURL string, dependencyBytes []byte, requirements margo.HTMLRequirements) (string, error) {
	var builder strings.Builder
	builder.WriteString(`<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><link rel="icon" href="` + stdhtml.EscapeString(iconURL) + `"><title>` + stdhtml.EscapeString(page.Title) + `</title><meta name="description" content="` + stdhtml.EscapeString(page.Description) + `"><link rel="canonical" href="` + stdhtml.EscapeString(page.Canonical) + `">`)
	builder.WriteString(`<meta property="og:url" content="` + stdhtml.EscapeString(page.Canonical) + `"><meta property="og:type" content="website"><meta property="og:title" content="` + stdhtml.EscapeString(page.Title) + `"><meta property="og:description" content="` + stdhtml.EscapeString(page.Description) + `"><meta property="og:site_name" content="` + stdhtml.EscapeString(b.config.Site.Name) + `"><meta property="og:image" content="` + stdhtml.EscapeString(b.socialURL()) + `"><meta property="og:image:type" content="` + stdhtml.EscapeString(b.socialMediaType) + `"><meta property="og:image:width" content="1280"><meta property="og:image:height" content="640"><meta property="og:image:alt" content="` + stdhtml.EscapeString(b.config.Site.SocialImage.Alt) + `"><meta property="og:locale" content="` + stdhtml.EscapeString(openGraphLocale(page.Locale)) + `">`)
	builder.WriteString(`<meta name="twitter:card" content="summary_large_image"><meta name="twitter:title" content="` + stdhtml.EscapeString(page.Title) + `"><meta name="twitter:description" content="` + stdhtml.EscapeString(page.Description) + `"><meta name="twitter:image" content="` + stdhtml.EscapeString(b.socialURL()) + `"><meta name="twitter:image:alt" content="` + stdhtml.EscapeString(b.config.Site.SocialImage.Alt) + `">`)
	for _, alternate := range page.Alternates {
		builder.WriteString(`<link rel="alternate" hreflang="` + stdhtml.EscapeString(alternate.Locale) + `" href="` + stdhtml.EscapeString(alternate.URL) + `">`)
	}
	if len(page.Alternates) > 0 {
		for _, alternate := range page.Alternates {
			builder.WriteString(`<meta property="og:locale:alternate" content="` + stdhtml.EscapeString(openGraphLocale(alternate.Locale)) + `">`)
		}
	}
	builder.WriteString(b.themeBootstrap())
	builder.WriteString(`<link rel="stylesheet" href="/margo-assets/site.css">`)
	builder.WriteString(`<script defer src="/` + stdhtml.EscapeString(pageActionsScriptPath) + `"></script>`)
	if b.profileMode {
		builder.WriteString(`<script defer src="/` + stdhtml.EscapeString(profileInteractionScriptPath) + `"></script>`)
	}
	for _, theme := range b.config.Themes {
		media := "not all"
		if theme.Name == b.config.Theme.Name {
			media = "all"
		}
		builder.WriteString(`<link rel="stylesheet" data-margo-theme-css="` + stdhtml.EscapeString(theme.Name) + `" media="` + media + `" href="/` + stdhtml.EscapeString(strings.TrimPrefix(theme.CSSURL, "/")) + `">`)
	}
	for _, css := range b.config.CustomCSS {
		builder.WriteString(`<link rel="stylesheet" href="/` + stdhtml.EscapeString(strings.TrimPrefix(css.CSSURL, "/")) + `">`)
	}
	builder.WriteString(string(dependencyBytes))
	goshtosoDependencies, err := b.configuredGoshtosoDependencyBytes()
	if err != nil {
		return "", err
	}
	goshtosoDependencies = withoutGoshtosoStylesheet(goshtosoDependencies, requirements)
	builder.WriteString(string(goshtosoDependencies))
	return builder.String(), nil
}

func (b *builder) themeBootstrap() string {
	themes := []string{"modern"}
	for _, theme := range b.config.Themes {
		themes = append(themes, theme.Name)
	}
	configuration, err := json.Marshal(struct {
		Theme       string   `json:"theme"`
		ColorMode   string   `json:"colorMode"`
		AllowSwitch bool     `json:"allowSwitch"`
		Available   []string `json:"available"`
	}{Theme: b.config.Theme.Name, ColorMode: b.config.Theme.ColorMode, AllowSwitch: b.config.Theme.AllowSwitchTheme, Available: themes})
	if err != nil {
		return ""
	}
	return `<script data-margo-theme-bootstrap>(function(c){var r=document.documentElement;function ok(t){return c.available.indexOf(t)!==-1}function mode(){if(c.colorMode!=="system")return c.colorMode;try{return window.matchMedia&&window.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light"}catch(_){return "light"}}function apply(t){r.dataset.theme=t;r.dataset.colorMode=mode();document.querySelectorAll("[data-margo-theme-css]").forEach(function(link){link.media=link.dataset.margoThemeCss===t?"all":"not all"})}var t=c.theme;if(c.allowSwitch){try{var stored=window.localStorage.getItem("margo.theme");if(ok(stored))t=stored}catch(_){}}apply(t);if(c.allowSwitch){document.addEventListener("click",function(event){var control=event.target.closest("[data-margo-theme-control]");if(!control)return;var index=c.available.indexOf(t);t=c.available[(index+1)%c.available.length];apply(t);try{window.localStorage.setItem("margo.theme",t)}catch(_){} })}if(c.colorMode==="system"&&window.matchMedia){try{window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change",function(){apply(t)})}catch(_){}}})(` + string(configuration) + `);</script>`
}

func (b *builder) socialURL() string {
	basePath := normalizedBasePath(b.config.BasePath)
	if basePath == "/" {
		basePath = ""
	}
	return strings.TrimSuffix(b.config.Site.BaseURL, "/") + basePath + "/" + strings.TrimPrefix(b.config.Site.SocialImage.Path, "/")
}

func validateConfiguredDocument(document []byte, page Page, bindings map[string][]ssg.AreaBinding) error {
	root, err := html.Parse(bytes.NewReader(document))
	if err != nil {
		return diagnostic("site.html_invalid", err.Error(), "Report the generated document defect.", page.Source)
	}
	counts := map[string]int{}
	var main *html.Node
	var skip int
	var markerStarts, markerEnds = map[string]int{}, map[string]int{}
	var walk func(*html.Node, bool)
	walk = func(node *html.Node, inMain bool) {
		if node.Type == html.ElementNode {
			counts[node.Data]++
			if node.Data == "main" {
				main = node
			}
			if node.Data == "a" && attributeValue(node, "href") == "#margo-document" {
				skip++
			}
			inMain = inMain || node.Data == "main"
			if node.Data == "h1" && !inMain {
				return
			}
		}
		if node.Type == html.CommentNode {
			parts := strings.Fields(node.Data)
			if len(parts) == 2 && parts[0] == "margo.ssg.area-payload:start" {
				markerStarts[parts[1]]++
			}
			if len(parts) == 2 && parts[0] == "margo.ssg.area-payload:end" {
				markerEnds[parts[1]]++
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child, inMain)
		}
	}
	walk(root, false)
	if counts["html"] != 1 || counts["head"] != 1 || counts["body"] != 1 || counts["main"] != 1 || counts["h1"] != 1 || skip != 1 || main == nil {
		return diagnostic("site.semantic_structure", fmt.Sprintf("route %s must contain one html, head, body, main, h1, and skip link", page.Output), "Keep the document binding inside the single Margo main host.", page.Source)
	}
	for area, areaBindings := range bindings {
		for _, binding := range areaBindings {
			if markerStarts[binding.Token] != 1 || markerEnds[binding.Token] != 1 {
				return diagnostic("site.binding_marker", fmt.Sprintf("binding %s in %s was not preserved exactly once", binding.Kind, area), "Keep paired Margo binding markers around every semantic fragment.", page.Source)
			}
		}
	}
	if err := validateRequiredHead(root, page); err != nil {
		return err
	}
	return nil
}

func validateRequiredHead(root *html.Node, page Page) error {
	head := firstElement(root, "head")
	if head == nil {
		return diagnostic("site.head_missing", "generated page has no head", "Render route metadata in initial HTML.", page.Source)
	}
	counts := map[string]int{}
	for child := head.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode {
			continue
		}
		switch child.Data {
		case "title":
			counts["title"]++
		case "link":
			if attributeValue(child, "rel") == "canonical" {
				counts["canonical"]++
			}
		case "meta":
			key := attributeValue(child, "name")
			if key == "" {
				key = attributeValue(child, "property")
			}
			counts[key]++
		}
	}
	for _, key := range []string{"title", "canonical", "description", "og:url", "og:type", "og:title", "og:description", "og:site_name", "og:image", "og:image:type", "og:image:width", "og:image:height", "og:image:alt", "twitter:card", "twitter:title", "twitter:description", "twitter:image", "twitter:image:alt"} {
		if counts[key] != 1 {
			return diagnostic("site.metadata_invalid", fmt.Sprintf("route %s must emit exactly one %s tag", page.Output, key), "Keep route metadata in the shared generated head.", page.Source)
		}
	}
	return nil
}

func countElements(fragment []byte, name string) int {
	root, err := html.ParseFragment(bytes.NewReader(fragment), &html.Node{Type: html.ElementNode, DataAtom: atom.Div, Data: "div"})
	if err != nil {
		return 0
	}
	count := 0
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == name {
			count++
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	for _, node := range root {
		walk(node)
	}
	return count
}

func firstElement(root *html.Node, name string) *html.Node {
	if root.Type == html.ElementNode && root.Data == name {
		return root
	}
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if found := firstElement(child, name); found != nil {
			return found
		}
	}
	return nil
}

func attributeValue(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func relativeAssetPath(from, target string) string {
	value, err := relativeSitePath(from, target)
	if err != nil || value == "" {
		return path.Base(target)
	}
	return value
}

func (b *builder) relativeAlternate(fromOutput, absolute string) string {
	parsed, err := url.Parse(absolute)
	if err != nil || parsed.Path == "" {
		return absolute
	}
	route := parsed.Path
	basePath := normalizedBasePath(b.config.BasePath)
	if basePath != "/" {
		if route == basePath {
			route = "/"
		} else if strings.HasPrefix(route, basePath+"/") {
			route = strings.TrimPrefix(route, basePath)
		} else {
			return absolute
		}
	}
	target := strings.TrimPrefix(route, "/")
	if target == "" {
		target = "index.html"
	}
	return relativeAssetPath(path.Dir(fromOutput), target)
}

func (b *builder) publicAlternatePath(absolute string) string {
	parsed, err := url.Parse(absolute)
	if err != nil || parsed.Path == "" {
		return absolute
	}
	return parsed.Path
}

func localizedLabel(locale, key string) string {
	if strings.EqualFold(locale, "pt-BR") {
		return map[string]string{"contents": "Conteúdo", "breadcrumbs": "Navegação estrutural", "article_navigation": "Navegação do artigo", "home": "Início", "language": "Idioma", "theme": "Tema", "toc": "Nesta página"}[key]
	}
	return map[string]string{"contents": "Contents", "breadcrumbs": "Breadcrumbs", "article_navigation": "Article navigation", "home": "Home", "language": "Language", "theme": "Theme", "toc": "On this page"}[key]
}

func openGraphLocale(locale string) string {
	return strings.ReplaceAll(locale, "-", "_")
}

func renderComponentBytes(component templ.Component) ([]byte, error) {
	var buffer bytes.Buffer
	if err := component.Render(context.Background(), &buffer); err != nil {
		return nil, err
	}
	return append([]byte(nil), buffer.Bytes()...), nil
}

func documentPayloadDigest(payload []byte) string {
	hash := sha256.New()
	_, _ = hash.Write([]byte("margo.ssg.document-payload/v1\x00"))
	_, _ = hash.Write(payload)
	return hex.EncodeToString(hash.Sum(nil))
}

func (b *builder) documentStyleDigest() string {
	hash := sha256.New()
	_, _ = hash.Write([]byte("margo.ssg.document-style/v1\x00"))
	layoutName := b.layoutName
	frameHash := b.frameHash
	if b.profileMode {
		layoutName = b.siteManifest.Layout
		frameHash = b.siteManifest.LayoutSchemaHash
	}
	_, _ = hash.Write([]byte(layoutName))
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(frameHash))
	_, _ = hash.Write([]byte(b.config.Theme.Name))
	_, _ = hash.Write([]byte(b.config.Theme.ColorMode))
	styles := map[string][]byte{"margo-assets/site.css": []byte(configuredSiteStylesheet(b.profileMode))}
	for _, theme := range b.config.Themes {
		if theme.Name == b.config.Theme.Name {
			if asset, ok := b.assets[theme.CSSURL]; ok {
				styles[theme.CSSURL] = asset.content
			}
			if asset, ok := b.assets[theme.TokenCatalog]; ok {
				styles[theme.TokenCatalog] = asset.content
			}
		}
	}
	for _, css := range b.config.CustomCSS {
		if asset, ok := b.assets[css.CSSURL]; ok {
			styles[css.CSSURL] = asset.content
		}
	}
	for name, content := range b.artifacts {
		if strings.HasSuffix(strings.ToLower(name), ".css") {
			styles[name] = content
		}
	}
	styleNames := make([]string, 0, len(styles))
	for name := range styles {
		styleNames = append(styleNames, name)
	}
	sort.Strings(styleNames)
	for _, name := range styleNames {
		_, _ = hash.Write([]byte(name))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write(styles[name])
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
