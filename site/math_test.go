package site

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/araihu/margo"
	"github.com/araihu/margo/internal/browserlaunch"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func TestMathSiteLocalInlineAndNestedRoutes(t *testing.T) {
	for _, mode := range []AssetMode{AssetsLocal, AssetsInline} {
		t.Run(string(mode), func(t *testing.T) {
			result, err := Build(context.Background(), Request{SourceRoot: t.TempDir(), Compiler: margo.New(margo.WithMath(true)), Assets: mode, Sources: []Source{
				{Path: "index.md", Content: []byte("# Home\n\n$x$\n")},
				{Path: "nested/guide.md", Content: []byte("# Guide\n\n$x$\n\n$$\\frac{1}{2}$$\n")},
				{Path: "plain.md", Content: []byte("# Plain\n\nCost $20 and $30.\n")},
			}})
			if err != nil {
				t.Fatal(err)
			}
			files := map[string][]byte{}
			engineFiles := 0
			for _, artifact := range result.Artifacts {
				files[artifact.Path] = artifact.Content
				if strings.HasSuffix(artifact.Path, "katex.min.js") {
					engineFiles++
				}
			}
			if mode == AssetsLocal && engineFiles != 1 {
				t.Fatalf("engine artifacts: %d", engineFiles)
			}
			if strings.Contains(string(files["plain.html"]), `data-margo-requirement="margo.math.engine"`) {
				t.Fatal("plain page includes math engine")
			}
			path := installedSiteTestChromium()
			if path == "" {
				t.Skip("installed Chromium unavailable")
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				name := strings.TrimPrefix(r.URL.Path, "/docs/")
				data, ok := files[name]
				if !ok {
					http.NotFound(w, r)
					return
				}
				switch {
				case strings.HasSuffix(name, ".html"):
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
				case strings.HasSuffix(name, ".js"):
					w.Header().Set("Content-Type", "application/javascript")
				case strings.HasSuffix(name, ".css"):
					w.Header().Set("Content-Type", "text/css")
				}
				w.Write(data)
			}))
			defer server.Close()
			a, cancel := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(path)...)
			defer cancel()
			b, cancel := chromedp.NewContext(a)
			defer cancel()
			ctx, cancel := context.WithTimeout(b, 25*time.Second)
			defer cancel()
			var state struct {
				Status         string
				Math           int
				ResourcesLocal bool
			}
			err = chromedp.Run(ctx, chromedp.Navigate(server.URL+"/docs/nested/guide.html"), chromedp.Evaluate(`(async()=>{await globalThis.margoRuntimeReady;return {status:document.documentElement.dataset.margoRuntimeStatus,math:document.querySelectorAll('.margo-math math').length,resourcesLocal:performance.getEntriesByType('resource').filter(r=>r.name.startsWith('http')).every(r=>r.name.startsWith(location.origin+'/docs/'))}})()`, &state, func(p *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams { return p.WithAwaitPromise(true) }))
			if err != nil {
				t.Fatal(err)
			}
			if state.Status != "ready" || state.Math != 2 || !state.ResourcesLocal {
				t.Fatalf("site state: %+v", state)
			}
		})
	}
}

func TestMathConfiguredSiteHTMXNavigationAndRecovery(t *testing.T) {
	for _, layout := range []string{"legacy", "typed"} {
		t.Run(layout, func(t *testing.T) { testMathConfiguredNavigation(t, layout) })
	}
}

func testMathConfiguredNavigation(t *testing.T, layout string) {
	path := installedSiteTestChromium()
	if path == "" {
		t.Skip("installed Chromium unavailable")
	}
	root := t.TempDir()
	writeConfigFile(t, filepath.Join(root, "content", "index.md"), "# Home\n\n[Math](math.md)\n")
	writeConfigFile(t, filepath.Join(root, "content", "math.md"), "# Math\n\n$x$\n\n[Broken](broken.md)\n")
	writeConfigFile(t, filepath.Join(root, "content", "broken.md"), "# Broken\n\n$\\unknownMath$\n\n[Recover](recovery.md)\n")
	writeConfigFile(t, filepath.Join(root, "content", "recovery.md"), "# Recovery\n\n$y$\n")
	copyMargoAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeConfigFile(t, filepath.Join(root, "site.yaml"), `version: 1
source: content
assets: local
base_path: /docs
site:
  name: Math
  description: Math navigation fixture.
  base_url: https://docs.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Math navigation fixture.
shell:
  builtin: componentdocshell
locales:
  default: en
  supported: [en]
`)
	if layout == "typed" {
		configPath := filepath.Join(root, "site.yaml")
		config, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}
		writeConfigFile(t, configPath, strings.Replace(string(config), "shell:\n  builtin: componentdocshell\n", "layout:\n  kind: docs\n", 1))
	}
	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml"), Compiler: margo.New(margo.WithMath(true))})
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, artifact := range result.Artifacts {
		files[artifact.Path] = artifact.Content
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/docs/")
		data, ok := files[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", siteBrowserContentType(name))
		w.Write(data)
	}))
	defer server.Close()
	a, cancel := browserlaunch.NewExecAllocator(context.Background(), siteTestChromiumAllocatorOptions(path)...)
	defer cancel()
	b, cancel := chromedp.NewContext(a)
	defer cancel()
	ctx, cancel := context.WithTimeout(b, 30*time.Second)
	defer cancel()
	if err := chromedp.Run(ctx, chromedp.EmulateViewport(1440, 900), chromedp.Navigate(server.URL+"/docs/index.html"), chromedp.Evaluate(`window.__mathNavigationProbe = 1; window.__mathSettled=0; document.addEventListener("htmx:after:settle",e=>{if(e.detail.task?.target?.id === "main-content")window.__mathSettled++})`, nil)); err != nil {
		t.Fatal(err)
	}
	for index, step := range []struct{ page, status string }{{"math", "ready"}, {"broken", "failed"}, {"recovery", "ready"}} {
		if err := chromedp.Run(ctx,
			chromedp.Poll(`(()=>{const n=document.querySelector('a[href="/docs/`+step.page+`.html"][hx-get]');if(!n)return false;const r=n.getBoundingClientRect();return n.contains(document.elementFromPoint(r.x+r.width/2,r.y+r.height/2))})()`, nil, chromedp.WithPollingTimeout(8*time.Second)),
			chromedp.Click(`a[href="/docs/`+step.page+`.html"][hx-get]`, chromedp.ByQuery),
			chromedp.Poll(`location.pathname.endsWith('/`+step.page+`.html') && !!document.querySelector('.margo-math[data-margo-runtime-status="`+map[string]string{"ready": "succeeded", "failed": "failed"}[step.status]+`"]')`, nil, chromedp.WithPollingTimeout(8*time.Second)),
			chromedp.Poll(`window.__mathSettled >= `+fmt.Sprint(index+1), nil, chromedp.WithPollingTimeout(8*time.Second)),
		); err != nil {
			var debug string
			chromedp.Run(ctx, chromedp.Evaluate(`JSON.stringify({url:location.href,status:document.documentElement.dataset.margoRuntimeStatus,math:[...document.querySelectorAll('.margo-math')].map(n=>({status:n.dataset.margoRuntimeStatus,text:n.textContent})),scripts:[...document.querySelectorAll('[data-margo-requirement]')].map(n=>n.dataset.margoRequirement)})`, &debug))
			t.Fatalf("navigation to %s: %v; %s", step.page, err, debug)
		}
		var state struct {
			Probe          int
			Status, Source string
			Engines        int
		}
		if err := chromedp.Run(ctx, chromedp.Evaluate(`({probe:window.__mathNavigationProbe,status:document.documentElement.dataset.margoRuntimeStatus,source:document.querySelector('.margo-math__source').textContent,engines:document.querySelectorAll('[data-margo-requirement="margo.math.engine"]').length})`, &state)); err != nil {
			t.Fatal(err)
		}
		if state.Probe != 1 || state.Status != step.status || state.Engines != 1 {
			t.Fatalf("%s: %+v", step.page, state)
		}
		if step.page == "recovery" && state.Source != "y" {
			t.Fatalf("recovery rendered stale math: %+v", state)
		}
	}
}
