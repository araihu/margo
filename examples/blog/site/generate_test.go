package site

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateBuildsBlogPagesWithPopularImageFormats(t *testing.T) {
	output := t.TempDir()
	if err := Generate(output); err != nil {
		t.Fatal(err)
	}

	index := readGeneratedFile(t, output, "index.html")
	for _, want := range []string{
		`<!doctype html>`, `<title>Margo Field Notes</title>`,
		`data-theme="margo-blog"`, `href="field-notes.html"`,
	} {
		if !strings.Contains(string(index), want) {
			t.Fatalf("index missing %q", want)
		}
	}

	article := readGeneratedFile(t, output, "field-notes.html")
	for _, want := range []string{
		`<title>Inside the Margo rendering atelier</title>`,
		`rel="canonical" href="https://margo.invalid/guide"`,
		`property="og:type" content="article"`,
		`class="margo-webpublication-byline"`,
		`<source type="image/avif" srcset="assets/atelier-hero.avif">`,
		`<source type="image/webp" srcset="assets/atelier-hero.webp">`,
		`src="assets/atelier-hero.jpg"`,
		`src="assets/format-study.png"`,
		`src="assets/format-study.gif"`,
	} {
		if !strings.Contains(string(article), want) {
			t.Fatalf("article missing %q", want)
		}
	}

	assertImageSignature(t, readGeneratedFile(t, output, "assets/atelier-hero.jpg"), "jpeg")
	assertImageSignature(t, readGeneratedFile(t, output, "assets/atelier-hero.webp"), "webp")
	assertImageSignature(t, readGeneratedFile(t, output, "assets/atelier-hero.avif"), "avif")
	assertImageSignature(t, readGeneratedFile(t, output, "assets/format-study.png"), "png")
	assertImageSignature(t, readGeneratedFile(t, output, "assets/format-study.gif"), "gif")
}

func readGeneratedFile(t *testing.T, root, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertImageSignature(t *testing.T, data []byte, format string) {
	t.Helper()
	switch format {
	case "jpeg":
		if len(data) < 3 || !bytes.Equal(data[:3], []byte{0xff, 0xd8, 0xff}) {
			t.Fatal("invalid JPEG signature")
		}
	case "webp":
		if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
			t.Fatal("invalid WebP signature")
		}
	case "avif":
		if len(data) < 32 || string(data[4:8]) != "ftyp" || !bytes.Contains(data[8:32], []byte("avif")) {
			t.Fatal("invalid AVIF signature")
		}
	case "png":
		if len(data) < 8 || !bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n")) {
			t.Fatal("invalid PNG signature")
		}
	case "gif":
		if len(data) < 6 || (string(data[:6]) != "GIF87a" && string(data[:6]) != "GIF89a") {
			t.Fatal("invalid GIF signature")
		}
	default:
		t.Fatalf("unknown format %q", format)
	}
}
