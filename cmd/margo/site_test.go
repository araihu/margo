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
	writeSiteFixture(t, filepath.Join(input, "index.md"), "# Home\n\n[Guide](guide/readme.md)\n\n![Logo](assets/logo.png)\n")
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
	if report.SchemaVersion != "margo-site-report/v1" || report.Artifacts < 5 || report.Manifest == "" || len(report.Pages) != 2 {
		t.Fatalf("report = %+v", report)
	}
	index := readSiteFixture(t, filepath.Join(output, "index.html"))
	guide := readSiteFixture(t, filepath.Join(output, "guide", "readme.html"))
	if !strings.Contains(index, `href="guide/readme.html"`) || !strings.Contains(guide, `href="../index.html#home"`) {
		t.Fatalf("links not rewritten:\nindex=%s\nguide=%s", index, guide)
	}
	if got := readSiteFixture(t, filepath.Join(output, "assets", "logo.png")); got != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("logo = %x", got)
	}
	manifest := readSiteFixture(t, filepath.Join(output, "margo-manifest.json"))
	if !strings.Contains(manifest, `"schemaVersion":"margo-site-manifest/v1"`) || !strings.Contains(manifest, `"index.html"`) {
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
