package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/araihu/margo/site"
)

func TestSiteCommandBuildsDirectoryAndManifest(t *testing.T) {
	input := t.TempDir()
	output := filepath.Join(t.TempDir(), "published")
	writeSiteFixture(t, filepath.Join(input, "index.md"), "---\ntitle: Home\nlanguage: en\nauthors: [Ana Silva]\npublishedAt: \"2026-08-25T12:00:00Z\"\nmodifiedAt: \"2026-08-26T12:00:00Z\"\ntags: [operations]\n---\n# Home\n\n[Guide](guide/readme.md)\n\n![Logo](assets/logo.png)\n")
	writeSiteFixture(t, filepath.Join(input, "guide", "readme.md"), "# Guide\n\n[Home](../index.md#home)\n")
	writeSiteFixture(t, filepath.Join(input, "assets", "logo.png"), "\x89PNG\r\n\x1a\n")

	var stdout, stderr bytes.Buffer
	command := NewRootCommand(Dependencies{Stdout: &stdout, Stderr: &stderr, Build: testBuildInfo()})
	command.SetArgs([]string{"site", input, "--output-dir", output, "--assets", "local", "--diagnostics", "json"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("site: %v\nstderr: %s", err, stderr.String())
	}
	var report struct {
		SchemaVersion string `json:"schemaVersion"`
		Artifacts     int    `json:"artifacts"`
		Manifest      string `json:"manifest"`
		Pages         []struct {
			Source string `json:"source"`
			Output string `json:"output"`
		} `json:"pages"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("report: %v: %s", err, stdout.String())
	}
	if report.SchemaVersion != "margo-site-report/v1" || report.Manifest == "" || len(report.Pages) != 2 {
		t.Fatalf("report = %+v", report)
	}
	index := readSiteFixture(t, filepath.Join(output, "index.html"))
	guide := readSiteFixture(t, filepath.Join(output, "guide", "readme.html"))
	if !strings.Contains(index, `href="guide/readme.html"`) || !strings.Contains(guide, `href="../index.html#home"`) {
		t.Fatalf("links not rewritten:\nindex=%s\nguide=%s", index, guide)
	}
	for _, document := range []string{index, guide} {
		for _, forbidden := range []string{
			"margo-page-actions", "margo-breadcrumbs", "margo-pagination",
			`id="left-nav"`, `id="right-nav"`, "data-margo-layout", "goshtoso",
		} {
			if strings.Contains(document, forbidden) {
				t.Fatalf("plain site HTML contains %q: %s", forbidden, document)
			}
		}
	}
	if got := readSiteFixture(t, filepath.Join(output, "assets", "logo.png")); got != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("logo = %x", got)
	}
	if got := readSiteFixture(t, filepath.Join(output, "margo-assets", "document.css")); got == "" {
		t.Fatal("semantic document stylesheet is empty")
	}
	for _, unwanted := range []string{
		"assets/styles.css", "margo-assets/site.css", "margo-assets/page-actions.css",
		"margo-assets/page-actions.js", "margo-assets/icons/page-actions.svg",
	} {
		if _, err := os.Stat(filepath.Join(output, filepath.FromSlash(unwanted))); !os.IsNotExist(err) {
			t.Fatalf("plain site unexpectedly publishes %q: %v", unwanted, err)
		}
	}
	manifest := readSiteFixture(t, filepath.Join(output, "margo-manifest.json"))
	if !strings.Contains(manifest, `"schemaVersion":"margo-site-manifest/v1"`) || !strings.Contains(manifest, `"index.html"`) || !strings.Contains(manifest, `"authors":["Ana Silva"]`) || !strings.Contains(manifest, `"publishedAt":"2026-08-25T12:00:00Z"`) || !strings.Contains(manifest, `"modifiedAt":"2026-08-26T12:00:00Z"`) || !strings.Contains(manifest, `"tags":["operations"]`) {
		t.Fatalf("manifest = %s", manifest)
	}
}

func TestSiteCommandBuildsYAMLConfig(t *testing.T) {
	root := t.TempDir()
	writeSiteFixture(t, filepath.Join(root, "docs", "index.md"), "# Home\n\nConfigured site.\n")
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

	var stdout, stderr bytes.Buffer
	command := NewRootCommand(Dependencies{Stdout: &stdout, Stderr: &stderr, Build: testBuildInfo()})
	command.SetArgs([]string{"site", filepath.Join(root, "site.yaml"), "--diagnostics", "json"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("site config: %v\nstderr: %s", err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, "dist", "index.html")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{site.SitemapPath, site.LLMSPath} {
		if _, err := os.Stat(filepath.Join(root, "dist", name)); err != nil {
			t.Fatalf("generated discovery artifact %q: %v", name, err)
		}
	}
	manifest := readSiteFixture(t, filepath.Join(root, "dist", "margo-manifest.json"))
	for _, required := range []string{`"configVersion":1`, `"layout":"frame:top-left-main-footer"`, `"basePath":"/docs"`, `"documentStyleDigest":"`} {
		if !strings.Contains(manifest, required) {
			t.Fatalf("config manifest missing %q: %s", required, manifest)
		}
	}
}

func TestSiteCommandBuildsDocumentedMinimalConfig(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	repository := filepath.Join(filepath.Dir(filename), "..", "..")
	docs, err := os.ReadFile(filepath.Join(repository, "showcase", "content", "cli", "site", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	config := extractDocumentedSiteConfig(t, string(docs))

	root := t.TempDir()
	writeSiteFixture(t, filepath.Join(root, "site.yaml"), config)
	writeSiteFixture(t, filepath.Join(root, "docs", "index.md"), "---\ntitle: Example Docs\ndescription: A small configured documentation site.\nlanguage: en\n---\n# Example Docs\n\nThis page is generated by the configured site contract.\n")
	copySiteConfigAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copySiteConfigAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")

	var stdout, stderr bytes.Buffer
	command := NewRootCommand(Dependencies{Stdout: &stdout, Stderr: &stderr, Build: testBuildInfo()})
	command.SetArgs([]string{"site", filepath.Join(root, "site.yaml"), "--diagnostics", "json"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("documented minimal config: %v\nstderr: %s", err, stderr.String())
	}
	var report struct {
		SchemaVersion string      `json:"schemaVersion"`
		Pages         []site.Page `json:"pages"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("site report: %v\n%s", err, stdout.String())
	}
	if report.SchemaVersion != "margo-site-report/v1" || len(report.Pages) != 1 || report.Pages[0].Output != "index.html" {
		t.Fatalf("site report = %+v", report)
	}
	for _, artifact := range []string{"index.html", "sitemap.xml", "llms.txt", "margo-manifest.json"} {
		if _, err := os.Stat(filepath.Join(root, "dist", artifact)); err != nil {
			t.Fatalf("documented config missing %q: %v", artifact, err)
		}
	}
}

func extractDocumentedSiteConfig(t *testing.T, docs string) string {
	t.Helper()
	sectionStart := strings.Index(docs, "### Minimal configured site")
	if sectionStart < 0 {
		t.Fatal("minimal site documentation section is missing")
	}
	section := docs[sectionStart:]
	fenceStart := strings.Index(section, "```yaml\n")
	if fenceStart < 0 {
		t.Fatal("minimal site YAML fence is missing")
	}
	configStart := fenceStart + len("```yaml\n")
	fenceEnd := strings.Index(section[configStart:], "\n```")
	if fenceEnd < 0 {
		t.Fatal("minimal site YAML fence is unterminated")
	}
	return section[configStart : configStart+fenceEnd]
}

func TestSiteCommandTypedLayoutManifestIncludesRouteIdentity(t *testing.T) {
	root := t.TempDir()
	writeSiteFixture(t, filepath.Join(root, "docs", "index.md"), "---\nlayout:\n  kind: landing\n---\n# Tour\n\nChoose a documentation path.\n")
	writeSiteFixture(t, filepath.Join(root, "docs", "module", "index.md"), "# Module\n\nModule overview.\n")
	writeSiteFixture(t, filepath.Join(root, "docs", "cli", "index.md"), "# CLI\n\nCLI overview.\n")
	writeSiteFixture(t, filepath.Join(root, "docs", "module", "_layout.yaml"), "values:\n  family: module\n")
	writeSiteFixture(t, filepath.Join(root, "docs", "cli", "_layout.yaml"), "values:\n  family: cli\n")
	copySiteConfigAsset(t, filepath.Join(root, "assets", "logo.svg"), "logo.svg")
	copySiteConfigAsset(t, filepath.Join(root, "assets", "social.jpg"), "social/margo-social-v2.jpg")
	writeSiteFixture(t, filepath.Join(root, "site.yaml"), `version: 1
source: docs
output: dist
assets: local
offline: true
site:
  name: Margo
  description: Typed-layout documentation.
  base_url: https://margo.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image:
    path: assets/social.jpg
    alt: Margo layout preview
layout:
  kind: docs
  default:
    families: [module, cli]
  values:
    family: default
navigation:
  mode: file-tree
locales:
  default: en
  supported: [en]
`)

	var stdout, stderr bytes.Buffer
	command := NewRootCommand(Dependencies{Stdout: &stdout, Stderr: &stderr, Build: testBuildInfo()})
	command.SetArgs([]string{"site", filepath.Join(root, "site.yaml"), "--diagnostics", "json"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("typed layout site config: %v\nstderr: %s", err, stderr.String())
	}
	manifest := readSiteFixture(t, filepath.Join(root, "dist", "margo-manifest.json"))
	var document struct {
		LayoutSchemaHash string      `json:"layoutSchemaHash"`
		Routes           []site.Page `json:"routes"`
	}
	if err := json.Unmarshal([]byte(manifest), &document); err != nil {
		t.Fatalf("layout manifest: %v: %s", err, manifest)
	}
	if document.LayoutSchemaHash == "" || document.LayoutSchemaHash == "legacy" {
		t.Fatalf("layoutSchemaHash = %q, want non-empty non-legacy value", document.LayoutSchemaHash)
	}
	want := map[string]struct {
		family string
		layout string
	}{
		"index.md":        {family: "", layout: "landing"},
		"module/index.md": {family: "module", layout: "docs"},
		"cli/index.md":    {family: "cli", layout: "docs"},
	}
	if len(document.Routes) != len(want) {
		t.Fatalf("layout routes = %+v", document.Routes)
	}
	for _, route := range document.Routes {
		expected, ok := want[route.Source]
		if !ok || route.Family != expected.family || route.Layout != expected.layout {
			t.Fatalf("layout route = %+v, want identities %+v", route, want)
		}
	}
}

func TestSiteCommandPublishesInteractiveEmbedAndPolicyIdentity(t *testing.T) {
	input := t.TempDir()
	output := filepath.Join(t.TempDir(), "published")
	policyPath := filepath.Join(t.TempDir(), "policy.json")
	policy := `{
  "schemaVersion": "margo-policy/v1",
  "rawHTML": "sanitized",
  "iframe": {
    "allowedOrigins": ["https://video.example.com/", "https://media.example.com"],
    "sandbox": ["allow-scripts", "allow-presentation"],
    "projections": {
      "html": "interactive",
      "pdf": "static-link",
      "site": "interactive",
      "deck": "deny"
    }
  }
}`
	writeSiteFixture(t, policyPath, policy)
	writeSiteFixture(t, filepath.Join(input, "index.md"), "# Home\n\n<iframe src=\"https://video.example.com/watch/123\" title=\"Architecture overview\"></iframe>\n")

	var stdout, stderr bytes.Buffer
	command := NewRootCommand(Dependencies{Stdout: &stdout, Stderr: &stderr, Build: testBuildInfo()})
	command.SetArgs([]string{"site", input, "--output-dir", output, "--policy", policyPath, "--diagnostics", "json"})
	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("site: %v\nstderr: %s", err, stderr.String())
	}
	parsedPolicy, err := parsePolicyDocument([]byte(policy))
	if err != nil {
		t.Fatal(err)
	}
	digest := parsedPolicy.Digest
	var report struct {
		Policy string `json:"policy"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("report: %v: %s", err, stdout.String())
	}
	if report.Policy != digest {
		t.Fatalf("report policy = %q", report.Policy)
	}
	page := readSiteFixture(t, filepath.Join(output, "index.html"))
	if !strings.Contains(page, `<iframe class="margo-embed__frame"`) {
		t.Fatalf("site page missing trusted embed: %s", page)
	}
	manifest := readSiteFixture(t, filepath.Join(output, "margo-manifest.json"))
	if !strings.Contains(manifest, `"policy":"`+digest+`"`) {
		t.Fatalf("manifest missing policy identity: %s", manifest)
	}
}

func TestWriteSiteReportIncludesPolicyIdentityInText(t *testing.T) {
	var output bytes.Buffer
	const digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := writeSiteReport(&output, diagnosticText, siteReport{Policy: digest}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "policy "+digest) {
		t.Fatalf("text report = %q", output.String())
	}
}

func TestSiteCommandRefusesExistingOutputWithoutMutation(t *testing.T) {
	input := t.TempDir()
	output := filepath.Join(t.TempDir(), "published")
	writeSiteFixture(t, filepath.Join(input, "index.md"), "# Home\n")
	writeSiteFixture(t, filepath.Join(output, "keep.txt"), "untouched")

	var stdout, stderr bytes.Buffer
	command := NewRootCommand(Dependencies{Stdout: &stdout, Stderr: &stderr, Build: testBuildInfo()})
	command.SetArgs([]string{"site", input, "--output-dir", output, "--diagnostics", "json"})
	err := command.ExecuteContext(context.Background())
	if cliDiagnosticCode(err) != "site.output_exists" {
		t.Fatalf("error = %v stderr = %s", err, stderr.String())
	}
	if got := readSiteFixture(t, filepath.Join(output, "keep.txt")); got != "untouched" {
		t.Fatalf("existing output mutated: %q", got)
	}
	if _, statErr := os.Stat(filepath.Join(output, "index.html")); !os.IsNotExist(statErr) {
		t.Fatalf("unexpected published file: %v", statErr)
	}
	if !strings.Contains(stderr.String(), `"code":"site.output_exists"`) {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

func TestRenameSiteDirectoryNoReplacePreservesConcurrentTarget(t *testing.T) {
	parent := t.TempDir()
	stage := filepath.Join(parent, "stage")
	target := filepath.Join(parent, "target")
	if err := os.Mkdir(stage, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSiteFixture(t, filepath.Join(stage, "new.txt"), "new")
	writeSiteFixture(t, filepath.Join(target, "keep.txt"), "keep")

	if err := renameSiteDirectoryNoReplace(stage, target); err == nil {
		t.Fatal("no-replace directory rename replaced an existing target")
	}
	if got := readSiteFixture(t, filepath.Join(target, "keep.txt")); got != "keep" {
		t.Fatalf("target changed: %q", got)
	}
	if got := readSiteFixture(t, filepath.Join(stage, "new.txt")); got != "new" {
		t.Fatalf("stage changed after refused commit: %q", got)
	}
}

func writeSiteFixture(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readSiteFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func copySiteConfigAsset(t *testing.T, target, relative string) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "..", "assets", relative))
	if err != nil {
		t.Fatal(err)
	}
	writeSiteFixture(t, target, string(data))
}
