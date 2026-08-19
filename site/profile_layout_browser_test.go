package site

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/araihu/margo/internal/browserlaunch"
	"github.com/chromedp/chromedp"
)

func TestProfileDocsFrameResponsiveComputedStyles(t *testing.T) {
	browserPath := installedSiteTestChromium()
	if browserPath == "" {
		t.Skip("installed Chromium-family browser unavailable")
	}

	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "---\nlayout:\n  kind: landing\n---\n# Tour\n\nTour documentation.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "index.md"), "# Module\n\nModule documentation.\n")
	writeConfigFile(t, filepath.Join(root, "docs", "module", "_layout.yaml"), "values:\n  family: module\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
assets: local
base_path: /docs
site:
  name: Margo
  description: Margo documentation
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo preview
layout:
  kind: docs
  default:
    families: [module]
  values:
    family: default
navigation:
  mode: file-tree
locales:
  default: en
  supported: [en]
theme:
  builtin: true
  name: modern
  color_mode: light
`)

	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	artifacts := make(map[string][]byte, len(result.Artifacts))
	for _, artifact := range result.Artifacts {
		artifacts["/"+strings.TrimPrefix(artifact.Path, "/")] = artifact.Content
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		artifactPath := request.URL.Path
		if strings.HasPrefix(artifactPath, "/docs") {
			artifactPath = strings.TrimPrefix(artifactPath, "/docs")
		}
		if artifactPath == "" || artifactPath == "/" {
			artifactPath = "/index.html"
		} else if strings.HasSuffix(artifactPath, "/") {
			artifactPath += "index.html"
		}
		content, ok := artifacts[artifactPath]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", siteBrowserContentType(artifactPath))
		_, _ = writer.Write(content)
	}))
	defer server.Close()

	allocatorContext, cancelAllocator := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(browserPath)...)
	defer cancelAllocator()
	browserContext, cancelBrowser := chromedp.NewContext(allocatorContext)
	defer cancelBrowser()
	ctx, cancel := context.WithTimeout(browserContext, 45*time.Second)
	defer cancel()

	type frameState struct {
		Display             string            `json:"display"`
		ColumnCount         int               `json:"columnCount"`
		GridAreas           string            `json:"gridAreas"`
		OverflowX           string            `json:"overflowX"`
		ClientWidth         float64           `json:"clientWidth"`
		ScrollWidth         float64           `json:"scrollWidth"`
		HasThreeColumnAreas bool              `json:"hasThreeColumnAreas"`
		HasStackedAreaRows  bool              `json:"hasStackedAreaRows"`
		AreaGridNames       map[string]string `json:"areaGridNames"`
	}
	for _, viewport := range []struct {
		width       int64
		name        string
		columns     int
		threeColumn bool
		twoColumn   bool
		stacked     bool
		overflow    string
	}{
		{width: 1440, name: "wide", columns: 3, threeColumn: true},
		{width: 720, name: "stacked-tablet", columns: 1, stacked: true, overflow: "clip"},
		{width: 900, name: "mid", columns: 3, threeColumn: true},
		{width: 390, name: "narrow", columns: 1, stacked: true, overflow: "clip"},
	} {
		var state frameState
		if err := chromedp.Run(ctx,
			chromedp.EmulateViewport(viewport.width, 844),
			chromedp.Navigate(server.URL+"/docs/module/"),
			chromedp.WaitVisible(`[data-margo-layout="docs"].margo-frame--top-left-main-right-footer`, chromedp.ByQuery),
			chromedp.Evaluate(`(() => {
				const frame = document.querySelector('[data-margo-layout="docs"].margo-frame--top-left-main-right-footer');
				const style = getComputedStyle(frame);
				const areaGridNames = Object.fromEntries([...document.querySelectorAll('[data-margo-area]')].map((area) => [area.dataset.margoArea, getComputedStyle(area).gridArea]));
				return {
					display: style.display,
					columnCount: style.gridTemplateColumns.trim().split(/\s+/).filter(Boolean).length,
					gridAreas: style.gridTemplateAreas,
					overflowX: style.overflowX,
					clientWidth: frame.clientWidth,
					scrollWidth: frame.scrollWidth,
					hasThreeColumnAreas: style.gridTemplateAreas.includes('left-nav main-content right-nav'),
					hasStackedAreaRows: style.gridTemplateAreas.includes('"left-nav"') && style.gridTemplateAreas.includes('"main-content"') && style.gridTemplateAreas.includes('"right-nav"'),
					areaGridNames,
				};
			})()`, &state),
		); err != nil {
			t.Fatalf("%s viewport browser check failed: %v", viewport.name, err)
		}
		if state.Display != "grid" || state.ColumnCount != viewport.columns {
			t.Fatalf("%s viewport frame geometry = %+v, want display grid with %d columns", viewport.name, state, viewport.columns)
		}
		hasTwoColumnAreas := state.ColumnCount == 2 && strings.Contains(state.GridAreas, `"left-nav main-content"`) && strings.Contains(state.GridAreas, `"right-nav right-nav"`)
		if viewport.threeColumn != state.HasThreeColumnAreas || viewport.twoColumn != hasTwoColumnAreas || viewport.stacked != state.HasStackedAreaRows {
			t.Fatalf("%s viewport grid areas = %+v", viewport.name, state)
		}
		for area, want := range map[string]string{"left-nav": "left-nav", "main-content": "main-content", "right-nav": "right-nav"} {
			if got := state.AreaGridNames[area]; got != want {
				t.Fatalf("%s viewport %s grid area = %q, want %q: %+v", viewport.name, area, got, want, state)
			}
		}
		if state.ScrollWidth > state.ClientWidth+1 {
			t.Fatalf("%s viewport overflows frame: %+v", viewport.name, state)
		}
		if viewport.overflow != "" && state.OverflowX != viewport.overflow {
			t.Fatalf("%s viewport overflow-x = %q, want %q: %+v", viewport.name, state.OverflowX, viewport.overflow, state)
		}
	}
}
