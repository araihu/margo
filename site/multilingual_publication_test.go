package site

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestMultilingualPublicationMetadata(t *testing.T) {
	root := t.TempDir()
	copyMargoAsset(t, filepath.Join(root, "assets/logo.svg"), "logo.svg")
	copyMargoAsset(t, filepath.Join(root, "assets/social.jpg"), "social/margo-social-v2.jpg")
	configuration := `version: 1
source: docs
site:
  name: Journal
  base_url: https://journal.example
  home: index.md
  logo: assets/logo.svg
  icon: assets/logo.svg
  social_image: {path: assets/social.jpg, alt: Journal}
locales:
  default: en
  supported: [en, pt-BR, es]
navigation:
  links:
    - {label: English destination, href: /, locale: en}
    - {label: Destino português, href: /pt-br/, locale: pt-BR}
    - {label: Destino español, href: /es/, locale: es}
  footer_links:
    - {label: English footer, href: /, locale: en}
    - {label: Pie español, href: /es/, locale: es}
`
	for _, prefix := range []string{"", "pt-br/", "es/"} {
		writeConfigFile(t, filepath.Join(root, "docs", prefix, "index.md"), "# Journal\n")
		writeConfigFile(t, filepath.Join(root, "docs", prefix, "article.md"), `---
title: Journal
authors: [Journal, Guest]
publishedAt: "2026-10-29T12:00:00Z"
structured_data:
  type: BlogPosting
  author_types: {Journal: Organization}
  author_urls: {Journal: https://journal.example/about/}
---
# Journal
`)
	}
	writeConfigFile(t, filepath.Join(root, "docs/hidden.md"), "---\nnoindex: true\n---\n# Hidden\n")
	writeConfigFile(t, filepath.Join(root, "docs/es/hidden.md"), "# Oculta\n")
	writeConfigFile(t, filepath.Join(root, "docs/es/only.md"), "# Solo español\n")
	for _, layout := range []string{"", "\nlayout:\n  kind: docs\n  default: {sidebar: false, toc: false, pagination: false}\n"} {
		writeConfigFile(t, filepath.Join(root, "site.yaml"), configuration+layout)
		result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
		if err != nil {
			t.Fatal(err)
		}
		for _, prefix := range []string{"", "pt-br/", "es/"} {
			page := string(configArtifact(t, result, prefix+"article.html"))
			for locale, route := range map[string]string{"en": "", "pt-BR": "pt-br/", "es": "es/", "x-default": ""} {
				want := fmt.Sprintf(`<link rel="alternate" hreflang="%s" href="https://journal.example/%sarticle.html"`, locale, route)
				if strings.Count(page, want) != 1 {
					t.Fatalf("%s missing unique alternate %s", prefix, want)
				}
			}
			for _, want := range []string{`"@type":"Organization"`, `"@type":"Person"`, `href="https://journal.example/` + prefix + `article.html"`} {
				if !strings.Contains(page, want) {
					t.Fatalf("%s missing %s", prefix, want)
				}
			}
			if layout != "" && prefix == "es/" && (!strings.Contains(page, "Destino español") || !strings.Contains(page, "Pie español") || strings.Contains(page, "English destination") || strings.Contains(page, "Destino português")) {
				t.Fatal("Spanish navigation mixes locale identities")
			}
			if prefix == "es/" && (!strings.Contains(page, "Saltar al contenido") || !strings.Contains(page, `aria-label="Autores"`)) {
				t.Fatal("Spanish publication chrome is not localized")
			}
		}
		for _, file := range []string{"hidden.html", "es/hidden.html", "es/only.html"} {
			if strings.Contains(string(configArtifact(t, result, file)), `<link rel="alternate" hreflang=`) {
				t.Fatalf("%s advertises an unavailable or noindex equivalent", file)
			}
		}
	}
}
