package site

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocaleSelectorConfiguration(t *testing.T) {
	for _, layout := range []string{"", "layout:\n  kind: docs\n"} {
		for _, mode := range []string{"omitted", "flags", "disabled", "missing"} {
			t.Run(fmt.Sprintf("%s/%s", strings.TrimSpace(layout), mode), func(t *testing.T) {
				root := t.TempDir()
				copyMargoAsset(t, filepath.Join(root, "assets/logo.svg"), "logo.svg")
				copyMargoAsset(t, filepath.Join(root, "assets/social.jpg"), "social/margo-social-v2.jpg")
				for _, prefix := range []string{"", "pt-br/", "es/"} {
					writeConfigFile(t, filepath.Join(root, "docs", prefix, "index.md"), "# Journal\n")
					writeConfigFile(t, filepath.Join(root, "docs", prefix, "article.md"), "# Article\n")
				}
				configuration := `version: 1
source: docs
base_path: /journal
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
`
				flag := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 3"><path fill="#080" d="M0 0h4v3H0z"/></svg>`)
				if mode != "omitted" {
					configuration += "  selector:\n    flags: {en: assets/flags/us.svg, pt-BR: assets/flags/br.svg, es: assets/flags/es.svg}\n"
					if mode == "disabled" {
						configuration += "    enabled: false\n"
					} else if mode == "flags" {
						for _, name := range []string{"us", "br", "es"} {
							writeConfigFile(t, filepath.Join(root, "assets/flags", name+".svg"), string(flag))
						}
					}
				}
				writeConfigFile(t, filepath.Join(root, "site.yaml"), configuration+layout)
				result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filepath.Join(root, "site.yaml")})
				if mode == "missing" {
					if err == nil {
						t.Fatal("missing configured flag silently ignored")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				page := string(configArtifact(t, result, "pt-br/article.html"))
				if !strings.Contains(page, `hreflang="en" href="https://journal.example/journal/article.html"`) || !strings.Contains(page, `hreflang="pt-BR" href="https://journal.example/journal/pt-br/article.html"`) {
					t.Fatal("selector configuration changed discovery metadata")
				}
				if mode == "disabled" {
					if strings.Contains(page, "margo-language-menu") || strings.Contains(page, `<nav class="margo-locale-controls"`) || strings.Contains(page, `<img class="margo-locale-flag"`) {
						t.Fatal("disabled controls still rendered")
					}
					if artifactExists(result, "assets/flags/us.svg") {
						t.Fatal("disabled selector read or staged unused flag")
					}
				} else if mode == "flags" {
					for _, name := range []string{"us", "br", "es"} {
						if !bytes.Equal(configArtifact(t, result, "assets/flags/"+name+".svg"), flag) {
							t.Fatal("flag bytes changed")
						}
					}
					if !strings.Contains(page, `assets/flags/us.svg`) || !strings.Contains(page, `aria-hidden="true"`) || !strings.Contains(page, "English") || !strings.Contains(page, "Español") {
						t.Fatal("flag URLs or visible language names missing")
					}
				} else if strings.Contains(page, `<img class="margo-locale-flag"`) {
					t.Fatal("omitted flags changed existing appearance")
				}
			})
		}
	}
}

func TestLocaleSelectorRejectsInvalidFlagMappings(t *testing.T) {
	for _, flags := range []map[string]string{
		{"fr": "assets/fr.svg"}, {"en": "../flag.svg"}, {"en": "https://flags.example/us.svg"}, {"en": "/flags/us.svg"}, {"en": ""}, {"en": "assets/us.svg?x"},
	} {
		if err := validateLocales(LocaleConfig{Default: "en", Supported: []string{"en"}, Selector: LocaleSelectorConfig{Flags: flags}}); err == nil {
			t.Fatalf("accepted invalid flag mapping: %v", flags)
		}
	}
}
