package site

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
	"golang.org/x/net/html"
)

func writeTestPNG(t *testing.T, filename string, width, height int) {
	t.Helper()
	var content bytes.Buffer
	if err := png.Encode(&content, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, content.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
}

func publicationFixture(t *testing.T, kind string, mode AssetMode) (string, string) {
	t.Helper()
	root := t.TempDir()
	writeTypedLayoutBuildConfig(t, root, "base_path: /journal\nlayout:\n  kind: "+kind+"\npublication:\n  date_format: '02/01/2006'\n")
	filename := filepath.Join(root, "site.yaml")
	config, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	config = bytes.Replace(config, []byte("assets: local"), []byte("assets: "+mode), 1)
	config = bytes.Replace(config, []byte("  icon: assets/logo.svg"), []byte(`  icon: assets/logo.svg
  icons:
    - {path: assets/logo.svg, sizes: any}
    - {path: assets/icon.png, sizes: 32x32}
    - {path: assets/touch.png, rel: apple-touch-icon, sizes: 180x180}
    - {path: assets/favicon.ico, rel: shortcut icon}
  webmanifest: app/site.webmanifest
  application_name: 'Notes & <news>'
  theme_color: '#007d9c'`), 1)
	if err := os.WriteFile(filename, config, 0644); err != nil {
		t.Fatal(err)
	}
	writeTestPNG(t, filepath.Join(root, "assets/icon.png"), 32, 32)
	writeTestPNG(t, filepath.Join(root, "assets/touch.png"), 180, 180)
	var iconPNG bytes.Buffer
	if err := png.Encode(&iconPNG, image.NewRGBA(image.Rect(0, 0, 32, 32))); err != nil {
		t.Fatal(err)
	}
	ico := make([]byte, 22)
	ico[2], ico[4], ico[6], ico[7] = 1, 1, 32, 32
	binary.LittleEndian.PutUint32(ico[14:18], uint32(iconPNG.Len()))
	binary.LittleEndian.PutUint32(ico[18:22], 22)
	ico = append(ico, iconPNG.Bytes()...)
	if err := os.WriteFile(filepath.Join(root, "assets/favicon.ico"), ico, 0644); err != nil {
		t.Fatal(err)
	}
	writeConfigFile(t, filepath.Join(root, "app/site.webmanifest"), `{"name":"Notes","icons":[{"src":"../assets/icon.png","sizes":"32x32","type":"image/png"}]}`)
	writeConfigFile(t, filepath.Join(root, "docs/index.md"), "# Home\n\n[Post](posts/one/index.md).\n")
	writeConfigFile(t, filepath.Join(root, "docs/posts/one/index.md"), `---
title: A post
authors: [Ana]
publishedAt: "2026-10-28T23:30:00-03:00"
modifiedAt: "2026-10-30T00:30:00+05:00"
structured_data:
  type: BlogPosting
social_image:
  path: ./assets/cover image.png
  alt: 'Cover & <news>'
---
# A post

Content.
`)
	writeTestPNG(t, filepath.Join(root, "docs/posts/one/assets/cover image.png"), 960, 540)
	return root, filename
}

func TestPublicationFeaturesConfiguredLayouts(t *testing.T) {
	for _, kind := range []string{"docs", "article", "landing"} {
		for _, mode := range []AssetMode{AssetsLocal, AssetsInline} {
			t.Run(kind+"/"+string(mode), func(t *testing.T) {
				_, filename := publicationFixture(t, kind, mode)
				result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filename})
				if err != nil {
					t.Fatalf("%#v", err)
				}
				home := string(configArtifact(t, result, "index.html"))
				post := string(configArtifact(t, result, "posts/one/index.html"))
				for _, want := range []string{`property="og:type" content="article"`, `property="og:image" content="https://margo.example/journal/posts/one/assets/cover%20image.png"`, `name="twitter:image" content="https://margo.example/journal/posts/one/assets/cover%20image.png"`, `property="og:image:width" content="960"`, `property="og:image:height" content="540"`, `property="og:image:alt" content="Cover &amp; &lt;news&gt;"`, `datetime="2026-10-28T23:30:00-03:00"`, `>28/10/2026</time>`, `>30/10/2026</time>`, `"datePublished":"2026-10-28T23:30:00-03:00"`} {
					if !strings.Contains(post, want) {
						t.Errorf("post missing %s", want)
					}
				}
				for _, content := range []string{home, post} {
					root, err := html.Parse(strings.NewReader(content))
					if err != nil {
						t.Fatal(err)
					}
					head := firstElement(root, "head")
					for node := head.FirstChild; node != nil; node = node.NextSibling {
						if node.Type == html.ElementNode && node.Data == "link" && attributeValue(node, "sizes") == "any" {
							href, err := url.Parse(attributeValue(node, "href"))
							if err != nil {
								t.Fatal(err)
							}
							base, _ := url.Parse("https://margo.example/journal/")
							if content == post {
								base, _ = url.Parse("https://margo.example/journal/posts/one/")
							}
							if base.ResolveReference(href).String() != "https://margo.example/journal/assets/logo.svg" {
								t.Errorf("icon resolves outside publication: %s", href)
							}
						}
					}

					for _, want := range []string{`type="image/svg+xml" sizes="any"`, `rel="apple-touch-icon" href="/journal/assets/touch.png"`, `rel="manifest" href="/journal/app/site.webmanifest"`, `name="application-name" content="Notes &amp; &lt;news&gt;"`, `name="theme-color" content="#007d9c"`} {
						if !strings.Contains(content, want) {
							t.Errorf("head missing %s", want)
						}
					}
					if strings.Count(content, `rel="icon"`) != 2 || strings.Count(content, `rel="manifest"`) != 1 {
						t.Error("duplicate or missing icon/manifest tags")
					}
				}
				if !strings.Contains(home, `property="og:type" content="website"`) || !strings.Contains(home, `property="og:image" content="https://margo.example/journal/assets/social.jpg"`) {
					t.Error("site fallback changed")
				}
				configArtifact(t, result, "posts/one/assets/cover image.png")
				configArtifact(t, result, "app/site.webmanifest")
				configArtifact(t, result, "assets/favicon.ico")
				for _, entry := range result.Manifest.Entries {
					if got := margo.ArtifactDigestOf(configArtifact(t, result, entry.Path)); got != entry.Digest {
						t.Errorf("stale digest %s", entry.Path)
					}
				}
				if result.Pages[1].PublishedAt != "2026-10-28T23:30:00-03:00" {
					t.Error("route timestamp changed")
				}
				// Parse HTML to assert single metadata values rather than trusting substrings.
				parsed, err := html.Parse(strings.NewReader(post))
				if err != nil {
					t.Fatalf("%#v", err)
				}
				if firstElement(firstElement(parsed, "head"), "link") == nil {
					t.Fatal("head missing links")
				}
			})
		}
	}
}

func TestPublicationInvalidInputs(t *testing.T) {
	for name, edit := range map[string]func(*testing.T, string, string){
		"missing cover": func(t *testing.T, root, _ string) {
			if err := os.Remove(filepath.Join(root, "docs/posts/one/assets/cover image.png")); err != nil {
				t.Fatal(err)
			}
		},
		"invalid cover": func(t *testing.T, root, _ string) {
			writeConfigFile(t, filepath.Join(root, "docs/posts/one/assets/cover image.png"), "not an image")
		},
		"invalid manifest": func(t *testing.T, root, _ string) {
			writeConfigFile(t, filepath.Join(root, "app/site.webmanifest"), "[]")
		},
		"invalid icon": func(t *testing.T, root, _ string) {
			writeConfigFile(t, filepath.Join(root, "assets/icon.png"), "not an icon")
		},
		"wrong icon size": func(t *testing.T, root, _ string) { writeTestPNG(t, filepath.Join(root, "assets/icon.png"), 16, 16) },
		"outside cover": func(t *testing.T, root, _ string) {
			p := filepath.Join(root, "docs/posts/one/index.md")
			d, _ := os.ReadFile(p)
			writeConfigFile(t, p, strings.Replace(string(d), "./assets/cover image.png", "../../../../secret.png", 1))
		},
		"remote cover": func(t *testing.T, root, _ string) {
			p := filepath.Join(root, "docs/posts/one/index.md")
			d, _ := os.ReadFile(p)
			writeConfigFile(t, p, strings.Replace(string(d), "./assets/cover image.png", "https://example.com/cover.png", 1))
		},
		"unknown cover option": func(t *testing.T, root, _ string) {
			p := filepath.Join(root, "docs/posts/one/index.md")
			d, _ := os.ReadFile(p)
			writeConfigFile(t, p, strings.Replace(string(d), "  path: ./assets/cover image.png", "  script: bad\n  path: ./assets/cover image.png", 1))
		},
	} {
		t.Run(name, func(t *testing.T) {
			root, filename := publicationFixture(t, "article", AssetsLocal)
			edit(t, root, filename)
			if _, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filename}); err == nil {
				t.Fatal("invalid publication accepted")
			}
		})
	}
	for _, entry := range []struct{ old, new string }{
		{"date_format: '02/01/2006'", "date_format: '  '"},
		{"'#007d9c'", "'red; background:url(x)'"},
		{"rel: shortcut icon", "rel: stylesheet"},
		{"app/site.webmanifest", "../outside.webmanifest"},
	} {
		t.Run(entry.new, func(t *testing.T) {
			_, filename := publicationFixture(t, "article", AssetsLocal)
			data, _ := os.ReadFile(filename)
			writeConfigFile(t, filename, strings.Replace(string(data), entry.old, entry.new, 1))
			if _, err := LoadConfig(filename); err == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
}

func TestFinalizePublicationArtifacts(t *testing.T) {
	initial := Result{Artifacts: []Artifact{{Path: "z.xml", Content: []byte("feed")}, {Path: "index.html", Content: []byte("changed HTML")}}, Pages: []Page{{Source: "index.md", Output: "index.html"}}, Site: SiteManifest{ConfigVersion: 1, BaseURL: "https://example.com", Routes: []Page{{Source: "index.md", Output: "index.html"}}}}
	finalized, err := Finalize(initial)
	if err != nil {
		t.Fatal(err)
	}
	if finalized.Artifacts[0].Path != "index.html" || finalized.Manifest.Entries[0].Digest != margo.ArtifactDigestOf([]byte("changed HTML")) {
		t.Fatal("final manifest did not describe current bytes")
	}
	finalized.Artifacts[0].Content[0] = 'X'
	if string(initial.Artifacts[1].Content) != "changed HTML" {
		t.Fatal("finalized bytes alias caller")
	}
	raw, err := MarshalManifest(initial, "trusted-policy")
	if err != nil {
		t.Fatal(err)
	}
	var document manifestDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	if document.Policy != "trusted-policy" || document.BaseURL != initial.Site.BaseURL || len(document.Routes) != 1 || document.Digest == "" {
		t.Fatal("lost publication identity")
	}
	initial.Artifacts[0], initial.Artifacts[1] = initial.Artifacts[1], initial.Artifacts[0]
	reordered, err := MarshalManifest(initial, "trusted-policy")
	if err != nil || !bytes.Equal(raw, reordered) {
		t.Fatal("manifest depends on input order")
	}
	initial.Artifacts[0].Content = []byte("newer HTML")
	changed, err := MarshalManifest(initial, "")
	if err != nil || bytes.Equal(raw, changed) {
		t.Fatal("manifest ignored changed bytes")
	}
	for _, artifacts := range [][]Artifact{
		{{Path: "../outside"}}, {{Path: "/absolute"}}, {{Path: "a/../b"}}, {{Path: "a\\b"}}, {{Path: "C:/x"}}, {{Path: ManifestPath}}, {{Path: ManifestPath + "/child"}},
		{{Path: "a"}, {Path: "A"}}, {{Path: "a"}, {Path: "a"}}, {{Path: "a"}, {Path: "a/b"}},
	} {
		if _, err := Finalize(Result{Artifacts: artifacts}); err == nil {
			t.Errorf("unsafe artifacts accepted: %+v", artifacts)
		}
	}
}

func TestPageSocialImageFormats(t *testing.T) {
	for _, sample := range []struct{ name, media string }{
		{"atelier-hero.jpg", "image/jpeg"}, {"atelier-hero.webp", "image/webp"}, {"format-study.gif", "image/gif"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			root := t.TempDir()
			data, err := os.ReadFile(filepath.Join("..", "examples/blog/site/assets", sample.name))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "cover.bin"), data, 0644); err != nil {
				t.Fatal(err)
			}
			b := builder{config: &Config{BasePath: "/journal", Site: SiteConfig{BaseURL: "https://example.com"}}, request: Request{SourceRoot: root, AssetReader: margo.FilesystemCheckAssetReader{}}, assets: map[string]cachedAsset{}, artifacts: map[string][]byte{}, artifactKeys: map[string]string{}}
			metadata := margo.Metadata{Additional: map[string]any{"social_image": map[string]any{"path": "./cover.bin", "alt": "Cover"}}}
			preview, err := b.preparePageSocialImage(context.Background(), Source{Path: "index.md"}, metadata)
			if err != nil {
				t.Fatalf("%#v", err)
			}
			dimensions, _, err := image.DecodeConfig(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if preview.MediaType != sample.media || preview.Width != dimensions.Width || preview.Height != dimensions.Height || preview.URL != "https://example.com/journal/cover.bin" {
				t.Fatalf("wrong image projection: %+v", preview)
			}
			before := b.assetBytes
			if _, err := b.preparePageSocialImage(context.Background(), Source{Path: "index.md"}, metadata); err != nil || b.assetBytes != before {
				t.Fatal("cached image counted twice")
			}
		})
	}
}

func TestArticleSocialTypeWithoutPublicationDates(t *testing.T) {
	for _, kind := range []string{"docs", "article", "landing"} {
		t.Run(kind, func(t *testing.T) {
			root, filename := publicationFixture(t, kind, AssetsLocal)
			writeConfigFile(t, filepath.Join(root, "docs/posts/one/index.md"), "---\nstructured_data:\n  type: Article\n---\n# An article\n")
			result, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filename})
			if err != nil {
				t.Fatal(err)
			}
			post := string(configArtifact(t, result, "posts/one/index.html"))
			if !strings.Contains(post, `property="og:type" content="article"`) || strings.Contains(post, `data-margo-publication-metadata`) {
				t.Fatal("explicit article requires invented editorial metadata")
			}
		})
	}
}

func TestPublicationDateFormattingPreservesTimestamp(t *testing.T) {
	for _, sample := range []struct{ timestamp, format, visible string }{
		{"2026-10-28T23:30:00-03:00", "02/01/2006", "28/10/2026"},
		{"2026-10-29T00:30:00+05:00", "02/01/2006", "29/10/2026"},
		{"2026-10-28T23:30:00.123-03:00", "2006-01-02", "2026-10-28"},
		{"2026-10-28T23:30:00-03:00", "", "2026-10-28T23:30:00-03:00"},
		{"2026-10-28", "02/01/2006", "28/10/2026"},
	} {
		node := publicationTime("published", sample.timestamp, sample.format)
		if attributeValue(node, "datetime") != sample.timestamp || node.FirstChild.Data != sample.visible {
			t.Errorf("incorrect date projection: %+v", sample)
		}
	}
}

func TestPageSocialImageRequiresCanonicalOrigin(t *testing.T) {
	_, filename := publicationFixture(t, "article", AssetsLocal)
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	writeConfigFile(t, filename, strings.Replace(string(data), "  base_url: https://margo.example\n", "", 1))
	if _, err := BuildConfig(context.Background(), ConfigRequest{ConfigPath: filename}); err == nil {
		t.Fatal("page social image accepted without a canonical origin")
	}
}
