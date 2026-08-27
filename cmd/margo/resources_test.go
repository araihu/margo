package main

import (
	"encoding/base64"
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

func TestMaterializeLocalImagesAcceptsSVGWithXMLDeclaration(t *testing.T) {
	root := t.TempDir()
	declaredSVG := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="32" height="20"><rect width="32" height="20"/></svg>`)
	if err := os.WriteFile(filepath.Join(root, "declared.svg"), declaredSVG, 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := materializeLocalImages(
		[]byte(`<html><body><img src="declared.svg"></body></html>`),
		"<stdin>",
		root,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "data:image/svg+xml;base64,") {
		t.Fatalf("output does not embed declared SVG: %s", output)
	}
}

func TestImageMediaTypeRejectsUnsafeOrNonSVGXML(t *testing.T) {
	tests := []struct {
		name string
		data string
		code string
	}{
		{
			name: "active SVG after declaration",
			data: `<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`,
			code: "cli.resource_svg_active",
		},
		{
			name: "non-SVG XML",
			data: `<?xml version="1.0"?><document>not an image</document>`,
			code: "cli.resource_format_unsupported",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := imageMediaType([]byte(test.data)); cliDiagnosticCode(err) != test.code {
				t.Fatalf("error = %v, want %s", err, test.code)
			}
		})
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

func TestMaterializeLocalImagesValidatesContentAndDataURLs(t *testing.T) {
	root := t.TempDir()
	unsafeSVG := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	if err := os.WriteFile(filepath.Join(root, "unsafe.png"), unsafeSVG, 0o600); err != nil {
		t.Fatal(err)
	}
	dataSVG := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(unsafeSVG)
	for _, source := range []string{"unsafe.png", dataSVG} {
		document := []byte(`<html><body><img src="` + source + `"></body></html>`)
		if _, err := materializeLocalImages(document, "<stdin>", root); cliDiagnosticCode(err) != "cli.resource_svg_active" {
			t.Fatalf("source %q error = %v", source, err)
		}
	}
	png := []byte("\x89PNG\r\n\x1a\n")
	mislabeled := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(png)
	if _, err := materializeLocalImages([]byte(`<html><body><img src="`+mislabeled+`"></body></html>`), "<stdin>", root); cliDiagnosticCode(err) != "cli.resource_format_unsupported" {
		t.Fatalf("mislabeled data image error = %v", err)
	}
}

func TestMaterializeLocalImagesSupportsPopularRasterFormats(t *testing.T) {
	root := t.TempDir()
	fixtures := map[string]string{
		"sample.png":  "../../examples/blog/site/assets/format-study.png",
		"sample.jpg":  "../../examples/blog/site/assets/atelier-hero.jpg",
		"sample.webp": "../../examples/blog/site/assets/atelier-hero.webp",
		"sample.gif":  "../../examples/blog/site/assets/format-study.gif",
		"sample.avif": "../../examples/blog/site/assets/atelier-hero.avif",
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
	for _, mediaType := range []string{"image/png", "image/jpeg", "image/webp", "image/gif", "image/avif"} {
		if !strings.Contains(string(output), "data:"+mediaType+";base64,") {
			t.Fatalf("output missing %s", mediaType)
		}
	}
}
