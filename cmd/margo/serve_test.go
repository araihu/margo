package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/araihu/margo/internal/devserver"
)

func TestServeCommandDefaultsToCurrentDirectory(t *testing.T) {
	var got serveRequest
	command := NewRootCommand(Dependencies{
		Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Build: testBuildInfo(),
		ServeSite: func(_ context.Context, request serveRequest) error {
			got = request
			return nil
		},
	})
	command.SetArgs([]string{"serve"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := serveRequest{Input: ".", Host: "127.0.0.1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("request = %+v, want %+v", got, want)
	}
}

func TestServeCommandPassesInputAndDevelopmentFlags(t *testing.T) {
	var got serveRequest
	command := NewRootCommand(Dependencies{
		Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, Build: testBuildInfo(),
		ServeSite: func(_ context.Context, request serveRequest) error {
			got = request
			return nil
		},
	})
	command.SetArgs([]string{"serve", "docs/site.yaml", "--host", "::1", "--port", "4321", "--open"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := serveRequest{Input: "docs/site.yaml", Host: "::1", Port: 4321, PortExplicit: true, Open: true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("request = %+v, want %+v", got, want)
	}
}

func TestServeCommandRejectsTooManyInputsAndInvalidPorts(t *testing.T) {
	for _, args := range [][]string{
		{"serve", "one", "two"},
		{"serve", "--port", "0"},
		{"serve", "--port", "65536"},
	} {
		var stderr bytes.Buffer
		called := false
		command := NewRootCommand(Dependencies{
			Stdout: &bytes.Buffer{}, Stderr: &stderr, Build: testBuildInfo(),
			ServeSite: func(context.Context, serveRequest) error {
				called = true
				return nil
			},
		})
		command.SetArgs(args)
		if err := command.ExecuteContext(context.Background()); err == nil {
			t.Fatalf("args %v succeeded", args)
		}
		if called {
			t.Fatalf("args %v called server", args)
		}
	}
}

func TestServeHelpIsExplicitlyDevelopmentOnly(t *testing.T) {
	var stdout bytes.Buffer
	command := NewRootCommand(Dependencies{Stdout: &stdout, Stderr: &bytes.Buffer{}, Build: testBuildInfo()})
	command.SetArgs([]string{"serve", "--help"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	help := strings.ToLower(stdout.String())
	for _, required := range []string{"development", "not for production", "live reload"} {
		if !strings.Contains(help, required) {
			t.Fatalf("serve help missing %q: %s", required, stdout.String())
		}
	}
}

func TestResolveServeProjectAutoSelectsSiteYAML(t *testing.T) {
	root := t.TempDir()
	writeSiteFixture(t, filepath.Join(root, "docs", "index.md"), "# Configured\n")
	copySiteConfigAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copySiteConfigAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeSiteFixture(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
output: dist
base_path: /docs
site:
  name: Margo
  base_url: https://margo.example
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo documentation preview
`)
	deps := normalizeDependencies(Dependencies{WorkingDirectory: root})
	project, err := resolveServeProject(deps, ".")
	if err != nil {
		t.Fatal(err)
	}
	if project.configPath != filepath.Join(root, "site.yaml") || project.root != root {
		t.Fatalf("project = %+v", project)
	}
	snapshot, err := project.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.BasePath() != "/docs" || snapshot.PageCount() != 1 {
		t.Fatalf("snapshot base = %q pages = %d", snapshot.BasePath(), snapshot.PageCount())
	}
	assertSnapshotRoute(t, snapshot, "/docs/margo-manifest.json", `"schemaVersion":"margo-site-manifest/v1"`)
	if _, err := os.Stat(filepath.Join(root, "dist")); !os.IsNotExist(err) {
		t.Fatalf("serve wrote configured output: %v", err)
	}
	if !project.Ignore(filepath.Join(root, "dist", "index.html")) {
		t.Fatal("configured output is not ignored")
	}
}

func TestServeConfiguredProjectIgnoresSiblingArtifactsAndWatchesInputs(t *testing.T) {
	root := t.TempDir()
	writeSiteFixture(t, filepath.Join(root, "docs", "index.md"), "# Configured\n")
	copySiteConfigAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copySiteConfigAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeSiteFixture(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
output: dist
assets: local
site:
  name: Margo
  base_url: https://margo.example
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo documentation preview
`)
	project, err := resolveServeProject(normalizeDependencies(Dependencies{WorkingDirectory: root}), ".")
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		"build/chromium.log",
		"logs/serve.log",
		"reports/site.json",
		"artifacts/screenshot.png",
		"dist/index.html",
	} {
		if !project.Ignore(filepath.Join(root, filepath.FromSlash(name))) {
			t.Errorf("sibling artifact %q was not ignored", name)
		}
	}
	for _, name := range []string{
		"site.yaml",
		"docs/index.md",
		"assets/logo.svg",
		"assets/social.jpg",
	} {
		if project.Ignore(filepath.Join(root, filepath.FromSlash(name))) {
			t.Errorf("configured input %q was ignored", name)
		}
	}
}

func TestServeConfiguredProjectKeepsArtifactNamedAssetAndSourceWatchable(t *testing.T) {
	root := t.TempDir()
	writeSiteFixture(t, filepath.Join(root, "build", "index.md"), "# Configured source\n")
	copySiteConfigAsset(t, filepath.Join(root, "build", "logo.svg"), "logo.svg")
	copySiteConfigAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeSiteFixture(t, filepath.Join(root, "site.yaml"), `version: 1
source: build
output: dist
assets: local
site:
  name: Margo
  base_url: https://margo.example
  logo: build/logo.svg
  icon: build/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo documentation preview
`)
	project, err := resolveServeProject(normalizeDependencies(Dependencies{WorkingDirectory: root}), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"build/index.md", "build/logo.svg", "assets/social.jpg"} {
		if project.Ignore(filepath.Join(root, filepath.FromSlash(name))) {
			t.Errorf("valid input %q was ignored", name)
		}
	}
}

func TestResolveServeProjectBuildsRawMarkdownTree(t *testing.T) {
	root := t.TempDir()
	writeSiteFixture(t, filepath.Join(root, "index.md"), "# Raw tree\n")
	deps := normalizeDependencies(Dependencies{WorkingDirectory: root})
	project, err := resolveServeProject(deps, ".")
	if err != nil {
		t.Fatal(err)
	}
	if project.configPath != "" || project.root != root {
		t.Fatalf("project = %+v", project)
	}
	snapshot, err := project.Build(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertSnapshotRoute(t, snapshot, "/", "Raw tree")
	assertSnapshotRoute(t, snapshot, "/margo-manifest.json", `"schemaVersion":"margo-site-manifest/v1"`)
}

func TestResolveServeProjectAcceptsExplicitYAMLAndRecoversEmptyTreeBuild(t *testing.T) {
	root := t.TempDir()
	config := filepath.Join(root, "custom.yml")
	writeSiteFixture(t, config, "version: 1\nsource: docs\n")
	deps := normalizeDependencies(Dependencies{WorkingDirectory: root})
	project, err := resolveServeProject(deps, "custom.yml")
	if err != nil {
		t.Fatal(err)
	}
	if project.configPath != config {
		t.Fatalf("config path = %q, want %q", project.configPath, config)
	}
	if _, err := project.Build(context.Background()); err == nil {
		t.Fatal("invalid initial config build succeeded")
	}

	empty := filepath.Join(root, "empty")
	if err := os.Mkdir(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	project, err = resolveServeProject(deps, "empty")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := project.Build(context.Background()); err == nil || !strings.Contains(err.Error(), "site.sources_empty") {
		t.Fatalf("empty build error = %v", err)
	}
}

func TestResolveServeProjectRejectsMissingAndNonYAMLFiles(t *testing.T) {
	root := t.TempDir()
	writeSiteFixture(t, filepath.Join(root, "config.toml"), "title = 'no'\n")
	deps := normalizeDependencies(Dependencies{WorkingDirectory: root})
	for _, input := range []string{"missing", "config.toml"} {
		if _, err := resolveServeProject(deps, input); err == nil {
			t.Fatalf("input %q accepted", input)
		}
	}
}

func TestServeProjectUpdatesOutputExclusionAfterConfigLoadBeforeFailedBuild(t *testing.T) {
	root := t.TempDir()
	writeSiteFixture(t, filepath.Join(root, "docs", "index.md"), "# Ready\n")
	configPath := filepath.Join(root, "site.yaml")
	writeSiteFixture(t, configPath, `version: 1
source: docs
output: dist
site:
  name: Margo
  base_url: https://margo.example
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo documentation preview
`)
	deps := normalizeDependencies(Dependencies{WorkingDirectory: root})
	project, err := resolveServeProject(deps, ".")
	if err != nil {
		t.Fatal(err)
	}
	writeSiteFixture(t, configPath, `version: 1
source: missing
output: public
site:
  name: Margo
  base_url: https://margo.example
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo documentation preview
`)
	if _, err := project.Build(context.Background()); err == nil {
		t.Fatal("build with missing source succeeded")
	}
	if !project.Ignore(filepath.Join(root, "public", "index.html")) {
		t.Fatal("new configured output is not ignored after failed build")
	}
	if project.Ignore(filepath.Join(root, "dist", "index.html")) {
		t.Fatal("stale configured output remains ignored")
	}
}

func assertSnapshotRoute(t *testing.T, snapshot devserver.Snapshot, requestPath, required string) {
	t.Helper()
	store := devserver.NewSnapshotStore()
	store.Replace(snapshot)
	response := httptest.NewRecorder()
	devserver.NewHandler(store, devserver.NewBroker()).ServeHTTP(response, httptest.NewRequest(http.MethodGet, requestPath, nil))
	data, err := io.ReadAll(response.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || !strings.Contains(string(data), required) {
		t.Fatalf("route %s = %d %q, missing %q", requestPath, response.Code, data, required)
	}
}
