package site

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func editorialFixture(t *testing.T, mode AssetMode) Result {
	t.Helper()
	root := t.TempDir()
	writeTypedLayoutBuildConfig(t, root, `layout:
  kind: docs
  default:
    sidebar: false
    pagination: false
navigation:
  links:
    - label: Blog & projects
      href: /posts/
    - label: RSS
      href: https://example.com/feed.xml
search:
  label: Search post
  placeholder: Find a post
  empty_text: No posts found.
`)
	configPath := filepath.Join(root, "site.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	config := strings.Replace(string(data), "assets: local", "assets: "+string(mode), 1)
	config = strings.Replace(config, "  logo: assets/logo.svg", "  logo: assets/logo.svg\n  logo_dark: assets/dark.svg\n  hide_name: true", 1)
	config = strings.Replace(config, "  color_mode: light", "  color_mode: light\n  allow_switch_theme: true", 1)
	writeConfigFile(t, configPath, config)
	copyMargoAsset(t, filepath.Join(root, "assets", "dark.svg"), "logo.svg")
	writeConfigFile(t, filepath.Join(root, "docs", "index.md"), "# Home\n\n## First section\n\n[Link](posts/index.md).\n\n"+strings.Repeat("Paragraph of editorial content.\n\n", 40)+"## Second section\n\n"+strings.Repeat("More content.\n\n", 30))
	writeConfigFile(t, filepath.Join(root, "docs", "posts", "index.md"), "---\nlayout:\n  values:\n    pagination: true\n---\n# Posts\n\n## Section\n\nDetails.\n")
	result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: configPath})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestEditorialComposition(t *testing.T) {
	for _, mode := range []AssetMode{AssetsLocal, AssetsInline} {
		t.Run(string(mode), func(t *testing.T) {
			result := editorialFixture(t, mode)
			home := string(configArtifact(t, result, "index.html"))
			for _, want := range []string{`class="component-doc-shell__header"`, `class="margo-header-links"`, `Blog &amp; projects`, `margo-theme-logo__dark`, `Search post`, `Find a post`, `No posts found.`, `data-margo-sidebar="false"`, `data-componentdocshell-toc`} {
				if !strings.Contains(home, want) {
					t.Errorf("missing %s", want)
				}
			}
			for _, unwanted := range []string{`class="margo-pagination"`, `class="component-doc-shell__brand-name"`, `Search documentation`} {
				if strings.Contains(home, unwanted) {
					t.Errorf("unexpected %s", unwanted)
				}
			}
			if strings.Count(home, `class="component-doc-shell__header"`) != 1 {
				t.Fatal("expected one header")
			}
			if !strings.Contains(string(configArtifact(t, result, "posts/index.html")), `class="margo-pagination"`) {
				t.Fatal("page override did not restore pagination")
			}
			configArtifact(t, result, "assets/dark.svg")
		})
	}
}

func TestEditorialConfigRejectsUnsafeValues(t *testing.T) {
	for _, extra := range []string{
		"navigation:\n  links:\n    - {label: Run, href: 'javascript:alert(1)'}\n",
		"navigation:\n  links:\n    - {label: Run, href: '//example.com/'}\n",
		"navigation:\n  links:\n    - {label: '', href: '/'}\n",
		"search:\n  label: '  '\n",
	} {
		t.Run(extra, func(t *testing.T) {
			root := t.TempDir()
			writeTypedLayoutBuildConfig(t, root, "layout:\n  kind: docs\n"+extra)
			if _, err := LoadConfig(filepath.Join(root, "site.yaml")); err == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
	root := t.TempDir()
	writeTypedLayoutBuildConfig(t, root, "layout:\n  kind: docs\n")
	config, err := LoadConfig(filepath.Join(root, "site.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	config.Site.LogoDark = "../secret.svg"
	if err := config.validate(); err == nil {
		t.Fatal("logo escaped asset root")
	}
	b := builder{config: &config}
	// Defaults remain localized and explicit text is preserved.
	if got := b.siteSearchConfig("pt-BR"); got.Label != "Buscar páginas" || got.EmptyText != "Nenhuma página encontrada." {
		t.Fatalf("localized search: %+v", got)
	}
}
