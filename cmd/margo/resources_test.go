package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaterializeLocalImagesEmbedsPNGAndSVG(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pixel.png"), []byte("\x89PNG\r\n\x1a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "mark.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0h1v1z"/></svg>`), 0o600); err != nil {
		t.Fatal(err)
	}
	document := []byte(`<html><body><img src="pixel.png"><img src="mark.svg"></body></html>`)
	output, err := materializeLocalImages(document, "<stdin>", root)
	if err != nil {
		t.Fatal(err)
	}
	for _, prefix := range []string{"data:image/png;base64,", "data:image/svg+xml;base64,"} {
		if !strings.Contains(string(output), prefix) {
			t.Fatalf("output missing %q: %s", prefix, output)
		}
	}
}

func TestMaterializeLocalImagesRejectsRemoteAndTraversal(t *testing.T) {
	root := t.TempDir()
	for _, source := range []string{"https://example.com/image.png", "../outside.png", "file:///tmp/image.png"} {
		document := []byte(`<html><body><img src="` + source + `"></body></html>`)
		if _, err := materializeLocalImages(document, "<stdin>", root); cliDiagnosticCode(err) != "cli.resource_external" {
			t.Fatalf("source %q error = %v", source, err)
		}
	}
}

func TestMaterializeLocalImagesSupportsPopularRasterFormats(t *testing.T) {
	root := t.TempDir()
	fixtures := map[string]string{
		"sample.png":  "../../examples/blog/site/assets/format-study.png",
		"sample.jpg":  "../../examples/blog/site/assets/atelier-hero.jpg",
		"sample.webp": "../../examples/blog/site/assets/atelier-hero.webp",
		"sample.gif":  "../../examples/blog/site/assets/format-study.gif",
	}
	var images strings.Builder
	for name, fixture := range fixtures {
		data, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
		images.WriteString(`<img src="` + name + `">`)
	}
	output, err := materializeLocalImages([]byte("<html><body>"+images.String()+"</body></html>"), "<stdin>", root)
	if err != nil {
		t.Fatal(err)
	}
	for _, mediaType := range []string{"image/png", "image/jpeg", "image/webp", "image/gif"} {
		if !strings.Contains(string(output), "data:"+mediaType+";base64,") {
			t.Fatalf("output missing %s", mediaType)
		}
	}
}
